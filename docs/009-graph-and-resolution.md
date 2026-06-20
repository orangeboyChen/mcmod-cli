<!--
File: docs/009-graph-and-resolution.md
Created: 2026-06-20
Description: Dependency graph and resolution.
-->
# Graph and Resolution

## Dependency Graph
- Built from packspec mods
- Each mod is a node with scope, source, and version info
- Edges represent dependencies between mods

## Cycle Detection
- Uses DFS-based cycle detection
- Returns first cycle found (empty = no cycles)

## Resolution
- Filters mods by loader
- Resolves sources via CurseForge/GitHub/Git/Local resolvers
- Generates lock entries