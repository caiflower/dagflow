package node_registry

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/caiflower/dagflow/proto/remote_executor"
)

// testRedisClient is a minimal v2.RedisClient adapter for testing.
type testRedisClient struct {
	client *redis.Client
}

func (c *testRedisClient) Cmd() v2.Cmdable {
	return &testCmd{Cmdable: c.client}
}
func (c *testRedisClient) GetRedis() redis.Cmdable { return c.client }
func (c *testRedisClient) AddHook(hook redis.Hook) {}
func (c *testRedisClient) Close()                  { c.client.Close() }

type testCmd struct {
	redis.Cmdable
}

func (c *testCmd) Key(key string) string { return key }

// testClock synchronizes with miniredis virtual time.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func setupRegistry(t *testing.T) (*NodeRegistry, *miniredis.Miniredis, *testClock) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rc := &testRedisClient{client: client}
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	clock := &testClock{now: time.Now()}
	mr.SetTime(clock.now)
	reg := &NodeRegistry{redis: rc, timeFunc: clock.Now}
	return reg, mr, clock
}

func advanceTime(mr *miniredis.Miniredis, clock *testClock, d time.Duration) {
	clock.advance(d)
	mr.FastForward(d)
}

func TestRegisterStoresNodeInfo(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	ctx := context.Background()

	resp, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Ok)

	// Verify via GetNode
	nodeResp, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.Equal(t, "node-1", nodeResp.Node.NodeId)
	assert.Equal(t, "localhost:50052", nodeResp.Node.Address)
	assert.Contains(t, nodeResp.Node.Functions, "funcA")
	assert.Contains(t, nodeResp.Node.Functions, "funcB")
	assert.Equal(t, "online", nodeResp.Node.Status)
	assert.Greater(t, nodeResp.Node.LastHeartbeat, int64(0))

	// Verify via ListNodes
	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Items, 1)
	assert.Equal(t, "node-1", listResp.Items[0].NodeId)
	assert.Equal(t, "online", listResp.Items[0].Status)
}

func TestHeartbeatUpdatesLastHeartbeat(t *testing.T) {
	reg, mr, clock := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// Get initial lastHeartbeat
	nodeResp, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	initialHB := nodeResp.Node.LastHeartbeat

	// Advance time and send heartbeat
	advanceTime(mr, clock, 20*time.Second)

	hbResp, err := reg.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.True(t, hbResp.Ok)

	// lastHeartbeat should be updated
	nodeResp, err = reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.Greater(t, nodeResp.Node.LastHeartbeat, initialHB)
}

func TestNodeGoesOfflineAfterThreshold(t *testing.T) {
	reg, mr, clock := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// Initially online
	nodeResp, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.Equal(t, "online", nodeResp.Node.Status)

	// Fast forward past heartbeat threshold but within node TTL
	advanceTime(mr, clock, 31*time.Second)

	// Now should be offline
	nodeResp, err = reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.Equal(t, "offline", nodeResp.Node.Status)
}

func TestGetNodeNotFound(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListNodesWithMixedStatus(t *testing.T) {
	reg, mr, clock := setupRegistry(t)
	ctx := context.Background()

	// Register two nodes
	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-online",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	_, err = reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-offline",
		Address:   "localhost:50053",
		Functions: []string{"funcB"},
	})
	require.NoError(t, err)

	// Fast forward past the heartbeat threshold
	advanceTime(mr, clock, 31*time.Second)

	// Both nodes were registered before the FastForward, so both should be offline
	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Items, 2)

	// Both should be offline after FastForward past threshold
	for _, item := range listResp.Items {
		assert.Equal(t, "offline", item.Status)
	}
}

func TestFunctionIndex(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)

	_, err = reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-2",
		Address:   "localhost:50053",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// GetNodesForFunc should return both nodes for funcA
	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Verify node IDs
	nodeIDs := make([]string, len(nodes))
	for i, n := range nodes {
		nodeIDs[i] = n.NodeID
	}
	assert.Contains(t, nodeIDs, "node-1")
	assert.Contains(t, nodeIDs, "node-2")
}

func TestGetNodesForFuncNoMatch(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	nodes, err := reg.GetNodesForFunc(ctx, "funcB")
	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestNodeExpiresAfterTTL(t *testing.T) {
	reg, mr, clock := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// Fast forward past node TTL (5 min)
	advanceTime(mr, clock, 6*time.Minute)

	// Node should be gone
	_, err = reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	assert.Error(t, err)

	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	assert.Empty(t, listResp.Items)
}

func TestListNodesEmpty(t *testing.T) {
	reg, _, _ := setupRegistry(t)
	ctx := context.Background()

	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	assert.Empty(t, listResp.Items)
	assert.Equal(t, int32(0), listResp.Total)
}
