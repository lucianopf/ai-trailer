// ai-trailer — Cross-platform CLI to detect AI coding tools and configure git trailers.
//
// Commands:
//
//	detect      Detect installed AI coding tools
//	configure   Interactive configuration wizard
//	status      Show current configuration status
//	record      Query AI tool usage records
//	uninstall   Remove all configuration
//
// Usage:
//
//	ai-trailer detect
//	ai-trailer configure
//	ai-trailer configure --tool claude-code
//	ai-trailer configure --all   # skip interactive prompts
//	ai-trailer status
//	ai-trailer record
//	ai-trailer record-commit <tool> <trailer> <repo> <hash>  # internal, called by hook
//	ai-trailer uninstall
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lucianopf/ai-trailer/internal/config"
	"github.com/lucianopf/ai-trailer/internal/detect"
	"github.com/lucianopf/ai-trailer/internal/record"
	"github.com/lucianopf/ai-trailer/internal/webhook"
)

// Version is set at build time via -ldflags, or defaults to this constant.
var Version = "v0.6.11"

// RepoURL is the base URL for downloading binaries from GitHub Releases.
const RepoURL = "https://github.com/lucianopf/ai-trailer/releases/latest/download"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "detect":
		cmdDetect(args)
	case "configure", "config":
		cmdConfigure(args)
	case "status":
		cmdStatus(args)
	case "record":
		cmdRecord(args)
	case "record-commit":
		cmdRecordCommit(args)
	case "uninstall":
		cmdUninstall(args)
	case "version", "-v", "--version":
		cmdVersion(args)
	case "update":
		cmdUpdate(args)
	case "test":
		cmdTest(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ai-trailer — AI Tool Git Trailer Manager

🖥  ` + config.Platform() + `

Commands:
  detect              Detect installed AI coding tools
  configure           Interactive wizard — select tools, install git hook
  configure --instructions, -i  Also inject into CLAUDE.md, AGENTS.md, etc.
  test                Dry-run: show what trailers would be written right now
  status              Show current configuration status
  record              Query AI tool usage records and statistics
  uninstall           Remove hook only (keep instruction files)
  uninstall --full    Remove hook + revert all instruction files
  version             Show version
  update              Download and install latest binary
  update --reconfigure  Update + re-run configure

Examples:
  ai-trailer detect                    # What AI tools are installed?
  ai-trailer configure                 # Interactive menu — SPACE to select, ENTER to confirm
  ai-trailer test                      # Is the hook working in this session?
  ai-trailer status                    # See what's configured
  ai-trailer record                    # View usage statistics

Model detection (automatic, no setup for most tools):
  Claude Code → CLAUDE_MODEL env var (inherited by git subprocess)
  Hermes      → HERMES_SESSION + HERMES_MODEL env vars
  OpenCode    → OPENCODE_MODEL env var
  Gemini CLI  → GEMINI_MODEL env var
  Codex       → ~/.ai-trailer/codex-model (written by PreToolUse hook)
  Cursor      → CURSOR_TRACE_ID env var + ~/.ai-trailer/cursor-model

For more: https://github.com/lucianopf/ai-trailer`)
}

// ── detect command ───────────────────────────────────────────────────

func cmdDetect(args []string) {
	fmt.Println("\n🔍 Detecting AI coding tools...")
	fmt.Println(strings.Repeat("─", 55))

	results := detect.DetectAll()
	found := 0
	for _, r := range results {
		if r.Found {
			fmt.Printf("  ✓ %-30s %s\n", r.Tool.Name, strings.Join(r.Reasons, "; "))
			found++
		} else {
			fmt.Printf("  ✗ %-30s (not found)\n", r.Tool.Name)
		}
	}
	fmt.Println(strings.Repeat("─", 55))
	fmt.Printf("\n📊 Found %d/%d tool(s) installed.\n", found, len(results))

// Log detection (local + webhook)
	rec, _ := record.New()
	var foundNames []string
	for _, r := range results {
		if r.Found {
			foundNames = append(foundNames, r.Tool.ID)
		}
	}
	rec.LogDetect(foundNames)

	// Send config event to Google Sheets with detection info
	if w := webhook.DefaultClient(); w.URL != "" {
		go w.Send(webhook.Payload{
			Event:         "detect",
			ToolsDetected: strings.Join(foundNames, ", "),
		})
	}
}

