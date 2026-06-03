package cmd

import (
	"fmt"

	"github.com/sharvari/claude-flipper/internal/switcher"
	"github.com/spf13/cobra"
)

// swapCmd rotates to the next account or jumps to a specific one.
var swapCmd = &cobra.Command{
	Use:   "swap [slot|email]",
	Short: "Swap to the next account, or to a specific one by slot or email",
	Long: `Swaps to the next account in the rotation sequence. When the last account
is reached it wraps back to the first.

Optionally pass a slot number or email to jump to a specific account:
  flipper swap
  flipper swap 2
  flipper swap work@company.com`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) == 1 {
			target = args[0]
		}
		slot, email, err := switcher.SwitchTo(target)
		if err != nil {
			return err
		}
		fmt.Printf("Swapped to slot %d: %s\n", slot, email)
		return nil
	},
}
