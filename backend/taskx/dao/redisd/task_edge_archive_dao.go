package redisd

import (
	"context"

	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
)

type taskEdgeArchiveDAO struct {
	client v2.RedisClient
	store  dao.Store
}

func NewTaskEdgeArchiveDAOWithClient(client v2.RedisClient) dao.TaskEdgeArchiveDAO {
	return NewTaskEdgeArchiveDAOWithConfig(client, nil)
}

func NewTaskEdgeArchiveDAOWithConfig(client v2.RedisClient, _ *KeyConfig) dao.TaskEdgeArchiveDAO {
	return &taskEdgeArchiveDAO{client: client, store: NewStore(client)}
}

func (d *taskEdgeArchiveDAO) GetStore() dao.Store { return d.store }

func (d *taskEdgeArchiveDAO) Insert(_ context.Context, _ *model.TaskEdgeArchive) (int64, error) {
	return 0, nil
}

func (d *taskEdgeArchiveDAO) BatchInsert(_ context.Context, _ []model.TaskEdgeArchive) (int64, error) {
	return 0, nil
}

func (d *taskEdgeArchiveDAO) GetByTaskID(_ context.Context, _ string) ([]model.TaskEdgeArchive, error) {
	return nil, nil
}
