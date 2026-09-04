// File: internal/cli/commands_version.go
// Created: 2026-06-20
// Description: Implements `mcmod version` subcommand.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print the mcmod CLI release version.

Example:
  mcmod version`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printVersion(cmd)
		},
	}

	return cmd
}

func printVersion(cmd *cobra.Command) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "mcmod version "+domain.Version)
	return err
}
