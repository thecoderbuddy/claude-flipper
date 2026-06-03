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

// AddResult is returned by AddCurrent to indicate whether the account was new or refreshed.
type AddResult struct {
	Slot  int
	Email string
	IsNew bool
}

// AddCurrent reads the currently active Claude Code account (config + credentials) and
// saves it as a new slot in sequence.json. If the account is already registered, it
// refreshes the stored credentials and config (useful after a token refresh).
func AddCurrent() (AddResult, error) {
	// Read live config (oauthAccount + userID).
	cfg, err := readLiveAccountConfig()
	if err != nil {
		return AddResult{}, fmt.Errorf("read live config: %w", err)
	}

	store := credentials.New()

	// Read live credentials.
	creds, err := store.ReadLive()
	if err != nil {
		return AddResult{}, fmt.Errorf("read live credentials: %w", err)
	}

	// Load sequence.
	seq, err := accounts.Load()
	if err != nil {
		return AddResult{}, fmt.Errorf("load sequence: %w", err)
	}

	// Register or find existing account.
	slot, isNew, err := accounts.AddAccount(seq, cfg.OAuthAccount)
	if err != nil {
		return AddResult{}, err
	}

	// Backup credentials (always refresh — tokens may have rotated).
	if err := store.WriteBackup(slot, cfg.OAuthAccount.EmailAddress, creds); err != nil {
		return AddResult{}, fmt.Errorf("backup credentials: %w", err)
	}

	// Backup config (oauthAccount + userID).
	if err := writeConfigBackup(slot, cfg); err != nil {
		return AddResult{}, fmt.Errorf("backup config: %w", err)
	}

	// If this is the very first account, mark it active.
	if seq.ActiveSlot == 0 {
		seq.ActiveSlot = slot
	}

	if err := accounts.Save(seq); err != nil {
		return AddResult{}, fmt.Errorf("save sequence: %w", err)
	}

	return AddResult{Slot: slot, Email: cfg.OAuthAccount.EmailAddress, IsNew: isNew}, nil
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

	// Snapshot current live credentials and config (for rollback + backup update).
	var liveCredsSnapshot *models.ClaudeCredentials
	if currentSlot != 0 {
		liveCredsSnapshot, _ = store.ReadLive()
	}
	var liveConfigSnapshot *models.AccountConfig
	if currentSlot != 0 {
		snap, _ := readLiveAccountConfig()
		liveConfigSnapshot = &snap
	}

	// Step 2: load target credentials from backup.
	targetCreds, err := store.ReadBackup(targetSlot, targetRec.Email)
	if err != nil {
		return 0, "", fmt.Errorf("load target credentials (slot %d): %w", targetSlot, err)
	}

	// Step 2b: refresh the access token if it has expired.
	// This handles the common case where the token expired naturally while the
	// other account was active. Falls back to the stored credentials silently
	// if the network is unavailable; returns an error if the refresh token
	// itself has been revoked (e.g. the user ran /logout in Claude Code).
	if refreshed, rerr := tryRefreshCredentials(targetCreds); rerr == nil {
		if refreshed != targetCreds {
			// Persist the refreshed token back to the backup so future swaps
			// start with a fresh token too.
			_ = store.WriteBackup(targetSlot, targetRec.Email, refreshed)
		}
		targetCreds = refreshed
	} else {
		// Non-fatal: proceed with stored credentials. Claude Code will show
		// "Not logged in" if the refresh token is also revoked (e.g. /logout was used).
		fmt.Fprintf(os.Stderr, "warning: could not refresh token for slot %d: %v\n", targetSlot, rerr)
	}

	// Step 3: load target config from backup.
	targetCfg, err := readConfigBackup(targetSlot, targetRec.Email)
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

	// Step 5: update ~/.claude.json oauthAccount and userID fields.
	if err := writeLiveAccountConfig(targetCfg); err != nil {
		runRollbacks(rollbacks)
		return 0, "", fmt.Errorf("write live config: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		if liveConfigSnapshot != nil {
			_ = writeLiveAccountConfig(*liveConfigSnapshot)
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

	// Update the backup for the previous account in case credentials/config changed since last add.
	if currentSlot != 0 && currentEmail != "" {
		if liveCredsSnapshot != nil {
			_ = store.WriteBackup(currentSlot, currentEmail, liveCredsSnapshot)
		}
		if liveConfigSnapshot != nil {
			_ = writeConfigBackup(currentSlot, *liveConfigSnapshot)
		}
	}

	return targetSlot, targetRec.Email, nil
}

// ReadLiveAccount is the exported form — used by the status command to show what
// Claude Code currently has active, independent of flipper tracking.
func ReadLiveAccount() (models.OAuthAccount, error) {
	return readLiveOAuthAccount()
}

// readLiveAccountConfig reads the oauthAccount and userID fields from ~/.claude.json.
func readLiveAccountConfig() (models.AccountConfig, error) {
	data, err := os.ReadFile(paths.ClaudeConfigFile())
	if err != nil {
		return models.AccountConfig{}, fmt.Errorf("read config file: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return models.AccountConfig{}, fmt.Errorf("parse config file: %w", err)
	}
	oauthRaw, ok := raw["oauthAccount"]
	if !ok {
		return models.AccountConfig{}, fmt.Errorf("oauthAccount not found in config; is Claude Code logged in?")
	}
	var acct models.OAuthAccount
	if err := json.Unmarshal(oauthRaw, &acct); err != nil {
		return models.AccountConfig{}, fmt.Errorf("parse oauthAccount: %w", err)
	}
	if acct.EmailAddress == "" {
		return models.AccountConfig{}, fmt.Errorf("oauthAccount.emailAddress is empty; is Claude Code logged in?")
	}
	var userID string
	if uidRaw, ok := raw["userID"]; ok {
		_ = json.Unmarshal(uidRaw, &userID)
	}
	return models.AccountConfig{OAuthAccount: acct, UserID: userID}, nil
}

// readLiveOAuthAccount is a thin wrapper used by callers that only need the OAuthAccount.
func readLiveOAuthAccount() (models.OAuthAccount, error) {
	cfg, err := readLiveAccountConfig()
	return cfg.OAuthAccount, err
}

// writeLiveAccountConfig replaces oauthAccount and userID in ~/.claude.json.
func writeLiveAccountConfig(cfg models.AccountConfig) error {
	configFile := paths.ClaudeConfigFile()
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	oauthBytes, err := json.Marshal(cfg.OAuthAccount)
	if err != nil {
		return fmt.Errorf("marshal oauthAccount: %w", err)
	}
	raw["oauthAccount"] = oauthBytes

	if cfg.UserID != "" {
		uidBytes, err := json.Marshal(cfg.UserID)
		if err != nil {
			return fmt.Errorf("marshal userID: %w", err)
		}
		raw["userID"] = uidBytes
	}

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

// writeConfigBackup saves the oauthAccount and userID for a given slot.
func writeConfigBackup(slot int, cfg models.AccountConfig) error {
	p := paths.ConfigBackupFile(slot, cfg.OAuthAccount.EmailAddress)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("create config backup dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
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

// readConfigBackup loads the AccountConfig backup for a given slot/email.
// It handles old backups that stored only OAuthAccount JSON (no userID field).
func readConfigBackup(slot int, email string) (models.AccountConfig, error) {
	p := paths.ConfigBackupFile(slot, email)
	data, err := os.ReadFile(p)
	if err != nil {
		return models.AccountConfig{}, fmt.Errorf("read config backup (slot %d): %w", slot, err)
	}
	// Try new format first (AccountConfig with oauthAccount + userID).
	var cfg models.AccountConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.AccountConfig{}, fmt.Errorf("parse config backup: %w", err)
	}
	// If oauthAccount is empty, the file was saved in old format (bare OAuthAccount).
	if cfg.OAuthAccount.EmailAddress == "" {
		var acct models.OAuthAccount
		if err := json.Unmarshal(data, &acct); err != nil {
			return models.AccountConfig{}, fmt.Errorf("parse config backup (legacy): %w", err)
		}
		cfg = models.AccountConfig{OAuthAccount: acct}
	}
	return cfg, nil
}

// runRollbacks executes rollback functions in reverse order (LIFO).
func runRollbacks(fns []func()) {
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}
