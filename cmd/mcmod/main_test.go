// File: cmd/mcmod/main_test.go
// Created: 2026-06-20
// Description: Smoke test for CLI entry point.
package main

import (
	"testing"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp returned nil")
	}
}
