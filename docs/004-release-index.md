<!--
File: docs/004-release-index.md
Created: 2026-09-04
Description: Defines generated release index files.
-->

# Release Index

Release indexes live at `locks/releases/<minecraftVersion>.json`. The index
contains `type`, `packName`, `minecraftVersion`, and `releases`. Each release
has a unique `version`, a `type`, optional GitHub metadata, and per-loader
artifact paths.

Use `mcmod lock release set`, `list`, `show`, and `delete` to manage records.
The index is generated project state and may be committed.
