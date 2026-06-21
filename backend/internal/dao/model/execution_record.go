package model

import (
	"time"

	"github.com/uptrace/bun"
)

// ExecutionRecord 执行记录映射表（execID → taskID）
// 仅存储索引关系，不维护状态字段。状态从 taskx 引擎层实时查询。
type ExecutionRecord struct {
	bun.BaseModel `bun:"table:execution_record"`
	ID            string    `bun:"id,pk" json:"id"`
	FlowID        string    `bun:"flow_id,notnull" json:"flowID"`
	FlowName      string    `bun:"flow_name,notnull" json:"flowName"`
	TaskID        string    `bun:"task_id,notnull" json:"taskID"`
	CreatedAt     time.Time `bun:"created_at,notnull" json:"createdAt"`
}
