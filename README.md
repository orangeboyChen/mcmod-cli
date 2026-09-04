<!--
File: README.md
Created: 2026-06-20
Description: mcmod CLI tool documentation (English).
-->

# mcmod

> [中文文档](./README.zh-CN.md)

A Go CLI for managing Minecraft modpack specifications, dependency locks,
jar resolution, downloading, build artifacts, and release indexes. The root
`packspec.json` is the single human-editable source of truth; the CLI turns
it into per-loader lock files, signed zips, and a release index.

## Features

- `packspec.json` as the only human-edited input
- Loaders: **NeoForge** and **Fabric** (mod-level loader filter supported)
- Sources: **CurseForge** (query), **GitHub Release** (tag + assetPattern),
  **Git** (downstream mcmod package), **Local** (template path)
- `mcmod lock` resolves sources, runs an incremental `kept / added / removed
  / failed` reconciliation against the existing lock, and writes
  `locks/dependencies/<mcVersion>-<loader>.json`
- `mcmod build` reads the lock, runs jar metadata validation (missing
  required deps, class conflicts), and produces client / server zips
- `mcmod lock release` maintains `locks/releases/<mcVersion>.json`
- `mcmod tree` renders the resolved dependency tree
- Cross-platform binaries via `go install` or GitHub releases
- The short executable name `mcm` is also available (`go install ./cmd/mcm`).

## Quick Start

```bash
# 1. Configure the CurseForge API key (user-level, recommended)
mcmod set cf-key <your-key>

# 2. Edit packspec.json
cat > packspec.json <<'JSON'
{
  "packName": "my-pack",
  "serverPackName": "my-pack-server",
  "packVersion": "0.1.0",
  "minecraftVersion": "1.21.1",
  "loaderName": ["neoforge:21.1.219"],
  "mods": {
    "jei": {
      "name": "Just Enough Items",
      "scope": "client",
      "source": { "type": "curseforge", "query": "Just Enough Items" }
    },
    "create": {
      "name": "Create",
      "scope": "shared",
      "source": { "type": "curseforge", "query": "Create" }
    }
  }
}
JSON

# 3. Resolve and lock the dependencies
mcmod lock 1.21.1 neoforge

# 4. Build client + server zips
mcmod build 1.21.1 neoforge
```

Outputs land in `locks/dependencies/1.21.1-neoforge.json` and
`releases/v0.1.0/my-pack-1.21.1-neoforge-21.1.219-{client,server}.zip`.

## Commands

| Command                                 | Description                                                  |
|-----------------------------------------|--------------------------------------------------------------|
| `mcmod set cf-key <key> [--project]`    | Configure the CurseForge API key.                             |
| `mcmod list`                            | List mods from `packspec.json` grouped by scope.              |
| `mcmod lock [<mc>] [<loader>]`          | Resolve / incrementally update the dependency lock.          |
| `mcmod lock list\|show\|add\|update\|delete\|tree` | Manage lock files and entries.                        |
| `mcmod lock release set\|list\|show\|delete`      | Maintain `locks/releases/<mcVersion>.json`.         |
| `mcmod build [<mc>] [<loader>]`         | Build client and server zips from the lock.                   |
| `mcmod tree [<mc>] [<loader>]`          | Render the resolved dependency tree.                         |
| `mcmod validate`                        | Validate `packspec.json`, lock, or release index.            |
| `mcmod --help`                          | Show the full command tree (see [docs/000-index.md](./docs/000-index.md)). |

Run `mcmod --help` for the full command tree with positional argument
placeholders.

## Project Layout

```text
packspec.json                       # human-edited spec
locks/
  dependencies/<mc>-<loader>.json   # resolved lock files
  releases/<mc>.json                # build release index
releases/                           # built zips (gitignored)
.cache/                             # jar download cache (gitignored)
.mcmod/                             # project-level CLI config: cfKey (gitignored)
.cache/resolved/                    # resolver id cache: mod key -> modId/fileId (gitignored)
internal/
  cli/                              # cobra commands
  domain/                           # data models, validation, store
  resolver/                         # source resolution (CF, GitHub, Git, Local)
  downloader/                       # jar downloader with cache
  metadata/                         # jar metadata (NeoForge / Fabric TOML, fabric.mod.json)
  graph/                            # dependency graph + version resolution
  service/                          # business logic (lock, build, release, tree)
  cache/                            # cache helpers
  config/                           # user / project / env config
cmd/mcmod/                          # CLI entry point
```

## Documentation

- [docs/000-index.md](./docs/000-index.md) — documentation index and
  reading order.
- [docs/001-spec.md](./docs/001-spec.md) — `packspec.json` schema and
  rules.
- [docs/002-cli-overview.md](./docs/002-cli-overview.md) — command surface
  and exit codes.
- [docs/003-lock-files.md](./docs/003-lock-files.md) — dependency lock
  format and reconciliation rules.
- [docs/004-release-index.md](./docs/004-release-index.md) — release index
  format.
- [docs/005-source-resolution.md](./docs/005-source-resolution.md) — how
  sources are resolved.
- [docs/008-build-pipeline.md](./docs/008-build-pipeline.md) — build
  pipeline and validation rules.

## TODO

`mcmod build` only packages the `mods/` directory today. The following
modpack artifacts are intentionally out of scope and not yet generated:

- **Mods** — fully supported. See `packspec.json` schema and lock/release
  docs.
- **Shaderpacks** (`shaderpacks/`) — not supported. No source type, no
  resolver, no zip entry. There is no way to declare or pin a shaderpack
  in `packspec.json` yet.
- **Resourcepacks** (`resourcepacks/`) — the client zip copies a project-root
  `resourcepacks/` directory verbatim when it exists, but the resolver and
  lock pipeline do not yet track packspec-level resourcepack entries.
- **Datapacks** (`datapacks/`) — not supported.
- **World saves** — not supported.
- **CurseForge modpack layout** (`manifest.json` + `modlist.html` +
  `overrides/{config,resourcepacks}/`) — implemented. `mcmod build
  --build-type cf` produces a manifest-only zip (no per-mod jars; the
  launcher downloads them at import time from `manifest.files[]`).
  See `docs/002-cli-overview.md` for the exact contract.
- **Modrinth `.mrpack` layout** — reserved by the docs but not yet
  accepted by the CLI.

Track these in issues; PRs that close any of them must update
`docs/002-cli-overview.md` and `AGENTS.md` in the same change per the
"docs win" rule.

## Building

```bash
go build ./cmd/mcmod
go test ./...
```

Requires Go 1.26+.

## License

MIT
