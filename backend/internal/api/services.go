package api

import (
	"github.com/caiflower/dagflow/internal/node_registry"
)

// gRPC 服务实例（包级变量，由 main.go 设置）
var (
	flowGrpcSvc      *FlowGrpcService
	protocolGrpcSvc  *ProtocolGrpcService
	executionGrpcSvc *ExecutionGrpcService
	nodeRegSvc       *node_registry.NodeRegistry
)

// SetFlowGrpcService 设置 Flow gRPC 服务
func SetFlowGrpcService(svc *FlowGrpcService) {
	flowGrpcSvc = svc
}

// SetProtocolGrpcService 设置 Protocol gRPC 服务
func SetProtocolGrpcService(svc *ProtocolGrpcService) {
	protocolGrpcSvc = svc
}

// SetExecutionGrpcService 设置 Execution gRPC 服务
func SetExecutionGrpcService(svc *ExecutionGrpcService) {
	executionGrpcSvc = svc
}

// SetNodeRegistryService 设置 Node Registry 服务
func SetNodeRegistryService(svc *node_registry.NodeRegistry) {
	nodeRegSvc = svc
}

// GetFlowGrpcService 获取 Flow gRPC 服务
func GetFlowGrpcService() *FlowGrpcService {
	return flowGrpcSvc
}

// GetProtocolGrpcService 获取 Protocol gRPC 服务
func GetProtocolGrpcService() *ProtocolGrpcService {
	return protocolGrpcSvc
}

// GetExecutionGrpcService 获取 Execution gRPC 服务
func GetExecutionGrpcService() *ExecutionGrpcService {
	return executionGrpcSvc
}
