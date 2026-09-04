<!--
File: docs/002-cli-overview.md
Created: 2026-06-20
Description: CLI command reference for mcmod.
-->
# CLI Overview

`mcmod` is a Go CLI for managing Minecraft modpack specifications, dependency
locks, jar resolution/download, build artifacts, and release indexes.
The same CLI is also available as the shorter `mcm` executable.

## Root Command

```text
$ mcmod --help
mcmod is a CLI for managing Minecraft modpack specifications,
dependency locks, jar resolution, downloading, build artifacts, and release indexes.

Usage:
  mcmod [command]

Available Commands:
  build       Build modpack artifacts (client/server zips)
  completion  Generate the autocompletion script for the specified shell
  config      Manage CLI configuration (API keys)
  help        Help about any command
  list        List mods from packspec.json
  lock        Resolve and lock mod dependencies
  set         Configure CLI keys and settings
  tree        Show dependency tree (alias for lock tree)
  validate    Validate packspec.json or lock files
  version     Print version information
```

## Command Reference

### `mcmod set`

Configure CLI keys and settings.

```text
Usage: mcmod set cf-key <key> [--project] [--global]
```

| Flag | Description |
|---|---|
| `--project` | Write to project config (`.mcmod/config.json`) |
| `--global`  | Write to user config (`~/.config/mcmod/config.json`) |

Resolution order on read: env `CURSEFORGE_API_KEY` → project → user.

Examples:

```bash
mcmod set cf-key "$CF_KEY"          # user-level (default)
mcmod set cf-key "$CF_KEY" --project # project-level
mcmod set cf-key "$CF_KEY" --global  # user-level (alias)
```

Output on success: `set cf-key` (printed to stdout).

### `mcmod list`

List mods from `packspec.json`, grouped by scope.

```text
$ mcmod list
pack v (1)
loader:
  - neoforge

[Server]
  (empty)

[Client]
  (empty)

[Shared]
  m | MyMod | 1.0 | local | mymod.jar
```

### `mcmod validate`

Validate `packspec.json`, lock files, or release indexes.

```text
Usage: mcmod validate [--spec <path>] [--lock <file>] [--release-index <file>]
```

| Flag | Description |
|---|---|
| `--spec`           | Path to a `packspec.json` file |
| `--lock`           | Path to a single lock file |
| `--release-index`  | Path to a release index JSON file |

Without flags, validates `./packspec.json`.

Output on success:
- `packspec.json is valid`
- `Lock file is valid`
- `Release index is valid`

### `mcmod lock`

Resolve and lock mod dependencies.

```text
Usage: mcmod lock [<minecraftVersion>] [<loader>]
       mcmod lock list <minecraftVersion> <loader>
       mcmod lock show <minecraftVersion> <loader> [<key>]
       mcmod lock add <minecraftVersion> <loader> <key> [flags]
       mcmod lock update <minecraftVersion> <loader> [<key>] [flags]
       mcmod lock delete <minecraftVersion> <loader> [<key>]
       mcmod lock tree <minecraftVersion> <loader>
       mcmod lock release set <minecraftVersion> [<loader>] [flags]
       mcmod lock release list <minecraftVersion>
       mcmod lock release show <minecraftVersion> <version> [<loader>]
       mcmod lock release delete <minecraftVersion> <version> [<loader>] [--target client|server]
```

Lock files are written to `locks/dependencies/<mc>-<loader>.json`.

#### `mcmod lock list <mc> <loader>`

Print the lock contents grouped by scope (`[Server]`, `[Client]`, `[Shared]`).

```text
$ mcmod lock list 1.21.1 neoforge
lock 1.21.1 neoforge
[Server]
  c | C | 3 | local | c.jar
[Client]
  b | B | 2 | local | b.jar
[Shared]
  a | A | 1 | local | a.jar
```

#### `mcmod lock show <mc> <loader> [<key>]`

Without `key`, dump the lock file as JSON. With `key`, print a single entry.

```text
$ mcmod lock show 1.21.1 neoforge a
key: a
name: A
version: 1
scope: shared
source:
  type: local
  fileName: a.jar
```

#### `mcmod lock add <mc> <loader> <key> [flags]`

| Flag | Description |
|---|---|
| `--name`        | Display name |
| `--version`     | Mod version |
| `--source`      | `local`, `curseforge`, or `github-release` |
| `--path`        | Local path (for `local`) |
| `--file-name`   | Destination file name in zip |
| `--mod-id`      | CurseForge mod ID |
| `--file-id`     | CurseForge file ID |
| `--repo`        | GitHub `owner/repo` |
| `--tag`         | GitHub release tag |
| `--asset-name`  | GitHub asset file name |

Errors if `<key>` already exists in the lock.

#### `mcmod lock update <mc> <loader> [<key>] [flags]`

Without `<key>`, refresh the lock from `packspec.json` (re-resolve every mod).
With `<key>`, update fields for that key (only the supplied flags change).

| Flag | Description |
|---|---|
| `--version` | New version |

