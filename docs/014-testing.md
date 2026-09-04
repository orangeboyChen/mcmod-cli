<!--
File: docs/014-testing.md
Created: 2026-06-20
Description: Test coverage and end-to-end tests.
-->
# Testing

## Test Structure
- `internal/domain/`: Unit tests for models, validation, normalization, store
- `internal/cli/`: Coverage and integration tests

## Coverage
- Target: 90%+ statement coverage (raised from 80% on 2026-06-21)
- Run: `go test ./... -coverprofile=coverage.out`
- Current: see `go tool cover -func=coverage.out` (must remain >= 90.0%)
- ~980 Ginkgo specs across the full test tree:
  - `test/e2e/`: subprocess-driven end-to-end CLI coverage
  - `internal/cli/`: 133 (coverage, extra, mass, boost, push, last80)
  - `internal/domain/`: 180 (validation, normalization, store, errors)
  - `internal/service/`: 67 (mod, lock, release-lock, tree, build)
  - `internal/resolver/`: 48 (curseforge, github, git, local)
  - `internal/downloader/`: 24
  - `internal/metadata/`: 22
  - `internal/cache/`: 13
  - `internal/config/`: 12
  - `internal/graph/`: 5
- `cmd/mcmod/`: Go unit test for the `main` entry point

## Linting
- Required: `golint ./...`
- Run before commit with gofmt

## End-to-End Tests
- Build the binary once per suite in `BeforeSuite` into a temp dir.
- Each test runs in its own temp dir (`d = GinkgoT().TempDir()`).
- Subprocess environment is isolated: `HOME=<d>`, `XDG_CONFIG_HOME=<d>`, `CURSEFORGE_API_KEY=""` so user-level config never leaks between tests or into the host.
- The same env-isolation pattern is also applied to in-process CLI unit tests via `chdirTemp()` in `internal/cli/boost_test.go` to keep `~/.config/mcmod/` clean.
- Validate packspec.json
- Test normalize key rules
- Test validation errors
- Test lock file round-trips
- Test every CLI subcommand, every flag, and every error path documented above
- Integration tests cover: real 21-mod packspec, generated e2e workspace, multi-loader specs, build target/force/build-type, lock add/update/delete/release set/list/show/delete, CURSEFORGE_API_KEY resolution order, set/config/help/error format, tree resolution, and zip content verification

## Test Stability
- Ginkgo randomises spec order; if a failure appears in CI, rerun with `go test -count=1` (or pin a seed via `-ginkgo.seed=N`).
- The `test/e2e/` suite shells out to the built `mcmod` binary, so each test fully re-initialises its working directory and env.
- `go test ./...` is expected to pass cleanly on a developer machine within ~10s after the binary is built once.

