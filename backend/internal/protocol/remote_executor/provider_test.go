package remote_executor

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

const bufSize = 1024 * 1024

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

// newTestRedisClient creates a miniredis-backed v2.RedisClient for testing.
func newTestRedisClient(t *testing.T) (v2.RedisClient, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rc := &testRedisClient{client: client}
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return rc, mr
}

// mockExecutorServer implements pb.RemoteExecutorServer for testing.
type mockExecutorServer struct {
	pb.UnimplementedRemoteExecutorServer
	output string
	errMsg string
}

func (m *mockExecutorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	if m.errMsg != "" {
		return &pb.ExecuteResponse{Error: m.errMsg}, nil
	}
	return &pb.ExecuteResponse{Output: []byte(m.output)}, nil
}

func (m *mockExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}

// startMockServer starts a gRPC server on a bufconn listener.
func startMockServer(t *testing.T, mock *mockExecutorServer) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterRemoteExecutorServer(s, mock)
	go func() {
		_ = s.Serve(lis)
	}()
	t.Cleanup(func() {
		s.Stop()
		lis.Close()
	})
	return lis
}

// newBufconnClientConn creates a *grpc.ClientConn that dials through
// the given bufconn listener. Used to pre-populate the ConnPool.
func newBufconnClientConn(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// setupProvider creates a RemoteFuncProvider with a miniredis-backed
// NodeRegistry and a fresh ConnPool. Callers must pre-populate the pool
// with bufconn-backed connections using preloadPoolConn.
func setupProvider(t *testing.T, addr string, funcs []string) (*RemoteFuncProvider, *miniredis.Miniredis) {
	t.Helper()
	rc, mr := newTestRedisClient(t)
	reg := node_registry.NewNodeRegistry(rc)

	ctx := context.Background()
	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "test-node",
		Address:   addr,
		Functions: funcs,
	})
	require.NoError(t, err)

	return &RemoteFuncProvider{
		FuncName: "testFunc",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     NewConnPool(),
	}, mr
}

// preloadPoolConn inserts a bufconn-backed *grpc.ClientConn into the pool
// under the given address so that GetConn can look it up without a real dial.
func preloadPoolConn(t *testing.T, pool *ConnPool, address string, conn *grpc.ClientConn) {
	t.Helper()
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.conns[address] = &connEntry{conn: conn, refs: 1}
}

// --- Tests ---

func TestExecuteNoNodes(t *testing.T) {
	rc, _ := newTestRedisClient(t)
	reg := node_registry.NewNodeRegistry(rc)

	p := &RemoteFuncProvider{
		FuncName: "nonexistent",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     NewConnPool(),
	}

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")
}

func TestExecuteWrongFuncName(t *testing.T) {
	mock := &mockExecutorServer{output: `{"ok":true}`}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"otherFunc"})
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")
}

func TestExecuteSuccess(t *testing.T) {
	mock := &mockExecutorServer{output: `{"ok":true}`}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"testFunc"})
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	result, err := p.Execute(context.Background(), &executor.TaskData{
		TaskId: "t1",
		Input:  `{"name":"test"}`,
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(result.([]byte), &m))
	assert.True(t, m["ok"].(bool), "ok flag should be true")
}

func TestExecuteRemoteError(t *testing.T) {
	mock := &mockExecutorServer{errMsg: "something went wrong"}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"testFunc"})
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something went wrong")
	assert.Contains(t, err.Error(), "testFunc")
}

func TestProtocol(t *testing.T) {
	rc, _ := newTestRedisClient(t)
	reg := node_registry.NewNodeRegistry(rc)

	p := &RemoteFuncProvider{
		FuncName: "testFunc",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     NewConnPool(),
	}

	assert.Equal(t, ProtocolRemoteFunc, p.Protocol())
	assert.Equal(t, executor.Protocol("remoteFunc"), p.Protocol())
}

func TestExecuteTimeout(t *testing.T) {
	mock := &mockExecutorServer{output: `{"ok":true}`}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"testFunc"})
	p.Timeout = 1 * time.Nanosecond // force immediate deadline
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	_, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote execute")
}

func TestExecuteConnectionReused(t *testing.T) {
	mock := &mockExecutorServer{output: `{"ok":true}`}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"testFunc"})
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	// First call — GetConn increments refs 1→2, Release decrements to 1.
	result, err := p.Execute(context.Background(), &executor.TaskData{
		TaskId: "t1",
		Input:  `{"name":"first"}`,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Connection should still be cached (refs=1 after release).
	p.Pool.mu.Lock()
	_, exists := p.Pool.conns["bufconn"]
	p.Pool.mu.Unlock()
	assert.True(t, exists, "connection should stay cached after single execute")

	// Second call — should reuse the cached connection.
	result2, err2 := p.Execute(context.Background(), &executor.TaskData{
		TaskId: "t2",
		Input:  `{"name":"second"}`,
	})
	require.NoError(t, err2)
	assert.NotNil(t, result2)
}

func TestExecuteEmptyInput(t *testing.T) {
	mock := &mockExecutorServer{output: `{}`}
	lis := startMockServer(t, mock)
	conn := newBufconnClientConn(t, lis)

	p, _ := setupProvider(t, "bufconn", []string{"testFunc"})
	preloadPoolConn(t, p.Pool, "bufconn", conn)

	result, err := p.Execute(context.Background(), &executor.TaskData{TaskId: "t1", Input: ``})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(result.([]byte), &m))
	assert.NotNil(t, m)
}
