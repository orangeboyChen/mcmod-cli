<!--
File: docs/001-spec.md
Created: 2026-09-04
Description: Defines the root packspec.json contract.
-->

# Pack Specification

`packspec.json` at the project root is the only hand-edited dependency input.
It requires `packName`, `packVersion`, `minecraftVersion`, and `loaderName`.
`loaderName` contains `neoforge:<version>` and/or `fabric:<version>`.

`mods` is a map keyed by normalized names. Each mod has optional `name`,
`scope`, and `loader`, plus a required `source` object. Scope defaults to
`shared`; supported scopes are `shared`, `client`, and `server`.

```json
{
  "packName": "example",
  "packVersion": "0.1.0",
  "minecraftVersion": "1.21.1",
  "loaderName": ["neoforge:21.1.219"],
  "mods": {
    "bundle": {"source": {"type": "git", "repo": "owner/bundle"}},
    "jei": {"scope": "client", "source": {"type": "curseforge", "query": "Just Enough Items"}}
  }
}
```

Do not write resolved IDs, asset names, hashes, or download URLs into this
file. Those are lock-file outputs.
