# Possible Home Server Improvements

This document outlines potential improvements and new features for the home server setup. Each item includes details to help implement the changes.

## I. Critical Security Improvements

- [ ] **Externalize Hardcoded Secrets**
    - **Issue**: Sensitive information (API keys, passwords, tokens) is hardcoded in Docker Compose files.
    - **Affected Files & Action Details**:
        - `apps/primary/01-gluetun/docker-compose.yml`: The `WIREGUARD_PRIVATE_KEY` for Gluetun VPN is hardcoded. Move this key to `env.sh` (e.g., `export WIREGUARD_PRIVATE_KEY="your_actual_key"`) and update the `docker-compose.yml` to use `environment: - WIREGUARD_PRIVATE_KEY=${WIREGUARD_PRIVATE_KEY}`.
        - `apps/primary/cloudflared/docker-compose.yml`: The Cloudflare tunnel token is hardcoded in the `command` for the `cloudflared` service. Move this token to `env.sh` (e.g., `export CLOUDFLARE_TOKEN_PRIMARY="your_token_here"`) and update the `command` in `docker-compose.yml` to use this environment variable (e.g., `tunnel --no-autoupdate run --token ${CLOUDFLARE_TOKEN_PRIMARY}`).
        - `apps/cloud/nfl/docker-compose.yml`: A different Cloudflare tunnel token is hardcoded in the `command` for its `cloudflared` service. Move this token to `env.sh` (e.g., `export CLOUDFLARE_TOKEN_NFL="your_other_token_here"`) and update its `command` similarly.
        - `apps/primary/plex/docker-compose.yml`: The `PLEX_CLAIM` token is hardcoded. This token is typically for initial server claiming. Remove it from the `docker-compose.yml`. If needed for a fresh setup, it can be temporarily set via an environment variable or during Plex's web UI setup.
        - `apps/network/dns/docker-compose.yml`: The Pi-hole `WEBPASSWORD` is hardcoded. Move this to `env.sh` (e.g., `export PIHOLE_WEBPASSWORD="your_strong_pihole_password"`) and update the `docker-compose.yml` to use `environment: - WEBPASSWORD=${PIHOLE_WEBPASSWORD}`.

- [ ] **Secure Traefik Reverse Proxy**
    - **Issue 1**: Traefik dashboard and API are insecure (`api.insecure=true`).
        - **Affected Files**: `apps/network/traefik/docker-compose.yml` (command arguments) and `apps/network/traefik/traefik.yml`.
        - **Action Details**: 
            - In `docker-compose.yml`, remove `--api.insecure=true` from the `command` arguments for the Traefik service. In `traefik.yml`, change `insecure: true` to `insecure: false` under the `api` section or remove the line to use the default (false).
            - To secure the dashboard, add authentication middleware. For example, using BasicAuth, add to `apps/network/traefik/routes.yml` (for the Traefik dashboard router):
              ```yaml
              # http:
              #   routers:
              #     router7: # Assuming this is your traefik dashboard router
              #       # ... existing config ...
              #       middlewares:
              #         - traefik-auth
              #   middlewares:
              #     traefik-auth:
              #       basicAuth:
              #         users:
              #           - "admin:$apr1$yourgeneratedpasswordhash" # Replace with your user and hashed password
              ```
    - **Issue 2**: Traefik skips TLS verification for backend services (`serversTransport.insecureSkipVerify=true`).
        - **Affected File**: `apps/network/traefik/docker-compose.yml` (command arguments).
        - **Action Details**: Remove the `--serversTransport.insecureSkipVerify=true` argument from the Traefik service `command`. If backend services use self-signed certificates, configure Traefik to trust your internal CA, or use valid certificates for internal services.
    - **Issue 3**: Services are exposed over HTTP only.
        - **Affected Files**: `apps/network/traefik/docker-compose.yml` (command arguments or `traefik.yml` for static config) and `apps/network/traefik/routes.yml` (for dynamic config).
        - **Action Details**:
            - Configure a Let's Encrypt (ACME) certificate resolver in Traefik's static configuration. Add to `traefik.yml` or as command arguments in `docker-compose.yml`:
              ```yaml
              # certificatesResolvers:
              #   myresolver: # Choose a name for your resolver
              #     acme:
              #       email: your-email@example.com
              #       storage: /etc/traefik/acme.json # Path inside the container for acme.json
              #       httpChallenge:
              #         entryPoint: web # Use the HTTP entrypoint for challenges
              ```
              Ensure the `acme.json` path is mapped to a persistent volume.
            - Update routers in `apps/network/traefik/routes.yml` to use the `websecure` entrypoint and specify the TLS resolver. For each router:
              ```yaml
              # entryPoints:
              #   - websecure
              # rule: Host(`service.${DOMAIN}`)
              # tls:
              #   certResolver: myresolver
              ```
            - Implement HTTP to HTTPS redirection. In `traefik.yml` or as command arguments, configure the `web` entrypoint to redirect:
              ```yaml
              # entryPoints:
              #   web:
              #     address: ":80"
              #     http:
              #       redirections:
              #         entryPoint:
              #           to: websecure
              #           scheme: https
              #   websecure:
              #     address: ":443"
              ```

