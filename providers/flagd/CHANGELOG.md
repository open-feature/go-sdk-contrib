# Changelog

## [0.1.9](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.8...providers/flagd/v0.1.9) (2023-02-24)


### Bug Fixes

* **deps:** update module github.com/open-feature/flagd to v0.3.7 ([#106](https://github.com/open-feature/go-sdk-contrib/issues/106)) ([497ed34](https://github.com/open-feature/go-sdk-contrib/commit/497ed34add9d3f77fdcd3eb48e175aa39cc4388f))

## [0.1.8](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.7...providers/flagd/v0.1.8) (2023-02-21)


### Features

* aligned environment variables application with flagd provider spec ([#119](https://github.com/open-feature/go-sdk-contrib/issues/119)) ([5ee1f2c](https://github.com/open-feature/go-sdk-contrib/commit/5ee1f2c8af0d41eb3820d32ca7ffe30777a2d12a))


### Bug Fixes

* **deps:** update module github.com/bufbuild/connect-go to v1.5.2 ([#118](https://github.com/open-feature/go-sdk-contrib/issues/118)) ([0207626](https://github.com/open-feature/go-sdk-contrib/commit/0207626f688d61a6d26dbfd3086e25277241401b))
* **deps:** update module github.com/open-feature/go-sdk to v1.2.0 ([#103](https://github.com/open-feature/go-sdk-contrib/issues/103)) ([eedb577](https://github.com/open-feature/go-sdk-contrib/commit/eedb577745fd98d5189132ebbaa8eb82bdf99dd8))

## [0.1.7](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.6...providers/flagd/v0.1.7) (2023-02-13)


### Features

* exposed WithTLS provider option. Allow tls to be used without cert path (default to host system's CAs) ([#112](https://github.com/open-feature/go-sdk-contrib/issues/112)) ([c5bae5f](https://github.com/open-feature/go-sdk-contrib/commit/c5bae5f32b473796bdc2b7c8614439be53a37739))

## [0.1.6](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.5...providers/flagd/v0.1.6) (2023-02-03)


### Bug Fixes

* **deps:** update module github.com/bufbuild/connect-go to v1.5.1 ([#99](https://github.com/open-feature/go-sdk-contrib/issues/99)) ([0f7c8a4](https://github.com/open-feature/go-sdk-contrib/commit/0f7c8a435b4acfc75317a186c871b020c1432aed))

## [0.1.5](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.4...providers/flagd/v0.1.5) (2023-01-31)


### Bug Fixes

* **deps:** update module github.com/open-feature/flagd to v0.3.4 ([#83](https://github.com/open-feature/go-sdk-contrib/issues/83)) ([958c9fa](https://github.com/open-feature/go-sdk-contrib/commit/958c9fa81637cbacf59259d100d74407f41cd87c))
* **deps:** update module golang.org/x/net to v0.5.0 ([#56](https://github.com/open-feature/go-sdk-contrib/issues/56)) ([168d6cf](https://github.com/open-feature/go-sdk-contrib/commit/168d6cf9b7047ba412c239f2349d2e3d4b02a21d))

## [0.1.4](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.3...providers/flagd/v0.1.4) (2023-01-26)


### Bug Fixes

* tidy workspaces ([#97](https://github.com/open-feature/go-sdk-contrib/issues/97)) ([c71a5ec](https://github.com/open-feature/go-sdk-contrib/commit/c71a5ec7686ec0572bb47f17dbca7e0ec48252d7))

## [0.1.3](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.2...providers/flagd/v0.1.3) (2023-01-25)


### Features

* expose ProviderOption to set logr logger & implemented structured logging with levels ([#93](https://github.com/open-feature/go-sdk-contrib/issues/93)) ([ac5e8dd](https://github.com/open-feature/go-sdk-contrib/commit/ac5e8dd274c9fd811dccaca85d3aba33994b480b))

## [0.1.2](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.1...providers/flagd/v0.1.2) (2023-01-07)


### Bug Fixes

* Purge cache if config change event handling fails ([#85](https://github.com/open-feature/go-sdk-contrib/issues/85)) ([bf47049](https://github.com/open-feature/go-sdk-contrib/commit/bf4704959411f3957a8c9266f0756b768c915ce1))

## [0.1.1](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd-v0.1.0...providers/flagd/v0.1.1) (2023-01-06)


### Features

* handle consolidated configuration change event ([#66](https://github.com/open-feature/go-sdk-contrib/issues/66)) ([69cb619](https://github.com/open-feature/go-sdk-contrib/commit/69cb619b6cf0a3095ae0bb2f6544e22fb3d5786e))
* create blocking mechanism until provider ready ([#78](https://github.com/open-feature/go-sdk-contrib/issues/68)) ([9937b5e](https://github.com/open-feature/go-sdk-contrib/commit/9937b5ed934155b987520c90754827d5376a4b04))
