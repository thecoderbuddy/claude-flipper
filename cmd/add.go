package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/switcher"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Save the currently logged-in Claude Code account as a new slot",
	Long: `Reads the active Claude Code account from ~/.claude/.config.json and the
platform credential store, then saves it as a new managed account slot.

Run this once per account while that account is logged in to Claude Code.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := switcher.AddCurrent()
		if err != nil {
			return err
		}
		if result.IsNew {
			fmt.Printf("Saved account %q as slot %d.\n", result.Email, result.Slot)
		} else {
			fmt.Printf("Refreshed slot %d for %q.\n", result.Slot, result.Email)
		}
		fmt.Println("Run 'flipper list' to see all managed accounts.")
		return nil
	},
}
