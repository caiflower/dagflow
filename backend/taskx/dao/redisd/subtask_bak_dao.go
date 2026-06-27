package redisd

import (
	"context"
	"fmt"

	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/redis/go-redis/v9"
)

type subtaskBakDAO struct {
	client v2.RedisClient
	store  dao.Store
	keys   *keyBuilder
}

func NewSubtaskBakDAOWithClient(client v2.RedisClient) dao.SubtaskBakDAO {
	return NewSubtaskBakDAOWithConfig(client, nil)
}

func NewSubtaskBakDAOWithConfig(client v2.RedisClient, keyCfg *KeyConfig) dao.SubtaskBakDAO {
	return &subtaskBakDAO{
		client: client,
		store:  NewStore(client),
		keys:   newKeyBuilder(keyCfg),
	}
}

func (d *subtaskBakDAO) GetStore() dao.Store { return d.store }

func (d *subtaskBakDAO) Insert(_ context.Context, _ *model.SubtaskBak) (int64, error) {
	return 0, nil
}

func (d *subtaskBakDAO) BatchInsert(_ context.Context, _ []model.SubtaskBak) (int64, error) {
	return 0, nil
}

func (d *subtaskBakDAO) GetByTaskID(ctx context.Context, taskID string) ([]model.SubtaskBak, error) {
	c := cmd(d.client)

	ids, err := c.SMembers(ctx, d.keys.bakSubtaskIndexKey(taskID)).Result()
	if err != nil {
		return nil, fmt.Errorf("subtaskBak GetByTaskID SMembers: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	pipe := c.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(ids))
	for i, id := range ids {
		cmds[i] = pipe.HGetAll(ctx, d.keys.bakSubtaskKey(id))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("subtaskBak GetByTaskID pipeline: %w", err)
	}

	var subtasks []model.SubtaskBak
	for _, hcmd := range cmds {
		m, err := hcmd.Result()
		if err != nil || len(m) == 0 {
			continue
		}
		s := new(model.SubtaskBak)
		if err := fromHash(m, s); err != nil {
			continue
		}
		if s.Status > 0 {
			subtasks = append(subtasks, *s)
		}
	}
	return subtasks, nil
}
