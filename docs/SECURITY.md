# Security Best Practices

This document outlines security best practices, authentication mechanisms, and credential management for the Service Distribute Management system.

## Table of Contents
- [Security Overview](#security-overview)
- [Authentication Mechanisms](#authentication-mechanisms)
- [Credential Management](#credential-management)
- [Network Security](#network-security)
- [Data Security](#data-security)
- [Token Management](#token-management)
- [Deployment Security](#deployment-security)
- [Security Checklist](#security-checklist)

---

## Security Overview

The Service Distribute Management system implements multiple layers of security:

1. **Authentication**: Basic Auth + Token-based authentication
2. **Authorization**: Role-based access (Admin vs Agent)
3. **Credential Protection**: Environment-based secrets, no hardcoded credentials
4. **Network Isolation**: Docker networking, internal-only services
5. **Data Protection**: SQLite database with filesystem permissions
6. **Token Rotation**: API endpoint for rotating agent tokens

### Security Architecture

```
┌─────────────────────────────────────────────────┐
│  External Access (requires authentication)      │
│  ├─ Admin API (Basic Auth: ADMIN_USER/PASS)    │
│  └─ Agent Registration (Basic: AGENT_USER/PASS)│
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  Controller (Port 8080)                         │
│  ├─ Agent Token Validation (Bearer tokens)     │
│  ├─ SQLite Database (agents, configs, tokens)  │
│  └─ Optional Redis Pub/Sub                     │
└─────────────────┬───────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────┐
│  Internal Network (Docker/Private)              │
│  ├─ Agent (Port 8081 - internal only)          │
│  └─ Worker (Port 8082 - internal only)         │
└─────────────────────────────────────────────────┘
```

---

## Authentication Mechanisms

### 1. Basic Authentication

Used for initial agent registration and admin operations.

#### Admin Authentication

**Endpoints:**
- `PUT /controller/config` - Update configuration
- `GET /agents` - List all agents
- `GET /agents/:id` - Get agent details
- `PUT /agents/:id/poll-interval` - Update poll interval
- `POST /agents/:id/token/rotate` - Rotate agent token
- `DELETE /agents/:id` - Delete agent

**Credentials:**
```bash
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-admin-password
```

**HTTP Header:**
```
Authorization: Basic YWRtaW46eW91ci1zZWN1cmUtYWRtaW4tcGFzc3dvcmQ=
```

#### Agent Registration Authentication

**Endpoint:**
- `POST /register` - Register new agent

**Credentials:**
```bash
AGENT_USER=agent
AGENT_PASSWORD=your-secure-agent-password
```

**HTTP Header:**
```
Authorization: Basic YWdlbnQ6eW91ci1zZWN1cmUtYWdlbnQtcGFzc3dvcmQ=
```

### 2. Token-Based Authentication

After registration, agents use bearer tokens for all subsequent requests.

**Endpoints (Token Auth):**
- `GET /controller/config` - Fetch configuration
- `POST /heartbeat` - Send heartbeat

**HTTP Header:**
```
Authorization: Bearer <agent-token>
```

**Token Generation:**
- Automatically generated during agent registration
- Stored in `agent_configs` table (hashed)
- Never exposed in JSON responses
- Can be rotated via admin API

### 3. No Authentication (Internal Services)

Worker service endpoints are internal-only and do not require authentication:
- `POST /config` - Receive configuration from Agent
- `POST /hit` - Proxy HTTP request
- `GET /health` - Health check

**Security:** These endpoints should NOT be exposed publicly. Use Docker networking or firewall rules to restrict access.

---

## Credential Management

### Environment Variables

**CRITICAL:** Never hardcode credentials in source code or configuration files.

#### Production Deployment

```bash
# .env (DO NOT commit to version control)
ADMIN_PASSWORD=$(openssl rand -base64 32)
AGENT_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
```

**Add to `.gitignore`:**
```
.env
.env.local
.env.production
*.db
```

#### Docker Compose Secrets

For Docker Swarm, use secrets instead of environment variables:

```yaml
services:
  controller:
    secrets:
      - admin_password
      - agent_password
    environment:
      ADMIN_PASSWORD_FILE: /run/secrets/admin_password
      AGENT_PASSWORD_FILE: /run/secrets/agent_password

secrets:
  admin_password:
    external: true
  agent_password:
    external: true
```

#### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: controller-credentials
type: Opaque
data:
  admin-password: <base64-encoded>
  agent-password: <base64-encoded>
```

```yaml
env:
  - name: ADMIN_PASSWORD
    valueFrom:
      secretKeyRef:
        name: controller-credentials
        key: admin-password
```

### Default Password Security

**IMPORTANT:** The default passwords (`admin`, `password`, `agentpass`) are for development only.

**Pre-deployment checklist:**
- [ ] `ADMIN_PASSWORD` changed from default
- [ ] `AGENT_PASSWORD` changed from default
- [ ] `REDIS_PASSWORD` set (if using Redis)
- [ ] All passwords are >= 16 characters
- [ ] Passwords stored in secure secret management (Vault, AWS Secrets Manager, etc.)

### Password Requirements

**Recommendations:**
- Minimum length: 16 characters
- Use random generation: `openssl rand -base64 32`
- Rotate credentials every 90 days
- Use different passwords for each environment (dev/staging/prod)

---

## Network Security

### Docker Networking

**Best Practices:**

1. **Create isolated networks:**
```yaml
networks:
  sdm-network:
    driver: bridge
    internal: false  # Controller needs external access
  
  sdm-internal:
    driver: bridge
    internal: true  # Agent-Worker communication only
```

2. **Expose only Controller:**
```yaml
services:
  controller:
    ports:
      - "8080:8080"  # External access
    networks:
      - sdm-network

  agent:
    # No ports exposed externally
    networks:
      - sdm-network
      - sdm-internal

  worker:
    # No ports exposed externally
    networks:
      - sdm-internal
```

### Firewall Rules

**Controller (Port 8080):**
- Allow: Agent registration from known IPs
- Allow: Admin API from management network
- Deny: All other traffic

**Example (iptables):**
```bash
# Allow agent registration from specific network
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/24 -j ACCEPT

# Allow admin access from management network
iptables -A INPUT -p tcp --dport 8080 -s 192.168.1.0/24 -j ACCEPT

# Deny all other access to 8080
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

**Agent/Worker:**
- Should NOT be exposed to public internet
- Use internal Docker network or VPN

**Redis:**
- Should NOT be exposed to public internet
- Require password authentication (`REDIS_PASSWORD`)
- Use TLS for Redis connections (future enhancement)

### TLS/HTTPS

The current implementation uses HTTP. For production:

**Option 1: Reverse Proxy (Recommended)**

Use Nginx or Traefik in front of Controller:

```nginx
server {
    listen 443 ssl http2;
    server_name controller.example.com;

    ssl_certificate /etc/ssl/certs/controller.crt;
    ssl_certificate_key /etc/ssl/private/controller.key;

    location / {
        proxy_pass http://controller:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**Option 2: Built-in TLS (Future Enhancement)**

Add TLS support directly to Fiber server:

```go
app.ListenTLS(":8443", "./cert.pem", "./key.pem")
```

### VPN/Wireguard

For distributed Agent+Worker deployments, consider VPN:

- Agents connect to Controller via Wireguard VPN
- All traffic encrypted end-to-end
- No need to expose Controller to public internet

---

## Data Security

### Database Security

**SQLite (Controller):**

1. **File Permissions:**
```bash
chmod 600 /data/controller.db  # Owner read/write only
chown controller:controller /data/controller.db
```

2. **Encryption at Rest:**
SQLite doesn't support built-in encryption. Options:
- Use encrypted filesystem (LUKS, dm-crypt)
- Encrypted Docker volumes
- Migrate to PostgreSQL with pgcrypto

3. **Backup Security:**
```bash
# Encrypt backups
tar czf - controller.db | gpg --encrypt --recipient admin@example.com > backup.tar.gz.gpg
```

### Sensitive Data Storage

**Current Implementation:**
- Agent tokens stored in database (plain text - **RISK**)
- Configuration data stored as JSON (plain text)

**Recommendations:**
- Hash agent tokens before storing (bcrypt, argon2)
- Encrypt sensitive configuration fields (AES-256)
- Use database-level encryption for compliance

### Redis Security

**Configuration:**
```bash
# Require password
REDIS_PASSWORD=your-secure-password

# Disable dangerous commands (redis.conf)
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command CONFIG ""

# Bind to internal network only
bind 127.0.0.1 10.0.0.10
```

---

## Token Management

### Agent Token Lifecycle

1. **Registration:** Agent calls `/register` with Basic Auth
2. **Token Issuance:** Controller generates UUID token, stores in database
3. **Token Usage:** Agent uses token for `/controller/config` and `/heartbeat`
4. **Token Rotation:** Admin rotates token via `/agents/:id/token/rotate`
5. **Token Revocation:** Admin deletes agent via `/agents/:id`

### Token Rotation

**When to Rotate:**
- Every 90 days (recommended)
- When agent is compromised
- When employee with access leaves
- After security incident

**How to Rotate:**
```bash
# Rotate agent token
curl -X POST http://localhost:8080/agents/{agent-id}/token/rotate \
  -u admin:password \
  -H "Content-Type: application/json"

# Response includes new token
{
  "success": true,
  "data": {
    "agentId": "agent-001",
    "newToken": "new-uuid-token-here"
  }
}
```

**Agent Update:**
Agent must be reconfigured with new token (restart required).

### Token Security Best Practices

- [ ] Tokens stored securely (hashed in database - **TODO**)
- [ ] Tokens transmitted over HTTPS only
- [ ] Token rotation policy in place (90 days)
- [ ] Revoke tokens for deleted agents
- [ ] Monitor token usage (log authentication attempts)
- [ ] Use short-lived tokens (future enhancement)

---

## Deployment Security

### Docker Security

**1. Use Non-Root User:**

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder
...
FROM alpine:latest
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

**2. Read-Only Filesystem:**

```yaml
services:
  controller:
    read_only: true
    tmpfs:
      - /tmp
    volumes:
      - controller-data:/data  # Writable volume for database
```

**3. Drop Capabilities:**

```yaml
services:
  controller:
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Only if binding to port < 1024
```

**4. Security Scanning:**

```bash
# Scan Docker images for vulnerabilities
docker scan service-distribute-management/controller:latest
```

### Kubernetes Security

**1. Pod Security Policy:**

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  runAsUser:
    rule: MustRunAsNonRoot
  fsGroup:
    rule: RunAsAny
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'persistentVolumeClaim'
```

**2. Network Policies:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: controller-policy
spec:
  podSelector:
    matchLabels:
      app: controller
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
      - podSelector:
          matchLabels:
            app: agent
      ports:
        - protocol: TCP
          port: 8080
```

### Environment-Specific Security

**Development:**
- Default credentials OK
- HTTP acceptable
- Logging: DEBUG level
- No token rotation required

**Staging:**
- Unique credentials (different from prod)
- HTTPS recommended
- Logging: INFO level
- Test token rotation

**Production:**
- Strong credentials (32+ characters)
- HTTPS required (TLS 1.3)
- Logging: WARN level (or INFO with log aggregation)
- Token rotation every 90 days
- Database backups encrypted
- Firewall rules enforced
- Monitoring and alerting enabled

---

## Security Checklist

### Pre-Deployment

- [ ] All default passwords changed
- [ ] Strong passwords (16+ characters, random)
- [ ] Credentials stored in secret management system
- [ ] `.env` files in `.gitignore`
- [ ] TLS/HTTPS configured (reverse proxy)
- [ ] Firewall rules configured
- [ ] Docker/Kubernetes security policies applied
- [ ] Non-root user in containers
- [ ] Database file permissions set (600)
- [ ] Redis password protected
- [ ] Internal services not exposed publicly
- [ ] Log aggregation configured
- [ ] Security alerts configured

### Post-Deployment

- [ ] Monitor authentication failures
- [ ] Review access logs weekly
- [ ] Rotate tokens every 90 days
- [ ] Update dependencies monthly (security patches)
- [ ] Backup database daily (encrypted)
- [ ] Test disaster recovery quarterly
- [ ] Security audit annually
- [ ] Vulnerability scanning (Docker images, dependencies)

### Incident Response

**If agent token compromised:**
1. Rotate token immediately via admin API
2. Review access logs for suspicious activity
3. Update agent with new token
4. Investigate how token was compromised
5. Implement additional security controls

**If admin credentials compromised:**
1. Change admin password immediately
2. Rotate all agent tokens
3. Review all configuration changes
4. Check for unauthorized agent registrations
5. Audit database for modifications

**If database compromised:**
1. Take database offline
2. Restore from last known good backup
3. Rotate all credentials (admin, agent, tokens)
4. Re-register all agents with new tokens
5. Investigate breach vector
6. Implement additional security controls

---

## Compliance Considerations

### GDPR

If storing personal data in configurations:
- [ ] Document what data is collected
- [ ] Implement data retention policy
- [ ] Provide mechanism to delete agent data
- [ ] Encrypt sensitive configuration data

### SOC 2 / ISO 27001

- [ ] Credential rotation policy documented
- [ ] Access control matrix defined
- [ ] Audit logging enabled
- [ ] Backup and recovery tested
- [ ] Security incident response plan
- [ ] Regular security assessments

---

## Future Security Enhancements

**Recommended Improvements:**

1. **Token Hashing:** Store bcrypt/argon2 hashes instead of plain tokens
2. **Short-Lived Tokens:** JWT with expiration (15-60 minutes)
3. **Mutual TLS:** Agent-Controller authentication via certificates
4. **Audit Logging:** Dedicated audit log for all authentication events
5. **Rate Limiting:** Prevent brute-force attacks on authentication
6. **2FA/MFA:** Multi-factor authentication for admin API
7. **Database Encryption:** Encrypt sensitive fields in SQLite
8. **Redis TLS:** Encrypt Redis connections
9. **RBAC:** Fine-grained role-based access control
10. **Security Headers:** Add security headers to HTTP responses

---

## Additional Resources

- [Environment Variables](ENVIRONMENT.md)
- [Performance Guide](PERFORMANCE.md)
- [Deployment Examples](DEPLOYMENT.md)
- [Docker Guide](DOCKER.md)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