## II. Operational Best Practices & Enhancements

- [ ] **Use Non-Root Container Users**
    - **Issue**: Some services run as root (`PUID=0`/`PGID=0`), which is a security risk.
    - **Affected Files & Action Details**:
        - `apps/primary/books/docker-compose.yml` (Readarr): Change `PUID` and `PGID` environment variables from `0` to a non-root user ID (e.g., `1001`, matching other services).
        - `apps/primary/plex/docker-compose.yml` (Plex): Change `PUID` and `PGID` environment variables from `0` to a non-root user ID (e.g., `1001`).
    - **General Action**: After changing PUID/PGID, ensure file permissions on the corresponding host volumes (e.g., under `$DOCKER_VOLUME_PATH/readarr`, `$DOCKER_VOLUME_PATH/plex`, and relevant media paths) are owned by the new PUID/PGID. This might require `chown user:group /path/to/volume` commands on the host system.

- [ ] **Standardize Pi-hole Volume Path**
    - **Issue**: Pi-hole in `apps/network/dns/docker-compose.yml` uses `$HOME/etc-pihole` for its configuration volume and `./etc-dnsmasq.d` for Dnsmasq settings.
    - **Action Details**: For consistency with other services using `$DOCKER_VOLUME_PATH`, consider changing these paths. 
        - Modify `apps/network/dns/docker-compose.yml` volumes section:
          ```yaml
          # volumes:
          #   - "$DOCKER_VOLUME_PATH/pihole/config:/etc/pihole"
          #   - "$DOCKER_VOLUME_PATH/pihole/dnsmasq.d:/etc/dnsmasq.d"
          ```
        - Before restarting, move existing content from `$HOME/etc-pihole` to `$DOCKER_VOLUME_PATH/pihole/config` and from `./etc-dnsmasq.d` (relative to the compose file) to `$DOCKER_VOLUME_PATH/pihole/dnsmasq.d` on the host.

- [ ] **Remove qBittorrent Service**
    - **Reason**: Switched to using Transmission as the primary download client, making qBittorrent redundant.
    - **Action Details**:
        - Locate the `docker-compose.yml` file that defines the `qbittorrent` service (likely in a path such as `apps/primary/downloads/docker-compose.yml` or similar).
        - Remove the entire service definition for `qbittorrent` from that file.
        - Delete any associated qBittorrent configuration volumes on the host system (e.g., `$DOCKER_VOLUME_PATH/qbittorrent` or similar) after ensuring no critical data remains.
        - If qBittorrent's port (e.g., 8081) was specifically mapped in Gluetun's `docker-compose.yml` for direct access and is no longer needed by any other service, remove that port mapping from Gluetun's configuration.

