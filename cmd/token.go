package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/switcher"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the active account's access token (refreshing if needed)",
	Long: `Prints the raw OAuth access token for the currently active account.
Refreshes the token if it is about to expire.

Intended for use in a shell wrapper:
  claude() { ANTHROPIC_AUTH_TOKEN="$(flipper token)" command claude "$@"; }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := switcher.GetActiveToken()
		if err != nil {
			return err
		}
		fmt.Print(token)
		return nil
	},
}
