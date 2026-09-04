<!--
File: docs/003-lock-files.md
Created: 2026-09-04
Description: Defines generated dependency lock files.
-->

# Lock Files

`mcmod lock` writes `locks/dependencies/<minecraftVersion>-<loader>.json`.
The top-level fields are `minecraftVersion`, `loader`, optional
`loaderVersion`, and `mods`.

Each mod entry stores display metadata, scope, resolved source, optional
identity and dependency references, plus a source fingerprint. Git bundles are
expanded before locking: their child mods appear as flattened entries with
repository-namespaced keys. The root `packspec.json` is never changed.

Repeated locks keep unchanged entries, add new entries, and remove entries no
longer reachable from the root specification. Progress and partial failures
are printed to stderr, not persisted in the lock JSON.
