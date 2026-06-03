//go:build windows

package credentials

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danieljoos/wincred"
	"github.com/sharvari/claude-flipper/internal/models"
	"github.com/sharvari/claude-flipper/internal/paths"
)

const (
	// wincredTargetLive is the Credential Manager target name that Claude Code uses.
	wincredTargetLive = "Claude Code-credentials"
	// wincredTargetPrefix is the prefix flipper uses for its backups.
	wincredTargetPrefix = "claude-flipper"
)

// New returns the Windows Credential Manager backed credential store.
func New() Store {
	return &windowsStore{}
}

type windowsStore struct{}

// ReadLive reads the Claude Code credential from Windows Credential Manager.
func (w *windowsStore) ReadLive() (*models.ClaudeCredentials, error) {
	cred, err := wincred.GetGenericCredential(wincredTargetLive)
	if err != nil {
		return nil, fmt.Errorf("wincred read (live): %w", err)
	}
	return unmarshalCreds(cred.CredentialBlob)
}

// WriteLive writes credentials to Windows Credential Manager under Claude Code's target name.
func (w *windowsStore) WriteLive(creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}
	cred := wincred.NewGenericCredential(wincredTargetLive)
	cred.CredentialBlob = data
	if err := cred.Write(); err != nil {
		return fmt.Errorf("wincred write (live): %w", err)
	}
	return nil
}

// ReadBackup reads a flipper backup from Windows Credential Manager, falling back to
// the file backup if the Credential Manager entry is missing.
func (w *windowsStore) ReadBackup(slot int, email string) (*models.ClaudeCredentials, error) {
	target := backupTarget(slot, email)
	cred, err := wincred.GetGenericCredential(target)
	if err == nil {
		return unmarshalCreds(cred.CredentialBlob)
	}
	// Fall back to file.
	p := paths.CredentialsBackupFile(slot, email)
	data, ferr := os.ReadFile(p)
	if ferr != nil {
		return nil, fmt.Errorf("read credentials backup (slot %d, %s): wincred: %w; file: %w", slot, email, err, ferr)
	}
	return unmarshalCreds(data)
}

// WriteBackup saves credentials to both Windows Credential Manager and a file backup.
func (w *windowsStore) WriteBackup(slot int, email string, creds *models.ClaudeCredentials) error {
	data, err := marshalCreds(creds)
	if err != nil {
		return err
	}

	// Write to Credential Manager.
	target := backupTarget(slot, email)
	cred := wincred.NewGenericCredential(target)
	cred.CredentialBlob = data
	if credErr := cred.Write(); credErr != nil {
		return fmt.Errorf("wincred write (backup %s): %w", target, credErr)
	}

	// Also write a file backup.
	p := paths.CredentialsBackupFile(slot, email)
	if mkErr := os.MkdirAll(filepath.Dir(p), 0700); mkErr == nil {
		_ = atomicWriteFile(p, data)
	}
	return nil
}

// DeleteBackup removes the Credential Manager entry and file backup (best-effort).
func (w *windowsStore) DeleteBackup(slot int, email string) {
	target := backupTarget(slot, email)
	if cred, err := wincred.GetGenericCredential(target); err == nil {
		_ = cred.Delete()
	}
	_ = os.Remove(paths.CredentialsBackupFile(slot, email))
}

func backupTarget(slot int, email string) string {
	return fmt.Sprintf("%s-%d-%s", wincredTargetPrefix, slot, email)
}
