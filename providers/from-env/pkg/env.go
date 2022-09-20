package from_env

import (
	"encoding/json"
	"errors"
	"os"
)

type envFetch struct{}

func (ef *envFetch) fetchStoredFlag(key string) (StoredFlag, error) {
	v := StoredFlag{}
	if val := os.Getenv(key); val != "" {
		if err := json.Unmarshal([]byte(val), &v); err != nil {
			return v, errors.New(ErrorParse)
		}
		return v, nil
	}
	return v, errors.New(ErrorFlagNotFound)
}
