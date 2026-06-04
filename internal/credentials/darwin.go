//go:build darwin

package credentials

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sharvari/claude-flipper/internal/models"
	"github.com/sharvari/claude-flipper/internal/paths"
)

const (
	// keychainServiceLive is the service name Claude Code uses for its own credentials.
	keychainServiceLive = "Claude Code-credentials"
	// keychainServicePrefix is the prefix flipper uses for its backups.
	keychainServicePrefix = "claude-flipper"
)

// New returns the macOS Keychain-backed credential store.
func New() Store {
	return &darwinStore{}
}

type darwinStore struct{}

// ReadLive reads the Claude Code Keychain entry and returns the parsed credentials.
// Tries all matching service names and returns the first one that parses successfully.
func (d *darwinStore) ReadLive() (*models.ClaudeCredentials, error) {
	services := keychainServices()
	var lastErr error
	for _, svc := range services {
		out, err := exec.Command(
			"security", "find-generic-password",
			"-s", svc,
			"-w",
		).Output()
		if err != nil {
			lastErr = fmt.Errorf("keychain read (%s): %w", svc, err)
			continue
		}
		creds, err := unmarshalCreds(decodeKeychainOutput(string(out)))
		if err != nil {
			lastErr = err
			continue
		}
		return creds, nil
	}
	return nil, fmt.Errorf("keychain read (live): %w", lastErr)
}

// WriteLive writes credentials to all Claude Code Keychain entries.
// The plain entry is written first (it's the primary one Claude Code reads).
// Hashed variants are updated best-effort — a failure there does not abort the swap.
//
// We delete-then-create instead of using -U (update-in-place) because the original
// Claude Code entries carry restrictive ACLs. Updating with -U preserves those ACLs
// and causes Claude Code to silently fail to read the new token.
//
// We add the Claude Code binary as a trusted application (-T) so macOS does not
// show a Keychain access dialog when Claude Code reads the entry we write.
func (d *darwinStore) WriteLive(creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	user := currentUser()

	services := keychainServices()

	for i, svc := range services {
		// Delete existing entry first so the fresh entry gets a clean ACL.
		// Ignore error — entry may not exist.
		_ = exec.Command("security", "delete-generic-password", "-s", svc).Run()

		// -A: allow any application to access this item without a Keychain
		// access dialog. Without this, macOS prompts every time Claude Code
		// (or any new process) reads an entry written by a different process.
		cmd := exec.Command(
			"security", "add-generic-password",
			"-s", svc,
			"-a", user,
			"-w", string(data),
			"-A",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			if i == 0 {
				// Plain entry is required — fail fast.
				return fmt.Errorf("keychain write (%s): %w — %s", svc, err, strings.TrimSpace(string(out)))
			}
			// Hashed variant — best-effort, skip on failure.
		}
	}
	return nil
}


// keychainServices returns all Keychain service names matching "Claude Code-credentials*".
// The plain entry ("Claude Code-credentials") is always first because that is the entry
// Claude Code writes fresh tokens to; hashed variants are secondary.
func keychainServices() []string {
	out, err := exec.Command("security", "dump-keychain").Output()
	if err != nil {
		return []string{keychainServiceLive}
	}

	seen := map[string]bool{}
	var hashed []string // hashed variants e.g. "Claude Code-credentials-753cc65a"

	for _, line := range strings.Split(string(out), "\n") {
		// Find the pattern ="Claude Code-credentials..." to extract the service name.
		// We look for this pattern rather than taking between first/last quote because
		// dump-keychain output can contain deeply nested attribute lines like:
		//   "svce"<blob>="svce"<blob>="Claude Code-credentials-753cc65a"
		// which would cause the naive first/last-quote extraction to include garbage.
		idx := strings.Index(line, `="Claude Code-credentials`)
		if idx < 0 {
			continue
		}
		// after = `"Claude Code-credentials..."`
		after := line[idx+1:]
		if len(after) == 0 || after[0] != '"' {
			continue
		}
		end := strings.Index(after[1:], `"`)
		if end < 0 {
			continue
		}
		svc := after[1 : end+1]
		// Reject garbage: valid service names only contain safe characters.
		if strings.ContainsAny(svc, `"<>=`) {
			continue
		}
		if seen[svc] {
			continue
		}
		seen[svc] = true
		if svc == keychainServiceLive {
			// plain entry — handled separately so it always goes first
		} else {
			hashed = append(hashed, svc)
		}
	}

	// Plain entry first (Claude Code's primary credential store), then hashed variants.
	services := []string{keychainServiceLive}
	services = append(services, hashed...)
	return services
}

// ReadBackup reads a flipper backup from the Keychain.
func (d *darwinStore) ReadBackup(slot int, email string) (*models.ClaudeCredentials, error) {
	svc := backupService(slot, email)
	out, err := exec.Command(
		"security", "find-generic-password",
		"-s", svc,
		"-w",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("keychain read (backup %s): %w", svc, err)
	}
	return unmarshalCreds(decodeKeychainOutput(string(out)))
}

// WriteBackup saves credentials to the Keychain under a cs-specific service name.
func (d *darwinStore) WriteBackup(slot int, email string, creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	svc := backupService(slot, email)
	user := currentUser()
	cmd := exec.Command(
		"security", "add-generic-password",
		"-U",
		"-s", svc,
		"-a", user,
		"-X", hex.EncodeToString(data),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write (backup %s): %w — %s", svc, err, strings.TrimSpace(string(out)))
	}
	// Also write a file backup as a safety net so the backup dir exists and
	// callers can enumerate backed-up slots without querying Keychain.
	filePath := paths.CredentialsBackupFile(slot, email)
	if mkErr := os.MkdirAll(filepath.Dir(filePath), 0700); mkErr == nil {
		_ = atomicWriteFile(filePath, data)
	}
	return nil
}

// DeleteBackup removes the flipper Keychain backup for a given slot/email (best-effort).
func (d *darwinStore) DeleteBackup(slot int, email string) {
	svc := backupService(slot, email)
	_ = exec.Command("security", "delete-generic-password", "-s", svc).Run()
	_ = os.Remove(paths.CredentialsBackupFile(slot, email))
}

// backupService builds the Keychain service name for a flipper backup entry.
func backupService(slot int, email string) string {
	return fmt.Sprintf("%s-%d-%s", keychainServicePrefix, slot, email)
}

// currentUser returns the current OS username, falling back to "claude".
func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "claude"
}

// decodeKeychainOutput handles the macOS security command returning hex-encoded
// data when the stored value contains control characters (e.g. JSON newlines).
func decodeKeychainOutput(raw string) []byte {
	trimmed := strings.TrimSpace(raw)
	decoded, err := hex.DecodeString(trimmed)
	if err == nil {
		return decoded
	}
	return []byte(trimmed)
}
