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

package model

import (
	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/uptrace/bun"
)

// Flow 工作流定义表
type Flow struct {
	bun.BaseModel `bun:"table:flow"`
	ID            string     `bun:"id,pk" json:"id"`
	Name          string     `bun:"name,notnull" json:"name"`
	Description   string     `bun:"description" json:"description"`
	NodesJSON     string     `bun:"nodes_json,type:text" json:"nodesJSON"` // 节点定义 JSON
	EdgesJSON     string     `bun:"edges_json,type:text" json:"edgesJSON"` // 边定义 JSON
	Version       int        `bun:"version,notnull,default:1" json:"version"`
	Status        int8       `bun:"status,notnull,default:1" json:"status"` // 1=启用 0=禁用
	CreateTime    basic.Time `bun:"create_time" json:"createTime"`
	UpdateTime    basic.Time `bun:"update_time" json:"updateTime"`
}

// FlowFilter 查询过滤条件
type FlowFilter struct {
	Page        int      `json:"page"`
	PageSize    int      `json:"pageSize"`
	DisablePage bool     `json:"disablePage"`
	Orders      []string `json:"orders,omitempty"`
	Name        string   `json:"name,omitempty"`
	Status      *int8    `json:"status,omitempty"`
}

func (f *FlowFilter) GetPage() (offset int, limit int, disable bool) {
	if f.DisablePage {
		return 0, 0, true
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	size := f.PageSize
	if size <= 0 {
		size = 20
	}
	return (page - 1) * size, size, false
}

func (f *FlowFilter) Filter(db bun.IDB) *bun.SelectQuery {
	q := db.NewSelect()
	q.Where("status > 0")
	if f.Name != "" {
		q.Where("name LIKE ?", "%"+f.Name+"%")
	}
	if f.Status != nil {
		q.Where("status = ?", *f.Status)
	}
	if len(f.Orders) > 0 {
		q.Order(f.Orders...)
	} else {
		q.Order("id DESC")
	}
	return q
}
