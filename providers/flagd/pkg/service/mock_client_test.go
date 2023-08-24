package service

import (
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	connect "github.com/bufbuild/connect-go"
)

// MockClient is a test mock for service client
type MockClient struct {
}

func (m *MockClient) ResolveAll(context.Context, *connect.Request[v1.ResolveAllRequest]) (*connect.Response[v1.ResolveAllResponse], error) {
	return &connect.Response[v1.ResolveAllResponse]{
		Msg: &v1.ResolveAllResponse{},
	}, nil
}

func (m *MockClient) ResolveBoolean(context.Context, *connect.Request[v1.ResolveBooleanRequest]) (*connect.Response[v1.ResolveBooleanResponse], error) {
	return &connect.Response[v1.ResolveBooleanResponse]{
		Msg: &v1.ResolveBooleanResponse{},
	}, nil
}

func (m *MockClient) ResolveString(context.Context, *connect.Request[v1.ResolveStringRequest]) (*connect.Response[v1.ResolveStringResponse], error) {
	return &connect.Response[v1.ResolveStringResponse]{
		Msg: &v1.ResolveStringResponse{},
	}, nil
}

func (m *MockClient) ResolveFloat(context.Context, *connect.Request[v1.ResolveFloatRequest]) (*connect.Response[v1.ResolveFloatResponse], error) {
	return &connect.Response[v1.ResolveFloatResponse]{
		Msg: &v1.ResolveFloatResponse{},
	}, nil
}

func (m *MockClient) ResolveInt(context.Context, *connect.Request[v1.ResolveIntRequest]) (*connect.Response[v1.ResolveIntResponse], error) {
	return &connect.Response[v1.ResolveIntResponse]{
		Msg: &v1.ResolveIntResponse{},
	}, nil
}

func (m *MockClient) ResolveObject(context.Context, *connect.Request[v1.ResolveObjectRequest]) (*connect.Response[v1.ResolveObjectResponse], error) {
	return &connect.Response[v1.ResolveObjectResponse]{
		Msg: &v1.ResolveObjectResponse{},
	}, nil
}

func (m *MockClient) EventStream(context.Context, *connect.Request[v1.EventStreamRequest]) (*connect.ServerStreamForClient[v1.EventStreamResponse], error) {
	return &connect.ServerStreamForClient[v1.EventStreamResponse]{}, nil
}
