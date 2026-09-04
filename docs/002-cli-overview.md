<!--
File: docs/002-cli-overview.md
Created: 2026-09-04
Description: Documents the supported mcmod command surface.
-->

# CLI Overview

Use `mcmod` or `mcm`.

```text
mcmod list
mcmod lock [minecraftVersion] [loader]
mcmod lock list|show|add|update|delete|tree
mcmod lock release set|list|show|delete
mcmod build [minecraftVersion] [loader] [--target client|server|both] [--build-type cf|all]
mcmod tree [minecraftVersion] [loader]
mcmod config [set-cf-key <key>]
mcmod set cf-key <key> [--project] [--global]
mcmod validate [--spec <path>] [--lock <path>] [--release-index <path>]
mcmod version
```

`lock`, `build`, and `tree` use the packspec defaults when version or loader
is omitted. `--project` and `--global` are compatibility flags: all key
commands write the current project file `.mcmod/config.json`.

`mcmod tree` prints resolved roots and recursively indents dependencies by two
spaces. Cyclic components are emitted once, with recursion guarded.

```text
Create curseforge:328085 0.6.0
  Flywheel curseforge:486392 0.6.10
```
