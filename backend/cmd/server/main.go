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
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/caiflower/common-tools/cluster"
	dbv1 "github.com/caiflower/common-tools/db/v1"
	"github.com/caiflower/common-tools/global"
	"github.com/caiflower/common-tools/pkg/bean"
	"github.com/caiflower/common-tools/pkg/logger"
	v2 "github.com/caiflower/common-tools/redis/v2"
	"github.com/caiflower/common-tools/web"
	"github.com/caiflower/common-tools/web/app/server/config"
	"github.com/caiflower/dagflow/internal/dao"
	"github.com/caiflower/dagflow/internal/dao/model"
	"github.com/caiflower/dagflow/internal/node_registry"
	"github.com/caiflower/dagflow/internal/protocol/remote_executor"

	"github.com/caiflower/dagflow/constants"
	"github.com/caiflower/dagflow/internal/api"
	dagflowpb "github.com/caiflower/dagflow/internal/proto"
	"github.com/caiflower/dagflow/internal/protocol"
	"github.com/caiflower/dagflow/internal/service"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx"
	taskxModel "github.com/caiflower/dagflow/taskx/dao/model"
	"google.golang.org/grpc"
)

var engine *web.Engine

// dbClient 全局数据库客户端引用
var dbClient *dbv1.Client

// protocolRegistry 全局协议注册中心
var protocolRegistry *protocol.Registry

// dagflowCluster 单节点集群引用
var dagflowCluster cluster.ICluster

var nodeReg *node_registry.NodeRegistry
var remotePool *remote_executor.ConnPool

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

	// 6. 初始化 taskx 集群调度（注册 cluster + DAO beans）
	initTaskx()

	// 7. 注入所有 autowired 依赖
	bean.Ioc()

	// 8. Ioc 完成后，启动 taskx receiver 和 cluster
	startTaskx()

	// 9. 创建 gRPC 服务实例并设置到 API 层
	initGrpcServices()

	// 10. 初始化 Web 引擎（路由注册在 gRPC 服务就绪后执行）
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

	// Initialize RemoteExecutor connection pool
	remotePool = remote_executor.NewConnPool()
	service.SetRemoteExecutorPool(remotePool)
}

// initGrpcServices 创建 gRPC 服务实例并设置到 API 层
func initGrpcServices() {
	flowSvc := bean.GetBeanT[*service.FlowService]()
	execSvc := bean.GetBeanT[*service.ExecutionService]()

	api.SetFlowGrpcService(api.NewFlowGrpcService(flowSvc))
	api.SetProtocolGrpcService(api.NewProtocolGrpcService(protocolRegistry))
	api.SetExecutionGrpcService(api.NewExecutionGrpcService(execSvc))

	// Initialize NodeRegistry with Redis client from config
	var redisClient v2.RedisClient
	if constants.Prop.RedisEmbedded {
		mr, err := miniredis.Run()
		if err != nil {
			logger.Error("failed to start embedded miniredis: %v", err)
			panic(err)
		}
		redisClient, err = v2.NewRedisClient(v2.Config{
			Addrs: []string{mr.Addr()},
		})
		if err != nil {
			logger.Error("failed to create embedded Redis client: %v", err)
			panic(err)
		}
		global.DefaultResourceManger.Add(mr)

		logger.Info("NodeRegistry using embedded miniredis at %s", mr.Addr())
	} else {
		redisCfg := constants.DefaultConfig.GetRedisConfigByName("dagflow")
		if redisCfg == nil {
			panic("redis config 'dagflow' not found in default.yaml")
		}
		var err error
		redisClient, err = v2.NewRedisClient(*redisCfg)
		if err != nil {
			logger.Error("failed to create Redis client: %v", err)
			panic(err)
		}
		logger.Info("NodeRegistry initialized with Redis at %v", redisCfg.Addrs)
	}

	nodeReg = node_registry.NewNodeRegistry(redisClient)
	service.SetNodeRegistry(nodeReg)

	// Start unified gRPC server as daemon (managed by global.ResourceManger)
	startGrpcServer()
}

// GrpcServer wraps a gRPC server hosting all DAGFlow services as a daemon resource.
type GrpcServer struct {
	server *grpc.Server
	lis    net.Listener
}

func (g *GrpcServer) Name() string { return "GrpcServer" }

