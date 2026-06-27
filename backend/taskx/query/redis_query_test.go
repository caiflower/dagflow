package query

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/caiflower/dagflow/taskx/dao/redisd"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRedisClient struct {
	client *redis.Client
}

func (c *testRedisClient) Cmd() v2.Cmdable         { return &testCmd{Cmdable: c.client} }
func (c *testRedisClient) GetRedis() redis.Cmdable { return c.client }
func (c *testRedisClient) AddHook(hook redis.Hook)  {}
func (c *testRedisClient) Close()                    { c.client.Close() }

type testCmd struct {
	redis.Cmdable
}

func (c *testCmd) Key(key string) string { return key }

func newTestRedis(t *testing.T) (v2.RedisClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rc := &testRedisClient{client: client}
	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})
	return rc, mr
}

func TestRedisTaskQuery_EmptyIDs(t *testing.T) {
	rc, _ := newTestRedis(t)
	svc := &RedisTaskQueryService{
		TaskDao:     redisd.NewTaskDAOWithClient(rc),
		SubtaskDao:  redisd.NewSubtaskDAOWithClient(rc),
		TaskEdgeDao: redisd.NewTaskEdgeDAOWithClient(rc),
	}

	ctx := context.Background()
	details, err := svc.GetTasks(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, details)

	details, err = svc.GetTasks(ctx, []string{})
	require.NoError(t, err)
	assert.Nil(t, details)
}

func TestRedisTaskQuery_GetTasks(t *testing.T) {
	rc, _ := newTestRedis(t)
	taskDao := redisd.NewTaskDAOWithClient(rc)
	subDao := redisd.NewSubtaskDAOWithClient(rc)
	edgeDao := redisd.NewTaskEdgeDAOWithClient(rc)

	svc := &RedisTaskQueryService{
		TaskDao:     taskDao,
		SubtaskDao:  subDao,
		TaskEdgeDao: edgeDao,
	}

	ctx := context.Background()

	_, err := taskDao.Insert(ctx, &model.Task{ID: "t1", TaskName: "demo", State: "pending", Status: 1})
	require.NoError(t, err)

	_, err = subDao.BatchInsert(ctx, []model.Subtask{
		{ID: "s1", TaskID: "t1", TaskName: "step1", State: "pending", Status: 1},
		{ID: "s2", TaskID: "t1", TaskName: "step2", State: "pending", Status: 1},
	})
	require.NoError(t, err)

	_, err = edgeDao.BatchInsert(ctx, []model.TaskEdge{
		{ID: "e1", TaskID: "t1", FromSubtaskID: "s1", ToSubtaskID: "s2", EdgeType: "control"},
	})
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"t1"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, "demo", details[0].Task.TaskName)
	assert.Len(t, details[0].Subtasks, 2)
	assert.Len(t, details[0].Edges, 1)
}

func TestRedisTaskQuery_GetTasks_MissingTask(t *testing.T) {
	rc, _ := newTestRedis(t)
	svc := &RedisTaskQueryService{
		TaskDao:     redisd.NewTaskDAOWithClient(rc),
		SubtaskDao:  redisd.NewSubtaskDAOWithClient(rc),
		TaskEdgeDao: redisd.NewTaskEdgeDAOWithClient(rc),
	}

	ctx := context.Background()

	_, err := svc.TaskDao.Insert(ctx, &model.Task{ID: "t-exists", TaskName: "exists", State: "pending", Status: 1})
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"t-missing"})
	require.NoError(t, err)
	assert.Empty(t, details)
}

func TestRedisTaskQuery_GetTasks_SoftDeleted(t *testing.T) {
	rc, _ := newTestRedis(t)
	taskDao := redisd.NewTaskDAOWithClient(rc)

	svc := &RedisTaskQueryService{
		TaskDao:     taskDao,
		SubtaskDao:  redisd.NewSubtaskDAOWithClient(rc),
		TaskEdgeDao: redisd.NewTaskEdgeDAOWithClient(rc),
	}

	ctx := context.Background()

	// Insert a soft-deleted task (Status = -1)
	_, err := taskDao.Insert(ctx, &model.Task{ID: "t-deleted", TaskName: "deleted", State: "succeeded", Status: -1})
	require.NoError(t, err)

	// GetByIDs skips tasks with Status <= 0
	details, err := svc.GetTasks(ctx, []string{"t-deleted"})
	require.NoError(t, err)

	for _, d := range details {
		if d.Task.ID == "t-deleted" {
			t.Error("soft-deleted task should not be returned")
		}
	}
}

func TestRedisTaskQuery_GetTasks_MultipleTasks(t *testing.T) {
	rc, _ := newTestRedis(t)
	taskDao := redisd.NewTaskDAOWithClient(rc)

	svc := &RedisTaskQueryService{
		TaskDao:     taskDao,
		SubtaskDao:  redisd.NewSubtaskDAOWithClient(rc),
		TaskEdgeDao: redisd.NewTaskEdgeDAOWithClient(rc),
	}

	ctx := context.Background()

	_, err := taskDao.Insert(ctx, &model.Task{ID: "tm1", TaskName: "multi1", State: "pending", Status: 1})
	require.NoError(t, err)
	_, err = taskDao.Insert(ctx, &model.Task{ID: "tm2", TaskName: "multi2", State: "succeeded", Status: 1})
	require.NoError(t, err)
	_, err = taskDao.Insert(ctx, &model.Task{ID: "tm3", TaskName: "multi3", State: "failed", Status: 1})
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"tm1", "tm2", "tm3", "tm-nonexistent"})
	require.NoError(t, err)
	assert.Len(t, details, 3)

	names := make(map[string]bool)
	for _, d := range details {
		names[d.Task.TaskName] = true
	}
	assert.True(t, names["multi1"])
	assert.True(t, names["multi2"])
	assert.True(t, names["multi3"])
}
