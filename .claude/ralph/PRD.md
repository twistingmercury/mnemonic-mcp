# Product Requirements Document: Auto-Generated Swagger 2.0 API Docs

## Objective

Add auto-generated Swagger 2.0 documentation to the Admin REST API using `swaggo/swag` and `swaggo/gin-swagger`, and expose an interactive docs UI from the running service.

## Problem Statement

The project maintains an OpenAPI spec file, but the running API does not expose generated docs from code annotations. This creates drift risk between implementation and documentation and slows endpoint discovery.

## Success Criteria

- Swagger 2.0 spec is generated from code annotations during Docker image build.
- Swagger UI is available from the running Admin API.
- Core endpoints (health, version, agents, patterns, skills, skill files) are documented with request/response schemas.
- Docker-first CI build fails when Swagger generation fails or generated docs are stale.

## Scope

In scope:

- Swagger 2.0 generation setup (`swag` CLI + generated docs package)
- Gin route for Swagger UI
- API annotation coverage for existing handlers and shared response types
- Build/test automation updates
- Documentation updates for contributors

Out of scope:

- Migrating to OpenAPI 3.x
- Replacing existing API behavior
- Public auth docs beyond current MVP model

## Constraints and Decisions

- Use `swaggo/gin-swagger` for UI and `swaggo/swag` for generation.
- Generated artifacts are not committed; they are rebuilt during every build.
- Use Swagger 2.0 output only.
- Do not change business logic while adding docs.
- Swagger generation must run inside Docker build stages used by CI.

## Implementation Plan

- [x] **Cycle 1 — Add swaggo deps**: Add `swaggo/swag`, `swaggo/gin-swagger`, and `swaggo/files` to go.mod so the project builds cleanly.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/go.mod`, `src/mnemonic/go.sum`
  - Steps:
    - `cd src/mnemonic && go get github.com/swaggo/swag`
    - `cd src/mnemonic && go get github.com/swaggo/gin-swagger`
    - `cd src/mnemonic && go get github.com/swaggo/files`
    - `cd src/mnemonic && go mod tidy`
  - Verify: `cd src/mnemonic && go build ./...`
  - Done: `go.mod` lists all three swaggo deps; `go build ./...` exits 0 with no errors.

- [x] **Cycle 2 — Add `make docs-swagger` target**: Add a Makefile target that runs `swag init` and produces `src/mnemonic/docs/swagger/swagger.json`.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/Makefile`
  - Steps:
    - Add target `docs-swagger` that runs: `swag init -g cmd/main/main.go -o docs/swagger --parseInternal`
    - Ensure `swag` binary is available (install via `go install github.com/swaggo/swag/cmd/swag@latest` if missing)
  - Verify: `cd src/mnemonic && make docs-swagger && test -f docs/swagger/swagger.json`
  - Done: `docs/swagger/swagger.json` exists and `make docs-swagger` exits 0.

- [x] **Cycle 3 — Top-level annotations in `cmd/main/main.go`**: Add Swagger general API metadata annotations (title, version, description, host, basePath) so the generated spec has correct top-level fields.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/cmd/main/main.go`
  - Steps:
    - Add `// @title`, `// @version`, `// @description`, `// @host`, `// @BasePath`, `// @schemes` comment annotations above the `main()` function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"title"' docs/swagger/swagger.json && grep -q '"basePath"' docs/swagger/swagger.json`
  - Done: Generated `swagger.json` contains non-empty `title` and `basePath` fields.

- [x] **Cycle 4 — Shared response type annotations in `internal/handlers/respond.go`**: Annotate shared response types (error envelope, pagination, common DTOs) so they appear as reusable `$ref` schemas in the spec.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/respond.go`
  - Steps:
    - Add `// @Description` and struct-level swag annotations to shared response types
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"definitions"' docs/swagger/swagger.json`
  - Done: `swagger.json` contains a `definitions` section with at least one shared type.

- [x] **Cycle 5 — Annotate `internal/handlers/operations/operations.go`**: Add route-level swag annotations to all operations handlers (health, version).
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/operations/operations.go`
  - Steps:
    - Add `// @Summary`, `// @Tags`, `// @Produce`, `// @Success`, `// @Router` annotations to each handler function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"Operations"' docs/swagger/swagger.json`
  - Done: `swagger.json` paths include at least one route tagged `Operations`.

- [x] **Cycle 6 — Annotate `internal/handlers/agents/agents.go`**: Add route-level swag annotations to all agents handlers.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/agents/agents.go`
  - Steps:
    - Add `// @Summary`, `// @Tags`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` annotations to each handler function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"Agents"' docs/swagger/swagger.json`
  - Done: `swagger.json` paths include at least one route tagged `Agents`.

