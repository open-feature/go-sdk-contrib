# OpenFeature Golang Contributions

![Experimental](https://img.shields.io/badge/experimental-breaking%20changes%20allowed-yellow)
![Alpha](https://img.shields.io/badge/alpha-release-red)

This repository is intended for OpenFeature contributions which are not included in the [OpenFeature SDK](https://github.com/open-feature/golang-sdk).

The project includes:

- [Providers](./providers)
- [Hooks](./hooks)

## Environment
Run the following command to setup the go workspace.
```
make workspace-init
```  
Additional workspace commands:
```
make workspace-update           // sync go.work with current modules
make test                       // test all go modules
make lint                       // lint all go modules
make PROVIDER=NAME new-provider // create and setup new provider directory
make HOOK=NAME new-hook         // create and setup new hook directory
```

## License

Apache 2.0 - See [LICENSE](./LICENSE) for more information.
