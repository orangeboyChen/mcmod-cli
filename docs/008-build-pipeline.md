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
