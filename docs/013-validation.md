<!-- File: docs/013-validation.md; Created: 2026-09-04; Description: Validation rules. -->
# Validation

`mcmod validate` checks required fields, supported loaders and scopes,
normalized keys, source-specific fields, lock integrity, and release index
uniqueness. Resolved IDs, file names, hashes, and download URLs belong in
locks, never in the root spec.
