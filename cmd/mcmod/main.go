// File: cmd/mcmod/main.go
// Created: 2026-06-20
// Description: Main entry point for the mcmod CLI tool.

package main

import (
	"github.com/orangeboyChen/mcmod-cli/internal/cli"
)

func main() {
	cli.Run()
}

// NewApp is exported for testing.
func NewApp() interface{} {
	return cli.NewApp()
}
