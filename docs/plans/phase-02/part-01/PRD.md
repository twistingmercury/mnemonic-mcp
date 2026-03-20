# PRD: Remove REST API and Migrations from Mnemonic

*Gralph processes cycles in this document from top to bottom. Checklist markers are significant: `- [ ]` (open), `- [x]` (complete), `- [~]` (abandoned). Each cycle must be small, independently verifiable, and assigned to exactly one agent.*

## Objective

Strip the mnemonic binary down to MCP server + enrichment worker only. The Admin REST API has moved to a separate `mnemonic-api` project. Database migrations have moved to `mnemonic-dbs`, which produces pre-built Docker images. Remove all REST API code, migration wiring, and related tests from this repository.

## Problem Statement

The mnemonic binary currently bundles the Admin REST API (Gin, port 8080), MCP server (port 8081), and enrichment worker in one process. The REST API is now a separate project. Migration infrastructure now lives in `~/dev/mnemonic-dbs`, which produces:

- `ghcr.io/twistingmercury/mnemonic-postgres:v1.0.0-dev`
- `ghcr.io/twistingmercury/mnemonic-neo4j:v1.0.0-dev`

These images have the schema already applied. The `migrate/migrate` service and raw `pgvector/pgvector` / `neo4j:5` images are no longer needed.

## Success Criteria

- `cd src/mnemonic && go build ./...` exits 0 with no REST API handler packages in the module.
- `cd src/mnemonic && go test ./...` exits 0.
- `docker compose config --quiet` exits 0 with the new database images and no `migrate` service.
- The binary starts MCP server + enrichment worker; `/health` and `/version` remain accessible via Gin on port 8080.

## Scope

### In scope

- Update `docker-compose.yaml`: new DB images, remove `migrate` service, keep all env vars explicit.
- Remove `src/migrations/` from version control (physically moved to `mnemonic-dbs`).
- Delete REST API handler packages: `internal/handlers/agents/`, `handlers/patterns/`, `handlers/skillfiles/`, `handlers/skills/`, `handlers/respond.go` — keep `handlers/operations/`.
- Delete REST-only service packages: `internal/service/agent/`, `service/skill/`, `service/skillfile/`.
- Delete REST-only repository packages: `internal/repository/skill/`, `repository/skillfile/`.
- Refactor `internal/server/server.go`: remove `RegisterAPIRoutes` call, slim `wireDependencies` (drop `agentSvc`, `skillSvc`, `skillFileSvc`), remove `Services` type usage.
- Delete `internal/server/routes.go` (defines `Services` and `RegisterAPIRoutes`).
- Remove `VocabularyConfig` from `internal/config/` (used only by the deleted pattern handler).
- Remove `src/mnemonic/config.yaml` (vocabulary list).
- Remove swag generation steps from `build/Dockerfile`.
- Run `go mod tidy` to drop swaggo and other REST-only deps.
- Delete `src/mnemonic/tests/` (E2E tests against the REST API).
- Delete `docs/openapi/mnemonic-v1.yaml` (now owned by `mnemonic-api`).
- Update `.github/workflows/mnemonic-ci.yaml`: remove `src/mnemonic/tests/**` path trigger.

### Out of scope

- Changes to MCP tool logic or enrichment worker behavior.
- Removing `internal/middleware/` (still used by the Gin router for `/health` and `/version`).
- Removing `internal/handlers/operations/` (still used for `/health` and `/version`).
- Moving code to `mnemonic-api`.
- Authentication or multi-user support.

## Constraints and Decisions

- Go 1.26.1, module path `github.com/twistingmercury/mnemonic`.
- New postgres image: `ghcr.io/twistingmercury/mnemonic-postgres:v1.0.0-dev`.
- New neo4j image: `ghcr.io/twistingmercury/mnemonic-neo4j:v1.0.0-dev`.
- All env vars in `docker-compose.yaml` remain explicit even though they are baked into the images.
- Gin stays as a dependency — the admin Gin router is kept for `/health` and `/version` on port 8080.
- `internal/handlers/operations/` is kept; it registers `/health` and `/version` on the Gin router.
- `cmd/main/main.go` `--health` flag continues to probe port 8080 — no change needed.
- `agentrepo` stays in `wireDependencies`: `searchSvc` and `patternSvc` still depend on it.
- Named volumes (`dev_postgres_data`, `dev_neo4j_data`) are kept — runtime data still needs persistence.

## Implementation Plan

- [x] **Cycle 1 - Update docker-compose**: Replace raw database images with the pre-built mnemonic-dbs images and remove the `migrate` service.
  - Agent: `devops engineer`
  - Files: `docker-compose.yaml`
  - Steps:
    - Change `dev_postgres.image` from `pgvector/pgvector:pg16` to `ghcr.io/twistingmercury/mnemonic-postgres:v1.0.0-dev`.
    - Change `dev_neo4j.image` from `neo4j:5` to `ghcr.io/twistingmercury/mnemonic-neo4j:v1.0.0-dev`.
    - Remove the entire `migrate` service block.
    - In `dev_api.depends_on`, remove `migrate: condition: service_completed_successfully`.
    - Keep all other env vars, ports, volumes, healthchecks, and networks unchanged.
  - Verify: `docker compose config --quiet`
  - Done: `docker compose config` exits 0; output contains `mnemonic-postgres` and `mnemonic-neo4j`; no `migrate/migrate` image appears.