#### `mcmod lock delete <mc> <loader> [<key>]`

- With `<key>`: remove only that entry.
- Without `<key>`: remove the entire lock file.

#### `mcmod lock tree <mc> <loader>`

```text
$ mcmod lock tree 1.21.1 neoforge
dependency tree 1.21.1 neoforge
A [shared] v1 (local)
```

#### `mcmod lock release set <mc> [<loader>] [flags]`

Create or update a release entry under `locks/releases/<mc>.json`.

| Flag | Description |
|---|---|
| `--version`         | Pack version (required) |
| `--repo`            | GitHub `owner/repo` (required) |
| `--tag`             | Release tag (required) |
| `--name`            | Release display name |
| `--body`            | Release notes |
| `--draft`           | Mark as draft |
| `--prerelease`      | Mark as prerelease |
| `--artifact-client` | Client zip file name |
| `--artifact-server` | Server zip file name |

If `<loader>` is supplied, attach artifacts to that loader only.

#### `mcmod lock release list <mc>`

```text
$ mcmod lock release list 1.21.1
releases 1.21.1
  0.1.0 [github-release] tag=v0.1.0
```

#### `mcmod lock release show <mc> <version> [<loader>]`

Print a single release entry.

#### `mcmod lock release delete <mc> <version> [<loader>] [--target client|server]`

- Without `--target`: remove the whole version entry.
- With `--target client|server`: remove only that artifact under the given loader.

If the version does not exist, the command errors out.

### `mcmod build`

Build modpack artifacts.

```text
Usage: mcmod build [<minecraftVersion> [<loader>]] [--target client|server|both]
                       [--build-type cf|all] [--force]
                       [--mc-version <mc>] [--loader <loader>]
```

| Flag | Description |
|---|---|
| `--target`     | `client`, `server`, or `both` (default: `both`) |
| `--build-type` | `cf` (CurseForge modpack layout) or `all` (default mcmod zip layout, default) |
| `--force`      | Overwrite existing zips |
| `--mc-version` | Override packspec `minecraftVersion` |
| `--loader`     | Override packspec loader |

Default (`--build-type all`) output goes to
`releases/v<packVersion>/<packName>-<mc>-<loader>-<loaderVersion>-<target>.zip`
and the matching `<serverPackName>-...-server.zip`.

`--build-type cf` produces a single CurseForge modpack zip at
`releases/v<packVersion>/<packName>-<mc>-<loader>-<loaderVersion>-cf.zip`.
The zip contains three categories of content:

- `manifest.json` — CurseForge modpack manifest (snake_case schema; see
  `internal/service/build_zip_cf.go` for the field set). One entry per
  shared+client mod whose source is `curseforge` AND whose lock entry
  carries non-zero `modId` and `fileId`. Sorted by
  `(projectID, fileID)` for a stable on-disk artifact. The launcher
  downloads these jars at import time from `manifest.files[]`.
- `modlist.html` — human-readable HTML summary, one bullet per
  shared+client mod. Curseforge mods link to the CurseForge project
  page; non-cf mods are annotated with the path inside
  `overrides/mods/` so the operator can verify the bundle.
- `overrides/config/`, `overrides/resourcepacks/` — copied from the
  project root when those directories exist. Per the CurseForge schema,
  anything that should land in the user's `.minecraft/` root goes under
  `overrides/`.
- `overrides/mods/<key>/<jar>` — every shared+client mod whose source
  is NOT curseforge (i.e. `github-release`, `git`, `local`, `url`).
  The launcher picks jars up from `overrides/mods/` when
  `manifest.files[]` cannot resolve them, so non-cf mods stay
  installed after import. Each mod lands in its own
  `<key>/` subdirectory so two mods with the same file name do
  not collide.

`--build-type cf` is single-target: it ignores `--target` and always
emits the client flavor (CurseForge publishes a single client pack; the
server zip is an mcmod-only concept). Server-scope mods are silently
omitted. Curseforge mods whose lock entry is missing `modId` or
`fileId` are reported to stderr and dropped from the manifest. If a
non-cf mod's jar cannot be resolved on disk (cache miss plus download
failure, or a local path that does not exist) the build fails with a
hard error: a partial zip that silently drops a mod is worse than a
failure, because the user would not notice the missing mod until they
opened the modpack in-game. The build still requires at least one
curseforge mod to be eligible so `manifest.json` stays a valid CF
import payload.

### `mcmod tree`

Alias for `mcmod lock tree <mc> <loader>`.

```text
$ mcmod tree 1.21.1 neoforge
dependency tree 1.21.1 neoforge
A [shared] v1 (local)
```

### `mcmod config`

Inspect or set the API key for the current project.

```text
$ mcmod config
CurseForge API key: <key>   # or (not set)

$ mcmod config set-cf-key <key>
config: CurseForge API key saved
```

### `mcmod version`

Print the version string.

```text
$ mcmod version
mcmod version 0.1.0
```

## Exit Codes

- `0` on success.
- `1` on validation, I/O, or resolver error.
