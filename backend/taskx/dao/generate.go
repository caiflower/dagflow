package dao

// Generate model structs and sqld DAO implementations from MySQL table definitions.
// Requires: MySQL running at 127.0.0.1:3306 with database 'dagflow' and tables created via sqld/ddl/table.sql.
//
//go:generate go run -mod=mod github.com/caiflower/common-tools/db/v1/cmd@v0.0.0-20260620153008-279ce8fa318d -dialect mysql -host 127.0.0.1 -port 3306 -user root -password root123 -db dagflow -charset utf8 -timeout 60 -pkg github.com/caiflower/dagflow/taskx/dao/sqld -tables task_archive,subtask_archive,task_edge_archive -struct_out ./model -dao_out ./sqld
