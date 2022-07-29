# OpenFeature Golang Providers

Providers are responsible for performing flag evaluation. They provide an abstraction between the underlying flag management system and OpenFeature itself. This allows providers to be changed without requiring a major code refactor. Please see the [spec](https://github.com/open-feature/spec/blob/main/specification/provider/providers.md) for more details.

To contibute a new provider, fork this repository and create a new module, it will then be discoverable by `make workspace-init` and `make workspace-update`.