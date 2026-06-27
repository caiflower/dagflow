package model

import (
	"github.com/caiflower/common-tools/pkg/basic"
	"github.com/uptrace/bun"
)

// TaskEdgeArchive 任务边归档表模型（字段与 TaskEdge 一致）
type TaskEdgeArchive struct {
	bun.BaseModel `bun:"table:task_edge_archive"`
	ID            string     `bun:"id,pk,notnull" json:"id"`
	TaskID        string     `bun:"task_id,notnull" json:"taskID"`
	FromSubtaskID string     `bun:"from_subtask_id,notnull" json:"fromSubtaskID"`
	ToSubtaskID   string     `bun:"to_subtask_id,notnull" json:"toSubtaskID"`
	EdgeType      string     `bun:"edge_type,notnull" json:"edgeType"`
	FieldMappings string     `bun:"field_mappings" json:"fieldMappings"`
	CreateTime    basic.Time `bun:"create_time" json:"createTime"`
}
