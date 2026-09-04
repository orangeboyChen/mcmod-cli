<!--
File: docs/011-configuration.md
Created: 2026-09-04
Description: Project configuration.
-->
# Configuration

The only persistent project configuration is `.mcmod/config.json` in the
current directory. Set it with `mcmod config set-cf-key <key>` or
`mcmod set cf-key <key>`. `CURSEFORGE_API_KEY` overrides it for one process.
No `HOME`, `XDG_CONFIG_HOME`, or user/global path is read.
