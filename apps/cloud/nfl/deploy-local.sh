#!/bin/bash
# Local deploy script - SSHs into server and runs deploy.sh
# This file is excluded from deployment

set -e

# Source env.sh from project root for PRIMARY_IP
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../../env.sh"

SSH_USER="${SSH_USER:-root}"
SSH_HOST="${SSH_HOST:-$PRIMARY_IP}"
REMOTE_PATH="/root/code/home-server/apps/cloud/nfl"

echo "Deploying NFL app to $SSH_USER@$SSH_HOST..."
ssh "$SSH_USER@$SSH_HOST" "cd $REMOTE_PATH && ./deploy.sh"
echo "Done!"
