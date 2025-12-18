# Unofficial LaunchDarkly OpenFeature Provider for Go

This provider is maintained by the Open Feature community.

## Installation

```
# LaunchDarkly SDK
go get github.com/launchdarkly/go-sdk-common/v3/...
go get github.com/launchdarkly/go-server-sdk/v7/...

# Open Feature SDK
go get go.openfeature.dev/openfeature/v2
go get go.openfeature.dev/contrib/providers/launchdarkly/v2/pkg
```

## Usage

See [example_test.go](./example_test.go)

## Representing LaunchDarkly (multi) contexts

The LaunchDarkly provider expects contexts to be either single- or
multi-context, matching [LaunchDarkly's concept of Contexts](https://docs.launchdarkly.com/guides/flags/intro-contexts).
The representation of LaunchDarkly context(s) within the OpenFeature
context needs to be well-formed.

### Single context

```javascript
{
  // The "kind" of the context. Required.
  // Cannot be missing, empty, "multi", or "kind".
  // Must match `[a-zA-Z0-9._-]*`
  // (The default LaunchDarkly kind is "user".)
  kind: string,

  // The targeting key. One of the following is required to be
  // present and non-empty. If both are present, `targetingKey`
  // takes precedence.
  key: string,
  targetingKey: string,

  // Private attribute annotations. Optional.
  // See https://docs.launchdarkly.com/sdk/features/private-attributes
  // for the formatting specifications.
  privateAttributes: [string],

  // Anonymous annotation. Optional.
  // See https://docs.launchdarkly.com/sdk/features/anonymous
  anonymous: bool,

  // Name. Optional.
  // If present, used by LaunchDarkly as the display name of the context.
  name: string|null,

  // Further attributes, in the normal OpenFeature format.
  // Attribute names can be any non-empty string except "_meta".
  //
  // Repeated `string: any`
}
```

### Multi context

```javascript
{
  // The "kind" of the context. Required.
  // Must be "multi".
  kind: "multi",

  // Sub-contexts. Each further key is taken to be a "kind" (and
  // thus must match `[a-zA-Z0-9._-]`).
  // The value is should be an object, and is processed using the
  // rules described in [Single context](#single-context) above,
  // except that the "kind" attribute is ignored if present.
  //
  // Repeated `string: object`
}
```

### References

- <https://docs.openfeature.dev/blog/creating-a-provider-for-the-go-sdk/>
