//go:build darwin

package switcher

import (
	"bytes"
	"os/exec"
)

// claudeAppRunning returns true if the Claude.app desktop process is running.
// On macOS, Claude.app overwrites Keychain entries while open — swapping while
// it is running would be immediately undone.
func claudeAppRunning() bool {
	out, err := exec.Command("pgrep", "-f", "Claude.app/Contents/MacOS/Claude").Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) > 0
}
