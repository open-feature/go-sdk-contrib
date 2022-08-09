package http_service

type HTTPServiceOption func(*HTTPService)

type ProtocolType int

const (
	HTTP ProtocolType = iota
	HTTPS
)

// NewHTTPService creates a new HTTPService taking configuration options to override default values
func NewHTTPService(opts ...HTTPServiceOption) *HTTPService {
	const (
		port = 8013
		host = "localhost"
	)
	svc := &HTTPService{
		HTTPServiceConfiguration: &HTTPServiceConfiguration{
			Port:     port,
			Host:     host,
			Protocol: "http",
		},
		Client: &httpClient{},
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// WithPort overrides the default flagd http port (8013)
func WithPort(port uint16) HTTPServiceOption {
	return func(s *HTTPService) {
		s.HTTPServiceConfiguration.Port = port
	}
}

// WithHost overrides the default flagd host name (localhost)
func WithHost(host string) HTTPServiceOption {
	return func(s *HTTPService) {
		s.HTTPServiceConfiguration.Host = host
	}
}

// WithProtocol overrides the default protocol (http) (currently only http is supported)
func WithProtocol(protocol ProtocolType) HTTPServiceOption {
	if protocol == HTTPS {
		return func(s *HTTPService) {
			s.HTTPServiceConfiguration.Protocol = "https"
		}
	}
	return func(s *HTTPService) {
		s.HTTPServiceConfiguration.Protocol = "http"
	}
}
