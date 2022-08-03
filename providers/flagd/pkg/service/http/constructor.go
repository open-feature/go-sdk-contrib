package http_service

type HTTPServiceOption func(*HTTPService)

type ProtocolType int

const (
	HTTP ProtocolType = iota
	HTTPS
)

// NewHTTPService creates a new HTTPService taking configuration options to overide default values
func NewHTTPService(opts ...HTTPServiceOption) *HTTPService {
	const (
		port     = 8080
		host     = "localhost"
		protocol = HTTP
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

// WithPort overides the default flagd http port (8080)
func WithPort(port int32) HTTPServiceOption {
	return func(s *HTTPService) {
		s.HTTPServiceConfiguration.Port = port
	}
}

// WithHost overides the default flagd host name (localhost)
func WithHost(host string) HTTPServiceOption {
	return func(s *HTTPService) {
		s.HTTPServiceConfiguration.Host = host
	}
}

// WithProtocol overides the default flagd protocol (http) (currently only http is supported)
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
