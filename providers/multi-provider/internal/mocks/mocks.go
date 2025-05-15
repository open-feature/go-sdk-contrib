//go:generate go run go.uber.org/mock/mockgen -destination=../../internal/mocks/openfeature_mocks.go -package=mocks  "github.com/open-feature/go-sdk/openfeature" FeatureProvider,Hook,StateHandler,EventHandler
package mocks
