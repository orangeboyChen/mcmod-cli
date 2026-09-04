// File: internal/cli/commands_set.go
// Created: 2026-06-20
// Description: Implements `mcmod set` subcommand for storing API keys.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/service"
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
