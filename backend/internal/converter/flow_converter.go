package converter

import (
	"encoding/json"
	"fmt"

	"github.com/caiflower/dagflow/internal/model"
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

// FlowToTask 将 Flow 转换为 taskx.Task
// providerFactory: 根据协议名和配置创建 ExecutorProvider
func FlowToTask(flow *model.Flow, providerFactory func(protocol string, config map[string]any) (executor.ExecutorProvider, error)) (*taskx.Task, error) {
	nodes, edges, err := ParseFlowJSON(flow)
	if err != nil {
		return nil, err
	}

	task := taskx.NewTask(flow.Name)

	// 构建子任务映射
	subtaskMap := make(map[string]*taskx.Subtask)
	for _, n := range nodes {
		if n.Type == "start" || n.Type == "end" {
			continue
		}
		provider, err := providerFactory(n.Protocol, n.Config)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", n.Name, err)
		}
		st := taskx.NewSubtask(n.Name, provider)
		subtaskMap[n.ID] = st
		if err := task.AddSubtask(st); err != nil {
			return nil, fmt.Errorf("add subtask %s: %w", n.Name, err)
		}
	}

	// 构建边
	for _, e := range edges {
		src, ok1 := subtaskMap[e.Source]
		dst, ok2 := subtaskMap[e.Target]
		if !ok1 || !ok2 {
			continue // 跳过 start/end 节点的边
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
		default: // control
			if err := task.AddControlEdge(src, dst); err != nil {
				return nil, fmt.Errorf("add control edge %s->%s: %w", e.Source, e.Target, err)
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
