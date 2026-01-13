# Memory MCP Server

> The docker compose file is for a local Cognee deployment for development purposes. It doesn't represent how the
> memory server would be deployed in K8s or in the cloud.

- **Cognee MCP Server** - For Claude Code integration via MCP protocol (port 4000)
- **Cognee REST API** - For HTTP/REST API access from any application (port 8000)
- **PostgreSQL with pgvector** - Relational database and vector storage
- **Neo4j** - Knowledge graph database

## Architecture

The deployment runs separate MCP and REST API servers that share common database backends (`docker-compose.yaml`).

```mermaid
graph TB
    subgraph Applications["Your Applications"]
        Claude["Claude Code<br/>(MCP)"]
        HTTP["HTTP Clients<br/>(curl/Postman)"]
    end

    Claude -->|":4000"| MCP["Cognee MCP Server<br/>(port 4000)"]
    HTTP -->|":8000"| API["Cognee REST API Server<br/>(port 8000)"]

    MCP --> Databases
    API --> Databases

    subgraph Databases["Shared Databases"]
        Postgres["PostgreSQL + pgvector<br/>Relational + Vector Store"]
        Neo4j["Neo4j<br/>Knowledge Graph"]
    end

    style Applications fill:#E0E7FF,stroke:#6366F1,stroke-width:2px,color:#000
    style Claude fill:#4A9EFF,stroke:#2563EB,stroke-width:2px,color:#000
    style HTTP fill:#4A9EFF,stroke:#2563EB,stroke-width:2px,color:#000
    style MCP fill:#F59E0B,stroke:#D97706,stroke-width:2px,color:#000
    style API fill:#F59E0B,stroke:#D97706,stroke-width:2px,color:#000
    style Databases fill:#FEF3C7,stroke:#F59E0B,stroke-width:2px,color:#000
    style Postgres fill:#A78BFA,stroke:#7C3AED,stroke-width:2px,color:#000
    style Neo4j fill:#34D399,stroke:#059669,stroke-width:2px,color:#000
```

**Note:** Both servers share the same database backends. Data added through one interface is immediately available through the other.

## Quick Start

### 1. Start Services

```bash
docker compose up -d
```

### 2. Verify Services

```bash
docker compose ps
```

All services should show as "healthy".

### 3. Connect to Cognee

#### Option A: Claude Code (MCP)

From your development machine:

```bash
claude mcp add --scope user --transport sse cognee http://YOUR_SERVER_IP:4000/sse
```

Verify connection:

```bash
claude mcp list
```

#### Option B: REST API

The REST API is available at `http://localhost:8000`

**API Documentation:** <http://localhost:8000/docs> (Swagger UI)

**Example API calls:**

```bash
# Health check
curl http://localhost:8000/health

# Add data
curl -X POST http://localhost:8000/api/v1/add \
  -H "Content-Type: application/json" \
  -d '{"data": "Your text or file path here"}'

# Search
curl -X POST http://localhost:8000/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query_text": "your search query"}'
```

## Service Endpoints

| Service               | Endpoint                       | Purpose                                    |
| --------------------- | ------------------------------ | ------------------------------------------ |
| **Cognee MCP**        | `http://localhost:4000/sse`    | MCP Server (SSE transport) for Claude Code |
| **Cognee REST API**   | `http://localhost:8000`        | REST API for HTTP clients                  |
| **API Documentation** | `http://localhost:8000/docs`   | OpenAPI/Swagger UI                         |
| **Health Check**      | `http://localhost:8000/health` | Service health status                      |
| Neo4j Browser         | `http://localhost:7474`        | Graph database UI                          |
| Neo4j Bolt            | `bolt://localhost:7687`        | Graph database protocol                    |
| PostgreSQL            | `localhost:5432`               | Relational database + pgvector             |

## Credentials

**Neo4j:**

- Username: `neo4j`
- Password: `cognee_neo4j_password`

**PostgreSQL:**

- Database: `cognee_db`
- Username: `cognee`
- Password: `cognee_password`

## Management Commands

```bash
# View MCP server logs
docker compose logs -f cognee-mcp

# View REST API server logs
docker compose logs -f cognee-api

# View all logs
docker compose logs -f

# Stop services
docker compose down

# Restart specific service
docker compose restart cognee-mcp
docker compose restart cognee-api

# Check service health
docker compose ps
```

## REST API Endpoints

The Cognee REST API provides comprehensive access to all Cognee functionality:

### Core Operations

- `POST /api/v1/add` - Add data (text, files, URLs)
- `POST /api/v1/cognify` - Process data into knowledge graph
- `POST /api/v1/search` - Search knowledge graph
- `DELETE /api/v1/delete` - Delete data
- `GET /api/v1/datasets` - List datasets
- `POST /api/v1/datasets` - Create dataset

### Code Analysis

- `POST /api/v1/code-pipeline/index` - Index code repository
- `POST /api/v1/code-pipeline/retrieve` - Retrieve code information

### Authentication (Optional)

- `POST /api/v1/auth/register` - Register user
- `POST /api/v1/auth/login` - Login
- `GET /api/v1/auth/me` - Get current user

**Full API Documentation:** <http://localhost:8000/docs>

## Data Persistence

All data is stored in Docker volumes:

- `postgres_data` - Relational data and vector embeddings (pgvector)
- `neo4j_data` - Knowledge graph
- `cognee_data` - Processed documents
- `cognee_system` - System metadata

## Reset Everything

To completely remove all data and start fresh:

```bash
docker compose down -v
```

This removes all volumes and data.

