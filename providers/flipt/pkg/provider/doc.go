// This package provides a [Flipt] [OpenFeature Provider] for interacting with the Flipt service backend using the [OpenFeature Go SDK].
//
// From the [OpenFeature Specification]:
// Providers are the "translator" between the flag evaluation calls made in application code, and the flag management system that stores flags and in some cases evaluates flags.
//
// You can configure the provider to connect to Flipt using any of the provided "[Option]"s.
// This configuration allows you to specify the "[ServiceType]" (protocol), and to configure the host, port and other properties to connect to the Flipt service.
//
// [Flipt]: https://github.com/flipt-io/flipt
// [OpenFeature Provider]: https://docs.openfeature.dev/docs/specification/sections/providers
// [OpenFeature Go SDK]: https://github.com/open-feature/go-sdk
// [OpenFeature Specification]: https://docs.openfeature.dev/docs/specification/sections/providers
package flipt
