<!--
File: AGENTS.md
Created: 2026-06-20
Description: Repository instructions for agents working on mcmod.
-->

# AGENTS.md

This file is the contract between the human maintainers and any agent
(AI or otherwise) editing this repository. It is the **single source of
truth** for project rules; if a rule is not listed here, it is not a rule.

## Project Purpose

`mcmod` is a Go CLI for managing Minecraft modpack specifications,
dependency locks, jar resolution/download, build artifacts, and release
indexes.

- Root `packspec.json` is the editable source of truth.
- Root `locks/` contains stable lock results that **can be committed**.
- `.cache/`, `.mcmod/`, `releases/`, `dist/`, `coverage.out`, and local
  binaries are generated artifacts. **Never commit them.**

## AI Agent Workflow (Read Before Doing Anything)

Before editing any file, an agent MUST read, in this order:

1. `AGENTS.md` (this file) — rules of engagement.
2. `docs/000-index.md` — full documentation table of contents.
3. The doc(s) covering the area you are about to touch:
   - Spec, packspec, mod sources → `docs/001-spec.md`, `docs/005-source-resolution.md`, `docs/006-mod-key-normalization.md`, `docs/007-scope-and-loader.md`
   - Lock files → `docs/003-lock-files.md`
   - Release index → `docs/004-release-index.md`
   - CLI commands / flags / output → `docs/002-cli-overview.md`
   - Build pipeline → `docs/008-build-pipeline.md`, `docs/010-downloader.md`
   - Graph / cycle detection → `docs/009-graph-and-resolution.md`
   - Config & API key resolution → `docs/011-configuration.md`
   - Validation rules → `docs/013-validation.md`
   - Metadata parsing (NeoForge/Fabric jars) → `docs/012-metadata-parsing.md`
   - Testing conventions, smoke tests, coverage → `docs/014-testing.md`
4. The relevant Go package's existing tests and `*_test.go` before adding
   a new behavior, to match style and table layout.
5. `README.md` and `README.zh-CN.md` if the change is user-visible.

If docs and code disagree, **docs win**; fix the code and update the
relevant doc on the same change. Never ship a change that contradicts an
existing `docs/NNN-*.md` file.

## Hard Rules

1. Do not reintroduce legacy `mod` / `entry` / `package` command separation.
2. Do not add spec CRUD commands such as `mcmod add`, `mcmod show`, `mcmod update`, or `mcmod delete`.
3. Do not reintroduce MCP support. `mcmod` is a CLI, not a server.
4. Do not write CurseForge `modId`, `fileId`, `fileName`, GitHub `assetName`, or download URLs back into `packspec.json`.
5. Do not ignore or delete `locks/`; lock files are stable project outputs.
6. Do not commit `.cache/`, `.mcmod/`, `releases/`, `dist/`, archives, coverage files, local binaries, or `.env`.
7. Do not introduce new top-level dependencies without explicit maintainer approval. Re-evaluate `go.mod`/`go.sum` after `go mod tidy` and revert unrelated changes.
8. Do not add `//nolint:`, `//nolintlint:`, file-level `//go:build` workarounds, or `.golangci.yml` exclusions to silence a finding. **Fix the root cause.**
9. Do not use `panic` for control flow. Do not use `fmt.Println` for user-facing output in `internal/cli/...`. Use the helpers in `internal/cli` and write errors to `cmd.ErrOrStderr()`.
10. Do not introduce `init()` with side effects outside `internal/testutil`.
11. Do not use the `os` package's `Getenv` directly for `CURSEFORGE_API_KEY`; use `internal/config`.
12. Do not log API keys, lock file paths containing user home dirs, or raw jar bytes. Use `internal/netutil` for any HTTP I/O.
13. Do not use `context.Background()` in CLI command bodies without justification. Prefer the cobra command's `cmd.Context()`.

## Repository Layout

