module go.openfeature.dev/contrib/providers/flagd/v2

go 1.25.0

require (
	buf.build/gen/go/open-feature/flagd/connectrpc/go v1.18.1-20250529171031-ebdc14163473.1
	buf.build/gen/go/open-feature/flagd/grpc/go v1.5.1-20250529171031-ebdc14163473.2
	buf.build/gen/go/open-feature/flagd/protocolbuffers/go v1.36.6-20250529171031-ebdc14163473.1
	connectrpc.com/connect v1.18.1
	connectrpc.com/otelconnect v0.7.2
	github.com/go-logr/logr v1.4.3
	github.com/google/go-cmp v0.7.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/open-feature/flagd/core v0.12.1
	go.openfeature.dev/openfeature/v2 v2.0.0-20251226144241-81ba50856f77
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.9
)

require (
	github.com/barkimedes/go-deepcopy v0.0.0-20220514131651-17c30cfc62df // indirect
	github.com/diegoholiveira/jsonlogic/v3 v3.8.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/open-feature/flagd-schemas v0.2.9-0.20250707123415-08b4c52d3b86 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/open-feature/go-sdk-contrib/tests/flagd => ../../tests/flagd
