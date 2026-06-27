package dao

import (
	"context"

	"github.com/caiflower/dagflow/taskx/dao/model"
)

// SubtaskBakDAO defines the storage-agnostic interface for subtask archive persistence.
type SubtaskBakDAO interface {
	GetStore() Store
	Insert(ctx context.Context, data *model.SubtaskBak) (int64, error)
	BatchInsert(ctx context.Context, data []model.SubtaskBak) (int64, error)
	GetByTaskID(ctx context.Context, taskID string) ([]model.SubtaskBak, error)
}
