// Package cmd wires up all flipper sub-commands via cobra.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "flipper",
	Short: "Swap between multiple Claude Code accounts",
	Long: `flipper manages multiple Claude Code accounts and lets you swap between them
without logging out and back in. Accounts are stored locally in ~/.claude-flipper/.`,
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(swapCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(resetCmd)
}
