package backup

import (
	"context"
	"fmt"
	"time"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/dagflow/taskx/dao"
	taskmodel "github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/caiflower/dagflow/taskx/types"
	"github.com/uptrace/bun"
)

type SQLBackupManager struct {
	DBClient       dbv1.DB                `autowired:""`
	TaskDao        dao.TaskDAO            `autowired:""`
	TaskBakDao     dao.TaskBakDAO         `autowired:""`
	SubtaskDao     dao.SubtaskDAO         `autowired:""`
	SubtaskBakDao  dao.SubtaskBakDAO      `autowired:""`
	TaskEdgeDao    dao.TaskEdgeDAO        `autowired:""`
	TaskEdgeBakDao dao.TaskEdgeArchiveDAO `autowired:""`
}

func (m *SQLBackupManager) BackupTasks(ctx context.Context, cfg types.BackupConfig) (int, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if len(cfg.FinalStates) == 0 {
		cfg.FinalStates = []string{string(types.TaskFailed), string(types.TaskSucceeded)}
	}

	tasks, err := m.TaskDao.GetOldTasks(ctx, cfg.FinalStates, basic.NewFromTime(time.Now().Add(-cfg.Age)))
	if err != nil {
		logger.Error("[SQLBackupManager] query tasks for backup failed. err: %v", err)
		return 0, err
	}
	if len(tasks) == 0 {
		return 0, nil
	}
	if len(tasks) > cfg.BatchSize {
		tasks = tasks[:cfg.BatchSize]
	}

	taskIDs := make([]string, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}

	tx := dbv1.NewBatchTx(m.DBClient.GetDB())

	taskArchives := make([]taskmodel.TaskBak, len(tasks))
	for i, task := range tasks {
		taskArchives[i] = taskmodel.TaskBak{
			ID: task.ID, RequestID: task.RequestID, TaskName: task.TaskName,
			Input: task.Input, Output: task.Output, Worker: task.Worker,
			Retry: task.Retry, RetryInterval: task.RetryInterval,
			Urgent: task.Urgent, State: task.State, Description: task.Description,
			CreateTime: task.CreateTime, LastRunTime: task.LastRunTime,
			ExecuteTime: task.ExecuteTime, Status: task.Status,
			AffinityType: task.AffinityType, PrimaryWorker: task.PrimaryWorker,
			RollbackStrategy: task.RollbackStrategy,
		}
	}
	tx.Add(func(tx *bun.Tx) error {
		_, err := tx.NewInsert().Model(&taskArchives).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] insert task archive failed. err: %v", err)
			return err
		}
		return nil
	})

	var subtasks []taskmodel.Subtask
	tx.Add(func(tx *bun.Tx) error {
		return tx.NewSelect().Table("subtask").Where("task_id IN (?)", bun.In(taskIDs)).Order("id").Limit(100).Scan(ctx, &subtasks)
	})
	var subtaskArchives []taskmodel.SubtaskBak
	tx.Add(func(tx *bun.Tx) error {
		subtaskArchives = make([]taskmodel.SubtaskBak, len(subtasks))
		for i, s := range subtasks {
			subtaskArchives[i] = taskmodel.SubtaskBak{
				ID: s.ID, TaskID: s.TaskID, PreSubtaskID: s.PreSubtaskID,
				TaskName: s.TaskName, Input: s.Input, Output: s.Output,
				State: s.State, Worker: s.Worker, Retry: s.Retry,
				RetryInterval: s.RetryInterval, Rollback: s.Rollback,
				LastRunTime: s.LastRunTime, Status: s.Status,
			}
		}
		return nil
	})
	tx.Add(func(tx *bun.Tx) error {
		_, err := tx.NewInsert().Model(&subtaskArchives).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] insert subtask archive failed. err: %v", err)
			return err
		}
		return nil
	})

	var taskEdges []taskmodel.TaskEdge
	tx.Add(func(tx *bun.Tx) error {
		return tx.NewSelect().Table("task_edge").Where("task_id IN (?)", bun.In(taskIDs)).Scan(ctx, &taskEdges)
	})
	var edgeArchives []taskmodel.TaskEdgeArchive
	tx.Add(func(tx *bun.Tx) error {
		edgeArchives = make([]taskmodel.TaskEdgeArchive, len(taskEdges))
		for i, e := range taskEdges {
			edgeArchives[i] = taskmodel.TaskEdgeArchive{
				ID: e.ID, TaskID: e.TaskID, FromSubtaskID: e.FromSubtaskID,
				ToSubtaskID: e.ToSubtaskID, EdgeType: e.EdgeType,
				FieldMappings: e.FieldMappings, CreateTime: e.CreateTime,
			}
		}
		return nil
	})
	tx.Add(func(tx *bun.Tx) error {
		_, err := tx.NewInsert().Model(&edgeArchives).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] insert task edge archive failed. err: %v", err)
			return err
		}
		return nil
	})

	tx.Add(func(tx *bun.Tx) error {
		result, err := tx.NewDelete().Table("subtask").Where("task_id IN (?)", bun.In(taskIDs)).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] delete subtasks failed. err: %v", err)
			return err
		}
		rows, _ := result.RowsAffected()
		logger.Info("[SQLBackupManager] deleted %d subtasks for tasks: %v", rows, taskIDs)
		return nil
	})
	tx.Add(func(tx *bun.Tx) error {
		result, err := tx.NewDelete().Table("task_edge").Where("task_id IN (?)", bun.In(taskIDs)).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] delete task_edges failed. err: %v", err)
			return err
		}
		rows, _ := result.RowsAffected()
		logger.Trace("[SQLBackupManager] deleted %d task_edges for tasks: %v", rows, taskIDs)
		return nil
	})
	tx.Add(func(tx *bun.Tx) error {
		result, err := tx.NewDelete().Table("task").Where("id IN (?)", bun.In(taskIDs)).Exec(ctx)
		if err != nil {
			logger.Error("[SQLBackupManager] delete tasks failed. err: %v", err)
			return err
		}
		rows, _ := result.RowsAffected()
		logger.Info("[SQLBackupManager] deleted %d tasks: %v", rows, taskIDs)
		return nil
	})

	if err := tx.Submit(); err != nil {
		logger.Error("[SQLBackupManager] tx submit failed. err: %v", err)
		logger.Warn("[SQLBackupManager] backup operation failed for task IDs: %v", taskIDs)
		return 0, fmt.Errorf("SQLBackupManager tx submit: %w", err)
	}

	logger.Info("[SQLBackupManager] successfully archived and deleted %d tasks: %v", len(taskIDs), taskIDs)
	return len(taskIDs), nil
}
