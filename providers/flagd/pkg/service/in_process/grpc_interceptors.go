package process

import (
	"context"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// selectorUnaryInterceptor adds the flagd-selector metadata header to unary gRPC calls
func selectorUnaryInterceptor(selector string) googlegrpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *googlegrpc.ClientConn,
		invoker googlegrpc.UnaryInvoker,
		opts ...googlegrpc.CallOption,
	) error {
		if selector != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "flagd-selector", selector)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// selectorStreamInterceptor adds the flagd-selector metadata header to streaming gRPC calls
func selectorStreamInterceptor(selector string) googlegrpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *googlegrpc.StreamDesc,
		cc *googlegrpc.ClientConn,
		method string,
		streamer googlegrpc.Streamer,
		opts ...googlegrpc.CallOption,
	) (googlegrpc.ClientStream, error) {
		if selector != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "flagd-selector", selector)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
