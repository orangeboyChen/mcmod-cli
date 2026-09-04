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
			// Use cmd.OutOrStdout() so the version line lands on stdout,
			// matching spec 7.3. cobra's cmd.Println() falls back to
			// stderr when the outWriter is not set, which would make
			// piped consumers (and our smoke tests) miss the output.
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "mcmod version "+domain.Version)
			return err
		},
	}

	return cmd
}