- [x] **Add Healthchecks to More Services**
    - **Issue**: Previously, only `transmission` had a healthcheck. Many other services lacked them or had suboptimal configurations.
    - **Action Details**: Healthchecks were added or updated for the following services:
        - **Plex**: Added healthcheck using `http://localhost:32400/identity`.
        - **qBittorrent**: Added healthcheck using `http://localhost:8081/`.
        - **Readarr**: Updated healthcheck to use the `/ping` endpoint (`http://localhost:8787/ping`).
        - **Sonarr**: Updated healthcheck to use the `/ping` endpoint (`http://localhost:8989/ping`).
        - **Radarr**: Updated healthcheck to use the `/ping` endpoint (`http://localhost:7878/ping`).
        - **Pi-hole**: Initially added a custom healthcheck (`http://localhost/admin/api.php?status`), then switched to using the image's built-in healthcheck.
        - **Prometheus**: Added healthcheck using `http://localhost:9090/-/healthy`.
        - **Grafana**: Added healthcheck using `http://localhost:3000/api/health`.
        - **Traefik**: Added healthcheck using `traefik healthcheck --ping`.
        - **Gluetun**: Initially updated custom healthcheck to use `wget`, then switched to using the image's built-in healthcheck.
        - **Cloudflared (`apps/primary/cloudflared`)**: Updated healthcheck to use the `/ready` endpoint of its metrics server (e.g., `test: [ "CMD", "cloudflared", "tunnel", "--metrics", "localhost:60123", "ready" ]`). (Note: `apps/cloud/nfl/cloudflared` might need similar attention).
    - **Original Examples**: Plex, Sonarr, Radarr, qBittorrent, Prowlarr, Overseerr, Pi-hole. (Many of these are now addressed).
    - **Original Generic Example**: (Retained for reference if adding more healthchecks)
        ```yaml
        # healthcheck:
        #   test: ["CMD", "curl", "-f", "http://localhost:SERVICE_PORT_INSIDE_CONTAINER/"]
        #   interval: 1m30s
        #   timeout: 10s
        #   retries: 3
        #   start_period: 40s # Optional: give time for service to start
        ```

- [ ] **Review `network_mode: host` for Plex**
    - **Issue**: Plex in `apps/primary/plex/docker-compose.yml` uses `network_mode: host`, granting it full access to the host's network stack.
    - **Action Details (Optional)**: For increased network isolation, consider removing `network_mode: host`. Explicitly publish Plex's required ports. Traefik already handles external access via `plex.${DOMAIN}`.
        - Modify `apps/primary/plex/docker-compose.yml`:
          ```yaml
          # # Remove network_mode: host
          # ports:
          #   - "32400:32400/tcp" # Main Plex port
          #   # Add other necessary Plex ports for discovery if needed, e.g.:
          #   # - "3005:3005/tcp"
          #   # - "8324:8324/tcp"
          #   # - "32469:32469/tcp"
          #   # - "1900:1900/udp" # GDM
          #   # - "32410:32410/udp" # GDM
          #   # - "32412:32412/udp" # GDM
          #   # - "32413:32413/udp" # GDM
          #   # - "32414:32414/udp" # GDM
          ```
        - Ensure Plex is part of a Docker network (e.g., `internal-net`) that Traefik can access if `network_mode: host` is removed.

- [ ] **Consolidate Cloudflare Tunnel Strategy**
    - **Issue**: Multiple `cloudflared` services are defined (`apps/primary/cloudflared/docker-compose.yml`, `apps/cloud/nfl/docker-compose.yml`), each with its own tunnel token.
    - **Action Details (Consideration)**: Evaluate if a single `cloudflared` service instance could manage all required public hostnames. This can be done by configuring ingress rules within your Cloudflare Zero Trust dashboard for a single tunnel. This would simplify management and reduce the number of tunnel tokens to manage. If separate tunnels are intentionally kept for strict isolation, ensure all tokens are externalized to `env.sh` as per the secrets management point.

