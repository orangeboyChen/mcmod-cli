<!--
File: docs/001-spec.md
Created: 2026-06-20
Description: packspec.json schema documentation.
-->

# Specification

`packspec.json` is the single human-editable configuration file for mcmod.

## Top-Level Fields

- `packName` (string, required): Pack display name
- `packVersion` (string, required): Pack version
- `minecraftVersion` (string, required): Target Minecraft version
- `loaderName` (array[string], required): List of loaders (neoforge, fabric)
- `author` (string, optional): Pack author
- `mods` (object, optional): Mod definitions keyed by normalized ID

## Mod Entries

Each mod entry supports:
- `name` (string, optional): Display name
- `scope` (string, optional): "shared", "client", or "server"
- `loader` (array[string], optional): Loader filter
- `source` (object, required): Source configuration
## Source Object

The `source` object tells `mcmod lock` how to resolve a mod to a downloadable
file. The exact field set depends on the source `type`.

### Common Source Fields
- `type` (string, required): One of `curseforge`, `github-release`, `git`,
  `local`, `url`.
- `url` (string, optional): Operator-supplied CDN download URL. When present,
  the build step uses it directly and skips the rate-limited
  `/download-url` endpoint. Wins over `urlPattern` when both are set.
- `urlPattern` (string, optional): Template for the CDN download URL with
  placeholders that are filled at lock time. See "URLPattern Placeholders"
  below for the full list.

For CurseForge sources, when neither `url` nor `urlPattern` is set, mcmod
uses `https://edge.forgecdn.net/files/{fileId4}/{fileName}` by default.
Set `MCMOD_CURSEFORGE_USE_DOWNLOAD_URL=1` to opt into the rate-limited
`/download-url` API during download.

### CurseForge Source
- `type: "curseforge"`
- `query` (string, optional): Human-readable mod name. Used by the resolver
  to search CurseForge.
- `slug` (string, optional): CurseForge slug. The resolver prefers an exact
  slug match over a fuzzy `query` search. Recommended.
- `modId` (int, optional): Pre-resolved CurseForge project id. Skips the
  search step when both `modId` and `fileId` are present.
- `fileId` (int, optional): Pre-resolved CurseForge file id. The resolver
  validates the file still exists at lock time and renders the CDN URL
  from `urlPattern`.

### GitHub Release Source
- `type: "github-release"`
- `repo` (string, required): `"owner/name"`.
- `tag` (string, required): Release tag. May include the `{mcVersion}`
  placeholder.
- `assetPattern` (string, optional): Glob pattern for the asset filename
  inside the release. Supports the `*` wildcard.
- `assetPatternByLoader` (object, optional): Per-loader asset patterns.

### Git Source
- `type: "git"`
- `repo` (string, required): `"owner/name"`.

### Local Source
- `type: "local"`
- `path` (string, required): Relative path to the local jar. Supports the
  `{mcVersion}` and `{loader}` placeholders.

### URL Source
- `type: "url"`
- `modId` (int, required): Numeric id used for the local `.cache/` path
  layout. Any unique value works.
- `fileId` (int, required): Numeric id used for the local `.cache/` path
  layout. Any unique value works.
- `fileName` (string, required): Filename of the jar to download.
- `url` or `urlPattern` (one of them required): Download URL.

The `url` source type does not call any API. It is intended for mods whose
file is mirrored on a CDN like `edge.forgecdn.net` or
`mediafilez.forgecdn.net` and where the upstream registry either has no
API or is rate-limiting public keys.

## URLPattern Placeholders

`urlPattern` is a template string with the following placeholders expanded
at lock time:

| Placeholder    | Example value           | Description                                     |
| -------------- | ----------------------- | ----------------------------------------------- |
| `{modId}`      | `676721`                | CurseForge project id                           |
| `{fileId}`     | `8240058`               | CurseForge file id (full digits)                |
| `{fileId4}`    | `8240/58`               | First 4 digits, `/`, then rest without leading 0 |
| `{fileName}`   | `create-...bundled-1.21.1-1.3.0.jar` | Resolved file name                  |
| `{fileNameUrl}`| `create-...bundled-1.21.1-1.3.0.jar` | URL-escaped file name             |
| `{modVersion}` | `1.3.0`                 | Mod's own version parsed from the file name     |
| `{mcVersion}`  | `1.21.1`                | Lock's Minecraft version                        |

`{fileId4}` is the recommended placeholder for the CDN path segment because
both `edge.forgecdn.net` and `mediafilez.forgecdn.net` use the layout
`/files/{first4}/{rest without leading zero}/{fileName}`. Using the
leading-zero form returns 403 on some CDNs.

### Example
```json
{
  "type": "curseforge",
  "query": "Create Aeronautics",
  "slug": "create-aeronautics",
  "urlPattern": "https://mediafilez.forgecdn.net/files/{fileId4}/create-aeronautics-bundled-{mcVersion}-{modVersion}.jar"
}
```

## Spec Fingerprint (lock-time change detection)

`mcmod lock` computes a stable SHA256 fingerprint of each mod's `source`
object (canonical JSON with sorted keys, first 16 bytes hex) and writes it
into the lock file as `mods.<key>.hash`. On the next run, if the spec
fingerprint for a key does not match the value stored in the previous
lock file, the resolver is re-invoked for that mod even if a cached lock
entry exists. This lets you edit a mod's `source` (e.g. update a `slug`
or `urlPattern`) and have just that one mod re-resolve on the next lock.
