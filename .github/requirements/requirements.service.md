# PLAN: Distributed Configuration Management

Date: 2026-01-26

## Objectives
- Establish a distributed configuration and job orchestration system to support web scraping.
- Implement three services in Go: Controller, Agent, and Worker.
- Use Redis pub/sub for event distribution and SQLite for simple persistence.

## Architecture Overview
- Controller Service: The Controller acts as the central hub for configuration management and agent registration.
- Agent Service: The Agent acts as a bridge between the Controller and the Worker, managing configuration synchronization.
- Worker Service: The Worker is an HTTP service that executes tasks based on dynamically received configurations.

## Dependencies
- Go 1.21
- Redis client: github.com/redis/go-redis/v9
- SQLite driver: modernc.org/sqlite (CGO-free)
- HTTP: fiber or fasthttp

## Components & Responsibilities
### Controller
Configuration management:
1. Accept configuration updates from administrators via API
2. Persist configurations in a database (simple SQLite is also acceptable)
3. Make configurations available for agents to poll
4. Track configuration versions to detect changes
5. Single configuration applies to all registered agents
Agent Registration:
1. Provide a protected registration endpoint (static credentials acceptable)
2. Authenticate incoming agent registration requests
3. Generate unique agent IDs for registered agents
After registration:
1. Generate Unique Agent ID: Create a unique identifier for the agent (use UUID)
2. Store Agent Information: Persist agent registration details in the database
3. Create Initial Configuration: Store default or latest configuration for the agent
4. Return Connection Details, respond with:
    - Agent ID (unique identifier)
    - Poll endpoint URL
    - Polling interval (in second unit)
### Agent
1. Controller-Poller:
  a. Maintains periodic connection to the Controller
  b. Polls for configuration updates at regular intervals
  c. Detects configuration changes
  d. Caches current configuration and its version
2. Worker-Manager:
  a. Forwards received configurations to the Worker via /config endpoint
  b. Only sends updates when configuration has changed
  c. Manages Worker lifecycle (optional. Worker lifecycle can be managed outside of agent)

### Worker
Configuration Management:
1. Start with empty/default configuration
2. Accept configuration updates via /config endpoint from the Agent
3. Store and apply the latest configuration
4. Log configuration changes to console

## API
### Controller
POST /register
1.Authentication: Required (static credentials for the agent to register)
2.Request: Agent identification information
3.Response example:
json
 {
    "agent_id": "uuid-here",
    "poll_url": "/config",
    "poll_interval_seconds": 30
  }

POST /config
1.Authentication: Required (for administrator)
2.Request: New configuration data
3.Response: Success confirmation
4.Action: Updates global configuration version, applied to all agents

GET /config
1.Authentication: Required (agent credentials, can use credential used by agent to register itself)
2.Response: Current worker configuration (same for all agents)
3.Headers: Include ETag or version for change detection
4.Behavior: Returns the latest global configuration

### Worker
POST /config
- Caller: Agent
- Request: Worker configuration (JSON)
- Response: Success confirmation
- Action: Logs new configuration to console

GET /hit
- Caller: External user
- Response: Result from configured URL
- Behavior: Executes GET request to configured URL and proxies response
