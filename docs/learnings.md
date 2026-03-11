# Learnings

Notes from infrastructure work, intended to help future sessions.

## Deployment

- **deploy.sh is legacy** — replaced by Ansible deploy playbook + Makefile. The bash script still exists but the Makefile is the preferred entry point.
- **rsync + SSH** is the deployment model. No git repos on servers. Compose files and env.sh are synced to the remote, then `docker compose` runs there.
- **Remote deploy paths**: `/root/home-server/` (primary, network), `/home/admin/home-server/` (dublin).
- **`internal-net`** is an external Docker network on primary that must exist before any compose stack starts. The deploy playbook handles this.
- **macOS bash is 3.2** — avoid bash 4+ features (associative arrays, `declare -A`) in scripts.

## Ansible

- **`action` is a reserved variable name** in Ansible. Use `deploy_action` or similar.
- **`--start-at-task` doesn't work with `include_tasks`** — subtasks aren't visible at the top level. For partial runs, use tags or ad-hoc commands instead.
- **Dotfiles/git/stow tasks are macOS-only** — don't include them in server playbooks (they try to use `brew` on Debian).
- **`primary.yml` targets `hosts: primary_lxc`** using `hosts/lxcs` inventory. The old `vms` inventory only has network_lxc.
- **Proxmox host** is in `hosts/proxmox` inventory, uses `proxmox.yml` playbook (unattended-upgrades only).

## Docker

- **Grafana + Prometheus run as `user: "1001"`** — volume dirs must be owned `1001:1001`. After image updates, may need `chown -R 1001:1001` on their data dirs.
- **After Proxmox kernel upgrades**, LXC containers may fail with sysctl permission errors. Rebooting the LXC from Proxmox (`pct reboot <id>`) fixes it.
- **`nfl-cloudflared-1` and `nfl-ratings`** are long-running unhealthy containers — pre-existing, not caused by deploys.

## Network LXC — handle with care

- Runs **Tailscale, Traefik, Pi-hole**. If it goes down, remote access is lost.
- **Never deploy/restart network LXC remotely** unless you have physical access or an alternative way in.

## Makefile

- `make help` lists all targets.
- Docker `{{.Names}}` format strings work fine in Makefile — no escaping needed for Go template braces.
- Use `@` prefix on commands to suppress echoing (e.g. status commands).
