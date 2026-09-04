<!-- File: docs/005-source-resolution.md; Created: 2026-09-04; Description: Source backends. -->
# Source Resolution

CurseForge resolves a query or slug to a file for the requested Minecraft
version and loader. GitHub releases resolve a tag and asset pattern. Local
resolves a project-relative jar path. URL uses an explicit download URL.

Git sources point to a GitHub repository containing `packspec.json` at its
root on `main` or `master`. `mcmod lock` reads, filters, and recursively
expands that file. The repository itself is never treated as a jar.
