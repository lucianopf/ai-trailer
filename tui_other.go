//go:build !linux
// +build !linux

// Package main — interactive menu for non-Linux (macOS, Windows).
// Uses ANSI escape codes via stty raw mode for a full checkbox TUI.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func showInteractiveMenu(items []menuItem) []menuItem {
	// Fish shell and dumb terminals break ANSI escape codes
	if isBrokenShell() {
		return showTextMenu(items)
	}

	// Disable canonical mode (char-by-char reads) while keeping output processing
	// intact so \n → \r\n still works. "stty raw" disables opost which breaks
	// column alignment; "-icanon min 1" achieves the same read behaviour without it.
	rawMode := exec.Command("stty", "-icanon", "min", "1", "-echo")
	rawMode.Stdin = os.Stdin
	if err := rawMode.Run(); err != nil {
		return showTextMenu(items)
	}

	// Restore terminal on exit
	defer func() {
		restore := exec.Command("stty", "sane")
		restore.Stdin = os.Stdin
		restore.Run()
	}()

	// Handle Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restore := exec.Command("stty", "sane")
		restore.Stdin = os.Stdin
		restore.Run()
		clearScreen()
		fmt.Println("  Interrupted.")
		os.Exit(1)
	}()
	defer signal.Stop(sigCh)

	cursor := 0
	helpText := "↑↓ move  •  SPACE toggle  •  ENTER confirm  •  a select all  •  n deselect all  •  q quit"

	for {
		renderMenu(items, cursor, helpText)

		buf := make([]byte, 1)
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			break
		}
		key := buf[0]

		switch {
		case key == 13: // Enter
			clearScreen()
			return items
		case key == 'q' || key == 'Q':
			clearScreen()
			fmt.Println("  Cancelled.")
			return nil
		case key == 3: // Ctrl+C
			clearScreen()
			fmt.Println("  Cancelled.")
			return nil
		case key == ' ':
			items[cursor].Selected = !items[cursor].Selected
		case key == 'a' || key == 'A':
			for i := range items {
				items[i].Selected = true
			}
		case key == 'n' || key == 'N':
			for i := range items {
				items[i].Selected = false
			}
		case key == 'j':
			if cursor < len(items)-1 {
				cursor++
			}
		case key == 'k':
			if cursor > 0 {
				cursor--
			}
		case key == 'w':
			if cursor > 0 {
				cursor--
			}
		case key == 's':
			if cursor < len(items)-1 {
				cursor++
			}
		case key == 27: // Escape sequence: arrows
			more := make([]byte, 2)
			os.Stdin.Read(more)
			if more[0] == '[' {
				switch more[1] {
				case 'A': // Up
					if cursor > 0 {
						cursor--
					}
				case 'B': // Down
					if cursor < len(items)-1 {
						cursor++
					}
				}
			}
		}
	}
	clearScreen()
	return items
}

// pr prints a line with explicit \r\n so the cursor returns to column 0
// regardless of whether the terminal has opost/onlcr enabled.
func pr(format string, args ...any) {
	if len(args) == 0 {
		fmt.Print(format + "\r\n")
	} else {
		fmt.Printf(format+"\r\n", args...)
	}
}

func renderMenu(items []menuItem, cursor int, help string) {
	fmt.Print("\033[2J\033[H") // clear screen, cursor to home
	pr("")
	pr("  ⚙  AI Trailer Configuration")
	pr("  " + strings.Repeat("─", 55))
	pr("")

	selectedCount := 0
	for _, item := range items {
		if item.Selected {
			selectedCount++
		}
	}
	pr("  Select AI tools to configure (%d/%d selected)", selectedCount, len(items))
	pr("")

	for i, item := range items {
		line := "  "
		if i == cursor {
			line = "  \033[7m"
		}
		if item.Selected {
			line += "[✓] "
		} else {
			line += "[ ] "
		}
		display := item.ToolName
		if item.IsNative {
			display += " (native)"
		}
		line += padRight(display, 36)
		if item.Detected {
			line += " \033[32m● detected\033[0m"
		} else {
			line += " \033[90m○ not found\033[0m"
		}
		if i == cursor {
			line += "\033[0m"
		}
		pr("%s", line)
		if i == cursor {
			trailerLine := "Ai-tool: " + item.ToolID + "  •  " + item.Trailer
			pr("     \033[7m↳ %s\033[0m", truncateStr(trailerLine, 72))
		}
	}
	pr("")
	pr("  " + strings.Repeat("─", 55))
	fmt.Print("  \033[90m" + help + "\033[0m")
}

func clearScreen() { fmt.Print("\033[2J\033[H\r") }
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
