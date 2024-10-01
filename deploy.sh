#!/bin/bash

set -eo pipefail

SCRIPT_DIR=$(dirname "$0")
source "${SCRIPT_DIR}/env.sh"

pull_only=false
restart=false

function usage() {
  echo "Usage: ./deploy.sh <primary|network> [options]"
  echo "Options:"
  echo "  -r, --restart  Restart the app"
  echo "  -p, --pull-only  Pull images only"
}

function deploy() {
  deploys=$1
  for dir in $deploys; do
    echo "🚀 $dir"
    options=""
    if [ "$restart" = true ]; then
      options="up --force-recreate -d"
    fi
    if [ "$pull_only" = true ]; then
      options="pull"
    else
      options="up -d"
    fi
    full_cmd="docker compose -f $dir/docker-compose.yml $options"
    eval "$full_cmd"
  done
}

command=$1
if [ "$command" = "primary" ]; then
  apps=$(find apps/primary -mindepth 1 -maxdepth 1 -type d)

elif [ "$command" = "network" ]; then
  apps=$(find apps/network -mindepth 1 -maxdepth 1 -type d)
else
  echo "Unknown command: $command"
  usage
  exit 1
fi

while [[ "$#" -gt 1 ]]; do
  case $2 in
  -r | --restart) restart=true ;;
  -p | --pull-only) pull_only=true ;;
  *)
    echo "Unknown option: $2$"
    usage
    exit 1
    ;;
  esac
  shift
done

deploy "$apps"

echo "👍 Done"
