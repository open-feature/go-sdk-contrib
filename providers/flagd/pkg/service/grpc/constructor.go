package grpc_service

type GRPCServiceOption func(*GRPCServiceConfiguration)

// NewGRPCService creates a new GRPCService taking configuration options to overide default values
func NewGRPCService(opts ...GRPCServiceOption) *GRPCService {
	const (
		port = 8080
		host = "localhost"
	)
	serviceConfiguration := &GRPCServiceConfiguration{
		Port: port,
		Host: host,
	}
	svc := &GRPCService{
		Client: &gRPCClient{
			GRPCServiceConfiguration: serviceConfiguration,
		},
	}
	for _, opt := range opts {
		opt(serviceConfiguration)
	}
	return svc
}

// WithPort overides the default flagd dial port (8080)
func WithPort(port uint16) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Port = port
	}
}

// WithHost overides the flagd dial host name (localhost)
func WithHost(host string) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Host = host
	}
}
