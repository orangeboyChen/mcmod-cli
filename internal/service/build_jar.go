// File: internal/service/build_jar.go
// Created: 2026-06-20
// Description: Build-time jar resolution and cache population.

package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/cache"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/downloader"
)

// resolveModJar returns the on-disk path to the mod jar for a given lock entry.
// local sources use the spec's path directly; remote sources look in .cache.
func (bc *buildContext) resolveModJar(key string, locked domain.LockedMod) (string, error) {
	src := locked.Source
	switch src.Type {
	case "local":
		var path string
		if bc.Spec != nil {
			if mod, ok := bc.Spec.Mods[key]; ok {
				path = mod.Source.Path
			}
		}
		if path == "" {
			path = src.Path
		}
		// Fallback: if the path is empty but FileName is set, try to find it in
		// .cache/local/ and the project root.
		if path == "" && src.FileName != "" {
			candidates := []string{
				filepath.Join(bc.RootDir, ".cache", "local", src.FileName),
				filepath.Join(bc.RootDir, src.FileName),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					path = c
					break
				}
			}
		}
		if path == "" {
			return "", fmt.Errorf("mod %s: local source missing path", key)
		}
		// Replace template variables in path
		path = strings.ReplaceAll(path, "{mcVersion}", bc.McVersion)
		path = strings.ReplaceAll(path, "{loader}", bc.Loader)
		if !filepath.IsAbs(path) {
			path = filepath.Join(bc.RootDir, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("mod %s: local file %q not found: %w", key, path, err)
		}
		return path, nil
	case "curseforge":
		if src.ModID == 0 || src.FileID == 0 || src.FileName == "" {
			return "", fmt.Errorf("mod %s: curseforge source missing modId/fileId/fileName in lock", key)
		}
		p := filepath.Join(bc.RootDir, ".cache", "curseforge", fmt.Sprint(src.ModID), fmt.Sprint(src.FileID), src.FileName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		if err := bc.populateCache(key, &src); err != nil {
			return "", fmt.Errorf("mod %s: %w", key, err)
		}
		return p, nil
	case "github-release":
		if src.Repo == "" || src.Tag == "" || src.AssetName == "" {
			return "", fmt.Errorf("mod %s: github-release source missing repo/tag/assetName in lock", key)
		}
		parts := strings.SplitN(src.Repo, "/", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("mod %s: invalid github repo %q", key, src.Repo)
		}
		p := filepath.Join(bc.RootDir, ".cache", "github-release", parts[0], parts[1], src.Tag, src.AssetName)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		if err := bc.populateCache(key, &src); err != nil {
			return "", fmt.Errorf("mod %s: %w", key, err)
		}
		return p, nil
	default:
		return "", fmt.Errorf("mod %s: unsupported source type %q in lock", key, src.Type)
	}
}

// populateCache downloads a remote mod jar into the local .cache/ tree on
// demand. It is called when resolveModJar cannot find a cached file so the
// build step stays self-contained (lock is purely a resolve step).
func (bc *buildContext) populateCache(key string, src *domain.LockedSource) error {
	if err := cache.EnsureCacheDir(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "build: caching mod %s from %s\n", key, src.Type)
	if err := downloader.Download(src, key); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "build: cached mod %s\n", key)
	return nil
}

// modsForTarget returns mod keys that should be packaged for the given target.
// client: shared + client; server: shared + server.
func (bc *buildContext) modsForTarget(target string) []string {
	keys := make([]string, 0)
	for key, locked := range bc.Lock.Mods {
		scope := locked.Scope
		if scope == "" {
			scope = "shared"
		}
		if target == "client" {
			if scope == "shared" || scope == "client" {
				keys = append(keys, key)
			}
		} else if target == "server" {
			if scope == "shared" || scope == "server" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}
