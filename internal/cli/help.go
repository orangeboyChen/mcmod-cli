// File: internal/cli/help.go
// Created: 2026-06-20
// Description: Help text and usage templates for mcmod CLI per spec section 7.7.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// fullHelpTemplate is the template used for both `mcmod --help` and
// `mcmod <sub> --help`. It renders the command's Long description, the
// positional Usage block mandated by spec 7.7, the command's local flags
// (if any) and the global help flag. We install this template via
// SetHelpTemplate on every command so the default --help implementation
// in cobra uses it instead of its built-in "Long + UsageString" path
// (which would otherwise print the Long block twice).
const fullHelpTemplate = `{{.Long | trimTrailingWhitespaces}}

Usage:
  mcmod set cf-key <key>
  mcmod list
  mcmod lock [<minecraftVersion>] [<loader>]
  mcmod lock list [<minecraftVersion>] [<loader>]
  mcmod lock show [<minecraftVersion>] [<loader>]
  mcmod lock show <minecraftVersion> <loader> <key>
  mcmod lock update [<minecraftVersion>] [<loader>]
  mcmod lock update <minecraftVersion> <loader> <key> [options]
  mcmod lock delete [<minecraftVersion>] [<loader>]
  mcmod lock delete <minecraftVersion> <loader> <key>
  mcmod lock add <minecraftVersion> <loader> <key> [options]
  mcmod lock release set [<minecraftVersion>] [<loader>] --version <packVersion> --type github-release --repo <owner/repo> --tag <tag> [options]
  mcmod lock release list [<minecraftVersion>]
  mcmod lock release show <minecraftVersion> <packVersion>
  mcmod lock release delete <minecraftVersion> <packVersion> [<loader>] [--target client|server|both]
  mcmod build [<minecraftVersion>] [<loader>] [options]
  mcmod tree [<minecraftVersion>] [<loader>]
{{- if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Global options:
  -h, --help                Show help
  -v, --version             Print version information
`

// rootLong is the long description for the root command. It is rendered by
// fullHelpTemplate into the first paragraph of `mcmod --help` output.
const rootLong = `mcmod is a CLI for managing Minecraft modpack specifications,
dependency locks, jar resolution, downloading, build artifacts, and release indexes.

Top-level commands:
  set        Configure the CurseForge API key (mcmod set cf-key <key>).
  list       Show mods from packspec.json grouped by scope.
  lock       Resolve and lock mod dependencies, manage lock entries.
  build      Build client/server modpack zips from the dependency lock.
  tree       Show the resolved dependency tree and version decisions.
  validate   Validate packspec.json, dependency lock, or release index files.
  version    Print mcmod version information.
  config     Compatibility alias for set (mcmod config set-cf-key <key>).
  help       Show help for any command.

Error format:
  error: <command>: <reason>
  hint:   <actionable fix>

See "mcmod <command> --help" for command-specific help.`

// fallbackLong returns a Long description derived from the command's
// own Short field. Used by applyHelpTemplate to populate Long for
// subcommands that did not declare one, so the {{.Long}} placeholder
// in fullHelpTemplate resolves to the Short text instead of an empty
// paragraph.
func fallbackLong(c *cobra.Command) string {
	return fmt.Sprintf("%s", c.Short)
}

func init() {
	cobra.AddTemplateFunc("HelpLong", func() string { return rootLong })
}

func usageTemplate() string {
	return fullHelpTemplate
}
