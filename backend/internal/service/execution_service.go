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
	"github.com/caiflower/dagflow/taskx/executor"
)

// Execution 执行记录（内存存储，后续可持久化）
type Execution struct {
	ID        string         `json:"id"`
	FlowID    int64          `json:"flowID"`
	FlowName  string         `json:"flowName"`
	State     string         `json:"state"` // pending, running, succeeded, failed
	StartTime basic.Time     `json:"startTime"`
	EndTime   basic.Time     `json:"endTime"`
	Task      *taskx.Task    `json:"-"`
	Nodes     []NodeStatus   `json:"nodes"`
}

// NodeStatus 节点执行状态
type NodeStatus struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// ExecutionService 执行管理服务
type ExecutionService struct {
	FlowDAO *dao.FlowDAO `autowired:""`

	mu          sync.RWMutex
	executions  map[string]*Execution
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

	// 转换 Flow 为 taskx.Task（使用空 provider factory 作为占位）
	task, err := converter.FlowToTask(flow, func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		// TODO: 在后续变更中实现真实的 provider 创建
		return &stubProvider{protocol: protocol}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("convert flow to task: %w", err)
	}

	execID := fmt.Sprintf("exec-%d-%d", flow.ID, time.Now().UnixMilli())
	exec := &Execution{
		ID:        execID,
		FlowID:    flow.ID,
		FlowName:  flow.Name,
		State:     "pending",
		StartTime: basic.NewFromTime(time.Now()),
		Task:      task,
	}

	// 解析节点状态
	nodes, _, _ := converter.ParseFlowJSON(flow)
	for _, n := range nodes {
		exec.Nodes = append(exec.Nodes, NodeStatus{
			ID:    n.ID,
			Name:  n.Name,
			State: "pending",
		})
	}

	s.mu.Lock()
	if s.executions == nil {
		s.executions = make(map[string]*Execution)
	}
	s.executions[execID] = exec
	s.mu.Unlock()

	// 异步执行（简化版）
	go s.execute(exec)

	return exec, nil
}

// execute 执行 task（简化版）
func (s *ExecutionService) execute(exec *Execution) {
	exec.State = "running"
	logger.Info(fmt.Sprintf("execution %s started for flow %s", exec.ID, exec.FlowName))

	// TODO: 在后续变更中实现真实的 DAG 执行
	// 当前仅模拟成功
	time.Sleep(100 * time.Millisecond)

	s.mu.Lock()
	for i := range exec.Nodes {
		exec.Nodes[i].State = "succeeded"
	}
	exec.State = "succeeded"
	exec.EndTime = basic.NewFromTime(time.Now())
	s.mu.Unlock()

	logger.Info(fmt.Sprintf("execution %s completed: %s", exec.ID, exec.State))
}

// GetStatus 查询执行状态
func (s *ExecutionService) GetStatus(ctx context.Context, execID string) (*Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, ok := s.executions[execID]
	if !ok {
		return nil, fmt.Errorf("execution %s not found", execID)
	}
	return exec, nil
}

// ListExecutions 列出所有执行记录
func (s *ExecutionService) ListExecutions(ctx context.Context) []*Execution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Execution, 0, len(s.executions))
	for _, exec := range s.executions {
		result = append(result, exec)
	}
	return result
}

// InitExec 注册 ExecutionService bean
func InitExec() {
	bean.AddBean(&ExecutionService{})
}

// stubProvider 占位执行器
type stubProvider struct {
	protocol string
}

func (p *stubProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	return map[string]string{"status": "stub", "protocol": p.protocol}, nil
}

func (p *stubProvider) Protocol() executor.Protocol {
	return executor.Protocol(p.protocol)
}
