package api

import (
	"context"
	"encoding/json"

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/internal/model"
	pb "github.com/caiflower/dagflow/internal/proto"
	"github.com/caiflower/dagflow/internal/service"
)

// FlowGrpcService Flow gRPC 服务实现
type FlowGrpcService struct {
	pb.UnimplementedFlowServiceServer
	svc *service.FlowService
}

func NewFlowGrpcService(svc *service.FlowService) *FlowGrpcService {
	return &FlowGrpcService{svc: svc}
}

func (s *FlowGrpcService) Create(ctx context.Context, req *pb.CreateFlowRequest) (*pb.FlowResponse, error) {
	nodes := pbNodesToInternal(req.Nodes)
	edges := pbEdgesToInternal(req.Edges)
	flow, err := s.svc.Create(ctx, &service.CreateFlowReq{
		Name:        req.Name,
		Description: req.Description,
		Nodes:       nodes,
		Edges:       edges,
	})
	if err != nil {
		return nil, err
	}
	return &pb.FlowResponse{Flow: flowToProto(flow)}, nil
}

func (s *FlowGrpcService) Get(ctx context.Context, req *pb.GetFlowRequest) (*pb.FlowResponse, error) {
	flow, err := s.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.FlowResponse{Flow: flowToProto(flow)}, nil
}

func (s *FlowGrpcService) List(ctx context.Context, req *pb.ListFlowRequest) (*pb.ListFlowResponse, error) {
	flows, total, err := s.svc.List(ctx, &service.ListFlowReq{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Name:     req.Name,
	})
	if err != nil {
		return nil, err
	}
	items := make([]*pb.Flow, len(flows))
	for i := range flows {
		items[i] = flowToProto(&flows[i])
	}
	return &pb.ListFlowResponse{Items: items, Total: int32(total)}, nil
}

func (s *FlowGrpcService) Update(ctx context.Context, req *pb.UpdateFlowRequest) (*pb.FlowResponse, error) {
	updateReq := &service.UpdateFlowReq{
		ID:          req.Id,
		Name:        req.Name,
		Description: req.Description,
	}
	if req.Nodes != nil {
		updateReq.Nodes = pbNodesToInternal(req.Nodes)
	}
	if req.Edges != nil {
		updateReq.Edges = pbEdgesToInternal(req.Edges)
	}
	flow, err := s.svc.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return &pb.FlowResponse{Flow: flowToProto(flow)}, nil
}

func (s *FlowGrpcService) Delete(ctx context.Context, req *pb.DeleteFlowRequest) (*pb.DeleteFlowResponse, error) {
	if err := s.svc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.DeleteFlowResponse{Status: "ok"}, nil
}

func (s *FlowGrpcService) Validate(ctx context.Context, req *pb.ValidateFlowRequest) (*pb.ValidateFlowResponse, error) {
	nodes := pbNodesToInternal(req.Nodes)
	edges := pbEdgesToInternal(req.Edges)
	result, err := s.svc.Validate(ctx, &service.ValidateReq{Nodes: nodes, Edges: edges})
	if err != nil {
		return nil, err
	}
	resp := &pb.ValidateFlowResponse{Valid: true}
	if m, ok := result.(map[string]interface{}); ok {
		if v, ok := m["valid"].(bool); ok {
			resp.Valid = v
		}
		if e, ok := m["error"].(string); ok {
			resp.Error = e
		}
	}
	return resp, nil
}

// ===== Proto ↔ Internal 转换 =====

func pbNodesToInternal(nodes []*pb.FlowNode) []converter.FlowNode {
	result := make([]converter.FlowNode, len(nodes))
	for i, n := range nodes {
		var config map[string]any
		if n.ConfigJson != "" {
			_ = json.Unmarshal([]byte(n.ConfigJson), &config)
		}
		result[i] = converter.FlowNode{
			ID:       n.Id,
			Name:     n.Name,
			Type:     n.Type,
			Protocol: n.Protocol,
			Config:   config,
			Position: &converter.Position{X: n.PositionX, Y: n.PositionY},
		}
	}
	return result
}

func pbEdgesToInternal(edges []*pb.FlowEdge) []converter.FlowEdge {
	result := make([]converter.FlowEdge, len(edges))
	for i, e := range edges {
		result[i] = converter.FlowEdge{
			ID:     e.Id,
			Source: e.Source,
			Target: e.Target,
			Type:   e.Type,
			Expr:   e.Expr,
		}
	}
	return result
}

func flowToProto(f *model.Flow) *pb.Flow {
	if f == nil {
		return nil
	}
	return &pb.Flow{
		Id:          f.ID,
		Name:        f.Name,
		Description: f.Description,
		NodesJson:   f.NodesJSON,
		EdgesJson:   f.EdgesJSON,
		Version:     int32(f.Version),
		Status:      int32(f.Status),
		CreateTime:  f.CreateTime.String(),
		UpdateTime:  f.UpdateTime.String(),
	}
}
