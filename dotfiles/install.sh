#!/bin/bash
set -e

# Minimal bootstrap for a fresh macOS laptop.
# Gets Claude Code running with the laptop-setup skill, then Claude handles everything else.
#
# On a fresh machine:
#   curl -fsSL https://gist.githubusercontent.com/mcgizzle/04942da061d62a43b74ec489e2fcd1de/raw/install.sh | bash

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
echo "========================================"
echo "  Bootstrap complete!"
echo ""
echo "  1. Open 1Password and sign in"
echo "  2. Open a new terminal and run: claude"
echo "  3. Type: /laptop-setup"
echo "========================================"
