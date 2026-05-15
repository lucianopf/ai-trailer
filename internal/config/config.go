// Package config manages git hook installation and tool-specific configuration.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	hookDirName  = "ai-trailers"
	hookFileName = "prepare-commit-msg"
)

// HookDir returns the global git hooks directory path.
func HookDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".git-hooks", hookDirName), nil
}

// HookPath returns the full path to the hook script.
func HookPath() (string, error) {
	dir, err := HookDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hookFileName), nil
}

// IsHookInstalled checks if the git hook is currently configured.
func IsHookInstalled() bool {
	path, err := HookPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	// Also check git config
	dir, _ := HookDir()
	cmd := exec.Command("git", "config", "--global", "core.hooksPath")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	configured := strings.TrimSpace(string(out))
	return configured == dir
}

// InstallHook writes the git hook script and configures git globally.
func InstallHook() error {
	hookDir, err := HookDir()
	if err != nil {
		return fmt.Errorf("cannot determine hook directory: %w", err)
	}

	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("cannot create hook directory: %w", err)
	}

	hookPath, _ := HookPath()
	if err := os.WriteFile(hookPath, []byte(HookScript), 0755); err != nil {
		return fmt.Errorf("cannot write hook script: %w", err)
	}

	// Set git config
	cmd := exec.Command("git", "config", "--global", "core.hooksPath", hookDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config failed: %w\nOutput: %s", err, string(out))
	}

	return nil
}

// UninstallHook removes the hook and git config.
func UninstallHook() error {
	hookPath, _ := HookPath()
	os.Remove(hookPath)

	hookDir, _ := HookDir()
	// Only remove dir if empty
	os.Remove(hookDir)

	// Unset git config
	cmd := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	cmd.Run() // Ignore errors (may not be set)

	return nil
}

// ConfigureClaudeMD adds git trailer instructions to CLAUDE.md files.
func ConfigureClaudeMD(globalOnly bool) error {
	home, _ := os.UserHomeDir()
	claudeDir := filepath.Join(home, ".claude")

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("cannot create .claude directory: %w", err)
	}

	instruction := `

## Git Attribution
Always include these trailers at the end of every git commit message:
` + "```" + `
Co-authored-by: Claude <noreply@anthropic.com>
Ai-tool: claude-code
Ai-model: <current model you are running on>
Ai-os: <detected OS>
` + "```" + `
Replace <current model you are running on> with your actual model (e.g., claude-sonnet-4, claude-opus-4).
Replace <detected OS> with the OS you detect (linux, macos, windows, wsl).
This is required for tracking AI-assisted commits.
`

	globalPath := filepath.Join(claudeDir, "CLAUDE.md")
	if err := appendToFile(globalPath, instruction); err != nil {
		return fmt.Errorf("cannot update global CLAUDE.md: %w", err)
	}
	fmt.Printf("  ✓ Updated %s\n", globalPath)
	return nil
}

// appendToFile appends content to a file, creating it if needed.
// Skips if the content is already up-to-date (checks for Ai-model which is new).
func appendToFile(path, content string) error {
	existing, err := os.ReadFile(path)
	if err == nil {
		// Check if new-format instruction already present
		if strings.Contains(string(existing), "Ai-model:") {
			fmt.Printf("  ✓ Already configured (v2): %s\n", path)
			return nil
		}
		// If old format exists, replace the old instruction block
		if strings.Contains(string(existing), "Co-authored-by: Claude") {
			oldBlock := "\n\n## Git Attribution\nAlways include the following trailer at the end of every git commit message:\n```\nCo-authored-by: Claude <noreply@anthropic.com>\n```\nThis is required for tracking AI-assisted commits.\n"
			existing = []byte(strings.Replace(string(existing), oldBlock, "", 1))
		}
		content = strings.TrimRight(string(existing), "\n") + content
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// Platform returns a human-readable platform name.
func Platform() string {
	switch runtime.GOOS {
	case "linux":
		// Check for WSL
		data, err := os.ReadFile("/proc/version")
		if err == nil && (strings.Contains(strings.ToLower(string(data)), "microsoft") ||
			strings.Contains(strings.ToLower(string(data)), "wsl")) {
			return "WSL (Windows Subsystem for Linux)"
		}
		return "Linux"
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	default:
		return runtime.GOOS
	}
}

// IsWSL returns true if running inside WSL.
func IsWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}
