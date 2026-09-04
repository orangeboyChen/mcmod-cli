// File: internal/cli/commands_release.go
// Created: 2026-06-20
// Description: Implements `mcmod lock release` subcommands (set/list/show/delete).

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
)

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
