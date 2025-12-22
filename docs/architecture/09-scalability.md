# How We Scale

**Document:** Scalability  
**Version:** 1.0  
**Last Updated:** December 22, 2025

Let's talk about how the system grows from 10 users to 100,000.

## Horizontal vs Vertical Scaling

**Vertical:** Bigger machines (2 CPU → 4 CPU → 8 CPU)  
**Horizontal:** More machines (1 server → 2 servers → 10 servers)

We prefer horizontal. It's more cost-effective and has no ceiling.

## Component Scaling Profiles

Different parts scale differently:

**API Server (CPU-bound):**

- Bottleneck: Request processing, JSON parsing
- Resource: 1-2 CPU, 2-4GB RAM
- Scale: 3-10 replicas
- Trigger: CPU > 70%

**Cognee Service (Memory-bound):**

- Bottleneck: Pattern embeddings, graph queries
- Resource: 1-2 CPU, 4-8GB RAM
- Scale: 2-5 replicas
- Trigger: Memory > 75%

**Databases (Managed):**

- PostgreSQL: Vertical scaling + read replicas
- Neo4j: Vertical scaling + read replicas
- Redis: Cluster mode sharding

## Scaling Timeline

### Stage 1: MVP (0-100 users)

**Infrastructure:**

- API: 3 pods (1 CPU, 2GB each)
- Cognee: 2 pods (1 CPU, 4GB each)
- PostgreSQL: db.t3.medium
- Neo4j: Professional tier
- Redis: cache.t3.medium

**Cost:** ~$500/month  
**Capacity:** 100 concurrent users, 10K requests/day

### Stage 2: Early Adoption (100-1K users)

**Infrastructure:**

- API: 3-5 pods (auto-scale)
- Cognee: 2-3 pods
- PostgreSQL: db.t3.large + 1 read replica
- Neo4j: Professional tier
- Redis: cache.t3.medium

**Cost:** ~$1,500/month  
**Capacity:** 1K concurrent users, 100K requests/day

### Stage 3: Growth (1K-10K users)

**Infrastructure:**

- API: 5-10 pods
- Cognee: 3-5 pods
- PostgreSQL: db.r6g.xlarge + 2 read replicas
- Neo4j: Enterprise tier
- Redis: Cluster mode (3 nodes)

**Cost:** ~$5,000/month  
**Capacity:** 10K concurrent users, 1M requests/day

### Stage 4: Scale (10K-100K users)

**Infrastructure:**

- API: 10-20 pods
- Cognee: 5-10 pods
- PostgreSQL: db.r6g.2xlarge + 3 read replicas
- Neo4j: Enterprise tier (larger)
- Redis: Cluster mode (5 nodes)
- Consider multi-region

**Cost:** ~$20,000/month  
**Capacity:** 100K concurrent users, 10M requests/day

## Caching Strategy

Multi-level cache reduces database load:

**L1 (In-Memory):** Per-pod cache, 100MB, 1-min TTL  
**L2 (Redis):** Shared cache, 5-min TTL  
**L3 (Database):** Source of truth

**Cache hit rates:**

- User context: 90% (Redis)
- Pattern content: 70% (Redis)
- Agent definitions: 95% (in-memory)

## Database Scaling

### PostgreSQL Strategy

**Phase 1:** Single instance (vertical scaling)

```text
db.t3.medium → db.t3.large → db.t3.xlarge
```

**Phase 2:** Add read replicas (horizontal)

```text
Primary: All writes + critical reads
Replica 1: Analytics queries
Replica 2: Reporting queries
```

**Phase 3:** Partition large tables

```sql
-- Usage records partitioned by month
CREATE TABLE usage_2024_12 PARTITION OF usage
  FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');
```

Archive old partitions to S3 for cost savings.

### Neo4j Strategy

**Phase 1:** Single instance (vertical)  
**Phase 2:** Add read replicas for queries  
**Phase 3:** Causal cluster for HA (3 core + N read replicas)

