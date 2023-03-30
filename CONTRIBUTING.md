# Contributing

## System Requirements

go 1.18+  is recommended.

## Setup workspace
 
Run the following command to setup the go workspace.
```
make workspace-init
```

sync go.work with current modules

```
make workspace-update
```

## Compilation target(s)

In Go, the compilation target refers to the operating system and CPU architecture that the Go compiler is targeting when building an executable binary.The operating system can be Windows, Linux, macOS, or other Unix-like systems, and the CPU architecture can be x86, x86-64, ARM, MIPS, or others.

```
GOOS={os} GOARCH={cpu} go build
```

## Adding a module

The project has some specifications in `makefile`  for creating [hooks](https://docs.openfeature.dev/docs/reference/concepts/hooks) and [providers](https://docs.openfeature.dev/docs/reference/concepts/provider).


To contibute a new hook, or provider fork this repository and create a new go module, it will then be discoverable by `make workspace-init` and `make workspace-update`.

create and setup new provider directory (requires jq)
```
make MODULE_NAME=NAME new-provider
```

create and setup new hook directory (requires jq)
```
make MODULE_NAME=NAME new-hook 
```

[jq documentation](https://stedolan.github.io/jq/download/)

## Documentation

Any published modules must have documentation in their root directory, explaining the basic purpose of the module as well as installation and usage instructions.
Instructions for how to develop a module should also be included (required system dependencies, instructions for testing locally, etc).

## Testing

Any published modules must have reasonable test coverage.

Testing packages provide shared testing functionality across OpenFeature components, avoiding duplication.

To test all go modules
```
make test
```

## Releases

This repo uses _Release Please_ to release packages. Release Please sets up a running PR that tracks all changes for the library components, and maintains the versions according to [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), generated when [PRs are merged](https://github.com/amannn/action-semantic-pull-request). When Release Please's running PR is merged, any changed artifacts are published.

## Dependencies

Keep dependencies to a minimum, especially non-dev dependencies.
The GO-SDK should be a _peer dependency_ of your module.
Run `go mod`, and then verify the dependencies in `go.mod` file are appropriate.
Keep in mind, though one version of the GO-SDK is used for all modules in testing, each module may have a different peer-dependency requirement for the GO-SDK (e.g: one module may require ^1.18.0 while another might require ^1.19.0).
Be sure to properly express the GO-SDK peer dependency version your module requires.
