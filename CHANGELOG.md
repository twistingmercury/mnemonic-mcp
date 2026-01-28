# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- GitHub Actions CI workflow for Mnemonic service (`mnemonic-ci.yaml`)
- GitHub Actions CD workflow for Mnemonic service (`mnemonic-cd.yaml`)
- Docker image artifact passing between CI and CD workflows
- E2E test execution during CI builds
- Cleanup trap in build script for docker compose

### Changed

- Build script supports `LOCAL_BUILD` flag for CI vs local behavior
- Push logic moved from build script to CD workflow

### Fixed

- Typo in Mnemonic E2E tests docker-compose.yaml (`menmonic_tests` → `mnemonic_tests`)
