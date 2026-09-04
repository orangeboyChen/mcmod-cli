<!-- File: docs/010-downloader.md; Created: 2026-09-04; Description: Downloads and cache. -->
# Downloader

Jars are cached under `.mcmod/cache/curseforge/...` or
`.mcmod/cache/github-release/...`.
CurseForge uses its CDN by default and can opt into the download-url API with
`MCMOD_CURSEFORGE_USE_DOWNLOAD_URL=1`. Local and Git packspec sources require
no jar download.
