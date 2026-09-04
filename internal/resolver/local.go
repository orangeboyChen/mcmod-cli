// File: internal/resolver/local.go
// Created: 2026-06-20
// Description: Local source resolver per spec section 4.4.

package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// ResolveLocalSource resolves a local jar file into a locked source.
func ResolveLocalSource(path, mcVersion, loader string) (*domain.LockedSource, error) {
	resolved := strings.ReplaceAll(strings.ReplaceAll(path, "{mcVersion}", mcVersion), "{loader}", loader)
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("local: file not found %q", resolved)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("local: path must be a file, not directory %q", resolved)
	}
	if !strings.HasSuffix(resolved, ".jar") {
		return nil, fmt.Errorf("local: file must be a .jar %q", resolved)
	}
	return &domain.LockedSource{Type: "local", Path: resolved, FileName: info.Name()}, nil
}
