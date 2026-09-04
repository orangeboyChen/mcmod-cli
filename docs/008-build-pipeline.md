<!--
File: docs/008-build-pipeline.md
Created: 2026-06-20
Description: Build artifact generation pipeline.
-->
# Build Pipeline

1. Read packspec.json
2. Load dependency lock for target loader
3. Download mod jars (if not cached)
4. Assemble client/server mods based on scope
5. Package into ZIP archives
6. Update release index

Builds require a complete dependency lock: every mod in `packspec.json` must
have a lock entry, and every selected mod jar must resolve successfully. If
the lock is partial or a jar cannot be resolved, the build exits non-zero and
writes no artifact for that target.

Output path: `releases/v<version>/<packName>-<mcVersion>-<loader>-<loaderVersion>-<target>.zip`

## CLI Release Automation

The `mcm` executable reports the hard-coded `domain.Version` with either
`mcm version`, `mcm -v`, or `mcm --version`.

Stable releases use these workflows:

1. Run `Bump Stable Release Version` manually and select `major`, `minor`, or
   `patch`; `minor` is the default.
2. Optionally provide `base_version` as `x.y.z`. When present, it is used
   directly and the selected increment is ignored.
3. The workflow creates a version PR with the `release` label.
4. After merge, `Tag Stable Release` creates `vX.Y.Z`; the tag starts the
   parallel native-runner build and release workflow.

The published native archives cover Linux amd64/arm64, Windows amd64, and
macOS amd64/arm64. Release notes include commits with authors and the asset
list.

`Publish Beta Release` follows the same version inputs and publishes a
`vX.Y.Z-canary.N` prerelease directly. With no custom `base_version`, an
unfinished canary series increments its number; otherwise the selected
`major`/`minor`/`patch` component starts a new `canary.1` series.
