<!--
File: docs/009-graph-and-resolution.md
Created: 2026-09-04
Description: Dependency graph and recursive packspec resolution.
-->

# Graph and Resolution

`mcmod lock` resolves a flat lock from a root `packspec.json`. A mod whose
source is `git` is a packspec bundle: mcmod reads its remote `packspec.json`
and recursively expands nested git bundles. Git repositories are not treated
as jar downloads.

## Expansion

```text
root packspec
  └─ git: owner/bundle
       └─ bundle packspec mods
            └─ git: owner/shared
                 └─ shared packspec mods
```

Only mods matching the requested Minecraft loader are included. Child mods
inherit the parent scope when they do not declare one. Expanded keys are
namespaced with the repository, for example `owner-bundle-common`.

## Cycles and conflicts

The expansion stack detects `repo-a -> repo-b -> repo-a` and fails with a
cycle error. A duplicate namespaced key also fails instead of overwriting.

## Resolution boundary

After expansion, every non-git mod is resolved by its declared source. The
expanded dependency set is transient and is not written to the root spec.
