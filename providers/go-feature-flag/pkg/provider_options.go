package gofeatureflag

// ProviderOptions is the struct containing the provider options you can
// use while initializing GO Feature Flag
type ProviderOptions struct {
	// Endpoint (mandatory) contains the DNS of your GO Feature Flag relay proxy (ex: http://localhost:1031)
	Endpoint string

	// HTTPClient (optional) is the HTTP Client we will use to contact GO Feature Flag.
	// By default, we are using a custom HTTPClient with a timeout configure to 10000 milliseconds.
	HTTPClient HTTPClient
}
