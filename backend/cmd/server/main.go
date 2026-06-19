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

package main

import (
	"context"
	"runtime"

	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/global"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/logger"
	"github.com/caiflower/common-tools/web"
	"github.com/caiflower/common-tools/web/app/server/config"

	"github.com/caiflower/dagflow/constants"
	"github.com/caiflower/dagflow/internal/api"
	"github.com/caiflower/dagflow/internal/model"
	"github.com/caiflower/dagflow/internal/model/dao"
	"github.com/caiflower/dagflow/internal/protocol"
	"github.com/caiflower/dagflow/internal/service"
	taskxModel "github.com/caiflower/dagflow/taskx/dao/model"
)

var engine *web.Engine

// protocolRegistry 全局协议注册中心
var protocolRegistry *protocol.Registry

func init() {
	// 1. 加载配置
	constants.InitConfig()

	// 2. 初始化日志
	logger.InitLogger(&constants.DefaultConfig.LoggerConfig)

	// 3. 设置 GOMAXPROCS
	if constants.Prop.GoMaxProcs > 0 {
		runtime.GOMAXPROCS(constants.Prop.GoMaxProcs)
	}

	// 4. 注册手动 bean
	initBean()

	// 5. 初始化数据库
	initDB()

	// 6. 注入所有 autowired 依赖
	bean.Ioc()

	// 7. Ioc 完成后，创建 gRPC 服务实例并设置到 API 层
	initGrpcServices()

	// 8. 初始化 Web 引擎（路由注册在 gRPC 服务就绪后执行）
	initWeb()
}

func main() {
	logger.Info("DAGFlow server starting...")
	global.DefaultResourceManger.Signal()
}

// initBean 注册手动管理的 bean
func initBean() {
	// 初始化协议注册中心
	protocolRegistry = protocol.NewRegistry()
	protocol.RegisterBuiltinProtocols(protocolRegistry)
	bean.AddBean(protocolRegistry)

	// 初始化 FlowService
	service.Init()
	// 初始化 ExecutionService
	service.InitExec()
}

// initGrpcServices 创建 gRPC 服务实例并设置到 API 层
func initGrpcServices() {
	flowSvc := bean.GetBeanT[*service.FlowService]()
	execSvc := bean.GetBeanT[*service.ExecutionService]()

	api.SetFlowGrpcService(api.NewFlowGrpcService(flowSvc))
	api.SetProtocolGrpcService(api.NewProtocolGrpcService(protocolRegistry))
	api.SetExecutionGrpcService(api.NewExecutionGrpcService(execSvc))
}

// initWeb 创建 Web 引擎并注册路由
func initWeb() {
	webCfg := constants.DefaultConfig.GetWebConfigByName("dagflow")
	if webCfg == nil {
		panic("web config 'dagflow' not found in default.yaml")
	}

	engine = web.Default(
		config.WithName(webCfg.Name),
		config.WithAddr(webCfg.Addr),
		config.WithEnablePprof(webCfg.EnablePprof),
		config.WithEnableSwagger(webCfg.EnableSwagger),
	)

	// 注册 API 路由（使用 gRPC handler 方式）
	api.RegisterRoutes(engine)

	// 注册为 Daemon 资源，由 global 管理生命周期
	global.DefaultResourceManger.AddDaemon(engine)
}

// initDB 初始化数据库连接
func initDB() {
	dbCfg := constants.DefaultConfig.GetDatabaseConfigByName("dagflow")
	if dbCfg == nil {
		logger.Warn("database config 'dagflow' not found, skipping DB init")
		return
	}

	client, err := dbv1.NewDBClient(*dbCfg)
	if err != nil {
		panic("failed to init database: " + err.Error())
	}

	// 注册 DB client 为 bean（FlowDAO 通过 autowired 注入）
	bean.SetBeanOverwrite(dbCfg.Name, client)

	// 自动建表
	ctx := context.Background()
	tables := []interface{}{
		// dagflow 业务表
		(*model.Flow)(nil),
		// taskx 任务调度表
		(*taskxModel.Task)(nil),
		(*taskxModel.Subtask)(nil),
		(*taskxModel.TaskEdge)(nil),
		(*taskxModel.TaskBak)(nil),
		(*taskxModel.SubtaskBak)(nil),
	}
	for _, t := range tables {
		_, err := client.DB.NewCreateTable().
			Model(t).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			logger.Error("failed to create table: " + err.Error())
		}
	}

	// 初始化 DAO
	dao.Init()

	logger.Info("database initialized successfully")
}
