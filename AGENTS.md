<!--
File: AGENTS.md
Created: 2026-06-20
Description: Repository instructions for agents working on mcmod.
-->

# AGENTS.md

## Project Purpose

This repository contains `mcmod`, a Go CLI for managing Minecraft modpack specifications, dependency locks, jar resolution/download, build artifacts, and release indexes.

The root `packspec.json` is the editable source of truth. The root `locks/` directory contains stable lock results that can be committed. Runtime caches and build outputs are generated files.

## Hard Rules

1. Do not reintroduce legacy `mod` / `entry` / `package` command separation.
2. Do not add spec CRUD commands such as `mcmod add`, `mcmod show`, `mcmod update`, or `mcmod delete`.
3. Do not reintroduce MCP support. `mcmod` is a CLI.
4. Do not write CurseForge `modId`, `fileId`, `fileName`, GitHub `assetName`, or download URLs back into `packspec.json`.
5. Do not ignore or delete `locks/`; lock files are stable project outputs.
6. Do not commit `.cache/`, `.mcmod/`, `releases/`, archives, coverage files, or local binaries.

## Commit Requirements

All commit messages must use Conventional Commits.

Allowed examples:

```text
feat(cli): add dependency lock resolver
fix(build): reject missing required mod dependency
docs(spec): clarify curseforge download flow
test(resolver): cover github release asset matching
chore(ci): add release checksum generation
```

Do not commit until lint and tests pass.

Required commands before commit:

```bash
go mod tidy
gofmt -w $(find . -name '*.go' -not -path './.cache/*')
golint ./...
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go build ./cmd/mcmod
```

The total statement coverage must be at least `80.0%`.
Review `go.mod` and `go.sum` after `go mod tidy`; do not keep unrelated dependency changes.

## Linting

`golint` is required for this project.

Install it if missing:

```bash
go install golang.org/x/lint/golint@latest
```

CI and release workflows must run `golint ./...` before publishing or merging.

## Comments

All code comments must be written in English.

Every newly created source, script, workflow, or config file must start with a file header comment. The `Created` date must use the current date from the environment.

Go file header template:

```go
// File: internal/example/example.go
// Created: 2026-06-20
// Description: Implements example behavior for mcmod.
```

Shell file header template:

```sh
# File: scripts/example.sh
# Created: 2026-06-20
# Description: Runs an example mcmod validation workflow.
```

YAML file header template:

```yaml
# File: .github/workflows/example.yml
# Created: 2026-06-20
# Description: Runs an example GitHub Actions workflow for mcmod.
```

Markdown file header template:

```markdown
<!--
File: docs/example.md
Created: 2026-06-20
Description: Documents an example mcmod workflow.
-->
```

For existing files without headers, add a header only when the edit is substantial and the file format supports comments safely.

## GitHub Actions And Release

CI and release workflows must stay aligned with the repository docs and implemented CLI behavior.
GitHub Actions should cache Go modules and Go build cache, but must not cache project `.cache/`, `.mcmod/`, `releases/`, release archives, coverage output, or secrets.

The CLI release assets must use these names:

```text
mcmod_cli_<version>_linux_amd64.tar.gz
mcmod_cli_<version>_linux_arm64.tar.gz
mcmod_cli_<version>_windows_amd64.zip
mcmod_cli_<version>_darwin_arm64.tar.gz
mcmod_cli_<version>_checksums.txt
```

Release builds must be triggered by `v*` tags and must only publish after lint, tests, coverage, and build checks pass.

## Documentation

README and docs must stay aligned with:

1. `packspec.json` schema.
2. `locks/dependencies/` schema.
3. `locks/releases/` schema.
4. CLI commands and help output.
5. Jar resolver/downloader behavior.
6. Build output paths and names.
7. GitHub Actions release artifact names.

Docs under `docs/` must use numbered file names such as `000-index.md`, `001-spec.md`, and `002-cli-overview.md`. These docs are primarily for AI agents, so they must be explicit, self-contained, and include commands, expected outputs, file paths, JSON fields, and error examples.
