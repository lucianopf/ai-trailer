// Package detect provides AI coding tool detection across platforms.
package detect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tool represents an AI coding tool with its metadata and detection methods.
type Tool struct {
	ID            string   // Unique identifier (e.g., "claude-code")
	Name          string   // Human-readable name
	Slug          string   // Short slug for CLI
	Trailer       string   // Co-authored-by trailer string
	NativeTrailer bool     // Whether tool natively adds trailers
	Binaries      []string // Binary names to search in PATH
	NpmPackages   []string // npm global packages
	PipPackages   []string // pip packages
	BrewPackages  []string // Homebrew formula names
	ConfigPaths   []string // Config file paths (with ~/ expanded)
	EnvMarkers    []string // Environment variables that indicate tool is running
	ProcNames     []string // Process names for parent-process detection
}

// Tools is the registry of all known AI coding tools.
var Tools = []Tool{
	// ── Major CLI Agents ──────────────────────────────────────────
	{
		ID:            "claude-code",
		Name:          "Claude Code (Anthropic)",
		Slug:          "claude",
		Trailer:       "Co-authored-by: Claude <noreply@anthropic.com>",
		NativeTrailer: true,
		Binaries:      []string{"claude"},
		NpmPackages:   []string{"@anthropic-ai/claude-code"},
		ConfigPaths:   []string{"~/.claude/settings.json"},
		EnvMarkers:    []string{"CLAUDE_CODE_SIMPLE", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY"},
		ProcNames:     []string{"claude", "claude-code"},
	},
	{
		ID:            "gemini-cli",
		Name:          "Gemini CLI (Google)",
		Slug:          "gemini",
		Trailer:       "Co-authored-by: Gemini <noreply@google.com>",
		NativeTrailer: false,
		Binaries:      []string{"gemini"},
		NpmPackages:   []string{"@google/gemini-cli", "@google-gemini/cli"},
		PipPackages:   []string{"google-genai"},
		ConfigPaths:   []string{"~/.config/gemini", "~/.gemini"},
		EnvMarkers:    []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"},
		ProcNames:     []string{"gemini", "gemini-cli"},
	},
	{
		ID:            "codex",
		Name:          "OpenAI Codex CLI",
		Slug:          "codex",
		Trailer:       "Co-authored-by: OpenAI Codex <noreply@openai.com>",
		NativeTrailer: false,
		Binaries:      []string{"codex"},
		NpmPackages:   []string{"@openai/codex"},
		ConfigPaths:   []string{"~/.codex/config.toml"},
		EnvMarkers:    []string{"OPENAI_API_KEY"},
		ProcNames:     []string{"codex", "codex-cli"},
	},
	{
		ID:            "github-copilot",
		Name:          "GitHub Copilot",
		Slug:          "copilot",
		Trailer:       "Co-authored-by: GitHub Copilot <noreply@github.com>",
		NativeTrailer: false,
		Binaries:      []string{"copilot", "gh-copilot"},
		NpmPackages:   []string{"@github/copilot-cli"},
		ConfigPaths:   []string{"~/.config/github-copilot"},
		EnvMarkers:    []string{"COPILOT_GITHUB_TOKEN", "GITHUB_COPILOT", "COPILOT_CLI_PATH"},
		ProcNames:     []string{"copilot", "github-copilot", "gh-copilot"},
	},
	{
		ID:            "hermes",
		Name:          "Hermes Agent (Nous Research)",
		Slug:          "hermes",
		Trailer:       "Co-authored-by: Hermes Agent <noreply@nousresearch.com>",
		NativeTrailer: false,
		Binaries:      []string{"hermes"},
		PipPackages:   []string{"hermes-agent"},
		ConfigPaths:   []string{"~/.hermes/config.yaml"},
		EnvMarkers:    []string{"HERMES_HOME", "HERMES_SESSION_ID", "_HERMES_GATEWAY"},
		ProcNames:     []string{"hermes"},
	},
	{
		ID:            "opencode",
		Name:          "OpenCode",
		Slug:          "opencode",
		Trailer:       "Co-authored-by: OpenCode <noreply@opencode.ai>",
		NativeTrailer: false,
		Binaries:      []string{"opencode"},
		NpmPackages:   []string{"opencode-ai"},
		BrewPackages:  []string{"opencode"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"opencode"},
	},
	{
		ID:            "kilocode",
		Name:          "KiloCode",
		Slug:          "kilocode",
		Trailer:       "Co-authored-by: KiloCode <noreply@kilocode.ai>",
		NativeTrailer: false,
		Binaries:      []string{"kilocode"},
		NpmPackages:   []string{"@kilocode/cli"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"kilocode"},
	},

	// ── Open-Source AI Coding Assistants ──────────────────────────
	{
		ID:            "aider",
		Name:          "Aider (AI Pair Programming)",
		Slug:          "aider",
		Trailer:       "Co-authored-by: Aider <noreply@aider.chat>",
		NativeTrailer: false,
		Binaries:      []string{"aider"},
		PipPackages:   []string{"aider-chat"},
		BrewPackages:  []string{"aider"},
		ConfigPaths:   []string{"~/.aider.conf.yml", "~/.aider.conf"},
		EnvMarkers:    []string{"AIDER_"},
		ProcNames:     []string{"aider", "aider-chat"},
	},
	{
		ID:            "continue",
		Name:          "Continue.dev",
		Slug:          "continue",
		Trailer:       "Co-authored-by: Continue <noreply@continue.dev>",
		NativeTrailer: false,
		Binaries:      []string{"continue"},
		NpmPackages:   []string{"@continuedev/continue"},
		ConfigPaths:   []string{"~/.continue/config.json", "~/.continue/config.ts"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"continue"},
	},
	{
		ID:            "cody",
		Name:          "Cody (Sourcegraph)",
		Slug:          "cody",
		Trailer:       "Co-authored-by: Cody <noreply@sourcegraph.com>",
		NativeTrailer: false,
		Binaries:      []string{"cody"},
		NpmPackages:   []string{"@sourcegraph/cody"},
		ConfigPaths:   []string{"~/.cody"},
		EnvMarkers:    []string{"SRC_ACCESS_TOKEN"},
		ProcNames:     []string{"cody", "cody-agent"},
	},

	// ── IDEs & Extensions ─────────────────────────────────────────
	{
		ID:            "cursor",
		Name:          "Cursor IDE",
		Slug:          "cursor",
		Trailer:       "Co-authored-by: Cursor <noreply@cursor.sh>",
		NativeTrailer: false,
		Binaries:      []string{"cursor"},
		ConfigPaths:   []string{"~/.cursor", "~/.config/Cursor"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"cursor", "Cursor"},
	},
	{
		ID:            "windsurf",
		Name:          "Windsurf IDE (Codeium)",
		Slug:          "windsurf",
		Trailer:       "Co-authored-by: Windsurf <noreply@codeium.com>",
		NativeTrailer: false,
		Binaries:      []string{"windsurf"},
		ConfigPaths:   []string{"~/.windsurf", "~/.codeium"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"windsurf", "codeium"},
	},

	// ── Other Cloud Tools ─────────────────────────────────────────
	{
		ID:            "amazon-q",
		Name:          "Amazon Q Developer",
		Slug:          "q",
		Trailer:       "Co-authored-by: Amazon Q <noreply@aws.amazon.com>",
		NativeTrailer: false,
		Binaries:      []string{"q"},
		NpmPackages:   []string{"@amazon/q"},
		ConfigPaths:   []string{"~/.aws/q"},
		EnvMarkers:    []string{"AWS_PROFILE"},
		ProcNames:     []string{"q", "amazon-q"},
	},
	{
		ID:            "tabnine",
		Name:          "Tabnine",
		Slug:          "tabnine",
		Trailer:       "Co-authored-by: Tabnine <noreply@tabnine.com>",
		NativeTrailer: false,
		Binaries:      []string{"tabnine"},
		NpmPackages:   []string{"tabnine-cli"},
		ConfigPaths:   []string{"~/.tabnine"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"tabnine"},
	},
	{
		ID:            "coderabbit",
		Name:          "CodeRabbit",
		Slug:          "coderabbit",
		Trailer:       "Co-authored-by: CodeRabbit <noreply@coderabbit.ai>",
		NativeTrailer: false,
		Binaries:      []string{"coderabbit"},
		NpmPackages:   []string{"coderabbit-cli"},
		ConfigPaths:   []string{"~/.coderabbit"},
		EnvMarkers:    []string{},
		ProcNames:     []string{"coderabbit"},
	},
}

// DetectionResult holds the result of detecting a single tool.
type DetectionResult struct {
	Tool    Tool
	Found   bool
	Reasons []string // Human-readable reasons for detection
}

// ExpandPath resolves ~ and environment variables in a path.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	return os.ExpandEnv(path)
}

