<!-- File: docs/012-metadata-parsing.md; Created: 2026-09-04; Description: Jar metadata validation. -->
# Metadata Parsing

Build validation reads NeoForge TOML and Fabric `fabric.mod.json` to identify
jars and check required dependencies. This is separate from recursive Git
packspec expansion, which never unpacks jars.
