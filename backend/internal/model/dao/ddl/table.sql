-- DAGFlow 建表脚本 (MySQL)
-- 与 internal/model/flow.go 中 bun model 定义保持一致

-- 创建 Flow 表（工作流定义）
CREATE TABLE IF NOT EXISTS `flow` (
    `id`          BIGINT       NOT NULL AUTO_INCREMENT COMMENT 'Flow ID',
    `name`        VARCHAR(255) NOT NULL                COMMENT 'Flow 名称',
    `description` VARCHAR(512) NOT NULL DEFAULT ''     COMMENT 'Flow 描述',
    `nodes_json`  TEXT                                 COMMENT '节点定义 JSON（FlowNode[]）',
    `edges_json`  TEXT                                 COMMENT '边定义 JSON（FlowEdge[]）',
    `version`     INT          NOT NULL DEFAULT 1      COMMENT '版本号',
    `status`      TINYINT      NOT NULL DEFAULT 1      COMMENT '状态(0:禁用, 1:启用)',
    `create_time` TIMESTAMP(3)                         COMMENT '创建时间',
    `update_time` TIMESTAMP(3)                         COMMENT '更新时间',
    PRIMARY KEY (`id`),
    INDEX idx_name (`name`),
    INDEX idx_status (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- 创建 execution_record 表（执行记录映射，execID→taskID 索引）
CREATE TABLE IF NOT EXISTS `execution_record` (
    `id`         VARCHAR(64)  NOT NULL                COMMENT '执行记录 ID',
    `flow_id`    BIGINT       NOT NULL                COMMENT 'Flow ID',
    `flow_name`  VARCHAR(128) NOT NULL                COMMENT 'Flow 名称',
    `task_id`    VARCHAR(64)  NOT NULL                COMMENT 'taskx Task ID',
    `created_at` DATETIME     NOT NULL                COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX idx_flow_id (`flow_id`),
    INDEX idx_task_id (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
