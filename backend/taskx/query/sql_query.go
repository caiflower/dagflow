package query

import (
	"context"

	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/caiflower/dagflow/taskx/types"
)

type SQLTaskQueryService struct {
	TaskDao        dao.TaskDAO            `autowired:""`
	TaskBakDao     dao.TaskBakDAO         `autowired:""`
	SubtaskDao     dao.SubtaskDAO         `autowired:""`
	SubtaskBakDao  dao.SubtaskBakDAO      `autowired:""`
	TaskEdgeDao    dao.TaskEdgeDAO        `autowired:""`
	TaskEdgeBakDao dao.TaskEdgeArchiveDAO `autowired:""`
}

func (s *SQLTaskQueryService) GetTasks(ctx context.Context, taskIDs []string) ([]types.TaskDetail, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}

	tasks, err := s.TaskDao.GetByIDs(ctx, taskIDs)
	if err != nil {
		logger.Error("[SQLTaskQuery] query main table failed. err: %v", err)
		return nil, err
	}

	foundIDs := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		foundIDs[t.ID] = true
	}
	var missingIDs []string
	for _, id := range taskIDs {
		if !foundIDs[id] {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		bakTasks, err := s.TaskBakDao.GetByIDs(ctx, missingIDs)
		if err != nil {
			logger.Error("[SQLTaskQuery] query archive table failed. err: %v", err)
		} else {
			for _, bt := range bakTasks {
				tasks = append(tasks, taskBakToTask(&bt))
			}
		}
	}

	details := make([]types.TaskDetail, 0, len(tasks))
	for _, task := range tasks {
		detail := types.TaskDetail{Task: task}
		if subtasks, _ := s.SubtaskDao.GetByTaskID(ctx, task.ID); subtasks != nil {
			detail.Subtasks = subtasks
		} else if bakSubtasks, _ := s.SubtaskBakDao.GetByTaskID(ctx, task.ID); bakSubtasks != nil {
			for _, bs := range bakSubtasks {
				detail.Subtasks = append(detail.Subtasks, subtaskBakToSubtask(&bs))
			}
		}
		if edges, _ := s.TaskEdgeDao.GetByTaskID(ctx, task.ID); edges != nil {
			detail.Edges = edges
		} else if bakEdges, _ := s.TaskEdgeBakDao.GetByTaskID(ctx, task.ID); bakEdges != nil {
			for _, be := range bakEdges {
				detail.Edges = append(detail.Edges, taskEdgeBakToTaskEdge(&be))
			}
		}
		details = append(details, detail)
	}
	return details, nil
}

func taskBakToTask(bak *model.TaskBak) model.Task {
	return model.Task{
		ID: bak.ID, RequestID: bak.RequestID, TaskName: bak.TaskName,
		Input: bak.Input, Output: bak.Output, Worker: bak.Worker,
		Retry: bak.Retry, RetryInterval: bak.RetryInterval,
		Urgent: bak.Urgent, State: bak.State, Description: bak.Description,
		CreateTime: bak.CreateTime, LastRunTime: bak.LastRunTime,
		ExecuteTime: bak.ExecuteTime, Status: bak.Status,
		AffinityType: bak.AffinityType, PrimaryWorker: bak.PrimaryWorker,
		RollbackStrategy: bak.RollbackStrategy,
	}
}

func subtaskBakToSubtask(bak *model.SubtaskBak) model.Subtask {
	return model.Subtask{
		ID: bak.ID, TaskID: bak.TaskID, PreSubtaskID: bak.PreSubtaskID,
		TaskName: bak.TaskName, TriggerMode: bak.TriggerMode,
		Priority: bak.Priority, Timeout: bak.Timeout,
		Input: bak.Input, Output: bak.Output, State: bak.State,
		Worker: bak.Worker, Retry: bak.Retry, RetryInterval: bak.RetryInterval,
		Rollback: bak.Rollback, LastRunTime: bak.LastRunTime,
		Status: bak.Status, Settings: bak.Settings,
	}
}

func taskEdgeBakToTaskEdge(bak *model.TaskEdgeArchive) model.TaskEdge {
	return model.TaskEdge{
		ID: bak.ID, TaskID: bak.TaskID, FromSubtaskID: bak.FromSubtaskID,
		ToSubtaskID: bak.ToSubtaskID, EdgeType: bak.EdgeType,
		FieldMappings: bak.FieldMappings, CreateTime: bak.CreateTime,
	}
}
