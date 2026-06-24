package remote_executor

import (
	"context"

	"google.golang.org/grpc"
)

// NodeRegistry_ListNodes_Handler exports the generated ListNodes handler for engine.GRPC().
func NodeRegistry_ListNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _NodeRegistry_ListNodes_Handler(srv, ctx, dec, interceptor)
}

// NodeRegistry_GetNode_Handler exports the generated GetNode handler for engine.GRPC().
func NodeRegistry_GetNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return _NodeRegistry_GetNode_Handler(srv, ctx, dec, interceptor)
}
