package cmd

import (
	"fmt"
	"time"

	"github.com/sharvari/claude-flipper/internal/accounts"
	"github.com/sharvari/claude-flipper/internal/credentials"
	"github.com/sharvari/claude-flipper/internal/switcher"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose credential state for all slots",
	RunE: func(cmd *cobra.Command, args []string) error {
		seq, err := accounts.Load()
		if err != nil {
			return err
		}

		store := credentials.New()
		now := time.Now()

		fmt.Println("=== Slot Backups ===")
		for _, slot := range accounts.SortedSlots(seq) {
			key := fmt.Sprintf("%d", slot)
			rec := seq.Accounts[key]
			active := ""
			if seq.ActiveSlot == slot {
				active = " [ACTIVE]"
			}
			fmt.Printf("\nSlot %d: %s%s\n", slot, rec.Email, active)

			creds, err := store.ReadBackup(slot, rec.Email)
			if err != nil {
				fmt.Printf("  backup creds : ERROR: %v\n", err)
				continue
			}
			tok := creds.ClaudeAiOauth
			expiresAt := time.UnixMilli(tok.ExpiresAt)
			expired := now.After(expiresAt)
			expiredStr := "OK"
			if expired {
				expiredStr = "EXPIRED"
			}
			fmt.Printf("  access token : %s (expires %s) [%s]\n",
				maskToken(tok.AccessToken), expiresAt.Format("2006-01-02 15:04 UTC"), expiredStr)
			if tok.RefreshToken != "" {
				fmt.Printf("  refresh token: present (%s...)\n", maskToken(tok.RefreshToken))
			} else {
				fmt.Printf("  refresh token: MISSING\n")
			}
		}

		fmt.Println("\n=== Live Keychain ===")
		liveCreds, err := store.ReadLive()
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		} else {
			tok := liveCreds.ClaudeAiOauth
			expiresAt := time.UnixMilli(tok.ExpiresAt)
			expired := now.After(expiresAt)
			expiredStr := "OK"
			if expired {
				expiredStr = "EXPIRED"
			}
			fmt.Printf("  access token : %s (expires %s) [%s]\n",
				maskToken(tok.AccessToken), expiresAt.Format("2006-01-02 15:04 UTC"), expiredStr)
			if tok.RefreshToken != "" {
				fmt.Printf("  refresh token: present\n")
			} else {
				fmt.Printf("  refresh token: MISSING\n")
			}
		}

		fmt.Println("\n=== Live ~/.claude.json ===")
		liveAcct, err := switcher.ReadLiveAccount()
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		} else {
			fmt.Printf("  email  : %s\n", liveAcct.EmailAddress)
			fmt.Printf("  uuid   : %s\n", liveAcct.UUID)
		}

		printProcessStatus()

		return nil
	},
}

func maskToken(t string) string {
	if len(t) < 12 {
		if t == "" {
			return "(empty)"
		}
		return t
	}
	return t[:8] + "..." + t[len(t)-4:]
}
