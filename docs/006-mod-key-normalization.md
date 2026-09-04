<!-- File: docs/006-mod-key-normalization.md; Created: 2026-09-04; Description: Mod keys. -->
# Mod Keys

Keys are lowercased; spaces, underscores, and punctuation become `-`;
apostrophes are removed; repeated and edge hyphens are collapsed or trimmed.
`Farmer's Delight` becomes `farmers-delight`.

Expanded Git keys use a normalized repository prefix such as
`owner-bundle-common`.
