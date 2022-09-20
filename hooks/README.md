# OpenFeature Go Hooks

Hooks are a mechanism whereby application developers can add arbitrary behavior to flag evaluation. They operate similarly to middleware in many web frameworks. Please see the [spec](https://github.com/open-feature/spec/blob/main/specification/flag-evaluation/hooks.md) for more details.

To contibute a new hook, fork this repository and create a new go module, it will then be discoverable by `make workspace-init` and `make workspace-update`.