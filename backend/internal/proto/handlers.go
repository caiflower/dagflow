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

package proto

import (
	"context"

	"google.golang.org/grpc"
)

// Exported handler wrappers for engine.GRPC() registration.
//
// 命名规则: Flow_<Method>_Handler (大写开头 = exported)
// engine.GRPC() 的正则 `_([^_]+)_Handler` 从 "Flow_Create_Handler" 中提取 "Create"，
// 再通过反射在 srv 实例上查找 Create 方法。

// ===== FlowService =====

func Flow_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_Create_Handler(srv, ctx, dec, interceptor)
}

func Flow_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_Get_Handler(srv, ctx, dec, interceptor)
}

func Flow_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_List_Handler(srv, ctx, dec, interceptor)
}

func Flow_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_Update_Handler(srv, ctx, dec, interceptor)
}

func Flow_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_Delete_Handler(srv, ctx, dec, interceptor)
}

func Flow_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _FlowService_Validate_Handler(srv, ctx, dec, interceptor)
}

// ===== ProtocolService =====

func Protocol_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ProtocolService_List_Handler(srv, ctx, dec, interceptor)
}

func Protocol_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ProtocolService_Get_Handler(srv, ctx, dec, interceptor)
}

// ===== ExecutionService =====

func Execution_Run_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ExecutionService_Run_Handler(srv, ctx, dec, interceptor)
}

func Execution_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ExecutionService_Get_Handler(srv, ctx, dec, interceptor)
}

func Execution_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ExecutionService_List_Handler(srv, ctx, dec, interceptor)
}

func ExecutionService_Retry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _ExecutionService_Retry_Handler(srv, ctx, dec, interceptor)
}
