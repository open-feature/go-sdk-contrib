package from_env

import (
	"encoding/json"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"os"
)

type envFetch struct{}

func (ef *envFetch) fetchStoredFlag(key string) (StoredFlag, error) {
	v := StoredFlag{}
	if val := os.Getenv(key); val != "" {
		if err := json.Unmarshal([]byte(val), &v); err != nil {
			return v, openfeature.NewParseErrorResolutionError(err.Error())
		}
		return v, nil
	}
	return v, openfeature.NewFlagNotFoundResolutionError("")
}
