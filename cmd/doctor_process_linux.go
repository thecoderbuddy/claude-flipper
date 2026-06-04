//go:build linux

package cmd

import "fmt"

func printProcessStatus() {
	fmt.Println("\n=== Claude Process ===")
	fmt.Println("  process check not applicable on Linux")
}
