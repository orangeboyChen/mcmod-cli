<!--
File: docs/012-metadata-parsing.md
Created: 2026-06-20
Description: Jar metadata readers for NeoForge and Fabric.
-->
# Metadata Parsing

## NeoForge
- Reads `META-INF/neoforge.mods.toml` or `META-INF/mods.toml`
- Parses simple TOML for modid and version
- Returns ModInfo with ModID, Version, Dependencies

## Fabric
- Reads `fabric.mod.json`
- Parses JSON for id, version, depends
- Returns ModInfo with ModID, Version, Dependencies

## Auto-Detection
- `ReadJarMetadata()` tries NeoForge first, then Fabric
- Returns error if neither format is detected