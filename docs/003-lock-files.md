<!--
File: docs/003-lock-files.md
Created: 2026-09-04
Description: Defines generated dependency lock files.
-->

# Lock Files

`mcmod lock` writes `locks/dependencies/<minecraftVersion>-<loader>.json`.
The top-level fields are `minecraftVersion`, `loader`, optional
`loaderVersion`, and `mods`.

Each mod entry stores `name`, `version`, `scope`, and a resolved `source`.
It may also contain `identity`, `dependencies` (an array of `{id, required}`
references), and the resolver-written `hash` fingerprint. Git bundles are
expanded before locking: their child mods appear as flattened entries with
repository-namespaced keys. The root `packspec.json` is never changed.

Repeated locks keep unchanged entries, add new entries, and remove entries no
longer reachable from the root specification. Progress and partial failures
are printed to stderr, not persisted in the lock JSON.

```json
{
  "minecraftVersion": "1.21.1",
  "loader": "neoforge",
  "loaderVersion": "21.1.219",
  "mods": {
    "create": {
      "name": "Create",
      "version": "0.6.0",
      "scope": "shared",
      "source": {"type": "curseforge", "modId": 328085, "fileId": 5812340, "fileName": "create.jar"},
      "dependencies": [{"id": "flywheel", "required": true}],
      "hash": "sha256:..."
    }
  }
}
```
