package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const wrapperMarker = "# claude-flipper"

const zshBashWrapper = `
# claude-flipper: inject active account token
claude() { ANTHROPIC_AUTH_TOKEN="$(flipper token)" command claude "$@"; }
`

const fishWrapper = `
# claude-flipper: inject active account token
function claude
    set -x ANTHROPIC_AUTH_TOKEN (flipper token)
    command claude $argv
end
`

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Add the claude shell wrapper to your shell config",
	Long: `Adds a shell function to your rc file (~/.zshrc, ~/.bashrc, or fish config)
that injects ANTHROPIC_AUTH_TOKEN when you run claude, so account switching works.

Run once after installing claude-flipper.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := filepath.Base(os.Getenv("SHELL"))
		rcFile, wrapper, err := resolveShell(shell)
		if err != nil {
			return err
		}

		// Check if already installed.
		existing, _ := os.ReadFile(rcFile)
		if strings.Contains(string(existing), wrapperMarker) {
			fmt.Printf("Already set up in %s\n", rcFile)
			return nil
		}

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open %s: %w", rcFile, err)
		}
		defer f.Close()

		if _, err := f.WriteString(wrapper); err != nil {
			return fmt.Errorf("write to %s: %w", rcFile, err)
		}

		fmt.Printf("Added claude wrapper to %s\n", rcFile)
		fmt.Printf("Run: source %s\n", rcFile)
		return nil
	},
}

func resolveShell(shell string) (rcFile, wrapper string, err error) {
	home, _ := os.UserHomeDir()
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc"), zshBashWrapper, nil
	case "bash":
		rc := filepath.Join(home, ".bashrc")
		if _, e := os.Stat(rc); os.IsNotExist(e) {
			rc = filepath.Join(home, ".bash_profile")
		}
		return rc, zshBashWrapper, nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), fishWrapper, nil
	default:
		return "", "", fmt.Errorf("unsupported shell %q — add this to your rc file manually:\n%s", shell, zshBashWrapper)
	}
}