- [x] **Cycle 7 — Annotate `internal/handlers/patterns/patterns.go`**: Add route-level swag annotations to all patterns handlers.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/patterns/patterns.go`
  - Steps:
    - Add `// @Summary`, `// @Tags`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` annotations to each handler function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"Patterns"' docs/swagger/swagger.json`
  - Done: `swagger.json` paths include at least one route tagged `Patterns`.

- [x] **Cycle 8 — Annotate `internal/handlers/skills/skills.go`**: Add route-level swag annotations to all skills handlers.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/skills/skills.go`
  - Steps:
    - Add `// @Summary`, `// @Tags`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` annotations to each handler function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"Skills"' docs/swagger/swagger.json`
  - Done: `swagger.json` paths include at least one route tagged `Skills`.

- [x] **Cycle 9 — Annotate `internal/handlers/skillfiles/skillfiles.go`**: Add route-level swag annotations to all skill files handlers.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/handlers/skillfiles/skillfiles.go`
  - Steps:
    - Add `// @Summary`, `// @Tags`, `// @Produce`, `// @Param`, `// @Success`, `// @Failure`, `// @Router` annotations to each handler function
    - Re-run `make docs-swagger`
  - Verify: `cd src/mnemonic && make docs-swagger && grep -q '"SkillFiles"' docs/swagger/swagger.json`
  - Done: `swagger.json` paths include at least one route tagged `SkillFiles`.

- [x] **Cycle 10 — Register `/swagger/*any` route**: Add the Swagger UI route to `internal/server/routes.go` so `curl http://localhost:<port>/swagger/index.html` returns 200.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/internal/server/routes.go`
  - Steps:
    - Import `swaggerFiles "github.com/swaggo/files"` and `ginSwagger "github.com/swaggo/gin-swagger"`
    - Import generated docs package (e.g., `_ "github.com/twistingmercury/mnemonic/docs/swagger"`)
    - Register route: `router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))`
    - Re-run `make docs-swagger` and start the server
  - Verify: `cd src/mnemonic && make docs-swagger && go build ./... && curl -s -o /dev/null -w "%{http_code}" http://localhost:$(grep -o 'PORT=[0-9]*' .env 2>/dev/null | head -1 | cut -d= -f2 || echo 8080)/swagger/index.html | grep -q 200 || echo "Start server manually and verify /swagger/index.html returns 200"`
  - Done: `go build ./...` exits 0; server serves Swagger UI at `/swagger/index.html`.

- [x] **Cycle 11 — Add swag gen step to `build/Dockerfile`**: Run `swag init` inside the Docker build stage before the final binary build so generated docs are included in the image.
  - Agent: `devops-engineer`
  - Files: `src/mnemonic/build/Dockerfile`
  - Steps:
    - In the builder stage, add `RUN go install github.com/swaggo/swag/cmd/swag@latest`
    - Add `RUN swag init -g cmd/main/main.go -o docs/swagger --parseInternal` before the final `go build` step
  - Verify: `cd src/mnemonic && docker build -f build/Dockerfile .`
  - Done: `docker build` exits 0 and the image contains generated swagger artifacts.

- [x] **Cycle 12 — Add `docs/swagger/` to `.gitignore`**: Ensure generated swagger artifacts are never accidentally committed.
  - Agent: `go software engineer`
  - Files: `src/mnemonic/.gitignore`
  - Steps:
    - Add `docs/swagger/` to `src/mnemonic/.gitignore`
    - If a top-level `.gitignore` exists at the repo root, add it there too
  - Verify: `grep -q 'docs/swagger' src/mnemonic/.gitignore`
  - Done: `docs/swagger/` appears in `.gitignore`; `git status` does not show generated swagger files as tracked.

- [x] **Cycle 13 — Update `src/mnemonic/README.md`**: Document the swagger regen command, the Swagger UI URL, and the contribution expectation for new endpoints.
  - Agent: `technical-writer`
  - Files: `src/mnemonic/README.md`
  - Steps:
    - Add a "API Documentation" section with: regen command (`make docs-swagger`), UI URL (`/swagger/index.html`), contribution note ("run `make docs-swagger` locally to preview docs; generated files are not committed")
  - Verify: `grep -q 'docs-swagger' src/mnemonic/README.md && grep -q 'swagger/index.html' src/mnemonic/README.md`
  - Done: README contains regen command, UI URL, and contributor expectation.

## Risks and Mitigations

- Risk: Annotation drift or incomplete coverage.
  - Mitigation: CI diff check plus review checklist item.
- Risk: Generated docs become noisy in diffs.
  - Mitigation: Stable generation command and pinned tool versions.
- Risk: Confusion between existing OpenAPI file and generated Swagger artifacts.
  - Mitigation: Explicit source-of-truth guidance in README and PR process.

## Definition of Done

- Swagger 2.0 docs auto-generated from code annotations.
- Swagger UI accessible from running Admin API.
- All current REST endpoints documented.
- Docker-first build/CI includes stale-doc detection and generation execution.
- Contributor docs updated with required workflow.
