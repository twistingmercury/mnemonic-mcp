# Docker Compose Quick Start

Get Mnemonic running locally with all dependencies in under 2 minutes.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- OpenAI API key (get one at https://platform.openai.com/api-keys)

## Quick Start

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Add your OpenAI API key to .env
echo "MNEMONIC_OPENAI_API_KEY=sk-your-key-here" >> .env

# 3. Start all services
docker compose up -d

# 4. Verify services are running
docker compose ps

# 5. Check health
curl http://localhost:8080/ops/health
```

## Services

| Service   | URL                       | Credentials (dev only)          |
|-----------|---------------------------|---------------------------------|
| Mnemonic  | http://localhost:8080     | N/A                             |
| Metrics   | http://localhost:9090     | N/A                             |
| Neo4j UI  | http://localhost:7475     | neo4j / mnemonic_dev            |
| Postgres  | localhost:5433            | mnemonic / mnemonic_dev         |

## Common Commands

```bash
# View logs
docker compose logs -f mnemonic

# Restart a service
docker compose restart mnemonic

# Stop all services
docker compose down

# Clean slate (removes volumes)
docker compose down -v

# Rebuild after code changes
docker compose up -d --build
```

## Troubleshooting

**Build fails:**
```bash
docker compose build --no-cache
```

**Port conflicts:**
Ports are offset from defaults (5433 instead of 5432, 7475/7688 instead of 7474/7687) to avoid conflicts with other local services. If you still have conflicts, stop other services or modify the ports in `docker-compose.yaml`.

**Database connection errors:**
Wait for health checks to pass:
```bash
docker compose ps
# Both postgres and neo4j should show "healthy"
```

**OpenAI errors:**
Verify your API key:
```bash
grep MNEMONIC_OPENAI_API_KEY .env
```

## Next Steps

See `build/README.md` for detailed documentation on:
- Manual builds
- Production deployment
- Health checks
- Database credentials
- Advanced configuration
