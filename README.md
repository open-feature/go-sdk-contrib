# OpenFeature Go Contributions

This repository is intended for OpenFeature contributions which are not included in the [OpenFeature SDK](https://github.com/open-feature/go-sdk).

The project includes:

- [Providers](./providers)
- [Hooks](./hooks)

## Environment

Run the following command to setup the go workspace.

```
make workspace-init
```

Additional workspace commands:

```go
make workspace-update               // sync go.work with current modules
make test                           // test all go modules
make lint                           // lint all go modules
make MODULE_NAME=NAME new-provider  // create and setup new provider directory
make MODULE_NAME=NAME new-hook      // create and setup new hook directory
```

## Releases

This repo uses _Release Please_ to release packages. Release Please sets up a running PR that tracks all changes for the library components, and maintains the versions according to [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), generated when [PRs are merged](https://github.com/amannn/action-semantic-pull-request). When Release Please's running PR is merged, any changed artifacts are published.

## License

Apache 2.0 - See [LICENSE](./LICENSE) for more information.