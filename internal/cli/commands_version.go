// File: internal/cli/commands_version.go
// Created: 2026-06-20
// Description: Implements `mcmod version` subcommand.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
