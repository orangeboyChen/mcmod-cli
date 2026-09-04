// File: internal/cli/commands_validate.go
// Created: 2026-06-20
// Description: Implements `mcmod validate` subcommand for spec/lock/release index checks.

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

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
