<!--
File: docs/007-scope-and-loader.md
Created: 2026-06-20
Description: Mod scoping and loader filtering rules.
-->
# Scope and Loader

## Scope
- `shared`: Included in both client and server builds
- `client`: Included only in client builds
- `server`: Included only in server builds

## Loader
- `loaderName` in packspec: Declares supported loaders
- Mod-level `loader`: Filters which loaders a mod applies to
- CLI `lock`/`build` commands accept a `--loader` flag
- Supported loaders: `neoforge`, `fabric`