```
cmd/mcmod/                # main entry + main_test.go
internal/
  cache/                  # .cache/ on disk (SHA256, jar cache)
  cli/                    # cobra commands, help template, output helpers
  config/                 # CURSEFORGE_API_KEY resolution (env > project > user)
  domain/                 # packspec, lock, release models + validation
  downloader/             # jar download + integrity verification
  graph/                  # dependency graph + cycle detection
  metadata/               # NeoForge / Fabric jar readers
  netutil/                # HTTP client, retries, timeouts
  resolver/               # CurseForge, GitHub release, git, local, url
  service/                # orchestration: lock, build, release, tree
  testutil/               # shared Ginkgo/Gomega helpers, tmpdir, fake_http
test/                     # end-to-end Ginkgo smoke + integration specs
docs/                     # 000-index.md + 001..014 topic files
packspec.json             # editable source of truth
locks/                    # commit-able lock and release outputs
.golangci.yml             # lint config — single source of truth for linting
```

Rules:

- New CLI subcommands go under `internal/cli/`. Reuse `newXxxCmd()` and
  register on the root command via `init()` in that file only.
- Reusable orchestration goes under `internal/service/`. Pure logic that
  has no I/O goes under `internal/domain/`.
- Anything that talks to the network goes through `internal/netutil`.
  Do not call `net/http` from resolvers, downloaders, or CLI directly.
- Shared test helpers go in `internal/testutil`. Do not duplicate fixtures
  in each `*_test.go` file.

### Architecture Layering

The internal package graph forms a strict DAG. Lower layers know nothing
about higher layers. The graph, in dependency order from innermost to
outermost:

```
  internal/domain            <-- no internal imports
  internal/cache
  internal/metadata
  internal/config
  internal/graph
  internal/netutil
  internal/resolver
  internal/downloader
  internal/service           <-- orchestrates everything below
  internal/cli               <-- depends on service
  internal/testutil          <-- standalone helpers, may import any layer
```

Rules:

- `internal/domain` MUST NOT import any other `internal/...` package. It
  defines data shapes and pure functions over them.
- `internal/{cache,metadata,config,graph,netutil}` may import `domain` and
  each other. They MUST NOT import `service`, `cli`, `resolver`, or
  `downloader`.
- `internal/resolver` and `internal/downloader` may import `domain`,
  `cache`, `metadata`, `config`, `graph`, `netutil`. They MUST NOT
  import `service` or `cli`.
- `internal/service` may import everything below it. It MUST NOT import
  `internal/cli`.
- `internal/cli` may import `service` and below. It MUST NOT be imported
  by anything except `cmd/mcmod`.
- A new package MUST be slotted into the right layer and update this
  diagram. If a lower layer needs a helper from a higher layer, push the
  helper down — do not introduce an upward import.

Violations are caught by `golangci-lint` (`forbidigo`, `depguard`) and
the pre-commit checks. See `docs/000-index.md` for the package-by-package
responsibility list.

## Coding Style (Go)

### General

- Go version: the version pinned in `go.mod` (currently `go 1.26`). Use
  features appropriate to that toolchain; do not require a newer one.
- All identifiers, comments, log messages, and user-facing strings are
  **English only**. Translations live in `README.zh-CN.md` only.
- One package per directory. Package names are short, lowercase, no
  underscores, no stutter (`cli`, not `mcmodcli`).
- File names are `snake_case.go`. Test files are `snake_case_test.go`.
- Every new or substantially edited file MUST start with the header
  template from the **Comments** section below, using today's date from
  the environment for `Created`.
- Prefer the standard library. Pulling in a new dependency is a Hard Rule
  violation unless pre-approved.
- No empty interfaces. Define a typed contract instead.
- Prefer composition. Avoid deep inheritance / embedding chains.

### Imports

- Group imports in this order, separated by a blank line:
  1. Standard library
  2. Third-party
  3. `github.com/orangeboyChen/mcmod-cli/internal/...`
- `goimports` / `gofmt` is the authority on import ordering. Do not
  hand-format imports.

### Errors

- Wrap with `fmt.Errorf("action: %w", err)` so the chain is traceable.
  Prefix the wrap with the verb phrase ("read packspec", "download jar").
  Lowercase first word, no trailing period, use `\n` for the
  multi-line "hint:" form.