// ── configure command ───────────────────────────────────────────────

func cmdConfigure(args []string) {
	injectInstructions := false
	for _, a := range args {
		if a == "--instructions" || a == "-i" {
			injectInstructions = true
		}
	}

	results := detect.DetectAll()

	// Build menu items from ALL tools, pre-select detected ones
	var menuItems []menuItem
	for _, r := range results {
		menuItems = append(menuItems, menuItem{
			ToolName: r.Tool.Name,
			ToolID:   r.Tool.ID,
			Trailer:  r.Tool.Trailer,
			Selected: r.Found,
			Detected: r.Found,
			IsNative: r.Tool.NativeTrailer,
		})
	}

	detectedCount := 0
	for _, r := range results {
		if r.Found {
			detectedCount++
		}
	}
	if detectedCount == 0 {
		fmt.Println("\n⚠  No AI coding tools auto-detected.")
		fmt.Println("   Showing all supported tools — select the ones you have installed.")
		fmt.Println()
	}

	menuItems = showInteractiveMenu(menuItems)
	if menuItems == nil {
		return
	}

	var selected []detect.DetectionResult
	for i, m := range menuItems {
		if m.Selected && i < len(results) {
			selected = append(selected, results[i])
		}
	}
	if len(selected) == 0 {
		fmt.Println("\n  No tools selected. Exiting.")
		return
	}

	rec, _ := record.New()

	fmt.Println("\n⚙  Applying configuration...")
	fmt.Println(strings.Repeat("─", 55))

	// ── Git hook ──
	if !config.IsHookInstalled() {
		fmt.Println("\n🪝  Installing git hook...")
		if err := config.InstallHook(); err != nil {
			fmt.Printf("  ✗ Error installing hook: %v\n", err)
		} else {
			hookDir, _ := config.HookDir()
			fmt.Printf("  ✓ Hook installed at: %s\n", hookDir)
			fmt.Println("  ✓ Git configured: core.hooksPath set globally")
			rec.LogConfigure("hook", "git-hook", "Installed prepare-commit-msg hook")
		}
	} else {
		config.InstallHook() // re-install to update
		fmt.Println("  ✓ Git hook updated.")
	}

	// ── Session file hooks for tools without guaranteed env vars ──
	fmt.Println("\n🔧 Setting up session file hooks...")
	for _, r := range selected {
		switch r.Tool.ID {
		case "codex":
			if err := installCodexHooks(); err != nil {
				fmt.Printf("  ✗ Codex: %v\n", err)
			} else {
				fmt.Printf("  ✓ Codex: PreToolUse hook → ~/.ai-trailer/codex-model\n")
				fmt.Printf("     Note: run /hooks in Codex and trust the hook to activate.\n")
			}
		case "cursor":
			if err := installCursorHooks(); err != nil {
				fmt.Printf("  ✗ Cursor: %v\n", err)
			}
		default:
			// Other tools detected via env vars — no extra setup needed
		}
	}

	// ── Instruction files (opt-in) ──
	var updatedFiles []string
	if injectInstructions {
		fmt.Println("\n📄 Updating instruction files...")
		var selectedIDs []string
		for _, r := range selected {
			selectedIDs = append(selectedIDs, r.Tool.ID)
		}
		updatedFiles = config.ConfigureAllInstructionFiles(selectedIDs)
		for _, f := range updatedFiles {
			fmt.Printf("  ✓ Injected into %s\n", f)
		}
		if !config.IsInGitRepo() {
			fmt.Println("  ℹ  Not in a git repo — project-level files skipped.")
		}
	} else {
		fmt.Println("\n  ℹ  Instruction files unchanged (use --instructions to inject).")
	}

	// ── Webhook ──
	hookInstalled := config.IsHookInstalled()
	var detectedNames, configuredNames []string
	for _, r := range results {
		if r.Found {
			detectedNames = append(detectedNames, r.Tool.ID)
		}
	}
	for _, r := range selected {
		configuredNames = append(configuredNames, r.Tool.ID)
	}
	if w := webhook.DefaultClient(); w.URL != "" {
		go w.SendConfigure(detectedNames, configuredNames, hookInstalled, updatedFiles)
	}

	fmt.Println("\n" + strings.Repeat("═", 55))
	fmt.Println("✅ Configuration complete!")
	fmt.Println("\n  Verify with: git log -1 --format=\"%B\"")
	fmt.Println("  View stats:  ai-trailer record")
}

