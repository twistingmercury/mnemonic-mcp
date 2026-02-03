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
- Configuration package implementation (`internal/config`)
- Layered configuration loading (defaults → file → environment variables)
- Comprehensive validation with clear error messages for all configuration sections
- Server integration with configurable timeouts, TLS, and graceful shutdown
- Telemetry package (`internal/telemetry`) with otelx integration for unified OpenTelemetry initialization
- Middleware package (`internal/middleware`) with tracing and request metrics middleware for Gin
- Metrics package (`internal/metrics`) with domain-specific metrics for routing, patterns, and database operations
- Distributed tracing support via otelgin middleware
- Structured logging with trace correlation via otelx
- Request metrics: count, duration histograms, in-flight counters
- Version package (`internal/version`) for build metadata, eliminating upward dependencies
- Agent repository package (`internal/repository`) with PostgreSQL implementation
- Agent schema migrations (002_create_agents.up.sql and 002_create_agents.down.sql)
- Repository error types: ErrAgentExists, ErrAgentNotFound, ErrAgentInUse
- List options for pagination support in repository queries
- Comprehensive unit tests for agent repository with pgxmock

### Changed

- Build script supports `LOCAL_BUILD` flag for CI vs local behavior
- Push logic moved from build script to CD workflow
- Server now initializes telemetry and registers observability middleware
- Configuration validation includes log level validation (fail-fast on invalid level)
- Moved MetricsRegistry from server package to telemetry package for proper dependency direction
- Refactored cmd/version to delegate to internal/version, fixing internal->cmd dependency violation

### Fixed

- Typo in Mnemonic E2E tests docker-compose.yaml (`menmonic_tests` → `mnemonic_tests`)
