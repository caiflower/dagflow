package dao

import (
	"context"

	"github.com/caiflower/dagflow/taskx/dao/model"
)

// TaskBakDAO defines the storage-agnostic interface for task archive persistence.
type TaskBakDAO interface {
	GetStore() Store
	Insert(ctx context.Context, data *model.TaskBak) (int64, error)
	GetByID(ctx context.Context, id string) (*model.TaskBak, error)
	GetByIDs(ctx context.Context, ids []string) ([]model.TaskBak, error)
	BatchInsert(ctx context.Context, data []model.TaskBak) (int64, error)
}
