package http_service_test

import (
	"testing"

	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
)

type TestConstructorArgs struct {
	name    string
	port    int32
	host    string
	options []service.HTTPServiceOption
}

func TestNewHTTPService(t *testing.T) {
	tests := []TestConstructorArgs{
		{
			name:    "default",
			port:    8080,
			host:    "localhost",
			options: nil,
		},
		{
			name: "withHost",
			port: 8080,
			host: "not localhost",
			options: []service.HTTPServiceOption{
				service.WithHost("not localhost"),
			},
		},
		{
			name: "withPort",
			port: 1,
			host: "localhost",
			options: []service.HTTPServiceOption{
				service.WithPort(1),
			},
		},
	}
	for _, test := range tests {
		svc := service.NewHTTPService(test.options...)
		if svc == nil {
			t.Error("recieved nil service from NewHTTPService")
			t.FailNow()
		}
		if svc.HttpServiceConfiguration == nil {
			t.Error("svc.HTTPServiceConfiguration is nil")
			t.FailNow()
		}
		if svc.HttpServiceConfiguration.Host != test.host {
			t.Errorf(
				"recieved unexpected HTTPServiceConfiguration.Host from NewHTTPService, expected %s got %s",
				test.host,
				svc.HttpServiceConfiguration.Host,
			)
		}
		if svc.HttpServiceConfiguration.Port != test.port {
			t.Errorf(
				"recieved unexpected HTTPServiceConfiguration.Port from NewHTTPService, expected %d got %d",
				test.port,
				svc.HttpServiceConfiguration.Port,
			)
		}

	}
}
