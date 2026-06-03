//go:build linux

package credentials

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sharvari/claude-flipper/internal/models"
	"github.com/sharvari/claude-flipper/internal/paths"
)

// New returns the Linux file-based credential store.
func New() Store {
	return &linuxStore{}
}

type linuxStore struct{}

// ReadLive reads credentials from ~/.claude/.credentials.json.
func (l *linuxStore) ReadLive() (*models.ClaudeCredentials, error) {
	data, err := os.ReadFile(paths.ClaudeCredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("read live credentials: %w", err)
	}
	return unmarshalCreds(data)
}

// WriteLive atomically writes credentials to ~/.claude/.credentials.json.
func (l *linuxStore) WriteLive(creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(paths.ClaudeCredentialsFile()), 0700); err != nil {
		return fmt.Errorf("create claude config dir: %w", err)
	}
	return atomicWriteFile(paths.ClaudeCredentialsFile(), data)
}

// ReadBackup reads a previously saved credentials backup file.
func (l *linuxStore) ReadBackup(slot int, email string) (*models.ClaudeCredentials, error) {
	p := paths.CredentialsBackupFile(slot, email)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read credentials backup (slot %d, %s): %w", slot, email, err)
	}
	return unmarshalCreds(data)
}

// WriteBackup saves credentials to the data-dir backup location.
func (l *linuxStore) WriteBackup(slot int, email string, creds *models.ClaudeCredentials) error {
	p := paths.CredentialsBackupFile(slot, email)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("create credentials backup dir: %w", err)
	}
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	return atomicWriteFile(p, data)
}

// DeleteBackup removes the backup file (best-effort).
func (l *linuxStore) DeleteBackup(slot int, email string) {
	_ = os.Remove(paths.CredentialsBackupFile(slot, email))
}
