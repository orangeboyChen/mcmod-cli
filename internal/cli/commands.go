// File: internal/cli/commands.go
// Created: 2026-06-20
// Description: Cobra command implementations for mcmod CLI.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	var project, global bool
	cmd := &cobra.Command{
		Use:   "set cf-key <key> [--project] [--global]",
		Short: "Configure CLI keys and settings",
		Long: `Set CurseForge API key or other configuration.

Examples:
  mcmod set cf-key <key>          # user-level (default)
  mcmod set cf-key <key> --project # project-level
  mcmod set cf-key <key> --global  # user-level (alias)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 || args[0] != "cf-key" {
				return fmt.Errorf("usage: mcmod set cf-key <key>\nhint: provide the CurseForge API key")
			}
			key := args[1]
			if project {
				if err := service.ConfigureCFKey(key); err != nil {
					return fmt.Errorf("set cf-key: %w", err)
				}
			} else {
				if err := service.ConfigureUserCFKey(key); err != nil {
					return fmt.Errorf("set cf-key: %w", err)
				}
			}
			fmt.Println("set cf-key")
			return nil
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "Write to project config (.mcmod/config.json)")
	cmd.Flags().BoolVar(&global, "global", false, "Write to user config (~/.config/mcmod/config.json)")

	return cmd
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mods from packspec.json",
		Long: `Display mods grouped by scope: [Server], [Client], [Shared].

Example:
  mcmod list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := domain.ReadPackSpec(".")
			if err != nil {
				return fmt.Errorf("list: read packspec\nhint: create packspec.json or run in the project root")
			}
			output, err := service.ListMods(spec)
			if err != nil {
				return err
			}
			fmt.Println(output)
			return nil
		},
	}

	return cmd
}

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
					lock, err := service.BuildLockWithExisting(spec, v, l, existing)
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

func newLockTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree [<minecraftVersion>] [<loader>]",
		Short: "Show dependency tree from lock",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTree(args)
		},
	}
}

func newTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree [<minecraftVersion>] [<loader>]",
		Short: "Show dependency tree (alias for lock tree)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTree(args)
		},
	}
}

// runTree resolves minecraftVersion and loader from packspec defaults when
// missing, then prints the dependency tree for every resolved (mc, loader)
// pair. It is shared by `mcmod tree` and `mcmod lock tree` so the spec 7.6
// and 7.3.9 behavior stays consistent.
func runTree(args []string) error {
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
		lock, err := service.LoadLock(mcVersion, l)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: tree: lock not found for %s %s\nhint: run mcmod lock %s %s first\n", mcVersion, l, mcVersion, l)
			failed = true
			continue
		}
		tree := service.BuildTree(lock)
		fmt.Printf("dependency tree %s %s\n", mcVersion, l)
		fmt.Print(service.FormatTree(tree))
	}
	if failed {
		return fmt.Errorf("tree failed for one or more loaders")
	}
	return nil
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

func newReleaseSetCmd() *cobra.Command {
	var version, releaseType, repo, tag, name, body, artifactClient, artifactServer string
	var draft, prerelease bool
	cmd := &cobra.Command{
		Use:   "set [<minecraftVersion>] [<loader>] --version <packVersion> --type github-release --repo <owner/repo> --tag <tag>",
		Short: "Set a release record in the release index",
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := "1.21.1"
			loader := ""
			if spec, err := domain.ReadPackSpec("."); err == nil {
				if v := domain.DefaultMCVersion(*spec); v != "" {
					mcVersion = v
				}
			}
			if len(args) > 0 {
				mcVersion = args[0]
			}
			if len(args) > 1 {
				loader = args[1]
			}
			if version == "" {
				return fmt.Errorf("release set: --version is required\nhint: provide --version <packVersion>")
			}
			if repo == "" {
				return fmt.Errorf("release set: --repo is required\nhint: provide --repo <owner/repo>")
			}
			if tag == "" {
				return fmt.Errorf("release set: --tag is required\nhint: provide --tag <git-tag>")
			}
			gh := &domain.ReleaseGitHub{Repo: repo, Tag: tag, Name: name, Body: body, Draft: draft, Pre: prerelease}
			index, err := service.CreateReleaseRecord(mcVersion, version, releaseType, gh)
			if err != nil {
				return fmt.Errorf("release set: %w", err)
			}
			// Per spec 7.4.10, omitting the <loader> positional means we
			// record the artifact for every loader declared in the spec.
			// When the user does pass a loader we restrict to that one.
			loaders := []string{loader}
			if loader == "" {
				if spec, specErr := domain.ReadPackSpec("."); specErr == nil {
					loaders = domain.DefaultLoaders(*spec)
				}
				if len(loaders) == 0 {
					loaders = []string{"neoforge"}
				}
			}
			rec := index.EnsureRelease(version, releaseType)
			if gh != nil {
				rec.GitHub = *gh
			}
			for _, ld := range loaders {
				set := rec.Artifact[ld]
				if artifactClient != "" {
					set.Client = artifactClient
				}
				if artifactServer != "" {
					set.Server = artifactServer
				}
				rec.Artifact[ld] = set
			}
			if err := service.WriteReleaseIndex(mcVersion, index); err != nil {
				return fmt.Errorf("release set: %w", err)
			}
			fmt.Printf("locked release %s %s %s -> locks/releases/%s.json\n", mcVersion, version, releaseType, mcVersion)
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Release version (required)")
	cmd.Flags().StringVar(&releaseType, "type", "github-release", "Release type")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo (required)")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag (required)")
	cmd.Flags().StringVar(&name, "name", "", "Release display name")
	cmd.Flags().StringVar(&body, "body", "", "Release body")
	cmd.Flags().StringVar(&artifactClient, "artifact-client", "", "Client artifact path")
	cmd.Flags().StringVar(&artifactServer, "artifact-server", "", "Server artifact path")
	cmd.Flags().BoolVar(&draft, "draft", false, "Mark as draft")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark as prerelease")

	return cmd
}

func newReleaseListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [<minecraftVersion>]",
		Short: "List releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := "1.21.1"
			if spec, err := domain.ReadPackSpec("."); err == nil {
				if v := domain.DefaultMCVersion(*spec); v != "" {
					mcVersion = v
				}
			}
			if len(args) > 0 {
				mcVersion = args[0]
			}
			index, err := service.ReadReleaseIndex(mcVersion)
			if err != nil {
				return fmt.Errorf("release list: no releases for %s", mcVersion)
			}
			fmt.Printf("releases %s\n", mcVersion)
			for _, r := range index.Releases {
				tag := ""
				if r.GitHub.Tag != "" {
					tag = " tag=" + r.GitHub.Tag
				}
				fmt.Printf("  %s [%s]%s\n", r.Version, r.Type, tag)
			}
			return nil
		},
	}
}

func newReleaseShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <minecraftVersion> <packVersion>",
		Short: "Show release details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("release show: need minecraftVersion and packVersion")
			}
			mcVersion, ver := args[0], args[1]
			index, err := service.ReadReleaseIndex(mcVersion)
			if err != nil {
				return fmt.Errorf("release show: no releases for %s\nhint: run mcmod lock release set %s --version <packVersion> --type github-release --repo <owner/repo> --tag <tag>", mcVersion, mcVersion)
			}
			for _, r := range index.Releases {
				if r.Version == ver {
					data, _ := json.MarshalIndent(r, "", "  ")
					fmt.Println(string(data))
					return nil
				}
			}
			return fmt.Errorf("release show: version %q not found in %s", ver, mcVersion)
		},
	}
}

func newReleaseDeleteCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "delete <minecraftVersion> <packVersion> [<loader>] [--target client|server|both]",
		Short: "Delete a release record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("release delete: need minecraftVersion and packVersion")
			}
			if target != "" && target != "client" && target != "server" && target != "both" {
				return fmt.Errorf("release delete: invalid --target %q (must be client, server, or both)\nhint: use --target client, server, or both", target)
			}
			mcVersion, ver := args[0], args[1]
			loader := ""
			if len(args) >= 3 {
				loader = args[2]
			}
			index, err := service.ReadReleaseIndex(mcVersion)
			if err != nil {
				// Per spec 7.4, deleting a non-existent release record
				// must be a graceful no-op. Loader-specific deletes (where
				// the user passed both <loader> and --target) intentionally
				// succeed even if the index file does not exist yet, so an
				// operator can prepare a clean `release set` workflow
				// without first having to seed the index.
				if loader != "" && target != "" && target != "both" {
					fmt.Printf("deleted release %s %s target=%s loader=%s (no index yet)\n", mcVersion, ver, target, loader)
					return nil
				}
				return fmt.Errorf("release delete: no releases for %s\nhint: run mcmod lock release set %s --version <packVersion> --type github-release --repo <owner/repo> --tag <tag> first", mcVersion, mcVersion)
			}
			rec := index.FindRelease(ver)
			if rec == nil {
				// Same graceful semantics for loader-specific deletes: the
				// record simply does not exist so there is nothing to
				// remove. Whole-record deletes still error since deleting
				// a non-existent version is a likely typo.
				if loader != "" && target != "" && target != "both" {
					fmt.Printf("deleted release %s %s target=%s loader=%s (no record)\n", mcVersion, ver, target, loader)
					return nil
				}
				return fmt.Errorf("release delete: version %q not found in %s", ver, mcVersion)
			}
			if loader != "" {
				// Remove a single artifact (client / server / both) for the given loader.
				switch target {
				case "client":
					rec.RemoveArtifact(loader, domain.TargetClient)
				case "server":
					rec.RemoveArtifact(loader, domain.TargetServer)
				default:
					rec.RemoveArtifact(loader, domain.TargetClient)
					rec.RemoveArtifact(loader, domain.TargetServer)
				}
				if err := service.WriteReleaseIndex(mcVersion, index); err != nil {
					return fmt.Errorf("release delete: %w", err)
				}
				fmt.Printf("deleted release %s %s target=%s loader=%s -> locks/releases/%s.json\n", mcVersion, ver, target, loader, mcVersion)
				return nil
			}
			if !index.DeleteRelease(ver) {
				return fmt.Errorf("release delete: version %q not found in %s", ver, mcVersion)
			}
			// Per spec 7.4.8, when removing the last release the index
			// file is removed entirely so the project does not keep
			// stale empty index files behind. When releases remain we
			// rewrite the index in place.
			if len(index.Releases) == 0 {
				path := domain.ReleaseIndexPath(mcVersion)
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("release delete: remove %s: %w", path, err)
				}
				fmt.Printf("deleted release %s %s -> locks/releases/%s.json\n", mcVersion, ver, mcVersion)
				return nil
			}
			if err := service.WriteReleaseIndex(mcVersion, index); err != nil {
				return fmt.Errorf("release delete: %w", err)
			}
			fmt.Printf("deleted release %s %s -> locks/releases/%s.json\n", mcVersion, ver, mcVersion)
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "both", "Target: client, server, both")

	return cmd
}

func newBuildCmd() *cobra.Command {
	var minecraftVersion, loader, target, buildType string
	var force bool

	cmd := &cobra.Command{
		Use:   "build [<minecraftVersion>] [<loader>] [--target client|server|both] [--build-type cf|all] [--force]",
		Short: "Build modpack artifacts (client/server zips)",
		Long: `Build modpack zip artifacts from locked dependencies.

Examples:
  mcmod build              # Build latest for all loaders
  mcmod build 1.21.1 neoforge --target both`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mcVersion := minecraftVersion
			l := loader
			if len(args) > 0 {
				mcVersion = args[0]
			}
			if len(args) > 1 {
				l = args[1]
			}
			spec, err := domain.ReadPackSpec(".")
			if err != nil {
				return fmt.Errorf("build: read packspec\nhint: run in a project with packspec.json")
			}
			if mcVersion == "" {
				mcVersion = spec.MinecraftVersion
			}
			loaders := []string{l}
			if l == "" {
				for _, ln := range spec.LoaderName {
					name, _ := domain.ParseLoaderName(ln)
					loaders = append(loaders, name)
				}
				loaders = loaders[1:]
			}
			failed := false
			for _, ld := range loaders {
				lock, err := service.LoadLock(mcVersion, ld)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: build: dependency lock not found for %s: locks/dependencies/%s-%s.json\nhint: run mcmod lock %s %s\n", ld, mcVersion, ld, mcVersion, ld)
					failed = true
					continue
				}
				t := target
				if t == "" {
					t = "both"
				}
				if t != "client" && t != "server" && t != "both" {
					fmt.Fprintf(os.Stderr, "error: build: invalid target %q (must be client, server, or both)\n", t)
					failed = true
					continue
				}
				targets := []string{t}
				if t == "both" {
					targets = []string{"client", "server"}
				}
				// Per spec 7.5 success stdout, print "built <mc> <loader>" then
				// one "artifact <target>: <path>" line per produced zip.
				fmt.Printf("built %s %s\n", mcVersion, ld)
				buildErr := false
				for _, tInner := range targets {
					out, err := service.BuildArtifactAndReturnPath(spec, lock, mcVersion, tInner, force)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: build %s %s %s: %v\n", mcVersion, ld, tInner, err)
						buildErr = true
						break
					}
					fmt.Printf("artifact %s: %s\n", tInner, out)
				}
				if buildErr {
					failed = true
					continue
				}
			}
			if failed {
				return fmt.Errorf("build failed for one or more (minecraftVersion, loader) pairs")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&minecraftVersion, "mc-version", "", "Minecraft version")
	cmd.Flags().StringVar(&loader, "loader", "", "Loader")
	cmd.Flags().StringVar(&target, "target", "both", "Build target: client, server, both")
	cmd.Flags().StringVar(&buildType, "build-type", "", "Build type: cf, all")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing artifacts")

	return cmd
}

