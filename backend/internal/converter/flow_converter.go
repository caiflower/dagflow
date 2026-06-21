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

// FlowToTask 将 Flow 转换为 taskx.Task
// providerFactory: 根据协议名和配置创建 ExecutorProvider
// nodeInputs: 可选的节点输入参数 (nodeName → JSON input)
func FlowToTask(flow *model.Flow, providerFactory func(protocol string, config map[string]any) (executor.ExecutorProvider, error), nodeInputs map[string]string) (*taskx.Task, error) {
	nodes, edges, err := ParseFlowJSON(flow)
	if err != nil {
		return nil, err
	}

	task := taskx.NewTask(flow.Name)

	// 构建子任务映射（task + branch 节点均创建 subtask）
	subtaskMap := make(map[string]*taskx.Subtask)
	for _, n := range nodes {
		if n.Type == "start" || n.Type == "end" {
			continue
		}

		var provider executor.ExecutorProvider
		if n.Type == "branch" {
			// 分支节点使用透传 provider，仅做路由决策
			provider = &branchPassthroughProvider{node: n, edges: edges}
		} else {
			provider, err = providerFactory(n.Protocol, n.Config)
			if err != nil {
				return nil, fmt.Errorf("node %s: %w", n.Name, err)
			}
		}

		st := taskx.NewSubtask(n.Name, provider)
		// 如果用户提供了该节点的输入，设置到 subtask
		if input, ok := nodeInputs[n.Name]; ok && input != "" {
			st.SetInput(input)
		}
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

	// 为分支节点注册 AddBranch
	for _, n := range nodes {
		if n.Type != "branch" {
			continue
		}
		branchSt, ok := subtaskMap[n.ID]
		if !ok {
			continue
		}

		// 收集分支节点的出边目标
		endNodes := make(map[string]bool)
		for _, e := range edges {
			if e.Source == n.ID {
				if targetSt, ok := subtaskMap[e.Target]; ok {
					endNodes[targetSt.GetName()] = true
				}
			}
		}
		if len(endNodes) < 2 {
			continue // 分支节点至少需要 2 个出边目标
		}

		// 构建分支条件：基于边的 expr 表达式选择目标
		branchEdges := collectOutgoingEdges(n.ID, edges)
		if err := task.AddBranch(branchSt, &taskx.Branch{
			Condition: makeBranchCondition(branchEdges),
			EndNodes:  endNodes,
		}); err != nil {
			return nil, fmt.Errorf("add branch for %s: %w", n.Name, err)
		}
	}

	return task, nil
}

// collectOutgoingEdges 收集节点的出边
func collectOutgoingEdges(nodeID string, edges []FlowEdge) []FlowEdge {
	var result []FlowEdge
	for _, e := range edges {
		if e.Source == nodeID {
			result = append(result, e)
		}
	}
	return result
}

// makeBranchCondition 基于出边创建分支条件函数
// 当前实现：选择第一个目标（默认分支），后续可扩展 expr 表达式求值
func makeBranchCondition(outEdges []FlowEdge) func(ctx interface{}, input any) (string, error) {
	return func(ctx interface{}, input any) (string, error) {
		// 尝试基于 expr 匹配
		if input != nil {
			for _, e := range outEdges {
				if e.Expr != "" && matchExpr(e.Expr, input) {
					return e.Target, nil
				}
			}
		}
		// 默认选择第一个目标
		if len(outEdges) > 0 {
			return outEdges[0].Target, nil
		}
		return "", fmt.Errorf("branch has no outgoing edges")
	}
}

// matchExpr 简单的表达式匹配（后续可扩展为完整的表达式引擎）
// 当前支持：input 中包含 expr 指定的 key 且值为 truthy 时匹配
func matchExpr(expr string, input any) bool {
	m, ok := input.(map[string]any)
	if !ok {
		return false
	}
	if val, exists := m[expr]; exists {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v != ""
		default:
			return v != nil
		}
	}
	return false
}

// ===== 分支透传 Provider =====

// branchPassthroughProvider 分支节点执行器
// 透传输入数据，分支路由决策由 taskx AddBranch 的 Condition 处理
type branchPassthroughProvider struct {
	node  FlowNode
	edges []FlowEdge
}

func (p *branchPassthroughProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	// 透传：将输入原样传递给分支条件判断
	if data != nil && data.Input != "" {
		var parsed any
		if err := json.Unmarshal([]byte(data.Input), &parsed); err == nil {
			return parsed, nil
		}
		return data.Input, nil
	}
	return map[string]string{"branch": p.node.Name}, nil
}

func (p *branchPassthroughProvider) Protocol() executor.Protocol {
	return "branch"
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
