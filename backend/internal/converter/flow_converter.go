package converter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caiflower/dagflow/internal/dao/model"
	"github.com/caiflower/dagflow/taskx"
	"github.com/caiflower/dagflow/taskx/executor"
)

// FlowNode 前端 Flow 节点定义
type FlowNode struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`     // task, branch, start, end
	Protocol string         `json:"protocol"` // http, grpc, local, mcp
	Config   map[string]any `json:"config"`   // 协议配置
	Position *Position      `json:"position,omitempty"`
}

// Position 节点位置（前端编辑器用）
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// FlowEdge 前端 Flow 边定义
type FlowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // control, data, control+data
	Expr   string `json:"expr"` // 条件表达式（分支边）
}

// branchInfo holds pending branch node data for wiring after all subtasks are created
type branchInfo struct {
	node     FlowNode
	provider executor.ExecutorProvider
}

// validateBranchProvider checks that a branch provider returns string (ID or taskName)
func validateBranchProvider(p executor.ExecutorProvider) error {
	result, err := p.Execute(context.Background(), &executor.TaskData{SubTaskId: "validate"})
	if err != nil {
		return fmt.Errorf("branch provider execution failed: %w", err)
	}
	if _, ok := result.(string); !ok {
		return fmt.Errorf("branch provider must return string (ID or taskName), got %T", result)
	}
	return nil
}

// isLocalProvider returns true for providers that can be validated at conversion time
func isLocalProvider(p executor.ExecutorProvider) bool {
	proto := p.Protocol()
	return proto == "local" || proto == "branch" || proto == ""
}

// isBranchNode checks if a node ID belongs to a branch-type node
func isBranchNode(nodes []FlowNode, id string) bool {
	for _, n := range nodes {
		if n.ID == id && n.Type == "branch" {
			return true
		}
	}
	return false
}

// FlowToTask 将 Flow 转换为 taskx.Task
// providerFactory: 根据协议名和配置创建 ExecutorProvider
// nodeInputs: 可选的节点输入参数 (nodeName → JSON input)
func FlowToTask(flow *model.Flow, providerFactory func(protocol string, config map[string]any) (executor.ExecutorProvider, error), nodeInputs map[string]string) (*taskx.Task, error) {
	nodes, edges, err := ParseFlowJSON(flow)
	if err != nil {
		return nil, err
	}
	return FlowToTaskWithNodes(flow.Name, nodes, edges, providerFactory, nodeInputs)
}

// FlowToTaskWithNodes creates a taskx.Task directly from nodes and edges (for testing)
func FlowToTaskWithNodes(name string, nodes []FlowNode, edges []FlowEdge, providerFactory func(string, map[string]any) (executor.ExecutorProvider, error), nodeInputs map[string]string) (*taskx.Task, error) {
	task := taskx.NewTask(name)

	var pendingBranches []branchInfo
	subtaskMap := make(map[string]*taskx.Subtask)

	for _, n := range nodes {
		if n.Type == "start" || n.Type == "end" {
			continue
		}

		if n.Type == "branch" {
			provider, err := providerFactory(n.Protocol, n.Config)
			if err != nil {
				return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
			}
			// Validate return type for local providers
			if isLocalProvider(provider) {
				if err := validateBranchProvider(provider); err != nil {
					return nil, fmt.Errorf("branch node %s: %w", n.Name, err)
				}
			}
			pendingBranches = append(pendingBranches, branchInfo{node: n, provider: provider})
			continue
		}

		provider, err := providerFactory(n.Protocol, n.Config)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", n.Name, err)
		}

		st := taskx.NewSubtask(n.Name, provider)
		if input, ok := nodeInputs[n.Name]; ok && input != "" {
			st.SetInput(input)
		}
		subtaskMap[n.ID] = st
		if err := task.AddSubtask(st); err != nil {
			return nil, fmt.Errorf("add subtask %s: %w", n.Name, err)
		}
	}

	// Build edges (skip edges involving branch nodes — AddBranch handles wiring)
	for _, e := range edges {
		if isBranchNode(nodes, e.Source) || isBranchNode(nodes, e.Target) {
			continue
		}
		src, ok1 := subtaskMap[e.Source]
		dst, ok2 := subtaskMap[e.Target]
		if !ok1 || !ok2 {
			continue
		}
		switch e.Type {
		case "data":
			if err := task.AddDataEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add data edge %s->%s: %w", e.Source, e.Target, err)
			}
		case "control+data":
			if err := task.AddEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add edge %s->%s: %w", e.Source, e.Target, err)
			}
		default:
			if err := task.AddControlEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add control edge %s->%s: %w", e.Source, e.Target, err)
			}
		}
	}

	// Wire branch nodes: create branch subtasks via AddBranch with protocol provider
	for _, bi := range pendingBranches {
		// Find predecessors (nodes with edges to this branch node)
		var predIDs []string
		for _, e := range edges {
			if e.Target == bi.node.ID {
				predIDs = append(predIDs, e.Source)
			}
		}
		if len(predIDs) == 0 {
			continue
		}

		// Find successor names
		succNames := make(map[string]bool)
		for _, e := range edges {
			if e.Source == bi.node.ID {
				succNames[e.Target] = true
			}
		}
		if len(succNames) < 2 {
			continue
		}

		// Resolve successor names from target node IDs
		resolvedEndNodes := make(map[string]bool, len(succNames))
		for targetID := range succNames {
			for _, n := range nodes {
				if n.ID == targetID && n.Type == "task" {
					resolvedEndNodes[n.Name] = true
					break
				}
			}
		}

		// Create branch for each predecessor
		for _, predID := range predIDs {
			predSt, ok := subtaskMap[predID]
			if !ok {
				continue
			}
			if err := task.AddBranch(predSt, taskx.NewBranch(bi.provider, resolvedEndNodes)); err != nil {
				return nil, fmt.Errorf("add branch %s: %w", bi.node.Name, err)
			}
		}
	}

	return task, nil
}

// ParseFlowJSON 解析 Flow 的节点和边 JSON
func ParseFlowJSON(flow *model.Flow) ([]FlowNode, []FlowEdge, error) {
	var nodes []FlowNode
	var edges []FlowEdge

	if flow.NodesJSON != "" {
		if err := json.Unmarshal([]byte(flow.NodesJSON), &nodes); err != nil {
			return nil, nil, fmt.Errorf("parse nodes JSON: %w", err)
		}
	}
	if flow.EdgesJSON != "" {
		if err := json.Unmarshal([]byte(flow.EdgesJSON), &edges); err != nil {
			return nil, nil, fmt.Errorf("parse edges JSON: %w", err)
		}
	}
	return nodes, edges, nil
}

// ValidateFlow 校验 Flow 定义合法性
func ValidateFlow(nodes []FlowNode, edges []FlowEdge) error {
	if len(nodes) == 0 {
		return fmt.Errorf("flow must have at least one node")
	}

	nodeIDs := make(map[string]bool)
	for _, n := range nodes {
		if n.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		if n.Name == "" {
			return fmt.Errorf("node name cannot be empty")
		}
		if nodeIDs[n.ID] {
			return fmt.Errorf("duplicate node ID: %s", n.ID)
		}
		nodeIDs[n.ID] = true
	}

	for _, e := range edges {
		if !nodeIDs[e.Source] {
			return fmt.Errorf("edge source %q not found", e.Source)
		}
		if !nodeIDs[e.Target] {
			return fmt.Errorf("edge target %q not found", e.Target)
		}
	}

	return nil
}

// NodesToJSON 将节点列表序列化为 JSON
func NodesToJSON(nodes []FlowNode) string {
	b, _ := json.Marshal(nodes)
	return string(b)
}

// EdgesToJSON 将边列表序列化为 JSON
func EdgesToJSON(edges []FlowEdge) string {
	b, _ := json.Marshal(edges)
	return string(b)
}
