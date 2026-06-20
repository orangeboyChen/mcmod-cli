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

Output path: `releases/v<version>/<packName>-<mcVersion>-<loader>-<loaderVersion>-<target>.zip`