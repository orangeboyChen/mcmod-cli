// File: internal/resolver/git.go
// Created: 2026-06-20
// Description: Git package resolver - reads via API, not clone.

package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/netutil"
)

// ResolveGitPackage reads the root packspec.json from a GitHub repository.
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
		if err := domain.ValidateSpec(spec); err != nil {
			return nil, fmt.Errorf("git: invalid packspec.json from %s: %w", repo, err)
		}
		return &spec, nil
	}
	return nil, fmt.Errorf("git: failed to read packspec.json from %s", repo)
}

// ExpandGitDependencies expands git-backed packspec entries into a flat mod
// map. Git sources are treated as packspec bundles rather than downloadable
// jars. Nested bundles are expanded recursively and their child keys are
// prefixed with the normalized repository path.
func ExpandGitDependencies(spec *domain.PackSpec, mcVersion, loader string) (map[string]domain.ModSpec, error) {
	if spec == nil {
		return nil, fmt.Errorf("git: packspec is nil")
	}
	result := make(map[string]domain.ModSpec)
	for key, mod := range collectMods(*spec) {
		if !modAppliesToLoader(mod, loader) {
			continue
		}
		if err := expandGitMod(result, key, mod, "", mcVersion, loader, make(map[string]bool)); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func expandGitMod(result map[string]domain.ModSpec, key string, mod domain.ModSpec, prefix, mcVersion, loader string, stack map[string]bool) error {
	if mod.Source.Type != domain.SourceGit {
		finalKey := key
		if prefix != "" {
			finalKey = prefix + "-" + domain.NormalizeModKey(key)
		}
		if _, exists := result[finalKey]; exists {
			return fmt.Errorf("git: duplicate expanded mod key %q", finalKey)
		}
		result[finalKey] = mod
		return nil
	}
	if mod.Source.Repo == "" {
		return fmt.Errorf("git: mod %q has empty repository", key)
	}
	repoID := canonicalRepoID(mod.Source.Repo)
	if stack[repoID] {
		return fmt.Errorf("git: recursive packspec cycle detected at %s", mod.Source.Repo)
	}
	stack[repoID] = true
	defer delete(stack, repoID)

	childSpec, err := ResolveGitPackage(mod.Source.Repo, mcVersion, loader)
	if err != nil {
		return fmt.Errorf("git: expand %s: %w", mod.Source.Repo, err)
	}
	if len(childSpec.LoaderName) > 0 && !loaderMatches(childSpec.LoaderName, loader) {
		return fmt.Errorf("git: packspec %s does not support loader %q", mod.Source.Repo, loader)
	}
	repoPrefix := domain.NormalizeModKey(strings.ReplaceAll(repoID, "/", "-"))
	if prefix != "" {
		repoPrefix = prefix + "-" + repoPrefix
	}
	for childKey, childMod := range collectMods(*childSpec) {
		if !modAppliesToLoader(childMod, loader) {
			continue
		}
		if childMod.Scope == "" {
			childMod.Scope = mod.Scope
		}
		if err := expandGitMod(result, childKey, childMod, repoPrefix, mcVersion, loader, stack); err != nil {
			return err
		}
	}
	return nil
}

func canonicalRepoID(repo string) string {
	return strings.ToLower(strings.TrimSpace(repo))
}

func collectMods(spec domain.PackSpec) map[string]domain.ModSpec {
	mods := make(map[string]domain.ModSpec, len(spec.Mods))
	for key, mod := range spec.Mods {
		mods[key] = mod
	}
	if len(mods) > 0 {
		return mods
	}
	add := func(entries []domain.ModSpec, scope string) {
		for _, mod := range entries {
			if mod.Scope == "" {
				mod.Scope = scope
			}
			key := domain.NormalizeModKey(mod.Name)
			if key != "" {
				mods[key] = mod
			}
		}
	}
	add(spec.SharedMods, domain.ScopeShared)
	add(spec.ClientMods, domain.ScopeClient)
	add(spec.ServerMods, domain.ScopeServer)
	add(spec.Dependencies, domain.ScopeShared)
	return mods
}

func loaderMatches(loaders []string, loader string) bool {
	for _, entry := range loaders {
		name, _ := domain.ParseLoaderName(entry)
		if name == loader {
			return true
		}
	}
	return false
}

func modAppliesToLoader(mod domain.ModSpec, loader string) bool {
	if len(mod.Loader) == 0 {
		return true
	}
	return loaderMatches(mod.Loader, loader)
}
