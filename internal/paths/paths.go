package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ClaudeConfigDir returns the path to Claude Code's config directory.
func ClaudeConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// ClaudeConfigFile returns the path to Claude Code's main config file.
// Claude Code stores config at ~/.claude.json (not inside ~/.claude/).
func ClaudeConfigFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude.json")
}

// ClaudeCredentialsFile returns the path to Claude Code's credentials file (Linux/Windows).
func ClaudeCredentialsFile() string {
	return filepath.Join(ClaudeConfigDir(), ".credentials.json")
}

// DataDir returns the claude-flipper data directory.
func DataDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "linux":
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "claude-flipper")
		}
		return filepath.Join(home, ".local", "share", "claude-flipper")
	default:
		return filepath.Join(home, ".claude-flipper")
	}
}

// SequenceFile returns the path to the master metadata file.
func SequenceFile() string {
	return filepath.Join(DataDir(), "sequence.json")
}

// CredentialsBackupFile returns the path to a backup credentials file for a given slot/email.
func CredentialsBackupFile(slot int, email string) string {
	return filepath.Join(DataDir(), "credentials", fmt.Sprintf("%d-%s.json", slot, sanitize(email)))
}

// ConfigBackupFile returns the path to a backup config file for a given slot/email.
func ConfigBackupFile(slot int, email string) string {
	return filepath.Join(DataDir(), "configs", fmt.Sprintf("%d-%s.json", slot, sanitize(email)))
}

// LockFile returns the path to the cross-process lock file.
func LockFile() string {
	return filepath.Join(DataDir(), ".lock")
}

// sanitize strips characters unsafe for filenames, keeping alphanumerics, @, dot, hyphen.
func sanitize(email string) string {
	safe := ""
	for _, c := range email {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '@' || c == '.' || c == '-' {
			safe += string(c)
		} else {
			safe += "_"
		}
	}
	return safe
}
