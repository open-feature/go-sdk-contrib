# Changelog

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
