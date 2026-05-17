//go:build linux
// +build linux

// Package main — interactive terminal UI for Linux (raw syscalls).
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"unsafe"
)

type termState struct {
	termios syscall.Termios
	fd      int
	old     syscall.Termios
	saved   bool
}

func makeRaw(fd int) (*termState, error) {
	var st termState
	st.fd = fd
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		syscall.TCGETS, uintptr(unsafe.Pointer(&st.old)), 0, 0, 0); err != 0 {
		return nil, err
	}
	st.termios = st.old
	st.termios.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	st.termios.Oflag &^= syscall.OPOST
	st.termios.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	st.termios.Cflag &^= syscall.CSIZE | syscall.PARENB
	st.termios.Cflag |= syscall.CS8
	st.termios.Cc[syscall.VMIN] = 1
	st.termios.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		syscall.TCSETS, uintptr(unsafe.Pointer(&st.termios)), 0, 0, 0); err != 0 {
		return nil, err
	}
	st.saved = true
	return &st, nil
}

func (st *termState) restore() {
	if st.saved {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(st.fd),
			syscall.TCSETS, uintptr(unsafe.Pointer(&st.old)), 0, 0, 0)
		st.saved = false
	}
}

func isTerminal(fd int) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		syscall.TCGETS, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

func showInteractiveMenu(items []menuItem) []menuItem {
	// Fish shell and dumb terminals break ANSI escape codes
	if isBrokenShell() {
		return showTextMenu(items)
	}

	fd := int(os.Stdin.Fd())
	if !isTerminal(fd) {
		for i := range items {
			items[i].Selected = items[i].Detected
		}
		return items
	}

	state, err := makeRaw(fd)
	if err != nil {
		for i := range items {
			items[i].Selected = items[i].Detected
		}
		return items
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		state.restore()
		clearScreen()
		fmt.Println("  Interrupted.")
		os.Exit(1)
	}()
	defer signal.Stop(sigCh)
	defer state.restore()

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
		case key == 27: // Escape sequence: arrows
			more := make([]byte, 2)
			os.Stdin.Read(more)
			if more[0] == '[' {
				switch more[1] {
				case 'A':
					if cursor > 0 {
						cursor--
					}
				case 'B':
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

func renderMenu(items []menuItem, cursor int, help string) {
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println("  ⚙  AI Trailer Configuration")
	fmt.Println("  " + strings.Repeat("─", 55))
	fmt.Println()

	selectedCount := 0
	for _, item := range items {
		if item.Selected {
			selectedCount++
		}
	}
	fmt.Printf("  Select AI tools to configure (%d/%d selected)\n\n", selectedCount, len(items))

	for i, item := range items {
		if i == cursor {
			fmt.Print("  \033[7m")
		} else {
			fmt.Print("  ")
		}
		if item.Selected {
			fmt.Print("[✓] ")
		} else {
			fmt.Print("[ ] ")
		}
		display := item.ToolName
		if item.IsNative {
			display += " (native)"
		}
		fmt.Print(padRight(display, 32))
		if item.Detected {
			fmt.Print(" \033[32m● detected\033[0m")
		} else {
			fmt.Print(" \033[90m○ not found\033[0m")
		}
		if i == cursor {
			fmt.Print("\n   \033[7m  ↳ " + truncateStr(item.Trailer, 55) + "\033[0m")
		}
		if i == cursor {
			fmt.Print("\033[0m")
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("  " + strings.Repeat("─", 55))
	fmt.Print("  \033[90m" + help + "\033[0m")
}

func clearScreen() { fmt.Print("\033[2J\033[H") }
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
