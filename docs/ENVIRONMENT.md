# Environment Variables Reference

This document provides a comprehensive reference for all environment variables used across the Service Distribute Management system.

## Table of Contents
- [Controller Service](#controller-service)
- [Agent Service](#agent-service)
- [Worker Service](#worker-service)
- [Redis Configuration](#redis-configuration)
- [Common Configuration](#common-configuration)

---

## Controller Service

The Controller service is the central management hub for distributed configuration.

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CONTROLLER_ADDR` | HTTP server bind address and port | `:8080` | No |
| `DATABASE_PATH` | Path to SQLite database file | `./data/controller.db` | No |
| `LOG_FORMAT` | Logging format: `json` or `console` | `console` | No |
| `LOG_LEVEL` | Logging level: `debug`, `info`, `error` | `info` | No |

### Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ADMIN_USER` | Admin username for protected endpoints | `admin` | No |
| `ADMIN_PASSWORD` | Admin password for protected endpoints | `password` | Yes* |
| `AGENT_USER` | Agent username for registration | `agent` | No |
| `AGENT_PASSWORD` | Agent password for registration | `agentpass` | Yes* |

*Required in production. Change from defaults for security.

### Polling Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POLL_INTERVAL` | Default polling interval in seconds for agents | `5` | No |

### Redis Configuration (Optional)

See [Redis Configuration](#redis-configuration) section below.

### Example Configuration

```bash
# Production Controller
CONTROLLER_ADDR=:8080
DATABASE_PATH=/data/controller.db
LOG_FORMAT=json
LOG_LEVEL=info

# Security (Change these!)
ADMIN_PASSWORD=your-secure-admin-password
AGENT_PASSWORD=your-secure-agent-password

# Polling
POLL_INTERVAL=10

# Redis (Optional - for push notifications)
REDIS_ENABLED=true
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0
```

---

## Agent Service

The Agent service polls the Controller for configuration updates and forwards them to the Worker.

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AGENT_ADDR` | HTTP server bind address and port | `:8081` | No |
| `LOG_FORMAT` | Logging format: `json` or `console` | `console` | No |
| `LOG_LEVEL` | Logging level: `debug`, `info`, `error` | `info` | No |

### Service URLs

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CONTROLLER_URL` | Base URL of the Controller service | `http://localhost:8080` | Yes |
| `WORKER_URL` | Base URL of the Worker service | `http://localhost:8082` | Yes |

### Polling Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `POLL_INTERVAL` | Configuration polling interval in seconds | `5` | No |
| `FALLBACK_POLL_ENABLED` | Enable fallback polling when Redis unavailable | `true` | No |
| `FALLBACK_POLL_INTERVAL` | Fallback polling interval in seconds | `10` | No |

### HTTP Client Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REQUEST_TIMEOUT` | HTTP request timeout in seconds | `10` | No |

### Registration & Retry Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REGISTRATION_MAX_RETRIES` | Maximum registration retry attempts | `5` | No |
| `REGISTRATION_INITIAL_BACKOFF` | Initial backoff duration (e.g., `1s`, `500ms`) | `1s` | No |
| `REGISTRATION_MAX_BACKOFF` | Maximum backoff duration | `30s` | No |
| `REGISTRATION_BACKOFF_MULTIPLIER` | Backoff multiplier for exponential backoff | `2.0` | No |

### Heartbeat Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `HEARTBEAT_ENABLED` | Enable heartbeat to Controller | `true` | No |
| `HEARTBEAT_INTERVAL` | Heartbeat interval in seconds | `30` | No |

### Redis Configuration (Optional)

See [Redis Configuration](#redis-configuration) section below.

### Example Configuration

```bash
# Production Agent
AGENT_ADDR=:8081
LOG_FORMAT=json
LOG_LEVEL=info

# Service URLs
CONTROLLER_URL=http://controller:8080
WORKER_URL=http://worker:8082

# Polling
POLL_INTERVAL=10
FALLBACK_POLL_ENABLED=true
FALLBACK_POLL_INTERVAL=15

# HTTP
REQUEST_TIMEOUT=15

# Registration Retry
REGISTRATION_MAX_RETRIES=10
REGISTRATION_INITIAL_BACKOFF=2s
REGISTRATION_MAX_BACKOFF=60s
REGISTRATION_BACKOFF_MULTIPLIER=2.0

# Heartbeat
HEARTBEAT_ENABLED=true
HEARTBEAT_INTERVAL=60

# Redis (Optional - for push notifications)
REDIS_ENABLED=true
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0
```

---

## Worker Service

The Worker service receives configuration from the Agent and executes HTTP proxy requests.

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `WORKER_ADDR` | HTTP server bind address and port | `:8082` | No |
| `LOG_FORMAT` | Logging format: `json` or `console` | `console` | No |
| `LOG_LEVEL` | Logging level: `debug`, `info`, `error` | `info` | No |

### HTTP Client Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REQUEST_TIMEOUT` | HTTP request timeout in seconds | `10` | No |

### Example Configuration

```bash
# Production Worker
WORKER_ADDR=:8082
LOG_FORMAT=json
LOG_LEVEL=info
REQUEST_TIMEOUT=30
```

---

## Redis Configuration

Redis is optional and provides pub/sub functionality for push-based configuration updates. If not configured, the system falls back to polling-only mode.

### Common Redis Variables

These variables are used by both Controller and Agent services when Redis is enabled.

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_ENABLED` | Enable Redis pub/sub | `false` | No |
| `REDIS_HOST` | Redis server hostname | `localhost` | If enabled |
| `REDIS_PORT` | Redis server port | `6379` | No |
| `REDIS_PASSWORD` | Redis authentication password | `` | If auth enabled |
| `REDIS_DB` | Redis database number | `0` | No |

### Example Configuration

```bash
# Redis enabled
REDIS_ENABLED=true
REDIS_HOST=redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=your-secure-redis-password
REDIS_DB=0
```

```bash
# Redis disabled (polling only)
REDIS_ENABLED=false
```

---

## Common Configuration

### Logging

All services support these logging configuration options:

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `LOG_FORMAT` | Output format | `json`, `console` | `console` |
| `LOG_LEVEL` | Minimum log level | `debug`, `info`, `warn`, `error`, `fatal` | `info` |

**Recommendations:**
- Development: `LOG_FORMAT=console`, `LOG_LEVEL=debug`
- Production: `LOG_FORMAT=json`, `LOG_LEVEL=info`

---

## Docker Compose Example

Example `.env` file for Docker Compose deployment:

```bash
# Controller
CONTROLLER_ADDR=:8080
DATABASE_PATH=/data/controller.db
ADMIN_USER=admin
ADMIN_PASSWORD=secure-admin-pass-123
AGENT_USER=agent
AGENT_PASSWORD=secure-agent-pass-456
POLL_INTERVAL=10

# Agent
AGENT_ADDR=:8081
CONTROLLER_URL=http://controller:8080
WORKER_URL=http://worker:8082
HEARTBEAT_ENABLED=true
HEARTBEAT_INTERVAL=60
REGISTRATION_MAX_RETRIES=10

# Worker
WORKER_ADDR=:8082
REQUEST_TIMEOUT=30

# Redis (Optional)
REDIS_ENABLED=true
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=secure-redis-pass-789
REDIS_DB=0

# Logging (All Services)
LOG_FORMAT=json
LOG_LEVEL=info
```

---

## Configuration Validation

### Required Variables Check

Before starting services, ensure these critical variables are set:

**Controller:**
- `ADMIN_PASSWORD` (change from default!)
- `AGENT_PASSWORD` (change from default!)

**Agent:**
- `CONTROLLER_URL`
- `WORKER_URL`

**All Services (Production):**
- `LOG_FORMAT=json`

### Security Checklist

- [ ] All default passwords changed
- [ ] Redis password set (if using Redis)
- [ ] Firewall rules configured for service ports
- [ ] HTTPS/TLS configured for production (use reverse proxy)
- [ ] Database path has appropriate permissions
- [ ] Log files rotation configured

---

## Troubleshooting

### Agent Cannot Register

Check these variables:
- `CONTROLLER_URL` - Must be reachable from agent
- `AGENT_PASSWORD` - Must match Controller's `AGENT_PASSWORD`
- `REGISTRATION_MAX_RETRIES` - Increase if network is unreliable

### Configuration Not Updating

Check these variables:
- `POLL_INTERVAL` - Lower value for faster updates (higher load)
- `REDIS_ENABLED` - Ensure Redis is running if enabled
- `FALLBACK_POLL_ENABLED` - Should be `true` for reliability

### High CPU/Memory Usage

Adjust these variables:
- `POLL_INTERVAL` - Increase to reduce polling frequency
- `REQUEST_TIMEOUT` - Reduce to fail faster on network issues
- `LOG_LEVEL` - Set to `warn` or `error` to reduce logging overhead

---

## Additional Resources

- [Docker Deployment Guide](DOCKER.md)
- [Performance Tuning](PERFORMANCE.md)
- [Security Best Practices](SECURITY.md)
- [Deployment Examples](DEPLOYMENT.md)
