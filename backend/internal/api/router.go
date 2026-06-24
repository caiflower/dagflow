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

package api

import (
	"context"
	"fmt"

	"github.com/caiflower/common-tools/web"

	pb "github.com/caiflower/dagflow/internal/proto"
	remote_executor "github.com/caiflower/dagflow/proto/remote_executor"
)

// GetNodeReq 获取节点请求
type GetNodeReq struct {
	ID string `path:"id" verf:"required"`
}

// RegisterRoutes 注册所有 API 路由（统一 gRPC handler 方式）
func RegisterRoutes(engine *web.Engine) {
	engine.Use(corsMiddleware)
	engine.Use(requestLogMiddleware)
	engine.Use(recoveryMiddleware)

	v1 := engine.Group("/api/v1")

	// ===== Flow Service =====
	v1.GRPC("POST", "/flows", pb.Flow_Create_Handler, flowGrpcSvc)
	v1.GRPC("GET", "/flows", pb.Flow_List_Handler, flowGrpcSvc)
	v1.GRPC("GET", "/flows/:id", pb.Flow_Get_Handler, flowGrpcSvc)
	v1.GRPC("PUT", "/flows/:id", pb.Flow_Update_Handler, flowGrpcSvc)
	v1.GRPC("DELETE", "/flows/:id", pb.Flow_Delete_Handler, flowGrpcSvc)
	v1.GRPC("POST", "/flows/validate", pb.Flow_Validate_Handler, flowGrpcSvc)

	// ===== Protocol Service =====
	v1.GRPC("GET", "/protocols", pb.Protocol_List_Handler, protocolGrpcSvc)
	v1.GRPC("GET", "/protocols/:name", pb.Protocol_Get_Handler, protocolGrpcSvc)

	// ===== Execution Service =====
	v1.GRPC("POST", "/executions/run", pb.Execution_Run_Handler, executionGrpcSvc)
	v1.GRPC("GET", "/executions/:id", pb.Execution_Get_Handler, executionGrpcSvc)
	v1.GRPC("GET", "/executions", pb.Execution_List_Handler, executionGrpcSvc)

	// ===== Node Registry Service =====
	v1.GET("/nodes", listNodesHandler)
	v1.GET("/nodes/:id", getNodeHandler)

	// 健康检查
	v1.GET("/health", healthCheck)
}

// listNodesHandler 列出所有注册节点
func listNodesHandler(ctx context.Context) (*remote_executor.ListNodesResponse, error) {
	if nodeRegSvc == nil {
		return nil, fmt.Errorf("node registry not initialized")
	}
	return nodeRegSvc.ListNodes(ctx, &remote_executor.ListNodesRequest{})
}

// getNodeHandler 获取单个节点详情
func getNodeHandler(ctx context.Context, req *GetNodeReq) (*remote_executor.NodeDetail, error) {
	if nodeRegSvc == nil {
		return nil, fmt.Errorf("node registry not initialized")
	}
	resp, err := nodeRegSvc.GetNode(ctx, &remote_executor.GetNodeRequest{NodeId: req.ID})
	if err != nil {
		return nil, err
	}
	return resp.Node, nil
}

// healthCheck 健康检查
func healthCheck(ctx context.Context) (interface{}, error) {
	return map[string]string{"status": "ok", "service": "dagflow"}, nil
}
