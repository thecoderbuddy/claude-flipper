// Package switcher implements the core account-switching logic with rollback support.
package switcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sharvari/claude-flipper/internal/accounts"
	"github.com/sharvari/claude-flipper/internal/credentials"
	"github.com/sharvari/claude-flipper/internal/lock"
	"github.com/sharvari/claude-flipper/internal/models"
	"github.com/sharvari/claude-flipper/internal/paths"
)

// AddCurrent reads the currently active Claude Code account (config + credentials) and
// saves it as a new slot in sequence.json.
func AddCurrent() (int, string, error) {
	// Read live config.
	oauthAcct, err := readLiveOAuthAccount()
	if err != nil {
		return 0, "", fmt.Errorf("read live config: %w", err)
	}

	store := credentials.New()

	// Read live credentials.
	creds, err := store.ReadLive()
	if err != nil {
		return 0, "", fmt.Errorf("read live credentials: %w", err)
	}

	// Load sequence.
	seq, err := accounts.Load()
	if err != nil {
		return 0, "", fmt.Errorf("load sequence: %w", err)
	}

	// Register account in sequence.
	slot, err := accounts.AddAccount(seq, oauthAcct)
	if err != nil {
		return 0, "", err
	}

	// Backup credentials.
	if err := store.WriteBackup(slot, oauthAcct.EmailAddress, creds); err != nil {
		return 0, "", fmt.Errorf("backup credentials: %w", err)
	}

	// Backup config.
	if err := writeConfigBackup(slot, oauthAcct); err != nil {
		return 0, "", fmt.Errorf("backup config: %w", err)
	}

	// If this is the very first account, mark it active.
	if seq.ActiveSlot == 0 {
		seq.ActiveSlot = slot
	}

	if err := accounts.Save(seq); err != nil {
		return 0, "", fmt.Errorf("save sequence: %w", err)
	}

	return slot, oauthAcct.EmailAddress, nil
}

// SwitchTo switches the active Claude Code account to the one identified by slotOrEmail.
// Pass "" to rotate to the next account in sequence.
func SwitchTo(slotOrEmail string) (int, string, error) {
	l, err := lock.Acquire(paths.LockFile())
	if err != nil {
		return 0, "", fmt.Errorf("acquire lock: %w", err)
	}
	defer l.Release()

	seq, err := accounts.Load()
	if err != nil {
		return 0, "", fmt.Errorf("load sequence: %w", err)
	}

	var targetSlot int
	var targetRec models.AccountRecord

	if slotOrEmail == "" {
		targetSlot, targetRec, err = accounts.NextInSequence(seq)
	} else {
		targetSlot, targetRec, err = accounts.FindAccount(seq, slotOrEmail)
	}
	if err != nil {
		return 0, "", err
	}

	if seq.ActiveSlot == targetSlot {
		return targetSlot, targetRec.Email, fmt.Errorf("already on account %q (slot %d)", targetRec.Email, targetSlot)
	}

	store := credentials.New()

	// ---- Rollback bookkeeping ----
	var rollbacks []func()

	// Step 1: backup the current live credentials and config (so we can restore them on failure).
	currentSlot := seq.ActiveSlot
	var currentEmail string
	if currentSlot != 0 {
		curKey := strconv.Itoa(currentSlot)
		if curRec, ok := seq.Accounts[curKey]; ok {
			currentEmail = curRec.Email
		}
	}

	// Snapshot current live credentials (for rollback only; don't overwrite existing backup).
	var liveCredsSnapshot *models.ClaudeCredentials
	if currentSlot != 0 {
		liveCredsSnapshot, _ = store.ReadLive()
	}
	var liveConfigSnapshot *models.OAuthAccount
	if currentSlot != 0 {
		snap, _ := readLiveOAuthAccount()
		liveConfigSnapshot = &snap
	}

	// Step 2: load target credentials from backup.
	targetCreds, err := store.ReadBackup(targetSlot, targetRec.Email)
	if err != nil {
		return 0, "", fmt.Errorf("load target credentials (slot %d): %w", targetSlot, err)
	}

	// Step 3: load target config from backup.
	targetOAuth, err := readConfigBackup(targetSlot, targetRec.Email)
	if err != nil {
		return 0, "", fmt.Errorf("load target config (slot %d): %w", targetSlot, err)
	}

	// Step 4: write target credentials as live.
	if err := store.WriteLive(targetCreds); err != nil {
		return 0, "", fmt.Errorf("write live credentials: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		if liveCredsSnapshot != nil {
			_ = store.WriteLive(liveCredsSnapshot)
		}
	})

	// Step 5: update ~/.claude/.config.json oauthAccount section.
	if err := writeLiveOAuthAccount(targetOAuth); err != nil {
		runRollbacks(rollbacks)
		return 0, "", fmt.Errorf("write live config: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		if liveConfigSnapshot != nil {
			_ = writeLiveOAuthAccount(*liveConfigSnapshot)
		}
	})

	// Step 6: update sequence activeSlot.
	oldActiveSlot := seq.ActiveSlot
	seq.ActiveSlot = targetSlot
	if err := accounts.Save(seq); err != nil {
		runRollbacks(rollbacks)
		seq.ActiveSlot = oldActiveSlot
		return 0, "", fmt.Errorf("save sequence: %w", err)
	}

	// Update the backup for the previous account in case credentials changed since last add.
	if currentSlot != 0 && currentEmail != "" && liveCredsSnapshot != nil {
		_ = store.WriteBackup(currentSlot, currentEmail, liveCredsSnapshot)
	}

	return targetSlot, targetRec.Email, nil
}