// ── status command ──────────────────────────────────────────────────

func cmdStatus(args []string) {
	fmt.Println("\n📊 AI Trailer Status")
	fmt.Println(strings.Repeat("─", 55))

	// Platform
	fmt.Printf("  Platform:     %s\n", config.Platform())

	// Git hook
	if config.IsHookInstalled() {
		hookDir, _ := config.HookDir()
		fmt.Printf("  Git hook:     ✓ Installed (%s)\n", hookDir)
	} else {
		fmt.Println("  Git hook:     ✗ Not installed")
	}

	// Installed tools
	results := detect.DetectAll()
	fmt.Println("\n  Detected tools:")
	for _, r := range results {
		if r.Found {
			fmt.Printf("    ✓ %s\n", r.Tool.Name)
		}
	}

	// Records
	rec, _ := record.New()
	summary, _ := rec.FormatSummary(5)
	fmt.Println("\n" + summary)
}

// ── record command ──────────────────────────────────────────────────

func cmdRecord(args []string) {
	rec, err := record.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	summary, err := rec.FormatSummary(10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println(summary)
}

// cmdRecordCommit is called by the git hook to log AI-assisted commits.
// Only logs locally — commit tracking via GitHub API separately.
func cmdRecordCommit(args []string) {
	if len(args) < 4 {
		return // Silently ignore — don't block commits
	}
	toolID := args[0]
	trailer := args[1]
	repo := args[2]
	commitHash := args[3]

	// Local record only
	rec, err := record.New()
	if err == nil {
		rec.LogCommit(toolID, trailer, repo, commitHash)
	}
}

// ── uninstall command ───────────────────────────────────────────────

func cmdUninstall(args []string) {
	fullFlag := false
	for _, a := range args {
		if a == "--full" || a == "-f" {
			fullFlag = true
		}
	}

	fmt.Println("\n🗑  Uninstalling AI Trailer configuration...")
	fmt.Println(strings.Repeat("─", 55))

	if err := config.UninstallHook(); err != nil {
		fmt.Printf("  ✗ Error removing hook: %v\n", err)
	} else {
		fmt.Println("  ✓ Git hook removed")
		fmt.Println("  ✓ Git config unset")
	}

	// ── Full uninstall: also revert instruction files ──
	if fullFlag {
		fmt.Println("\n📄 Reverting instruction files...")
		results := detect.DetectAll()
		var allIDs []string
		for _, r := range results {
			allIDs = append(allIDs, r.Tool.ID)
		}
		reverted := config.RevertInstructionFiles(allIDs)
		if len(reverted) > 0 {
			for _, f := range reverted {
				fmt.Printf("  ✓ Reverted %s\n", f)
			}
		}
	} else {
		home, _ := os.UserHomeDir()
		claudeMD := home + "/.claude/CLAUDE.md"
		if _, err := os.Stat(claudeMD); err == nil {
			fmt.Printf("  ℹ  %s was NOT modified (use --full to revert all instruction files)\n", claudeMD)
		} else {
			fmt.Println("  ℹ  Instruction files unchanged. Use --full to revert them.")
		}
	}

	rec, _ := record.New()
	rec.Log(record.Entry{
		Event:  "uninstall",
		Action: "Removed all AI trailer configuration",
	})

	// Send uninstall event to webhook
	hookWasInstalled := true
	if w := webhook.DefaultClient(); w.URL != "" {
		var detected []string
		for _, r := range detect.DetectAll() {
			if r.Found {
				detected = append(detected, r.Tool.ID)
			}
		}
		extra := ""
		if fullFlag {
			extra = "Full uninstall: instruction files reverted"
		}
		go w.Send(webhook.Payload{
			Event:         "uninstall",
			ToolsDetected: strings.Join(detected, ", "),
			HookInstalled: hookWasInstalled,
			Extra:         extra,
		})
	}

	fmt.Println(strings.Repeat("─", 55))
	fmt.Println("✅ Uninstall complete.")
	fmt.Println("   Record log preserved at ~/.ai-trailer/records.jsonl")
}

// ── version command ─────────────────────────────────────────────────

func cmdVersion(args []string) {
	fmt.Printf("ai-trailer %s\n", Version)
	fmt.Printf("  Platform: %s\n", config.Platform())
	fmt.Printf("  Repo:     https://github.com/lucianopf/ai-trailer\n")
}

// ── update command ──────────────────────────────────────────────────

func cmdUpdate(args []string) {
	reconfigure := false
	for _, a := range args {
		if a == "--reconfigure" || a == "-r" {
			reconfigure = true
		}
	}

	fmt.Println("\n🔄 Updating ai-trailer...")
	fmt.Println(strings.Repeat("─", 55))

	// Determine binary name
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64" // keep
	}
	binaryName := fmt.Sprintf("ai-trailer-%s-%s", osName, arch)
	if osName == "windows" {
		binaryName += ".exe"
	}

	downloadURL := fmt.Sprintf("%s/%s", RepoURL, binaryName)
	fmt.Printf("  Downloading: %s\n", downloadURL)

	// Download to temp file
	tmpDir, err := os.MkdirTemp("", "ai-trailer-update")
	if err != nil {
		fmt.Printf("  ✗ Cannot create temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, binaryName)
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Printf("  ✗ Download failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("  ✗ Download failed: HTTP %d\n", resp.StatusCode)
		return
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		fmt.Printf("  ✗ Cannot create temp file: %v\n", err)
		return
	}

	written, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		fmt.Printf("  ✗ Download failed: %v\n", err)
		return
	}
	fmt.Printf("  ✓ Downloaded %d bytes\n", written)

	// macOS: strip quarantine
	if runtime.GOOS == "darwin" {
		// Use xattr if available
		fmt.Println("  Stripping quarantine attribute...")
		exec.Command("xattr", "-d", "com.apple.quarantine", tmpPath).Run()
		exec.Command("xattr", "-cr", tmpPath).Run()
	}

	// Make executable
	os.Chmod(tmpPath, 0755)

	// Find current binary location
	currentPath, err := os.Executable()
	if err != nil {
		fmt.Printf("  ✗ Cannot find current binary: %v\n", err)
		return
	}
	// Resolve symlinks
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		fmt.Printf("  ✗ Cannot resolve binary path: %v\n", err)
		return
	}

	fmt.Printf("  Current binary: %s\n", currentPath)

	// Replace: rename current to .old, copy new in place
	backupPath := currentPath + ".old"
	if err := os.Rename(currentPath, backupPath); err != nil {
		fmt.Printf("  ✗ Cannot backup current binary: %v\n", err)
		fmt.Println("  Try running with sudo or check permissions.")
		return
	}

	// Copy new binary
	src, _ := os.Open(tmpPath)
	dst, err := os.Create(currentPath)
	if err != nil {
		// Restore backup
		os.Rename(backupPath, currentPath)
		fmt.Printf("  ✗ Cannot write new binary: %v\n", err)
		return
	}
	io.Copy(dst, src)
	src.Close()
	dst.Close()
	os.Chmod(currentPath, 0755)

	// Remove backup (best effort)
	os.Remove(backupPath)

	fmt.Println("  ✓ Binary updated")

	// Reconfigure if requested
	if reconfigure {
		fmt.Println("\n  Re-running configuration...")
		// Re-run configure --all
		results := detect.DetectAll()
		var allIDs []string
		for _, r := range results {
			if r.Found {
				allIDs = append(allIDs, r.Tool.ID)
			}
		}
		config.ConfigureAllInstructionFiles(allIDs)
		config.InstallHook()
	}

	fmt.Println(strings.Repeat("─", 55))
	// Ask the newly installed binary for its own version string.
	newVersion := Version
	if out, err := exec.Command(currentPath, "version").Output(); err == nil {
		if fields := strings.Fields(string(out)); len(fields) >= 2 {
			newVersion = fields[1]
		}
	}
	fmt.Println("✅ Update complete. New version:", newVersion)
	if !reconfigure {
		fmt.Println("   Run 'ai-trailer configure' to refresh hooks/files if needed.")
	}
}

