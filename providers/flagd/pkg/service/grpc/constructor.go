package grpc_service

type GRPCServiceOption func(*GRPCServiceConfiguration)

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
		Client: &GRPCClient{
			GRPCServiceConfiguration: serviceConfiguration,
		},
	}
	for _, opt := range opts {
		opt(serviceConfiguration)
	}
	return svc
}

func WithPort(port int32) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Port = port
	}
}

func WithHost(host string) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Host = host
	}
}