- Hint format for user-recoverable errors in CLI commands:
  `return fmt.Errorf("lock: read packspec\nhint: create packspec.json in the project root")`.
- Never return a bare `error` from cobra's `RunE` without a wrap.
- `errors.Is` / `errors.As` are required when checking wrapped errors.
- Sentinel errors live in `internal/domain` or the package that owns
  them; do not re-declare sentinels in callers.

### Naming

- Exported identifiers get a doc comment starting with the identifier
  name. Example: `// ReadPackSpec parses the packspec.json at root.`
- Acronyms are all-caps: `URL`, `ID`, `SHA256`, `HTTP`, `JSON`, `TOML`.
  But mod source type constants in `internal/domain/common.go` use the
  values spelled out (`SourceCurseForge = "curseforge"`,
  `SourceGitHubRelease = "github-release"`) — follow that precedent.
- Use `mcVersion` / `loader` (not `mcVer` / `l`). Follow existing
  parameter names in the file you are editing.
- Receiver names are short (1–2 letters) and consistent across a type.
- Avoid stutter in identifiers: `cli.NewApp`, not `cli.NewCliApp`.

### CLI Conventions

- Subcommand construction returns `*cobra.Command`. The factory is
  `newXxxCmd()` and is wired in the same file's `init()`.
- `SilenceErrors` and `SilenceUsage` are `true` on the root command
  (see `internal/cli/app.go`). Do not flip them.
- `DisableAutoGenTag` is `true`. Do not add the "Auto generated by
  spf13/cobra" comment.
- Persistent flags live on the root. Per-command flags stay local.
- Flag names use kebab-case (`--mc-version`, `--build-type`).
- Help output goes through the spec 7.7 template in `internal/cli/help.go`.
  Do not call `cmd.UsageString()` from non-help code paths.
- Errors in CLI commands go to `cmd.ErrOrStderr()`. Success output goes
  to `cmd.OutOrStdout()`. Never write to `os.Stdout` / `os.Stderr` from
  `internal/cli`.
- Exit codes: `0` success, `1` on validation, I/O, or resolver error.
  Match `mcmod` runtime contract — do not introduce new exit codes.
- Hidden persistent flags use `MarkHidden` only on flags the user should
  never pass explicitly (e.g. the merged `-h, --help`).

### JSON & File I/O

- All on-disk JSON uses `encoding/json` with struct tags matching
  `docs/001-spec.md`, `docs/003-lock-files.md`, and `docs/004-release-index.md`.
- Field names are stable once published. Adding a field is fine; renaming
  or removing one requires a doc + spec update in the same change.
- For "lock file round-trip" code paths, parse then re-marshal the
  example in `docs/003-lock-files.md` and diff.
- Use `internal/cache` for any `.cache/` writes. Do not call
  `os.MkdirAll(".cache/...")` from CLI or service code.

### Concurrency & Context

- Long-running calls (HTTP, file I/O across many files) take a
  `context.Context` as the first parameter.
- Cobra commands use `cmd.Context()`. Add a timeout in the resolver or
  downloader if a single call can hang.
- Channel sends from a single goroutine to a receiver must use
  `select { case ch <- v: default: }` or a done channel — never block
  the sender on an unguarded send.

### Dependencies

- `go mod tidy` must be a no-op for unrelated modules after your change.
  If `go.sum` grows, justify every line in the PR description.
- Vendoring is disabled. The repo relies on the module cache.
- Pinned to the version in `go.mod` (currently `go 1.26`).

## Testing

- Framework: Ginkgo v2 + Gomega. Test files follow Ginkgo layout
  (`var _ = Describe(...)`, `BeforeEach`, `It`, `Specify`).
- In-process CLI tests use `internal/testutil` and `chdirTemp()` to keep
  `~/.config/mcmod/` clean.
- Subprocess smoke tests in `test/` build the binary once in
  `BeforeSuite` and run each `It` in a fresh `t.TempDir()`.
- Subprocess env is always isolated: `HOME=<d>`, `XDG_CONFIG_HOME=<d>`,
  `CURSEFORGE_API_KEY=""` cleared. Never inherit the host environment.
