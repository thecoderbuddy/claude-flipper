package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/accounts"
	"github.com/sharvari/claude-flipper/internal/switcher"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the currently active Claude Code account",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show what cs thinks is active.
		seq, err := accounts.Load()
		if err != nil {
			return err
		}

		slot, rec, ok := accounts.ActiveAccount(seq)
		if !ok {
			fmt.Println("No active account tracked by cs.")
			fmt.Println("Run 'cs add' while logged in to register your current account.")
		} else {
			org := rec.OrganizationName
			if org == "" {
				org = rec.OrganizationUUID
			}
			if org == "" {
				org = "(personal)"
			}
			fmt.Printf("Active slot : %d\n", slot)
			fmt.Printf("Email       : %s\n", rec.Email)
			fmt.Printf("Org         : %s\n", org)
			fmt.Printf("UUID        : %s\n", rec.UUID)
			fmt.Printf("Added       : %s\n", rec.AddedAt.Format("2006-01-02 15:04:05 UTC"))
		}

		// Also show what Claude Code itself reports as live (best-effort).
		fmt.Println()
		liveAcct, err := switcher.ReadLiveAccount()
		if err != nil {
			fmt.Printf("Live Claude config: (could not read: %v)\n", err)
		} else {
			match := ""
			if ok && rec.Email == liveAcct.EmailAddress {
				match = " (matches cs)"
			} else if ok {
				match = " (!!! MISMATCH with cs)"
			}
			fmt.Printf("Live Claude config : %s%s\n", liveAcct.EmailAddress, match)
		}

		return nil
	},
}
