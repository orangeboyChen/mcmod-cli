// File: internal/cli/commands_lock.go
// Created: 2026-06-20
// Description: Implements `mcmod lock` subcommand tree (lock/list/show/add/update/delete/tree/release).

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
)

func newLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock [<minecraftVersion>] [<loader>]",
		Short: "Resolve and lock mod dependencies",
		Long: `Generate or update dependency lock files.

Subcommands:
  lock list                      List lock entries
  lock show [<mc>] [<loader>]    Show lock details
  lock show <mc> <loader> <key>  Show single lock entry
  lock add <mc> <loader> <key>   Add lock-only entry
  lock update [<mc>] [<loader>]  Refresh locks
  lock update <mc> <loader> <key> Update single entry
  lock delete [<mc>] [<loader>]  Delete locks
  lock delete <mc> <loader> <key> Delete single entry
  lock release set ...           Set release index
  lock release list [<mc>]      List releases
  lock release show <mc> <ver>  Show release
  lock release delete <mc> <ver> Delete release`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := ""
			loader := ""
			if len(args) > 0 {
				mcVersion = args[0]
			}
			if len(args) > 1 {
				loader = args[1]
			}
			spec, err := domain.ReadPackSpec(".")
			if err != nil {
				return fmt.Errorf("lock: read packspec\nhint: create packspec.json in the project root")
			}
			versions := []string{mcVersion}
			if mcVersion == "" {
				versions = []string{spec.MinecraftVersion}
			}
			loaders := []string{loader}
			if loader == "" {
				for _, ln := range spec.LoaderName {
					name, _ := domain.ParseLoaderName(ln)
					loaders = append(loaders, name)
				}
				loaders = loaders[1:]
			}
			failed := false
			for _, v := range versions {
				for _, l := range loaders {
					if !domain.LoaderMatches(*spec, l) {
						fmt.Fprintf(os.Stderr, "error: lock: unsupported loader %q\nhint: use neoforge or fabric\n", l)
						failed = true
						continue
					}
					// Incremental lock: carry over mods that already resolved
					// in a previous run so we only re-resolve new or failed mods.
					existing, _ := service.ReadLockFile(service.LockFilePath(v, l))
					lock, resolveFailures, err := service.BuildLockWithExisting(spec, v, l, existing)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: lock %s %s: %v\n", v, l, err)
						failed = true
						continue
					}
					// A lock with zero mods is still valid (it just means the
					// spec has no mod entries for this loader). Write the empty
					// lock so subsequent lock runs can carry it forward.
					_ = lock.Mods // keep the var referenced
					if err := service.WriteLockFile(lock); err != nil {
						fmt.Fprintf(os.Stderr, "error: write lock %s %s: %v\n", v, l, err)
						failed = true
						continue
					}
					fmt.Printf("locked %s %s -> locks/dependencies/%s-%s.json\n", v, l, v, l)
					if resolveFailures > 0 {
						// Mods that the resolver could not pin are
						// intentionally left out of the lock file (the
						// file is a clean source of truth per
						// docs/003-lock-files.md). Surface the partial
						// failure as a hard error so a downstream
						// `mcmod build` does not silently drop the
						// missing mod.
						fmt.Fprintf(os.Stderr, "error: lock %s %s: %d mod(s) failed to resolve; lock file is partial\nhint: see the per-mod failure list above and re-run `mcmod lock %s %s` once they are fixed\n", v, l, resolveFailures, v, l)
						failed = true
					}
				}
			}
			if failed {
				return fmt.Errorf("lock failed for one or more (minecraftVersion, loader) pairs")
			}
			return nil
		},
	}
	cmd.AddCommand(newLockListCmd())
	cmd.AddCommand(newLockShowCmd())
	cmd.AddCommand(newLockAddCmd())
	cmd.AddCommand(newLockUpdateCmd())
	cmd.AddCommand(newLockDeleteCmd())
	cmd.AddCommand(newLockTreeCmd())
	cmd.AddCommand(newLockReleaseCmd())

	return cmd
}

func newLockListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [<minecraftVersion>] [<loader>]",
		Short: "List lock entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := "1.21.1"
			loader := "neoforge"
			var spec *domain.PackSpec
			if s, err := domain.ReadPackSpec("."); err == nil {
				spec = s
				if v := domain.DefaultMCVersion(*s); v != "" {
					mcVersion = v
				}
				if l := domain.DefaultLoader(*s); l != "" {
					loader = l
				}
			}
			if len(args) > 0 {
				mcVersion = args[0]
			}
			if len(args) > 1 {
				loader = args[1]
			}
			loaders := []string{loader}
			if loader == "" && spec != nil && len(spec.LoaderName) > 0 {
				loaders = domain.DefaultLoaders(*spec)
			}
			if len(loaders) == 0 {
				loaders = []string{"neoforge"}
			}
			failed := false
			for _, l := range loaders {
				if err := printLockList(mcVersion, l); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					failed = true
				}
			}
			if failed {
				return fmt.Errorf("lock list failed for one or more loaders")
			}
			return nil
		},
	}
}

// printLockList prints the lock entry list for a single (mcVersion, loader)
// pair. It is shared by `lock list` and any future alias so the format
// stays consistent across multi-loader invocations.
func printLockList(mcVersion, loader string) error {
	lock, err := service.LoadLock(mcVersion, loader)
	if err != nil {
		return fmt.Errorf("lock list: lock not found for %s/%s\nhint: run mcmod lock %s %s first", mcVersion, loader, mcVersion, loader)
	}
	scopes := []string{"server", "client", "shared"}
	fmt.Printf("lock %s %s\n", mcVersion, loader)
	for _, scope := range scopes {
		fmt.Printf("[%s]\n", strings.Title(scope))
		var lines []string
		for key, m := range lock.Mods {
			if m.Scope == scope || (scope == "shared" && m.Scope == "") {
				lines = append(lines, fmt.Sprintf("  %s | %s | %s | %s | %s", key, m.Name, m.Version, m.Source.Type, m.Source.FileName))
			}
		}
		sort.Strings(lines)
		if len(lines) == 0 {
			fmt.Println("  (empty)")
		} else {
			for _, l := range lines {
				fmt.Println(l)
			}
		}
	}
	return nil
}

func newLockShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [<minecraftVersion>] [<loader>] [<key>]",
		Short: "Show lock details",
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := "1.21.1"
			loader := "neoforge"
			var spec *domain.PackSpec
			if s, err := domain.ReadPackSpec("."); err == nil {
				spec = s
				if v := domain.DefaultMCVersion(*s); v != "" {
					mcVersion = v
				}
				if l := domain.DefaultLoader(*s); l != "" {
					loader = l
				}
			}
			if len(args) > 0 {
				mcVersion = args[0]
			}
			if len(args) > 1 {
				loader = args[1]
			}
			loaders := []string{loader}
			if loader == "" && spec != nil && len(spec.LoaderName) > 0 {
				loaders = domain.DefaultLoaders(*spec)
			}
			if len(loaders) == 0 {
				loaders = []string{"neoforge"}
			}
			if len(args) >= 3 {
				// key given: require a single loader; default if missing
				targetLoader := loader
				if targetLoader == "" {
					targetLoader = domain.DefaultLoader(*spec)
					if targetLoader == "" && len(loaders) > 0 {
						targetLoader = loaders[0]
					}
				}
				return printLockEntry(mcVersion, targetLoader, args[2])
			}
			// no key: show each lock file
			failed := false
			for _, l := range loaders {
				if err := printLockFile(mcVersion, l); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					failed = true
				}
			}
			if failed {
				return fmt.Errorf("lock show failed for one or more loaders")
			}
			return nil
		},
	}
}

// printLockFile prints the full JSON contents of a single lock file.
func printLockFile(mcVersion, loader string) error {
	lock, err := service.LoadLock(mcVersion, loader)
	if err != nil {
		return fmt.Errorf("lock show: lock not found for %s/%s\nhint: run mcmod lock %s %s first", mcVersion, loader, mcVersion, loader)
	}
	data, _ := json.MarshalIndent(lock, "", "  ")
	fmt.Println(string(data))
	return nil
}