// Which finds an executable in PATH. Cross-platform version of `which`.
func Which(program string) (string, bool) {
	// Check with common extensions on Windows
	exts := []string{""}
	if os.PathSeparator == '\\' {
		exts = append(exts, ".exe", ".cmd", ".bat", ".ps1")
	}
	for _, ext := range exts {
		p, err := exec.LookPath(program + ext)
		if err == nil {
			return p, true
		}
	}
	return "", false
}

// checkNpmPackage checks if an npm package is installed globally.
func checkNpmPackage(pkg string) bool {
	cmd := exec.Command("npm", "list", "-g", "--depth=0")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), pkg)
}

// checkPipPackage checks if a pip package is installed.
func checkPipPackage(pkg string) bool {
	for _, pip := range []string{"pip3", "pip"} {
		cmd := exec.Command(pip, "show", pkg)
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

// checkBrewPackage checks if a Homebrew formula is installed.
func checkBrewPackage(pkg string) bool {
	cmd := exec.Command("brew", "list", "--formula", pkg)
	return cmd.Run() == nil
}

// checkVSCodeExtension checks common VS Code extension directories.
func checkVSCodeExtension(extID string) bool {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".vscode", "extensions"),
		filepath.Join(home, ".vscode-server", "extensions"),
		filepath.Join(home, ".cursor", "extensions"),
	}
	if os.PathSeparator == '\\' {
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			dirs = append(dirs, filepath.Join(appdata, "Code", "User")) // falls through to files
		}
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(extID)) {
				return true
			}
		}
	}
	return false
}

