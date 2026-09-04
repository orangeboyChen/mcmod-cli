<!--
File: docs/010-downloader.md
Created: 2026-06-20
Description: File download and caching.
-->
# Downloader

## Sources
- CurseForge: Downloads via `edge.forgecdn.net/files/{fileId4}/{fileName}`
  by default; set `MCMOD_CURSEFORGE_USE_DOWNLOAD_URL=1` to use the API
  `/download-url` endpoint instead
- GitHub Release: Downloads via direct asset URL
- Local: No download needed (file reference only)

## Cache
- `.cache/curseforge/<modId>/<fileId>/<fileName>`
- `.cache/github-release/<owner>/<repo>/<tag>/<assetName>`
- SHA256 computation for integrity verification