## III. New Features & Further Improvements

- [ ] **Implement a Backup Strategy for Configuration Volumes**
    - **Issue**: No explicitly defined automated backup strategy for persistent Docker volume data.
    - **Action Details**:
        - Identify all critical configuration volumes (primarily directories under `$DOCKER_VOLUME_PATH`, and the Pi-hole configuration directory if moved as suggested).
        - Choose a backup tool/method. Options include:
            - **Duplicati**: Can run as a Docker container, offers web UI, encryption, scheduling, and various backends (local, cloud).
            - **Restic / BorgBackup**: Powerful command-line backup tools, good for scripting, offer deduplication and encryption.
            - **Simple rsync scripts**: Cron jobs that `rsync` volume data to an external drive, NAS, or cloud storage (e.g., via `rclone`).
        - Configure regular, automated backups. Document the backup schedule and restore procedure.

- [ ] **Set Up Centralized Logging**
    - **Issue**: Container logs are currently managed by Docker and viewed per container (e.g., `docker logs <container_name>`).
    - **Action Details**: Implement a centralized logging stack.
        - **Popular choices**: Grafana Loki with Promtail, ELK Stack (Elasticsearch, Logstash, Kibana), EFK Stack (Elasticsearch, Fluentd, Kibana).
        - **Example with Loki/Promtail** (since Grafana is already in use):
            - Deploy Loki as a Docker service.
            - Deploy Promtail as a Docker service. Configure Promtail to discover Docker container logs on the host and send them to Loki (e.g., by mounting `/var/run/docker.sock` and Docker log directories).
            - Add Loki as a data source in your existing Grafana instance and create dashboards for log exploration.

- [ ] **Explore Automated Container Updates (Watchtower)**
    - **Issue**: Container images are updated manually via the `./deploy.sh --update` script.
    - **Action Details**: Consider deploying Watchtower for automated updates.
        - Create a `docker-compose.yml` for Watchtower or run it directly:
          ```yaml
          # services:
          #   watchtower:
          #     image: containrrr/watchtower
          #     container_name: watchtower
          #     restart: unless-stopped
          #     volumes:
          #       - /var/run/docker.sock:/var/run/docker.sock
          #     environment:
          #       - WATCHTOWER_CLEANUP=true # Remove old images
          #       - WATCHTOWER_SCHEDULE="0 0 4 * * *" # Example: Run daily at 4 AM
          #       # - WATCHTOWER_NOTIFICATIONS=email # Optional: configure notifications
          #       # - WATCHTOWER_NOTIFICATION_EMAIL_FROM=...
          #       # - WATCHTOWER_NOTIFICATION_EMAIL_TO=...
          #       # - WATCHTOWER_NOTIFICATION_EMAIL_SERVER=...
          #       # - WATCHTOWER_MONITOR_ONLY=true # For just notifications without auto-updating
          ```
        - Review Watchtower documentation for advanced configuration (e.g., specific container monitoring, notifications).

- [ ] **Define Resource Limits for Containers**
    - **Issue**: No resource limits (CPU, memory) are set for containers, potentially allowing one service to impact others.
    - **Action Details (Optional)**: For resource-intensive applications (e.g., Plex during transcoding, download clients, Prometheus) or to improve overall system stability, add resource limits to their service definitions in the relevant `docker-compose.yml` files using the `deploy.resources` key.
      ```yaml
      #   deploy:
      #     resources:
      #       limits:
      #         cpus: '1.0'  # Limit to 1 CPU core
      #         memory: 1024M # Limit to 1GB RAM
      #       reservations: # Optional: guarantee resources
      #         cpus: '0.5'
      #         memory: 512M
      ```
      Note: `deploy` key usage might depend on how you run `docker-compose` (e.g., `docker stack deploy` for swarm mode, or compose v2+ for standalone).
