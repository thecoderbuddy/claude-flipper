//go:build darwin

package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

func printProcessStatus() {
	fmt.Println("\n=== Claude.app Process ===")
	out, err := exec.Command("pgrep", "-f", "Claude.app/Contents/MacOS/Claude").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		fmt.Println("  not running (good)")
	} else {
		pids := strings.TrimSpace(string(out))
		fmt.Printf("  RUNNING (PIDs: %s) — quit before swapping\n", pids)
	}
}
