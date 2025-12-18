# OpenTelemetry Hook

## Usage

## Metrics hook

This hook performs metric collection by tapping into various hook stages. Given below are the metrics are extracted by this hook,

- `feature_flag.evaluation_requests_total`
- `feature_flag.evaluation_success_total`
- `feature_flag.evaluation_error_total`
- `feature_flag.evaluation_active_count`

There are two ways to create hooks:

### Using Global MeterProvider

Global provider should be set somewhere using `otel.SetMeterProvider` before calling this constructor.

```go
// Derive metric hook from reader
metricsHook, err := hooks.NewMetricsHook()
if err != nil {
    return err
}

// Register OpenFeature API level hooks
openfeature.AddHooks(metricsHook)
```

### Passing MeterProvider to Constructor

```go
// provider must be configured and provided to constructor based on application configurations
var provider *metric.MeterProvider

// Derive metric hook from reader
metricsHook, err := hooks.NewMetricsHookForProvider(provider)
if err != nil {
    return err
}

// Register OpenFeature API level hooks
openfeature.AddHooks(metricsHook)
```

### Options

#### WithMetricsAttributeSetter

This constructor options allows to provide a custom callback to extract dimensions from `FlagMetadata`.
These attributes are added at the `After` stage of the hook.

```go

NewMetricsHookForProvider(provider,
    WithMetricsAttributeSetter(
    func(metadata openfeature.FlagMetadata) []attribute.KeyValue {
  // custom attribute extraction logic

        return attributes
    }))
```

#### WithFlagMetadataDimensions

This constructor option allows to configure dimension descriptions to be extracted from `openfeature.FlagMetadata`.
If present, these dimension will be added to the `feature_flag.evaluation_success_total` metric.
Missing metadata keys will be ignored by the implementation.

```go
NewMetricsHook(MeterProvider,
    WithFlagMetadataDimensions(
        DimensionDescription{
            Key:  "scope",
            Type: String,
        }))
```

## Traces hook

The traces hook taps into the after and error methods of the hook lifecycle to write `events` and `attributes`to an existing `span`.
A `context.Context` containing a `span` must be passed to the client evaluation method, otherwise the hook will be no-op.

```go

// Register traces hook
openfeature.AddHooks(hooks.NewTracesHook())
client := openfeature.NewClient("methodA")

// Initialize otel span
spanCtx, span := tracer.Start(context.Background(), "myBoolFlag")
client.BooleanValueDetails(spanCtx, "myBoolFlag", false, openfeature.EvaluationContext{})

...

span.End()
```

### Options

#### WithTracesAttributeSetter

This constructor options allows to provide a custom callback to extract dimensions from `FlagMetadata`.
These attributes are added at the `Finally` stage of the hook.

```go

NewTracesHook(WithTracesAttributeSetter(
    func(metadata openfeature.FlagMetadata) []attribute.KeyValue {
  // custom attribute extraction logic

        return attributes
    }))
```
