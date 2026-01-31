# Deployment Examples

This document provides comprehensive deployment architecture examples and configuration templates for the Service Distribute Management system.

## Table of Contents
- [Deployment Architectures](#deployment-architectures)
- [Single-Server All-in-One](#single-server-all-in-one)
- [Standalone Controller](#standalone-controller)
- [Distributed Agent-Worker Pairs](#distributed-agent-worker-pairs)
- [High Availability Setup](#high-availability-setup)
- [Cloud Deployments](#cloud-deployments)
- [Migration Strategies](#migration-strategies)

---

## Deployment Architectures

### Architecture Decision Matrix

| Architecture | Use Case | Complexity | Scalability | Cost |
|--------------|----------|------------|-------------|------|
| **All-in-One** | Development, Testing, Small deployments | Low | Limited | Low |
| **Standalone Controller** | Multiple remote locations | Medium | Good | Medium |
| **Distributed Agent-Worker** | Large-scale distributed system | High | Excellent | High |
| **High Availability** | Production, critical systems | High | Excellent | High |

---

## Single-Server All-in-One

Deploy all services (Controller, Agent, Worker, Redis) on a single machine.

### Use Cases
- Development and testing
- Small deployments (< 10 agents)
- Proof-of-concept
- Internal tools

### Architecture Diagram

```mermaid
graph TB
    subgraph "Single Server"
        Controller[Controller :8080]
        Redis[Redis :6379]
        Agent[Agent :8081]
        Worker[Worker :8082]
        
        Controller -.Redis Pub/Sub.-> Redis
        Agent -.Register & Poll.-> Controller
        Agent -.Subscribe.-> Redis
        Agent -->|Config Update| Worker
        Worker -->|Proxy Request| Target[Target URLs]
    end
    
    Admin[Admin User] -->|HTTPS/TLS| Controller
```

### Docker Compose Configuration

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  controller:
    image: service-distribute-management/controller:latest
    container_name: sdm-controller
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - CONTROLLER_ADDR=:8080
      - DATABASE_PATH=/data/controller.db
      - ADMIN_USER=admin
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
      - AGENT_USER=agent
      - AGENT_PASSWORD=${AGENT_PASSWORD}
      - POLL_INTERVAL=10
      - REDIS_ENABLED=true
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    volumes:
      - controller-data:/data
    networks:
      - sdm-network
    depends_on:
      - redis
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3

  redis:
    image: redis:7-alpine
    container_name: sdm-redis
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - sdm-network
    healthcheck:
      test: ["CMD", "redis-cli", "--pass", "${REDIS_PASSWORD}", "ping"]
      interval: 30s
      timeout: 3s
      retries: 3

  agent:
    image: service-distribute-management/agent:latest
    container_name: sdm-agent
    restart: unless-stopped
    environment:
      - AGENT_ADDR=:8081
      - CONTROLLER_URL=http://controller:8080
      - WORKER_URL=http://worker:8082
      - POLL_INTERVAL=10
      - REQUEST_TIMEOUT=10
      - HEARTBEAT_ENABLED=true
      - HEARTBEAT_INTERVAL=60
      - REDIS_ENABLED=true
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - FALLBACK_POLL_ENABLED=true
      - FALLBACK_POLL_INTERVAL=30
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    networks:
      - sdm-network
    depends_on:
      - controller
      - worker
      - redis

  worker:
    image: service-distribute-management/worker:latest
    container_name: sdm-worker
    restart: unless-stopped
    ports:
      - "8082:8082"
    environment:
      - WORKER_ADDR=:8082
      - REQUEST_TIMEOUT=30
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    networks:
      - sdm-network

networks:
  sdm-network:
    driver: bridge

volumes:
  controller-data:
  redis-data:
```

**.env:**
```bash
ADMIN_PASSWORD=change-me-in-production
AGENT_PASSWORD=change-me-in-production
REDIS_PASSWORD=change-me-in-production
```

### Deployment Steps

```bash
# 1. Clone repository
git clone <repository-url>
cd service-distribute-management

# 2. Create .env file
cat > .env << EOF
ADMIN_PASSWORD=$(openssl rand -base64 32)
AGENT_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
EOF

# 3. Start services
docker-compose up -d

# 4. Verify all services running
docker-compose ps

# 5. Test controller
curl -u admin:$(grep ADMIN_PASSWORD .env | cut -d= -f2) http://localhost:8080/health
```

---

## Standalone Controller

Deploy Controller as a central service with multiple remote Agent+Worker pairs connecting to it.

### Use Cases
- Multiple geographic locations
- Edge computing scenarios
- Centralized configuration management
- 10-1000 remote sites

### Architecture Diagram

```mermaid
graph TB
    subgraph "Central Server"
        Controller[Controller :8080]
        Redis[Redis :6379]
        DB[(SQLite DB)]
        
        Controller --> DB
        Controller -.Pub/Sub.-> Redis
    end
    
    subgraph "Location 1"
        Agent1[Agent]
        Worker1[Worker]
        Agent1 -->|Config| Worker1
    end
    
    subgraph "Location 2"
        Agent2[Agent]
        Worker2[Worker]
        Agent2 -->|Config| Worker2
    end
    
    subgraph "Location N"
        AgentN[Agent]
        WorkerN[Worker]
        AgentN -->|Config| WorkerN
    end
    
    Agent1 -.Register & Poll.-> Controller
    Agent2 -.Register & Poll.-> Controller
    AgentN -.Register & Poll.-> Controller
    
    Agent1 -.Subscribe.-> Redis
    Agent2 -.Subscribe.-> Redis
    AgentN -.Subscribe.-> Redis
    
    Admin[Admin] -->|HTTPS| Controller
```

### Central Controller Configuration

**docker-compose.controller.yml:**
```yaml
version: '3.8'

services:
  controller:
    image: service-distribute-management/controller:latest
    container_name: sdm-controller
    restart: unless-stopped
    ports:
      - "8080:8080"  # Exposed to public/VPN
    environment:
      - CONTROLLER_ADDR=:8080
      - DATABASE_PATH=/data/controller.db
      - ADMIN_USER=admin
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
      - AGENT_USER=agent
      - AGENT_PASSWORD=${AGENT_PASSWORD}
      - POLL_INTERVAL=15
      - REDIS_ENABLED=true
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    volumes:
      - /var/lib/sdm/controller:/data
    networks:
      - sdm-network
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

  redis:
    image: redis:7-alpine
    container_name: sdm-redis
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "6379:6379"  # Exposed to agents (use VPN/firewall)
    volumes:
      - /var/lib/sdm/redis:/data
    networks:
      - sdm-network
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M

networks:
  sdm-network:
    driver: bridge
```

### Remote Agent+Worker Configuration

**docker-compose.agent-worker.yml:**
```yaml
version: '3.8'

services:
  agent:
    image: service-distribute-management/agent:latest
    container_name: sdm-agent
    restart: unless-stopped
    environment:
      - AGENT_ADDR=:8081
      - CONTROLLER_URL=${CONTROLLER_URL}
      - WORKER_URL=http://worker:8082
      - POLL_INTERVAL=15
      - REQUEST_TIMEOUT=15
      - HEARTBEAT_ENABLED=true
      - HEARTBEAT_INTERVAL=120
      - REGISTRATION_MAX_RETRIES=10
      - REGISTRATION_INITIAL_BACKOFF=2s
      - REGISTRATION_MAX_BACKOFF=60s
      - REDIS_ENABLED=${REDIS_ENABLED:-false}
      - REDIS_HOST=${REDIS_HOST}
      - REDIS_PORT=${REDIS_PORT:-6379}
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - FALLBACK_POLL_ENABLED=true
      - FALLBACK_POLL_INTERVAL=30
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    networks:
      - sdm-network
    depends_on:
      - worker

  worker:
    image: service-distribute-management/worker:latest
    container_name: sdm-worker
    restart: unless-stopped
    ports:
      - "8082:8082"
    environment:
      - WORKER_ADDR=:8082
      - REQUEST_TIMEOUT=60
      - LOG_FORMAT=json
      - LOG_LEVEL=info
    networks:
      - sdm-network
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M

networks:
  sdm-network:
    driver: bridge
```

**Remote .env:**
```bash
# Central controller URL (change to your domain/IP)
CONTROLLER_URL=https://controller.example.com

# Redis (optional - if controller Redis is accessible)
REDIS_ENABLED=true
REDIS_HOST=controller.example.com
REDIS_PORT=6379
REDIS_PASSWORD=<same-as-controller>
```

### Deployment Steps

**Central Controller:**
```bash
# On central server
cd /opt/sdm
docker-compose -f docker-compose.controller.yml up -d

# Configure firewall (example: iptables)
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 6379 -s 10.0.0.0/8 -j ACCEPT
```

**Remote Locations:**
```bash
# On each remote server
cd /opt/sdm
export CONTROLLER_URL=https://controller.example.com
docker-compose -f docker-compose.agent-worker.yml up -d
```

---

## Distributed Agent-Worker Pairs

Deploy multiple Agent+Worker pairs across different locations, all connecting to a central Controller.

### Use Cases
- Large-scale distributed systems (100+ locations)
- Multi-region deployments
- Edge computing with centralized management
- CDN-like configuration distribution

### Architecture Diagram

```mermaid
graph TB
    subgraph "Central Control Plane"
        Controller[Controller]
        Redis[Redis]
        LB[Load Balancer]
        
        LB --> Controller
        Controller -.-> Redis
    end
    
    subgraph "Region: US-East"
        subgraph "Location 1"
            A1[Agent] --> W1[Worker]
        end
        subgraph "Location 2"
            A2[Agent] --> W2[Worker]
        end
    end
    
    subgraph "Region: EU-West"
        subgraph "Location 3"
            A3[Agent] --> W3[Worker]
        end
        subgraph "Location 4"
            A4[Agent] --> W4[Worker]
        end
    end
    
    subgraph "Region: APAC"
        subgraph "Location 5"
            A5[Agent] --> W5[Worker]
        end
    end
    
    A1 -.-> LB
    A2 -.-> LB
    A3 -.-> LB
    A4 -.-> LB
    A5 -.-> LB
    
    Admin[Admin Console] --> LB
```

### Configuration

Same as [Standalone Controller](#standalone-controller), but with:

1. **Load Balancer** in front of Controller (Nginx, HAProxy, AWS ALB)
2. **Geographic DNS** for routing agents to nearest Controller
3. **Regional Redis** instances (optional) for reduced latency

**Nginx Load Balancer:**
```nginx
upstream controller-backend {
    least_conn;
    server controller1:8080 max_fails=3 fail_timeout=30s;
    server controller2:8080 max_fails=3 fail_timeout=30s backup;
}

server {
    listen 443 ssl http2;
    server_name controller.example.com;

    ssl_certificate /etc/ssl/certs/controller.crt;
    ssl_certificate_key /etc/ssl/private/controller.key;

    location / {
        proxy_pass http://controller-backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 5s;
        proxy_send_timeout 10s;
        proxy_read_timeout 10s;
    }
}
```

---

## High Availability Setup

Production-grade deployment with redundancy and failover.

### Architecture Diagram

```mermaid
graph TB
    subgraph "Load Balancer Layer"
        LB1[Load Balancer 1]
        LB2[Load Balancer 2]
    end
    
    subgraph "Controller Cluster"
        C1[Controller 1]
        C2[Controller 2]
        C3[Controller 3]
    end
    
    subgraph "Data Layer"
        DB[(PostgreSQL Primary)]
        DB_Replica[(PostgreSQL Replica)]
        Redis_Master[Redis Master]
        Redis_Replica[Redis Replica]
    end
    
    subgraph "Agent-Worker Pairs"
        A1[Agent 1] --> W1[Worker 1]
        A2[Agent 2] --> W2[Worker 2]
        AN[Agent N] --> WN[Worker N]
    end
    
    LB1 --> C1
    LB1 --> C2
    LB1 --> C3
    LB2 --> C1
    LB2 --> C2
    LB2 --> C3
    
    C1 --> DB
    C2 --> DB
    C3 --> DB
    DB -.Replication.-> DB_Replica
    
    C1 -.-> Redis_Master
    C2 -.-> Redis_Master
    C3 -.-> Redis_Master
    Redis_Master -.Replication.-> Redis_Replica
    
    A1 -.-> LB1
    A2 -.-> LB1
    AN -.-> LB2
```

### Components

**1. PostgreSQL High Availability:**
```yaml
# Requires migration from SQLite to PostgreSQL
# Use Patroni, Stolon, or cloud-managed PostgreSQL
version: '3.8'
services:
  postgres-primary:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: sdm
      POSTGRES_USER: sdm
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_INITDB_ARGS: "-E UTF8 --locale=C"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    command: |
      postgres
      -c wal_level=replica
      -c max_wal_senders=3
      -c max_replication_slots=3
```

**2. Redis Sentinel (HA):**
```yaml
services:
  redis-master:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}

  redis-replica:
    image: redis:7-alpine
    command: redis-server --replicaof redis-master 6379 --masterauth ${REDIS_PASSWORD} --requirepass ${REDIS_PASSWORD}

  redis-sentinel:
    image: redis:7-alpine
    command: redis-sentinel /etc/redis/sentinel.conf
    configs:
      - source: sentinel-config
        target: /etc/redis/sentinel.conf
```

**3. Multiple Controller Instances:**
```yaml
services:
  controller-1:
    image: service-distribute-management/controller:latest
    environment:
      DATABASE_TYPE: postgres  # Future: PostgreSQL support
      DATABASE_URL: postgres://sdm:${DB_PASSWORD}@postgres-primary:5432/sdm
      REDIS_SENTINEL_ENABLED: true
      REDIS_SENTINEL_MASTER: mymaster
      REDIS_SENTINEL_ADDRS: sentinel1:26379,sentinel2:26379,sentinel3:26379

  controller-2:
    image: service-distribute-management/controller:latest
    # Same config as controller-1
```

### Deployment Considerations

**Load Balancing:**
- Use DNS round-robin or dedicated load balancer
- Health check endpoint: `GET /health`
- Session affinity: Not required (stateless)

**Database Migration:**
- Migrate from SQLite to PostgreSQL for multi-instance support
- Use connection pooling (pgBouncer)
- Enable read replicas for scalability

**Monitoring:**
- Prometheus + Grafana for metrics
- Loki or ELK for log aggregation
- Alerts for service failures

---

## Cloud Deployments

### AWS Deployment

**Architecture:**
```
┌─────────────────────────────────────┐
│  Route 53 (DNS)                     │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│  Application Load Balancer          │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│  ECS Fargate (Controller)           │
│  - Task Definition                  │
│  - Auto Scaling (2-10 instances)    │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│  RDS PostgreSQL (Multi-AZ)          │
│  ElastiCache Redis (Cluster Mode)   │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│  EC2 Instances (Agent+Worker pairs) │
│  - Auto Scaling Group               │
│  - Across multiple AZs              │
└─────────────────────────────────────┘
```

**ECS Task Definition (Controller):**
```json
{
  "family": "sdm-controller",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "controller",
      "image": "<ecr-repo>/controller:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "DATABASE_URL",
          "value": "postgres://..."
        },
        {
          "name": "REDIS_HOST",
          "value": "redis-cluster.abc123.ng.0001.use1.cache.amazonaws.com"
        }
      ],
      "secrets": [
        {
          "name": "ADMIN_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:sdm/admin-password"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/sdm-controller",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "controller"
        }
      }
    }
  ]
}
```

### Kubernetes Deployment

**Controller Deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sdm-controller
  namespace: sdm
spec:
  replicas: 3
  selector:
    matchLabels:
      app: controller
  template:
    metadata:
      labels:
        app: controller
    spec:
      containers:
      - name: controller
        image: service-distribute-management/controller:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONTROLLER_ADDR
          value: ":8080"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: controller-secrets
              key: database-url
        - name: ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: controller-secrets
              key: admin-password
        - name: REDIS_HOST
          value: "redis-master.sdm.svc.cluster.local"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: controller
  namespace: sdm
spec:
  type: LoadBalancer
  selector:
    app: controller
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
```

**Agent+Worker DaemonSet:**
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: sdm-agent-worker
  namespace: sdm
spec:
  selector:
    matchLabels:
      app: agent-worker
  template:
    metadata:
      labels:
        app: agent-worker
    spec:
      containers:
      - name: agent
        image: service-distribute-management/agent:latest
        env:
        - name: CONTROLLER_URL
          value: "http://controller.sdm.svc.cluster.local:8080"
        - name: WORKER_URL
          value: "http://localhost:8082"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
      - name: worker
        image: service-distribute-management/worker:latest
        ports:
        - containerPort: 8082
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
```

---

## Migration Strategies

### From All-in-One to Distributed

**Phase 1: Separate Services**
1. Split Docker Compose into separate files
2. Move Agent+Worker to separate host
3. Update `CONTROLLER_URL` in Agent

**Phase 2: Add Redundancy**
1. Deploy second Agent+Worker pair
2. Verify both agents register
3. Test configuration distribution to both

**Phase 3: Scale Controller**
1. Migrate SQLite to PostgreSQL
2. Deploy second Controller instance
3. Add load balancer
4. Enable Redis for push notifications

### From SQLite to PostgreSQL

**Preparation:**
```bash
# Export SQLite data
sqlite3 controller.db .dump > dump.sql

# Transform to PostgreSQL format
sed 's/AUTOINCREMENT/SERIAL/g' dump.sql > postgres-dump.sql

# Import to PostgreSQL
psql -U sdm -d sdm -f postgres-dump.sql
```

**Update Controller:**
```bash
# Change environment variables
DATABASE_TYPE=postgres
DATABASE_URL=postgres://sdm:password@postgres:5432/sdm
```

---

## Additional Resources

- [Docker Guide](DOCKER.md)
- [Environment Variables](ENVIRONMENT.md)
- [Performance Guide](PERFORMANCE.md)
- [Security Best Practices](SECURITY.md)
