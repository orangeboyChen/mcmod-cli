<!--
File: docs/003-lock-files.md
Created: 2026-09-04
Description: Dependency lock format and recursive expansion output.
-->

# Lock Files

Lock files live at `locks/dependencies/<minecraftVersion>-<loader>.json`.

The `mods` map contains the complete resolved set, including mods discovered
inside recursive Git packspec bundles. Expanded keys are namespaced and are
not added to the root `packspec.json`.

Each entry contains `name`, `scope`, `source`, optional `identity`, and a
fingerprint `hash`. `dependencies` is reserved for dependency references
reported by resolved artifacts; Git packspec expansion is represented by the
flattened entries and their source metadata.

Run `mcmod lock 1.21.1 neoforge` to regenerate the file. Incremental locking
keeps matching entries and removes entries no longer present after expansion.
Run summaries and failures are written to stderr; the JSON remains a clean
source of truth.
