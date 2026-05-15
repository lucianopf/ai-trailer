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

// ── Instruction File Injection ──────────────────────────────────────

// instructionConfig maps a tool to its instruction file paths and trailer.
type instructionConfig struct {
	toolID      string   // internal tool ID (matches detect.Tool.ID)
	toolName    string   // display name for the instruction header
	trailer     string   // full Co-authored-by line
	aiToolTag   string   // Ai-tool value
	modelHint   string   // example models for the instruction text
	globalPaths []string // paths relative to home dir (~ expanded)
	repoPaths   []string // paths relative to git repo root
}

// instructionConfigs maps every tool that can commit on its own
// to the instruction files it reads.
var instructionConfigs = []instructionConfig{
	{
		toolID: "claude-code", toolName: "Claude",
		trailer:   "Co-authored-by: Claude <noreply@anthropic.com>",
		aiToolTag: "claude-code", modelHint: "claude-sonnet-4, claude-opus-4",
		globalPaths: []string{".claude/CLAUDE.md"},
		repoPaths:   []string{"CLAUDE.md"},
	},
	{
		toolID: "codex", toolName: "OpenAI Codex",
		trailer:   "Co-authored-by: OpenAI Codex <noreply@openai.com>",
		aiToolTag: "codex", modelHint: "gpt-5, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{"AGENTS.md"},
	},
	{
		toolID: "opencode", toolName: "OpenCode",
		trailer:   "Co-authored-by: OpenCode <noreply@opencode.ai>",
		aiToolTag: "opencode", modelHint: "claude-sonnet-4, gpt-5",
		globalPaths: []string{},
		repoPaths:   []string{"AGENTS.md"},
	},
	{
		toolID: "gemini-cli", toolName: "Gemini",
		trailer:   "Co-authored-by: Gemini <noreply@google.com>",
		aiToolTag: "gemini-cli", modelHint: "gemini-2.5-pro, gemini-2.5-flash",
		globalPaths: []string{".gemini/GEMINI.md"},
		repoPaths:   []string{"GEMINI.md"},
	},
	{
		toolID: "cursor", toolName: "Cursor",
		trailer:   "Co-authored-by: Cursor <noreply@cursor.sh>",
		aiToolTag: "cursor", modelHint: "claude-sonnet-4, gpt-5",
		globalPaths: []string{},
		repoPaths:   []string{".cursorrules"},
	},
	{
		toolID: "github-copilot", toolName: "GitHub Copilot",
		trailer:   "Co-authored-by: GitHub Copilot <noreply@github.com>",
		aiToolTag: "github-copilot", modelHint: "gpt-5, claude-sonnet-4",
		globalPaths: []string{},
		repoPaths:   []string{".github/copilot-instructions.md"},
	},
	{
		toolID: "aider", toolName: "Aider",
		trailer:   "Co-authored-by: Aider <noreply@aider.chat>",
		aiToolTag: "aider", modelHint: "claude-sonnet-4, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{"CONVENTION.md"},
	},
	{
		toolID: "windsurf", toolName: "Windsurf",
		trailer:   "Co-authored-by: Windsurf <noreply@codeium.com>",
		aiToolTag: "windsurf", modelHint: "claude-sonnet-4, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{".windsurfrules"},
	},
	{
		toolID: "cody", toolName: "Cody",
		trailer:   "Co-authored-by: Cody <noreply@sourcegraph.com>",
		aiToolTag: "cody", modelHint: "claude-sonnet-4, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{".codyrules"},
	},
	{
		toolID: "kilocode", toolName: "KiloCode",
		trailer:   "Co-authored-by: KiloCode <noreply@kilocode.ai>",
		aiToolTag: "kilocode", modelHint: "claude-sonnet-4, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{"CLAUDE.md"},
	},
	{
		toolID: "amazon-q", toolName: "Amazon Q",
		trailer:   "Co-authored-by: Amazon Q <noreply@aws.amazon.com>",
		aiToolTag: "amazon-q", modelHint: "claude-sonnet-4, gpt-4o",
		globalPaths: []string{},
		repoPaths:   []string{".amazonq/rules/ai.md"},
	},
}

// generateInstruction builds the markdown instruction block for a tool.
func generateInstruction(cfg instructionConfig) string {
	return fmt.Sprintf(`

## Git Attribution — %s
Always include these trailers at the end of every git commit message:
`+"```"+`
%s
Ai-tool: %s
Ai-model: <current model you are running on>
Ai-os: <detected OS>
`+"```"+`
Replace <current model you are running on> with your actual model (e.g., %s).
Replace <detected OS> with the OS you detect (linux, macos, windows, wsl).
This is required for tracking AI-assisted commits.
`, cfg.toolName, cfg.trailer, cfg.aiToolTag, cfg.modelHint)
}

