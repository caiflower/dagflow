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

-- 创建 execution_record 表（执行记录映射）
CREATE TABLE IF NOT EXISTS "execution_record" (
    "id"         TEXT    NOT NULL PRIMARY KEY,
    "flow_id"    INTEGER NOT NULL,
    "flow_name"  TEXT    NOT NULL,
    "task_id"    TEXT    NOT NULL,
    "created_at" TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_er_flow_id ON "execution_record" ("flow_id");
CREATE INDEX IF NOT EXISTS idx_er_task_id ON "execution_record" ("task_id");
