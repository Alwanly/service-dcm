# Podman Local Development

Prerequisites
- Podman (desktop or engine) installed
- PowerShell (Windows) or Bash (Linux/macOS)
- Built images (see `build-images.ps1`)

Quick start (Windows PowerShell)

1. Build images:

```powershell
.\.copilot\plans\podman-local-dev\scripts\build-images.ps1
```

2. Create pods and start services:

```powershell
.\.copilot\plans\podman-local-dev\scripts\Setup-Pods.ps1
```

3. Verify setup:

```powershell
.\.copilot\plans\podman-local-dev\scripts\Verify-Setup.ps1
```

Stopping and cleanup

```powershell
.\.copilot\plans\podman-local-dev\scripts\Remove-Pods.ps1
```

Troubleshooting
- Network "subnet already used": another host network is using the desired subnet. Either remove that network or edit `Setup-Pods.ps1` to use a different network name/subnet.
- Pod names vs container names: services inside pods share the pod network namespace; use the infra pod name (e.g. `controller-pod`) or `localhost` when addressing services from containers inside the same pod.
- Redis DNS/timeouts: if agents report `failed to connect to redis` or `i/o timeout`, ensure Redis is in the same pod as the controller and agents are configured to reach the controller via the pod name.
- Empty agent hostname: ensure `AGENT_HOSTNAME` is set in `.copilot/plans/podman-local-dev/env/.env.pair1` and `.env.pair2` before starting pods.

Files
- PowerShell scripts: `.copilot/plans/podman-local-dev/scripts/Setup-Pods.ps1`, `Remove-Pods.ps1`, `Verify-Setup.ps1`
- Env files: `.copilot/plans/podman-local-dev/env/.env.controller`, `.env.pair1`, `.env.pair2`

Notes
- This setup uses Podman pods to colocate agent+worker pairs so `WORKER_URL=http://localhost:8082` works inside the pod.
- The scripts prefer pod-based management rather than `podman-compose` to avoid build-context and subnet issues on Windows.