// ── test command ────────────────────────────────────────────────────

func cmdTest(_ []string) {
	fmt.Println("\n🧪 ai-trailer test — dry run (current env)")
	fmt.Println(strings.Repeat("─", 55))

	type detection struct {
		tool    string
		model   string
		trigger string
	}

	home, _ := os.UserHomeDir()
	var det *detection

	if m := os.Getenv("CLAUDE_MODEL"); m != "" {
		det = &detection{tool: "claude-code", model: m, trigger: "CLAUDE_MODEL=" + m}
	} else if s := os.Getenv("HERMES_SESSION"); s != "" {
		m := os.Getenv("HERMES_MODEL")
		det = &detection{tool: "hermes", model: m, trigger: "HERMES_SESSION=" + s}
	} else if m := os.Getenv("OPENCODE_MODEL"); m != "" {
		det = &detection{tool: "opencode", model: m, trigger: "OPENCODE_MODEL=" + m}
	} else if m := os.Getenv("GEMINI_MODEL"); m != "" {
		det = &detection{tool: "gemini-cli", model: m, trigger: "GEMINI_MODEL=" + m}
	} else if t := os.Getenv("CURSOR_TRACE_ID"); t != "" {
		sf := filepath.Join(home, ".ai-trailer", "cursor-model")
		m := ""
		if data, err := os.ReadFile(sf); err == nil {
			m = strings.TrimSpace(string(data))
		}
		det = &detection{tool: "cursor", model: m, trigger: "CURSOR_TRACE_ID=" + t}
	} else {
		sf := filepath.Join(home, ".ai-trailer", "codex-model")
		if fi, err := os.Stat(sf); err == nil && time.Since(fi.ModTime()) < time.Hour {
			data, _ := os.ReadFile(sf)
			m := strings.TrimSpace(string(data))
			det = &detection{tool: "codex", model: m, trigger: "~/.ai-trailer/codex-model (recent)"}
		}
	}

	if det == nil {
		fmt.Println("  No AI tool detected in current environment.")
		fmt.Println("  Run this command inside an active AI tool session.")
		return
	}

	trailerMap := map[string]string{
		"claude-code": "Co-authored-by: Claude <noreply@anthropic.com>",
		"hermes":      "Co-authored-by: Hermes Agent <noreply@nousresearch.com>",
		"opencode":    "Co-authored-by: OpenCode <noreply@opencode.ai>",
		"gemini-cli":  "Co-authored-by: Gemini <noreply@google.com>",
		"cursor":      "Co-authored-by: Cursor <noreply@cursor.sh>",
		"codex":       "Co-authored-by: OpenAI Codex <noreply@openai.com>",
	}

	fmt.Printf("  Env detected:  %s\n", det.trigger)
	fmt.Println("\n  Would append:")
	fmt.Printf("    %s\n", trailerMap[det.tool])
	fmt.Printf("    Ai-tool: %s\n", det.tool)
	if det.model != "" {
		fmt.Printf("    Ai-model: %s\n", det.model)
	}
	aiOs := runtime.GOOS
	if aiOs == "darwin" {
		aiOs = "macos"
	}
	fmt.Printf("    Ai-os: %s\n", aiOs)
}
