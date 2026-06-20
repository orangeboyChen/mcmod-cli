// File: internal/cli/commands_build.go
// Created: 2026-06-20
// Description: Implements `mcmod build` subcommand for zip artifact production.

package cli

import (
	"fmt"
	"os"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
	"github.com/spf13/cobra"
)

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
			// --build-type accepts "all" (default) and "cf". The output of
			// "cf" is identical to the default for now; a future change will
			// emit a CurseForge modpack layout here. Both values are accepted
			// today so the flag is exercisable from the CLI and from tests.
			if buildType != "" && buildType != "all" && buildType != "cf" {
				fmt.Fprintf(os.Stderr, "error: build: invalid --build-type %q (must be \"all\" or \"cf\")\n", buildType)
				return fmt.Errorf("build: invalid --build-type %q", buildType)
			}
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
