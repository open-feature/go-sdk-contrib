# Changelog

## [0.1.2](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd-in-process/v0.1.1...providers/flagd-in-process/v0.1.2) (2023-12-05)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/flagd to v0.4.2 ([#364](https://github.com/open-feature/go-sdk-contrib/issues/364)) ([1218631](https://github.com/open-feature/go-sdk-contrib/commit/1218631af53b82eba43768da085e0b88ec2ef296))
* **deps:** update module github.com/open-feature/flagd/core to v0.6.7 ([#350](https://github.com/open-feature/go-sdk-contrib/issues/350)) ([8bccc11](https://github.com/open-feature/go-sdk-contrib/commit/8bccc119f0f454adb3aa35e5c74bff43c50a80d3))
* **deps:** update module github.com/open-feature/go-sdk to v1.8.0 ([#329](https://github.com/open-feature/go-sdk-contrib/issues/329)) ([c99b527](https://github.com/open-feature/go-sdk-contrib/commit/c99b52728bad9dce52bfb78a08ae5f4eea83a397))
* **deps:** update module google.golang.org/grpc to v1.58.3 [security] ([#356](https://github.com/open-feature/go-sdk-contrib/issues/356)) ([d4a5098](https://github.com/open-feature/go-sdk-contrib/commit/d4a5098e84fdeb5aa9936ac496e75404f234e247))
* fix flagd dependencies ([#380](https://github.com/open-feature/go-sdk-contrib/issues/380)) ([b7baa69](https://github.com/open-feature/go-sdk-contrib/commit/b7baa6990e05f46637917d83b07dbe0f741d0036))


### ✨ New Features

* deprecate dedicated flagd in-process provider ([#396](https://github.com/open-feature/go-sdk-contrib/issues/396)) ([62ac471](https://github.com/open-feature/go-sdk-contrib/commit/62ac4711a37195579de9c04e628d3785ca06f32f))


### 🧹 Chore

* fixed the spec URL on the readme ([0a483a8](https://github.com/open-feature/go-sdk-contrib/commit/0a483a83053c468eb9ae93287fb661da0abea8cb))


### 🔄 Refactoring

* migrate to connectrpc/connect-go ([#372](https://github.com/open-feature/go-sdk-contrib/issues/372)) ([aba4f4e](https://github.com/open-feature/go-sdk-contrib/commit/aba4f4e3cba0b9af23e665f0e736ec715d3f2fdc))

## [0.1.1](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd-in-process-v0.1.0...providers/flagd-in-process/v0.1.1) (2023-09-20)


### 🐛 Bug Fixes

* error handing, add e2e tests ([#334](https://github.com/open-feature/go-sdk-contrib/issues/334)) ([dfc3b5a](https://github.com/open-feature/go-sdk-contrib/commit/dfc3b5a73e6708aa852a2f2651468de96a754694))
* fractional operator name ([#336](https://github.com/open-feature/go-sdk-contrib/issues/336)) ([d419274](https://github.com/open-feature/go-sdk-contrib/commit/d4192741db354568c05f0fc1306846b2553caff4))


### ✨ New Features

* implement in-process-flagd provider ([#327](https://github.com/open-feature/go-sdk-contrib/issues/327)) ([f9a68b1](https://github.com/open-feature/go-sdk-contrib/commit/f9a68b10d42149b87f87fad03d6829eb77443735))
