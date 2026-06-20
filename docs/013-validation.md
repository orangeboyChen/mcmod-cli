<!--
File: docs/013-validation.md
Created: 2026-06-20
Description: Validation rules for spec, lock, and release index files.
-->
# Validation

## PackSpec Validation
- Required fields: packName, packVersion, minecraftVersion, loaderName
- Loader names: must be neoforge or fabric
- Source types: curseforge, github-release, git, local
- Source-specific field requirements (query for CF, repo+tag for GitHub, etc.)
- No forbidden fields: modId, fileId, fileName, download URLs must not appear in spec

## Lock Validation
- Required: loader, minecraftVersion, mods
- Locked mods must have valid source type with required fields
- Source-specific requirements for each type

## Release Index Validation
- Required: type, packName, minecraftVersion
- No duplicate versions
- Each release must have a type