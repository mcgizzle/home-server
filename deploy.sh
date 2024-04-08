#!/bin/bash

SCRIPT_DIR=$(dirname "$0")
source "${SCRIPT_DIR}/.env"

# Initialize variables
pull_only=false

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
  case $1 in
    --pull-only) pull_only=true ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
  shift
done

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
  echo "Processing $dir"
  (cd "$dir" && if [ "$pull_only" = true ]; then
    echo "Pulling images in $dir"
    docker compose pull
  else
    echo "Starting server in $dir"
    docker compose up -d
  fi)
done

if [ "$pull_only" = true ]; then
  echo "All images pulled."
else
  echo "All projects started."
fi
