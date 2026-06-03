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
// Claude Code may use a hashed service name (e.g. "Claude Code-credentials-753cc65a")
// in addition to the plain one, so we update all matching entries.
func (d *darwinStore) WriteLive(creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	user := currentUser()

	// Find all Claude Code credential service names in the Keychain.
	services := keychainServices()

	var lastErr error
	for _, svc := range services {
		cmd := exec.Command(
			"security", "add-generic-password",
			"-U",
			"-s", svc,
			"-a", user,
			"-w", string(data),
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("keychain write (%s): %w — %s", svc, err, strings.TrimSpace(string(out)))
		}
	}
	return lastErr
}

// keychainServices returns all Keychain service names matching "Claude Code-credentials*".
func keychainServices() []string {
	out, err := exec.Command("security", "dump-keychain").Output()
	if err != nil {
		return []string{keychainServiceLive}
	}

	seen := map[string]bool{}
	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Claude Code-credentials") {
			// Extract the service name from: 0x00000007 <blob>="Claude Code-credentials-abc123"
			start := strings.Index(line, `"`)
			end := strings.LastIndex(line, `"`)
			if start >= 0 && end > start {
				svc := line[start+1 : end]
				if !seen[svc] {
					seen[svc] = true
					services = append(services, svc)
				}
			}
		}
	}

	if len(services) == 0 {
		return []string{keychainServiceLive}
	}
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
