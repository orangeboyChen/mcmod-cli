<!--
File: docs/003-lock-files.md
Created: 2026-06-20
Description: Dependency lock file format.
-->
# Lock Files

## Location
`locks/dependencies/<minecraftVersion>-<loader>.json`

## Schema
- `loader` (string): Loader name
- `loaderVersion` (string, optional): Loader version
- `minecraftVersion` (string): Minecraft version
- `mods` (object): Locked mod entries keyed by normalized ID

## LockedMod
- `name` (string, optional): Display name
- `version` (string, optional): Resolved version
- `scope` (string): "shared", "client", or "server"
- `identity` (object, optional): Identity info
- `dependencies` (array): Dependency references
- `source` (object): Locked source details

## LockedSource
- `type` (string): One of "curseforge", "github-release", "git", "local"
- The remaining fields depend on `type`. See docs/005-source-resolution.md
  for the per-type schema.

## Lock Run Summary
The `mcmod lock` command does NOT write per-failure or per-removal records
into the lock file itself. Instead, the run summary (added/kept/removed/failed
counts and per-failure details) is written to stderr so the lock file
stays a clean source of truth that matches this schema. The CLI is
responsible for surfacing partial failures to the operator.