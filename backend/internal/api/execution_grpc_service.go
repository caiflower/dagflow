package api

import (
	"context"

	pb "github.com/caiflower/dagflow/internal/proto"
	"github.com/caiflower/dagflow/internal/service"
)

// ExecutionGrpcService Execution gRPC 服务实现
type ExecutionGrpcService struct {
	pb.UnimplementedExecutionServiceServer
	svc *service.ExecutionService
}

func NewExecutionGrpcService(svc *service.ExecutionService) *ExecutionGrpcService {
	return &ExecutionGrpcService{svc: svc}
}

func (s *ExecutionGrpcService) Run(ctx context.Context, req *pb.RunFlowRequest) (*pb.ExecutionResponse, error) {
	exec, err := s.svc.Run(ctx, &service.RunFlowReq{FlowID: req.FlowId})
	if err != nil {
		return nil, err
	}
	return &pb.ExecutionResponse{Execution: execToProto(exec)}, nil
}

func (s *ExecutionGrpcService) Get(ctx context.Context, req *pb.GetExecutionRequest) (*pb.ExecutionResponse, error) {
	exec, err := s.svc.GetStatus(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.ExecutionResponse{Execution: execToProto(exec)}, nil
}

func (s *ExecutionGrpcService) List(ctx context.Context, _ *pb.ListExecutionRequest) (*pb.ListExecutionResponse, error) {
	execs := s.svc.ListExecutions(ctx)
	items := make([]*pb.Execution, len(execs))
	for i, e := range execs {
		items[i] = execToProto(e)
	}
	return &pb.ListExecutionResponse{Items: items}, nil
}

func execToProto(e *service.Execution) *pb.Execution {
	if e == nil {
		return nil
	}
	nodes := make([]*pb.NodeStatus, len(e.Nodes))
	for i, n := range e.Nodes {
		nodes[i] = &pb.NodeStatus{Id: n.ID, Name: n.Name, State: n.State}
	}
	return &pb.Execution{
		Id:        e.ID,
		FlowId:    e.FlowID,
		FlowName:  e.FlowName,
		State:     e.State,
		StartTime: e.StartTime.String(),
		EndTime:   e.EndTime.String(),
		Nodes:     nodes,
	}
}
