<!--
File: docs/011-configuration.md
Created: 2026-06-20
Description: Configuration management for API keys.
-->
# Configuration

## CurseForge API Key Priority
1. `CURSEFORGE_API_KEY` environment variable
2. `.mcmod/config.json` (project-level)
3. `~/.config/mcmod/config.json` (user-level)

## Commands
```
mcmod config set-cf-key <key>    # Set project-level key
mcmod config                      # Show current key
```