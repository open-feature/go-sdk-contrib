package from_env

import (
	"encoding/json"
	"errors"
	"os"
)

type EnvFetch struct{}

func (ef *EnvFetch) FetchStoredFlag(key string) (StoredFlag, error) {
	v := StoredFlag{}
	if val := os.Getenv(key); val != "" {
		if err := json.Unmarshal([]byte(val), &v); err != nil {
			return v, errors.New(ErrorParse)
		}
		return v, nil
	}
	return v, errors.New(ErrorFlagNotFound)
}