## End-to-End Test Coverage
- [x] Build the `mcmod` binary before running end-to-end tests.
- [x] Verify root `--help` lists real top-level commands only.
- [x] Verify `mcmod help` and subcommand help output.
- [x] Verify `mcmod version` writes the version to stdout.
- [x] Verify `mcmod set cf-key <key>` writes user config.
- [x] Verify `mcmod set cf-key <key> --project` writes `.mcmod/config.json`.
- [x] Verify `mcmod set cf-key <key> --global` behaves as the user-level alias.
- [x] Verify bad `set` arguments fail with a hint.
- [x] Verify `mcmod config` prints configured key state.
- [x] Verify `mcmod config set-cf-key <key>` writes project config.
- [x] Verify `mcmod list` groups shared, client, and server mods.
- [x] Verify `mcmod list` reports missing `packspec.json`.
- [x] Verify `mcmod validate` accepts the default `packspec.json`.
- [x] Verify `mcmod validate --spec <path>` accepts valid specs and rejects invalid paths.
- [x] Verify `mcmod validate --lock <file>` accepts valid lock files and rejects malformed locks.
- [x] Verify `mcmod validate --release-index <file>` accepts valid release indexes and rejects malformed JSON.
- [x] Verify `mcmod lock` reports missing specs.
- [x] Verify `mcmod lock <mc> <loader>` writes lock files for local-only mods.
- [x] Verify `mcmod lock <mc>` iterates configured loaders.
- [x] Verify unsupported lock loaders emit an error hint.
- [x] Verify `mcmod lock list <mc> <loader>` lists lock entries by scope.
- [x] Verify `mcmod lock list` reports missing lock files.
- [x] Verify `mcmod lock show <mc> <loader>` dumps full lock JSON.
- [x] Verify `mcmod lock show <mc> <loader> <key>` prints curseforge fields.
- [x] Verify `mcmod lock show <mc> <loader> <key>` prints GitHub release fields.
- [x] Verify `mcmod lock show` rejects missing args and missing keys.
- [x] Verify `mcmod lock add` can create curseforge entries.
- [x] Verify `mcmod lock add` can create GitHub release entries.
- [x] Verify `mcmod lock add` can create local entries.
- [x] Verify `mcmod lock add` rejects missing args and duplicate keys.
- [x] Verify `mcmod lock update <mc> <loader> <key> --version <version>` updates one entry.
- [x] Verify `mcmod lock update` refreshes local-only specs.
- [x] Verify `mcmod lock update` rejects missing locks and missing keys.
- [x] Verify `mcmod lock delete <mc> <loader> <key>` removes one lock entry.
- [x] Verify `mcmod lock delete` reports missing locks and missing keys.
- [x] Verify `mcmod lock tree <mc> <loader>` renders dependency tree output.
- [x] Verify `mcmod tree <mc> <loader>` works as the top-level alias.
- [x] Verify tree commands report missing locks.
- [x] Verify `mcmod lock release set <mc> <loader>` writes GitHub release metadata.
- [x] Verify release set optional `--name`, `--body`, `--draft`, and `--prerelease` flags.
- [x] Verify release set `--artifact-client` and `--artifact-server` write loader artifacts.
- [x] Verify release set rejects missing required flags.
- [x] Verify `mcmod lock release list [<mc>]` lists release records.
- [x] Verify `mcmod lock release list` reports missing indexes.
- [x] Verify `mcmod lock release show <mc> <version>` prints one release.
- [x] Verify release show rejects missing versions.
- [x] Verify `mcmod lock release delete <mc> <version>` removes a full release.
- [x] Verify `mcmod lock release delete <mc> <version> <loader> --target client` removes only the client artifact.
- [x] Verify `mcmod lock release delete <mc> <version> <loader> --target server` removes only the server artifact.
- [x] Verify release delete reports missing indexes and missing versions.
- [x] Verify `mcmod build <mc> <loader>` reports missing locks with a hint.
- [x] Verify `mcmod build <mc> <loader>` creates artifacts from local locked jars.
- [x] Verify build `--target client`, `--target server`, `--target both`, `--build-type`, and `--force` flags are accepted.
- [x] Verify build reports missing `packspec.json`.
- [x] Verify unknown commands fail.
- [x] Verify generated fixture jars cover NeoForge and Fabric metadata parsing.
- [x] Verify cache hit, miss, atomic move, and checksum paths used by generated fixtures.
- [x] Verify duplicate class path fixture detection.
- [x] Verify `mcmod` (no args) prints `Usage` banner.
- [x] Verify `mcmod help <unknown-topic>` prints help to stderr and exits non-zero.
- [x] Verify `mcmod build --help`, `mcmod validate --help`, `mcmod list --help`, and `mcmod set --help` all print per-subcommand help.
- [x] Verify `mcmod set cf-key` with one argument fails.
- [x] Verify `mcmod set cf-key <key> --project` writes `.mcmod/config.json` containing the key.
- [x] Verify `mcmod set cf-key <key>` (no flag) does not create `.mcmod/config.json`.
- [x] Verify `mcmod set cf-key <key> --global` does not create `.mcmod/config.json`.
- [x] Verify `mcmod set cf-key <key> --project` (twice) overwrites the previous project key.
- [x] Verify `mcmod list` renders a `[Server]` section for server-only mods.
- [x] Verify `mcmod list` renders `[Server]`, `[Client]`, and `[Shared]` for mixed-scope packs.
- [x] Verify `mcmod list` lists every configured loader.
- [x] Verify `mcmod list` reports missing `packspec.json` with a hint.
- [x] Verify `mcmod validate --lock <missing>` fails.
- [x] Verify `mcmod validate` and `mcmod validate --spec` both validate a project `packspec.json`.
- [x] Verify `mcmod validate --release-index <missing>` fails.
- [x] Verify `mcmod validate --spec` rejects specs missing required fields.
- [x] Verify `mcmod validate --lock` accepts valid lock files.
- [x] Verify `mcmod validate` in an empty directory fails with a hint.
- [x] Verify `mcmod lock` without a spec fails.
- [x] Verify `mcmod lock <mc> <loader>` writes all configured mods to the lock file.
- [x] Verify `mcmod lock <mc>` writes per-loader lock files for every configured loader.
- [x] Verify `mcmod lock` with empty mods writes an empty map.
- [x] Verify `mcmod lock` output starts with `Locked` for the printed loader summary.
- [x] Verify `mcmod lock` with an unsupported loader prints a hint to stderr.
- [x] Verify `mcmod lock list` (no args) uses default mc/loader.
- [x] Verify `mcmod lock list` shows all three scope sections.
- [x] Verify `mcmod lock list` with empty mods prints `(empty)` three times.
- [x] Verify `mcmod lock list <mc> <loader>` fails with a hint when the file is missing.
- [x] Verify `mcmod lock list` treats empty-scope mods as shared.
- [x] Verify `mcmod lock list` lines include key, name, and version.
- [x] Verify `mcmod lock show` with zero or one arg fails.
- [x] Verify `mcmod lock show <mc> <loader>` (no key) dumps the full lock as JSON.
- [x] Verify `mcmod lock show <mc> <loader> <key>` prints scope and source fields.
- [x] Verify `mcmod lock show <mc> <loader> <missing>` fails.
- [x] Verify `mcmod lock show <mc> <loader> <key>` with a missing lock file fails.
- [x] Verify `mcmod lock add` with 0/1/2 args fails.
- [x] Verify `mcmod lock add` writes `modId` and `fileId` for curseforge sources.
- [x] Verify `mcmod lock add` creates a new lock file when none exists.
- [x] Verify `mcmod lock add` appends to an existing lock without overwriting it.
- [x] Verify `mcmod lock update` with 0 args and no spec fails.
- [x] Verify `mcmod lock update <mc> <loader>` (no key) fails.
- [x] Verify `mcmod lock update <mc> <loader> <key>` fails when the lock is missing.
- [x] Verify `mcmod lock update <mc> <loader> <missing>` fails with a not-found error.
- [x] Verify `mcmod lock update <mc> <loader> <key>` without `--version` still saves.
- [x] Verify `mcmod lock update` (no args) re-runs every configured loader.
- [x] Verify `mcmod lock delete` with 0/1/2 args prints a hint.
- [x] Verify `mcmod lock delete <mc> <loader> <key>` removes the entry but keeps the rest.
- [x] Verify `mcmod lock delete` fails for missing lock files and missing keys.
- [x] Verify `mcmod lock tree` with 0 args uses default mc/loader.
- [x] Verify `mcmod lock tree <mc>` uses the default loader.
- [x] Verify `mcmod tree` (alias) with 0 args uses default mc/loader.
- [x] Verify tree output includes the `dependency tree` header.
- [x] Verify `mcmod tree <mc> <loader>` fails with a hint when the lock is missing.
- [x] Verify `mcmod build` with 0 args uses default mc/loader.
- [x] Verify `mcmod build --target server` builds only the server artifact.
- [x] Verify `mcmod build --build-type all` and `--build-type cf` are both accepted.
- [x] Verify `mcmod build --force` is accepted.
- [x] Verify `mcmod build <mc> <loader>` without a lock prints a hint to stderr.
- [x] Verify `mcmod build` without a spec fails with a hint.
- [x] Verify `mcmod lock release list <mc>` shows one or more release versions.
- [x] Verify `mcmod lock release show <mc> <version>` prints the GitHub repo and tag.
- [x] Verify `mcmod lock release show` with 0 or 1 arg fails.
- [x] Verify `mcmod lock release delete` with 0 or 1 arg fails.
- [x] Verify `mcmod lock release set` requires `--repo`, `--tag`, and `--version`.
- [x] Verify `mcmod lock release set --name --body` writes the optional fields.
- [x] Verify `mcmod lock release set --draft --prerelease` writes draft and prerelease flags.
- [x] Verify `mcmod lock release set --artifact-client --artifact-server` writes artifact paths.
- [x] Verify `mcmod lock release list <mc>` fails when the index is missing.
- [x] Verify `mcmod config` with no args prints `(not set)` when no key is configured.
- [x] Verify `mcmod config set-cf-key <key>` writes project config and prints confirmation.
- [x] Verify `mcmod config` reflects the key after `set-cf-key`.
- [x] Verify `mcmod config <unknown>` falls back to printing the current state.
- [x] Verify NeoForge and Fabric fixture jars can be created.
- [x] Verify cache miss/hit detection for fixture files.
- [x] Verify an unknown top-level command fails.
- [x] Verify `mcmod set` with no arguments fails.
- [x] Verify `mcmod validate` in an empty directory fails with a hint.
- [x] Verify `mcmod lock show` fails on a missing lock file.
- [x] Verify `mcmod tree` fails with a hint when the lock is missing.

