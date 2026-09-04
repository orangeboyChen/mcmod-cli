// File: internal/cli/commands_tree.go
// Created: 2026-06-20
// Description: Implements `mcmod tree` and `mcmod lock tree` subcommands.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
)

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
