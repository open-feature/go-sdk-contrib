module github.com/open-feature/go-sdk-contrib/providers/configcat

go 1.13

replace go.buf.build/open-feature/flagd-connect/open-feature/flagd v1.1.3 => buf.build/gen/go/open-feature/flagd/bufbuild/connect-go v1.10.0-20230720212818-3675556880a1.1

replace go.buf.build/open-feature/flagd-connect/open-feature/flagd v1.1.4 => buf.build/gen/go/open-feature/flagd/bufbuild/connect-go v1.10.0-20230720212818-3675556880a1.1

require (
	github.com/configcat/go-sdk/v7 v7.10.1
	github.com/open-feature/go-sdk v1.5.1
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.9.0 // indirect
)
