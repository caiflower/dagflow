package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/logger"

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/internal/model/dao"
	"github.com/caiflower/dagflow/taskx"
	taskxDAO "github.com/caiflower/dagflow/taskx/dao"
)

// Execution 执行记录
type Execution struct {
	ID        string       `json:"id"`
	FlowID    int64        `json:"flowID"`
	FlowName  string       `json:"flowName"`
	State     string       `json:"state"` // pending, running, succeeded, failed
	StartTime basic.Time   `json:"startTime"`
	EndTime   basic.Time   `json:"endTime"`
	Nodes     []NodeStatus `json:"nodes"`

	// 内部字段
	taskID   string            // taskx Task ID
	nodeInfo map[string]string // name → nodeType (start/end/task/branch)
}

// NodeStatus 节点执行状态
type NodeStatus struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// ExecutionService 执行管理服务
type ExecutionService struct {
	FlowDAO    *dao.FlowDAO        `autowired:""`
	TaskDAO    taskxDAO.TaskDAO    `autowired:""`
	SubtaskDAO taskxDAO.SubtaskDAO `autowired:""`

	mu         sync.RWMutex
	executions map[string]*Execution
}

// RunFlowReq 执行 Flow 请求
type RunFlowReq struct {
	FlowID int64 `json:"flowId" verf:"required"`
}

// Run 触发 Flow 执行
func (s *ExecutionService) Run(ctx context.Context, req *RunFlowReq) (*Execution, error) {
	flow, err := s.FlowDAO.GetByID(ctx, req.FlowID)
	if err != nil {
		return nil, fmt.Errorf("flow not found: %w", err)
	}

	// 解析节点和边（用于构建 Execution 记录）
	flowNodes, _, err := converter.ParseFlowJSON(flow)
	if err != nil {
		return nil, fmt.Errorf("parse flow: %w", err)
	}

	// 使用 FlowToTask 构建 taskx.Task，使用 createProvider 作为 provider 工厂
	task, err := converter.FlowToTask(flow, createProvider)
	if err != nil {
		return nil, fmt.Errorf("build task: %w", err)
	}

	// 编译 DAG（校验环、起始/终止节点等）
	if _, err := task.Compile(); err != nil {
		return nil, fmt.Errorf("compile task: %w", err)
	}

	// 标记为紧急任务，提交后立即调度
	task.SetUrgent()

	// 提交到 taskx（持久化到 DB，触发集群调度）
	if err := taskx.SubmitTask(ctx, task); err != nil {
		return nil, fmt.Errorf("submit task: %w", err)
	}

	execID := fmt.Sprintf("exec-%d-%d", flow.ID, time.Now().UnixMilli())
	now := basic.NewFromTime(time.Now())

	// 构建节点信息（name → type 映射 + 初始状态）
	nodeInfo := make(map[string]string)
	var nodes []NodeStatus
	for _, n := range flowNodes {
		nodeInfo[n.Name] = n.Type
		state := "pending"
		if n.Type == "start" {
			state = "succeeded" // start 节点视为立即成功
		}
		nodes = append(nodes, NodeStatus{
			ID:    n.ID,
			Name:  n.Name,
			State: state,
		})
	}

	exec := &Execution{
		ID:        execID,
		FlowID:    flow.ID,
		FlowName:  flow.Name,
		State:     "running",
		StartTime: now,
		Nodes:     nodes,
		taskID:    task.GetID(),
		nodeInfo:  nodeInfo,
	}

	s.mu.Lock()
	if s.executions == nil {
		s.executions = make(map[string]*Execution)
	}
	s.executions[execID] = exec
	s.mu.Unlock()

	logger.Info("execution %s submitted, taskID=%s, flow=%s", execID, task.GetID(), flow.Name)
	return exec, nil
}

// GetStatus 查询执行状态（从 taskx DB 查询真实状态）
func (s *ExecutionService) GetStatus(ctx context.Context, execID string) (*Execution, error) {
	s.mu.RLock()
	exec, ok := s.executions[execID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("execution %s not found", execID)
	}

	// 查询 taskx Task 状态
	taskModel, err := s.TaskDAO.GetByID(ctx, exec.taskID)
	if err != nil {
		// DB 查询失败，返回内存中的状态
		return s.snapshot(exec), nil
	}

	// 查询所有 Subtask 状态
	subtasks, err := s.SubtaskDAO.GetByTaskID(ctx, exec.taskID)
	if err != nil {
		return s.snapshot(exec), nil
	}

	// 构建 subtask name → state 映射
	subtaskStates := make(map[string]string)
	for _, st := range subtasks {
		subtaskStates[st.TaskName] = mapTaskxState(st.State)
	}

	// 构建返回结果
	snapshot := s.snapshot(exec)
	snapshot.State = mapTaskxState(taskModel.State)

	// 更新节点状态
	for i := range snapshot.Nodes {
		node := &snapshot.Nodes[i]
		nodeType := exec.nodeInfo[node.Name]
		if nodeType == "start" || nodeType == "end" {
			// start/end 节点：如果所有 task 节点都完成，end 也标记为 succeeded
			if nodeType == "end" && snapshot.State == "succeeded" {
				node.State = "succeeded"
			}
			continue
		}
		if st, ok := subtaskStates[node.Name]; ok {
			node.State = st
		}
	}

	// 设置结束时间
	if snapshot.State == "succeeded" || snapshot.State == "failed" {
		if snapshot.EndTime.IsZero() {
			snapshot.EndTime = basic.NewFromTime(time.Now())
			// 更新内存中的 EndTime
			s.mu.Lock()
			exec.EndTime = snapshot.EndTime
			exec.State = snapshot.State
			s.mu.Unlock()
		}
	}

	return snapshot, nil
}

// ListExecutions 列出所有执行记录
func (s *ExecutionService) ListExecutions(ctx context.Context) []*Execution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Execution, 0, len(s.executions))
	for _, exec := range s.executions {
		result = append(result, s.snapshotUnsafe(exec))
	}
	return result
}

// snapshot 返回执行记录的快照（带锁保护）
func (s *ExecutionService) snapshot(exec *Execution) *Execution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotUnsafe(exec)
}

// snapshotUnsafe 返回执行记录的快照（调用者需持有读锁）
func (s *ExecutionService) snapshotUnsafe(exec *Execution) *Execution {
	snap := &Execution{
		ID:        exec.ID,
		FlowID:    exec.FlowID,
		FlowName:  exec.FlowName,
		State:     exec.State,
		StartTime: exec.StartTime,
		EndTime:   exec.EndTime,
		Nodes:     make([]NodeStatus, len(exec.Nodes)),
	}
	copy(snap.Nodes, exec.Nodes)
	return snap
}

// mapTaskxState 将 taskx 状态映射为前端可展示的状态
func mapTaskxState(state string) string {
	switch state {
	case string(taskx.TaskPending):
		return "pending"
	case string(taskx.TaskRunning), string(taskx.TaskSubtaskRunning):
		return "running"
	case string(taskx.TaskSucceeded):
		return "succeeded"
	case string(taskx.TaskFailed):
		return "failed"
	case string(taskx.TaskSkipped):
		return "skipped"
	default:
		return state
	}
}

// InitExec 注册 ExecutionService bean
func InitExec() {
	bean.AddBean(&ExecutionService{})
}
