package service

import (
	schemaConnectV1 "buf.build/gen/go/open-feature/flagd/bufbuild/connect-go/schema/v1/schemav1connect"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/bufbuild/connect-go"
	otelconnect "github.com/bufbuild/connect-opentelemetry-go"
	"log"
	"net"
	"net/http"
	"os"
)

type Configuration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
	TLSEnabled      bool
	OtelInterceptor bool
}

type Client struct {
	client schemaConnectV1.ServiceClient
}

func NewClient(cfg *Configuration) Client {
	var dialContext func(ctx context.Context, network string, addr string) (net.Conn, error)
	var tlsConfig *tls.Config
	url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	// socket
	if cfg.SocketPath != "" {
		dialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", cfg.SocketPath)
		}
	}
	// tls
	if cfg.TLSEnabled {
		url = fmt.Sprintf("https://%s:%d", cfg.Host, cfg.Port)
		tlsConfig = &tls.Config{}
		if cfg.CertificatePath != "" {
			caCert, err := os.ReadFile(cfg.CertificatePath)
			if err != nil {
				log.Fatal(err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				log.Fatalf(
					"failed to AppendCertsFromPEM, certificate %s is malformed",
					cfg.CertificatePath,
				)
			}
			tlsConfig.RootCAs = caCertPool
		}
	}

	// build options
	var options []connect.ClientOption

	if cfg.OtelInterceptor {
		options = append(options, connect.WithInterceptors(
			otelconnect.NewInterceptor(),
		))
	}

	return Client{
		client: schemaConnectV1.NewServiceClient(
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
					DialContext:     dialContext,
				},
			},
			url,
			options...,
		),
	}
}

// Instance returns an instance of schemaConnectV1.ServiceClient
func (c *Client) Instance() schemaConnectV1.ServiceClient {
	return c.client
}
