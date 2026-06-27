package backup

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
	"github.com/caiflower/dagflow/taskx/types"
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

func TestSQLBackupManager_NoTasks(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	mgr := &SQLBackupManager{
		DBClient:       client,
		TaskDao:        sqld.NewTaskDAOWithClient(client),
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     sqld.NewSubtaskDAOWithClient(client),
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()
	cfg := types.BackupConfig{
		Age:         24 * time.Hour,
		BatchSize:   100,
		FinalStates: []string{string(types.TaskFailed), string(types.TaskSucceeded)},
	}
	n, err := mgr.BackupTasks(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestSQLBackupManager_ArchivesAndDeletesExpiredTasks(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	taskDao := sqld.NewTaskDAOWithClient(client)
	subtaskDao := sqld.NewSubtaskDAOWithClient(client)

	taskBakDao := sqld.NewTaskBakDAOWithClient(client)
	subtaskBakDao := sqld.NewSubtaskBakDAOWithClient(client)
	edgeBakDao := sqld.NewTaskEdgeArchiveDAOWithClient(client)

	mgr := &SQLBackupManager{
		DBClient:       client,
		TaskDao:        taskDao,
		TaskBakDao:     taskBakDao,
		SubtaskDao:     subtaskDao,
		SubtaskBakDao:  subtaskBakDao,
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: edgeBakDao,
	}

	ctx := context.Background()

	oldTime := basic.NewFromTime(time.Now().Add(-48 * time.Hour))
	task := &model.Task{
		ID:          "t-archive",
		TaskName:    "archived-task",
		State:       "succeeded",
		Status:      1,
		ExecuteTime: oldTime,
		CreateTime:  oldTime,
	}
	_, err := taskDao.Insert(ctx, task)
	require.NoError(t, err)

	// Insert subtasks
	_, err = subtaskDao.BatchInsert(ctx, []model.Subtask{
		{ID: "st-archive", TaskID: "t-archive", TaskName: "archived-subtask", State: "succeeded", Status: 1},
	})
	require.NoError(t, err)

	// Insert edge
	_, err = client.DB.NewInsert().Model(&model.TaskEdge{
		ID: "e-archive", TaskID: "t-archive", FromSubtaskID: "st-archive",
		ToSubtaskID: "st-archive", EdgeType: "control",
	}).Exec(ctx)
	require.NoError(t, err)

	// Insert a pending task that should NOT be archived
	_, err = taskDao.Insert(ctx, &model.Task{
		ID: "t-pending", TaskName: "pending-task", State: "pending", Status: 1,
		ExecuteTime: oldTime,
	})
	require.NoError(t, err)

	cfg := types.BackupConfig{
		Age:         1 * time.Hour,
		BatchSize:   100,
		FinalStates: []string{string(types.TaskFailed), string(types.TaskSucceeded)},
	}
	n, err := mgr.BackupTasks(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "should archive 1 task")

	// Verify main table no longer has t-archive
	got, _ := taskDao.GetByID(ctx, "t-archive")
	assert.Nil(t, got, "t-archive should be deleted from main table")

	// Verify archive table has t-archive
	bakTasks, err := taskBakDao.GetByIDs(ctx, []string{"t-archive"})
	require.NoError(t, err)
	require.Len(t, bakTasks, 1)
	assert.Equal(t, "archived-task", bakTasks[0].TaskName)

	// Verify subtasks archived
	bakSubtasks, err := subtaskBakDao.GetByTaskID(ctx, "t-archive")
	require.NoError(t, err)
	require.Len(t, bakSubtasks, 1)
	assert.Equal(t, "archived-subtask", bakSubtasks[0].TaskName)

	// Verify edges archived
	bakEdges, err := edgeBakDao.GetByTaskID(ctx, "t-archive")
	require.NoError(t, err)
	require.Len(t, bakEdges, 1)
	assert.Equal(t, "e-archive", bakEdges[0].ID)

	// Verify pending task remains
	gotPending, _ := taskDao.GetByID(ctx, "t-pending")
	require.NotNil(t, gotPending)
	assert.Equal(t, "pending", gotPending.State)
}

func TestSQLBackupManager_RespectsAge(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	taskDao := sqld.NewTaskDAOWithClient(client)

	mgr := &SQLBackupManager{
		DBClient:       client,
		TaskDao:        taskDao,
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     sqld.NewSubtaskDAOWithClient(client),
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()

	recent := basic.NewFromTime(time.Now().Add(-1 * time.Minute))
	_, err := taskDao.Insert(ctx, &model.Task{
		ID: "t-recent", TaskName: "recent", State: "succeeded", Status: 1,
		ExecuteTime: recent,
	})
	require.NoError(t, err)

	cfg := types.BackupConfig{
		Age:         1 * time.Hour,
		BatchSize:   100,
		FinalStates: []string{string(types.TaskFailed), string(types.TaskSucceeded)},
	}
	n, err := mgr.BackupTasks(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "recent task should not be archived")

	got, _ := taskDao.GetByID(ctx, "t-recent")
	require.NotNil(t, got)
}

func TestSQLBackupManager_RespectsBatchSize(t *testing.T) {
	client := createTestDB(t)
	defer client.Close()

	taskDao := sqld.NewTaskDAOWithClient(client)
	subtaskDao := sqld.NewSubtaskDAOWithClient(client)

	mgr := &SQLBackupManager{
		DBClient:       client,
		TaskDao:        taskDao,
		TaskBakDao:     sqld.NewTaskBakDAOWithClient(client),
		SubtaskDao:     subtaskDao,
		SubtaskBakDao:  sqld.NewSubtaskBakDAOWithClient(client),
		TaskEdgeDao:    sqld.NewTaskEdgeDAOWithClient(client),
		TaskEdgeBakDao: sqld.NewTaskEdgeArchiveDAOWithClient(client),
	}

	ctx := context.Background()

	oldTime := basic.NewFromTime(time.Now().Add(-48 * time.Hour))
	for i := 0; i < 5; i++ {
		taskID := "tb-" + string(rune('a'+i))
		_, err := taskDao.Insert(ctx, &model.Task{
			ID:          taskID,
			TaskName:    "batch",
			State:       "succeeded",
			Status:      1,
			ExecuteTime: oldTime,
		})
		require.NoError(t, err)
		// Add a subtask with an edge so archive inserts are not empty
		_, err = subtaskDao.BatchInsert(ctx, []model.Subtask{
			{ID: "s-" + taskID, TaskID: taskID, TaskName: "step", State: "succeeded", Status: 1},
		})
		require.NoError(t, err)
		_, err = client.DB.NewInsert().Model(&model.TaskEdge{
			ID: "e-" + taskID, TaskID: taskID,
			FromSubtaskID: "s-" + taskID, ToSubtaskID: "s-" + taskID,
			EdgeType: "control",
		}).Exec(ctx)
		require.NoError(t, err)
	}

	cfg := types.BackupConfig{
		Age:         1 * time.Hour,
		BatchSize:   2,
		FinalStates: []string{string(types.TaskFailed), string(types.TaskSucceeded)},
	}
	n, err := mgr.BackupTasks(ctx, cfg)
	require.NoError(t, err)
	assert.LessOrEqual(t, n, 2)
}
