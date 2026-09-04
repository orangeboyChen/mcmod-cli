// File: internal/cli/app.go
// Created: 2026-06-20
// Description: Cobra CLI app entry and root command definition.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orangeboyChen/mcmod-cli/internal/config"
	"github.com/orangeboyChen/mcmod-cli/internal/domain"
)

// NewApp creates a new mcmod CLI application.
func NewApp() *cobra.Command {
	root := &cobra.Command{
		Use:                        "mcmod",
		Short:                      "Minecraft modpack management tool",
		Long:                       rootLong,
		Version:                    domain.Version,
		SilenceErrors:              true,
		SilenceUsage:               true,
		DisableAutoGenTag:          true,
		SuggestionsMinimumDistance: 2,
	}
	root.SetVersionTemplate("mcmod version {{.Version}}\n")
	// Register a single persistent `-h, --help` flag on the root command
	// so every subcommand inherits it via mergePersistentFlags. This
	// prevents cobra from lazily adding a per-subcommand "help for X"
	// local flag on first Execute(), which would otherwise surface as a
	// duplicate entry in our rendered "Flags:" block. The flag is
	// hidden (cobra's "--help" still works, the parser just doesn't
	// advertise it twice). The spec 7.7 help template also prints
	// "Global options: -h, --help" so the user still sees it once.
	root.PersistentFlags().BoolP("help", "h", false, "Show help for any command")
	_ = root.PersistentFlags().MarkHidden("help")
	// Install the spec 7.7 help template on every command. Cobra's
	// default --help implementation prints Long first, then the rendered
	// UsageTemplate; by setting both the HelpTemplate (used when the
	// user passes --help) and UsageTemplate (used by the help subcommand
	// and the silence-broken usage paths) to the same string we make
	// sure the Long block never appears twice. See help.go for the
	// template body.
	applyHelpTemplate(root)
	// Replace cobra's auto-generated help subcommand. The default prints
	// "Unknown help topic" to stdout which conflicts with spec 7.8's
	// "errors go to stderr" requirement. Our custom help command
	// routes unknown topics to stderr while still forwarding valid
	// subcommand names to the corresponding command's help.
	root.SetHelpCommand(newHelpCmd())

	root.AddCommand(
		newSetCmd(),
		newLockCmd(),
		newBuildCmd(),
		newListCmd(),
		newValidateCmd(),
		newTreeCmd(),
		newVersionCmd(),
		// `config` is a compatibility alias for `set` so existing
		// `mcmod config set-cf-key <key>` invocations keep working. Spec
		// 7.7 only requires `mcmod set cf-key`; `config` is intentionally
		// not listed in the root usage template but is registered so the
		// smoke / integration tests can hit it.
		newConfigCmd(),
	)
	// Re-apply the help template after subcommands are attached so the
	// grandchildren (e.g. `mcmod lock release set`, `mcmod lock add`)
	// also pick up the same full help layout.
	applyHelpTemplate(root)

	return root
}

// applyHelpTemplate walks every command in the tree and installs the
// spec 7.7 help template on it. It also ensures the Long field is
// non-empty for subcommands that did not set one so the {{.Long}}
// placeholder in the template is always populated. The template handles
// both root and subcommand cases identically.
func applyHelpTemplate(root *cobra.Command) {
	if root.Long == "" {
		root.Long = fallbackLong(root)
	}
	root.SetHelpTemplate(fullHelpTemplate)
	root.SetUsageTemplate(fullHelpTemplate)
	for _, c := range root.Commands() {
		applyHelpTemplate(c)
	}
}

// newHelpCmd creates the custom help subcommand that replaces cobra's
// default. The default prints "Unknown help topic" to stdout; this version
// routes the same diagnostic to stderr per spec 7.8.
func newHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type mcmod help [path to command] for full details.`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			if len(args) == 0 {
				// Render the root usage template to stdout. The default
				// writer for UsageFunc is stderr (cobra's OutOrStderr)
				// which is correct for --help and error paths, but
				// `mcmod help` is a user-invoked informational command
				// whose content should land on stdout so that piping
				// (e.g. `mcmod help | grep lock`) works.
				origOut := root.OutOrStderr()
				root.SetOut(os.Stdout)
				root.SetErr(os.Stdout)
				defer func() {
					root.SetOut(origOut)
					root.SetErr(origOut)
				}()
				root.UsageFunc()(root)
				return nil
			}
			topic := args[0]
			target, _, err := root.Find([]string{topic})
			if err != nil || target == root {
				// Unknown topic: print the banner and the full command
				// tree to stderr so the operator sees them as a
				// diagnostic per spec 7.8.
				fmt.Fprintf(cmd.ErrOrStderr(), "Unknown help topic %q\nRun \"mcmod help\" for available commands.\n\n", topic)
				renderUsageToStderr(root)
				return nil
			}
			// Forward the help request to the matching subcommand. Pass
			// remaining args so `mcmod help lock release set` can drill
			// down into the grand-subcommand.
			target.HelpFunc()(target, args[1:])
			return nil
		},
	}
}

// renderUsageToStderr renders cmd's usage template to stderr by swapping
// the command's output writer to stderr for the duration of the call.
// The original writer is restored before returning so subsequent usage
// output goes back to the default destination.
func renderUsageToStderr(cmd *cobra.Command) {
	origOut := cmd.OutOrStderr()
	cmd.SetOut(os.Stderr)
	cmd.SetErr(os.Stderr)
	defer func() {
		cmd.SetOut(origOut)
		cmd.SetErr(origOut)
	}()
	cmd.UsageFunc()(cmd)
}

// Run executes the CLI with given args.
func Run() {
	if err := loadDotEnvFromRepoRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load .env: %v\n", err)
	}
	cmd := NewApp()
	// Spec 7.3 requires success results to go to stdout and errors to
	// stderr. Cobra's defaults already map --help / errors to stderr,
	// but we set Err explicitly so user-visible runtime errors that
	// are not produced through cobra's own machinery (e.g. helper
	// functions that call fmt.Fprintln directly) also land on stderr.
	cmd.SetErr(os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// loadDotEnvFromRepoRoot walks up from the current working directory looking
// for go.mod, then loads .env from that repo root if present. Falls back to
// the working directory when no go.mod is found above us.
func loadDotEnvFromRepoRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			root = dir
			break
		}
		root = parent
	}
	return config.LoadDotEnv(filepath.Join(root, ".env"))
}
