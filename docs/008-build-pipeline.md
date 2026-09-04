<!-- File: docs/008-build-pipeline.md; Created: 2026-09-04; Description: Build pipeline. -->
# Build Pipeline

The pipeline reads the root spec, loads the selected lock, resolves cached
files, validates jars, and packages client/server artifacts. `--build-type
all` produces mcmod zips; `--build-type cf` produces a manifest-only
CurseForge zip. An incomplete lock fails before artifact output.

Before writing any artifact, the build scans every selected mod JAR. It rejects
duplicate `.class` paths, unreadable JARs, and missing required Fabric/NeoForge
dependencies. Errors are aggregated into one stable report listing all
conflicting classes, owning mods/JARs, and every missing dependency. CurseForge
builds resolve and validate all client JARs first, so manifest-only output
cannot bypass these checks. A failed validation leaves no new artifact.

## CLI Release Automation

`mcm version`, `mcm -v`, and `mcm --version` share the hard-coded
`domain.Version` value. Stable and beta workflows build native Linux
amd64/arm64, Windows amd64, and macOS amd64/arm64 archives in parallel.
Stable bumps create a `release`-labeled version PR; beta publishes a
`vX.Y.Z-canary.N` prerelease directly. Both accept an optional `base_version`
(`x.y.z`), which is used directly when supplied.