// buildLockedSource constructs a domain.LockedSource from flag values and
// validates required fields per source type (spec 7.3.7). It also clears
// fields that do not belong to the requested source type so a switch from
// curseforge to github-release does not leave behind stale modId / fileId.
func buildLockedSource(sourceType, modID, fileID, fileName, repo, tag, assetName, sourcePath string) (domain.LockedSource, error) {
	ls := domain.LockedSource{Type: sourceType}
	switch sourceType {
	case "curseforge":
		if modID == "" || fileID == "" || fileName == "" {
			return ls, fmt.Errorf("source curseforge requires --mod-id, --file-id, and --file-name")
		}
		if _, err := fmt.Sscanf(modID, "%d", &ls.ModID); err != nil {
			return ls, fmt.Errorf("invalid --mod-id %q", modID)
		}
		if _, err := fmt.Sscanf(fileID, "%d", &ls.FileID); err != nil {
			return ls, fmt.Errorf("invalid --file-id %q", fileID)
		}
		ls.FileName = fileName
	case "github-release":
		if repo == "" || tag == "" || assetName == "" || fileName == "" {
			return ls, fmt.Errorf("source github-release requires --repo, --tag, --asset-name, and --file-name")
		}
		ls.Repo = repo
		ls.Tag = tag
		ls.AssetName = assetName
		ls.FileName = fileName
	case "git":
		if repo == "" || fileName == "" {
			return ls, fmt.Errorf("source git requires --repo and --file-name")
		}
		ls.Repo = repo
		ls.FileName = fileName
	case "local":
		if sourcePath == "" {
			return ls, fmt.Errorf("source local requires --path")
		}
		ls.Path = sourcePath
		ls.FileName = fileName
	default:
		return ls, fmt.Errorf("unsupported --source %q (must be curseforge, github-release, git, or local)", sourceType)
	}
	return ls, nil
}

// printLockEntry prints a single key's lock entry.
func printLockEntry(mcVersion, loader, key string) error {
	lock, err := service.LoadLock(mcVersion, loader)
	if err != nil {
		return fmt.Errorf("lock show: lock not found for %s/%s\nhint: run mcmod lock %s %s first", mcVersion, loader, mcVersion, loader)
	}
	m, ok := lock.Mods[key]
	if !ok {
		return fmt.Errorf("lock show: key %q not found in lock for %s/%s", key, mcVersion, loader)
	}
	fmt.Printf("key: %s\nname: %s\nversion: %s\nscope: %s\nsource:\n  type: %s\n", key, m.Name, m.Version, m.Scope, m.Source.Type)
	if m.Source.ModID != 0 {
		fmt.Printf("  modId: %d\n", m.Source.ModID)
	}
	if m.Source.FileID != 0 {
		fmt.Printf("  fileId: %d\n", m.Source.FileID)
	}
	if m.Source.FileName != "" {
		fmt.Printf("  fileName: %s\n", m.Source.FileName)
	}
	if m.Source.Repo != "" {
		fmt.Printf("  repo: %s\n", m.Source.Repo)
	}
	if m.Source.Tag != "" {
		fmt.Printf("  tag: %s\n", m.Source.Tag)
	}
	return nil
}

func newLockAddCmd() *cobra.Command {
	var name, version, scope, sourceType, modID, fileID, fileName, repo, tag, assetName, sourcePath string
	cmd := &cobra.Command{
		Use:   "add <minecraftVersion> <loader> <key> [options]",
		Short: "Add lock-only entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("lock add: need minecraftVersion, loader, and key")
			}
			mcVersion, loader, key := args[0], args[1], args[2]
			s := domain.ScopeShared
			if scope != "" {
				s = scope
			}
			ls, err := buildLockedSource(sourceType, modID, fileID, fileName, repo, tag, assetName, sourcePath)
			if err != nil {
				return fmt.Errorf("lock add: %v\nhint: provide the required flags for --source %s", err, sourceType)
			}
			lock, err := service.LoadLock(mcVersion, loader)
			if err != nil {
				lock = &domain.PackLock{Loader: loader, MinecraftVersion: mcVersion, Mods: make(map[string]domain.LockedMod)}
			}
			if _, exists := lock.Mods[key]; exists {
				return fmt.Errorf("lock add: key %q already exists in lock for %s/%s", key, mcVersion, loader)
			}
			lock.Mods[key] = domain.LockedMod{Name: name, Version: version, Scope: s, Source: ls}
			if err := service.SaveLock(mcVersion, loader, lock); err != nil {
				return fmt.Errorf("lock add: %w", err)
			}
			fmt.Printf("added lock mod %s -> locks/dependencies/%s-%s.json\n", key, mcVersion, loader)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringVar(&version, "version", "", "Mod version")
	cmd.Flags().StringVar(&scope, "scope", "shared", "Scope: shared, client, server")
	cmd.Flags().StringVar(&sourceType, "source", "curseforge", "Source type")
	cmd.Flags().StringVar(&modID, "mod-id", "", "CurseForge mod ID")
	cmd.Flags().StringVar(&fileID, "file-id", "", "CurseForge file ID")
	cmd.Flags().StringVar(&fileName, "file-name", "", "Download file name")
	cmd.Flags().StringVar(&repo, "repo", "", "Repository")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag")
	cmd.Flags().StringVar(&assetName, "asset-name", "", "Release asset name")
	cmd.Flags().StringVar(&sourcePath, "path", "", "Local file path")

	return cmd
}

