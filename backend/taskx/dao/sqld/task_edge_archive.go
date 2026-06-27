package sqld

import (
	"context"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/uptrace/bun"
)

const DefaultTableNameOfTaskEdgeArchive = "task_edge_archive"

type taskEdgeArchiveDAO struct {
	client    *dbv1.Client
	store     dao.Store
	tableName string
}

func (d *taskEdgeArchiveDAO) db(ctx context.Context) bun.IDB {
	if tx := dao.TxFromContext(ctx); tx != nil {
		if bunTx, ok := tx.(*bun.Tx); ok {
			return bunTx
		}
	}
	return d.client.GetDB()
}

func NewTaskEdgeArchiveDAOWithClient(client *dbv1.Client) dao.TaskEdgeArchiveDAO {
	return &taskEdgeArchiveDAO{client: client, store: NewStore(client), tableName: DefaultTableNameOfTaskEdgeArchive}
}

func NewTaskEdgeArchiveDAO() dao.TaskEdgeArchiveDAO {
	return &taskEdgeArchiveDAO{tableName: DefaultTableNameOfTaskEdgeArchive}
}

func NewTaskEdgeArchiveDAOWithConfig(client *dbv1.Client, tableName string) dao.TaskEdgeArchiveDAO {
	if tableName == "" {
		tableName = DefaultTableNameOfTaskEdgeArchive
	}
	return &taskEdgeArchiveDAO{client: client, store: NewStore(client), tableName: tableName}
}

func (d *taskEdgeArchiveDAO) GetStore() dao.Store { return d.store }

func (d *taskEdgeArchiveDAO) Insert(ctx context.Context, data *model.TaskEdgeArchive) (int64, error) {
	return d.client.GetRowsAffected(d.db(ctx).NewInsert().Model(data).ModelTableExpr(d.tableName).Exec(ctx))
}

func (d *taskEdgeArchiveDAO) BatchInsert(ctx context.Context, data []model.TaskEdgeArchive) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}
	pageNumber := 1
	batch := 50
	count := int64(0)
	for {
		canSplit, start, end := dbv1.SplitIndex(pageNumber, batch, len(data))
		if !canSplit {
			break
		}
		batchList := data[start:end]
		cnt, err := d.client.GetRowsAffected(d.db(ctx).NewInsert().Model(&batchList).ModelTableExpr(d.tableName).Exec(ctx))
		if err != nil {
			return count, err
		}
		count += cnt
		pageNumber++
	}
	return count, nil
}

func (d *taskEdgeArchiveDAO) GetByTaskID(ctx context.Context, taskID string) ([]model.TaskEdgeArchive, error) {
	var edges []model.TaskEdgeArchive
	err := d.client.GetDB().NewSelect().Model(&edges).ModelTableExpr(d.tableName).ColumnExpr("*").Where("task_id = ?", taskID).Scan(ctx, &edges)
	if err != nil {
		return nil, err
	}
	return edges, nil
}