## Integration Test Coverage

This section enumerates the additional integration-level tasks that exercise
the real `mcmod` binary against realistic packspec/lock/release workspaces.
The goal is to make sure every CLI command, subcommand, flag combination, and
error path documented in `specification.md` is invoked end-to-end.

### Real PackSpec Integration (I-series)

- [x] **I01** Copy the real project `packspec.json` to a temp dir, run `mcmod list`, and verify all 21 mods are grouped by `[Server]` / `[Client]` / `[Shared]` and match the expected scope assignments.
- [x] **I02** Run `mcmod validate` against the real `packspec.json` and confirm `packspec.json is valid`.
- [x] **I03** Run `mcmod lock list 1.21.1 neoforge` against the example lock file and assert the entries (create / jei / server-enhanced) appear in the correct scope.
- [x] **I04** Run `mcmod lock show 1.21.1 neoforge create` and assert the curseforge `modId`, `fileId`, `fileName` fields are printed.
- [x] **I05** Run `mcmod lock show 1.21.1 neoforge server-enhanced` and assert the github-release `repo`, `tag`, `assetName` fields are printed.
- [x] **I06** Run `mcmod tree 1.21.1 neoforge` and assert the dependency tree header is printed with all four mods.
- [x] **I07** Run `mcmod lock release list 1.21.1` against the example release index and assert one record `0.1.0` is printed.
- [x] **I08** Run `mcmod lock release show 1.21.1 0.1.0` and assert the GitHub repo/tag and per-loader client/server artifact paths are printed.
- [x] **I09** Run `mcmod version` and assert `mcmod version 0.1.0` is printed.
- [x] **I10** Run `mcmod help`, `mcmod help lock`, `mcmod help lock release`, `mcmod help build`, `mcmod help set`, `mcmod help list`, `mcmod help validate`, `mcmod help config`, `mcmod help version`, `mcmod help tree` — assert each prints `Usage:` and the documented subcommands.
- [x] **I11** Run `mcmod <unknown-command>` from a real packspec dir and assert a non-zero exit with an error message.