func newLockUpdateCmd() *cobra.Command {
	var fName, fVersion, fScope, fSource, fModID, fFileID, fFileName, fRepo, fTag, fAssetName, fPath string
	cmd := &cobra.Command{
		Use:   "update [<minecraftVersion>] [<loader>] [<key>]",
		Short: "Refresh or update lock entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return runLockUpdateBulk(args)
			}
			mcVersion, loader, key := args[0], args[1], args[2]
			lock, err := service.LoadLock(mcVersion, loader)
			if err != nil {
				return fmt.Errorf("lock update: lock not found for %s/%s\nhint: run mcmod lock %s %s first", mcVersion, loader, mcVersion, loader)
			}
			m, ok := lock.Mods[key]
			if !ok {
				return fmt.Errorf("lock update: key %q not found in %s/%s", key, mcVersion, loader)
			}
			if fName != "" {
				m.Name = fName
			}
			if fVersion != "" {
				m.Version = fVersion
			}
			if fScope != "" {
				m.Scope = fScope
			}
			if fSource != "" {
				// Explicit --source switch: validate the new source and
				// overwrite the entry's source wholesale. Other fields
				// (name, version, scope) are kept as-is.
				ls, err := buildLockedSource(fSource, fModID, fFileID, fFileName, fRepo, fTag, fAssetName, fPath)
				if err != nil {
					return fmt.Errorf("lock update: %w", err)
				}
				m.Source = ls
			} else {
				// No --source change. Touch individual source fields by
				// name (--file-name, --repo, etc.) starting from the
				// existing source so unchanged fields are preserved. We
				// deliberately do NOT call buildLockedSource here: that
				// would re-validate required fields the operator may not
				// have intended to retype (e.g. updating only --version
				// would still demand every curseforge id field).
				ls := m.Source
				changed := false
				if fModID != "" {
					if _, err := fmt.Sscanf(fModID, "%d", &ls.ModID); err == nil {
						changed = true
					}
				}
				if fFileID != "" {
					if _, err := fmt.Sscanf(fFileID, "%d", &ls.FileID); err == nil {
						changed = true
					}
				}
				if fFileName != "" {
					ls.FileName = fFileName
					changed = true
				}
				if fRepo != "" {
					ls.Repo = fRepo
					changed = true
				}
				if fTag != "" {
					ls.Tag = fTag
					changed = true
				}
				if fAssetName != "" {
					ls.AssetName = fAssetName
					changed = true
				}
				if fPath != "" {
					ls.Path = fPath
					changed = true
				}
				if changed {
					m.Source = ls
				}
			}
			lock.Mods[key] = m
			if err := service.SaveLock(mcVersion, loader, lock); err != nil {
				return err
			}
			fmt.Printf("updated lock mod %s -> locks/dependencies/%s-%s.json\n", key, mcVersion, loader)
			return nil
		},
	}
	cmd.Flags().StringVar(&fName, "name", "", "Display name")
	cmd.Flags().StringVar(&fVersion, "version", "", "Mod version")
	cmd.Flags().StringVar(&fScope, "scope", "", "Scope: shared, client, server")
	cmd.Flags().StringVar(&fSource, "source", "", "Source type (curseforge, github-release, git, local)")
	cmd.Flags().StringVar(&fModID, "mod-id", "", "CurseForge mod ID")
	cmd.Flags().StringVar(&fFileID, "file-id", "", "CurseForge file ID")
	cmd.Flags().StringVar(&fFileName, "file-name", "", "Download file name")
	cmd.Flags().StringVar(&fRepo, "repo", "", "Repository")
	cmd.Flags().StringVar(&fTag, "tag", "", "Git tag")
	cmd.Flags().StringVar(&fAssetName, "asset-name", "", "Release asset name")
	cmd.Flags().StringVar(&fPath, "path", "", "Local file path")

	return cmd
}

