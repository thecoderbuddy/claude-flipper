package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/accounts"
	"github.com/sharvari/claude-flipper/internal/credentials"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <slot|email>",
	Short: "Remove an account from the managed set",
	Long: `Removes the account identified by slot number or email address.
The account must not be the currently active one — switch away first.

This removes the entry from sequence.json and deletes the credential backup.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		seq, err := accounts.Load()
		if err != nil {
			return err
		}

		slot, rec, err := accounts.FindAccount(seq, args[0])
		if err != nil {
			return err
		}

		removedSlot, err := accounts.RemoveAccount(seq, args[0])
		if err != nil {
			return err
		}

		// Delete credential and config backups (best-effort).
		store := credentials.New()
		store.DeleteBackup(removedSlot, rec.Email)

		if err := accounts.Save(seq); err != nil {
			return fmt.Errorf("save sequence: %w", err)
		}

		fmt.Printf("Removed slot %d: %s\n", slot, rec.Email)
		return nil
	},
}
