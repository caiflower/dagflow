package dao

import (
	"context"

	"github.com/caiflower/dagflow/taskx/dao/model"
)

// TaskEdgeArchiveDAO defines the storage-agnostic interface for task edge archive persistence.
type TaskEdgeArchiveDAO interface {
	GetStore() Store
	Insert(ctx context.Context, data *model.TaskEdgeArchive) (int64, error)
	BatchInsert(ctx context.Context, data []model.TaskEdgeArchive) (int64, error)
	GetByTaskID(ctx context.Context, taskID string) ([]model.TaskEdgeArchive, error)
}
