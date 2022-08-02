package http_service

import (
	"testing"
)

type TestConstructorArgs struct {
	name    string
	port    int32
	host    string
	options []HTTPServiceOption
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
			options: []HTTPServiceOption{
				WithHost("not localhost"),
			},
		},
		{
			name: "withPort",
			port: 1,
			host: "localhost",
			options: []HTTPServiceOption{
				WithPort(1),
			},
		},
	}
	for _, test := range tests {
		svc := NewHTTPService(test.options...)
		if svc == nil {
			t.Error("recieved nil service from NewHTTPService")
			t.FailNow()
		}
		if svc.httpServiceConfiguration == nil {
			t.Error("svc.HTTPServiceConfiguration is nil")
			t.FailNow()
		}
		if svc.httpServiceConfiguration.host != test.host {
			t.Errorf(
				"recieved unexpected HTTPServiceConfiguration.Host from NewHTTPService, expected %s got %s",
				test.host,
				svc.httpServiceConfiguration.host,
			)
		}
		if svc.httpServiceConfiguration.port != test.port {
			t.Errorf(
				"recieved unexpected HTTPServiceConfiguration.Port from NewHTTPService, expected %d got %d",
				test.port,
				svc.httpServiceConfiguration.port,
			)
		}

	}
}
