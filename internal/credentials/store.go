// Package credentials provides a unified interface for reading and writing
// Claude Code credentials across macOS (Keychain), Linux (file), and Windows (Credential Manager).
package credentials

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sharvari/claude-flipper/internal/models"
)

// Store is the platform-specific credential backend.
// Each platform (darwin/linux/windows) provides its own implementation.
type Store interface {
	// ReadLive reads the credentials currently used by Claude Code.
	ReadLive() (*models.ClaudeCredentials, error)

	// WriteLive writes credentials so that Claude Code picks them up on next launch.
	WriteLive(creds *models.ClaudeCredentials) error

	// ReadBackup reads a previously saved backup for the given slot/email.
	ReadBackup(slot int, email string) (*models.ClaudeCredentials, error)

	// WriteBackup saves credentials as a backup for the given slot/email.
	WriteBackup(slot int, email string, creds *models.ClaudeCredentials) error

	// DeleteBackup removes the backup for the given slot/email (best-effort).
	DeleteBackup(slot int, email string)
}

// marshalCreds serialises credentials to pretty JSON.
func marshalCreds(creds *models.ClaudeCredentials) ([]byte, error) {
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal credentials: %w", err)
	}
	return data, nil
}

// unmarshalCreds deserialises JSON into a ClaudeCredentials struct.
func unmarshalCreds(data []byte) (*models.ClaudeCredentials, error) {
	var creds models.ClaudeCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return &creds, nil
}

// atomicWriteFile writes data to path atomically (tmp + rename) with mode 0600.
func atomicWriteFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
