#!/usr/bin/env bash

set -eo pipefail

SCRIPT_DIR=$(dirname "$0")
source "${SCRIPT_DIR}/env.sh"

pull_only=false
restart=false
down=false
update=false

function usage() {
  echo "Usage: ./deploy.sh <primary|network|dublin> [options]"
  echo "Options:"
  echo "  -r, --restart  Restart the app"
  echo "  -p, --pull-only  Pull latest image"
  echo "  -d, --down  Stop the app & remove container"
  echo "  -u, --update Pull latest image & restart"
}

function get_ssh_target() {
  case $1 in
    primary) echo "root@${PRIMARY_IP}" ;;
    network) echo "root@${NETWORK_IP}" ;;
    dublin)  echo "admin@192.168.1.38" ;;
  esac
}

function get_remote_path() {
  case $1 in
    dublin) echo "/home/admin/home-server" ;;
    *)      echo "/root/home-server" ;;
  esac
}

function sync_files() {
  local target=$1
  local ssh_target=$(get_ssh_target "$target")
  local remote_path=$(get_remote_path "$target")

  echo "📦 Syncing files to ${ssh_target}:${remote_path}"

  ssh -o ConnectTimeout=5 "${ssh_target}" "mkdir -p ${remote_path}/apps/${target}"

  rsync -az --delete \
    -e "ssh -o ConnectTimeout=5" \
    "${SCRIPT_DIR}/apps/${target}/" \
    "${ssh_target}:${remote_path}/apps/${target}/"

  rsync -az \
    -e "ssh -o ConnectTimeout=5" \
    "${SCRIPT_DIR}/env.sh" \
    "${ssh_target}:${remote_path}/env.sh"
}

function ensure_network() {
  local target=$1
  local ssh_target=$(get_ssh_target "$target")

  if [ "$target" = "primary" ]; then
    echo "🔗 Ensuring internal-net network exists"
    ssh "${ssh_target}" "docker network inspect internal-net >/dev/null 2>&1 || docker network create internal-net"
  fi
}

function deploy() {
  local target=$1
  local ssh_target=$(get_ssh_target "$target")
  local remote_path=$(get_remote_path "$target")
  local app_dirs

  app_dirs=$(ssh "${ssh_target}" "find ${remote_path}/apps/${target} -mindepth 1 -maxdepth 1 -type d | sort")

  for dir in $app_dirs; do
    echo "🚀 $(basename "$dir")"
    local cmd="cd ${remote_path} && source env.sh && docker compose -f ${dir}/docker-compose.yml"

    if [ "$restart" = true ]; then
      ssh "${ssh_target}" "${cmd} up --force-recreate -d"
    elif [ "$pull_only" = true ]; then
      ssh "${ssh_target}" "${cmd} pull"
    elif [ "$down" = true ]; then
      ssh "${ssh_target}" "${cmd} down"
    elif [ "$update" = true ]; then
      ssh "${ssh_target}" "${cmd} pull"
      ssh "${ssh_target}" "${cmd} up --force-recreate -d"
    else
      ssh "${ssh_target}" "${cmd} up -d"
    fi
  done
}

target=$1
if [ -z "$(get_ssh_target "$target")" ]; then
  echo "Unknown target: $target"
  usage
  exit 1
fi

while [[ "$#" -gt 1 ]]; do
  case $2 in
  -r | --restart) restart=true ;;
  -p | --pull-only) pull_only=true ;;
  -d | --down) down=true ;;
  -u | --update) update=true ;;
  *)
    echo "Unknown option: $2"
    usage
    exit 1
    ;;
  esac
  shift
done

sync_files "$target"
ensure_network "$target"
deploy "$target"

echo "👍 Done"
