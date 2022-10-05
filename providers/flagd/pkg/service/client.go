package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

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
func (c *Client) Instance() schemaConnectV1.ServiceClient {
	if c.client == nil {
		var dialContext func(ctx context.Context, network string, addr string) (net.Conn, error)
		var tlsConfig *tls.Config
		var url string = fmt.Sprintf("http://%s:%d", c.ServiceConfiguration.Host, c.ServiceConfiguration.Port)
		// socket
		if c.ServiceConfiguration.SocketPath != "" {
			dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", c.ServiceConfiguration.SocketPath)
			}
		}
		// cert
		if c.ServiceConfiguration.CertificatePath != "" {
			url = fmt.Sprintf("https://%s:%d", c.ServiceConfiguration.Host, c.ServiceConfiguration.Port)
			caCert, err := os.ReadFile(c.ServiceConfiguration.CertificatePath)
			if err != nil {
				log.Fatal(err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig = &tls.Config{
				RootCAs: caCertPool,
			}
		}

		c.client = schemaConnectV1.NewServiceClient(
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
					DialContext:     dialContext,
				},
			},
			url,
		)
	}
	return c.client
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
