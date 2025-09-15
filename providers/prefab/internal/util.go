package internal

import (
	"fmt"
	"strings"

	of "github.com/open-feature/go-sdk/openfeature"
	prefab "github.com/prefab-cloud/prefab-cloud-go/pkg"
)

func ToPrefabContext(evalCtx of.FlattenedContext) (prefab.ContextSet, error) {
	if len(evalCtx) == 0 {
		return prefab.ContextSet{}, nil
	}
	prefabContext := prefab.NewContextSet()
	for k, v := range evalCtx {
		// val, ok := toStr(v)
		parts := strings.SplitN(k, ".", 2)
		if len(parts) < 2 {
			return *prefabContext, fmt.Errorf("context key structure should be in the form of x.y: %s", k)
		}
		key, subkey := parts[0], parts[1]
		if _, exists := prefabContext.Data[key]; !exists {
			prefabContext.WithNamedContextValues(key, map[string]interface{}{
				subkey: v,
			})
		} else {
			prefabContext.Data[key].Data[subkey] = v
		}
	}
	return *prefabContext, nil
}