### CLI Subcommand Coverage Matrix

- [x] `mcmod` (no args) prints root help and exits 0.
- [x] `mcmod --help` prints root help and exits 0.
- [x] `mcmod help` prints root help and exits 0.
- [x] `mcmod help lock` prints the lock subcommand help and exits 0.
- [x] `mcmod help lock release` prints the release subcommand help and exits 0.
- [x] `mcmod help build` prints the build help with all flags.
- [x] `mcmod help set` prints the set help with all flags.
- [x] `mcmod help list` prints the list help.
- [x] `mcmod help validate` prints the validate help with all flags.
- [x] `mcmod help config` prints the config help.
- [x] `mcmod help version` prints the version help.
- [x] `mcmod help tree` prints the tree help.
- [x] `mcmod help <unknown>` prints an error and exits non-zero.
- [x] `mcmod <unknown-command>` prints an error and exits non-zero.
- [x] `mcmod set` (no subcommand) fails with hint.
- [x] `mcmod set cf-key` (no value) fails with hint.
- [x] `mcmod set cf-key <key>` writes user-level config and prints `set cf-key`.
- [x] `mcmod set cf-key <key> --project` writes project-level `.mcmod/config.json` and prints `set cf-key`.
- [x] `mcmod set cf-key <key> --global` writes user-level config and prints `set cf-key`.
- [x] `mcmod set cf-key <key> --project` does not touch user-level config.
- [x] `mcmod set cf-key <key> --global` does not touch project-level config.
- [x] `mcmod set cf-key <wrong-subkey>` fails with hint.
- [x] `mcmod config` (no args) prints the currently configured key (or `(not set)` when none).
- [x] `mcmod config set-cf-key <key>` writes project config and prints confirmation.
- [x] `mcmod config set-cf-key <key>` after a prior `set` shows the new key.
- [x] `mcmod config <unknown-sub>` falls back to printing the current state.
- [x] `mcmod list` (no spec) fails with hint.
- [x] `mcmod list` on a real packspec shows `[Server]`, `[Client]`, `[Shared]` sections.
- [x] `mcmod list` shows `(empty)` for empty scope sections.
- [x] `mcmod list` lists every configured loader.
- [x] `mcmod validate` accepts the real `packspec.json`.
- [x] `mcmod validate --spec <path>` accepts valid spec.
- [x] `mcmod validate --spec <missing>` fails.
- [x] `mcmod validate --lock <file>` accepts valid lock.
- [x] `mcmod validate --lock <missing>` fails.
- [x] `mcmod validate --release-index <file>` accepts valid release index.
- [x] `mcmod validate --release-index <missing>` fails.
- [x] `mcmod validate --spec` rejects spec missing `packName` / `minecraftVersion` / `loaderName`.
- [x] `mcmod validate --lock` rejects lock with malformed JSON.
- [x] `mcmod validate` in an empty dir fails with hint.
- [x] `mcmod lock` (no spec) fails with hint.
- [x] `mcmod lock <mc> <loader>` writes the lock file with all configured mods.
- [x] `mcmod lock <mc>` iterates every configured loader.
- [x] `mcmod lock` with empty mods writes an empty map.
- [x] `mcmod lock` output starts with `Locked` for the printed summary.
- [x] `mcmod lock` with an unsupported loader prints a hint to stderr.
- [x] `mcmod lock list` (no args) uses default mc/loader.
- [x] `mcmod lock list` shows all three scope sections.
- [x] `mcmod lock list` with empty mods prints `(empty)` three times.
- [x] `mcmod lock list <mc> <loader>` fails with hint when the file is missing.
- [x] `mcmod lock list` treats empty-scope mods as shared.
- [x] `mcmod lock list` lines include key, name, and version.
- [x] `mcmod lock show` with 0 args fails.
- [x] `mcmod lock show` with 1 arg fails.
- [x] `mcmod lock show <mc> <loader>` (no key) dumps the full lock JSON.
- [x] `mcmod lock show <mc> <loader> <key>` prints scope and source fields for curseforge.
- [x] `mcmod lock show <mc> <loader> <key>` prints scope and source fields for github-release.
- [x] `mcmod lock show <mc> <loader> <missing-key>` fails.
- [x] `mcmod lock show <mc> <loader> <key>` with missing lock file fails.
- [x] `mcmod lock add` with 0/1/2 args fails.
- [x] `mcmod lock add <mc> <loader> <key>` writes curseforge `modId`/`fileId`/`fileName`.
- [x] `mcmod lock add <mc> <loader> <key>` writes github-release `repo`/`tag`/`assetName`/`fileName`.
- [x] `mcmod lock add <mc> <loader> <key>` writes local `path`/`fileName`.
- [x] `mcmod lock add <mc> <loader> <key>` rejects duplicate keys.
- [x] `mcmod lock add` creates a new lock file when none exists.
- [x] `mcmod lock add` appends to an existing lock without overwriting it.
- [x] `mcmod lock update` with 0 args and no spec fails.
- [x] `mcmod lock update <mc> <loader>` (no key) fails.
- [x] `mcmod lock update <mc> <loader> <key>` fails when the lock is missing.
- [x] `mcmod lock update <mc> <loader> <missing-key>` fails.
- [x] `mcmod lock update <mc> <loader> <key> --version <v>` updates the version.
- [x] `mcmod lock update <mc> <loader> <key>` without `--version` still saves.
- [x] `mcmod lock update` (no args) re-runs every configured loader.
- [x] `mcmod lock delete` with 0/1/2 args prints hint.
- [x] `mcmod lock delete <mc> <loader> <key>` removes only the entry.
- [x] `mcmod lock delete <mc> <loader> <key>` fails when key is missing.
- [x] `mcmod lock delete <mc> <loader>` removes the entire lock file.
- [x] `mcmod lock delete <mc>` removes every loader lock for the given mc version.
- [x] `mcmod lock tree` (no args) uses default mc/loader.
- [x] `mcmod lock tree <mc>` uses the default loader.
- [x] `mcmod tree` (alias) works with 0 args.
- [x] `mcmod tree` output contains the `dependency tree` header.
- [x] `mcmod tree` with missing lock file fails with hint.
- [x] `mcmod lock release set <mc> <loader>` writes GitHub release metadata.
- [x] `mcmod lock release set --name --body` writes optional fields.
- [x] `mcmod lock release set --draft --prerelease` writes draft/prerelease flags.
- [x] `mcmod lock release set --artifact-client --artifact-server` writes artifact paths.
- [x] `mcmod lock release set` rejects missing `--repo`, `--tag`, or `--version`.
- [x] `mcmod lock release list <mc>` lists one or more release versions.
- [x] `mcmod lock release list` reports missing indexes.
- [x] `mcmod lock release show <mc> <version>` prints one release.
- [x] `mcmod lock release show` with 0/1 arg fails.
- [x] `mcmod lock release delete <mc> <version>` removes a full release.
- [x] `mcmod lock release delete <mc> <version> <loader> --target client` removes only the client artifact.
- [x] `mcmod lock release delete <mc> <version> <loader> --target server` removes only the server artifact.
- [x] `mcmod lock release delete <mc> <version> <loader> --target both` removes both artifacts.
- [x] `mcmod lock release delete` reports missing indexes and missing versions.
- [x] `mcmod build` (no args) uses default mc/loader.
- [x] `mcmod build --target client` writes only the client zip.
- [x] `mcmod build --target server` writes only the server zip.
- [x] `mcmod build --target both` writes both zips.
- [x] `mcmod build --build-type cf` is accepted.
- [x] `mcmod build --build-type all` is accepted.
- [x] `mcmod build --force` allows overwriting an existing zip.
- [x] `mcmod build` without a lock prints a hint to stderr.
- [x] `mcmod build` without a spec prints a hint to stderr.
- [x] `mcmod build` with an invalid `--target` fails.
- [x] `mcmod build` writes a zip whose contents include `mods/`, `config/`, `defaultconfigs/`, `resourcepacks/`, and `server.properties` from overrides.
- [x] `mcmod build` zip filename format is `<packName>-<mcVersion>-<loader>-<loaderVersion>-client.zip` and `<serverPackName>-<mcVersion>-<loader>-<loaderVersion>-server.zip`.

