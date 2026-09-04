<!-- File: docs/007-scope-and-loader.md; Created: 2026-09-04; Description: Scope and loader rules. -->
# Scope and Loader

`shared` is packaged for both targets; `client` and `server` are packaged
only for their target. A Git bundle child inherits its parent scope when it
does not declare one. Empty `loader` means all supported loaders; otherwise
the requested loader must be listed.
