<!-- File: docs/008-build-pipeline.md; Created: 2026-09-04; Description: Build pipeline. -->
# Build Pipeline

The pipeline reads the root spec, loads the selected lock, resolves cached
files, validates jars, and packages client/server artifacts. `--build-type
all` produces mcmod zips; `--build-type cf` produces a manifest-only
CurseForge zip. An incomplete lock fails before artifact output.
