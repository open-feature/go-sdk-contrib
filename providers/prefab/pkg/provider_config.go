package prefab

type ProviderConfig struct {
	Configs map[string]interface{}
	APIKey  string
	APIURLs []string
	Sources []string
	// EnvironmentNames             []string
	// ProjectEnvID                 int64
	// InitializationTimeoutSeconds float64
	// OnInitializationFailure      OnInitializationFailure
}
