module github.com/open-feature/go-sdk-contrib/providers/flagd

go 1.18

require (
	buf.build/gen/go/open-feature/flagd/bufbuild/connect-go v1.9.0-20230720212818-3675556880a1.1
	buf.build/gen/go/open-feature/flagd/protocolbuffers/go v1.31.0-20230720212818-3675556880a1.1
	github.com/bufbuild/connect-go v1.10.0
	github.com/bufbuild/connect-opentelemetry-go v0.4.0
	github.com/go-logr/logr v1.3.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.6.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/open-feature/flagd/core v0.6.7
	github.com/open-feature/go-sdk v1.8.0
	golang.org/x/net v0.17.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.19.0 // indirect
	go.opentelemetry.io/otel/trace v1.19.0 // indirect
	golang.org/x/exp v0.0.0-20230811145659-89c5cff77bcb // indirect
)
