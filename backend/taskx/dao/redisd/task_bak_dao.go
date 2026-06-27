package redisd

import (
	"context"
	"fmt"

	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
)

type taskBakDAO struct {
	client v2.RedisClient
	store  dao.Store
	keys   *keyBuilder
}

func NewTaskBakDAOWithClient(client v2.RedisClient) dao.TaskBakDAO {
	return NewTaskBakDAOWithConfig(client, nil)
}

func NewTaskBakDAOWithConfig(client v2.RedisClient, keyCfg *KeyConfig) dao.TaskBakDAO {
	return &taskBakDAO{
		client: client,
		store:  NewStore(client),
		keys:   newKeyBuilder(keyCfg),
	}
}

func (d *taskBakDAO) GetStore() dao.Store { return d.store }

func (d *taskBakDAO) Insert(_ context.Context, _ *model.TaskBak) (int64, error) {
	return 0, nil
}

func (d *taskBakDAO) BatchInsert(_ context.Context, _ []model.TaskBak) (int64, error) {
	return 0, nil
}

func (d *taskBakDAO) GetByID(ctx context.Context, id string) (*model.TaskBak, error) {
	key := d.keys.bakTaskKey(id)
	c := cmd(d.client)

	m, err := c.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("taskBak GetByID HGetAll: %w", err)
	}
	if len(m) == 0 {
		return nil, nil
	}
	bak := new(model.TaskBak)
	if err := fromHash(m, bak); err != nil {
		return nil, fmt.Errorf("taskBak GetByID fromHash: %w", err)
	}
	if bak.Status == -1 {
		return nil, nil
	}
	return bak, nil
}

func (d *taskBakDAO) GetByIDs(_ context.Context, _ []string) ([]model.TaskBak, error) {
	return nil, nil
}
