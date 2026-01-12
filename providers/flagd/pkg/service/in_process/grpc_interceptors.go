package process

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// selectorUnaryInterceptor adds the flagd-selector metadata header to unary gRPC calls
func selectorUnaryInterceptor(selector string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if selector != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "flagd-selector", selector)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// selectorStreamInterceptor adds the flagd-selector metadata header to streaming gRPC calls
func selectorStreamInterceptor(selector string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if selector != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "flagd-selector", selector)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
