# Code Readiness Hook

The `codereadiness` hook allows controlling feature flag evaluation based on the version of the application code. 
It does this by comparing the current application version with a required minimum version specified in the flag's metadata. 
If the comparison fails (i.e., the application version is lower than the required version), the hook returns an error, causing the flag evaluation to resolve to its configured default value.

## Setup

First, import the OpenFeature SDK and the code readiness hook:

```go
import (
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk-contrib/hooks/codereadiness/pkg"
)
```

Then, configure the hook with the current version of the application code and register it:

```go
// currentVersion is the current version of the code, which can be retrieved 
// from environment variables, build tags, or configuration files.
currentVersion := "v1.0.0"

codeReadinessHook, err := codereadiness.New(currentVersion)
if err != nil {
	// handle error
}

// Register the hook globally at the OpenFeature API level
openfeature.AddHooks(codeReadinessHook)
```

## How It Works

1. The hook runs during the **After** phase of flag evaluation.
2. It extracts the metadata associated with the evaluated flag.
3. It looks for a specific metadata key (by default, `minCodeVersion`).
4. If found, it compares the current application version against the required minimum version using the configured comparator (by default, a semver comparison).
5. If the current version is **lower** than the required version, it returns an error. This triggers the OpenFeature SDK's fallback mechanism, returning the flag's **default value** to the caller.

## Options

The behavior of the hook can be customized by passing options to the constructor:

### Require Validation

By default, the hook will **not** fail if the `minCodeVersion` metadata or the current application version is missing. To enforce version validation and return an error when these versions are missing, use `WithValidationRequired(true)`.

```go
codeReadinessHook, err := codereadiness.New(
	"v1.0.0", 
	codereadiness.WithValidationRequired(true),
)
if err != nil {
	// handle error
}
```

### Custom Metadata Key

To configure the hook to look for a key other than the default `"minCodeVersion"` in the flag's metadata, use `WithMetadataMinVerKey()`.

```go
codeReadinessHook, err := codereadiness.New(
	"v1.0.0", 
	codereadiness.WithMetadataMinVerKey("customMetadataKey"),
)
if err != nil {
	// handle error
}
```

### Custom Comparator

By default, the hook performs a standard semver comparison. If the application uses a different versioning scheme (such as date-based versioning, revision numbers, or custom build numbers), a custom comparison function can be provided using `WithComparator()`.

```go
import (
	"fmt"
)

customComparator := func(current, required string) error {
	// Custom comparison logic: return nil if current is ready/sufficient, 
	// or an error if current is lower than required.
	if current < required {
		return fmt.Errorf("current version %s is not ready for %s", current, required)
	}
	return nil
}

codeReadinessHook, err := codereadiness.New(
	"123", 
	codereadiness.WithComparator(customComparator),
)
if err != nil {
	// handle error
}
```

### Custom Logger

By default, the hook uses `slog.Default()` as its logger. Custom logger can be provided using `WithLogger()` method.

```go
codeReadinessHook, err := codereadiness.New(
	"v1.0.0", 
	codereadiness.WithLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
)
if err != nil {
	// handle error
}
```
