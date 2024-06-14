package from_env

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-feature/go-sdk/openfeature"
)

type envFetch struct {
	mapper FlagToEnvMapper
}

func (ef *envFetch) fetchStoredFlag(key string) (StoredFlag, error) {
	v := StoredFlag{}
	mappedKey := key

	if ef.mapper != nil {
		mappedKey = ef.mapper(key)
	}

	if val := os.Getenv(mappedKey); val != "" {
		if err := json.Unmarshal([]byte(val), &v); err != nil {
			return v, openfeature.NewParseErrorResolutionError(err.Error())
		}
		return v, nil
	}

	msg := fmt.Sprintf("key %s not found in environment variables", mappedKey)

	return v, openfeature.NewFlagNotFoundResolutionError(msg)
}