- [x] **Cycle 2 - Remove migrations from version control**: The `src/migrations/` directory was physically moved to `mnemonic-dbs`. Remove it from this git repository.
  - Agent: `go software engineer`
  - Files: removes `src/migrations/`
  - Steps:
    - Run `git rm -r src/migrations/` from the repo root.
  - Verify: `git ls-files src/migrations | wc -l | grep -q '^0$'`
  - Done: `git ls-files src/migrations` returns no output.

- [x] **Cycle 3 - Refactor server package**: Slim `server.go` to remove REST API wiring; delete `routes.go`.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/server/server.go`, `src/mnemonic/internal/server/routes.go`
  - Steps:
    - Delete `routes.go` with `git rm src/mnemonic/internal/server/routes.go`. This removes the `Services` struct and `RegisterAPIRoutes` function.
    - In `server.go`, remove the `RegisterAPIRoutes(router, svc, cfg.Vocabulary)` call from `ListenAndServe`.
    - In `wireDependencies`, remove construction of `agentSvc`, `skillSvc`, `skillFileSvc`. Change the return type from `(Services, mcpserver.ToolDependencies, *enricher.Worker, error)` to `(mcpserver.ToolDependencies, *enricher.Worker, error)`.
    - Update the `wireDependencies` call site in `ListenAndServe` to match the new return signature (drop the `svc` variable).
    - Remove all imports that are now unused: handler packages, `swaggo`, `service/agent`, `service/skill`, `service/skillfile`, and any others the compiler flags.
    - Keep `setupRouter`, `CreateHTTPServer`, `runHTTPServer`, `operations.SetupHandlers`, and all Gin middleware — the admin Gin server still runs for `/health` and `/version`.
  - Verify: `cd src/mnemonic && go build ./internal/server/... && go test ./internal/server/...`
  - Done: Both commands exit 0.

- [x] **Cycle 4 - Delete REST API handler packages**: Remove the handler packages that served the REST API. Keep `operations/`.
  - Agent: `go software engineer`
  - Files: removes `internal/handlers/agents/`, `handlers/patterns/`, `handlers/skillfiles/`, `handlers/skills/`, `handlers/respond.go`, `handlers/respond_test.go`
  - Steps:
    - Run from the repo root: `git rm -r src/mnemonic/internal/handlers/agents src/mnemonic/internal/handlers/patterns src/mnemonic/internal/handlers/skillfiles src/mnemonic/internal/handlers/skills`
    - Run: `git rm src/mnemonic/internal/handlers/respond.go src/mnemonic/internal/handlers/respond_test.go`
    - Confirm `src/mnemonic/internal/handlers/operations/` and `handlers/doc.go` are untouched.
  - Verify: `cd src/mnemonic && go build ./...`
  - Done: `go build ./...` exits 0.

- [x] **Cycle 5 - Delete REST-only service and repository packages**: Remove service and repository packages that existed solely to support the REST API.
  - Agent: `go software engineer`
  - Files: removes `internal/service/agent/`, `service/skill/`, `service/skillfile/`, `repository/skill/`, `repository/skillfile/`
  - Steps:
    - Run from the repo root: `git rm -r src/mnemonic/internal/service/agent src/mnemonic/internal/service/skill src/mnemonic/internal/service/skillfile src/mnemonic/internal/repository/skill src/mnemonic/internal/repository/skillfile`
  - Verify: `cd src/mnemonic && go build ./...`
  - Done: `go build ./...` exits 0.

- [x] **Cycle 6 - Remove VocabularyConfig from config**: Drop the vocabulary type and its validation — it was only used by the deleted pattern handler.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/config/config.go`, `src/mnemonic/internal/config/config_test.go`, `src/mnemonic/config.yaml`
  - Steps:
    - In `config.go`, remove the `VocabularyConfig` struct, the `Vocabulary VocabularyConfig` field from `MnemonicConfig`, the `c.Vocabulary.validate()` call in `Validate()`, and the `VocabularyConfig.validate()` method.
    - In `config_test.go`, remove all vocabulary-related test cases and remove any `Vocabulary` field from test fixtures.
    - Delete `src/mnemonic/config.yaml` (`git rm src/mnemonic/config.yaml`).
  - Verify: `cd src/mnemonic && go build ./internal/config/... && go test ./internal/config/...`
  - Done: Both commands exit 0.