// injectInstructionToFile appends the instruction block to a file if
// the tool's trailer is not already present.
// Returns true if the file was created or updated.
func injectInstructionToFile(path string, cfg instructionConfig) bool {
	instruction := generateInstruction(cfg)

	existing, err := os.ReadFile(path)
	if err == nil {
		// Already present? Skip.
		if strings.Contains(string(existing), cfg.trailer) {
			// Check if old format without Ai-model — upgrade it
			if !strings.Contains(string(existing), "Ai-model:") {
				// Remove old block and append new instruction
				// Find the section containing the trailer and remove it
				oldStart := strings.Index(string(existing), "## Git Attribution")
				if oldStart >= 0 {
					oldEnd := strings.Index(string(existing)[oldStart:], "```\nThis is required")
					if oldEnd >= 0 {
						oldEnd = oldStart + oldEnd + len("```\nThis is required for tracking AI-assisted commits.\n")
						upgraded := string(existing)[:oldStart] + strings.TrimRight(string(existing)[oldEnd:], "\n") + instruction
						os.WriteFile(path, []byte(upgraded), 0644)
						fmt.Printf("  ✓ Upgraded %s (added Ai-model/Ai-os/Ai-tool trailers)\n", path)
						return true
					}
				}
			}
			// Already up to date
			fmt.Printf("  ✓ Already configured: %s\n", path)
			return false
		}
		// Append to existing file
		updated := strings.TrimRight(string(existing), "\n") + instruction
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			fmt.Printf("  ✗ Error updating %s: %v\n", path, err)
			return false
		}
		fmt.Printf("  ✓ Updated %s\n", path)
		return true
	}

	// File doesn't exist — create it
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fmt.Printf("  ✗ Cannot create directory for %s: %v\n", path, err)
		return false
	}
	content := strings.TrimLeft(instruction, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("  ✗ Error creating %s: %v\n", path, err)
		return false
	}
	fmt.Printf("  ✓ Created %s\n", path)
	return true
}

// IsInGitRepo returns true if the current directory is inside a git repo.
func IsInGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GitRepoRoot returns the absolute path to the git repo root.
func GitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ConfigureAllInstructionFiles injects git trailer instructions into
// all relevant instruction files for the given tool IDs.
// Returns the list of files that were updated.
func ConfigureAllInstructionFiles(toolIDs []string) (updatedFiles []string) {
	home, _ := os.UserHomeDir()

	// Build a set of selected tool IDs for quick lookup
	selected := make(map[string]bool)
	for _, id := range toolIDs {
		selected[id] = true
	}

	// Check if we're in a git repo for project-level files
	inRepo := IsInGitRepo()
	var repoRoot string
	if inRepo {
		repoRoot, _ = GitRepoRoot()
	}

	for _, cfg := range instructionConfigs {
		if !selected[cfg.toolID] {
			continue
		}

		// ── Global paths (always injected) ──
		for _, p := range cfg.globalPaths {
			fullPath := filepath.Join(home, p)
			if injectInstructionToFile(fullPath, cfg) {
				updatedFiles = append(updatedFiles, fullPath)
			}
		}

		// ── Repo paths (injected if in a git repo and file exists or can be created) ──
		if inRepo && repoRoot != "" {
			for _, p := range cfg.repoPaths {
				fullPath := filepath.Join(repoRoot, p)
				if injectInstructionToFile(fullPath, cfg) {
					updatedFiles = append(updatedFiles, fullPath)
				}
			}
		}
	}

	return updatedFiles
}

// revertInstructionFromFile removes the ## Git Attribution block for a
// specific tool from a file. Returns true if the file was modified.
func revertInstructionFromFile(path string, cfg instructionConfig) bool {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false // file doesn't exist, nothing to revert
	}

	content := string(existing)

	// Find the marker for this tool's block
	marker := "## Git Attribution — " + cfg.toolName
	start := strings.Index(content, marker)
	if start < 0 {
		return false // block not found
	}

	// Find the end of the block: "This is required for tracking AI-assisted commits.\n"
	endMarker := "This is required for tracking AI-assisted commits.\n"
	end := strings.Index(content[start:], endMarker)
	if end < 0 {
		// Try without trailing newline
		endMarker = "This is required for tracking AI-assisted commits."
		end = strings.Index(content[start:], endMarker)
		if end < 0 {
			fmt.Printf("  ⚠  Found start marker but not end marker in %s — skipping\n", path)
			return false
		}
	}
	end = start + end + len(endMarker)

	// Remove leading newlines before the block to avoid leaving gaps
	// Walk back from start to find preceding newlines
	trimStart := start
	for trimStart > 0 && (content[trimStart-1] == '\n' || content[trimStart-1] == '\r') {
		trimStart--
	}
	// Leave exactly one trailing newline after removal
	cleaned := content[:trimStart] + content[end:]

	if err := os.WriteFile(path, []byte(cleaned), 0644); err != nil {
		fmt.Printf("  ✗ Error reverting %s: %v\n", path, err)
		return false
	}
	fmt.Printf("  ✓ Reverted %s (removed %s block)\n", path, cfg.toolName)
	return true
}

// RevertInstructionFiles removes all injected ## Git Attribution blocks
// from all instruction files (global + repo-level) for the given tools.
// Returns the list of files that were modified.
func RevertInstructionFiles(toolIDs []string) (revertedFiles []string) {
	home, _ := os.UserHomeDir()

	selected := make(map[string]bool)
	for _, id := range toolIDs {
		selected[id] = true
	}

	inRepo := IsInGitRepo()
	var repoRoot string
	if inRepo {
		repoRoot, _ = GitRepoRoot()
	}

	for _, cfg := range instructionConfigs {
		if !selected[cfg.toolID] {
			continue
		}

		// ── Global paths ──
		for _, p := range cfg.globalPaths {
			fullPath := filepath.Join(home, p)
			if revertInstructionFromFile(fullPath, cfg) {
				revertedFiles = append(revertedFiles, fullPath)
			}
		}

		// ── Repo paths ──
		if inRepo && repoRoot != "" {
			for _, p := range cfg.repoPaths {
				fullPath := filepath.Join(repoRoot, p)
				if revertInstructionFromFile(fullPath, cfg) {
					revertedFiles = append(revertedFiles, fullPath)
				}
			}
		}
	}

	return revertedFiles
}

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
