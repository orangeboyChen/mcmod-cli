<!--
File: docs/008-build-pipeline.md
Created: 2026-06-20
Description: Build artifact generation pipeline.
-->
# Build Pipeline

1. Read packspec.json
2. Load dependency lock for target loader
3. Download mod jars (if not cached)
4. Validate every selected jar before writing an artifact
5. Assemble client/server mods based on scope
6. Package into ZIP archives
7. Update release index

Builds require a complete dependency lock: every mod in `packspec.json` must
have a lock entry, and every selected mod jar must resolve successfully. If
the lock is partial or a jar cannot be resolved, the build exits non-zero and
writes no artifact for that target.

Before packaging, `mcmod build` scans every selected jar. It reports all
duplicate `.class` paths together with every owning mod and jar, all jars that
cannot be opened, and all required Fabric/NeoForge dependencies missing from
the target set. A jar without recognized Fabric/NeoForge metadata is allowed
but is skipped for dependency validation. Validation output is stable-sorted
and a failed validation leaves no new artifact.

`--build-type cf` performs the same client-set validation, resolving or
downloading CurseForge jars when they are not already cached. This means the
first CF build can perform network downloads before producing its manifest.

Output path: `releases/v<version>/<packName>-<mcVersion>-<loader>-<loaderVersion>-<target>.zip`
