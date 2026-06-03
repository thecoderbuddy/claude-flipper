package cmd

import (
	"fmt"
	"os"

	"github.com/sharvari/claude-flipper/internal/paths"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Remove all saved accounts and wipe claude-flipper data",
	Long:  `Deletes all saved accounts, credentials, and config backups. Claude Code itself is not affected.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("This will remove all saved accounts. Are you sure? [y/N] ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Aborted.")
			return nil
		}

		if err := os.RemoveAll(paths.DataDir()); err != nil {
			return fmt.Errorf("reset failed: %w", err)
		}

		fmt.Println("All accounts removed.")
		return nil
	},
}
