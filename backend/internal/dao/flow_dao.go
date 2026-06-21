/*
 * Copyright 2026 caiflower Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dao

import (
	"context"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/dagflow/internal/dao/model"
)

// FlowDAO Flow 数据访问对象
type FlowDAO struct {
	DB *dbv1.Client `autowired:"dagflow"`
}

// GetByID 根据 ID 查询 Flow
func (d *FlowDAO) GetByID(ctx context.Context, id string) (*model.Flow, error) {
	flow := &model.Flow{ID: id}
	err := d.DB.GetDB().NewSelect().
		Model(flow).
		WherePK().
		Scan(ctx)
	return flow, d.DB.ParseErr(err)
}

// List 分页查询 Flow 列表
func (d *FlowDAO) List(ctx context.Context, filter *model.FlowFilter) ([]model.Flow, int, error) {
	var result []model.Flow
	total, err := d.DB.QueryPage(ctx, &result, filter)
	return result, total, err
}

// Insert 创建 Flow
func (d *FlowDAO) Insert(ctx context.Context, flow *model.Flow) (string, error) {
	_, err := d.DB.Insert(ctx, flow)
	return flow.ID, err
}

// Update 更新 Flow
func (d *FlowDAO) Update(ctx context.Context, flow *model.Flow) error {
	_, err := d.DB.GetDB().NewUpdate().
		Model(flow).
		WherePK().
		Exec(ctx)
	return err
}

// Delete 删除 Flow（软删除，status=0）
func (d *FlowDAO) Delete(ctx context.Context, id string) error {
	flow := &model.Flow{ID: id, Status: 0}
	_, err := d.DB.GetDB().NewUpdate().
		Model(flow).
		Column("status").
		WherePK().
		Exec(ctx)
	return err
}

// Init 初始化 DAO 并注册 bean
func Init() {
	bean.AddBean(&FlowDAO{})
}
