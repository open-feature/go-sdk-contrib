package rpc

import (
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"github.com/bufbuild/connect-go"
)

// MockClient is a test mock for service client
type MockClient struct {
	booleanResponse v1.ResolveBooleanResponse
	stringResponse  v1.ResolveStringResponse
	floatResponse   v1.ResolveFloatResponse
	intResponse     v1.ResolveIntResponse
	objResponse     v1.ResolveObjectResponse

	error error
}

func (m *MockClient) ResolveBoolean(context.Context, *connect.Request[v1.ResolveBooleanRequest]) (*connect.Response[v1.ResolveBooleanResponse], error) {
	return &connect.Response[v1.ResolveBooleanResponse]{
		Msg: &m.booleanResponse,
	}, m.error
}

func (m *MockClient) ResolveString(context.Context, *connect.Request[v1.ResolveStringRequest]) (*connect.Response[v1.ResolveStringResponse], error) {
	return &connect.Response[v1.ResolveStringResponse]{
		Msg: &m.stringResponse,
	}, m.error
}

func (m *MockClient) ResolveFloat(context.Context, *connect.Request[v1.ResolveFloatRequest]) (*connect.Response[v1.ResolveFloatResponse], error) {
	return &connect.Response[v1.ResolveFloatResponse]{
		Msg: &m.floatResponse,
	}, m.error
}

func (m *MockClient) ResolveInt(context.Context, *connect.Request[v1.ResolveIntRequest]) (*connect.Response[v1.ResolveIntResponse], error) {
	return &connect.Response[v1.ResolveIntResponse]{
		Msg: &m.intResponse,
	}, m.error
}

func (m *MockClient) ResolveObject(context.Context, *connect.Request[v1.ResolveObjectRequest]) (*connect.Response[v1.ResolveObjectResponse], error) {
	return &connect.Response[v1.ResolveObjectResponse]{
		Msg: &m.objResponse,
	}, m.error
}

func (m *MockClient) EventStream(context.Context, *connect.Request[v1.EventStreamRequest]) (*connect.ServerStreamForClient[v1.EventStreamResponse], error) {
	// note - mocking this is impossible
	return &connect.ServerStreamForClient[v1.EventStreamResponse]{}, m.error
}

func (m *MockClient) ResolveAll(context.Context, *connect.Request[v1.ResolveAllRequest]) (*connect.Response[v1.ResolveAllResponse], error) {
	return &connect.Response[v1.ResolveAllResponse]{
		Msg: &v1.ResolveAllResponse{},
	}, m.error
}
