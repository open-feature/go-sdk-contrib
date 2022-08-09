package grpc_service

type GRPCServiceOption func(*GRPCServiceConfiguration)

// NewGRPCService creates a new GRPCService taking configuration options to override default values
func NewGRPCService(opts ...GRPCServiceOption) *GRPCService {
	const (
		port = 8013
		host = "localhost"
	)
	serviceConfiguration := &GRPCServiceConfiguration{
		Port:            port,
		Host:            host,
		CertificatePath: "",
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

// WithPort overrides the default flagd dial port (8013)
func WithPort(port uint16) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Port = port
	}
}

// WithHost overrides the flagd dial host name (localhost)
func WithHost(host string) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.Host = host
	}
}

// WithCertificatePath specifies the fliepath of the certificate to be used in the gRPC dial credentials. If certificate loading fails or no path is provided insecure credentials will be used instead
func WithCertificatePath(cPath string) GRPCServiceOption {
	return func(s *GRPCServiceConfiguration) {
		s.CertificatePath = cPath
	}
}
