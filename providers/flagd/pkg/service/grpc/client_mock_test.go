package grpc_service_test

import (
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type MockClient struct {
	Client    schemaV1.ServiceClient
	NilClient bool
}

func (m *MockClient) Instance() schemaV1.ServiceClient {
	if m.NilClient {
		return nil
	}
	return m.Client
}

func (m *MockClient) Configuration() *service.GRPCServiceConfiguration {
	return nil
}
