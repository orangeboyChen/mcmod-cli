<!--
File: docs/005-source-resolution.md
Created: 2026-06-20
Description: Source resolution for CurseForge, GitHub, Git, and local files.
-->
# Source Resolution

## CurseForge
- API base: `https://api.curseforge.com/v1`
- Resolves by query or mod ID + file ID
- Requires API key (env, project, or user config)
- Default download URL: `https://edge.forgecdn.net/files/{fileId4}/{fileName}`
- Set `MCMOD_CURSEFORGE_USE_DOWNLOAD_URL=1` to use the API download-url endpoint

## GitHub Release
- Resolves by repo, tag, and asset pattern
- Supports `{mcVersion}`, `{tag}`, `{loader}` placeholders
- Supports `*` wildcard matching for tags and patterns

## Git
- Reads `packspec.json` from remote repos via raw.githubusercontent.com
- Supports "main" and "master" branches

## Local
- Resolves local `.jar` files by path
- Supports `{mcVersion}` and `{loader}` placeholder substitution
