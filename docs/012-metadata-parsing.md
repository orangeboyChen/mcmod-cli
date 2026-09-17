<!-- File: docs/012-metadata-parsing.md; Created: 2026-09-04; Description: Jar metadata validation. -->
# Metadata Parsing

Build validation reads NeoForge TOML and Fabric `fabric.mod.json` to identify
jars and check required dependencies. This is separate from recursive Git
packspec expansion, which never unpacks jars.

## NeoForge mod identity

A `neoforge.mods.toml` declares the mod's own id under `[[mods]]` and each
dependency's id under `[[dependencies.<modId>]]`. Both tables use the key name
`modId`, so a flat key/value scan picks up whichever `modId=` appears last —
normally a dependency. The jar is then indexed under its dependency's id and
also counts as *providing* that id, so a genuinely missing dependency passes
validation silently.

The mod id is therefore read from the `[[mods]]` table only. Dependencies
never contribute an id to the provided set, only to the required set.
