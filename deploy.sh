#!/bin/bash

# List of directories with docker-compose.yml files
directories=(
  'apps/dns'
  'apps/reverse-proxy'
  'apps/pvr'
  'apps/vpn'
  'apps/monitoring'
  'apps/qbit'
  'apps/media/plex'
  'apps/portainer'
  'apps/dashboard'
  'apps/watchtower'
  'apps/tailscale'
)

for dir in "${directories[@]}"; do
  echo "Starting server in $dir"
  (cd "$dir" && docker compose up -d)
done

echo "All projects started."
