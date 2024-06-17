# Changelog

## [0.3.3](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.3.2...hooks/open-telemetry/v0.3.3) (2024-06-17)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.11.0 ([#501](https://github.com/open-feature/go-sdk-contrib/issues/501)) ([3f0eaa5](https://github.com/open-feature/go-sdk-contrib/commit/3f0eaa575500baa663dc24dbfc6cf8214565471f))


### ✨ New Features

* **otel:** add NewMetricsHook, use api interface instead of SDK type ([#530](https://github.com/open-feature/go-sdk-contrib/issues/530)) ([0472b0d](https://github.com/open-feature/go-sdk-contrib/commit/0472b0d59732be7f93b5e79875c0a61fcd4a35e6))

## [0.3.2](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.3.1...hooks/open-telemetry/v0.3.2) (2024-04-11)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.10.0 ([#469](https://github.com/open-feature/go-sdk-contrib/issues/469)) ([21810af](https://github.com/open-feature/go-sdk-contrib/commit/21810afc33fce9a3940ec9dc59e65f140fcbaa57))
* **deps:** update opentelemetry-go monorepo to v1.25.0 ([#493](https://github.com/open-feature/go-sdk-contrib/issues/493)) ([6241825](https://github.com/open-feature/go-sdk-contrib/commit/62418255a6a0c48cd6ad8b94c9fd5a4c6943b1da))


### 🧹 Chore

* bump Go to version 1.21 ([#452](https://github.com/open-feature/go-sdk-contrib/issues/452)) ([7ec90ce](https://github.com/open-feature/go-sdk-contrib/commit/7ec90ce4f9b06670187561afd9e342eed4228be1))

## [0.3.1](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.3.0...hooks/open-telemetry/v0.3.1) (2024-02-08)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.7.0 ([#315](https://github.com/open-feature/go-sdk-contrib/issues/315)) ([3f049ad](https://github.com/open-feature/go-sdk-contrib/commit/3f049ad34e93c3b9b9d4cf5a2e56f3777eb858e6))
* **deps:** update module github.com/open-feature/go-sdk to v1.8.0 ([#329](https://github.com/open-feature/go-sdk-contrib/issues/329)) ([c99b527](https://github.com/open-feature/go-sdk-contrib/commit/c99b52728bad9dce52bfb78a08ae5f4eea83a397))
* **deps:** update module go.opentelemetry.io/otel/sdk/metric to v1 ([#371](https://github.com/open-feature/go-sdk-contrib/issues/371)) ([50fcef6](https://github.com/open-feature/go-sdk-contrib/commit/50fcef618aa7eb3800d9ab476dbebf61f5ee401c))
* **deps:** update opentelemetry-go monorepo ([#318](https://github.com/open-feature/go-sdk-contrib/issues/318)) ([d3c8e87](https://github.com/open-feature/go-sdk-contrib/commit/d3c8e8752762a9df8bf796afe4f93c2741887463))
* **deps:** update opentelemetry-go monorepo to v1.21.0 ([#383](https://github.com/open-feature/go-sdk-contrib/issues/383)) ([f417648](https://github.com/open-feature/go-sdk-contrib/commit/f417648ccb2875562eb9215eb830b0a0eba2e44c))


### 🧹 Chore

* update to go-sdk 1.9.0 ([#404](https://github.com/open-feature/go-sdk-contrib/issues/404)) ([11fa3ab](https://github.com/open-feature/go-sdk-contrib/commit/11fa3aba065a6dd81caca30e76efc16fb64a25e3))

## [0.3.0](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.8...hooks/open-telemetry/v0.3.0) (2023-08-21)


### ⚠ BREAKING CHANGES

* attribute setter callbacks for otel hooks and remove deprecated constructors  ([#311](https://github.com/open-feature/go-sdk-contrib/issues/311))

### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.6.0 ([#289](https://github.com/open-feature/go-sdk-contrib/issues/289)) ([13eeb48](https://github.com/open-feature/go-sdk-contrib/commit/13eeb482ee3d69c5fb8100563501c2250b6454f1))


### ✨ New Features

* attribute setter callbacks for otel hooks and remove deprecated constructors  ([#311](https://github.com/open-feature/go-sdk-contrib/issues/311)) ([27f7ca7](https://github.com/open-feature/go-sdk-contrib/commit/27f7ca7d17667b33e2ed8206b96dc304e5d33454))

## [0.2.8](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.7...hooks/open-telemetry/v0.2.8) (2023-07-21)


### 🐛 Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.5.1 ([#263](https://github.com/open-feature/go-sdk-contrib/issues/263)) ([c75ffd6](https://github.com/open-feature/go-sdk-contrib/commit/c75ffd6017689a86860dec92c1a1564b6145f0c9))

## [0.2.7](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.6...hooks/open-telemetry/v0.2.7) (2023-07-03)


### ✨ New Features

* Metric hook API change to provide better hook constructor flexibility ([#254](https://github.
  com/open-feature/go-sdk-contrib/issues/254)) ([c855a67](https://github.com/open-feature/go-sdk-contrib/commit/c855a677e34d3f6b1d8b24bc721ce389a19f742f))

## [0.2.6](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.5...hooks/open-telemetry/v0.2.6) (2023-06-21)


### 🔄 Refactoring

* **hooks/open-telemetry:** Use semconv for trace attribites ([#245](https://github.com/open-feature/go-sdk-contrib/issues/245)) ([8bfbbf4](https://github.com/open-feature/go-sdk-contrib/commit/8bfbbf42e2872e86946fb8ea191fbe5036a6a063))

## [0.2.5](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.4...hooks/open-telemetry/v0.2.5) (2023-06-09)


### 🐛 Bug Fixes

* **deps:** update opentelemetry-go monorepo to v1.15.1 ([#189](https://github.com/open-feature/go-sdk-contrib/issues/189)) ([c42a1c4](https://github.com/open-feature/go-sdk-contrib/commit/c42a1c4371cc219cdfc7ae23c940641548482306))
* set error state with a message ([#205](https://github.com/open-feature/go-sdk-contrib/issues/205)) ([ce14e22](https://github.com/open-feature/go-sdk-contrib/commit/ce14e22870a9329fe02dd7dba5634d62f9845728))


### ✨ New Features

* metric hooks ([#217](https://github.com/open-feature/go-sdk-contrib/issues/217)) ([3a055e4](https://github.com/open-feature/go-sdk-contrib/commit/3a055e45a2ef549696ac2e7eb0a0c388ee3bbb83))
* otel hook error status override option ([#209](https://github.com/open-feature/go-sdk-contrib/issues/209)) ([48fd3f6](https://github.com/open-feature/go-sdk-contrib/commit/48fd3f6f12a07c2e0e6a92e516e5bab071e8bff0))


### 🧹 Chore

* bump interlinked deps ([#236](https://github.com/open-feature/go-sdk-contrib/issues/236)) ([ea2233c](https://github.com/open-feature/go-sdk-contrib/commit/ea2233cc92f0bbb20affa61776a7b9ac166f2575))
* rename constructor method ([#237](https://github.com/open-feature/go-sdk-contrib/issues/237)) ([b54f2c5](https://github.com/open-feature/go-sdk-contrib/commit/b54f2c50d878e95b07d7444e5912665a4433c80e))
* update module github.com/open-feature/go-sdk to v1.4.0 ([#223](https://github.com/open-feature/go-sdk-contrib/issues/223)) ([7c8ea46](https://github.com/open-feature/go-sdk-contrib/commit/7c8ea46e3e094f746dbf6d80ba6a1b606314e8d7))

## [0.2.4](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.3...hooks/open-telemetry/v0.2.4) (2023-04-13)


### 🧹 Chore

* upgrade go-sdk to v1.3.0. enforce otel hook to be valid Hook at compile time ([#137](https://github.com/open-feature/go-sdk-contrib/issues/137)) ([3944f05](https://github.com/open-feature/go-sdk-contrib/commit/3944f05aa6b9c109ef027e55d7e6d170a388b413))


### 🐛 Bug Fixes

* **deps:** update opentelemetry-go monorepo to v1.14.0 ([#108](https://github.com/open-feature/go-sdk-contrib/issues/108)) ([711bc52](https://github.com/open-feature/go-sdk-contrib/commit/711bc5286b0fcfbd23daf0d6c41253f07571e97b))

## [0.2.3](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.2...hooks/open-telemetry/v0.2.3) (2023-03-02)


### Features

* ⚠️ requires OpenFeature Go SDK v1.3.0 or above ⚠️ absorbed Hook API changes ([#130](https://github.com/open-feature/go-sdk-contrib/issues/130)) ([a65b009](https://github.com/open-feature/go-sdk-contrib/commit/a65b00957a425b89c261a979f81dcfdf2f5a2bcb))

## [0.2.2](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.1...hooks/open-telemetry/v0.2.2) (2023-02-21)


### Bug Fixes

* **deps:** update module github.com/open-feature/go-sdk to v1.2.0 ([#103](https://github.com/open-feature/go-sdk-contrib/issues/103)) ([eedb577](https://github.com/open-feature/go-sdk-contrib/commit/eedb577745fd98d5189132ebbaa8eb82bdf99dd8))

## [0.2.1](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry/v0.2.0...hooks/open-telemetry/v0.2.1) (2023-01-31)


### Bug Fixes

* **deps:** update opentelemetry-go monorepo to v1.12.0 ([#57](https://github.com/open-feature/go-sdk-contrib/issues/57)) ([e48e4a0](https://github.com/open-feature/go-sdk-contrib/commit/e48e4a0458a38eb1a028c5c3570ceb522c7e7319))

## [0.2.0](https://github.com/open-feature/go-sdk-contrib/compare/hooks/open-telemetry-v0.1.0...hooks/open-telemetry/v0.2.0) (2023-01-25)


### ⚠ BREAKING CHANGES

* Update OTel Hook to conform to official conventions ([#87](https://github.com/open-feature/go-sdk-contrib/issues/87))

### Features

* Update OTel Hook to conform to official conventions ([#87](https://github.com/open-feature/go-sdk-contrib/issues/87)) ([4e725ae](https://github.com/open-feature/go-sdk-contrib/commit/4e725ae4ebd80a95f617b64490f7a57ce2441fa5))
