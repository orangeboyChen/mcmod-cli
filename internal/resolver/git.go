// File: internal/resolver/git.go
// Created: 2026-06-20
// Description: Git package resolver - reads via API, not clone.

package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

// ResolveGitPackage clones a git repo and resolves it to a locked source.
func ResolveGitPackage(repo, mcVersion, loader string) (*domain.PackSpec, error) {
	for _, branch := range []string{"main", "master"} {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/packspec.json", repo, branch)
		resp, _, err := netutil.GetWithRetry(http.DefaultClient, url, netutil.DefaultRetry)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}

		var spec domain.PackSpec
		if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		return &spec, nil
	}
	return nil, fmt.Errorf("git: failed to read packspec.json from %s", repo)
}
