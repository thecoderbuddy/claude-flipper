//go:build windows

package switcher

// claudeAppRunning always returns false on Windows for now.
func claudeAppRunning() bool {
	return false
}
