-- DAGFlow 建表脚本 (SQLite)
-- 与 internal/model/flow.go 中 bun model 定义保持一致

-- 创建 Flow 表（工作流定义）
CREATE TABLE IF NOT EXISTS "flow" (
    "id"          INTEGER    PRIMARY KEY AUTOINCREMENT,
    "name"        TEXT       NOT NULL,
    "description" TEXT       NOT NULL DEFAULT '',
    "nodes_json"  TEXT,
    "edges_json"  TEXT,
    "version"     INTEGER    NOT NULL DEFAULT 1,
    "status"      INTEGER    NOT NULL DEFAULT 1,
    "create_time" TEXT,
    "update_time" TEXT
);

CREATE INDEX IF NOT EXISTS idx_flow_name ON "flow" ("name");
CREATE INDEX IF NOT EXISTS idx_flow_status ON "flow" ("status");
