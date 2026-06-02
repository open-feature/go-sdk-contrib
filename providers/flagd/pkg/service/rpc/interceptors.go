package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/flagdmeta"
)

// selectorInterceptor is a connect.Interceptor that adds the flagd-selector header.
type selectorInterceptor struct {
	selector string
}

func newSelectorInterceptor(selector string) *selectorInterceptor {
	return &selectorInterceptor{selector: selector}
}

func (i *selectorInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if i.selector != "" {
			req.Header().Set(flagdmeta.Selector, i.selector)
		}
		return next(ctx, req)
	}
}

func (i *selectorInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if i.selector != "" {
			conn.RequestHeader().Set(flagdmeta.Selector, i.selector)
		}
		return conn
	}
}

func (i *selectorInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	// server-side; not used by the client
	return next
}
