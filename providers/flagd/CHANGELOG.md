# Changelog

## [0.3.2](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.3.1...providers/flagd/v0.3.2) (2026-01-16)


### 🐛 Bug Fixes

* **flagd:** configurable retry backoff after each sync cycle error ([#756](https://github.com/open-feature/go-sdk-contrib/issues/756)) ([#806](https://github.com/open-feature/go-sdk-contrib/issues/806)) ([0791fa1](https://github.com/open-feature/go-sdk-contrib/commit/0791fa182c376bfc7c5109403e5f99799a64c87e))
* **flagd:** do not retry for certain status codes ([#756](https://github.com/open-feature/go-sdk-contrib/issues/756)) ([#799](https://github.com/open-feature/go-sdk-contrib/issues/799)) ([e01a99e](https://github.com/open-feature/go-sdk-contrib/commit/e01a99ed8a0b54a2c09ed9c2aa4f5cd658769e78))
* **flagd:** update flagd/core and remove vulnerable dependencies [security] ([#815](https://github.com/open-feature/go-sdk-contrib/issues/815)) ([f4adcf8](https://github.com/open-feature/go-sdk-contrib/commit/f4adcf8ac237f57a41c39765fade17cf512b644e))
* implement deadline to fix indefinite init ([#823](https://github.com/open-feature/go-sdk-contrib/issues/823)) ([24c1bba](https://github.com/open-feature/go-sdk-contrib/commit/24c1bbad9476b4be632e5dae263b40abc9a1b80d))
* tests/flagd version to get module correctly ([#813](https://github.com/open-feature/go-sdk-contrib/issues/813)) ([a32c55f](https://github.com/open-feature/go-sdk-contrib/commit/a32c55f9ced9b6b71517c1e53b5654942e41c8b5))


### ✨ New Features

* **flagd:** Add flagd-selector gRPC metadata header to in-process service. ([#790](https://github.com/open-feature/go-sdk-contrib/issues/790)) ([a891ba8](https://github.com/open-feature/go-sdk-contrib/commit/a891ba8d89b9b4233a7cb222668ead16b1036807))
* **flagd:** Configure in-process provider using FLAGD_SYNC_PORT environment variable ([#804](https://github.com/open-feature/go-sdk-contrib/issues/804)) ([6cf3902](https://github.com/open-feature/go-sdk-contrib/commit/6cf3902e31104b0cadd50368fca537b03f390c9c))

## [0.3.1](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.3.0...providers/flagd/v0.3.1) (2025-11-25)


### 🐛 Bug Fixes

* **deps:** bump open-feature/go-sdk from v1.11 to v1.15 ([#686](https://github.com/open-feature/go-sdk-contrib/issues/686)) ([ce87102](https://github.com/open-feature/go-sdk-contrib/commit/ce871021d0c45d3c992bb00b33c8b7a8e337e9a3))
* **deps:** update golang.org/x/exp digest to b7579e2 ([#679](https://github.com/open-feature/go-sdk-contrib/issues/679)) ([a6372f9](https://github.com/open-feature/go-sdk-contrib/commit/a6372f91b262d2f81b90bfa9e76d722ad480378b))
* **deps:** update jsonlogic module to fix race detection ([#691](https://github.com/open-feature/go-sdk-contrib/issues/691)) ([21f3de0](https://github.com/open-feature/go-sdk-contrib/commit/21f3de0d39a6d23000957bd6f278df466af385e4))
* **deps:** update module buf.build/gen/go/open-feature/flagd/connectrpc/go to v1.18.1-20250529171031-ebdc14163473.1 ([#699](https://github.com/open-feature/go-sdk-contrib/issues/699)) ([6c7044d](https://github.com/open-feature/go-sdk-contrib/commit/6c7044de8bf10d12ed07f4c66335e297c444a6fe))
* **deps:** update module buf.build/gen/go/open-feature/flagd/grpc/go to v1.5.1-20250529171031-ebdc14163473.2 ([#700](https://github.com/open-feature/go-sdk-contrib/issues/700)) ([4747395](https://github.com/open-feature/go-sdk-contrib/commit/474739580f3c7f72e031929b99ab0b86ba4812bb))
* **deps:** update module buf.build/gen/go/open-feature/flagd/protocolbuffers/go to v1.36.6-20250529171031-ebdc14163473.1 ([#706](https://github.com/open-feature/go-sdk-contrib/issues/706)) ([902021b](https://github.com/open-feature/go-sdk-contrib/commit/902021be1083336d9a53c1fd8388cbaaa8dc7959))
* **deps:** update module github.com/open-feature/flagd/core to v0.11.5 ([#666](https://github.com/open-feature/go-sdk-contrib/issues/666)) ([94b44c4](https://github.com/open-feature/go-sdk-contrib/commit/94b44c4aed982ac54b91bd82a2cf8400c1b622c0))
* **deps:** update module github.com/open-feature/go-sdk to v1.15.1 ([#681](https://github.com/open-feature/go-sdk-contrib/issues/681)) ([8fd544f](https://github.com/open-feature/go-sdk-contrib/commit/8fd544ff81fd25eed655a214aa1ae1906a436f0d))
* fix goroutine leaks around shutdown ([#716](https://github.com/open-feature/go-sdk-contrib/issues/716)) ([c3ea532](https://github.com/open-feature/go-sdk-contrib/commit/c3ea53271ed91d20c9a9afd762ea5e2c4c3c488a))
* **flagd:** missed error events, add e2e tests ([#760](https://github.com/open-feature/go-sdk-contrib/issues/760)) ([3750972](https://github.com/open-feature/go-sdk-contrib/commit/3750972d25d847ea56f6b9b5a7640407db67ab11))
* **security:** update module github.com/containerd/containerd/v2 to v2.1.5 [security] ([#797](https://github.com/open-feature/go-sdk-contrib/issues/797)) ([f74c0c3](https://github.com/open-feature/go-sdk-contrib/commit/f74c0c306759914c48364320f2f3a2db252f3d35))
* **security:** update module github.com/docker/compose/v2 to v2.40.2 [security] ([#785](https://github.com/open-feature/go-sdk-contrib/issues/785)) ([805823f](https://github.com/open-feature/go-sdk-contrib/commit/805823f5ded2d81359fd7663804beb50f30d52f7))
* **security:** update module golang.org/x/crypto to v0.45.0 [security] ([#803](https://github.com/open-feature/go-sdk-contrib/issues/803)) ([20b0ccd](https://github.com/open-feature/go-sdk-contrib/commit/20b0ccdf1261cacde5273f61882194b92dbd6650))
* **security:** update vulnerability-updates [security] ([#724](https://github.com/open-feature/go-sdk-contrib/issues/724)) ([629a535](https://github.com/open-feature/go-sdk-contrib/commit/629a5351c2c4b8fed00522f7453d5545920ceaaf))
* **security:** update vulnerability-updates [security] ([#773](https://github.com/open-feature/go-sdk-contrib/issues/773)) ([21628dc](https://github.com/open-feature/go-sdk-contrib/commit/21628dc0bc058c042f14c1afa45df2dfc3d93c72))


### ✨ New Features

* comprehensive flagd e2e testing framework with testcontainers integration ([#732](https://github.com/open-feature/go-sdk-contrib/issues/732)) ([e3ec17b](https://github.com/open-feature/go-sdk-contrib/commit/e3ec17bdc7140582582a5df1154b6044cbf5b640))
* **flagd:** add eventing with graceperiod for inprocess resolver ([#744](https://github.com/open-feature/go-sdk-contrib/issues/744)) ([a9fabb6](https://github.com/open-feature/go-sdk-contrib/commit/a9fabb623d22b6a1ef888722ffe68686031309b8))
* upgrade flagd dependencies to 0.12.1 ([#731](https://github.com/open-feature/go-sdk-contrib/issues/731)) ([8e8d888](https://github.com/open-feature/go-sdk-contrib/commit/8e8d888dea080a03ea2a709b79598c7de6a9eed8))

## [0.3.0](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.6...providers/flagd/v0.3.0) (2025-06-07)


### ⚠ BREAKING CHANGES

* **flagd:** add file mode to flagd provider ([#648](https://github.com/open-feature/go-sdk-contrib/issues/648))

### 🐛 Bug Fixes

* **deps:** Remove dependency on sigs.k8s.io/controller-runtime/pkg/lo… ([#639](https://github.com/open-feature/go-sdk-contrib/issues/639)) ([c2e1a73](https://github.com/open-feature/go-sdk-contrib/commit/c2e1a73e5d4297625b048b1a589101150b7c4136))
* **deps:** update golang.org/x/exp digest to 7e4ce0a ([#515](https://github.com/open-feature/go-sdk-contrib/issues/515)) ([4a04445](https://github.com/open-feature/go-sdk-contrib/commit/4a04445ee4e327bc0cfe497f0d7bab64697b8b61))
* **deps:** update module buf.build/gen/go/open-feature/flagd/connectrpc/go to v1.18.1-20250127221518-be6d1143b690.1 ([#605](https://github.com/open-feature/go-sdk-contrib/issues/605)) ([6ff206c](https://github.com/open-feature/go-sdk-contrib/commit/6ff206c11517e168d864e24fc1ad28f672599a1f))
* **deps:** update module buf.build/gen/go/open-feature/flagd/protocolbuffers/go to v1.36.6-20250127221518-be6d1143b690.1 ([#663](https://github.com/open-feature/go-sdk-contrib/issues/663)) ([665759d](https://github.com/open-feature/go-sdk-contrib/commit/665759d7ef1bc00aa5db9b658c7cfc135f3c7ea0))
* **deps:** update module connectrpc.com/otelconnect to v0.7.2 ([#664](https://github.com/open-feature/go-sdk-contrib/issues/664)) ([7762020](https://github.com/open-feature/go-sdk-contrib/commit/7762020a6f74dcc4d2e1dadc37b51d57ae589e01))
* **deps:** update module github.com/google/go-cmp to v0.7.0 ([#675](https://github.com/open-feature/go-sdk-contrib/issues/675)) ([b435fb4](https://github.com/open-feature/go-sdk-contrib/commit/b435fb47b3c31d5518079e4f8ed2edf4c8ff23ff))
* **deps:** update module go.uber.org/mock to v0.5.2 ([#667](https://github.com/open-feature/go-sdk-contrib/issues/667)) ([b609a70](https://github.com/open-feature/go-sdk-contrib/commit/b609a7089307f92cb4e43477f3f98736f7a6d2d2))
* **deps:** update module golang.org/x/net to v0.38.0 [security] ([#649](https://github.com/open-feature/go-sdk-contrib/issues/649)) ([0ccc7e3](https://github.com/open-feature/go-sdk-contrib/commit/0ccc7e36044e90e7972505651ac8dfbc6680c49c))


### ✨ New Features

* **flagd:** add file mode to flagd provider ([#648](https://github.com/open-feature/go-sdk-contrib/issues/648)) ([3ac923c](https://github.com/open-feature/go-sdk-contrib/commit/3ac923c17efb04959297fe8ba9fe1eb923bbbfc1))
* **flagd:** add Shutdown support for in-process sync ([#687](https://github.com/open-feature/go-sdk-contrib/issues/687)) ([ecf95d6](https://github.com/open-feature/go-sdk-contrib/commit/ecf95d60df95e679d6876795f4c9bffa799857fc))


### 🧹 Chore

* **deps:** update dependency go to v1.24.1 ([#564](https://github.com/open-feature/go-sdk-contrib/issues/564)) ([2a99abc](https://github.com/open-feature/go-sdk-contrib/commit/2a99abc0a4afbb54e8acc2149daaeecbecc3b694))
* **deps:** update dependency go to v1.24.3 ([#661](https://github.com/open-feature/go-sdk-contrib/issues/661)) ([bc218c1](https://github.com/open-feature/go-sdk-contrib/commit/bc218c12116e77af17f19fc773ee458df0d4d4b0))

## [0.2.6](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.5...providers/flagd/v0.2.6) (2025-02-22)


### 🐛 Bug Fixes

* **flagd:** Fixed possible nil pointer exception with svcMetadata in service.go ([#634](https://github.com/open-feature/go-sdk-contrib/issues/634)) ([50256e9](https://github.com/open-feature/go-sdk-contrib/commit/50256e9af89201ca09f3989161afd5a069d3a06e))


### ✨ New Features

* **flagd:** Added WithGrpcDialOptionsOverride provider option ([#638](https://github.com/open-feature/go-sdk-contrib/issues/638)) ([fe904bb](https://github.com/open-feature/go-sdk-contrib/commit/fe904bb054be86ca8e1cafa8577e8ac152dfefc8))


### 🧹 Chore

* **flagd:** Updates flagd core to v0.11.2 ([#636](https://github.com/open-feature/go-sdk-contrib/issues/636)) ([99d1a0c](https://github.com/open-feature/go-sdk-contrib/commit/99d1a0c9d206102774c8a83b2f40e2a33b29309f))

## [0.2.5](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.4...providers/flagd/v0.2.5) (2025-02-18)


### 🐛 Bug Fixes

* **deps:** update module github.com/cucumber/godog to v0.15.0 ([#632](https://github.com/open-feature/go-sdk-contrib/issues/632)) ([210e397](https://github.com/open-feature/go-sdk-contrib/commit/210e3970e676d5144e50e1dc602b55d198bbddfa))


### ✨ New Features

* **flagd:** Support supplying providerID for in-process resolver as an option ([#626](https://github.com/open-feature/go-sdk-contrib/issues/626)) ([2d35083](https://github.com/open-feature/go-sdk-contrib/commit/2d35083b2491692816f42653d86bbebd31648d1c))


### 🧹 Chore

* **flagd:** Updates flagd core to v.0.11.1, taking care of breaking changes ([#627](https://github.com/open-feature/go-sdk-contrib/issues/627)) ([e3d5fc7](https://github.com/open-feature/go-sdk-contrib/commit/e3d5fc79491f21f6d99f500b63dfe22a119a7657))

## [0.2.4](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.3...providers/flagd/v0.2.4) (2025-02-07)


### 🐛 Bug Fixes

* **deps:** update module golang.org/x/net to v0.33.0 [security] ([#601](https://github.com/open-feature/go-sdk-contrib/issues/601)) ([947cd3f](https://github.com/open-feature/go-sdk-contrib/commit/947cd3f07c32ea91fd089340db55fd407a7072a5))


### ✨ New Features

* Support supplying a custom sync provider for in-process flagd ([#598](https://github.com/open-feature/go-sdk-contrib/issues/598)) ([bfa642a](https://github.com/open-feature/go-sdk-contrib/commit/bfa642ad3e0726c2e01ad623312d582b0511e100))

## [0.2.3](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.2...providers/flagd/v0.2.3) (2024-10-29)


### 🐛 Bug Fixes

* **deps:** update module buf.build/gen/go/open-feature/flagd/connectrpc/go to v1.16.2-20240215170432-1e611e2999cc.1 ([#518](https://github.com/open-feature/go-sdk-contrib/issues/518)) ([44965d6](https://github.com/open-feature/go-sdk-contrib/commit/44965d69e5a2ff3621ba9de51d140ad0ea94bdfc))
* **deps:** update module buf.build/gen/go/open-feature/flagd/connectrpc/go to v1.17.0-20240906125204-0a6a901b42e8.1 ([#581](https://github.com/open-feature/go-sdk-contrib/issues/581)) ([51238a0](https://github.com/open-feature/go-sdk-contrib/commit/51238a0b37344077529ad1e0223d6eeca3c2752c))
* **deps:** update module buf.build/gen/go/open-feature/flagd/grpc/go to v1.5.1-20240215170432-1e611e2999cc.1 ([#512](https://github.com/open-feature/go-sdk-contrib/issues/512)) ([1c765e5](https://github.com/open-feature/go-sdk-contrib/commit/1c765e5db0f9129f2be44ffdf7ac212283bb0f6c))
* **deps:** update module connectrpc.com/otelconnect to v0.7.1 ([#558](https://github.com/open-feature/go-sdk-contrib/issues/558)) ([423790c](https://github.com/open-feature/go-sdk-contrib/commit/423790c1b45e32e786f5977f67701ae98d7d1c45))
* **deps:** update module github.com/cucumber/godog to v0.14.1 ([#513](https://github.com/open-feature/go-sdk-contrib/issues/513)) ([f15f019](https://github.com/open-feature/go-sdk-contrib/commit/f15f01969ea0537f66592a77870a68b0de5fd7cc))


### ✨ New Features

* added custom grpc resolver ([#587](https://github.com/open-feature/go-sdk-contrib/issues/587)) ([e509afa](https://github.com/open-feature/go-sdk-contrib/commit/e509afa1d0db8f8321d5d4518959c53a1418db8f))


### 🧹 Chore

* add license to module ([#554](https://github.com/open-feature/go-sdk-contrib/issues/554)) ([abb7657](https://github.com/open-feature/go-sdk-contrib/commit/abb76571c373582f36837587400104eb754c01b9))

## [0.2.2](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.1...providers/flagd/v0.2.2) (2024-06-14)


### ✨ New Features

* Default port to 8015 if in-process resolver is used. [#523](https://github.com/open-feature/go-sdk-contrib/issues/523) ([#524](https://github.com/open-feature/go-sdk-contrib/issues/524)) ([7db1ce3](https://github.com/open-feature/go-sdk-contrib/commit/7db1ce3c579c6e5072059c3159fa12f26e79393e))


### 🧹 Chore

* **deps:** update dependency go to v1.22.3 ([#500](https://github.com/open-feature/go-sdk-contrib/issues/500)) ([54e6bd8](https://github.com/open-feature/go-sdk-contrib/commit/54e6bd897b38d4491037f832345f30cf38e03bd5))

## [0.2.1](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.2.0...providers/flagd/v0.2.1) (2024-05-08)


### ✨ New Features

* update flagd core along with removal of deprecated methods ([#509](https://github.com/open-feature/go-sdk-contrib/issues/509)) ([ae16e3b](https://github.com/open-feature/go-sdk-contrib/commit/ae16e3b18ff5d839c883ae0f0ebe06d68c013290))

## [0.2.0](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.22...providers/flagd/v0.2.0) (2024-04-11)


### ⚠ BREAKING CHANGES

* use new eval/sync protos (requires flagd v0.7.3+) ([#451](https://github.com/open-feature/go-sdk-contrib/issues/451))

### 🐛 Bug Fixes

* **deps:** update golang.org/x/exp digest to 814bf88 ([#453](https://github.com/open-feature/go-sdk-contrib/issues/453)) ([80cdaaa](https://github.com/open-feature/go-sdk-contrib/commit/80cdaaa9fa5ffa25eee3dd0de98588ad66aa5f2e))
* **deps:** update golang.org/x/exp digest to a685a6e ([#479](https://github.com/open-feature/go-sdk-contrib/issues/479)) ([e55c610](https://github.com/open-feature/go-sdk-contrib/commit/e55c610e419589d9bfc3a90089391cbe615d71c7))
* **deps:** update module buf.build/gen/go/open-feature/flagd/protocolbuffers/go to v1.33.0-20240215170432-1e611e2999cc.1 ([#468](https://github.com/open-feature/go-sdk-contrib/issues/468)) ([aebf9d0](https://github.com/open-feature/go-sdk-contrib/commit/aebf9d0a7cc514f66dc26d50104bc4656408cb44))
* **deps:** update module github.com/open-feature/flagd/core to v0.8.1 ([#483](https://github.com/open-feature/go-sdk-contrib/issues/483)) ([4c3f005](https://github.com/open-feature/go-sdk-contrib/commit/4c3f005f587902b239ea904c8d050d054dc8afe7))
* **deps:** update module github.com/open-feature/go-sdk-contrib/tests/flagd to v1.4.1 ([#484](https://github.com/open-feature/go-sdk-contrib/issues/484)) ([6f4e7b7](https://github.com/open-feature/go-sdk-contrib/commit/6f4e7b746e8854b999ec6ece6a8259a5c9e77fdc))
* **deps:** update module go.uber.org/mock to v0.4.0 ([#425](https://github.com/open-feature/go-sdk-contrib/issues/425)) ([91f70c0](https://github.com/open-feature/go-sdk-contrib/commit/91f70c0dba1e1ff8d7214b05de8b86eead43a922))
* **deps:** update module google.golang.org/grpc to v1.62.1 ([#430](https://github.com/open-feature/go-sdk-contrib/issues/430)) ([c20613c](https://github.com/open-feature/go-sdk-contrib/commit/c20613c5079f2a9871c451771aca2b8ab56d7bcb))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.17.2 ([#434](https://github.com/open-feature/go-sdk-contrib/issues/434)) ([acaf0cb](https://github.com/open-feature/go-sdk-contrib/commit/acaf0cb8c479ff6978be4753cc192fdaac077ecb))


### ✨ New Features

* update to latest flagd core release ([#495](https://github.com/open-feature/go-sdk-contrib/issues/495)) ([4034850](https://github.com/open-feature/go-sdk-contrib/commit/40348500c7adc433bd2387f43bebcad83ae65153))
* use new eval/sync protos (requires flagd v0.7.3+) ([#451](https://github.com/open-feature/go-sdk-contrib/issues/451)) ([308bba1](https://github.com/open-feature/go-sdk-contrib/commit/308bba1656dfe05993b83bc9f2059082b41e79f0))


### 🧹 Chore

* improve contrib guide with e2e test details ([#447](https://github.com/open-feature/go-sdk-contrib/issues/447)) ([8dd5fc6](https://github.com/open-feature/go-sdk-contrib/commit/8dd5fc6a317665918b3432d6e4d7a4ba0598f554))
* move flagd specific submodule to flagd module ([#449](https://github.com/open-feature/go-sdk-contrib/issues/449)) ([243a69c](https://github.com/open-feature/go-sdk-contrib/commit/243a69cad40f1a36b302de3247a1de0068096867))
* update flagd e2e tests ([#466](https://github.com/open-feature/go-sdk-contrib/issues/466)) ([a8ee306](https://github.com/open-feature/go-sdk-contrib/commit/a8ee3068bd3b174bc75a6aeefa0441c61a5b43f7))

## [0.1.22](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.21...providers/flagd/v0.1.22) (2024-02-01)


### 🐛 Bug Fixes

* **deps:** update golang.org/x/exp digest to 1b97071 ([#379](https://github.com/open-feature/go-sdk-contrib/issues/379)) ([d2b2f1e](https://github.com/open-feature/go-sdk-contrib/commit/d2b2f1e19d5d6e9168174dc2d0196453a57ecac1))
* improvements to the shutdown of provider & support go 1.20 ([#442](https://github.com/open-feature/go-sdk-contrib/issues/442)) ([6056a8a](https://github.com/open-feature/go-sdk-contrib/commit/6056a8a6854d486a476ccd581ca4570148e1e025))


### ✨ New Features

* improve in-process error mapping ([#440](https://github.com/open-feature/go-sdk-contrib/issues/440)) ([1dee30b](https://github.com/open-feature/go-sdk-contrib/commit/1dee30b849fd93694d1945490b7aa53b82669770))

## [0.1.21](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.20...providers/flagd/v0.1.21) (2024-01-26)


### 🐛 Bug Fixes

* **deps:** update module connectrpc.com/connect to v1.14.0 ([#412](https://github.com/open-feature/go-sdk-contrib/issues/412)) ([f1ea55f](https://github.com/open-feature/go-sdk-contrib/commit/f1ea55feee1773f2582f8074044b36ca51480f36))
* **deps:** update module connectrpc.com/otelconnect to v0.7.0 ([#422](https://github.com/open-feature/go-sdk-contrib/issues/422)) ([a7a1540](https://github.com/open-feature/go-sdk-contrib/commit/a7a1540003298968b86b90ecfb80062e2a6f3671))
* **deps:** update module github.com/go-logr/logr to v1.4.1 ([#415](https://github.com/open-feature/go-sdk-contrib/issues/415)) ([77c597f](https://github.com/open-feature/go-sdk-contrib/commit/77c597f6d660b0c9920b809183aa9bb9fadf6238))
* **deps:** update module github.com/open-feature/flagd/core to v0.7.4 ([#384](https://github.com/open-feature/go-sdk-contrib/issues/384)) ([09b4224](https://github.com/open-feature/go-sdk-contrib/commit/09b42243885c1d74bb1d02d08aa3c5621ede7330))


### ✨ New Features

* flagd - make sure provider initialize only once ([#418](https://github.com/open-feature/go-sdk-contrib/issues/418)) ([c061cf4](https://github.com/open-feature/go-sdk-contrib/commit/c061cf4ac1e76cfe2aef9af52746fe0c14f6f610))
* flagd add scope to flag metadata ([#420](https://github.com/open-feature/go-sdk-contrib/issues/420)) ([b3949fe](https://github.com/open-feature/go-sdk-contrib/commit/b3949fe00318bee037fd2bd4e1655182f1fcdc31))
* flagd offline in-process support with flags sources from file ([#421](https://github.com/open-feature/go-sdk-contrib/issues/421)) ([8685cc0](https://github.com/open-feature/go-sdk-contrib/commit/8685cc0c1c4bee83ea38fa76e189c1a10840ec71))


### 📚 Documentation

* remove duplicate in doc ([#428](https://github.com/open-feature/go-sdk-contrib/issues/428)) ([b9a27d9](https://github.com/open-feature/go-sdk-contrib/commit/b9a27d9277b7e261d58cb76a34162033dbf5b971))

## [0.1.20](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.19...providers/flagd/v0.1.20) (2023-12-20)


### 🐛 Bug Fixes

* nil client causes panic ([#408](https://github.com/open-feature/go-sdk-contrib/issues/408)) ([74df958](https://github.com/open-feature/go-sdk-contrib/commit/74df9580ee627a5eb15427c9888b72ca10bf57e8))


### 🧹 Chore

* improve e2e test registration ([#400](https://github.com/open-feature/go-sdk-contrib/issues/400)) ([b22105c](https://github.com/open-feature/go-sdk-contrib/commit/b22105c392e24ce592020a5f1f652547bb5a89e0))
* update to go-sdk 1.9.0 ([#404](https://github.com/open-feature/go-sdk-contrib/issues/404)) ([11fa3ab](https://github.com/open-feature/go-sdk-contrib/commit/11fa3aba065a6dd81caca30e76efc16fb64a25e3))

## [0.1.19](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.18...providers/flagd/v0.1.19) (2023-12-05)


### ✨ New Features

* consolidate flagd in-process provider ([#394](https://github.com/open-feature/go-sdk-contrib/issues/394)) ([5f68c85](https://github.com/open-feature/go-sdk-contrib/commit/5f68c8520f2e2f480512c764ec39f6ecd0b30d1d))

## [0.1.18](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.17...providers/flagd/v0.1.18) (2023-11-14)


### 🐛 Bug Fixes

* **deps:** update module github.com/go-logr/logr to v1.3.0 ([#361](https://github.com/open-feature/go-sdk-contrib/issues/361)) ([d29fe69](https://github.com/open-feature/go-sdk-contrib/commit/d29fe69424c9a22f89a4147b70be090a1716981c))
* **deps:** update module github.com/google/go-cmp to v0.6.0 ([#354](https://github.com/open-feature/go-sdk-contrib/issues/354)) ([dde96df](https://github.com/open-feature/go-sdk-contrib/commit/dde96df31dc96629857e1866c026236c1c4a1843))
* **deps:** update module github.com/hashicorp/golang-lru/v2 to v2.0.7 ([#312](https://github.com/open-feature/go-sdk-contrib/issues/312)) ([cd2a830](https://github.com/open-feature/go-sdk-contrib/commit/cd2a83044ff88087a2d8ea03632e876b1cf00bd0))
* **deps:** update module github.com/open-feature/flagd/core to v0.6.6 ([#307](https://github.com/open-feature/go-sdk-contrib/issues/307)) ([b735aa8](https://github.com/open-feature/go-sdk-contrib/commit/b735aa849dfc63bbea0dcacffeb089390d91c782))
* **deps:** update module github.com/open-feature/flagd/core to v0.6.7 ([#350](https://github.com/open-feature/go-sdk-contrib/issues/350)) ([8bccc11](https://github.com/open-feature/go-sdk-contrib/commit/8bccc119f0f454adb3aa35e5c74bff43c50a80d3))
* **deps:** update module github.com/open-feature/go-sdk to v1.8.0 ([#329](https://github.com/open-feature/go-sdk-contrib/issues/329)) ([c99b527](https://github.com/open-feature/go-sdk-contrib/commit/c99b52728bad9dce52bfb78a08ae5f4eea83a397))
* **deps:** update module golang.org/x/net to v0.17.0 [security] ([#347](https://github.com/open-feature/go-sdk-contrib/issues/347)) ([a05689c](https://github.com/open-feature/go-sdk-contrib/commit/a05689cba2cea78c4741ae0e6a096fa370970b9c))


### 🔄 Refactoring

* migrate to connectrpc/connect-go ([#372](https://github.com/open-feature/go-sdk-contrib/issues/372)) ([aba4f4e](https://github.com/open-feature/go-sdk-contrib/commit/aba4f4e3cba0b9af23e665f0e736ec715d3f2fdc))

## [0.1.17](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.16...providers/flagd/v0.1.17) (2023-09-18)


### 🐛 Bug Fixes

* error handing, add e2e tests ([#334](https://github.com/open-feature/go-sdk-contrib/issues/334)) ([dfc3b5a](https://github.com/open-feature/go-sdk-contrib/commit/dfc3b5a73e6708aa852a2f2651468de96a754694))


### 🧹 Chore

* add flagd e2e tests ([#331](https://github.com/open-feature/go-sdk-contrib/issues/331)) ([b32ec64](https://github.com/open-feature/go-sdk-contrib/commit/b32ec64e212935d23e1302321bb60f1e5dc345dd))
* add missing go-cmp dep ([#330](https://github.com/open-feature/go-sdk-contrib/issues/330)) ([c9c7d66](https://github.com/open-feature/go-sdk-contrib/commit/c9c7d66b745ce09e49069b67af9e3bf8a92dfa85))

## [0.1.16](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.15...providers/flagd/v0.1.16) (2023-09-05)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.6.0 ([#289](https://github.com/open-feature/go-sdk-contrib/issues/289)) ([13eeb48](https://github.com/open-feature/go-sdk-contrib/commit/13eeb482ee3d69c5fb8100563501c2250b6454f1))
* **deps:** update module github.com/open-feature/go-sdk to v1.7.0 ([#315](https://github.com/open-feature/go-sdk-contrib/issues/315)) ([3f049ad](https://github.com/open-feature/go-sdk-contrib/commit/3f049ad34e93c3b9b9d4cf5a2e56f3777eb858e6))


### ✨ New Features

* Eventing support for flagd provider ([#317](https://github.com/open-feature/go-sdk-contrib/issues/317)) ([6b373cb](https://github.com/open-feature/go-sdk-contrib/commit/6b373cb393729c6f1f2a31b334cf06fac65dd369))

## [0.1.15](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.14...providers/flagd/v0.1.15) (2023-08-02)


### 🐛 Bug Fixes

* **deps:** update module github.com/bufbuild/connect-go to v1.10.0 ([#284](https://github.com/open-feature/go-sdk-contrib/issues/284)) ([a6a67b0](https://github.com/open-feature/go-sdk-contrib/commit/a6a67b01c0441b4cb4881945109265bd6d9e8cd6))
* **deps:** update module github.com/open-feature/flagd/core to v0.6.2 ([#286](https://github.com/open-feature/go-sdk-contrib/issues/286)) ([6290121](https://github.com/open-feature/go-sdk-contrib/commit/62901210bda2aa62333b6648ae9ab9dad9612d48))


### ✨ New Features

* flag evaluation metadata ([#278](https://github.com/open-feature/go-sdk-contrib/issues/278)) ([b0a61cd](https://github.com/open-feature/go-sdk-contrib/commit/b0a61cde5abcd7ea0bc10ef7c6174fc1be5cb423))

## [0.1.14](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.13...providers/flagd/v0.1.14) (2023-07-21)


### 🐛 Bug Fixes

* **deps:** update module buf.build/gen/go/open-feature/flagd/bufbuild/connect-go to v1.5.2-20230710190440-2333a9579c1a.1 ([#260](https://github.com/open-feature/go-sdk-contrib/issues/260)) ([35d2774](https://github.com/open-feature/go-sdk-contrib/commit/35d277438add84d139dbc416e0a9e180eade7ed2))
* **deps:** update module github.com/hashicorp/golang-lru/v2 to v2.0.4 ([#233](https://github.com/open-feature/go-sdk-contrib/issues/233)) ([dc053d0](https://github.com/open-feature/go-sdk-contrib/commit/dc053d05d49ab4500b443dd5c933a9feaec1ca6a))
* **deps:** update module github.com/open-feature/flagd/core to v0.5.4 ([#235](https://github.com/open-feature/go-sdk-contrib/issues/235)) ([b908b07](https://github.com/open-feature/go-sdk-contrib/commit/b908b07b7f73a345bd8aed1e85050e2e7fa0a878))
* **deps:** update module github.com/open-feature/flagd/core to v0.6.0 ([#259](https://github.com/open-feature/go-sdk-contrib/issues/259)) ([3121c53](https://github.com/open-feature/go-sdk-contrib/commit/3121c5388f24614d7131cfc5ff47402faa6e64b5))
* **deps:** update module github.com/open-feature/go-sdk to v1.5.1 ([#263](https://github.com/open-feature/go-sdk-contrib/issues/263)) ([c75ffd6](https://github.com/open-feature/go-sdk-contrib/commit/c75ffd6017689a86860dec92c1a1564b6145f0c9))
* **deps:** update module golang.org/x/net to v0.12.0 ([#240](https://github.com/open-feature/go-sdk-contrib/issues/240)) ([b4b161e](https://github.com/open-feature/go-sdk-contrib/commit/b4b161ecb8e135fe90215e63696bb664ed06b161))
* **deps:** update module google.golang.org/protobuf to v1.31.0 ([#249](https://github.com/open-feature/go-sdk-contrib/issues/249)) ([909b0b7](https://github.com/open-feature/go-sdk-contrib/commit/909b0b73fca87f5f0f374ff299eae4f7fa884f5d))

## [0.1.13](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.12...providers/flagd/v0.1.13) (2023-06-09)


### 🐛 Bug Fixes

* **deps:** update module buf.build/gen/go/open-feature/flagd/bufbuild/connect-go to v1.5.2-20230317150644-afd1cc2ef580.1 ([#170](https://github.com/open-feature/go-sdk-contrib/issues/170)) ([a04997e](https://github.com/open-feature/go-sdk-contrib/commit/a04997eea44f89bcf607625a4f02cfc45587914b))
* **deps:** update module github.com/bufbuild/connect-opentelemetry-go to v0.2.0 ([#192](https://github.com/open-feature/go-sdk-contrib/issues/192)) ([13c923f](https://github.com/open-feature/go-sdk-contrib/commit/13c923f91407c8ba2de002ab8d5e82d563f399c8))
* **deps:** update module golang.org/x/net to v0.10.0 ([#165](https://github.com/open-feature/go-sdk-contrib/issues/165)) ([e7249e2](https://github.com/open-feature/go-sdk-contrib/commit/e7249e26f20cfe4a6c99ecc4da8583971ba080fc))


### 📚 Documentation

* update flagd port in readme ([#207](https://github.com/open-feature/go-sdk-contrib/issues/207)) ([919d808](https://github.com/open-feature/go-sdk-contrib/commit/919d80855ae4e12ba7908626ddef6b81f34f570f))


### 🧹 Chore

* bump interlinked deps ([#236](https://github.com/open-feature/go-sdk-contrib/issues/236)) ([ea2233c](https://github.com/open-feature/go-sdk-contrib/commit/ea2233cc92f0bbb20affa61776a7b9ac166f2575))
* update module github.com/open-feature/go-sdk to v1.4.0 ([#223](https://github.com/open-feature/go-sdk-contrib/issues/223)) ([7c8ea46](https://github.com/open-feature/go-sdk-contrib/commit/7c8ea46e3e094f746dbf6d80ba6a1b606314e8d7))

## [0.1.12](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.11...providers/flagd/v0.1.12) (2023-05-10)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/flagd/core to v0.5.3 ([#195](https://github.com/open-feature/go-sdk-contrib/issues/195)) ([c9cd501](https://github.com/open-feature/go-sdk-contrib/commit/c9cd5011f18c1994b718423847c40adc88af2030))

## [0.1.11](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.10...providers/flagd/v0.1.11) (2023-04-27)


### 🔄 Refactoring

* reduce duplication of flagd provider service ([#135](https://github.com/open-feature/go-sdk-contrib/issues/135)) ([f444b38](https://github.com/open-feature/go-sdk-contrib/commit/f444b38b13f4230b1243e89ef7b4a942338025f0))


### 🐛 Bug Fixes

* **deps:** update module github.com/bufbuild/connect-go to v1.7.0 ([#177](https://github.com/open-feature/go-sdk-contrib/issues/177)) ([591759f](https://github.com/open-feature/go-sdk-contrib/commit/591759fd59c9425e0583c35b67bdeddca4173b88))
* **deps:** update module github.com/go-logr/logr to v1.2.4 ([#158](https://github.com/open-feature/go-sdk-contrib/issues/158)) ([5fdfa0e](https://github.com/open-feature/go-sdk-contrib/commit/5fdfa0e9cf21ef2ebf8a86fbc0c71cc591b185c9))
* **deps:** update module github.com/hashicorp/golang-lru/v2 to v2.0.2 ([#142](https://github.com/open-feature/go-sdk-contrib/issues/142)) ([e9149a3](https://github.com/open-feature/go-sdk-contrib/commit/e9149a3f451f65ddc1576cd09376a23158de9e15))
* **deps:** update module github.com/open-feature/flagd to v0.4.2 ([#134](https://github.com/open-feature/go-sdk-contrib/issues/134)) ([ad8c67e](https://github.com/open-feature/go-sdk-contrib/commit/ad8c67edbc095b4282b5ebfdd6970d8827ba45d1))
* **deps:** update module golang.org/x/net to v0.7.0 [security] ([#136](https://github.com/open-feature/go-sdk-contrib/issues/136)) ([d7455d6](https://github.com/open-feature/go-sdk-contrib/commit/d7455d68ff5ee1488ac1354dcfeaef0a2dd77e42))
* **deps:** update module golang.org/x/net to v0.8.0 ([#148](https://github.com/open-feature/go-sdk-contrib/issues/148)) ([6e695a3](https://github.com/open-feature/go-sdk-contrib/commit/6e695a3e21f48a52fc74b9aa389c4b0a1b51c009))
* **deps:** update module google.golang.org/protobuf to v1.29.1 ([#141](https://github.com/open-feature/go-sdk-contrib/issues/141)) ([f2a924f](https://github.com/open-feature/go-sdk-contrib/commit/f2a924ff331fbcfd479e948805223f02af9c032b))
* **deps:** update module google.golang.org/protobuf to v1.30.0 ([#151](https://github.com/open-feature/go-sdk-contrib/issues/151)) ([bf98120](https://github.com/open-feature/go-sdk-contrib/commit/bf98120d82218471c7acc2773c737d7bff64e401))


### 🧹 Chore

* fix flagd dependencies after mono repo split ([#172](https://github.com/open-feature/go-sdk-contrib/issues/172)) ([4b10a18](https://github.com/open-feature/go-sdk-contrib/commit/4b10a1833bad5b7f91c6fe2a4c4c2395e14657e4))


### ✨ New Features

* otel interceptor for flagd go-sdk ([#176](https://github.com/open-feature/go-sdk-contrib/issues/176)) ([17e5ab7](https://github.com/open-feature/go-sdk-contrib/commit/17e5ab796717c090bb203ebc766375e8efada23b))

## [0.1.10](https://github.com/open-feature/go-sdk-contrib/compare/providers/flagd/v0.1.9...providers/flagd/v0.1.10) (2023-03-02)


### Bug Fixes

* apply lru cache to provider ([#131](https://github.com/open-feature/go-sdk-contrib/issues/131)) ([79fe435](https://github.com/open-feature/go-sdk-contrib/commit/79fe435181fc9cfa95f2f7ef49a007a784cc2c88))

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