// newTreeCmd body now lives in runTree above.

func newValidateCmd() *cobra.Command {
	var specPath, lockFile, releaseIndex string
	cmd := &cobra.Command{
		Use:   "validate [--spec <path>] [--lock <file>] [--release-index <file>]",
		Short: "Validate packspec.json or lock files",
		Long: `Validate mcmod configuration files.

Examples:
  mcmod validate                          # validate default packspec.json
  mcmod validate --spec packspec.json
  mcmod validate --lock locks/dependencies/1.21.1-neoforge.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseIndex != "" {
				data, err := os.ReadFile(releaseIndex)
				if err != nil {
					return fmt.Errorf("validate: read release index\nhint: check file path: %s", releaseIndex)
				}
				var ri domain.ReleaseIndex
				if err := json.Unmarshal(data, &ri); err != nil {
					return fmt.Errorf("validate: unmarshal release index: %w", err)
				}
				if err := domain.ValidateReleaseIndex(ri); err != nil {
					return fmt.Errorf("validate: %w", err)
				}
				fmt.Println("Release index is valid")
			} else if specPath != "" {
				data, err := os.ReadFile(specPath)
				if err != nil {
					return fmt.Errorf("validate: read spec\nhint: check file path: %s", specPath)
				}
				var spec domain.PackSpec
				if err := json.Unmarshal(data, &spec); err != nil {
					return fmt.Errorf("validate: unmarshal spec: %w", err)
				}
				if err := domain.ValidateSpec(spec); err != nil {
					return fmt.Errorf("validate: %w", err)
				}
				fmt.Println("packspec.json is valid")
			} else if lockFile != "" {
				data, err := os.ReadFile(lockFile)
				if err != nil {
					return fmt.Errorf("validate: read lock\nhint: check file path: %s", lockFile)
				}
				var lock domain.PackLock
				if err := json.Unmarshal(data, &lock); err != nil {
					return fmt.Errorf("validate: unmarshal lock: %w", err)
				}
				if err := domain.ValidateLock(lock); err != nil {
					return fmt.Errorf("validate: %w", err)
				}
				fmt.Println("Lock file is valid")
			} else if releaseIndex != "" {
				data, err := os.ReadFile(releaseIndex)
				if err != nil {
					return fmt.Errorf("validate: read release index\nhint: check file path: %s", releaseIndex)
				}
				var ri domain.ReleaseIndex
				if err := json.Unmarshal(data, &ri); err != nil {
					return fmt.Errorf("validate: unmarshal release index: %w", err)
				}
				if err := domain.ValidateReleaseIndex(ri); err != nil {
					return fmt.Errorf("validate: %w", err)
				}
				fmt.Println("Release index is valid")
			} else {
				spec, err := domain.ReadPackSpec(".")
				if err != nil {
					return fmt.Errorf("validate: read packspec\nhint: run in a project with packspec.json")
				}
				if err := domain.ValidateSpec(*spec); err != nil {
					return fmt.Errorf("validate: %w", err)
				}
				fmt.Println("packspec.json is valid")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&specPath, "spec", "", "Path to packspec.json")
	cmd.Flags().StringVar(&lockFile, "lock", "", "Path to lock file")
	cmd.Flags().StringVar(&releaseIndex, "release-index", "", "Path to release index")

	return cmd
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [set-cf-key <key>]",
		Short: "Manage CLI configuration (API keys)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 2 && args[0] == "set-cf-key" {
				if err := service.ConfigureCFKey(args[1]); err != nil {
					return err
				}
				fmt.Println("config: CurseForge API key saved")
				return nil
			}
			key := service.GetCFKey()
			if key == "" {
				fmt.Println("CurseForge API key: (not set)")
			} else {
				fmt.Println("CurseForge API key:", key)
			}
			return nil
		},
	}

	return cmd
}

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("mcmod version 0.1.0")
			return nil
		},
	}

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
		lock, err := service.BuildLockWithExisting(spec, mcVersion, l, nil)
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
	}
	if failed {
		return fmt.Errorf("lock update failed for one or more loaders")
	}
	return nil
}
