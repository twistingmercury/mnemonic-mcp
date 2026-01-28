# Mnemonic

Mnemonic is the backend server for ACE (Agent Coordination Engine), providing deterministic routing and dynamic pattern retrieval via REST API.

## Overview

Mnemonic is a stateless Go service that manages:

- Agent definitions and system prompts
- Routing rules for prompt-to-agent matching
- Pattern storage with semantic search (via PGVector and Neo4j)
- Background enrichment of patterns with LLM-extracted metadata

See the [System Architecture documentation](/docs/architecture/03-system-architecture.md#mnemonic) for detailed design.

## Building Locally

To build Mnemonic locally:

```bash
cd src/mnemonic
LOCAL_BUILD=1 ./build/build.sh
```

The `LOCAL_BUILD` flag:

- Tags the Docker image with `${BUILD_VER}-localdev` suffix instead of `latest`
- Runs a `docker run --version` check after the build to verify the image
- E2E tests run automatically regardless of the `LOCAL_BUILD` value

## CI/CD

Mnemonic has automated CI/CD workflows:

- **CI** (`/.github/workflows/mnemonic-ci.yaml`) - Builds, tests, creates artifact
- **CD** (`/.github/workflows/mnemonic-cd.yaml`) - Pushes Docker image to registry

Manual builds trigger the entire test suite and verify all integration points.

## Documentation

- [API Specification](/docs/design/mnemonic_service/api-specification.md)
- [Pattern Processing](/docs/design/mnemonic_service/pattern-processing.md)
- [Routing Engine](/docs/design/mnemonic_service/routing-engine.md)
- [Configuration Reference](/docs/design/mnemonic_service/configuration.md)
