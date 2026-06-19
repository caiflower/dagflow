package api

// gRPC 服务实例（包级变量，由 main.go 设置）
var (
	flowGrpcSvc     *FlowGrpcService
	protocolGrpcSvc *ProtocolGrpcService
	executionGrpcSvc *ExecutionGrpcService
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
