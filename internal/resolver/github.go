// File: internal/resolver/github.go
// Created: 2026-06-20
// Description: GitHub Release resolver per spec section 4.2.

package resolver

import (
	"encoding/json"
	"fmt"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
	"net/http"
	"regexp"
	"strings"
)

// GHAPI is the GitHub API base URL.
const GHAPI = "https://api.github.com"

// ResolveGitHubRelease resolves a GitHub release asset matching the given pattern.
func ResolveGitHubRelease(repo, tag, assetPattern, mcVersion, loader string, modKey ...string) (*domain.LockedSource, error) {
	modLabel := strings.Join(modKey, " ")
	labelHTTP := func(req *http.Request) {
		if modLabel != "" {
			req.Header.Set("X-Netutil-Label", modLabel)
		}
	}
	replacedTag := strings.ReplaceAll(tag, "{mcVersion}", mcVersion)
	if strings.Contains(replacedTag, "*") {
		url := fmt.Sprintf("%s/repos/%s/releases", GHAPI, repo)
		req, _ := http.NewRequest("GET", url, nil)
		labelHTTP(req)
		resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
		if err != nil {
			return nil, fmt.Errorf("github releases list failed: %w", err)
		}
		defer resp.Body.Close()

		type Rel struct {
			Tag string `json:"tag_name"`
		}
		var rels []Rel
		if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
			return nil, fmt.Errorf("decode failed: %w", err)
		}

		ptn := regexp.QuoteMeta(replacedTag)
		ptn = strings.ReplaceAll(ptn, `\*`, ".*")
		rg, _ := regexp.Compile("^" + ptn + "$")
		for _, rel := range rels {
			if rg.MatchString(rel.Tag) {
				replacedTag = rel.Tag
				break
			}
		}
	}

	final := assetPattern
	final = strings.ReplaceAll(final, "{tag}", replacedTag)
	final = strings.ReplaceAll(final, "{mcVersion}", mcVersion)
	final = strings.ReplaceAll(final, "{loader}", loader)

	// If the asset name still contains a wildcard we need to pick the actual
	// asset that the release exposes. The tag-wildcard branch above already
	// resolved a concrete release; fetch its asset list and match the pattern.
	if strings.Contains(final, "*") {
		assets, err := listReleaseAssets(repo, replacedTag)
		if err != nil {
			return nil, fmt.Errorf("github assets list failed: %w", err)
		}
		ptn := strings.ReplaceAll(regexp.QuoteMeta(final), `\*`, ".*")
		rg, err := regexp.Compile("^" + ptn + "$")
		if err != nil {
			return nil, fmt.Errorf("github asset pattern %q invalid: %w", final, err)
		}
		matched := ""
		for _, name := range assets {
			if rg.MatchString(name) {
				matched = name
				break
			}
		}
		if matched == "" {
			return nil, fmt.Errorf("github release %s has no asset matching %q", replacedTag, final)
		}
		final = matched
	}

	return &domain.LockedSource{
		Type: "github-release", Repo: repo, Tag: replacedTag,
		AssetName: final, FileName: final,
	}, nil
}

// listReleaseAssets returns the names of every asset attached to the given
// release tag, falling back to an empty slice on any error.
func listReleaseAssets(repo, tag string) ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", GHAPI, repo, tag)
	req, _ := http.NewRequest("GET", url, nil)
	resp, _, err := netutil.DoWithRetry(http.DefaultClient, req, netutil.DefaultRetry)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github release %s returned %d", tag, resp.StatusCode)
	}
	type Asset struct {
		Name string `json:"name"`
	}
	type Release struct {
		Assets []Asset `json:"assets"`
	}
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(r.Assets))
	for _, a := range r.Assets {
		names = append(names, a.Name)
	}
	return names, nil
}

// matchAssetName returns the first asset whose name matches the glob pattern
// in the form produced by replacing the literal "*" with ".*". The list is
// scanned in order so the API's preferred order (typically most-recently
// uploaded) wins ties.
func matchAssetName(assets []string, pattern string) (string, error) {
	ptn := strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*")
	rg, err := regexp.Compile("^" + ptn + "$")
	if err != nil {
		return "", fmt.Errorf("asset pattern %q invalid: %w", pattern, err)
	}
	for _, a := range assets {
		if rg.MatchString(a) {
			return a, nil
		}
	}
	return "", fmt.Errorf("no asset matches %q", pattern)
}
