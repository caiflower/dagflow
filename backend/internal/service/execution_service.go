package service

import (
	"context"
	"fmt"
	"time"

	"github.com/caiflower/common-tools/web/common/e"
	"github.com/caiflower/dagflow/internal/dao"
	"github.com/caiflower/dagflow/internal/dao/model"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/common-tools/pkg/tools"

	"github.com/caiflower/dagflow/internal/converter"
	"github.com/caiflower/dagflow/taskx"
	taskxModel "github.com/caiflower/dagflow/taskx/dao/model"
	t "github.com/caiflower/dagflow/taskx/types"
)

// Execution 执行记录
type Execution struct {
	ID        string       `json:"id"`
	FlowID    string       `json:"flowID"`
	FlowName  string       `json:"flowName"`
	State     string       `json:"state"` // pending, running, succeeded, failed, archived
	StartTime basic.Time   `json:"startTime"`
	EndTime   basic.Time   `json:"endTime"`
	Nodes     []NodeStatus `json:"nodes"`
	TaskID    string       `json:"taskID"` // taskx Task ID
}

// NodeStatus 节点执行状态
type NodeStatus struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Input      string `json:"input"`
	Output     string `json:"output"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	DurationMs int64  `json:"durationMs"`
	NodeType   string `json:"nodeType"` // task, branch, start, end
	Protocol   string `json:"protocol"` // http, grpc, local, mcp
}

// ExecutionService 执行管理服务
type ExecutionService struct {
	FlowDAO            *dao.FlowDAO            `autowired:""`
	ExecutionRecordDAO *dao.ExecutionRecordDAO `autowired:""`
	TaskQuery          t.TaskQueryService      `autowired:""`
}

// RunFlowReq 执行 Flow 请求
type RunFlowReq struct {
	FlowID     string            `json:"flowId" verf:"required"`
	NodeInputs map[string]string `json:"nodeInputs"` // nodeName → JSON input
}

// Run 触发 Flow 执行
func (s *ExecutionService) Run(ctx context.Context, req *RunFlowReq) (*Execution, error) {
	flow, err := s.FlowDAO.GetByID(ctx, req.FlowID)
	if err != nil {
		return nil, e.NewApiError(e.NotFound, fmt.Sprintf("flow %s not found", req.FlowID), err)
	}

	// 构建 taskx.Task
	task, flowNodes, flowEdges, err := converter.FlowToTask(flow, createProvider, req.NodeInputs)
	if err != nil {
		return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
	}

	// 编译 DAG
	if _, err := task.Compile(); err != nil {
		return nil, e.NewApiError(e.InvalidArgument, err.Error(), err)
	}

	// 标记为紧急任务
	task.SetUrgent()

	// 提交到 taskx
	if err := taskx.SubmitTask(ctx, task); err != nil {
		return nil, e.NewApiError(e.Internal, err.Error(), err)
	}

	// 写入执行记录映射表
	execID := tools.GenerateId("exec")
	now := time.Now()
	record := &model.ExecutionRecord{
		ID:        execID,
		FlowID:    flow.ID,
		FlowName:  flow.Name,
		TaskID:    task.GetID(),
		CreatedAt: now,
	}
	if err := s.ExecutionRecordDAO.Insert(ctx, record); err != nil {
		logger.Error("failed to insert execution record: %v", err)
		// 不阻塞返回，记录已提交到 taskx
	}

	// 构建返回结果
	nodes := buildNodeStatuses(flowNodes, flowEdges, nil, nil)
	exec := &Execution{
		ID:        execID,
		FlowID:    flow.ID,
		FlowName:  flow.Name,
		State:     "running",
		StartTime: basic.NewFromTime(now),
		Nodes:     nodes,
		TaskID:    task.GetID(),
	}

	logger.Info("execution %s submitted, taskID=%s, flow=%s", execID, task.GetID(), flow.Name)
	return exec, nil
}

// GetStatus 查询执行状态（从 DB + taskx 实时查询）
func (s *ExecutionService) GetStatus(ctx context.Context, execID string) (*Execution, error) {
	// 1. 从执行记录表查 task_id
	record, err := s.ExecutionRecordDAO.GetByID(ctx, execID)
	if err != nil {
		return nil, e.NewApiError(e.NotFound, fmt.Sprintf("execution %s not found", execID), err)
	}
	if record == nil || record.FlowID == "" {
		return nil, fmt.Errorf("execution %s not found", execID)
	}

	// 2. 查询 Flow 定义（获取节点类型和协议信息）
	flow, err := s.FlowDAO.GetByID(ctx, record.FlowID)
	if err != nil {
		return s.buildArchivedExecution(record), nil
	}
	flowNodes, flowEdges, _ := converter.ParseFlowJSON(flow)

	// 3. 用 TaskQueryService 获取 Task + Subtask（含 archive 补查）
	details, err := s.TaskQuery.GetTasks(ctx, []string{record.TaskID})
	if err != nil || len(details) == 0 {
		// taskx 任务不存在，降级为 archived
		return s.buildArchivedExecution(record), nil
	}

	taskModel := &details[0].Task
	subtasks := details[0].Subtasks

	// 4. 组装返回结果
	exec := &Execution{
		ID:        record.ID,
		FlowID:    record.FlowID,
		FlowName:  record.FlowName,
		State:     mapTaskxState(taskModel.State),
		StartTime: basic.NewFromTime(record.CreatedAt),
		TaskID:    record.TaskID,
	}

	// 设置结束时间
	if exec.State == "succeeded" || exec.State == "failed" {
		if !taskModel.LastRunTime.IsZero() {
			exec.EndTime = taskModel.LastRunTime
		}
	}

	// 构建节点状态
	exec.Nodes = buildNodeStatuses(flowNodes, flowEdges, subtasks, taskModel)

	return exec, nil
}

// ListExecutions 查询执行记录列表（含 taskx 状态）
func (s *ExecutionService) ListExecutions(ctx context.Context, page, pageSize int, flowID string) ([]*Execution, int, error) {
	records, total, err := s.ExecutionRecordDAO.List(ctx, page, pageSize, flowID)
	if err != nil {
		logger.Error("failed to list execution records: %v", err)
		return nil, 0, err
	}
	if len(records) == 0 {
		return []*Execution{}, total, nil
	}

	// 批量查询 taskx Task 状态（含 archive 补查）
	taskIDs := make([]string, 0, len(records))
	for _, r := range records {
		taskIDs = append(taskIDs, r.TaskID)
	}
	details, _ := s.TaskQuery.GetTasks(ctx, taskIDs)
	taskMap := make(map[string]*taskxModel.Task, len(details))
	for i := range details {
		taskMap[details[i].Task.ID] = &details[i].Task
	}

	// 组装返回结果
	result := make([]*Execution, 0, len(records))
	for _, r := range records {
		t := taskMap[r.TaskID]
		state := "archived"
		var endTime basic.Time
		if t != nil {
			state = mapTaskxState(t.State)
			if state == "succeeded" || state == "failed" {
				if !t.LastRunTime.IsZero() {
					endTime = t.LastRunTime
				}
			}
		}
		result = append(result, &Execution{
			ID:        r.ID,
			FlowID:    r.FlowID,
			FlowName:  r.FlowName,
			State:     state,
			StartTime: basic.NewFromTime(r.CreatedAt),
			EndTime:   endTime,
			TaskID:    r.TaskID,
		})
	}
	return result, total, nil
}

// buildArchivedExecution 构建 archived 状态的执行记录
func (s *ExecutionService) buildArchivedExecution(record *model.ExecutionRecord) *Execution {
	return &Execution{
		ID:        record.ID,
		FlowID:    record.FlowID,
		FlowName:  record.FlowName,
		State:     "archived",
		StartTime: basic.NewFromTime(record.CreatedAt),
		TaskID:    record.TaskID,
	}
}

// buildNodeStatuses 从 Flow 节点定义和 taskx Subtask 状态组装 NodeStatus 列表（按 DAG 拓扑序输出）
func buildNodeStatuses(flowNodes []converter.FlowNode, flowEdges []converter.FlowEdge, subtasks []taskxModel.Subtask, taskModel *taskxModel.Task) []NodeStatus {
	// 拓扑排序：保证节点按 DAG 顺序输出
	sorted := topoSort(flowNodes, flowEdges)

	// 构建 subtask name → state 映射
	subtaskMap := make(map[string]*taskxModel.Subtask, len(subtasks))
	for i := range subtasks {
		subtaskMap[subtasks[i].TaskName] = &subtasks[i]
	}

	nodes := make([]NodeStatus, 0, len(sorted))
	for _, fn := range sorted {
		ns := NodeStatus{
			ID:       fn.ID,
			Name:     fn.Name,
			State:    "pending",
			NodeType: fn.Type,
			Protocol: fn.Protocol,
		}

		// start 节点视为立即成功
		if fn.Type == "start" {
			ns.State = "succeeded"
		}

		// 从 taskx Subtask 获取状态和详情
		// subtask TaskName = flow 节点 ID（converter 转换规则）
		st, ok := subtaskMap[fn.ID]
		if !ok {
			st, ok = subtaskMap[fn.Name]
		}
		if ok {
			ns.State = mapTaskxState(st.State)
			ns.Input = st.Input

			// 解析 Output JSON（taskx 存储格式: {"output":"...","err":"..."}）
			if st.Output != "" {
				var output taskx.Output
				if err := tools.Unmarshal([]byte(st.Output), &output); err == nil {
					if output.Err != "" {
						ns.Output = output.Err
					} else {
						ns.Output = output.Output
					}
				} else {
					ns.Output = st.Output
				}
			}

			// 时间信息
			if !st.LastRunTime.IsZero() {
				ns.StartTime = st.LastRunTime.String()
				// 如果节点已终态，用 task 级别的 LastRunTime 近似 endTime
				if (ns.State == "succeeded" || ns.State == "failed") && taskModel != nil && !taskModel.LastRunTime.IsZero() {
					ns.EndTime = taskModel.LastRunTime.String()
					startMs := st.LastRunTime.Time().UnixMilli()
					endMs := taskModel.LastRunTime.Time().UnixMilli()
					if endMs > startMs {
						ns.DurationMs = endMs - startMs
					}
				}
			}
		}

		// end 节点：如果整体任务成功，end 也标记为 succeeded
		if fn.Type == "end" && taskModel != nil && mapTaskxState(taskModel.State) == "succeeded" {
			ns.State = "succeeded"
		}

		nodes = append(nodes, ns)
	}
	return nodes
}

// topoSort 对 flowNodes 按 DAG 边做拓扑排序（Kahn 算法），无法排序时回退原序
func topoSort(flowNodes []converter.FlowNode, flowEdges []converter.FlowEdge) []converter.FlowNode {
	if len(flowNodes) == 0 {
		return flowNodes
	}
	// 构建 node id → index 映射
	idxMap := make(map[string]int, len(flowNodes))
	for i, n := range flowNodes {
		idxMap[n.ID] = i
	}
	// 入度 + 邻接表
	inDegree := make([]int, len(flowNodes))
	adj := make([][]int, len(flowNodes))
	for _, e := range flowEdges {
		src, ok1 := idxMap[e.Source]
		dst, ok2 := idxMap[e.Target]
		if ok1 && ok2 {
			adj[src] = append(adj[src], dst)
			inDegree[dst]++
		}
	}
	// BFS
	queue := make([]int, 0, len(flowNodes))
	for i, d := range inDegree {
		if d == 0 {
			queue = append(queue, i)
		}
	}
	result := make([]converter.FlowNode, 0, len(flowNodes))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, flowNodes[cur])
		for _, next := range adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	// 如果拓扑排序未覆盖所有节点（有环），回退原序
	if len(result) < len(flowNodes) {
		return flowNodes
	}
	return result
}

// mapTaskxState 将 taskx 状态映射为前端可展示的状态
func mapTaskxState(state string) string {
	switch state {
	case string(t.TaskPending):
		return "pending"
	case string(t.TaskRunning):
		return "running"
	case string(t.TaskSucceeded):
		return "succeeded"
	case string(t.TaskFailed):
		return "failed"
	case string(t.TaskSkipped):
		return "skipped"
	default:
		return state
	}
}

// Retry 重试失败的执行
func (s *ExecutionService) Retry(ctx context.Context, execID string) (int, error) {
	// 1. 从执行记录表查 task_id
	record, err := s.ExecutionRecordDAO.GetByID(ctx, execID)
	if err != nil {
		return 0, e.NewApiError(e.NotFound, fmt.Sprintf("execution %s not found", execID), err)
	}
	if record == nil || record.TaskID == "" {
		return 0, fmt.Errorf("execution %s not found or has no task ID", execID)
	}

	// 2. 调用 taskx.RetryTask
	count, err := taskx.RetryTask(ctx, record.TaskID)
	if err != nil {
		return 0, e.NewApiError(e.InvalidArgument, err.Error(), err)
	}

	logger.Info("execution %s retry initiated, taskID=%s, reset %d subtasks", execID, record.TaskID, count)
	return count, nil
}

// InitExec 注册 ExecutionService bean
func InitExec() {
	bean.AddBean(&ExecutionService{})
}
