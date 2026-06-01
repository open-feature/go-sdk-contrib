# GO Feature Flag Provider

> [!NOTE]  
> 🚀 **GO Feature Flag is a simple, complete and lightweight self-hosted feature flag solution 100% Open Source. 🎛️**  
> 👀 **Check https://gofeatureflag.org to know more about this simple feature flag management system.**

The GO Feature Flag provider connects the OpenFeature Go SDK to a GO Feature Flag relay-proxy.

It supports two evaluation modes:

- `INPROCESS` is the default. The provider fetches flag configuration from the relay-proxy, evaluates flags locally with the GO Feature Flag core library, and keeps the local configuration fresh by polling for updates.
- `REMOTE` delegates every evaluation to the relay-proxy via OFREP. Optionally, the provider can cache evaluation results on the client to reduce network calls. When caching is enabled, the provider polls for configuration changes and purges the cache when a flag change is detected, so stale evaluations are never served.

## Install dependencies

Install the provider and the OpenFeature Go SDK:

```shell
go get github.com/open-feature/go-sdk-contrib/providers/go-feature-flag
```

## Choose an evaluation mode

| Mode | Description | When to use it |
| --- | --- | --- |
| `INPROCESS` | Fetch configuration once, evaluate locally, poll for configuration changes. | Default choice for lower evaluation latency and the new `1.x.x` behavior. |
| `REMOTE` | Every evaluation is performed by the relay-proxy via OFREP, with optional client-side caching. | Use when the relay-proxy is the single source of truth. Enable caching to reduce network calls; the provider polls for ETag changes to keep the cache fresh. |

## Initialize your provider

### In-process mode

`INPROCESS` is the default, so setting `EvaluationType` is optional.

```go
import (
	"context"
	"net/http"
	"time"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
)

ctx := context.Background()

provider, err := gofeatureflag.NewProviderWithContext(ctx, gofeatureflag.ProviderOptions{
	Endpoint:                  "http://localhost:1031",
	FlagChangePollingInterval: 2 * time.Minute,
	HTTPClient: &http.Client{
		Timeout: 5 * time.Second,
	},
})
if err != nil {
	// handle the error
}
```

### Remote mode

Use `REMOTE` when you want the relay-proxy to be the single source of truth for every evaluation.

```go
import (
	"context"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
)

ctx := context.Background()

provider, err := gofeatureflag.NewProviderWithContext(ctx, gofeatureflag.ProviderOptions{
	Endpoint:       "http://localhost:1031",
	EvaluationType: gofeatureflag.EvaluationTypeRemote,
	APIKey:         "my-api-key",
})
if err != nil {
	// handle the error
}
```

To reduce network calls you can enable client-side caching. The provider polls for ETag changes and purges the cache automatically when flags are updated:

```go
import (
	"context"
	"time"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
)

ctx := context.Background()

provider, err := gofeatureflag.NewProviderWithContext(ctx, gofeatureflag.ProviderOptions{
	Endpoint:                  "http://localhost:1031",
	EvaluationType:            gofeatureflag.EvaluationTypeRemote,
	FlagCacheSize:             10000,
	FlagCacheTTL:              5 * time.Minute,
	FlagChangePollingInterval: 2 * time.Minute,
})
if err != nil {
	// handle the error
}
```

## Initialize your OpenFeature client

Register the provider in the OpenFeature SDK and then create a client:

```go
import (
	"context"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
)

ctx := context.Background()

provider, err := gofeatureflag.NewProviderWithContext(ctx, gofeatureflag.ProviderOptions{
	Endpoint: "http://localhost:1031",
})
if err != nil {
	// handle the error
}

if err := of.SetProviderAndWait(provider); err != nil {
	// handle the error
}

client := of.NewClient("my-app")
```

## Evaluate a flag

Create an `EvaluationContext` and use the OpenFeature client as usual.

