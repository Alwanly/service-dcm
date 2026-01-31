# Service Distribute Management

Distributed configuration management system with controller, agent, and worker services.

## Quick Start with Docker

### Prerequisites
- Docker Engine 20.10+
- Docker Compose 2.0+

### Setup
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd service-distribute-management
   ```

2. Create environment file:
   ```bash
   cp .env .env.local
   # Or use the .env file created during setup
   ```

3. Edit `.env` and set secure passwords:
   ```bash
   # IMPORTANT: Change default passwords!
   ADMIN_PASSWORD=your-secure-password
   AGENT_PASSWORD=your-secure-agent-password
   REDIS_PASSWORD=your-secure-redis-password
   ```

4. Start all services:
   ```bash
   docker-compose up -d
   ```

5. Verify services are running:
   ```bash
   docker-compose ps
   ```

6. Test the controller API:
   ```bash
   curl -u admin:your-password http://localhost:8080/controller/config
   ```

For detailed Docker documentation, see [docs/DOCKER.md](docs/DOCKER.md).

## Services
- **Controller**: http://localhost:8080 - Central management service
- **Worker**: http://localhost:8082 - Configuration execution service
- **Agent**: Internal client connecting controller and worker
- **Redis**: http://localhost:6379 - Optional pub/sub for push notifications

## Development

### Build from Source
```bash
# Controller
go build ./cmd/controller

# Agent
go build ./cmd/agent

# Worker
go build ./cmd/worker
```

### Run Locally
```bash
# Set environment variables
export CONTROLLER_ADDR=:8080
export DATABASE_PATH=./data.db
export ADMIN_USER=admin
export ADMIN_PASSWORD=admin

# Start controller
./controller
```

## Documentation
- [Docker Deployment Guide](docs/DOCKER.md)

## API Documentation

This project includes interactive Swagger/OpenAPI documentation for both Controller and Worker APIs.

### Accessing Swagger UI

**Controller API Documentation:**
- URL: http://localhost:8080/swagger/index.html
- Requires authentication for testing endpoints
- Default credentials:
   - Agent endpoints: `agent` / `agentpass`
   - Admin endpoints: `admin` / `password`

**Worker API Documentation:**
- URL: http://localhost:8082/swagger/index.html
- No authentication required

### Regenerating API Documentation

After modifying any API handlers or DTOs, regenerate the Swagger documentation:

```bash
make swagger-generate
```

The generated files are located in:
- `docs/controller/` - Controller API documentation
- `docs/worker/` - Worker API documentation

## License

[Add your license here]
