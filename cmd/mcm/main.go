// File: cmd/mcm/main.go
// Created: 2026-09-04
// Description: Short mcm executable entry point for the mcmod CLI.

package main

import "github.com/orangeboyChen/mcmod-cli/internal/cli"

func main() {
	cli.Run()
}

// NewApp is exported for testing.
func NewApp() interface{} {
	return cli.NewApp()
}
