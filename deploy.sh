#!/bin/bash

SCRIPT_DIR=$(dirname "$0")
source "${SCRIPT_DIR}/.env"

pull_only=false
restart=false

while [[ "$#" -gt 0 ]]; do
  case $1 in
    -r|--restart) restart=true ;;
    --pull-only) pull_only=true ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
  shift
done

directories=(
  'apps/vpn'
  'apps/qbit'
  'apps/reverse-proxy'
  'apps/pvr'
  'apps/cloudflared'
  'apps/media/plex'
  'apps/monitoring'
#  'apps/portainer'
#  'apps/dns'
#  'apps/tailscale'
)

for dir in "${directories[@]}"; do
  echo "Processing $dir"
  (cd "$dir" && if [ "$pull_only" = true ]; then
    echo "Pulling images in $dir"
    docker compose pull
  elif [ "$restart" = true ]; then
    echo "Restarting app in $dir"
    docker compose down
    docker compose up -d --force-recreate
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
