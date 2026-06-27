package backup

import (
	"context"
	"time"

	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/common-tools/pkg/logger"
	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/redisd"
	"github.com/caiflower/dagflow/taskx/types"
)

type RedisBackupManager struct {
	RedisClient v2.RedisClient `autowired:""`
	TaskDao     dao.TaskDAO    `autowired:""`
}

func (m *RedisBackupManager) BackupTasks(ctx context.Context, cfg types.BackupConfig) (int, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if len(cfg.FinalStates) == 0 {
		cfg.FinalStates = []string{string(types.TaskFailed), string(types.TaskSucceeded)}
	}

	c := m.RedisClient.Cmd()
	tasks, err := m.TaskDao.GetOldTasks(ctx, cfg.FinalStates, basic.NewFromTime(time.Now().Add(-cfg.Age)))
	if err != nil {
		logger.Error("[RedisBackupManager] query tasks for cleanup failed. err: %v", err)
		return 0, err
	}
	if len(tasks) == 0 {
		return 0, nil
	}
	if len(tasks) > cfg.BatchSize {
		tasks = tasks[:cfg.BatchSize]
	}

	prefix := redisd.DefaultKeyConfig().Prefix
	pipe := c.Pipeline()
	for _, task := range tasks {
		taskID := task.ID
		pipe.Del(ctx, redisd.TaskKey(prefix, taskID))
		pipe.SRem(ctx, redisd.TodoSetKey(prefix))
		subtaskIDs, _ := c.SMembers(ctx, redisd.SubtaskIndexKey(prefix, taskID)).Result()
		for _, sid := range subtaskIDs {
			pipe.Del(ctx, redisd.SubtaskKey(prefix, sid))
		}
		pipe.Del(ctx, redisd.SubtaskIndexKey(prefix, taskID))
		pipe.Del(ctx, redisd.EdgeIndexKey(prefix, taskID))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logger.Error("[RedisBackupManager] pipeline exec failed. err: %v", err)
		return 0, err
	}

	logger.Info("[RedisBackupManager] cleaned up %d expired tasks", len(tasks))
	return len(tasks), nil
}
