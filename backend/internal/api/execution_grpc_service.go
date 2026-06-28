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
	// 将 proto NodeInput 转换为 map[nodeName]input
	nodeInputs := make(map[string]string)
	for _, ni := range req.NodeInputs {
		if ni.NodeName != "" {
			nodeInputs[ni.NodeName] = ni.Input
		}
	}
	exec, err := s.svc.Run(ctx, &service.RunFlowReq{FlowID: req.FlowId, NodeInputs: nodeInputs})
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

func (s *ExecutionGrpcService) List(ctx context.Context, req *pb.ListExecutionRequest) (*pb.ListExecutionResponse, error) {
	execs, total, err := s.svc.ListExecutions(ctx, int(req.GetPage()), int(req.GetPageSize()), req.GetFlowId())
	if err != nil {
		return nil, err
	}
	items := make([]*pb.Execution, len(execs))
	for i, e := range execs {
		items[i] = execToProto(e)
	}
	return &pb.ListExecutionResponse{Items: items, Total: int32(total)}, nil
}

func (s *ExecutionGrpcService) Retry(ctx context.Context, req *pb.RetryExecutionRequest) (*pb.RetryExecutionResponse, error) {
	if req.Id == "" {
		return &pb.RetryExecutionResponse{
			Success: false,
			Message: "execution id is required",
		}, nil
	}

	count, err := s.svc.Retry(ctx, req.Id)
	if err != nil {
		return &pb.RetryExecutionResponse{
			Success:            false,
			Message:            err.Error(),
			ResetSubtasksCount: int32(count),
		}, nil
	}

	return &pb.RetryExecutionResponse{
		Success:            true,
		Message:            "Execution retry initiated successfully",
		ResetSubtasksCount: int32(count),
	}, nil
}

func execToProto(e *service.Execution) *pb.Execution {
	if e == nil {
		return nil
	}
	nodes := make([]*pb.NodeStatus, len(e.Nodes))
	for i, n := range e.Nodes {
		nodes[i] = &pb.NodeStatus{
			Id:         n.ID,
			Name:       n.Name,
			State:      n.State,
			Input:      n.Input,
			Output:     n.Output,
			StartTime:  n.StartTime,
			EndTime:    n.EndTime,
			DurationMs: n.DurationMs,
			NodeType:   n.NodeType,
			Protocol:   n.Protocol,
		}
	}
	return &pb.Execution{
		Id:        e.ID,
		FlowId:    e.FlowID,
		FlowName:  e.FlowName,
		State:     e.State,
		StartTime: e.StartTime.String(),
		EndTime:   e.EndTime.String(),
		Nodes:     nodes,
		TaskId:    e.TaskID,
	}
}
