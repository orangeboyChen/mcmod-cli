<!--
File: docs/004-release-index.md
Created: 2026-06-20
Description: Release index file format.
-->
# Release Index

## Location
`locks/releases/<minecraftVersion>.json`

## Schema
- `type` (string): Index type
- `packName` (string): Pack name
- `minecraftVersion` (string): Minecraft version
- `releases` (array): Release records

## ReleaseRecord
- `version` (string): Release version
- `type` (string): Release type
- `github` (object, optional): GitHub release metadata
- `artifact` (object): Artifact paths by loader