- Total statement coverage must stay ≥ **80.0%**. New packages and new
  files should be ≥ 80% too; the maintainer will ask for tests when a
  new public function is added without a corresponding `It` block.
- Random Ginkgo order is the default. If a CI flake appears, re-run
  with `go test -count=1` and/or `-ginkgo.seed=N` to bisect.
- HTTP tests use `internal/testutil/fake_http.go`. Do not start a real
  listener per test; reuse the helper.
- Per-package statement coverage target: **>= 85%**. Total project
  coverage target: **>= 85%** (sums to >= 85 even when helpers like
  `internal/testutil` count as 0%).

### Test File Layout (1:1 source<->test)

Every non-test Go file `foo.go` lives next to a test file `foo_test.go`
covering its behavior. To keep Ginkgo happy while honouring the 1:1
rule, follow this layout per package:

- One `<pkg>_suite_test.go` that contains ONLY the suite entry point and
  shared `BeforeSuite` / `AfterSuite` hooks:

  ```go
  func TestPkg(t *testing.T) {
      RegisterFailHandler(Fail)
      RunSpecs(t, "Pkg Suite")
  }
  ```

- One `foo_test.go` per source file `foo.go`. Each `Describe` block
  inside it covers a function or logical group of functions from
  `foo.go`. Cross-source scenarios (e.g. dispatcher -> sub-resolver)
  live in the test file of the source that owns the entry point.

- Optional `<pkg>_helpers_test.go` for shared test helpers (no
  `Describe` blocks). Constants and builders used by multiple
  `foo_test.go` files go here, not in a single source's test.

- Reusing a Ginkgo `Describe` name across files in the same package is
  fine; reusing an `It` name is NOT. Each `It` must have a unique
  label across the whole package.

- Splitting rules:
  - Source files that contain a single, focused function (e.g.
    `internal/domain/normalize.go`, `internal/resolver/local.go`) keep
    one `*_test.go` next to them. Do not split tiny files into test
    suites per function.
  - Mega-test files (>500 lines, mixing multiple sources) MUST be
    split so each source has its own `*_test.go`. A split is correct
    when `grep -c '^var _ = Describe' <file>` returns 1 per source.
  - When a `Describe` block is moved across files, also move the
    `var _ = Describe(...)` opening line; do not leave an empty
    `Describe("...")` shell behind.

- Each `*_test.go` file MUST start with the standard file header (see
  Comments section) and a one-line description naming the source file
  it covers.

### Coverage Discipline

- Coverage numbers are a floor, not a goal. A test that just bumps the
  counter is worse than no test. Each new `It` must exercise a
  meaningful code path or document a real error case.
- When a function has a network or filesystem branch that is hard to
  exercise in unit tests, prefer the existing helpers in
  `internal/testutil` (fake HTTP server, `MkdirTemp`) over skipping the
  branch.
- The 85% per-package target applies to every package in `internal/`
  except `internal/testutil` (which is helper-only and explicitly
  excluded). A package below the target is a release blocker.

## Commit Requirements

All commit messages must use Conventional Commits.

Allowed examples:

```text
feat(cli): add dependency lock resolver
fix(build): reject missing required mod dependency
docs(spec): clarify curseforge download flow
test(resolver): cover github release asset matching
chore(ci): add release checksum generation
refactor(domain): extract scope validator
perf(downloader): cache sha256 by file id
```

Do not commit until lint and tests pass.

Required commands before commit (in this order):

```bash
go mod tidy
gofmt -w $(find . -name '*.go' -not -path './.cache/*')
golangci-lint fmt ./...                # applies gofmt + goimports
golangci-lint run ./...                # 0 issues
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go build ./cmd/mcmod
```

`golangci-lint fmt --check ./...` (i.e. "no fixes needed") and
`golangci-lint run ./...` (i.e. "0 issues") must both pass. If a finding
is reported, fix the code (check the error, use a `defer` cleanup,
remove dead code, simplify the expression, replace a literal with a
named constant, add a test) rather than suppressing the linter. If you
believe a specific finding is a true false positive, discuss it in the
PR rather than adding an inline or config exclusion.