> In this example we evaluate a boolean flag, but the provider also supports string, integer, float, and object evaluations.
>
> See the [OpenFeature evaluation API documentation](https://openfeature.dev/docs/reference/concepts/evaluation-api#basic-evaluation) for the full API surface.

```go
evaluationCtx := of.NewEvaluationContext(
	"1d1b9238-2591-4a47-94cf-d2bc080892f1",
	map[string]any{
		"firstname": "john",
		"lastname":  "doe",
		"email":     "john.doe@gofeatureflag.org",
		"admin":     true,
		"anonymous": false,
	},
)

adminFlag, err := client.BooleanValue(context.TODO(), "flag-only-for-admin", false, evaluationCtx)
if err != nil {
	// handle the error
}

if adminFlag {
	// flag "flag-only-for-admin" evaluated to true
} else {
	// flag "flag-only-for-admin" evaluated to false
}
```
## Provider options

`Endpoint` is required. The other options are optional.

| Option | Description |
| --- | --- |
| `Endpoint` | Base URL of the GO Feature Flag relay-proxy (e.g. `http://localhost:1031`). Required. |
| `HTTPClient` | Custom HTTP client. If omitted, the provider uses a default client with a 10-second timeout. |
| `APIKey` | API key sent as `X-API-Key`. |
| `Headers` | Extra headers added to provider HTTP requests. Useful for custom auth headers such as `Authorization`. |
| `ExporterMetadata` | Metadata attached to exported evaluation and tracking events. |
| `EvaluationType` | Selects `INPROCESS` or `REMOTE`. Default is `INPROCESS`. |
| `FlagChangePollingInterval` | In `INPROCESS` mode: how often local flag configuration is refreshed. In `REMOTE` mode: how often the provider checks for flag changes to invalidate the evaluation cache. Default 2 minutes. |
| `DataCollectorMaxEventStored` | Maximum number of buffered events before the collector flushes the queue on a subsequent add. |
| `DataCollectorCollectInterval` | Interval used to send buffered events to the relay-proxy data collector. |
| `DataCollectorDisabled` | Disables event collection and tracking export. |
| `DataCollectorBaseURL` | Overrides the base URL used only for the data collector endpoint. |
| `Logger` | Custom `slog.Logger` used by the provider. |
| `DisableCache` | Set to `true` to disable client-side evaluation caching in `REMOTE` mode. Has no effect in `INPROCESS` mode. Default `false`. |
| `FlagCacheSize` | Maximum number of evaluation results held in the client-side cache (`REMOTE` mode only). Default 10 000. |
| `FlagCacheTTL` | How long a cached evaluation result is considered fresh (`REMOTE` mode only). Use `-1` for no expiry. Default 1 minute. |



## Tracking and event collection

### Flag usage events (automatic)

In `INPROCESS` mode, the provider automatically collects a flag evaluation event every time a flag is evaluated. These events are batched in memory and flushed to the relay-proxy data collector endpoint (`POST /v1/data/collector`) periodically or when the buffer is full.

Each event records:
- The flag key and the variation that was served
- The user key and whether the user was anonymous
- The resolved value and whether the SDK default was returned
- A creation timestamp

The flush is also triggered on provider shutdown so no buffered events are lost.

### Custom tracking events (OpenFeature Track API)

The provider implements the OpenFeature `Tracker` interface. Call `client.Track()` to send a named tracking event with arbitrary attributes:

```go
client.Track(
    context.TODO(),
    "user-checkout",
    of.NewEvaluationContext(
        "1d1b9238-2591-4a47-94cf-d2bc080892f1",
        map[string]any{
            "plan": "premium",
        },
    ),
    of.NewTrackingEventDetails(99.99),
)
```

Tracking events are queued in the same data collector buffer as flag evaluation events and sent to the same relay-proxy endpoint.

### Disabling and tuning event collection

Use the `DataCollector*` options to tune or disable collection:

| Option | Default | Description |
| --- | --- | --- |
| `DataCollectorDisabled` | `false` | Set to `true` to disable all event collection. |
| `DataCollectorCollectInterval` | 2 minutes | How often buffered events are flushed to the relay-proxy. |
| `DataCollectorMaxEventStored` | 100 000 | Buffer size. When the buffer is full the provider flushes immediately before queuing the next event. |
| `DataCollectorBaseURL` | same as `Endpoint` | Override the base URL used only for the data collector endpoint. |

## Operational notes

- In `INPROCESS` mode, the provider fetches configuration from `/v1/flag/configuration`, evaluates locally, and emits provider events such as `ProviderConfigChange`, `ProviderStale`, and `ProviderReady`.
- In `REMOTE` mode, evaluations go through OFREP instead of the local evaluator. When caching is enabled, the provider caches each evaluation result keyed on flag name and evaluation context. The cache is purged automatically when a flag configuration change is detected via ETag polling, ensuring stale results are never served after a flag update.
- The local data-collector hook is attached only in `INPROCESS` mode. `Track` still uses the provider data collector when collection is enabled.
- If your relay-proxy expects bearer-token auth instead of `X-API-Key`, set it explicitly with `Headers`, for example `Headers: map[string]string{"Authorization": "Bearer <token>"}`.

## Migrating from v0.x.x to v1.x.x

`INPROCESS` mode is the recommended target for all users. It eliminates a network round-trip per evaluation, reduces load on the relay-proxy, and enables automatic flag usage tracking. The migration below gets you there directly.

1. Upgrade the dependency to the new provider version.
2. Rename or remove old options that no longer exist.
3. Remove any explicit `EvaluationType` setting — `INPROCESS` is the default.
4. Tune `FlagChangePollingInterval` and data-collector settings for your production traffic profile.

If you need a temporary stepping stone while validating the upgrade, you can pin `REMOTE` mode to keep the old evaluation behavior:

```go
options := gofeatureflag.ProviderOptions{
	Endpoint:       "http://localhost:1031",
	EvaluationType: gofeatureflag.EvaluationTypeRemote,
}
```

Option mapping from `0.x.x` to `1.x.x`:

| `0.x.x` option | `1.x.x` status |
| --- | --- |
| `DisableCache` | Still available. Now applies only to `REMOTE` mode. No effect in `INPROCESS` mode. |
| `FlagCacheSize` | Still available. Now applies only to `REMOTE` mode. |
| `FlagCacheTTL` | Still available. Now applies only to `REMOTE` mode. |
| `DataFlushInterval` | Renamed to `DataCollectorCollectInterval`. |
| `DisableDataCollector` | Renamed to `DataCollectorDisabled`. |

Authentication migration note:

- In `0.x.x`, the provider's remote evaluation path used bearer-token auth for `APIKey`.
- In `1.x.x`, `APIKey` is sent as `X-API-Key`.
- If you still need `Authorization: Bearer ...`, use `Headers` instead of `APIKey`.

## Difference between `0.x.x` and `1.x.x`

| Topic | `0.x.x` | `1.x.x` |
| --- | --- | --- |
| Default behavior | Remote evaluation through the relay-proxy. | In-process evaluation is the default. |
| Evaluation model | Each evaluation goes through the remote path, optionally backed by the provider cache. | Configuration is fetched from the relay-proxy and evaluations happen locally. |
| Cache behavior | Provider-managed cache controlled by `DisableCache`, `FlagCacheSize`, and `FlagCacheTTL`. | No cache layer in `INPROCESS` mode — freshness comes from configuration polling. In `REMOTE` mode, an optional client-side evaluation cache is available (`DisableCache`, `FlagCacheSize`, `FlagCacheTTL`), purged automatically on ETag-detected flag changes. |
| Polling purpose | Polling was used to invalidate cached flag data. | Polling refreshes the in-process configuration used for local evaluations. |
| Compatibility mode | Not applicable. | `EvaluationTypeRemote` preserves the old remote pattern while migrating. |
| Auth behavior | `APIKey` on the remote path was bearer-token based. | `APIKey` uses `X-API-Key`; custom auth can be passed with `Headers`. |
| Data collector config | `DataFlushInterval` and `DisableDataCollector`. | `DataCollectorCollectInterval`, `DataCollectorDisabled`, and `DataCollectorBaseURL`. |
