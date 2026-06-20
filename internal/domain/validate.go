// File: internal/domain/validate.go
// Created: 2026-06-20
// Description: Validation for packspec.json per spec section 3 and 14.

package domain

import (
	"fmt"
	"strings"
)

var supportedLoaders = map[string]bool{"neoforge": true, "fabric": true}

// ValidateSpec validates a PackSpec and returns errors if invalid.
func ValidateSpec(spec PackSpec) error {
	var errs []string

	if spec.PackName == "" {
		errs = append(errs, "packspec.json: packName is required")
	}
	if spec.PackVersion == "" {
		errs = append(errs, "packspec.json: packVersion is required")
	}
	if spec.MinecraftVersion == "" {
		errs = append(errs, "packspec.json: minecraftVersion is required")
	}
	if len(spec.LoaderName) == 0 {
		errs = append(errs, "packspec.json: loaderName must be a non-empty array")
	}

	for _, ln := range spec.LoaderName {
		name, _ := ParseLoaderName(ln)
		if !supportedLoaders[name] {
			errs = append(errs, fmt.Sprintf("packspec.json: unsupported loader %q (supported: neoforge, fabric)", ln))
		}
	}

	// Check mods from unified map
	for key, mod := range spec.Mods {
		// Validate mod key is normalized
		normalizedKey := NormalizeModKey(key)
		if key != normalizedKey {
			errs = append(errs, fmt.Sprintf("packspec.json: mod key %q is not normalized (expected %q)", key, normalizedKey))
			continue
		}
		if mod.Source.Type == "" {
			errs = append(errs, fmt.Sprintf("packspec.json: mod %q has empty source type", key))
			continue
		}
		if mod.Scope != "" && !isValidScope(mod.Scope) {
			errs = append(errs, fmt.Sprintf("packspec.json: mod %q has invalid scope %q", key, mod.Scope))
		}
		if mod.Loader != nil {
			for _, l := range mod.Loader {
				if !supportedLoaders[l] {
					errs = append(errs, fmt.Sprintf("packspec.json: mod %q has unsupported loader %q", key, l))
				}
			}
		}
		if err := validateSourceMod(mod); err != "" {
			errs = append(errs, fmt.Sprintf("packspec.json: mod %q: %s", key, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func isValidScope(s string) bool {
	return s == ScopeShared || s == ScopeClient || s == ScopeServer
}

func validateSourceMod(mod ModSpec) string {
	switch mod.Source.Type {
	case SourceCurseForge:
		if mod.Source.Query == "" {
			return "curseforge source requires query"
		}
		if mod.Source.Slug != "" {
			// Per spec 4.1.0 packspec.json only accepts the `query` field
			// for CurseForge sources. The legacy `slug` field is a lock
			// artifact and must not be written back into the spec.
			return "curseforge source must not contain slug (use query only)"
		}
		if mod.Source.ModID != 0 || mod.Source.FileID != 0 {
			return "curseforge source must not contain modId/fileId (use query only)"
		}
		if mod.Source.FileName != "" {
			return "curseforge source must not contain fileName (use query only)"
		}
		return ""
	case SourceGitHubRelease:
		if mod.Source.Repo == "" {
			return "github-release source requires repo"
		}
		if mod.Source.Tag == "" {
			return "github-release source requires tag"
		}
		if mod.Source.AssetPattern == "" && len(mod.Source.AssetPatternByLoader) == 0 {
			return "github-release source requires assetPattern"
		}
		if mod.Source.FileName != "" {
			return "github-release source must not contain fileName"
		}
		return ""
	case SourceGit:
		if mod.Source.Repo == "" {
			return "git source requires repo"
		}
		if mod.Source.Query != "" {
			return "git source must not contain query"
		}
		return ""
	case SourceLocal:
		if mod.Source.Path == "" {
			return "local source requires path"
		}
		if mod.Source.URL != "" {
			return "local source must not contain URL"
		}
		return ""
	default:
		return fmt.Sprintf("unknown source type %q", mod.Source.Type)
	}
}

// ValidateLoaderName checks if a loader name is supported.
func ValidateLoaderName(name string) error {
	if !supportedLoaders[name] {
		supported := []string{"neoforge", "fabric"}
		return fmt.Errorf("unsupported loader %q (supported: %s)", name, strings.Join(supported, ", "))
	}
	return nil
}

// ValidateLock validates a PackLock file.
func ValidateLock(lock PackLock) error {
	if lock.Loader == "" {
		return fmt.Errorf("lock: loader is required")
	}
	if !supportedLoaders[lock.Loader] {
		return fmt.Errorf("lock: unsupported loader %q (supported: neoforge, fabric)", lock.Loader)
	}
	if lock.MinecraftVersion == "" {
		return fmt.Errorf("lock: minecraftVersion is required")
	}
	if len(lock.Mods) == 0 {
		return fmt.Errorf("lock: mods must not be empty")
	}
	for key, mod := range lock.Mods {
		if mod.Source.Type == "" {
			return fmt.Errorf("lock: mod %q has empty source type", key)
		}
		switch mod.Source.Type {
		case SourceCurseForge:
			if mod.Source.ModID == 0 || mod.Source.FileName == "" {
				return fmt.Errorf("lock: mod %q curseforge source requires modId and fileName", key)
			}
		case SourceGitHubRelease:
			if mod.Source.Repo == "" || mod.Source.Tag == "" || (mod.Source.AssetName == "" && mod.Source.LockAssetName == "") {
				return fmt.Errorf("lock: mod %q github-release source requires repo, tag, and fileName", key)
			}
		case SourceGit:
			if mod.Source.Repo == "" {
				return fmt.Errorf("lock: mod %q git source requires repo", key)
			}
		case SourceLocal:
			if mod.Source.Path == "" {
				return fmt.Errorf("lock: mod %q local source requires path", key)
			}
		case SourceURL:
			if mod.Source.URL == "" {
				return fmt.Errorf("lock: mod %q URL source requires URL", key)
			}
		}
	}
	return nil
}

// ValidateReleaseIndex validates a ReleaseIndex.
func ValidateReleaseIndex(ri ReleaseIndex) error {
	if ri.Type == "" {
		return fmt.Errorf("release index: type is required")
	}
	if ri.PackName == "" {
		return fmt.Errorf("release index: packName is required")
	}
	if ri.MinecraftVersion == "" {
		return fmt.Errorf("release index: minecraftVersion is required")
	}
	seenVersions := make(map[string]bool)
	for _, r := range ri.Releases {
		if seenVersions[r.Version] {
			return fmt.Errorf("release index: duplicate version %q", r.Version)
		}
		seenVersions[r.Version] = true
		if r.Type == "" {
			return fmt.Errorf("release index: release type is required for version %q", r.Version)
		}
	}
	return nil
}