### CURSEFORGE_API_KEY Resolution Order

- [x] `mcmod config` (no args) shows `(not set)` when no env, project, or user config key is configured.
- [x] `mcmod config` (no args) shows the project key when `.mcmod/config.json` exists.
- [x] `mcmod config` (no args) shows the user key when only `~/.config/mcmod/config.json` exists.
- [x] `CURSEFORGE_API_KEY=<env>` is used as the highest-priority source; project/user keys are ignored when the env var is set.

### Error Message Format

- [x] All CLI error messages follow `error: <command>: <reason>` on stderr.
- [x] All CLI error messages that are actionable include a `hint: <fix>` line on stderr.
- [x] `mcmod <unknown-command>` prints an error and exits non-zero.
- [x] `mcmod <typo>` does NOT print the cobra usage banner (SilenceUsage).

### Build Artifact Contents

- [x] `mcmod build --target client` zip contains `mods/<shared>.jar` and `mods/<client>.jar` and no `mods/<server>.jar`.
- [x] `mcmod build --target server` zip contains `mods/<shared>.jar` and `mods/<server>.jar` and no `mods/<client>.jar`.
- [x] `mcmod build --target both` writes two zips with the correct scope separation.
- [x] `mcmod build` zips include the `config/`, `defaultconfigs/`, `resourcepacks/`, and `server.properties` overrides files.

### CLI Subcommand Surface Audit

- [x] The root `mcmod --help` lists exactly: `set`, `list`, `lock`, `build`, `validate`, `tree`, `config`, `version`, and `help` (and `completion` from cobra).
- [x] The root `mcmod --help` does NOT list any legacy `add`, `show`, `update`, or `delete` top-level commands.
- [x] The root `mcmod --help` does NOT list any `mod` / `entry` / `package` commands.
- [x] `mcmod help lock` lists every lock subcommand: `list`, `show`, `add`, `update`, `delete`, `tree`, `release`.
- [x] `mcmod help lock release` lists every release subcommand: `set`, `list`, `show`, `delete`.
- [x] `mcmod build --help` lists `--target`, `--build-type`, `--force`, `--loader`, `--mc-version`.
- [x] `mcmod set --help` lists `--project`, `--global`.
- [x] `mcmod validate --help` lists `--spec`, `--lock`, `--release-index`.

### Coverage Requirements

- [x] Total statement coverage remains `>= 90.0%` after every change.
- [x] `go mod tidy` produces no unrelated dependency changes.
- [x] `gofmt` produces no diffs on the new test files.
- [x] `golint ./...` is run; pre-existing warnings are not regressions.
