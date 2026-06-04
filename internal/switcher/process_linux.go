//go:build linux

package switcher

// claudeAppRunning always returns false on Linux — there is no desktop app
// that holds credential files open the way Claude.app does on macOS.
func claudeAppRunning() bool {
	return false
}