// runLockUpdateBulk re-resolves the full lock for the requested
// (mcVersion, loader) range. It is used when `mcmod lock update` is
// called without a key positional, mirroring spec 7.3.2.
func runLockUpdateBulk(args []string) error {
	spec, err := domain.ReadPackSpec(".")
	if err != nil {
		return fmt.Errorf("lock update: read packspec\nhint: create packspec.json in the project root")
	}
	mcVersion := ""
	loader := ""
	if len(args) > 0 {
		mcVersion = args[0]
	}
	if len(args) > 1 {
		loader = args[1]
	}
	if mcVersion == "" {
		mcVersion = spec.MinecraftVersion
	}
	loaders := []string{loader}
	if loader == "" {
		for _, ln := range spec.LoaderName {
			name, _ := domain.ParseLoaderName(ln)
			loaders = append(loaders, name)
		}
		if len(loaders) > 0 {
			loaders = loaders[1:]
		}
	}
	if len(loaders) == 0 {
		loaders = []string{"neoforge"}
	}
	failed := false
	for _, l := range loaders {
		if !domain.LoaderMatches(*spec, l) {
			fmt.Fprintf(os.Stderr, "error: lock update: unsupported loader %q\nhint: use neoforge or fabric\n", l)
			failed = true
			continue
		}
		lock, resolveFailures, err := service.BuildLockWithExisting(spec, mcVersion, l, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: lock update %s %s: %v\n", mcVersion, l, err)
			failed = true
			continue
		}
		if err := service.WriteLockFile(lock); err != nil {
			fmt.Fprintf(os.Stderr, "error: write lock %s %s: %v\n", mcVersion, l, err)
			failed = true
			continue
		}
		fmt.Printf("updated lock %s/%s (%d mods)\n", mcVersion, l, len(lock.Mods))
		if resolveFailures > 0 {
			// Partial lock: see docs/003-lock-files.md and the matching
			// block in `lock` (no subcommand). Surface the count and
			// fail so a downstream `mcmod build` does not silently
			// drop the missing mod.
			fmt.Fprintf(os.Stderr, "error: lock update %s %s: %d mod(s) failed to resolve; lock file is partial\nhint: see the per-mod failure list above and re-run `mcmod lock update %s %s` once they are fixed\n", mcVersion, l, resolveFailures, mcVersion, l)
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("lock update failed for one or more loaders")
	}
	return nil
}

func newLockDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [<minecraftVersion>] [<loader>] [<key>]",
		Short: "Delete lock entries or files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 3 {
				mcVersion, loader, key := args[0], args[1], args[2]
				lock, err := service.LoadLock(mcVersion, loader)
				if err != nil {
					return fmt.Errorf("lock delete: lock not found for %s %s", mcVersion, loader)
				}
				if _, ok := lock.Mods[key]; !ok {
					return fmt.Errorf("lock delete: key %q not found in %s %s", key, mcVersion, loader)
				}
				delete(lock.Mods, key)
				if err := service.SaveLock(mcVersion, loader, lock); err != nil {
					return err
				}
				fmt.Printf("deleted lock mod %s -> locks/dependencies/%s-%s.json\n", key, mcVersion, loader)
				return nil
			}
			// Per spec 7.3.8: without key, delete the lock file(s).
			mcVersion := "1.21.1"
			spec, specErr := domain.ReadPackSpec(".")
			if specErr == nil && spec.MinecraftVersion != "" {
				mcVersion = spec.MinecraftVersion
			}
			loader := ""
			if len(args) >= 1 {
				mcVersion = args[0]
			}
			if len(args) >= 2 {
				loader = args[1]
			}
			loaders := []string{loader}
			if loader == "" {
				if specErr == nil {
					for _, ln := range spec.LoaderName {
						name, _ := domain.ParseLoaderName(ln)
						loaders = append(loaders, name)
					}
					loaders = loaders[1:]
				} else {
					loaders = []string{"neoforge"}
				}
			}
			failed := false
			for _, ld := range loaders {
				path := domain.LockFilePath(mcVersion, ld)
				if _, err := os.Stat(path); err != nil {
					fmt.Fprintf(os.Stderr, "error: lock delete: lock file %s not found\n", path)
					failed = true
					continue
				}
				if err := os.Remove(path); err != nil {
					fmt.Fprintf(os.Stderr, "error: lock delete: %v\n", err)
					failed = true
					continue
				}
				fmt.Printf("deleted lock file %s\n", path)
			}
			if failed {
				return fmt.Errorf("lock delete failed for one or more files")
			}
			return nil
		},
	}
}

func newLockReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release set|list|show|delete",
		Short: "Manage build release index",
	}
	cmd.AddCommand(newReleaseSetCmd())
	cmd.AddCommand(newReleaseListCmd())
	cmd.AddCommand(newReleaseShowCmd())
	cmd.AddCommand(newReleaseDeleteCmd())

	return cmd
}
