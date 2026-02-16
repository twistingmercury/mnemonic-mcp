# Phase 27: Deployment Update

> Part of the [MVP Implementation Plan](mvp-implementation-plan.md)

**Goal:** Update Docker and deployment configurations for the new binary and two-port architecture.

**Agent(s):** devops-engineer

**Dependencies:** Phase 25 (new entrypoint and two-port architecture)

---

## Step 1: Update Dockerfile

- Modify the Dockerfile to build `cmd/mnemonic/` as the entrypoint (instead of `cmd/main/`)
- Expose both ports: `EXPOSE 8080 8081`
- Agent: `devops-engineer`

## Step 2: Update docker-compose.yaml

- Add port mapping for MCP: `8081:8081`
- Verify admin port mapping: `8080:8080`
- Ensure Postgres and Neo4j services are present and healthy
- Agent: `devops-engineer`

## Step 3: Verify CI/CD pipeline

- Review `.github/workflows/` for any references to `cmd/main/` that need updating to `cmd/mnemonic/`
- Update build commands if needed
- Agent: `devops-engineer`

## Step 4: Verify deployment works end-to-end

- Run: `docker-compose build`
- Run: `docker-compose up -d`
- Verify: `curl http://localhost:8080/ops/health` returns 200
- Verify: `curl http://localhost:8080/api/agents` returns 200
- Verify MCP endpoint is reachable on port 8081
- Agent: `devops-engineer`

## Step 5: Commit

```bash
git add Dockerfile docker-compose.yaml .github/
git commit -m "feat(pivot): update deployment for dual-port architecture (admin:8080, mcp:8081)"
```
