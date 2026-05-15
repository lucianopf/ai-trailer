// Package main — shared types and text-based menu fallback for the TUI.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// menuItem represents a selectable tool in the interactive menu.
type menuItem struct {
	ToolName string
	Trailer  string
	Selected bool
	Detected bool // Was auto-detected (pre-selected)
	IsNative bool // Has native trailer support
}

// showTextMenu is the plain-text fallback used when:
// - Running under fish shell (ANSI escapes break)
// - Non-Linux platforms
// - Terminal doesn't support raw mode
func showTextMenu(items []menuItem) []menuItem {
	fmt.Println()
	fmt.Println("  ⚙  AI Trailer Configuration")
	fmt.Println("  " + strings.Repeat("─", 55))
	fmt.Println()

	detectedCount := 0
	for _, item := range items {
		if item.Detected {
			detectedCount++
		}
	}
	fmt.Printf("  %d tool(s) auto-detected [●]. Select which to configure:\n", detectedCount)
	fmt.Println()

	for i, item := range items {
		mark := "[ ]"
		if item.Selected {
			mark = "[✓]"
		}
		note := ""
		if item.Detected {
			note = " ●"
		} else {
			note = " ○"
		}
		if item.IsNative {
			note += " (native)"
		}
		fmt.Printf("  %2d. %s %s%s\n", i+1, mark, item.ToolName, note)
	}

	fmt.Println()
	fmt.Println("  Enter numbers to toggle (e.g., '1 3 5'), 'all', or press Enter for detected:")
	fmt.Print("  > ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return items
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return items
	}
	if input == "all" || input == "a" {
		for i := range items {
			items[i].Selected = true
		}
		return items
	}

	for _, part := range strings.Fields(input) {
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 && n <= len(items) {
			items[n-1].Selected = !items[n-1].Selected
		}
	}

	return items
}
