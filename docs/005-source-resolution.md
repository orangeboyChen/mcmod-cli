<!--
File: docs/005-source-resolution.md
Created: 2026-09-04
Description: Source resolution and git packspec bundles.
-->

# Source Resolution

## Git packspec bundles

`type: "git"` points to a GitHub repository containing `packspec.json` at
the root of either the `main` or `master` branch:

```json
{"name":"shared-bundle","source":{"type":"git","repo":"owner/shared-bundle"},"scope":"server"}
```

The repository is expanded recursively during `mcmod lock`. Its child mods
are resolved normally; the Git repository itself is not downloaded as a jar.

## Other sources

- `curseforge`: resolve by query or slug and select a file for Minecraft and loader.
- `github-release`: resolve a release tag and matching asset pattern.
- `local`: resolve a local jar path with `{mcVersion}` and `{loader}` placeholders.
- `url`: download an operator-supplied URL using the explicit cache identity.
