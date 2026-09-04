<!--
File: README.md
Created: 2026-09-04
Description: mcmod CLI project documentation.
-->

# mcmod

> [中文文档](./README.zh-CN.md)

`mcmod` is a Go CLI for Minecraft modpack specifications, recursive Git
packspec dependencies, lock files, downloads, builds, and release indexes.

## Quick start

```bash
mcmod set cf-key <your-curseforge-api-key>
mcmod lock 1.21.1 neoforge
mcmod build 1.21.1 neoforge
```

Builds validate every selected mod JAR for duplicate classes, unreadable files,
and missing required loader metadata dependencies before creating output. The
error report lists all causes together, and failed builds leave no new artifact.
CurseForge builds may download client JARs on their first run for this check.

The editable input is `packspec.json` at the project root. Resolved locks are
written to `locks/dependencies/`; build artifacts are written to `releases/`.

`mcm version`, `mcm -v`, and `mcm --version` print the hard-coded CLI version.
Stable and beta release workflows build native Linux amd64/arm64, Windows
amd64, and macOS amd64/arm64 archives. Stable bumps create a labeled PR;
beta publishes a `vX.Y.Z-canary.N` prerelease. Both accept an optional
`base_version` (`x.y.z`), which is used directly when supplied.

## Recursive Git packspecs

A Git source points to a repository containing its own `packspec.json`:

```json
{
  "mods": {
    "bundle": {
      "source": {"type": "git", "repo": "owner/repository"}
    }
  }
}
```

`mcmod lock` recursively reads nested `packspec.json` files, applies loader
filters and inherited scope, namespaces expanded keys by repository, and
detects cycles and key conflicts. Git repositories are packspec inputs; their
`packspec.json` is read directly and no JAR is unpacked for recursion.

## Project state

```text
packspec.json
locks/dependencies/<minecraft>-<loader>.json
locks/releases/<minecraft>.json
.mcmod/config.json                  # project CurseForge key (gitignored)
.mcmod/cache/                       # downloads and resolver IDs (gitignored)
.mcmod/cache/curseforge/...
.mcmod/cache/github-release/...
.mcmod/cache/resolved/...
```

The CLI only uses the current project directory for configuration and cache.
It does not read `HOME`, `XDG_CONFIG_HOME`, or a user/global config path.

## Commands

```text
mcmod list
mcmod lock [minecraftVersion] [loader]
mcmod tree [minecraftVersion] [loader]
mcmod build [minecraftVersion] [loader] [--target client|server|both]
mcmod validate
mcmod set cf-key <key>
mcmod config set-cf-key <key>
mcmod version
```

See [`docs/000-index.md`](./docs/000-index.md) for the complete contract.

## Development

```bash
go mod tidy
gofmt -w $(find . -name '*.go' -not -path './.mcmod/*')
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go build ./cmd/mcmod
```

Requires Go 1.26+.
