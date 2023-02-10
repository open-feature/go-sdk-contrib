package service

import (
	schemaConnectV1 "buf.build/gen/go/open-feature/flagd/bufbuild/connect-go/schema/v1/schemav1connect"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
		url := fmt.Sprintf("http://%s:%d", c.ServiceConfiguration.Host, c.ServiceConfiguration.Port)
		// socket
		if c.ServiceConfiguration.SocketPath != "" {
			dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", c.ServiceConfiguration.SocketPath)
			}
		}
		// tls
		if c.ServiceConfiguration.TLSEnabled {
			url = fmt.Sprintf("https://%s:%d", c.ServiceConfiguration.Host, c.ServiceConfiguration.Port)
			tlsConfig = &tls.Config{}
			if c.ServiceConfiguration.CertificatePath != "" {
				caCert, err := os.ReadFile(c.ServiceConfiguration.CertificatePath)
				if err != nil {
					log.Fatal(err)
				}
				caCertPool := x509.NewCertPool()
				if !caCertPool.AppendCertsFromPEM(caCert) {
					log.Fatalf(
						"failed to AppendCertsFromPEM, certificate %s is malformed",
						c.ServiceConfiguration.CertificatePath,
					)
				}
				tlsConfig.RootCAs = caCertPool
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
