package grpc_service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
		var credentials credentials.TransportCredentials
		var err error
		var address string

		// Handle certificate
		if s.GRPCServiceConfiguration.CertificatePath != "" {
			credentials, err = loadTLSCredentials(s.GRPCServiceConfiguration.CertificatePath)
			if err != nil {
				log.Error(err)
				credentials = insecure.NewCredentials()
			}
		} else {
			credentials = insecure.NewCredentials()
		}

		// Handle unix socket
		if s.GRPCServiceConfiguration.SocketPath != "" {
			address = fmt.Sprintf("passthrough:///unix://%s", s.GRPCServiceConfiguration.SocketPath)
		} else {
			address = fmt.Sprintf("%s:%d", s.GRPCServiceConfiguration.Host, s.GRPCServiceConfiguration.Port)
		}

		// Dial
		conn, err := grpc.Dial(
			address,
			grpc.WithTransportCredentials(credentials),
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

func loadTLSCredentials(serverCertPath string) (credentials.TransportCredentials, error) {
	pemServerCA, err := os.ReadFile(serverCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}
