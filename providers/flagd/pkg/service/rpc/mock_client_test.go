package rpc

import (
	v2 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/evaluation/v2"
	"connectrpc.com/connect"
	"context"
)

// MockClient is a test mock for service client
type MockClient struct {
	booleanResponse v2.ResolveBooleanResponse
	stringResponse  v2.ResolveStringResponse
	floatResponse   v2.ResolveFloatResponse
	intResponse     v2.ResolveIntResponse
	objResponse     v2.ResolveObjectResponse

	error error
}

func (m *MockClient) ResolveBoolean(context.Context, *connect.Request[v2.ResolveBooleanRequest]) (*connect.Response[v2.ResolveBooleanResponse], error) {
	return &connect.Response[v2.ResolveBooleanResponse]{
		Msg: &m.booleanResponse,
	}, m.error
}

func (m *MockClient) ResolveString(context.Context, *connect.Request[v2.ResolveStringRequest]) (*connect.Response[v2.ResolveStringResponse], error) {
	return &connect.Response[v2.ResolveStringResponse]{
		Msg: &m.stringResponse,
	}, m.error
}

func (m *MockClient) ResolveFloat(context.Context, *connect.Request[v2.ResolveFloatRequest]) (*connect.Response[v2.ResolveFloatResponse], error) {
	return &connect.Response[v2.ResolveFloatResponse]{
		Msg: &m.floatResponse,
	}, m.error
}

func (m *MockClient) ResolveInt(context.Context, *connect.Request[v2.ResolveIntRequest]) (*connect.Response[v2.ResolveIntResponse], error) {
	return &connect.Response[v2.ResolveIntResponse]{
		Msg: &m.intResponse,
	}, m.error
}

func (m *MockClient) ResolveObject(context.Context, *connect.Request[v2.ResolveObjectRequest]) (*connect.Response[v2.ResolveObjectResponse], error) {
	return &connect.Response[v2.ResolveObjectResponse]{
		Msg: &m.objResponse,
	}, m.error
}

func (m *MockClient) EventStream(context.Context, *connect.Request[v2.EventStreamRequest]) (*connect.ServerStreamForClient[v2.EventStreamResponse], error) {
	return &connect.ServerStreamForClient[v2.EventStreamResponse]{}, m.error
}
