package grpc_service

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type iGRPCClient interface {
	Instance() schemaV1.ServiceClient
	Configuration() *GRPCServiceConfiguration
}

type gRPCClient struct {
	conn                     *grpc.ClientConn
	GRPCServiceConfiguration *GRPCServiceConfiguration
}

func (s *gRPCClient) connect() {
	if s.conn == nil {
		conn, err := grpc.Dial(
			fmt.Sprintf("%s:%d", s.GRPCServiceConfiguration.Host, s.GRPCServiceConfiguration.Port),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			log.Errorf("grpc - fail to dial: %v", err)
			return
		}
		s.conn = conn
	}
}

// Instance returns an instance of schemaV1.ServiceClient using the shared *grpc.ClientConn
func (s *gRPCClient) Instance() schemaV1.ServiceClient {
	s.connect()
	if s.conn == nil {
		return nil
	}
	return schemaV1.NewServiceClient(s.conn)
}

// Configuration returns the current GRPCServiceConfiguration for the client
func (s *gRPCClient) Configuration() *GRPCServiceConfiguration {
	return s.GRPCServiceConfiguration
}
