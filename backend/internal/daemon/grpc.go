package daemon

import (
	"fmt"
	"net"

	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/dagflow/constants"
	"github.com/caiflower/dagflow/internal/api"
	"github.com/caiflower/dagflow/internal/node_registry"
	dagflowpb "github.com/caiflower/dagflow/internal/proto"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
	"google.golang.org/grpc"
)

// GrpcServer wraps a gRPC server hosting all DAGFlow services as a daemon resource.
type GrpcServer struct {
	server  *grpc.Server
	lis     net.Listener
	NodeReg *node_registry.NodeRegistry `autowired:""`
}

func NewGrpcServer() *GrpcServer {
	return &GrpcServer{}
}

func (g *GrpcServer) Name() string { return "GrpcServer" }

func (g *GrpcServer) Start() error {
	port := constants.Prop.GRPC.Port
	if port == 0 {
		port = 50051
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Error("failed to listen on gRPC port %d: %v", port, err)
		return err
	}

	s := grpc.NewServer()

	// Register NodeRegistry service
	pb.RegisterNodeRegistryServer(s, g.NodeReg)
	logger.Info("gRPC: NodeRegistry service registered")

	// Register Flow/Protocol/Execution services
	if flowSvc := api.GetFlowGrpcService(); flowSvc != nil {
		dagflowpb.RegisterFlowServiceServer(s, flowSvc)
		logger.Info("gRPC: Flow service registered")
	}
	if protoSvc := api.GetProtocolGrpcService(); protoSvc != nil {
		dagflowpb.RegisterProtocolServiceServer(s, protoSvc)
		logger.Info("gRPC: Protocol service registered")
	}
	if execSvc := api.GetExecutionGrpcService(); execSvc != nil {
		dagflowpb.RegisterExecutionServiceServer(s, execSvc)
		logger.Info("gRPC: Execution service registered")
	}

	g.server = s
	g.lis = lis

	logger.Info("gRPC server listening on %s", g.lis.Addr().String())

	go func() {
		if err := g.server.Serve(g.lis); err != nil {
			logger.Error("failed to start gRPC server: %v", err)
		}
	}()

	return nil
}

func (g *GrpcServer) Close() {
	logger.Info("gRPC server shutting down...")
	g.server.GracefulStop()
	logger.Info("gRPC server stopped")
}
