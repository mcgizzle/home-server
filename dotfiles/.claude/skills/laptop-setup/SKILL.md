---
name: laptop-setup
description: Bootstrap a fresh macOS laptop. Configures SSH via 1Password, clones the home-server repo, and runs the Ansible playbook to fully provision the machine. Use this skill when the user says they want to set up a new laptop, bootstrap their machine, or run the laptop setup.
allowed-tools: Bash, AskUserQuestion
---

# Fresh macOS Laptop Setup

This skill orchestrates the full setup of a new Mac. The bootstrap script (install.sh) has already installed Homebrew, 1Password, and Claude Code, and the user has signed into 1Password.

Run each step in order. Confirm success before moving to the next. If a step fails, diagnose and fix before continuing.

## Step 1: Verify 1Password SSH agent

```bash
test -S "$HOME/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock" && echo "OK: 1Password SSH agent is running" || echo "FAIL: SSH agent socket not found"
```

If this fails, ask the user to open 1Password > Settings > Developer > enable SSH Agent. Do not proceed until this passes.

## Step 2: Configure SSH to use 1Password agent

```bash
mkdir -p "$HOME/.ssh" && chmod 700 "$HOME/.ssh"
if [ ! -f "$HOME/.ssh/config" ]; then
  cat > "$HOME/.ssh/config" <<'SSHEOF'
Host *
	IdentityAgent "~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock"
SSHEOF
  chmod 644 "$HOME/.ssh/config"
  echo "Created ~/.ssh/config"
else
  echo "~/.ssh/config already exists, skipping"
fi
```

## Step 3: Verify SSH access to GitHub

```bash
ssh -T git@github.com 2>&1 || true
```

The user may see a 1Password prompt to authorize the SSH key. Tell them to approve it. Expected output contains "Hi mcgizzle!". If it says "Permission denied", the user needs to add their SSH key to their GitHub account via 1Password.

## Step 4: Install Ansible

```bash
eval "$(/opt/homebrew/bin/brew shellenv)"
brew install ansible 2>/dev/null || true
ansible-galaxy collection install community.general 2>/dev/null || true
```

## Step 5: Clone the home-server repo

```bash
mkdir -p "$HOME/code/personal"
if [ ! -d "$HOME/code/personal/home-server" ]; then
  git clone git@github.com:mcgizzle/home-server.git "$HOME/code/personal/home-server"
else
  echo "home-server repo already cloned"
fi
```

## Step 6: Run the Ansible playbook

This is the main event. It handles:
- Stows dotfiles from `home-server/dotfiles/` into `$HOME` via GNU Stow (Brewfile, shell configs, VS Code settings, app preferences, etc.)
- `brew bundle` from Brewfile (all packages, casks, VS Code extensions)
- Zsh + Zim framework
- Git config (user identity, SSH known_hosts)
- macOS defaults (Dock right + autohide, fast key repeat, dark mode, en_GB locale, screenshots to ~/Documents/Screenshots, menu bar icon spacing)
- App preferences (Stats, AltTab, Rectangle, Itsycal, HiddenBar imported via defaults)
- VS Code settings (symlinked from dotfiles)
- Dotfiles-sync LaunchAgent (auto-backup every Monday 9am)

```bash
eval "$(/opt/homebrew/bin/brew shellenv)"
ansible-playbook "$HOME/code/personal/home-server/infra/ansible/laptop.yml"
```

If ansible-playbook fails on a specific task, diagnose the error and retry. Common issues:
- SSH key not authorized: user needs to approve the key in 1Password
- brew bundle timeout: re-run, it picks up where it left off
- stow conflict: existing files backed up to `~/.config-backup`
- `community.general` collection missing: run `ansible-galaxy collection install community.general`

## Step 7: Post-setup

Tell the user:
1. Open a **new terminal** to pick up the shell config (Zsh + Zim)
2. **Log out and back in** for macOS defaults to fully apply (especially menu bar spacing and Dock position)
3. Open **IntelliJ IDEA** and sign in to JetBrains account — Settings Sync will restore all IDE settings automatically
4. Set the `FIGMA_API_KEY` environment variable for the Claude Figma MCP server (get the key from Figma > Settings > Personal Access Tokens)
5. The dotfiles-sync LaunchAgent is now running — it auto-commits Brewfile and app preference changes every Monday at 9am
