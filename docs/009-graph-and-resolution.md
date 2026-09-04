<!-- File: docs/009-graph-and-resolution.md; Created: 2026-09-04; Description: Recursive Git expansion. -->
# Graph and Resolution

The graph starts with root spec mods. A Git source loads a remote spec and
adds matching children; nested Git sources are expanded until all reachable
sources are concrete. Active repository stacks detect cycles, and duplicate
namespaced keys fail explicitly. The expansion is transient and is flattened
into the lock file.
