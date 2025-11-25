package ofrephandler

type Configuration struct {
	requirePathPrefix bool
	requirePOST       bool
	pathValueName     string
}

type Option func(*Configuration)

// WithoutPathPrefix makes the handler only accept requests from any path prefix.
// By default, it requires the path to start with "/ofrep/v1/evaluate/flags/".
func WithoutPathPrefix() Option {
	return func(conf *Configuration) {
		conf.requirePathPrefix = false
	}
}

// WithoutOnlyPOST makes the handler accept requests with any HTTP method.
// By default, it requires POST requests.
func WithoutOnlyPOST() Option {
	return func(conf *Configuration) {
		conf.requirePOST = false
	}
}

// WithKeyPathValueName sets the name of the path value to extract the flag key from.
// The default is "key".
func WithKeyPathValueName(pathValue string) Option {
	return func(conf *Configuration) {
		conf.pathValueName = pathValue
	}
}
