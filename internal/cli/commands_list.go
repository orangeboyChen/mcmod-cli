// File: internal/cli/commands_list.go
// Created: 2026-06-20
// Description: Implements `mcmod list` subcommand for displaying packspec mods.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
	"github.com/orangeboyChen/mcmod-cli/internal/service"
)

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
