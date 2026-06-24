package remote_executor

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

// fakeExecutorServer simulates an SDK node with dynamic handler registration.
type fakeExecutorServer struct {
	pb.UnimplementedRemoteExecutorServer
	handlers map[string]func(ctx context.Context, input []byte) ([]byte, error)
}

func (s *fakeExecutorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	handler, ok := s.handlers[req.FuncName]
	if !ok {
		return &pb.ExecuteResponse{Error: "function not registered"}, nil
	}
	output, err := handler(ctx, req.Input)
	if err != nil {
		return &pb.ExecuteResponse{Error: err.Error()}, nil
	}
	return &pb.ExecuteResponse{Output: output}, nil
}

func (s *fakeExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}

// startNodeServer starts a bufconn-backed gRPC server using fakeExecutorServer
// and returns a *grpc.ClientConn pointing at it.
func startNodeServer(t *testing.T, handlers map[string]func(ctx context.Context, input []byte) ([]byte, error)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterRemoteExecutorServer(s, &fakeExecutorServer{handlers: handlers})
	go s.Serve(lis)
	t.Cleanup(s.Stop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestIntegrationSingleNodeExecute tests the full flow:
// register node → get nodes → execute → get result.
func TestIntegrationSingleNodeExecute(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	conn := startNodeServer(t, map[string]func(ctx context.Context, input []byte) ([]byte, error){
		"echo": func(ctx context.Context, input []byte) ([]byte, error) {
			var m map[string]any
			json.Unmarshal(input, &m)
			m["echoed"] = true
			return json.Marshal(m)
		},
	})

	reg := node_registry.NewNodeRegistry(rc)
	ctx := context.Background()
	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "integration-node-1",
		Address:   "bufconn",
		Functions: []string{"echo"},
	})
	require.NoError(t, err)

	pool := NewConnPool()
	pool.conns["bufconn"] = &connEntry{conn: conn, refs: 1}

	provider := &RemoteFuncProvider{
		FuncName: "echo",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     pool,
	}

	result, err := provider.Execute(ctx, &executor.TaskData{
		TaskId: "task-1",
		Input:  `{"msg":"hello"}`,
	})
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, m["output"].(string), "echoed")

}

// TestIntegrationMultiNodeRandomPick registers two nodes and verifies
// the provider picks one (random selection, so we only check the output
// contains a known node identifier).
func TestIntegrationMultiNodeRandomPick(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	conn1 := startNodeServer(t, map[string]func(ctx context.Context, input []byte) ([]byte, error){
		"identify": func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"node":"node-1"}`), nil
		},
	})
	conn2 := startNodeServer(t, map[string]func(ctx context.Context, input []byte) ([]byte, error){
		"identify": func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"node":"node-2"}`), nil
		},
	})

	reg := node_registry.NewNodeRegistry(rc)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId: "node-1", Address: "bufconn-1", Functions: []string{"identify"},
	})
	require.NoError(t, err)
	_, err = reg.Register(ctx, &pb.RegisterRequest{
		NodeId: "node-2", Address: "bufconn-2", Functions: []string{"identify"},
	})
	require.NoError(t, err)

	pool := NewConnPool()
	pool.conns["bufconn-1"] = &connEntry{conn: conn1, refs: 1}
	pool.conns["bufconn-2"] = &connEntry{conn: conn2, refs: 1}

	provider := &RemoteFuncProvider{
		FuncName: "identify",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     pool,
	}

	result, err := provider.Execute(ctx, &executor.TaskData{TaskId: "task-2", Input: `{}`})
	require.NoError(t, err)
	m := result.(map[string]any)
	assert.Contains(t, m["output"].(string), "node-")

}

// TestIntegrationNodeHeartbeatTimeout verifies expired nodes are not scheduled.
func TestIntegrationNodeHeartbeatTimeout(t *testing.T) {
	rc, mr := newTestRedisClient(t)

	conn := startNodeServer(t, map[string]func(ctx context.Context, input []byte) ([]byte, error){
		"test": func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"ok":true}`), nil
		},
	})

	reg := node_registry.NewNodeRegistry(rc)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId: "ephemeral-node", Address: "bufconn-exp", Functions: []string{"test"},
	})
	require.NoError(t, err)

	// Fast forward past TTL so the node key expires.
	mr.FastForward(6 * time.Minute)

	pool := NewConnPool()
	pool.conns["bufconn-exp"] = &connEntry{conn: conn, refs: 1}

	provider := &RemoteFuncProvider{
		FuncName: "test",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     pool,
	}

	_, err = provider.Execute(ctx, &executor.TaskData{TaskId: "task-3", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")
}

// TestIntegrationUnregisteredFunction verifies error when function not found.
func TestIntegrationUnregisteredFunction(t *testing.T) {
	rc, _ := newTestRedisClient(t)

	conn := startNodeServer(t, map[string]func(ctx context.Context, input []byte) ([]byte, error){
		"knownFunc": func(ctx context.Context, input []byte) ([]byte, error) {
			return []byte(`{"ok":true}`), nil
		},
	})

	reg := node_registry.NewNodeRegistry(rc)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId: "node-x", Address: "bufconn-unknown", Functions: []string{"knownFunc"},
	})
	require.NoError(t, err)

	pool := NewConnPool()
	pool.conns["bufconn-unknown"] = &connEntry{conn: conn, refs: 1}

	provider := &RemoteFuncProvider{
		FuncName: "unknownFunc",
		Timeout:  5 * time.Second,
		Registry: reg,
		Pool:     pool,
	}

	_, err = provider.Execute(ctx, &executor.TaskData{TaskId: "task-4", Input: `{}`})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available nodes")

}
