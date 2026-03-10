# Deployment

## How it works

`deploy.sh` syncs files from your laptop to the target server via rsync, then runs docker compose over SSH.

```bash
./deploy.sh <primary|network|dublin> [options]
```

### Options

| Flag | Description |
|------|-------------|
| `-r` | Restart (force-recreate) |
| `-p` | Pull images only |
| `-d` | Stop and remove containers |
| `-u` | Update (pull + restart) |

### What happens

1. **Sync**: rsync copies `apps/<target>/` and `env.sh` to the remote server
2. **Network**: ensures the `internal-net` Docker network exists (primary only)
3. **Deploy**: iterates each app directory and runs `docker compose` over SSH

### Remote paths

| Target | SSH | Path |
|--------|-----|------|
| primary | `root@192.168.88.212` | `/root/home-server/` |
| network | `root@192.168.0.200` | `/root/home-server/` |
| dublin | `admin@192.168.1.38` | `/home/admin/home-server/` |

## Known issues

- **Grafana/Prometheus permissions**: these containers run as `user: "1001"`. After image updates, volume dirs may need `chown -R 1001:1001 /storage/docker/home-server/{grafana,prometheus}`.
- **bash compatibility**: macOS ships bash 3.2. The deploy script avoids bash 4+ features (e.g. associative arrays).
- **No resilience to SSH drops**: if your laptop disconnects mid-deploy, the running docker compose command on the server will be killed. For large updates, consider SSHing in manually.

## TODO

- [ ] Add tmux-based resilient mode for long-running deploys
- [ ] Test deploy for network and dublin targets
- [ ] Weekly docker system prune via Ansible cron
