package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/caiflower/common-tools/web/common/e"
	"github.com/caiflower/dagflow/internal/dao"
	"github.com/caiflower/dagflow/internal/dao/model"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/tools"

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/taskx"
)

// FlowService Flow 业务逻辑层
type FlowService struct {
	FlowDAO *dao.FlowDAO `autowired:""`
}

// CreateFlowReq 创建 Flow 请求
type CreateFlowReq struct {
	Name        string               `json:"name" verf:"required"`
	Description string               `json:"description"`
	Nodes       []converter.FlowNode `json:"nodes" verf:"required"`
	Edges       []converter.FlowEdge `json:"edges"`
}

// UpdateFlowReq 更新 Flow 请求
type UpdateFlowReq struct {
	ID          string               `json:"id" verf:"required"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Nodes       []converter.FlowNode `json:"nodes"`
	Edges       []converter.FlowEdge `json:"edges"`
}

// ListFlowReq 查询 Flow 列表请求
type ListFlowReq struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Name     string `json:"name"`
}

// Create 创建 Flow
func (s *FlowService) Create(ctx context.Context, req *CreateFlowReq) (*model.Flow, error) {
	if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
		return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
	}

	flow := &model.Flow{
		ID:          tools.GenerateId("f"),
		Name:        req.Name,
		Description: req.Description,
		NodesJSON:   converter.NodesToJSON(req.Nodes),
		EdgesJSON:   converter.EdgesToJSON(req.Edges),
		Version:     1,
		Status:      1,
		CreateTime:  basic.NewFromTime(time.Now()),
		UpdateTime:  basic.NewFromTime(time.Now()),
	}

	if _, err := s.FlowDAO.Insert(ctx, flow); err != nil {
		return nil, fmt.Errorf("insert flow: %w", err)
	}
	// Register providers for all nodes so other instances can resolve them
	RegisterFlowProviders(flow)
	return flow, nil
}

// Get 查询单个 Flow
func (s *FlowService) Get(ctx context.Context, id string) (*model.Flow, error) {
	flow, err := s.FlowDAO.GetByID(ctx, id)
	if err != nil {
		return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %s not found", id), err)
	}
	return flow, nil
}

// List 查询 Flow 列表
func (s *FlowService) List(ctx context.Context, req *ListFlowReq) ([]model.Flow, int, error) {
	filter := &model.FlowFilter{
		Page:     req.Page,
		PageSize: req.PageSize,
		Name:     req.Name,
	}
	return s.FlowDAO.List(ctx, filter)
}

// Update 更新 Flow
func (s *FlowService) Update(ctx context.Context, req *UpdateFlowReq) (*model.Flow, error) {
	existing, err := s.FlowDAO.GetByID(ctx, req.ID)
	if err != nil {
		return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %s not found", req.ID), err)
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Nodes != nil {
		if err := converter.ValidateFlow(req.Nodes, req.Edges); err != nil {
			return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
		}
		existing.NodesJSON = converter.NodesToJSON(req.Nodes)
	}
	if req.Edges != nil {
		existing.EdgesJSON = converter.EdgesToJSON(req.Edges)
	}
	existing.Version++
	existing.UpdateTime = basic.NewFromTime(time.Now())

	// Clear old providers and re-register
	taskx.ClearProviders(existing.Name)
	RegisterFlowProviders(existing)
	if err := s.FlowDAO.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update flow: %w", err)
	}
	return existing, nil
}

// Delete 删除 Flow（软删除）
func (s *FlowService) Delete(ctx context.Context, id string) error {
	flow, err := s.FlowDAO.GetByID(ctx, id)
	if err != nil {
		return e.NewApiError(e.NotFound, fmt.Sprintf("flow %s not found", id), err)
	}
	taskx.ClearProviders(flow.Name)
	return s.FlowDAO.Delete(ctx, id)
}

// ValidateReq 校验 Flow 请求
type ValidateReq struct {
	Nodes []converter.FlowNode `json:"nodes" verf:"required"`
	Edges []converter.FlowEdge `json:"edges"`
}

// Validate 校验 Flow 定义
func (s *FlowService) Validate(ctx context.Context, req *ValidateReq) (interface{}, error) {
	err := converter.ValidateFlow(req.Nodes, req.Edges)
	if err != nil {
		return map[string]interface{}{"valid": false, "error": err.Error()}, nil
	}
	return map[string]interface{}{"valid": true}, nil
}

// Init 注册 FlowService bean
func Init() {
	bean.AddBean(&FlowService{})
}

// Ensure json import is used
var _ = json.Marshal
