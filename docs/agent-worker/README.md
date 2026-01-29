# Agent + Worker Combined Deployment

This directory contains the Docker Compose configuration for deploying Agent and Worker services together as a distributed pair.

## Quick Start

1. **Configure Controller URL** (REQUIRED):

   Edit `.env.agent-worker` and set `CONTROLLER_URL` to your Controller's address (e.g. `http://localhost:8080`).

2. **Set authentication credentials** (must match Controller):

   Edit `AGENT_USER` / `AGENT_PASSWORD` in `.env.agent-worker` to match the Controller settings.

3. **Start Agent + Worker**:

```bash
docker-compose -f docker-compose.agent-worker.yml --env-file .env.agent-worker up -d --build
```

4. **Verify services are running**:

```bash
docker ps
curl http://localhost:8082/health
```

## Architecture

- Agent registers with the Controller and forwards configuration updates to Worker.
- Worker exposes `:8082` for receiving configurations and provides a `/health` endpoint.

## Networking

- Internal network: `sdm-agent-worker-network` (bridge).
- Worker reachable by Agent at `http://worker:8082` inside the network.

## Troubleshooting

- If Agent can't reach Controller, verify `CONTROLLER_URL` and network connectivity.
- Check logs:

```bash
docker-compose -f docker-compose.agent-worker.yml logs -f agent
docker-compose -f docker-compose.agent-worker.yml logs -f worker
```

## Clean Up

```bash
docker-compose -f docker-compose.agent-worker.yml down
```
