---
name: dotfiles-check
description: Proactively check whether system-level changes should be captured in the dotfiles repo and Ansible flow. Use this skill automatically whenever you create or modify LaunchAgents, shell configs, scripts in ~/.local/bin, plist files, cron jobs, Homebrew packages, macOS defaults, or any other system configuration outside the home-server repo.
allowed-tools: Bash, Read, Glob, Grep, AskUserQuestion
---

# Dotfiles / Ansible Integration Check

After making any system-level change, run through this checklist before considering the task done.

## What counts as system-level

- LaunchAgent plists (`~/Library/LaunchAgents/`)
- Scripts in `~/.local/bin/`
- Shell config changes (`.zshrc`, `.zshenv`, `.aliases.sh`, `.darwin.sh`)
- Homebrew packages (`brew install`/`brew install --cask`)
- macOS defaults (`defaults write`)
- SSH config changes
- App preference imports (`defaults import`)
- VS Code / Cursor extensions or settings
- Cron jobs or periodic tasks
- Any config file under `~/.config/`

## Repo structure

```
~/code/personal/home-server/
├── dotfiles/              # Stowed into $HOME via GNU Stow
│   ├── .aliases.sh
│   ├── .zshrc
│   ├── .config/
│   │   └── launchagents/  # Plist sources (symlinked into ~/Library/LaunchAgents/)
│   ├── .local/bin/        # User scripts
│   └── .claude/skills/    # Claude Code skills
├── infra/ansible/
│   ├── laptop.yml         # Main laptop playbook
│   └── tasks/             # Individual task files
└── Brewfile               # Homebrew bundle (stowed to ~/Brewfile)
```

## Checklist

For each system-level change, determine which of these apply:

1. **File in dotfiles/?** — If you created or modified a file that lives under `$HOME`, check if it should be in `dotfiles/` so stow manages it. Don't put generated/binary files in dotfiles.

2. **Stow re-run needed?** — If you added a new file to `dotfiles/`, run `stow -t $HOME dotfiles` from the repo root.

3. **LaunchAgent symlink?** — Plists go in `dotfiles/.config/launchagents/`. They need a symlink from `~/Library/LaunchAgents/` pointing to `~/.config/launchagents/<name>.plist`. This symlink is created by an Ansible task, not by stow.

4. **Ansible task needed?** — LaunchAgents need a task in `infra/ansible/tasks/` to create the symlink and load them. Other setup steps that can't be handled by stow alone (e.g. `defaults write`, directory creation, `launchctl load`) also need Ansible tasks.

5. **laptop.yml updated?** — If you created a new Ansible task file, include it in `infra/ansible/laptop.yml`.

6. **Brewfile updated?** — If you installed something via `brew install` or `brew install --cask`, check if it's in the Brewfile. Run `brew bundle dump --file=- --no-restart` to see what's currently installed and diff against the existing Brewfile.

## How to prompt the user

After identifying what needs to happen, tell the user concisely:

> These changes affect system config. To keep your dotfiles in sync:
> - [list only the items that apply]

Then ask if they want you to do it now. Don't do it silently — the user should know what's going into their dotfiles repo.
