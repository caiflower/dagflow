package dao

import (
	"context"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/uptrace/bun"
)

// ExecutionRecordDAO 执行记录映射表 DAO
type ExecutionRecordDAO struct {
	DB *dbv1.Client `autowired:"dagflow"`
}

// Insert 插入执行记录
func (d *ExecutionRecordDAO) Insert(ctx context.Context, record *ExecutionRecord) error {
	_, err := d.DB.GetDB().NewInsert().Model(record).Exec(ctx)
	return err
}

// GetByID 根据 ID 查询执行记录
func (d *ExecutionRecordDAO) GetByID(ctx context.Context, id string) (*ExecutionRecord, error) {
	record := &ExecutionRecord{}
	err := d.DB.GetDB().NewSelect().
		Model(record).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// GetByIDs 批量查询执行记录
func (d *ExecutionRecordDAO) GetByIDs(ctx context.Context, ids []string) ([]ExecutionRecord, error) {
	var records []ExecutionRecord
	err := d.DB.GetDB().NewSelect().
		Model(&records).
		Where("id IN (?)", bun.In(ids)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// List 查询执行记录列表（按创建时间降序，支持按 flow_id 筛选）
func (d *ExecutionRecordDAO) List(ctx context.Context, page, pageSize int, flowID int64) ([]ExecutionRecord, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// 构建查询条件
	countQuery := d.DB.GetDB().NewSelect().Model((*ExecutionRecord)(nil))
	if flowID > 0 {
		countQuery = countQuery.Where("flow_id = ?", flowID)
	}

	// 查询总数
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 查询分页数据
	var records []ExecutionRecord
	listQuery := d.DB.GetDB().NewSelect().Model(&records)
	if flowID > 0 {
		listQuery = listQuery.Where("flow_id = ?", flowID)
	}
	err = listQuery.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// InitExecutionRecord 注册 ExecutionRecordDAO bean
func InitExecutionRecord() {
	bean.AddBean(&ExecutionRecordDAO{})
}
