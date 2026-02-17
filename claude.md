# Home Server

## Architecture

Proxmox host (`192.168.88.211:8006`) running LXC containers:

- **Primary** (`192.168.88.212`) - Media services, monitoring, Cloudflare tunnel
- **Network** (`192.168.88.213` / `192.168.0.200`) - Traefik, Pi-hole, Tailscale
- **Dublin** (Raspberry Pi, `192.168.1.38`) - Tailscale

## SSH Access

```bash
ssh root@192.168.88.212  # Primary
ssh root@192.168.0.200   # Network LXC
ssh admin@192.168.1.38   # Dublin Pi
```

SSH key: `~/.ssh/id_ed25519`

## Environment

All env vars are defined in `env.sh` at the repo root. This must be sourced before running docker compose on the server. The deploy script (`deploy.sh`) handles this automatically.

Key variables: `DOCKER_VOLUME_PATH`, `CLOUDFLARE_TOKEN_PRIMARY`, `WIREGUARD_PRIVATE_KEY`, `PIHOLE_WEBPASSWORD`, IPs, domain (`mcg.lan`).

## Deployment

```bash
./deploy.sh <primary|network|dublin> [options]
```

Options: `-r` restart, `-p` pull-only, `-d` down, `-u` update (pull + restart)

Sources `env.sh`, then runs `docker compose up -d` for each app directory under `apps/<target>/`.

## Services

### Primary (`apps/primary/`)

| Service | Port | Notes |
|---------|------|-------|
| Cloudflared | - | Cloudflare tunnel, exposes Overseerr externally |
| Overseerr | 5055 | Request management, external at `requests.mcgizzle.casa` |
| Plex | 32400 | Media server |
| Radarr | 7878 | Movie management |
| Sonarr | 8989 | TV management |
| Prowlarr | 9696 | Indexer management |
| Readarr | 8787 | Book management |
| Transmission | 9091 | Torrent client |
| Gluetun | - | VPN (Surfshark WireGuard) |
| Prometheus | 9090 | Metrics |
| Grafana | 3000 | Dashboards |

### Network (`apps/network/`)

| Service | Port | Notes |
|---------|------|-------|
| Traefik | 80, 443, 8080 | Reverse proxy, routes defined in `traefik/routes.yml` |
| Pi-hole | 81 | DNS/ad-blocking |
| Tailscale | - | VPN mesh |

### Dublin (`apps/dublin/`)

| Service | Notes |
|---------|-------|
| Tailscale | VPN mesh |

## Cloudflare Tunnel

The primary tunnel authenticates via `CLOUDFLARE_TOKEN_PRIMARY` and routes `requests.mcgizzle.casa` to `http://overseerr:5055` over the `internal-net` Docker network.

A DDNS updater script (`infra/cloudflare-template.sh`) runs daily at 11:00 UTC via crontab to update the `requests.mcgizzle.casa` A record.

## Ansible

Provisioning playbooks in `infra/ansible/`:

- `primary.yml` - Provisions primary VMs (targets `vms` group from `infra/ansible/vms` inventory)
- `dublin.yml` - Provisions Dublin Pi (targets `dublin` group from `infra/ansible/hosts/dublin` inventory)
- Tasks in `infra/ansible/tasks/` (SSH config, etc.)

## Docker Networking

Services on primary share the `internal-net` bridge network, allowing container-to-container communication by service name (e.g., cloudflared reaches overseerr via `http://overseerr:5055`).