- [x] **Cycle 7 - Update Dockerfile**: Remove the swag generation steps that are no longer needed (routes.go no longer imports the generated swagger package).
  - Agent: `devops engineer`
  - Files: `src/mnemonic/build/Dockerfile`
  - Steps:
    - Remove the line `RUN go install github.com/swaggo/swag/cmd/swag@latest`.
    - Remove the line `RUN swag init -g cmd/main/main.go -o docs/swagger --parseInternal`.
    - Remove the comment above those lines referencing swagger doc generation.
  - Verify: `! grep -qE 'swag/cmd|swag init' src/mnemonic/build/Dockerfile && echo ok`
  - Done: The grep finds no swag installation or generation lines; `echo ok` prints.

- [x] **Cycle 8 - go mod tidy**: Remove unused dependencies (swaggo and any other packages dropped by the REST API removal).
  - Agent: `go software engineer`
  - Files: `src/mnemonic/go.mod`, `src/mnemonic/go.sum`
  - Steps:
    - Run `cd src/mnemonic && go mod tidy`.
    - Confirm that swaggo packages no longer appear as direct dependencies in `go.mod`.
  - Verify: `cd src/mnemonic && go build ./... && go mod verify`
  - Done: Both commands exit 0; `go.mod` contains no `swaggo` direct deps.

- [x] **Cycle 9 - Delete E2E tests and OpenAPI spec**: Remove the E2E test suite (tested the REST API) and the OpenAPI spec (now owned by `mnemonic-api`).
  - Agent: `go software engineer`
  - Files: removes `src/mnemonic/tests/`, `docs/openapi/`
  - Steps:
    - Run from the repo root: `git rm -r src/mnemonic/tests/ docs/openapi/`
  - Verify: `test ! -d src/mnemonic/tests && test ! -d docs/openapi && echo ok`
  - Done: Both directories are absent; `echo ok` prints.

- [ ] **Cycle 10 - Update CI workflow**: Remove the `src/mnemonic/tests/**` path trigger (the E2E test directory no longer exists).
  - Agent: `devops engineer`
  - Files: `.github/workflows/mnemonic-ci.yaml`
  - Steps:
    - Remove `- "src/mnemonic/tests/**"` from both the `push.paths` and `pull_request.paths` trigger lists.
    - No other changes needed — the workflow has no E2E step and no REST API image push to remove.
  - Verify: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/mnemonic-ci.yaml')); print('YAML valid')"`
  - Done: YAML parses without error; `YAML valid` prints.

- [ ] **Cycle 11 - Full test suite**: Verify the entire module builds cleanly and all remaining unit tests pass.
  - Agent: `go software engineer`
  - Files: any files requiring minor fixes to restore a passing state
  - Steps:
    - Run `cd src/mnemonic && go test ./...`.
    - Fix any compilation or test failures introduced by previous cycles.
    - Iterate until all tests pass.
  - Verify: `cd src/mnemonic && go build ./... && go test ./...`
  - Done: Both commands exit 0 with no test failures.

- [ ] **Cycle 12 - Full build**: Run the complete CI build pipeline via `make build` and fix any issues it surfaces.
  - Agent: `go software engineer`
  - Files: any files requiring fixes to make the build pass
  - Steps:
    - Run `cd src/mnemonic && make build`.
    - Fix any failures. The Docker build runs goimports, golangci-lint, govulncheck, gosec, and unit tests — fix whatever it reports.
    - Iterate until `make build` exits 0.
  - Verify: `cd src/mnemonic && make build`
  - Done: `make build` exits 0 and produces a Docker image.

## Risks and Mitigations

- Risk: `agentrepo` removal is accidentally included — it is still needed by `searchSvc` and `patternSvc`.
  - Mitigation: Cycle 5 explicitly lists only `service/agent`, `repository/skill`, and `repository/skillfile` for deletion; `repository/agent` is not listed.
- Risk: Deleting `respond.go` breaks the `handlers/doc.go` or `handlers/operations/` package.
  - Mitigation: `respond.go` is standalone; `operations/` does not import it. Cycle 4 verify runs `go build ./...`.
- Risk: Swag-generated `docs/swagger/` package is gitignored but still imported by `routes.go`.
  - Mitigation: `routes.go` is deleted in Cycle 3 before the swag package is removed in Cycle 8; the blank import disappears with the file.
- Risk: Cycle 3 removes `Services` from `wireDependencies` but a test in `internal/server` still references it.
  - Mitigation: Cycle 3 verify runs `go test ./internal/server/...`; any failure must be fixed before the cycle is marked done.

## Definition of Done

- `cd src/mnemonic && make build` exits 0 and produces a Docker image.
- `cd src/mnemonic && go test ./...` exits 0.
- `docker compose config --quiet` exits 0 with `mnemonic-postgres` and `mnemonic-neo4j` in the output.
- No `migrate/migrate` service in `docker-compose.yaml`.
- No `src/migrations/` directory tracked in git.
- No `internal/handlers/agents`, `patterns`, `skills`, or `skillfiles` directories exist.
- `internal/handlers/operations/` still exists and the binary exposes `/health` and `/version` on port 8080.
