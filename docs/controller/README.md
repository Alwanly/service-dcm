# Controller Standalone Deployment

This directory contains the Docker Compose configuration for deploying the Controller service as a standalone instance.

## Quick Start

1. **Configure credentials** (IMPORTANT - do this first in production):

   Copy the example env and edit passwords:

   ```bash
   cp .env.controller .env.controller.local
   # Edit .env.controller.local to set strong passwords
   ```

2. **Start the Controller**:

   ```bash
   docker-compose -f docker-compose.controller.yml --env-file .env.controller up -d --build
   ```

3. **Verify health**:

   ```bash
   curl http://localhost:8080/health
   ```

4. **Test admin access** (replace with your credentials):

   ```bash
   curl -u "$ADMIN_USER:$ADMIN_PASSWORD" -X GET http://localhost:8080/config
   ```

## Architecture

Controller container exposes an HTTP API on port 8080 and stores its SQLite database in a Docker volume `sdm-controller-data` mounted at `/app/data`.

## Endpoints

- `GET /health` - Health check (unauthenticated)
- `POST /register` - Agent registration (requires agent auth)
- `GET /config` - Get configuration (requires agent auth)
- `POST /config` - Set configuration (requires admin auth)
- `GET /swagger/index.html` - API documentation

## Data Persistence

The Controller uses a SQLite database stored in a Docker volume:

- **Volume name**: `sdm-controller-data`
- **Mount point**: `/app/data`
- **Database file**: `/app/data/data.db`

### Backup Database

```bash
# Create backup
docker run --rm -v sdm-controller-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/controller-data-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .

# Restore backup
docker run --rm -v sdm-controller-data:/data -v $(pwd):/backup alpine \
  sh -c "cd /data && tar xzf /backup/controller-data-YYYYMMDD-HHMMSS.tar.gz"
```

## Resource Limits

Adjust resources in `docker-compose.controller.yml` if needed.

## Security

Change default passwords in `.env.controller` before production. Use a reverse proxy for TLS/HTTPS in production.

## Troubleshooting

Check container logs:

```bash
docker-compose -f docker-compose.controller.yml logs -f controller
```

## Clean Up

```bash
docker-compose -f docker-compose.controller.yml down
```
