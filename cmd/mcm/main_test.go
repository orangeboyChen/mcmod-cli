// File: cmd/mcm/main_test.go
// Created: 2026-09-04
// Description: Smoke test for the short mcm CLI entry point.

package main

import "testing"

func TestNewApp(t *testing.T) {
	if app := NewApp(); app == nil {
		t.Fatal("NewApp returned nil")
	}
}
