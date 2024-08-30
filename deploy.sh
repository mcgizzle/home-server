#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(dirname "$0")
source "${SCRIPT_DIR}/.env"

pull_only=false
restart=false

command=$1
if [ "$command" = "primary" ]; then
  apps=$(ls "$SCRIPT_DIR"/apps/primary)
elif [ "$command" = "network" ]; then
  apps=$(ls "$SCRIPT_DIR"/apps/network)
else
  echo "Unknown command: $command"
  usage
  exit 1
fi

while [[ "$#" -gt 1 ]]; do
  case $2 in
    -r|--restart) restart=true ;;
    -p|--pull-only) pull_only=true ;;
    *) echo "Unknown option: $2$"; usage; exit 1 ;;
  esac
  shift
done

function usage () {
  echo "Usage: deploy.sh <primary|network> [options]"
  echo "Options:"
  echo "  -r, --restart  Restart the app"
  echo "  -p, --pull-only  Pull images only"
}

function deploy () {
  apps=$1
  for dir in "${apps[@]}"; do
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
}

deploy "$apps"

if [ "$pull_only" = true ]; then
  echo "All images pulled."
else
  echo "All projects started."
fi
