package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/accounts"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		seq, err := accounts.Load()
		if err != nil {
			return err
		}

		if len(seq.Accounts) == 0 {
			fmt.Println("No accounts managed yet. Run 'flipper add' while logged in to an account.")
			return nil
		}

		fmt.Printf("%-6s  %-3s  %-40s  %s\n", "SLOT", "ACT", "EMAIL", "ORG")
		fmt.Printf("%-6s  %-3s  %-40s  %s\n", "----", "---", "-----", "---")

		for _, slot := range accounts.SortedSlots(seq) {
			key := fmt.Sprintf("%d", slot)
			rec := seq.Accounts[key]

			active := " "
			if slot == seq.ActiveSlot {
				active = "*"
			}

			org := rec.OrganizationName
			if org == "" {
				org = rec.OrganizationUUID
			}
			if org == "" {
				org = "(personal)"
			}

			fmt.Printf("%-6d  %-3s  %-40s  %s\n", slot, active, rec.Email, org)
		}
		return nil
	},
}
