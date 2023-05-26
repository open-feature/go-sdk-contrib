module github.com/open-feature/go-sdk-contrib/providers/flagd

go 1.18

require (
	buf.build/gen/go/open-feature/flagd/bufbuild/connect-go v1.5.2-20230317150644-afd1cc2ef580.1
	buf.build/gen/go/open-feature/flagd/protocolbuffers/go v1.29.1-20230317150644-afd1cc2ef580.1
	github.com/bufbuild/connect-go v1.7.0
	github.com/bufbuild/connect-opentelemetry-go v0.2.0
	github.com/go-logr/logr v1.2.4
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.9
	github.com/hashicorp/golang-lru/v2 v2.0.2
	github.com/open-feature/flagd/core v0.5.3
	github.com/open-feature/go-sdk v1.4.0
	golang.org/x/net v0.10.0
	google.golang.org/protobuf v1.30.0
)

require (
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	go.opentelemetry.io/otel v1.15.1 // indirect
	go.opentelemetry.io/otel/metric v0.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.15.1 // indirect
)
