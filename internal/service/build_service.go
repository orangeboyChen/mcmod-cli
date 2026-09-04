// File: internal/service/build_service.go
// Created: 2026-06-20
// Description: Build artifact generation entry points.

package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// buildContext carries resolved information for one (mc, loader) build run.
type buildContext struct {
	Spec          *domain.PackSpec
	Lock          *domain.PackLock
	McVersion     string
	Loader        string
	LoaderVersion string
	RootDir       string
}

// BuildArtifact builds the artifact(s) for a single (mcVersion, target) pair
// using the default (non-forced) semantics. See BuildArtifactWith for the
// long-form documentation and force handling.
func BuildArtifact(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string) error {
	return BuildArtifactWith(spec, lock, mcVersion, target, false)
}

// BuildArtifactAndReturnPath behaves like BuildArtifactWith but returns the
// on-disk zip path that was produced for the given target. It is used by the
// CLI to print a "artifact <target>: <path>" line per spec 7.5.
func BuildArtifactAndReturnPath(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) (string, error) {
	loader, loaderVer := loaderFromLock(lock)
	bc := &buildContext{
		Spec:          spec,
		Lock:          lock,
		McVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVer,
		RootDir:       ".",
	}
	if target == "both" || target == "" {
		if err := buildOneArtifactWith(spec, lock, mcVersion, "client", force); err != nil {
			return "", err
		}
		if err := buildOneArtifactWith(spec, lock, mcVersion, "server", force); err != nil {
			return "", err
		}
		return bc.zipPath("both"), nil
	}
	if err := buildOneArtifactWith(spec, lock, mcVersion, target, force); err != nil {
		return "", err
	}
	return bc.zipPath(target), nil
}

// BuildArtifactWith behaves like BuildArtifact but optionally overwrites
// existing artifacts when force is true.
func BuildArtifactWith(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}
	if lock == nil {
		return fmt.Errorf("lock is nil")
	}
	if target == "both" || target == "" {
		if err := buildOneArtifactWith(spec, lock, mcVersion, "client", force); err != nil {
			return err
		}
		return buildOneArtifactWith(spec, lock, mcVersion, "server", force)
	}
	return buildOneArtifactWith(spec, lock, mcVersion, target, force)
}

// ValidateBuildLock ensures every mod in the packspec has a corresponding
// entry in the dependency lock before any artifact is written.
func ValidateBuildLock(spec *domain.PackSpec, lock *domain.PackLock, target string) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}
	if lock == nil {
		return fmt.Errorf("lock is nil")
	}
	missing := make([]string, 0)
	for key, mod := range spec.Mods {
		if len(mod.Loader) > 0 && !containsLoader(mod.Loader, lock.Loader) {
			continue
		}
		if target == "client" && mod.Scope == domain.ScopeServer {
			continue
		}
		if target == "server" && mod.Scope == domain.ScopeClient {
			continue
		}
		if _, ok := lock.Mods[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("build: dependency lock is incomplete; missing mod(s): %s\nhint: run `mcmod lock %s %s` before building", strings.Join(missing, ", "), lock.MinecraftVersion, lock.Loader)
	}
	return nil
}

func containsLoader(loaders []string, loader string) bool {
	for _, candidate := range loaders {
		if candidate == loader {
			return true
		}
	}
	return false
}

// buildOneArtifact creates a single zip for the given target.
func buildOneArtifact(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string) error {
	return buildOneArtifactWith(spec, lock, mcVersion, target, false)
}

// buildOneArtifactWith creates a single zip for the given target, optionally
// allowing overwrite of an existing artifact when force is true.
func buildOneArtifactWith(spec *domain.PackSpec, lock *domain.PackLock, mcVersion, target string, force bool) error {
	if target != "client" && target != "server" {
		return fmt.Errorf("unsupported build target %q (want client, server, or both)", target)
	}
	if err := ValidateBuildLock(spec, lock, target); err != nil {
		return err
	}
	loader, loaderVer := loaderFromLock(lock)
	bc := &buildContext{
		Spec:          spec,
		Lock:          lock,
		McVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVer,
		RootDir:       ".",
	}
	keys := bc.modsForTarget(target)
	if len(keys) == 0 {
		return fmt.Errorf("no mods for target %s", target)
	}
	modFiles := make(map[string]string)
	for _, key := range keys {
		locked, ok := lock.Mods[key]
		if !ok {
			return fmt.Errorf("mod %s missing from lock", key)
		}
		jarPath, err := bc.resolveModJar(key, locked)
		if err != nil {
			return fmt.Errorf("build: resolve mod %s: %w", key, err)
		}
		modFiles[key] = jarPath
	}
	if len(modFiles) == 0 {
		return fmt.Errorf("no mod jars resolved for target %s", target)
	}
	out := bc.zipPath(target)
	if err := bc.buildZipWith(target, out, modFiles, force); err != nil {
		return err
	}
	return nil
}

// BuildClientServerBuild processes the full build pipeline for all loaders.
func BuildClientServerBuild(spec *domain.PackSpec, mcVersion string) error {
	for _, ln := range spec.LoaderName {
		loader, _ := domain.ParseLoaderName(ln)
		lock, err := LoadLock(mcVersion, loader)
		if err != nil {
			return fmt.Errorf("load lock for %s: %w", loader, err)
		}
		if err := BuildArtifact(spec, lock, mcVersion, "client"); err != nil {
			return fmt.Errorf("build client %s: %w", loader, err)
		}
		if err := BuildArtifact(spec, lock, mcVersion, "server"); err != nil {
			return fmt.Errorf("build server %s: %w", loader, err)
		}
	}
	return nil
}

// loaderFromLock returns the loader name and version for a PackLock.
func loaderFromLock(lock *domain.PackLock) (string, string) {
	if lock == nil {
		return "", ""
	}
	return lock.Loader, lock.LoaderVersion
}
