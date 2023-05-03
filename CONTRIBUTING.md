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

## Adding a module

The project provides `makefile` targets for creating [hooks](https://openfeature.dev/docs/reference/concepts/hooks) and [providers](https://openfeature.dev/docs/reference/concepts/provider).


To contibute a new hook, or provider fork this repository and create a new go module, it will then be discoverable by `make workspace-init` and `make workspace-update`.

create and setup new provider directory (requires jq)
```
make MODULE_NAME=NAME new-provider
```

create and setup new hook directory (requires jq)
```
make MODULE_NAME=NAME new-hook 
```

Note - [jq documentation](https://stedolan.github.io/jq/download/)

### Versioning

The release version of the newly added module(hook/provider) is controlled by `.release-please-manifest.json`. You can 
control the versioning of your module by adding an entry  with desired initial version(ex: `"provider/acme":"0.0.1"` ). 
Otherwise, default versioning will start from `1.0.0`.

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

This repo uses _Release Please_ to release packages. Release Please sets up a running PR that tracks all changes for the library components, and maintains the versions according to [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), generated when [PRs are merged](https://github.com/amannn/action-semantic-pull-request).
Merging the Release Please PR will create a GitHub release with updated library versions.

## Dependencies

The [GO-SDK](https://github.com/open-feature/go-sdk) should be a _peer dependency_ of your module.

