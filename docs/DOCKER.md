# Docker Deployment Guide

> **Quick Links**
> - [Local Development](#local-development-all-in-one) - All services on one machine
> - [Standalone Controller](#standalone-controller-deployment) - Deploy Controller separately
> - [Distributed Agent+Worker](#distributed-agent--worker-deployment) - Deploy Agent+Worker pairs

## Deployment Scenarios

This project supports three deployment configurations:

### 1. Local Development (All-in-One)

All services running together on a single machine for development and testing.

Use when: Developing locally, running tests, quick demonstrations

File: `docker-compose.yml`

```bash
docker-compose up -d
```

### 2. Standalone Controller Deployment

Controller service with database, deployed independently from Agent+Worker pairs.

Use when: Setting up a central configuration management server

File: `docker-compose.controller.yml`

```bash
docker-compose -f docker-compose.controller.yml --env-file .env.controller up -d
```

### 3. Distributed Agent + Worker Deployment

Agent and Worker services deployed together, connecting to an external Controller.

Use when: Deploying distributed pairs across multiple locations, all managed by a central Controller

File: `docker-compose.agent-worker.yml`

```bash
docker-compose -f docker-compose.agent-worker.yml --env-file .env.agent-worker up -d
```

---

## Local Development (All-in-One)

### Quick Start

1. **Clone the repository and navigate to project root**

2. **Create environment file:**
  ```bash
  cp .env.example .env
  ```

3. **Edit `.env` and set secure passwords:**
  ```bash
  # Change these values!
  ADMIN_PASSWORD=your-secure-admin-password
  AGENT_PASSWORD=your-secure-agent-password
  ```

4. **Start all services:**
  ```bash
  docker-compose up -d
  ```

5. **Verify services are running:**
  ```bash
  docker-compose ps
  ```

6. **Test the controller health endpoint:**
  ```bash
  curl http://localhost:8080/health
  ```

This section supplements the existing content below.

This guide explains how to build, run, and manage the Service Distribute Management system using Docker and Docker Compose.

## Table of Contents
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Building Images](#building-images)
- [Running with Docker Compose](#running-with-docker-compose)
- [Running Individual Containers](#running-individual-containers)
- [Volume Management](#volume-management)
- [Networking](#networking)
- [Health Checks](#health-checks)
- [Troubleshooting](#troubleshooting)
- [Production Deployment](#production-deployment)

## Quick Start

1. **Clone the repository and navigate to project root**

2. **Create environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Edit `.env` and set secure passwords:**
   ```bash
   # Change these values!
   ADMIN_PASSWORD=your-secure-admin-password
   AGENT_PASSWORD=your-secure-agent-password
   ```

4. **Start all services:**
   ```bash
   docker-compose up -d
   ```

5. **Verify services are running:**
   ```bash
   docker-compose ps
   ```

6. **Test the controller:**
   ```bash
   curl -u admin:your-admin-password http://localhost:8080/controller/config
   ```

## Architecture

The system consists of three services:

- **Controller** (Port 8080): Central management service with SQLite database
- **Worker** (Port 8082): Configuration execution service
- **Agent** (No exposed ports): Polling service that bridges controller and worker

```
┌──────────┐         ┌────────────┐         ┌────────┐
│  Agent   │────────▶│ Controller │────────▶│ SQLite │
│ (Client) │         │   :8080    │         │   DB   │
└──────────┘         └────────────┘         └────────┘
     │
     │
     ▼
┌──────────┐
│  Worker  │
│  :8082   │
└──────────┘
```

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- 100MB free disk space minimum
- Ports 8080 and 8082 available

## Configuration

All services are configured via environment variables defined in `.env` file:

### Controller Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `CONTROLLER_ADDR` | `:8080` | Controller listen address |
| `DATABASE_PATH` | `/app/data/data.db` | SQLite database file path |
| `POLL_INTERVAL` | `5` | Configuration poll interval (seconds) |
| `ADMIN_USER` | `admin` | Admin username |
| `ADMIN_PASSWORD` | `admin` | Admin password ⚠️ CHANGE IN PRODUCTION |
| `AGENT_USER` | `agent` | Agent username |
| `AGENT_PASSWORD` | `agent` | Agent password ⚠️ CHANGE IN PRODUCTION |

### Agent Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `CONTROLLER_URL` | `http://controller:8080` | Controller service URL |
| `WORKER_URL` | `http://worker:8082` | Worker service URL |
| `POLL_INTERVAL` | `5` | Poll interval (seconds) |
| `REQUEST_TIMEOUT` | `10` | HTTP request timeout (seconds) |
| `AGENT_USER` | `agent` | Username for controller authentication |
| `AGENT_PASSWORD` | `agent` | Password for controller authentication |

### Worker Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_ADDR` | `:8082` | Worker listen address |
| `REQUEST_TIMEOUT` | `10` | HTTP request timeout (seconds) |

## Building Images

### Build All Services
```bash
docker-compose build
```

### Build Individual Services
```bash
# Controller
docker build -f Dockerfile.controller -t service-controller:latest .

# Agent
docker build -f Dockerfile.agent -t service-agent:latest .

# Worker
docker build -f Dockerfile.worker -t service-worker:latest .
```

### Build Arguments
For custom Go versions or build optimizations:
```bash
docker build \
  --build-arg GO_VERSION=1.24.4 \
  -f Dockerfile.controller \
  -t service-controller:latest .
```

## Running with Docker Compose

### Start Services
```bash
# Start in background
docker-compose up -d

# Start with logs visible
docker-compose up

# Start specific service
docker-compose up -d controller
```

### Stop Services
```bash
# Stop all services
docker-compose down

# Stop and remove volumes (⚠️ deletes database)
docker-compose down -v

# Stop without removing containers
docker-compose stop
```

### View Logs
```bash
# All services
To manually inspect the database inside a running controller container:

# Specific service
docker-compose logs -f controller
docker-compose exec controller sh

# Use sqlite3 inside the container (may need to install sqlite in the container for debug)
sqlite3 /app/data/data.db

# View all tables
.tables

# View schema
.schema

# Query agents
SELECT * FROM agents;

# Exit sqlite3
.quit
# Last 100 lines
docker-compose logs --tail=100 controller
```

### Service Management
```bash
# Restart a service
docker-compose restart controller

# Rebuild and restart
docker-compose up -d --build controller

# Scale agent (if needed)
docker-compose up -d --scale agent=3
```

## Running Individual Containers

### Controller
```bash
docker run -d \
  --name sdm-controller \
  -p 8080:8080 \
  -v sdm-data:/app/data \
  -e ADMIN_PASSWORD=secure-password \
  -e AGENT_PASSWORD=secure-password \
  service-controller:latest
```

### Worker
```bash
docker run -d \
  --name sdm-worker \
  -p 8082:8082 \
  service-worker:latest
```

### Agent
```bash
docker run -d \
  --name sdm-agent \
  -e CONTROLLER_URL=http://controller:8080 \
  -e WORKER_URL=http://worker:8082 \
  -e AGENT_PASSWORD=secure-password \
  --link sdm-controller:controller \
  --link sdm-worker:worker \
  service-agent:latest
```

## Volume Management

### Controller Data Volume
The controller uses a persistent volume for SQLite database:

```bash
# Inspect volume
docker volume inspect sdm-controller-data

# Backup database
docker run --rm \
  -v sdm-controller-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/db-backup.tar.gz -C /data .

# Restore database
docker run --rm \
  -v sdm-controller-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/db-backup.tar.gz -C /data

# Remove volume (⚠️ deletes all data)
docker volume rm sdm-controller-data
```

### View Database
```bash
docker run --rm -it \
  -v sdm-controller-data:/app/data \
  alpine sh -c "apk add sqlite && sqlite3 /app/data/data.db"
```

## Networking

### Network Details
- **Network Name:** `sdm-network`
- **Driver:** bridge
- **Internal DNS:** Services accessible by container name

### Service DNS Names
- `controller` → Controller service
- `worker` → Worker service
- `agent` → Agent service

### Testing Network Connectivity
```bash
# From agent to controller
docker-compose exec agent wget -O- http://controller:8080/controller/config

# From agent to worker
docker-compose exec agent wget -O- http://worker:8082/health
```

### Custom Network
To use a custom network:
```yaml
networks:
  sdm-network:
    external: true
    name: my-custom-network
```

## Health Checks

### Controller Health Check
```bash
# From host
curl http://localhost:8080/controller/register

# From another container
docker-compose exec agent wget -O- http://controller:8080/controller/register
```

### Worker Health Check
```bash
# From host
curl http://localhost:8082/health

# From another container
docker-compose exec agent wget -O- http://worker:8082/health
```

### Check Health Status
```bash
# View health status
docker-compose ps


## API Documentation (Swagger UI)

Both services include interactive Swagger documentation for easy testing and exploration.

**Access Swagger UI:**
- Controller API: http://localhost:8080/swagger/index.html
- Worker API: http://localhost:8082/swagger/index.html

**Testing via Swagger UI:**

1. Navigate to the Swagger UI URL in your browser
2. For Controller endpoints, click "Authorize" and enter credentials:
  - Agent auth: `agent` / `agentpass`
  - Admin auth: `admin` / `password`
3. Expand any endpoint and click "Try it out"
4. Execute the request and view the response

The Swagger UI provides complete API schemas, example requests, and interactive testing capabilities.

# Watch health checks
watch -n 1 docker-compose ps
```

## Troubleshooting

### Common Issues

#### 1. Container Fails to Start
```bash
# Check logs
docker-compose logs controller

# Common causes:
# - Port already in use → Change port in .env
# - Permission denied → Check volume permissions
# - Database locked → Stop conflicting services
```

#### 2. Agent Cannot Connect to Controller
```bash
# Check if controller is healthy
docker-compose ps controller

# Check network connectivity
docker-compose exec agent ping controller

# Verify credentials in .env match
```

#### 3. Database Initialization Fails
```bash
# Check controller logs
docker-compose logs controller | grep -i database

# Manually initialize
docker-compose exec controller sh -c "sqlite3 /app/data/data.db < /app/scripts/init-db.sql"

# Check volume permissions
docker-compose exec controller ls -la /app/data
```

#### 4. Build Failures
```bash
# Clear build cache
docker-compose build --no-cache

# Check Go version compatibility
docker run golang:1.24.4-alpine go version

# Verify go.mod and go.sum
docker-compose run --rm controller go mod verify
```

### Debugging

#### Interactive Shell Access
```bash
# Controller
docker-compose exec controller sh

# Agent
docker-compose exec agent sh

# Worker
docker-compose exec worker sh
```

#### Check Environment Variables
```bash
docker-compose exec controller env | grep -E 'CONTROLLER|DATABASE|ADMIN|AGENT'
docker-compose exec agent env | grep -E 'CONTROLLER|WORKER|AGENT'
docker-compose exec worker env | grep WORKER
```

#### Monitor Resource Usage
```bash
# All services
docker stats

# Specific service
docker stats sdm-controller
```

## Production Deployment

### Security Checklist
- [ ] Change all default passwords in `.env`
- [ ] Use strong passwords (16+ characters, mixed case, special chars)
- [ ] Consider using Docker secrets for sensitive data
- [ ] Enable HTTPS/TLS with reverse proxy (nginx, traefik)
- [ ] Limit container resources (CPU, memory)
- [ ] Use read-only root filesystem where possible
- [ ] Scan images for vulnerabilities
- [ ] Keep base images updated
- [ ] Enable Docker Content Trust
- [ ] Configure proper logging (syslog, JSON file)

### Resource Limits
Add to `docker-compose.yml`:
```yaml
services:
  controller:
    deploy:
      resources:
```

### Using Docker Secrets (Swarm)
```yaml
services:
  controller:
    secrets:
      - admin_password
      - agent_password
    environment:
      - ADMIN_PASSWORD_FILE=/run/secrets/admin_password
      - AGENT_PASSWORD_FILE=/run/secrets/agent_password

secrets:
  admin_password:
    external: true
  agent_password:
    external: true
```

### Reverse Proxy Example (nginx)
```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/ssl/certs/cert.pem;
    ssl_certificate_key /etc/ssl/private/key.pem;

    location /controller/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $real_addr;
    }

    location /worker/ {
        proxy_pass http://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $real_addr;
    }
}
```

### Monitoring
Consider integrating:
- **Prometheus** for metrics collection
- **Grafana** for visualization
- **cAdvisor** for container metrics
- **Loki** for log aggregation

### Backup Strategy
```bash
# Automated backup script
#!/bin/bash
BACKUP_DIR=/backups
DATE=$(date +%Y%m%d_%H%M%S)

docker run --rm \
  -v sdm-controller-data:/data:ro \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/sdm-backup-$DATE.tar.gz -C /data .

# Keep only last 7 days
find $BACKUP_DIR -name "sdm-backup-*.tar.gz" -mtime +7 -delete
```

### High Availability
For production HA setup:
1. Use external database (PostgreSQL/MySQL instead of SQLite)
2. Run multiple controller instances behind load balancer
3. Use container orchestration (Kubernetes, Docker Swarm)
4. Implement health monitoring and auto-restart
5. Set up centralized logging

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/compose-file/)
- [Go Alpine Best Practices](https://chemidy.medium.com/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324)
- [SQLite Docker Guide](https://github.com/docker-library/docs/tree/master/sqlite)
