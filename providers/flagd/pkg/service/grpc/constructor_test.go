package grpc_service_test

import (
	"testing"

	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
)

type TestConstructorArgs struct {
	name     string
	port     uint16
	host     string
	protocol string
	options  []service.GRPCServiceOption
}

func TestNewGRPCService(t *testing.T) {
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
			options: []service.GRPCServiceOption{
				service.WithHost("not localhost"),
			},
		},
		{
			name:     "withPort",
			port:     1,
			host:     "localhost",
			protocol: "http",
			options: []service.GRPCServiceOption{
				service.WithPort(1),
			},
		},
	}
	for _, test := range tests {
		svc := service.NewGRPCService(test.options...)
		if svc == nil {
			t.Error("received nil service from NewGRPCService")
			t.FailNow()
		}
		config := svc.Client.Configuration()
		if config == nil {
			t.Error("config is nil")
			t.FailNow()
		}
		if config.Host != test.host {
			t.Errorf(
				"received unexpected GRPCServiceConfiguration.Host from NewGRPCService, expected %s got %s",
				test.host,
				config.Host,
			)
		}
		if config.Port != test.port {
			t.Errorf(
				"received unexpected GRPCServiceConfiguration.Port from NewGRPCService, expected %d got %d",
				test.port,
				config.Port,
			)
		}
	}
}
