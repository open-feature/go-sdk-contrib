package prefab

type ProviderConfig struct {
	Configs map[string]any
	APIKey  string
	APIURLs []string
	Sources []string
	// EnvironmentNames             []string
	// ProjectEnvID                 int64
	// InitializationTimeoutSeconds float64
	//nolint OnInitializationFailure      OnInitializationFailure
}
