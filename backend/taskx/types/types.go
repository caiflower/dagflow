package types

import (
	"context"
	"time"

	"github.com/caiflower/dagflow/taskx/dao/model"
)

// BackupConfig 冷数据备份配置
type BackupConfig struct {
	Age         time.Duration // 超龄阈值：MySQL 默认 168h，Redis 默认 24h
	BatchSize   int           // 每批处理数量，默认 100
	FinalStates []string      // 终态列表，默认 ["failed", "succeeded"]
}

// BackupManager 冷数据备份接口，由 MySQL 和 Redis 分别实现
type BackupManager interface {
	BackupTasks(ctx context.Context, cfg BackupConfig) (int, error)
}

// TaskDetail 任务详情，包含 Task + Subtask 列表 + TaskEdge 列表
type TaskDetail struct {
	Task     model.Task
	Subtasks []model.Subtask
	Edges    []model.TaskEdge
}

// TaskQueryService 业务层任务查询接口，统一从主表/归档表获取任务详情
type TaskQueryService interface {
	GetTasks(ctx context.Context, taskIDs []string) ([]TaskDetail, error)
}
