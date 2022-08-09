package http_service_test

import (
	"testing"

	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
)

type TestConstructorArgs struct {
	name     string
	port     uint16
	host     string
	protocol string
	options  []service.HTTPServiceOption
}

func TestNewHTTPService(t *testing.T) {
	tests := []TestConstructorArgs{
		{
			name:     "default",
			port:     8013,
			host:     "localhost",
			protocol: "http",
			options:  nil,
		},
		{
			name:     "withHost",
			port:     8013,
			host:     "not localhost",
			protocol: "http",
			options: []service.HTTPServiceOption{
				service.WithHost("not localhost"),
			},
		},
		{
			name:     "withPort",
			port:     1,
			host:     "localhost",
			protocol: "http",
			options: []service.HTTPServiceOption{
				service.WithPort(1),
			},
		},
		{
			name:     "withProtocol",
			port:     8013,
			host:     "localhost",
			protocol: "https",
			options: []service.HTTPServiceOption{
				service.WithProtocol(service.HTTPS),
			},
		},
		{
			name:     "withProtocol http",
			port:     8013,
			host:     "localhost",
			protocol: "http",
			options: []service.HTTPServiceOption{
				service.WithProtocol(service.HTTP),
			},
		},
	}
	for _, test := range tests {
		svc := service.NewHTTPService(test.options...)
		if svc == nil {
			t.Error("received nil service from NewHTTPService")
			t.FailNow()
		}
		if svc.HTTPServiceConfiguration == nil {
			t.Error("svc.HTTPServiceConfiguration is nil")
			t.FailNow()
		}
		if svc.HTTPServiceConfiguration.Host != test.host {
			t.Errorf(
				"received unexpected HTTPServiceConfiguration.Host from NewHTTPService, expected %s got %s",
				test.host,
				svc.HTTPServiceConfiguration.Host,
			)
		}
		if svc.HTTPServiceConfiguration.Port != test.port {
			t.Errorf(
				"received unexpected HTTPServiceConfiguration.Port from NewHTTPService, expected %d got %d",
				test.port,
				svc.HTTPServiceConfiguration.Port,
			)
		}
		if svc.HTTPServiceConfiguration.Protocol != test.protocol {
			t.Errorf(
				"received unexpected HTTPServiceConfiguration.Protocol from NewHTTPService, expected %s got %s",
				test.protocol,
				svc.HTTPServiceConfiguration.Protocol,
			)
		}

	}
}
