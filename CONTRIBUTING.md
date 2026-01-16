# Contributing

## System Requirements

go 1.19+  is recommended.

## Setup workspace

Run the following command to set up the go workspace:

```shell
make workspace-init
```

To sync `go.work` with the current modules, run:

```shell
make workspace-update
```

## Adding a module

The project provides `makefile` targets for creating [hooks](https://openfeature.dev/docs/reference/concepts/hooks) and [providers](https://openfeature.dev/docs/reference/concepts/provider).
To contribute a new hook or provider, fork this repository and create a new go module, it will then be discoverable by `make workspace-init` and `make workspace-update`.

To automatically create and set up a new provider directory, use the following command (requires [jq](https://jqlang.github.io/jq/)):

```shell
make MODULE_NAME=NAME new-provider
```

To automatically create and set up a new hook directory, use the following command (requires [jq](https://jqlang.github.io/jq/)):

```shell
make MODULE_NAME=NAME new-hook 
```

Note - [jq documentation](https://stedolan.github.io/jq/download/)

### Versioning

The release version of the newly added module(hook/provider) is controlled by `.release-please-manifest.json`.
You can control the versioning of your module by adding an entry with desired initial version(ex:`"provider/acme":"0.0.1"`).
Otherwise, default versioning will start from `1.0.0`.

## Documentation

Any published modules must have documentation in their root directory, explaining the basic purpose of the module as well as installation and usage instructions.
Instructions for how to develop a module should also be included (required system dependencies, instructions for testing locally, etc).

## Testing

Any published modules must have reasonable test coverage.

To run tests in all existing go modules, use the command:

```shell
make test
```

It is recommended to include end-to-end (e2e) tests in your provider when possible.
If you have dependency services for your e2e tests, make sure to add them as service in the build pipeline.

You can run all tests, including e2e tests using the command:

```shell
make e2e
```

## Developer Certificate of Origin
Developer Certificate of Origin
The Developer Certificate of Origin (DCO) is a lightweight way for contributors to certify that they wrote or otherwise have the right to submit the code they are contributing to the project. To sign off that they adhere to these requirements, all commits need to have a Signed-off-by line, like:

```
fix: solve all the problems

Signed-off-by: John Doe <jd@example.org>
```

This is easy to add by using the -s/--signoff flag to git commit.

More info is available in the OpenFeature [community docs](https://openfeature.dev/community/technical-guidelines/#developer-certificate-of-origin).

## Releases

This repo uses _Release Please_ to release packages. Release Please sets up a running PR that tracks all changes for the library components, and maintains the versions according to [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), generated when [PRs are merged](https://github.com/amannn/action-semantic-pull-request).
Merging the Release Please PR will create a GitHub release with updated library versions.

## Dependencies

The [GO-SDK](https://github.com/open-feature/go-sdk) should be a _peer dependency_ of your module.
