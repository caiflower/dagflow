package sqld

import (
	"context"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/dagflow/taskx/dao"
	"github.com/caiflower/dagflow/taskx/dao/model"
	"github.com/uptrace/bun"
)

const DefaultTableNameOfTaskBak = "task_archive"

type taskBakDAO struct {
	client    *dbv1.Client
	store     dao.Store
	tableName string
}

func (d *taskBakDAO) db(ctx context.Context) bun.IDB {
	if tx := dao.TxFromContext(ctx); tx != nil {
		if bunTx, ok := tx.(*bun.Tx); ok {
			return bunTx
		}
	}
	return d.client.GetDB()
}

func NewTaskBakDAOWithClient(client *dbv1.Client) dao.TaskBakDAO {
	return &taskBakDAO{client: client, store: NewStore(client), tableName: DefaultTableNameOfTaskBak}
}

func NewTaskBakDAO() dao.TaskBakDAO {
	return &taskBakDAO{tableName: DefaultTableNameOfTaskBak}
}

func NewTaskBakDAOWithConfig(client *dbv1.Client, tableName string) dao.TaskBakDAO {
	if tableName == "" {
		tableName = DefaultTableNameOfTaskBak
	}
	return &taskBakDAO{client: client, store: NewStore(client), tableName: tableName}
}

func (d *taskBakDAO) GetStore() dao.Store { return d.store }

func (d *taskBakDAO) Insert(ctx context.Context, data *model.TaskBak) (int64, error) {
	return d.client.GetRowsAffected(d.db(ctx).NewInsert().Model(data).ModelTableExpr(d.tableName).Exec(ctx))
}

func (d *taskBakDAO) BatchInsert(ctx context.Context, data []model.TaskBak) (int64, error) {
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

func (d *taskBakDAO) GetByID(ctx context.Context, id string) (*model.TaskBak, error) {
	m := new(model.TaskBak)
	err := d.client.GetDB().NewSelect().Model(m).ModelTableExpr(d.tableName).ColumnExpr("*").Where("status>0").Where("id = ?", id).Limit(1).Scan(ctx)
	if err != nil {
		if d.client.ParseErr(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	return m, err
}

func (d *taskBakDAO) GetByIDs(ctx context.Context, ids []string) ([]model.TaskBak, error) {
	var tasks []model.TaskBak
	if len(ids) == 0 {
		return tasks, nil
	}
	err := d.client.GetDB().NewSelect().Model(&tasks).ModelTableExpr(d.tableName).ColumnExpr("*").Where("status>0").Where("id IN (?)", bun.In(ids)).Scan(ctx, &tasks)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *taskBakDAO) DeleteByID(ctx context.Context, id string) (int64, error) {
	result, err := d.db(ctx).NewDelete().Table(d.tableName).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (d *taskBakDAO) SoftDeleteByID(ctx context.Context, id string) (int64, error) {
	result, err := d.db(ctx).NewUpdate().Table(d.tableName).Set("status = ?", -1).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