`golangci-lint run ./...` must finish with **0 issues**. A non-zero exit
code or any reported issue blocks the commit. Fix the code (check the
error, use a `defer` cleanup, remove dead code, simplify the expression,
add a test) rather than suppressing the linter.

The total statement coverage must be at least `80.0%`. If the total
drops below 80%, add or extend tests in the same change.

Review `go.mod` and `go.sum` after `go mod tidy`; do not keep unrelated
dependency changes.

## Linting

`golangci-lint` is required for this project and is the single source
of truth for lint enforcement. The configuration lives in
`.golangci.yml` at the repository root.

The target is **0 warnings and 0 errors** from
`golangci-lint run ./...`. Any new code or change to existing code must
keep the project at zero issues. Do not add exclusions, `//nolint:`
directives, or per-line suppressions to make a finding disappear; fix
the underlying problem instead.

### Tier: "standard+ + style + security + complexity"

This project uses a curated bundle of linters across six categories.
The authoritative list lives in `.golangci.yml`. The categories and
rationale:

| Category            | Linters                                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------------------- |
| **Style**           | `revive`, `gocritic`, `godot`, `misspell`, `canonicalheader`, `asciicheck`, `bidichk`, `errname`, `err113`, `testpackage` |
| **Correctness**     | `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `nilerr`, `errorlint`, `bodyclose`, `contextcheck`, `fatcontext`, `containedctx`, `exhaustive`, `durationcheck`, `makezero`, `nosprintfhostport`, `gocheckcompilerdirectives`, `exptostd`, `decorder`, `forbidigo` |
| **Test discipline** | `paralleltest`, `ginkgolinter`                                                                       |
| **Performance**     | `prealloc`, `perfsprint`                                                                             |
| **Complexity**      | `cyclop` (max 15 / package avg 10), `gocognit` (min 20), `dupl` (threshold 80 tokens)                 |
| **Hygiene**         | `goconst`, `unparam`, `copyloopvar`                                                                  |
| **Security**        | `gosec` (severity/confidence medium)                                                                 |
| **Dependencies**    | `depguard` — enforces Hard Rule #7; denies `pkg/errors`, `yaml.v2`, `yaml.v3`, `testify`, `go-spew`, `go-difflib` |

In addition, the **`formatters`** block (v2-only concept) runs:

- `gofmt`
- `goimports` — with `local-prefixes: github.com/orangeboyChen/mcmod-cli`
  so the import grouping matches **Coding Style / Imports**.

#### Style consistency guarantees

The Style-category linters together enforce:

- Every exported identifier has a doc comment starting with its name
  (`revive: exported`).
- Sentinel errors are `Err*`; error types end in `Error` (`errname`).
- Comments end in a period, with file-scope comments required to be
  sentences (`godot: scope: declarations`).
- Identifiers are pure ASCII (`asciicheck`) and free of bidi tricks
  (`bidichk`).
- HTTP header keys are canonical (`canonicalheader`).
- English-only spelling with a project-specific allowlist (`misspell`).
- Errors are wrapped with `%w`, asserted via `errors.As`, and compared
  with `errors.Is` (`errorlint`).
- Errors are constructed with `errors.New` or wrapped with `fmt.Errorf`,
  not hand-built strings (`revive: use-errors-new`, `use-fmt-print`,
  `err113`).
- Tests live in `package foo_test` (white-box tests inside `package foo`
  are allowed but a black-box test file is required for each public
  package boundary) (`testpackage`).

#### `forbidigo` patterns (Hard Rule #9 enforcement)

`forbidigo` enforces Hard Rule #9 ("no `panic` / `log.Fatal` / `os.Exit`
outside `cmd/mcmod/main.go`; no `fmt.Println` / `fmt.Print` /
`log.Println` / `log.Print` from `internal/cli`"). The patterns are
listed in `.golangci.yml` under `linters.settings.forbidigo.forbid`.

### Failure budget

The `issues` block sets `max-issues-per-linter: 0` and
`max-same-issues: 0` so any regression fails the run, and
`severity.default: error` treats every enabled finding as a hard
failure. CI is green only when the suite reports zero issues.

### Install and run

Install it if missing:

```bash
brew install golangci-lint
# or
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

