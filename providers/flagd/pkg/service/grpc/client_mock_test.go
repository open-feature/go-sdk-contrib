package grpc_service_test

import (
	service "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/grpc"
	schemaV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
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
