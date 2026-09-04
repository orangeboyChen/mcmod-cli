<!--
File: docs/014-testing.md
Created: 2026-09-04
Description: Test commands.
-->
# Testing

Run `go mod tidy`, `gofmt`, `go test ./... -coverprofile=coverage.out`,
`go tool cover -func=coverage.out`, and `go build ./cmd/mcmod`. Tests must
cover recursive Git expansion, loader filtering, inherited scope, cycles,
namespaced conflicts, lock reconciliation, recursive trees, and project-local
`.mcmod/config.json` and `.mcmod/cache/` isolation.
