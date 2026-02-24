#!/bin/bash
set -e

# Bootstrap a fresh macOS laptop.
# Installs prerequisites, waits for 1Password, then hands off to Claude Code.
#
# On a fresh machine:
#   curl -fsSL https://gist.githubusercontent.com/mcgizzle/04942da061d62a43b74ec489e2fcd1de/raw/install.sh -o /tmp/install.sh && bash /tmp/install.sh

GIST_BASE="https://gist.githubusercontent.com/mcgizzle/04942da061d62a43b74ec489e2fcd1de/raw"
SKILL_DIR="$HOME/.claude/skills/laptop-setup"

echo "==> Installing Xcode Command Line Tools"
if ! xcode-select -p &>/dev/null; then
  xcode-select --install
  echo "    Waiting for Xcode CLT install to complete..."
  until xcode-select -p &>/dev/null; do sleep 5; done
fi

echo "==> Installing Homebrew"
if ! command -v brew &>/dev/null; then
  NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
eval "$(/opt/homebrew/bin/brew shellenv)"

echo "==> Installing 1Password and Claude Code"
brew install --cask 1password 2>/dev/null || true
brew install claude-code 2>/dev/null || true

echo "==> Installing laptop-setup skill"
mkdir -p "$SKILL_DIR"
curl -fsSL "$GIST_BASE/SKILL.md" -o "$SKILL_DIR/SKILL.md"

echo ""
echo "==> Opening 1Password"
open -a "1Password"

echo ""
echo "========================================"
echo "  Sign into 1Password and enable the"
echo "  SSH Agent (Settings > Developer)."
echo ""
echo "  Press Enter when ready..."
echo "========================================"
read -r

if [ -S "$HOME/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock" ]; then
  echo "==> 1Password SSH agent detected"
else
  echo "==> Warning: SSH agent not detected. Claude will help you fix this."
fi

echo "==> Handing off to Claude Code..."
exec claude "/laptop-setup"