// DetectTool checks if a specific tool is installed on this machine.
func DetectTool(tool Tool) DetectionResult {
	result := DetectionResult{Tool: tool}

	// 1. Check binaries in PATH
	for _, binary := range tool.Binaries {
		if path, ok := Which(binary); ok {
			result.Found = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("binary '%s' at %s", binary, path))
			break
		}
	}

	// 2. Check npm global packages
	if !result.Found {
		for _, pkg := range tool.NpmPackages {
			if checkNpmPackage(pkg) {
				result.Found = true
				result.Reasons = append(result.Reasons, fmt.Sprintf("npm: %s", pkg))
				break
			}
		}
	}

	// 3. Check pip packages
	if !result.Found {
		for _, pkg := range tool.PipPackages {
			if checkPipPackage(pkg) {
				result.Found = true
				result.Reasons = append(result.Reasons, fmt.Sprintf("pip: %s", pkg))
				break
			}
		}
	}

	// 4. Check brew packages
	if !result.Found {
		for _, pkg := range tool.BrewPackages {
			if checkBrewPackage(pkg) {
				result.Found = true
				result.Reasons = append(result.Reasons, fmt.Sprintf("brew: %s", pkg))
				break
			}
		}
	}

	// 5. Check config paths
	if !result.Found {
		for _, path := range tool.ConfigPaths {
			if _, err := os.Stat(ExpandPath(path)); err == nil {
				result.Found = true
				result.Reasons = append(result.Reasons, fmt.Sprintf("config at %s", path))
				break
			}
		}
	}

	// 6. Check VS Code extension dirs (for copilot, continue, cody, tabnine)
	if !result.Found {
		switch tool.ID {
		case "github-copilot":
			if checkVSCodeExtension("github.copilot") {
				result.Found = true
				result.Reasons = append(result.Reasons, "VS Code extension: github.copilot")
			}
		case "continue":
			if checkVSCodeExtension("continue.continue") {
				result.Found = true
				result.Reasons = append(result.Reasons, "VS Code extension: continue.continue")
			}
		case "cody":
			if checkVSCodeExtension("sourcegraph.cody") {
				result.Found = true
				result.Reasons = append(result.Reasons, "VS Code extension: sourcegraph.cody")
			}
		case "tabnine":
			if checkVSCodeExtension("tabnine.tabnine") {
				result.Found = true
				result.Reasons = append(result.Reasons, "VS Code extension: tabnine.tabnine")
			}
		}
	}

	return result
}

// DetectAll detects all known AI coding tools.
func DetectAll() []DetectionResult {
	var results []DetectionResult
	for _, tool := range Tools {
		results = append(results, DetectTool(tool))
	}
	return results
}
