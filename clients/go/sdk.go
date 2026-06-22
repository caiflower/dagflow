package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dagflowpb "github.com/caiflower/dagflow/internal/proto"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
)

type Config struct {
	NodeID     string
	EngineAddr string
	ListenAddr string
}

type SDK struct {
	config   Config
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
	server   *grpc.Server
}

type HandlerFunc func(ctx context.Context, raw []byte) ([]byte, error)

func New(cfg Config) *SDK {
	return &SDK{
		config:   cfg,
		handlers: make(map[string]HandlerFunc),
	}
}

func Register[In, Out any](s *SDK, name string, fn func(ctx context.Context, input In) (Out, error)) {
	wrapper := func(ctx context.Context, raw []byte) ([]byte, error) {
		var in In
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, fmt.Errorf("unmarshal input: %w", err)
			}
		}
		out, err := fn(ctx, in)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("marshal output: %w", err)
		}
		return data, nil
	}
	s.mu.Lock()
	s.handlers[name] = wrapper
	s.mu.Unlock()
}

func (s *SDK) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.config.ListenAddr, err)
	}

	s.server = grpc.NewServer()
	pb.RegisterRemoteExecutorServer(s.server, &ExecutorServer{Sdk: s})

	go func() {
		s.server.Serve(lis)
	}()

	funcs := s.functionList()
	if err := s.registerWithEngine(ctx, funcs); err != nil {
		s.server.Stop()
		return fmt.Errorf("register with engine: %w", err)
	}

	go s.heartbeatLoop(ctx)

	<-ctx.Done()
	s.server.GracefulStop()
	return nil
}

func (s *SDK) functionList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	funcs := make([]string, 0, len(s.handlers))
	for name := range s.handlers {
		funcs = append(funcs, name)
	}
	return funcs
}

func (s *SDK) registerWithEngine(ctx context.Context, funcs []string) error {
	conn, err := grpc.NewClient(s.config.EngineAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial engine %s: %w", s.config.EngineAddr, err)
	}
	defer conn.Close()

	client := pb.NewNodeRegistryClient(conn)
	_, err = client.Register(ctx, &pb.RegisterRequest{
		NodeId:    s.config.NodeID,
		Address:   s.config.ListenAddr,
		Functions: funcs,
	})
	return err
}

func (s *SDK) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	conn, err := grpc.NewClient(s.config.EngineAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return
	}
	defer conn.Close()
	client := pb.NewNodeRegistryClient(conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			client.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: s.config.NodeID})
		}
	}
}

// ExecutorServer implements the gRPC RemoteExecutor service, dispatching
// calls to registered SDK handler functions.
type ExecutorServer struct {
	pb.UnimplementedRemoteExecutorServer
	Sdk *SDK
}

func (s *ExecutorServer) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	s.Sdk.mu.RLock()
	handler, ok := s.Sdk.handlers[req.FuncName]
	s.Sdk.mu.RUnlock()

	if !ok {
		return &pb.ExecuteResponse{Error: fmt.Sprintf("function %q not registered", req.FuncName)}, nil
	}
	output, err := handler(ctx, req.Input)
	if err != nil {
		return &pb.ExecuteResponse{Error: err.Error()}, nil
	}
	return &pb.ExecuteResponse{Output: output}, nil
}

func (s *ExecutorServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Ok: true}, nil
}

// ===== DAGFlow gRPC Client =====

// Client provides gRPC access to DAGFlow engine services.
type Client struct {
	conn       *grpc.ClientConn
	Flow       dagflowpb.FlowServiceClient
	Protocol   dagflowpb.ProtocolServiceClient
	Execution  dagflowpb.ExecutionServiceClient
}

// NewClient creates a new DAGFlow gRPC client connected to the engine address.
func NewClient(engineAddr string) (*Client, error) {
	conn, err := grpc.NewClient(engineAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial engine %s: %w", engineAddr, err)
	}
	return &Client{
		conn:      conn,
		Flow:      dagflowpb.NewFlowServiceClient(conn),
		Protocol:  dagflowpb.NewProtocolServiceClient(conn),
		Execution: dagflowpb.NewExecutionServiceClient(conn),
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