Verify the config locally before pushing:

```bash
golangci-lint config verify     # schema check
golangci-lint run ./...         # must finish with 0 issues
golangci-lint fmt ./...         # auto-fix formatting (gofmt + goimports)
```

CI and release workflows must run `golangci-lint run ./...` and
`golangci-lint fmt --check ./...` before publishing or merging.

### Adding a new linter

When proposing a new linter:

1. Add it to `.golangci.yml` under `linters.enable`.
2. Run `golangci-lint run ./...` locally and collect the new findings.
3. Either fix every finding in the same PR, or land the config change
   **with** a follow-up issue that lists every file:line that must be
   fixed. Do not merge a config change that knowingly increases the
   issue count without that follow-up.
4. Update the table above in the same PR.
5. CI must pass before and after the change.

The same rule applies to linters we explicitly do **not** enable: a
deliberate `linters.disable` entry with a one-line rationale is allowed
if and only if the linter has a documented false-positive in this
codebase. Bare "we don't want noise" is not a justification.

## Comments

All code comments must be written in English.

Every newly created source, script, workflow, or config file must start
with a file header comment. The `Created` date must use the current date
from the environment.

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

For existing files without headers, add a header only when the edit is
substantial and the file format supports comments safely.

## Domain Rules

These are enforced by both code and validation. See the linked doc for
the full schema.

### packspec.json (docs/001-spec.md)

- Required: `packName`, `packVersion`, `minecraftVersion`, `loaderName`.
- `loaderName` is an array of strings, each `<loader>:<loaderVersion>`
  (e.g. `neoforge:21.1.219`). Supported loader names: `neoforge`,
  `fabric`. Reject anything else in validation.
- Mod key normalization (docs/006-mod-key-normalization.md):
  1. Lowercase.
  2. Spaces, underscores, punctuation → `-`.
  3. Strip apostrophes.
  4. Collapse consecutive `-`.
  5. Strip leading/trailing `-`.
- `scope` per mod: `shared` (default), `client`, `server`.
- `source.type` must be one of `curseforge`, `github-release`, `git`,
  `local`, `url`. See `docs/005-source-resolution.md`.
