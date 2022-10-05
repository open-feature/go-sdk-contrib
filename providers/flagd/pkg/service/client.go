package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	schemaConnectV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1/schemav1connect"
)

type iClient interface {
	Instance() schemaConnectV1.ServiceClient
	Configuration() *ServiceConfiguration
}

type Client struct {
	client               schemaConnectV1.ServiceClient
	ServiceConfiguration *ServiceConfiguration
}

// Instance returns an instance of schemaConnectV1.ServiceClient
func (s *Client) Instance() schemaConnectV1.ServiceClient {
	if s.client == nil {
		var err error
		var address string

		// Handle certificate
		tlsConfig, err := loadTLSConfig(s.ServiceConfiguration.CertificatePath)
		if err != nil {
			log.Errorf("connect - fail to load tls credentials: %v", err)
		}

		// Handle unix socket
		if s.ServiceConfiguration.SocketPath != "" {
			address = fmt.Sprintf("passthrough:///unix://%s", s.ServiceConfiguration.SocketPath)
		} else {
			address = fmt.Sprintf("%s:%d", s.ServiceConfiguration.Host, s.ServiceConfiguration.Port)
		}

		t := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		s.client = schemaConnectV1.NewServiceClient(
			&http.Client{
				Transport: t,
			},
			address,
		)
	}
	return s.client
}

// Configuration returns the current GRPCServiceConfiguration for the client
func (s *Client) Configuration() *ServiceConfiguration {
	return s.ServiceConfiguration
}

func loadTLSConfig(serverCertPath string) (*tls.Config, error) {
	if serverCertPath == "" {
		return nil, nil
	}
	pemServerCA, err := os.ReadFile(serverCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	return &tls.Config{
		RootCAs: certPool,
	}, nil
}
