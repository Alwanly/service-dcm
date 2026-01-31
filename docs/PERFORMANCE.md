# Performance Guide

This document provides guidance on performance optimization, scaling, and resource management for the Service Distribute Management system.

## Table of Contents
- [Performance Considerations](#performance-considerations)
- [Resource Limits](#resource-limits)
- [Scaling Strategies](#scaling-strategies)
- [Optimization Tips](#optimization-tips)
- [Monitoring & Metrics](#monitoring--metrics)
- [Benchmarking](#benchmarking)
- [Troubleshooting Performance Issues](#troubleshooting-performance-issues)

---

## Performance Considerations

### System Characteristics

The Service Distribute Management system has these performance characteristics:

**Controller:**
- SQLite database I/O for configuration storage
- Handles agent registration, heartbeats, and configuration requests
- Redis pub/sub for push notifications (optional)
- CPU-bound: Minimal, mostly I/O-bound
- Memory usage: ~50-100MB base + database cache

**Agent:**
- Polling-based configuration fetching
- HTTP client for Controller and Worker communication
- In-memory configuration caching with ETag validation
- CPU-bound: Minimal
- Memory usage: ~30-50MB base

**Worker:**
- HTTP proxy for target URL requests
- Idempotent configuration updates
- CPU-bound: Minimal
- Memory usage: ~30-50MB base + request buffering

### Bottlenecks

Common performance bottlenecks:

1. **SQLite Write Contention** (Controller)
   - Multiple agents registering simultaneously
   - Frequent heartbeat updates
   - Configuration version updates

2. **Polling Overhead** (Agent)
   - Too many agents with aggressive polling intervals
   - Network latency to Controller

3. **Redis Connection Pool** (Controller/Agent)
   - Limited connections for pub/sub
   - Network latency to Redis server

---

## Resource Limits

### Docker Resource Limits

Recommended resource limits for Docker deployment:

#### Controller

```yaml
services:
  controller:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
```

**Scaling Factors:**
- Add 128MB memory per 1000 agents
- Add 0.5 CPU per 500 concurrent requests
- SQLite database size: ~1KB per agent + configuration data

#### Agent

```yaml
services:
  agent:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
        reservations:
          cpus: '0.1'
          memory: 64M
```

**Scaling Factors:**
- Light workload: Defaults sufficient
- Memory increases with large configuration payloads

#### Worker

```yaml
services:
  worker:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
```

**Scaling Factors:**
- Add 0.5 CPU per 100 concurrent proxy requests
- Add 256MB memory per 50 concurrent requests
- Adjust based on target URL response sizes

#### Redis

```yaml
services:
  redis:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
        reservations:
          cpus: '0.1'
          memory: 64M
```

### Disk I/O

**Controller Database:**
- SQLite database file grows with agents and configuration history
- Typical size: 10KB-1MB for small deployments, 10-100MB for large deployments
- Use SSD for production deployments with >100 agents
- Consider WAL mode for better write performance (GORM default)

**Log Files:**
- JSON logging generates ~1KB per request
- Rotate logs daily or at 100MB size limit
- Use log aggregation (e.g., ELK, Grafana Loki) for production

---

## Scaling Strategies

### Horizontal Scaling

#### Agent + Worker Pairs

Agent and Worker services are designed to be deployed as distributed pairs:

```
┌──────────────┐
│  Controller  │ (Single instance)
└──────┬───────┘
       │
       ├─────────┬─────────┬─────────┐
       │         │         │         │
   ┌───▼───┐ ┌──▼────┐ ┌──▼────┐ ┌──▼────┐
   │ Agent │ │ Agent │ │ Agent │ │ Agent │
   └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
       │         │         │         │
   ┌───▼───┐ ┌──▼────┐ ┌──▼────┐ ┌──▼────┐
   │Worker │ │Worker │ │Worker │ │Worker │
   └───────┘ └───────┘ └───────┘ └───────┘
```

**Benefits:**
- Linear scaling with number of locations
- Each pair operates independently
- No inter-agent coordination required

**Limitations:**
- Each agent must register separately
- Controller must handle all agent connections

#### Controller Scaling

The Controller currently uses SQLite, which limits scalability:

**Current Limitations:**
- Single SQLite database (not clustered)
- Write operations are serialized
- Suitable for up to ~1000 agents with moderate activity

**Future Scaling Options:**
- Migrate to PostgreSQL/MySQL for horizontal scaling
- Read replicas for configuration queries
- Load balancer for multiple Controller instances
- Shared database cluster

### Vertical Scaling

For small-to-medium deployments, vertical scaling is often sufficient:

**Controller:**
- 2 CPU cores + 2GB RAM: ~500 agents
- 4 CPU cores + 4GB RAM: ~1000+ agents
- 8 CPU cores + 8GB RAM: ~5000+ agents (with database optimization)

**Redis:**
- 1 CPU core + 512MB RAM: ~1000 subscriptions
- 2 CPU cores + 1GB RAM: ~5000+ subscriptions

---

## Optimization Tips

### 1. Polling Interval Tuning

**Trade-offs:**
- Lower interval = Faster configuration propagation, higher load
- Higher interval = Lower load, slower propagation

**Recommendations:**
```bash
# Real-time requirements (< 10 seconds)
POLL_INTERVAL=5

# Normal operations (< 30 seconds)
POLL_INTERVAL=15

# Batch updates (< 5 minutes)
POLL_INTERVAL=60

# Low-priority (< 15 minutes)
POLL_INTERVAL=300
```

**Dynamic Adjustment:**
The Controller can adjust agent polling intervals via API:
```bash
curl -X PUT http://localhost:8080/agents/{id}/poll-interval \
  -u admin:password \
  -H "Content-Type: application/json" \
  -d '{"pollIntervalSeconds": 30}'
```

### 2. Enable Redis Pub/Sub

Redis push notifications significantly reduce polling load:

**Without Redis (Polling Only):**
- 100 agents × 5-second interval = 20 requests/second
- Network overhead: ~2KB per request
- Controller CPU: ~10-20% (1 core)

**With Redis (Hybrid Push/Pull):**
- Configuration push: Instant delivery to all agents
- Fallback polling: 10-60 second interval (safety net)
- Controller CPU: ~2-5% (1 core)
- Network overhead: Reduced by 80-90%

**Configuration:**
```bash
# Enable Redis for both Controller and Agent
REDIS_ENABLED=true
REDIS_HOST=redis
FALLBACK_POLL_ENABLED=true
FALLBACK_POLL_INTERVAL=60  # Safety net
```

### 3. Database Optimization

**SQLite Tuning (Controller):**

SQLite uses WAL mode by default (via GORM), which provides:
- Concurrent reads during writes
- Better write performance
- Crash recovery

**Additional optimizations:**
- Place database on SSD storage
- Regular `VACUUM` to reclaim space (automatic in GORM)
- Monitor database size and archive old data

**Future Consideration:**
For deployments exceeding 1000 agents, consider migrating to PostgreSQL:
- Connection pooling
- Better concurrency
- Replication support
- Advanced indexing

### 4. Request Timeout Tuning

**Agent Configuration:**
```bash
# Conservative (slower failure detection)
REQUEST_TIMEOUT=30

# Balanced
REQUEST_TIMEOUT=10

# Aggressive (fast failure, may retry unnecessarily)
REQUEST_TIMEOUT=5
```

**Impact:**
- Lower timeout = Faster error detection, more retries
- Higher timeout = Fewer retries, blocked on slow networks

### 5. Heartbeat Optimization

Heartbeat updates generate database writes. Tune based on requirements:

```bash
# High availability (detect failures in 1 minute)
HEARTBEAT_INTERVAL=30

# Normal monitoring (detect failures in 5 minutes)
HEARTBEAT_INTERVAL=120

# Low overhead (detect failures in 15 minutes)
HEARTBEAT_INTERVAL=300
```

### 6. Logging Optimization

**Production Settings:**
```bash
# Reduce logging overhead
LOG_LEVEL=info  # or 'warn' for high-traffic
LOG_FORMAT=json  # Structured, easier to parse

# Development Settings
LOG_LEVEL=debug
LOG_FORMAT=console  # Human-readable
```

**Log Volume:**
- `debug`: ~10KB per request
- `info`: ~1KB per request
- `warn`: ~100 bytes per request (errors only)

### 7. HTTP Connection Pooling

Go's `http.Client` automatically pools connections. Default settings are generally optimal, but can be tuned:

**Agent/Worker HTTP Client:**
```go
// Custom transport settings (if needed)
MaxIdleConns: 100
MaxIdleConnsPerHost: 10
IdleConnTimeout: 90 * time.Second
```

Current default settings in codebase are appropriate for most use cases.

---

## Monitoring & Metrics

### Health Endpoints

All services expose `/health` endpoints:

```bash
# Controller
curl http://localhost:8080/health

# Agent
curl http://localhost:8081/health

# Worker
curl http://localhost:8082/health
```

### Key Metrics to Monitor

**Controller:**
- Active agents count (`GET /agents`)
- Database file size
- Request latency (from logs)
- SQLite lock contention (from logs)
- Redis connection status

**Agent:**
- Registration status
- Last successful configuration fetch
- Configuration version
- Heartbeat status
- Retry counts

**Worker:**
- Configuration version
- Proxy request success/failure rate
- Target URL response times

### Structured Logging

Use `LOG_FORMAT=json` for production and ingest logs into monitoring systems:

**Example Log Entry:**
```json
{
  "level": "info",
  "ts": "2026-01-31T12:34:56.789Z",
  "caller": "handler/handler.go:123",
  "msg": "Configuration updated",
  "agent_id": "agent-001",
  "config_version": "v1.2.3",
  "duration_ms": 45
}
```

**Recommended Monitoring Stack:**
- **Logs:** Grafana Loki, ELK Stack, or CloudWatch
- **Metrics:** Prometheus + Grafana
- **Tracing:** Jaeger (for future enhancement)

### Alerts

Configure alerts for:
- Agent registration failures (> 5 consecutive)
- Agent heartbeat missed (> 2 intervals)
- Controller database size (> 80% disk)
- High request latency (> 1 second)
- Redis connection loss (if enabled)

---

## Benchmarking

### Load Testing

Use tools like `ab` (Apache Bench), `wrk`, or `k6` to benchmark:

**Controller Configuration Endpoint:**
```bash
# 100 concurrent requests, 1000 total
ab -n 1000 -c 100 -A admin:password \
  http://localhost:8080/controller/config
```

**Expected Results (baseline):**
- Requests per second: 500-1000 (depends on hardware)
- Mean latency: 10-50ms (SQLite read)
- 95th percentile: < 100ms

**Worker Proxy Endpoint:**
```bash
# Configure worker first, then test /hit endpoint
ab -n 1000 -c 50 -p hit.json -T application/json \
  http://localhost:8082/hit
```

**Expected Results:**
- Requests per second: Depends on target URL performance
- Mean latency: Target latency + 5-10ms overhead

### Agent Polling Load

Simulate multiple agents:

```bash
# Simulate 100 agents polling every 5 seconds for 60 seconds
# (Requires custom script or load testing tool)
```

**Expected Controller Load:**
- CPU: 5-15% (1 core) for 100 agents
- Memory: ~150-200MB
- Network: ~40KB/s incoming requests

---

## Troubleshooting Performance Issues

### High CPU Usage

**Controller:**
1. Check number of active agents: `curl -u admin:pass http://localhost:8080/agents`
2. Increase `POLL_INTERVAL` for all agents
3. Enable Redis to reduce polling load
4. Check for slow database queries in logs

**Agent/Worker:**
1. Increase `REQUEST_TIMEOUT` to avoid retry loops
2. Check network latency to Controller/Worker
3. Review logs for error patterns

### High Memory Usage

**Controller:**
1. Check database size: `ls -lh data/controller.db`
2. Archive or delete old agent records
3. Increase Docker memory limits if constrained

**Agent/Worker:**
1. Check for memory leaks in logs (unlikely with Go)
2. Review configuration payload size
3. Increase Docker memory limits if needed

### Database Lock Contention

**Symptoms:**
- Slow write operations
- Log entries with "database is locked" errors

**Solutions:**
1. Reduce heartbeat frequency (`HEARTBEAT_INTERVAL`)
2. Increase `POLL_INTERVAL` to reduce read load
3. Consider PostgreSQL migration for >1000 agents

### Slow Configuration Propagation

**Diagnostics:**
1. Check agent poll interval
2. Verify Redis pub/sub is working (if enabled)
3. Check network latency between services

**Solutions:**
1. Lower `POLL_INTERVAL` (increases load)
2. Enable Redis for push notifications
3. Verify `FALLBACK_POLL_ENABLED=true`

### Redis Connection Issues

**Symptoms:**
- Agents falling back to polling
- Redis connection errors in logs

**Solutions:**
1. Verify Redis is running: `redis-cli ping`
2. Check `REDIS_PASSWORD` matches
3. Ensure `FALLBACK_POLL_ENABLED=true` for resilience
4. Review Redis connection pool settings

---

## Performance Tuning Checklist

- [ ] **Polling intervals** tuned based on requirements
- [ ] **Redis enabled** for push notifications (recommended)
- [ ] **Heartbeat interval** set appropriately (60-300s)
- [ ] **Request timeouts** configured for network conditions
- [ ] **Docker resource limits** set based on load
- [ ] **Database on SSD** for production (Controller)
- [ ] **Log level** set to `info` or `warn` in production
- [ ] **Log rotation** configured
- [ ] **Monitoring** in place (health checks, logs, metrics)
- [ ] **Alerts** configured for critical issues
- [ ] **Load testing** performed before production
- [ ] **Backup strategy** for Controller database

---

## Additional Resources

- [Environment Variables](ENVIRONMENT.md)
- [Security Best Practices](SECURITY.md)
- [Deployment Examples](DEPLOYMENT.md)
- [Docker Guide](DOCKER.md)
