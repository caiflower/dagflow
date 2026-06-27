package query

import (
	"context"

	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/types"
)

type RedisTaskQueryService struct {
	TaskDao     dao.TaskDAO     `autowired:""`
	SubtaskDao  dao.SubtaskDAO  `autowired:""`
	TaskEdgeDao dao.TaskEdgeDAO `autowired:""`
}

func (s *RedisTaskQueryService) GetTasks(ctx context.Context, taskIDs []string) ([]types.TaskDetail, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	tasks, err := s.TaskDao.GetByIDs(ctx, taskIDs)
	if err != nil {
		logger.Error("[RedisTaskQuery] query tasks failed. err: %v", err)
		return nil, err
	}
	details := make([]types.TaskDetail, 0, len(tasks))
	for _, task := range tasks {
		detail := types.TaskDetail{Task: task}
		if subtasks, _ := s.SubtaskDao.GetByTaskID(ctx, task.ID); subtasks != nil {
			detail.Subtasks = subtasks
		}
		if edges, _ := s.TaskEdgeDao.GetByTaskID(ctx, task.ID); edges != nil {
			detail.Edges = edges
		}
		details = append(details, detail)
	}
	return details, nil
}
