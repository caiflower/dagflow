package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/caiflower/dagflow/taskx/dao/sqld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDB(t *testing.T) *dbv1.Client {
	t.Helper()
	dir := t.TempDir()
	cfg := dbv1.Config{
		Dialect: "sqlite",
		Url:     filepath.Join(dir, "test.db"),
	}
	client, err := dbv1.NewDBClient(cfg)
	if err != nil {
		t.Fatalf("create db client: %v", err)
	}

	wd, _ := os.Getwd()
	schemaPath := filepath.Join(wd, "..", "dao", "sqld", "ddl", "table-sqlite.sql")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err := client.DB.ExecContext(context.Background(), string(data)); err != nil {
		t.Fatalf("exec schema: %v", err)
	}
	return client
}

func TestSQLTaskQuery_EmptyIDs(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	svc := &SQLTaskQueryService{
		TaskDao:        sqld.NewTaskDAOWithClient(client),
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     sqld.NewSubtaskDAOWithClient(client),
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()
	details, err := svc.GetTasks(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, details)

	details, err = svc.GetTasks(ctx, []string{})
	require.NoError(t, err)
	assert.Nil(t, details)
}

func TestSQLTaskQuery_GetTasks_FromMainTable(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	taskDao := sqld.NewTaskDAOWithClient(client)
	subDao := sqld.NewSubtaskDAOWithClient(client)

	svc := &SQLTaskQueryService{
		TaskDao:        taskDao,
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     subDao,
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()

	_, err := taskDao.Insert(ctx, &model.Task{ID: "t1", TaskName: "demo", State: "pending", Status: 1})
	require.NoError(t, err)

	_, err = subDao.BatchInsert(ctx, []model.Subtask{
		{ID: "s1", TaskID: "t1", TaskName: "step1", State: "pending", Status: 1},
		{ID: "s2", TaskID: "t1", TaskName: "step2", State: "pending", Status: 1},
	})
	require.NoError(t, err)

	_, err = client.DB.NewInsert().Model(&model.TaskEdge{
		ID: "e1", TaskID: "t1", FromSubtaskID: "s1", ToSubtaskID: "s2", EdgeType: "control",
	}).Exec(ctx)
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"t1"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, "demo", details[0].Task.TaskName)
	assert.Len(t, details[0].Subtasks, 2)
	assert.Len(t, details[0].Edges, 1)
}

func TestSQLTaskQuery_GetTasks_FallbackToArchive(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	ctx := context.Background()

	taskBakDao := sqld.NewTaskBakDAOWithClient(client)
	subtaskBakDao := sqld.NewSubtaskBakDAOWithClient(client)
	edgeBakDao := sqld.NewTaskEdgeArchiveDAOWithClient(client)

	svc := &SQLTaskQueryService{
		TaskDao:        sqld.NewTaskDAOWithClient(client),
		TaskBakDao:     taskBakDao,
		SubtaskDao:     sqld.NewSubtaskDAOWithClient(client),
		SubtaskBakDao:  subtaskBakDao,
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: edgeBakDao,
	}

	now := basic.NewFromTime(time.Now())
	_, err := taskBakDao.Insert(ctx, &model.TaskBak{
		ID: "t-archived", TaskName: "archived-demo", State: "succeeded",
		Status: 1, CreateTime: now, LastRunTime: now, ExecuteTime: now,
	})
	require.NoError(t, err)

	_, err = subtaskBakDao.BatchInsert(ctx, []model.SubtaskBak{
		{ID: "sa1", TaskID: "t-archived", TaskName: "step1", State: "succeeded", Status: 1},
		{ID: "sa2", TaskID: "t-archived", TaskName: "step2", State: "succeeded", Status: 1},
	})
	require.NoError(t, err)

	_, err = edgeBakDao.BatchInsert(ctx, []model.TaskEdgeArchive{
		{ID: "ea1", TaskID: "t-archived", FromSubtaskID: "sa1", ToSubtaskID: "sa2", EdgeType: "control"},
	})
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"t-archived"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	assert.Equal(t, "archived-demo", details[0].Task.TaskName)
	assert.Len(t, details[0].Subtasks, 2)
	assert.Len(t, details[0].Edges, 1)
}

func TestSQLTaskQuery_GetTasks_MixedMainAndArchive(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	ctx := context.Background()

	taskDao := sqld.NewTaskDAOWithClient(client)
	subDao := sqld.NewSubtaskDAOWithClient(client)

	taskBakDao := sqld.NewTaskBakDAOWithClient(client)
	subtaskBakDao := sqld.NewSubtaskBakDAOWithClient(client)
	edgeBakDao := sqld.NewTaskEdgeArchiveDAOWithClient(client)

	svc := &SQLTaskQueryService{
		TaskDao:        taskDao,
		TaskBakDao:     taskBakDao,
		SubtaskDao:     subDao,
		SubtaskBakDao:  subtaskBakDao,
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: edgeBakDao,
	}

	// Task 1 in main table
	_, err := taskDao.Insert(ctx, &model.Task{ID: "tm-main", TaskName: "main-task", State: "pending", Status: 1})
	require.NoError(t, err)
	_, err = subDao.BatchInsert(ctx, []model.Subtask{
		{ID: "sm1", TaskID: "tm-main", TaskName: "step1", State: "pending", Status: 1},
	})
	require.NoError(t, err)

	// Task 2 in archive only
	now := basic.NewFromTime(time.Now())
	_, err = taskBakDao.Insert(ctx, &model.TaskBak{
		ID: "tm-arch", TaskName: "arch-task", State: "succeeded",
		Status: 1, CreateTime: now, LastRunTime: now, ExecuteTime: now,
	})
	require.NoError(t, err)
	_, err = subtaskBakDao.BatchInsert(ctx, []model.SubtaskBak{
		{ID: "sa-arch", TaskID: "tm-arch", TaskName: "step1", State: "succeeded", Status: 1},
	})
	require.NoError(t, err)

	details, err := svc.GetTasks(ctx, []string{"tm-main", "tm-arch", "tm-missing"})
	require.NoError(t, err)
	assert.Len(t, details, 2)

	names := make(map[string]bool)
	for _, d := range details {
		names[d.Task.TaskName] = true
	}
	assert.True(t, names["main-task"])
	assert.True(t, names["arch-task"])
}

func TestSQLTaskQuery_GetTasks_MissingFromBoth(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	svc := &SQLTaskQueryService{
		TaskDao:        sqld.NewTaskDAOWithClient(client),
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     sqld.NewSubtaskDAOWithClient(client),
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()
	details, err := svc.GetTasks(ctx, []string{"nonexistent"})
	require.NoError(t, err)
	assert.Empty(t, details)
}