// ReadLiveAccount is the exported form — used by the status command to show what
// Claude Code currently has active, independent of flipper tracking.
func ReadLiveAccount() (models.OAuthAccount, error) {
	return readLiveOAuthAccount()
}

// readLiveOAuthAccount reads the oauthAccount field from ~/.claude/.config.json.
func readLiveOAuthAccount() (models.OAuthAccount, error) {
	data, err := os.ReadFile(paths.ClaudeConfigFile())
	if err != nil {
		return models.OAuthAccount{}, fmt.Errorf("read config file: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return models.OAuthAccount{}, fmt.Errorf("parse config file: %w", err)
	}
	oauthRaw, ok := raw["oauthAccount"]
	if !ok {
		return models.OAuthAccount{}, fmt.Errorf("oauthAccount not found in config; is Claude Code logged in?")
	}
	var acct models.OAuthAccount
	if err := json.Unmarshal(oauthRaw, &acct); err != nil {
		return models.OAuthAccount{}, fmt.Errorf("parse oauthAccount: %w", err)
	}
	if acct.EmailAddress == "" {
		return models.OAuthAccount{}, fmt.Errorf("oauthAccount.emailAddress is empty; is Claude Code logged in?")
	}
	return acct, nil
}

// writeLiveOAuthAccount replaces only the oauthAccount field in ~/.claude/.config.json.
func writeLiveOAuthAccount(acct models.OAuthAccount) error {
	configFile := paths.ClaudeConfigFile()
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	oauthBytes, err := json.Marshal(acct)
	if err != nil {
		return fmt.Errorf("marshal oauthAccount: %w", err)
	}
	raw["oauthAccount"] = oauthBytes

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmp := configFile + ".tmp"
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, configFile); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp config: %w", err)
	}
	return nil
}

// writeConfigBackup saves just the oauthAccount section for a given slot/email.
func writeConfigBackup(slot int, acct models.OAuthAccount) error {
	p := paths.ConfigBackupFile(slot, acct.EmailAddress)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("create config backup dir: %w", err)
	}
	data, err := json.MarshalIndent(acct, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config backup: %w", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write temp config backup: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename config backup: %w", err)
	}
	return nil
}

// readConfigBackup loads the oauthAccount backup for a given slot/email.
func readConfigBackup(slot int, email string) (models.OAuthAccount, error) {
	p := paths.ConfigBackupFile(slot, email)
	data, err := os.ReadFile(p)
	if err != nil {
		return models.OAuthAccount{}, fmt.Errorf("read config backup (slot %d): %w", slot, err)
	}
	var acct models.OAuthAccount
	if err := json.Unmarshal(data, &acct); err != nil {
		return models.OAuthAccount{}, fmt.Errorf("parse config backup: %w", err)
	}
	return acct, nil
}

// runRollbacks executes rollback functions in reverse order (LIFO).
func runRollbacks(fns []func()) {
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}
