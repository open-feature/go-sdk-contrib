package http_service

type HTTPServiceOption func(*HTTPService)

// NewHTTPService creates a new HTTPService taking configuration options to overide default values
func NewHTTPService(opts ...HTTPServiceOption) *HTTPService {
	const (
		port     = 8080
		host     = "localhost"
		protocol = "http"
	)
	svc := &HTTPService{
		httpServiceConfiguration: &httpServiceConfiguration{
			port:     port,
			host:     host,
			protocol: protocol,
		},
		client: &httpClient{},
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// WithPort overides the default flagd http port (8080)
func WithPort(port int32) HTTPServiceOption {
	return func(s *HTTPService) {
		s.httpServiceConfiguration.port = port
	}
}

// WithHost overides the default flagd host name (localhost)
func WithHost(host string) HTTPServiceOption {
	return func(s *HTTPService) {
		s.httpServiceConfiguration.host = host
	}
}

// WithProtocol overides the default flagd protocol (http) (currently only http is supported)
func WithProtocol(protocol string) HTTPServiceOption {
	return func(s *HTTPService) {
		s.httpServiceConfiguration.protocol = protocol
	}
}
