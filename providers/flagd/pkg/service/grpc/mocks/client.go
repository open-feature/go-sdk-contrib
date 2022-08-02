package mock_grpc_service

import (
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
