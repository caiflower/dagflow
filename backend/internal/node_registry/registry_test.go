package node_registry

import (
	"context"
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

func setupRegistry(t *testing.T) (*NodeRegistry, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rc := &testRedisClient{client: client}
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return NewNodeRegistry(rc), mr
}

func TestRegisterStoresNodeInfo(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	resp, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Ok)

	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, "node-1", nodes[0].NodeID)
	assert.Equal(t, "localhost:50052", nodes[0].Address)
	assert.Contains(t, nodes[0].Functions, "funcA")
}

func TestHeartbeatRefreshesTTL(t *testing.T) {
	reg, mr := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	mr.FastForward(20 * time.Second)

	_, err = reg.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: "node-1"})
	require.NoError(t, err)

	mr.FastForward(20 * time.Second)
	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
}

func TestGetNodesForFuncNoMatch(t *testing.T) {
	reg, _ := setupRegistry(t)
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
	reg, mr := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	mr.FastForward(31 * time.Second)

	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Empty(t, nodes, "node should expire after TTL without heartbeat")
}
