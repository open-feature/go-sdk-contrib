package grpc_service

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type IGRPCClient interface {
	GetInstance() schemaV1.ServiceClient
}

type GRPCClient struct {
	conn                     *grpc.ClientConn
	GRPCServiceConfiguration *GRPCServiceConfiguration
}

func (s *GRPCClient) Connect() {
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

func (s *GRPCClient) GetInstance() schemaV1.ServiceClient {
	s.Connect()
	if s.conn == nil {
		return nil
	}
	return schemaV1.NewServiceClient(s.conn)
}