## Configuration

### LLM Configuration

Edit both `cognee-mcp` and `cognee-api` service environments in `docker-compose.yaml` with identical settings:

```yaml
# Required environment variables:
- LLM_API_KEY=your-api-key-here
- LLM_MODEL=gpt-4o # or gpt-4, claude-3-sonnet, etc.
- LLM_PROVIDER=openai # or anthropic, etc.
- EMBEDDING_PROVIDER=openai
- EMBEDDING_MODEL=text-embedding-3-small
- EMBEDDING_DIMENSIONS=1536 # 1536 for text-embedding-3-small, 3072 for 3-large
```

Both services must use the same LLM configuration to ensure consistent behavior across MCP and REST API interfaces.

## Troubleshooting

**Cognee server not starting:**

```bash
# Check MCP server logs
docker compose logs cognee-mcp

# Check REST API server logs
docker compose logs cognee-api
```

**Service shows as "unhealthy" but works:**

Some health checks may fail while the service is still functional. Verify by:

```bash
# Check MCP endpoint (exposed on port 4000)
curl http://localhost:4000/health

# Check REST API endpoint
curl http://localhost:8000/health
```

If the health endpoint returns a response, the service is working despite the health check status.

**Service won't start:**

```bash
# Check which service is failing
docker compose ps

# View specific service logs
docker compose logs cognee-mcp
docker compose logs cognee-api
docker compose logs postgres
docker compose logs neo4j
```

**Port already in use:**

If ports 4000 or 8000 are already in use:

```bash
# Check what's using the ports
lsof -i :4000
lsof -i :8000

# Stop conflicting services or edit docker-compose.yaml to use different ports
```

## Resource Usage

Approximate memory requirements:

- Cognee MCP Server: ~2-4GB
- Cognee REST API Server: ~2-4GB
- Neo4j: ~2-3GB
- PostgreSQL + pgvector: ~1GB

Total: ~7-12GB RAM recommended

## Network Access

To access from other machines on your LAN, ensure ports are accessible:

```bash
# Allow ports through firewall (example for ufw)
sudo ufw allow 4000/tcp  # Cognee MCP
sudo ufw allow 8000/tcp  # Cognee REST API
sudo ufw allow 7474/tcp  # Neo4j Browser (optional)
sudo ufw allow 5432/tcp  # PostgreSQL (optional)
```

Replace `localhost` with your server's LAN IP when connecting from other machines.

## Loading and Processing Patterns

The project includes shell scripts for loading pattern files into Cognee and processing them into knowledge graphs.

### Pattern Loading Workflow

#### Step 1: Load patterns into dataset

```bash
cd /path/to/team-agentic-setup
./scripts/load-patterns.sh
```

This script:

- Validates pattern metadata before loading
- Loads all `.md` files (except README.md) from `agent-patterns/`
- Adds files to the `patterns` dataset via `/api/v1/add` endpoint
- Writes dataset name to `memory-mcp-server/logs/datasets-loaded.txt`
- Logs to `memory-mcp-server/logs/load-patterns-TIMESTAMP.log`

**Environment variables:**

- `COGNEE_URL` - Cognee API base URL (default: `http://localhost:8000`)
- `PATTERNS_DIR` - Pattern files directory (default: `${PROJ_ROOT}/agent-patterns`)
- `DATASET_NAME` - Dataset name (default: `patterns`)

#### Step 2: Process into knowledge graph

Process specific datasets (from file):

```bash
cat memory-mcp-server/logs/datasets-loaded.txt | ./scripts/cognify-patterns.sh
```

Process specific dataset (via echo):

```bash
echo "patterns" | ./scripts/cognify-patterns.sh
```

Process ALL datasets (no stdin):

```bash
./scripts/cognify-patterns.sh
```

This script:

- Accepts dataset names via stdin (piped or redirected)
- If NO stdin data, cognifies ALL datasets
- Calls `/api/v1/cognify` endpoint to build knowledge graphs
- Logs to `memory-mcp-server/logs/cognify-patterns-TIMESTAMP.log`
- Processing runs asynchronously

**Important:** The `cognify-patterns.sh` script does NOT accept command-line arguments. Arguments like `./scripts/cognify-patterns.sh patterns` will show an error. Use stdin only.

**Environment variables:**

- `COGNEE_URL` - Cognee API base URL (default: `http://localhost:8000`)

#### Step 3: Monitor processing

Cognify operations run asynchronously. Monitor progress with:

```bash
docker compose logs -f cognee-api
```

### Complete Workflow Example

```bash
# Load patterns into Cognee dataset
./scripts/load-patterns.sh
# Output: Writes "patterns" to memory-mcp-server/logs/datasets-loaded.txt

# Process into knowledge graph (choose one):
cat memory-mcp-server/logs/datasets-loaded.txt | ./scripts/cognify-patterns.sh  # Process loaded datasets
./scripts/cognify-patterns.sh                                             # Process ALL datasets

# Monitor async processing
docker compose logs -f cognee-api
```

## CLI Tools

You can use the Cognee CLI directly within either container:

```bash
# Add data via CLI (using API container)
docker exec cognee_api python3 -m cognee add "Your text here"

# Search via CLI (using API container)
docker exec cognee_api python3 -m cognee search "search query" -t GRAPH_COMPLETION

# Process data (using API container)
docker exec cognee_api python3 -m cognee cognify

# View all CLI commands
docker exec cognee_api python3 -m cognee --help
```

**Note:** Use `cognee_api` for CLI operations since the API server provides the full REST API functionality.
