package grpc_service

type GRPCServiceOption func(*gRPCServiceConfiguration)

// NewGRPCService creates a new GRPCService taking configuration options to overide default values
func NewGRPCService(opts ...GRPCServiceOption) *GRPCService {
	const (
		port = 8080
		host = "localhost"
	)
	serviceConfiguration := &gRPCServiceConfiguration{
		port: port,
		host: host,
	}
	svc := &GRPCService{
		client: &gRPCClient{
			gRPCServiceConfiguration: serviceConfiguration,
		},
	}
	for _, opt := range opts {
		opt(serviceConfiguration)
	}
	return svc
}

// WithPort overides the default flagd dial port (8080)
func WithPort(port int32) GRPCServiceOption {
	return func(s *gRPCServiceConfiguration) {
		s.port = port
	}
}

// WithHost overides the flagd dial host name (localhost)
func WithHost(host string) GRPCServiceOption {
	return func(s *gRPCServiceConfiguration) {
		s.host = host
	}
}
