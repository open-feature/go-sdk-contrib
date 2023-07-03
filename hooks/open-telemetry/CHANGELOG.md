# Changelog

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
