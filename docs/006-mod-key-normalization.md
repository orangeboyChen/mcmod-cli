<!--
File: docs/006-mod-key-normalization.md
Created: 2026-06-20
Description: Mod key normalization rules.
-->
# Mod Key Normalization

Rules:
1. Lowercase
2. Spaces, underscores, punctuation → `-`
3. Apostrophes deleted
4. Consecutive `-` collapsed
5. Leading/trailing `-` stripped

Examples:
- `Farmer's Delight` → `farmers-delight`
- `Brewin' And Chewin'` → `brewin-and-chewin`
- `Create Crafts & Additions` → `create-crafts-additions`
- `Greenhouse Config` → `greenhouse-config`