- Forbidden in `packspec.json` (Hard Rule #4): CurseForge `modId`,
  `fileId`, `fileName`; GitHub `assetName`; any direct download URL.
  These belong in the lock file, not the spec.

### Lock files (docs/003-lock-files.md)

- Path: `locks/dependencies/<minecraftVersion>-<loader>.json`.
- Schema: top-level `loader`, `minecraftVersion`, `mods`. Per-mod
  `name`, `version`, `scope`, `source`, optional `identity`,
  `dependencies`, and the resolver-written `hash` fingerprint.
- The lock file is a clean source of truth. Per-run summaries
  (added/kept/removed/failed) go to **stderr**, not into the JSON.

### Release index (docs/004-release-index.md)

- Path: `locks/releases/<minecraftVersion>.json`.
- Each `releases[]` entry has `version`, `type`, optional `github`,
  and `artifact` (paths by loader).
- No duplicate `version` per Minecraft version.

### API key resolution (docs/011-configuration.md)

Strict priority order, first non-empty wins:

1. `CURSEFORGE_API_KEY` environment variable.
2. `.mcmod/config.json` (project-level).
3. `~/.config/mcmod/config.json` (user-level).

Do not add a fourth tier. Do not read keys from `packspec.json`.

### Build output (docs/008-build-pipeline.md)

- Path: `releases/v<packVersion>/<packName>-<mc>-<loader>-<loaderVersion>-<target>.zip`
  and the matching `<serverPackName>-...-server.zip` for the server build.
- `target` ∈ `client`, `server`, `both`. Default `both`.
- `build-type` ∈ `cf`, `github`, `all`. Default `all`.
- Cache layout (docs/010-downloader.md):
  `.cache/curseforge/<modId>/<fileId>/<fileName>` and
  `.cache/github-release/<owner>/<repo>/<tag>/<assetName>`.
  SHA256 is computed for every downloaded file.

## File & Path Conventions

- `packspec.json` lives at the repo root. Do not look for it in
  subdirectories.
- `.mcmod/config.json` is the project config; created by
  `mcmod set cf-key ... --project` or `mcmod config set-cf-key ...`.
- `~/.config/mcmod/config.json` is the user config; created by
  `mcmod set cf-key ...` (default) or `--global`.
- `.env` is for local development only and **must** stay in `.gitignore`.
  Do not commit secrets.
- `.golangci.yml` is the lint config. Editing it is a normal change,
  but every edit must keep `max-issues-per-linter: 0`.

## Forbidden Patterns

Agents must not introduce any of the following without a prior
discussion in the PR:

- `// TODO` / `// FIXME` / `// XXX` markers in code. Use an issue
  instead. If a quick marker is genuinely needed, link the issue.
- Bare `panic`, `log.Fatal`, or `os.Exit` outside `cmd/mcmod/main.go`.
- `init()` with side effects outside `internal/testutil`.
- `fmt.Println`, `log.Println` from `internal/cli/...`. Use the
  cobra output streams.
- `os.UserHomeDir` / hard-coded `~`. Use `os.UserConfigDir()` or the
  helpers in `internal/config`.
- New dependencies. Use the standard library first.
- New top-level directories under `internal/`. Extend an existing one.
- Renaming or removing exported identifiers. Add the new, deprecate the
  old, and remove in a later major.
- Inline JSON strings outside of `test/`, `docs/`, or `packspec.json`.
  Use struct literals.

## Pull Requests

- One logical change per PR. Squash-merge is the default.
- PR title follows Conventional Commits (same as the squash commit).
- PR description must list:
  - Which `docs/NNN-*.md` files (if any) were updated.
  - The `golangci-lint run ./...` output (must be empty).
  - The `go tool cover -func=coverage.out` total line.
  - Any user-visible CLI behavior change with a before/after example.
- Breaking changes to `packspec.json` or `locks/dependencies/` schemas
  require a `!` after the type in the conventional commit (e.g.
  `feat(spec)!: rename loaderName to loaders`) and a migration note in
  the matching doc.
- If a change touches a CLI command's flags, output format, or exit
  code, update `docs/002-cli-overview.md` in the same PR.

## GitHub Actions And Release

CI and release workflows must stay aligned with the repository docs
and implemented CLI behavior. GitHub Actions should cache Go modules
and the Go build cache, but must not cache project `.cache/`, `.mcmod/`,
`releases/`, `dist/`, release archives, coverage output, or secrets.

The CLI release assets must use these names:

```text
mcmod_cli_<version>_linux_amd64.tar.gz
mcmod_cli_<version>_linux_arm64.tar.gz
mcmod_cli_<version>_windows_amd64.zip
mcmod_cli_<version>_darwin_arm64.tar.gz
mcmod_cli_<version>_checksums.txt
```

Release builds must be triggered by `v*` tags and must only publish
after lint, tests, coverage, and build checks pass.

## Documentation

README and docs must stay aligned with:

1. `packspec.json` schema.
2. `locks/dependencies/` schema.
3. `locks/releases/` schema.
4. CLI commands and help output.
5. Jar resolver/downloader behavior.
6. Build output paths and names.
7. GitHub Actions release artifact names.

Docs under `docs/` must use numbered file names such as `000-index.md`,
`001-spec.md`, and `002-cli-overview.md`. The numbering is a hint to
agents about the reading order; do not renumber existing files in a
non-doc-only PR.

These docs are primarily for AI agents, so they must be explicit,
self-contained, and include commands, expected outputs, file paths,
JSON fields, and error examples. A doc without a runnable command
example is incomplete and should not be merged.

When a doc becomes out of date, the change that makes it out of date
**must** include the doc update. There is no "doc fixup" follow-up —
land it together or not at all.