### Redis Strategy

**Phase 1:** Single node  
**Phase 2:** Primary + 1 replica (HA)  
**Phase 3:** Cluster mode (sharding)

## Connection Pooling

Prevent database connection exhaustion:

**PostgreSQL pools:**

```text
API Server: 5-20 connections
Cognee: 10-30 connections
Total: 15-50 connections (well under RDS limit)
```

**Connection pool config:**

```yaml
min_idle: 5
max_open: 20
max_lifetime: 10m
```

## Load Distribution

### Round Robin (Default)

Traffic distributed evenly across pods:

```text
Request 1 → Pod 1
Request 2 → Pod 2
Request 3 → Pod 3
Request 4 → Pod 1  (repeat)
```

Good for stateless workloads.

### Session Affinity

Route same user to same pod:

```yaml
service:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    timeoutSeconds: 10800  # 3 hours
```

Good for cache warming but makes scaling harder.

## Bottleneck Identification

When performance degrades, check:

**High CPU:**

- More request processing than capacity
- Solution: Horizontal scale (add pods)

**High Memory:**

- Large caches, memory leaks
- Solution: Optimize or vertical scale

**High Latency:**

- Slow database queries, external API delays
- Solution: Add caching, optimize queries

**High Error Rate:**

- Downstream service failures, resource exhaustion
- Solution: Fix root cause, add circuit breakers

## Auto-Scaling Configuration

**API Server HPA:**

```yaml
minReplicas: 3
maxReplicas: 10
targetCPUUtilization: 70%
scaleUpStabilization: 60s   # Wait 1 min before scaling up
scaleDownStabilization: 300s  # Wait 5 min before scaling down
```

**Cognee Service HPA:**

```yaml
minReplicas: 2
maxReplicas: 5
targetMemoryUtilization: 75%
```

## Cost Optimization

**Right-size resources:**

- Monitor actual usage
- Adjust requests/limits based on data
- Don't over-provision

**Use reserved instances:**

- 1-year commitment: 40% savings
- For baseline capacity (not burst)

**Auto-scale down:**

- Reduce pods during low-traffic periods
- Nighttime, weekends (depending on usage patterns)

**Archive old data:**

- Move old usage records to S3 ($0.023/GB vs $0.115/GB in RDS)

## Regional Scaling

### Single Region (Current)

All infrastructure in one AWS region (us-east-1):

- Simple architecture
- Lower cost
- Good enough for <10K users

### Multi-Region (Future)

Deploy in multiple regions:

```text
US Region (primary):
  - Full deployment
  - Primary databases

EU Region (secondary):
  - Full deployment
  - Read replicas

GeoDNS routes users to nearest region
```

**Benefits:**

- Lower latency globally
- Higher availability
- Disaster recovery

**Challenges:**

- Data consistency across regions
- 2x infrastructure cost
- Deployment complexity

## Load Testing

Before scaling up, load test to find limits:

**Test scenario:**

```text
1. Baseline: 100 concurrent users
2. Ramp: +50 users every 5 minutes
3. Peak: Hold at 500 users for 15 minutes
4. Spike: Sudden 2x increase
5. Ramp down
```

**Measure:**

- Response times (p50, p95, p99)
- Error rate
- Resource usage (CPU, memory)
- Database performance

**Find breaking points:**

- At what load do errors spike?
- What resource exhausts first?
- How does system degrade?

## Key Takeaways

- **Horizontal over vertical** - Add machines, not bigger machines
- **Different components scale differently** - API is CPU-bound, Cognee is memory-bound
- **Cache aggressively** - Reduce database load with multi-level caching
- **Monitor and right-size** - Use actual data to guide scaling decisions
- **Cost optimize** - Reserved instances, auto-scale down, archive old data
- **Load test** - Know your limits before you hit them

Final doc: Trade-offs and alternatives we considered.

---

Copyright © 2025 Jeremy K. Johnson. All rights reserved.