func (g *GrpcServer) Start() error {
	logger.Info("gRPC server listening on %s", g.lis.Addr().String())

	go func() {
		if err := g.server.Serve(g.lis); err != nil {
			logger.Error("failed to start gRPC server: %v", err)
		}
	}()

	return nil
}

func (g *GrpcServer) Close() {
	logger.Info("gRPC server shutting down...")
	g.server.GracefulStop()
	logger.Info("gRPC server stopped")
}

func startGrpcServer() {
	port := constants.Prop.GRPC.Port
	if port == 0 {
		port = 50051
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Error("failed to listen on gRPC port %d: %v", port, err)
		return
	}

	s := grpc.NewServer()

	// Register NodeRegistry service
	if nodeReg != nil {
		pb.RegisterNodeRegistryServer(s, nodeReg)
		logger.Info("gRPC: NodeRegistry service registered")
	}

	// Register Flow/Protocol/Execution services
	if flowSvc := api.GetFlowGrpcService(); flowSvc != nil {
		dagflowpb.RegisterFlowServiceServer(s, flowSvc)
		logger.Info("gRPC: Flow service registered")
	}
	if protoSvc := api.GetProtocolGrpcService(); protoSvc != nil {
		dagflowpb.RegisterProtocolServiceServer(s, protoSvc)
		logger.Info("gRPC: Protocol service registered")
	}
	if execSvc := api.GetExecutionGrpcService(); execSvc != nil {
		dagflowpb.RegisterExecutionServiceServer(s, execSvc)
		logger.Info("gRPC: Execution service registered")
	}

	grpcServer := &GrpcServer{server: s, lis: lis}
	global.DefaultResourceManger.AddDaemon(grpcServer)
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
	dbClient = client

	// 注册 DB client 为 bean（FlowDAO 通过 autowired 注入）
	bean.SetBeanOverwrite(dbCfg.Name, client)

	// 自动建表
	ctx := context.Background()
	tables := []interface{}{
		// dagflow 业务表
		(*model.Flow)(nil),
		(*model.ExecutionRecord)(nil),
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
			logger.Error("failed to create table: %v", err.Error())
		}
	}

	// 初始化 DAO
	dao.Init()
	dao.InitExecutionRecord()

	logger.Info("database initialized successfully")
}

// initTaskx 初始化 taskx 集群调度基础设施
// 在 bean.Ioc() 之前调用，注册 cluster 和 taskx DAO beans
func initTaskx() {
	// 从 default.yaml 读取集群配置
	clusterCfg := constants.DefaultConfig.ClusterConfig

	c, err := cluster.NewClusterWithArgs(clusterCfg, logger.NewLogger(&logger.Config{}))
	if err != nil {
		panic("failed to create cluster: " + err.Error())
	}
	// 注册 cluster 为 bean，供 taskx dispatcher/receiver autowired 注入
	dagflowCluster = c
	bean.AddBean(c)

	// 初始化 taskx dispatcher（使用已有 DB client）
	taskx.InitTaskDispatcherWithDB(&taskx.Config{
		SubtaskWorker:            50,
		SubtaskQueueSize:         200,
		SubtaskRollbackWorker:    10,
		SubtaskRollbackQueueSize: 100,
		TaskWorker:               5,
		TaskQueueSize:            100,
		RemoteCallTimeout:        3 * time.Second,
	}, dbClient)
}

// startTaskx 启动 taskx receiver 和 cluster（在 bean.Ioc() 之后调用）
func startTaskx() {
	if err := taskx.StartReceiver(); err != nil {
		panic("failed to start taskx receiver: " + err.Error())
	}

	tracker := cluster.NewDefaultJobTracker(5, taskx.SingletonTaskDispatcher)
	if err := dagflowCluster.AddJobTracker(tracker); err != nil {
		panic("failed to add job tracker: " + err.Error())
	}
	if err := dagflowCluster.Start(); err != nil {
		panic("failed to start cluster: " + err.Error())
	}

	// 等待集群就绪
	go func() {
		for i := 0; i < 30; i++ {
			if dagflowCluster.IsReady() {
				logger.Info("taskx cluster ready")
				return
			}
			time.Sleep(time.Second)
		}
		logger.Warn("taskx cluster not ready after 30s")
	}()
}
