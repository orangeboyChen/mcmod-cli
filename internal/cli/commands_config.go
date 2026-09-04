// File: internal/cli/commands_config.go
// Created: 2026-06-20
// Description: Implements `mcmod config` subcommand for reading/writing API keys.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/service"
)

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
