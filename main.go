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
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lucianopf/ai-trailer/internal/config"
	"github.com/lucianopf/ai-trailer/internal/detect"
	"github.com/lucianopf/ai-trailer/internal/record"
	"github.com/lucianopf/ai-trailer/internal/webhook"
)

// Version is set at build time via -ldflags, or defaults to this constant.
var Version = "v0.3.0"

// RepoURL is the base URL for downloading binaries from the public repo.
const RepoURL = "https://raw.githubusercontent.com/lucianopf/ai-trailer/master"

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
  configure           Interactive configuration wizard (hook only)
  configure --all     Hook only, all detected tools (non-interactive)
  configure --tool X  Hook only, specific tool
  configure --model M --tool X  Set model override for tool
  configure --instructions  Also inject into CLAUDE.md, AGENTS.md, etc.
  status              Show current configuration status
  record              Query AI tool usage records and statistics
  uninstall           Remove hook only (keep instruction files)
  uninstall --full    Remove hook + revert all instruction files
  version             Show version
  update              Download and install latest binary
  update --reconfigure  Update + re-run configure

Examples:
  ai-trailer detect                    # What AI tools are installed?
  ai-trailer configure                 # Interactive menu — pick tools to configure
  ai-trailer configure --all           # Configure everything automatically
  ai-trailer status                    # See what's configured
  ai-trailer record                    # View usage statistics
  ai-trailer uninstall                 # Remove everything

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
	hasFlag := func(f string) bool {
		for _, a := range args {
			if a == f {
				return true
			}
		}
		return false
	}
	getArg := func(f string) string {
		for i, a := range args {
			if a == f && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}

	allFlag := hasFlag("--all") || hasFlag("-a")
	toolFlag := getArg("--tool")
	skipHook := hasFlag("--no-hook")
	injectInstructions := hasFlag("--instructions") || hasFlag("-i")
	modelFlag := getArg("--model")

	results := detect.DetectAll()

	// Build menu items from ALL tools (not just detected)
	// Pre-select detected ones
	var menuItems []menuItem
	detectedMap := make(map[string]bool)
	for _, r := range results {
		detectedMap[r.Tool.ID] = r.Found
		menuItems = append(menuItems, menuItem{
			ToolName: r.Tool.Name,
			Trailer:  r.Tool.Trailer,
			Selected: r.Found,            // Pre-select detected
			Detected: r.Found,
			IsNative: r.Tool.NativeTrailer,
		})
	}

	// If specific tool requested, pre-select only that one
	if toolFlag != "" {
		found := false
		for i := range menuItems {
			menuItems[i].Selected = false
		}
		for i, r := range results {
			if r.Tool.ID == toolFlag || r.Tool.Slug == toolFlag {
				menuItems[i].Selected = true
				found = true
			}
		}
		if !found {
			fmt.Printf("\n⚠  Tool '%s' not recognized. Available tools:\n", toolFlag)
			for _, r := range results {
				fmt.Printf("   - %s (%s)\n", r.Tool.ID, r.Tool.Name)
			}
			return
		}
	}

	// Count detected for summary
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

	// Show interactive TUI
	var selected []detect.DetectionResult
	if allFlag {
		// Non-interactive: select all detected
		for _, r := range results {
			if r.Found {
				selected = append(selected, r)
			}
		}
	} else if toolFlag != "" {
		// Pre-selected specific tool
		for _, r := range results {
			if r.Tool.ID == toolFlag || r.Tool.Slug == toolFlag {
				selected = append(selected, r)
			}
		}
	} else {
		// Interactive TUI menu
		menuItems = showInteractiveMenu(menuItems)
		if menuItems == nil {
			return // User cancelled
		}
		// Map selected menu items back to detection results
		for i, m := range menuItems {
			if m.Selected && i < len(results) {
				selected = append(selected, results[i])
			}
		}
	}

	if len(selected) == 0 {
		fmt.Println("\n  No tools selected. Exiting.")
		return
	}

	rec, _ := record.New()
	anyNeedsHook := false

	// Configure each selected tool
	fmt.Println("\n⚙  Applying configuration...")
	fmt.Println(strings.Repeat("─", 55))

	// Collect selected tool IDs for instruction file injection
	var selectedIDs []string
	for _, r := range selected {
		selectedIDs = append(selectedIDs, r.Tool.ID)
	}

	// ── Inject into instruction files (opt-in with --instructions) ──
	var updatedFiles []string
	if injectInstructions {
		fmt.Println("\n📄 Updating instruction files...")
		updatedFiles = config.ConfigureAllInstructionFiles(selectedIDs)
		if len(updatedFiles) > 0 {
			for _, f := range updatedFiles {
				rec.LogConfigure("instructions", f, "Injected git trailer instruction")
			}
		}
		if !config.IsInGitRepo() {
			fmt.Println("  ℹ  Not in a git repo — project-level files skipped.")
			fmt.Println("     Run 'ai-trailer configure --instructions' inside a repo to update them.")
		}
	} else {
		fmt.Println("\n  ℹ  Instruction files unchanged (use --instructions to inject).")
	}

	for _, r := range selected {
		fmt.Printf("\n📝 %s:\n", r.Tool.Name)

		if r.Tool.NativeTrailer {
			fmt.Println("  ✓ Native trailer support — instruction file updated above.")
		} else {
			anyNeedsHook = true
			fmt.Printf("  Added to hook configuration (trailer: %s)\n", r.Tool.Trailer)
			rec.LogConfigure(r.Tool.ID, r.Tool.Name, "Registered for git hook trailer insertion")
		}
	}

	// Install universal git hook (unless skipped)
	if !skipHook {
		if !config.IsHookInstalled() {
			if allFlag || anyNeedsHook || askYesNo("\n🪝  Install universal git prepare-commit-msg hook?") {
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
				fmt.Println("  ℹ  Hook skipped. AI trailer detection will NOT be active.")
			}
		} else {
			// Re-install hook to ensure it's up to date (new tools added)
			if err := config.InstallHook(); err != nil {
				fmt.Printf("  ✗ Error updating hook: %v\n", err)
			} else {
				fmt.Println("\n  ✓ Git hook updated with latest tool definitions.")
			}
		}
	} else {
		fmt.Println("\n  ℹ  Hook skipped (--no-hook flag).")
	}

	// ── Send config event to Google Sheets ──────────────────────
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

	// ── Save model override if --model provided with --tool ──
	if modelFlag != "" && len(selected) > 0 {
		modelFile := os.Getenv("HOME") + "/.ai-trailer/models"
		os.MkdirAll(os.Getenv("HOME")+"/.ai-trailer", 0755)
		// Read existing overrides
		existing, _ := os.ReadFile(modelFile)
		lines := strings.Split(string(existing), "\n")
		// Build new content: replace or append
		var newLines []string
		found := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, selected[0].Tool.ID+"=") {
				newLines = append(newLines, selected[0].Tool.ID+"="+modelFlag)
				found = true
			} else if line != "" {
				newLines = append(newLines, line)
			}
		}
		if !found {
			newLines = append(newLines, selected[0].Tool.ID+"="+modelFlag)
		}
		os.WriteFile(modelFile, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
		fmt.Printf("  ✓ Model override saved: %s=%s → %s\n", selected[0].Tool.ID, modelFlag, modelFile)
	}

	fmt.Println()
	if w := webhook.DefaultClient(); w.URL != "" {
		instructionFilesList := updatedFiles
		go w.SendConfigure(detectedNames, configuredNames, hookInstalled, instructionFilesList)
	}

	fmt.Println("\n" + strings.Repeat("═", 55))
	fmt.Println("✅ Configuration complete!")
	fmt.Println("\n  To verify, make a commit with an AI tool and check:")
	fmt.Println(`    git log -1 --format="%B"`)
	fmt.Println("\n  View usage stats:")
	fmt.Println("    ai-trailer record")
}
func askYesNo(prompt string) bool {
	fmt.Printf("%s [Y/n] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return answer == "" || answer == "y" || answer == "yes"
	}
	return true
}

func interactiveSelect(installed []detect.DetectionResult) []detect.DetectionResult {
	fmt.Println("\n  Select tools to configure (enter numbers, space-separated, or 'all'):")
	fmt.Print("  > ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" || input == "all" || strings.ToLower(input) == "a" {
		return installed
	}

	// Parse numbers
	var selected []detect.DetectionResult
	for _, part := range strings.Fields(input) {
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 && n <= len(installed) {
			selected = append(selected, installed[n-1])
		}
	}

	if len(selected) == 0 {
		fmt.Println("  No valid selection. Defaulting to all.")
		return installed
	}
	return selected
}

// ── setup command ───────────────────────────────────────────────────

func cmdSetup(args []string) {
	hasFlag := func(f string) bool {
		for _, a := range args {
			if a == f {
				return true
			}
		}
		return false
	}
	getArg := func(f string) string {
		for i, a := range args {
			if a == f && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}

	fmt.Println("\n🔧 Google Sheets Webhook Setup")
	fmt.Println(strings.Repeat("─", 55))

	// Check if already configured
	cfg, _ := webhook.LoadConfig()
	if cfg != nil && cfg.URL != "" {
		fmt.Printf("\n  Current webhook URL: %s\n", cfg.URL)
		if cfg.Token != "" {
			fmt.Println("  Auth token:         ******** (configured)")
		}
		if !hasFlag("--force") {
			fmt.Println("\n  Already configured. Use --force to reconfigure.")
			return
		}
	}

	url := getArg("--url")
	if url == "" {
		url = os.Getenv("AI_TRAILER_WEBHOOK_URL")
	}

	if url == "" && !hasFlag("--url") {
		fmt.Println("\n  No webhook URL provided.")
		fmt.Println("\n  Options:")
		fmt.Println("    1. Set env var:  export AI_TRAILER_WEBHOOK_URL=<your-url>")
		fmt.Println("    2. Use CLI flag:  ai-trailer setup --url <your-url>")
		fmt.Println("\n  Get your webhook URL from Google Apps Script:")
		fmt.Println("    Script Editor → Deploy → New Deployment → Web App → Copy URL")
		return
	}

	if url != "" {
		token := getArg("--token")
		if token == "" {
			token = os.Getenv("AI_TRAILER_WEBHOOK_TOKEN")
		}

		newCfg := webhook.Config{URL: url, Token: token}
		if err := webhook.SaveConfig(newCfg); err != nil {
			fmt.Printf("  ✗ Failed to save config: %v\n", err)
			return
		}
		fmt.Printf("\n  ✓ Webhook config saved to ~/.ai-trailer/webhook.json\n")
		fmt.Printf("    URL:   %s\n", url)
		if token != "" {
			fmt.Println("    Token: ******** (configured)")
		}

		// Test connection
		fmt.Print("\n  Testing connection... ")
		w := webhook.DefaultClient()
		if err := w.TestConnection(); err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
			fmt.Println("  Check your URL and ensure the AppScript is deployed as Web App.")
		} else {
			fmt.Println("✓ Connected!")
		}

		// Send setup event to webhook
		go w.SendSetup()
	}

	// Show current user info
	email, name, sysUser, host, plat := webhook.CollectUserInfo()
	fmt.Println("\n  User identification (sent with each event):")
	fmt.Printf("    Git email:   %s\n", email)
	fmt.Printf("    Git name:    %s\n", name)
	fmt.Printf("    System user: %s\n", sysUser)
	fmt.Printf("    Hostname:    %s\n", host)
	fmt.Printf("    Platform:    %s\n", plat)

	fmt.Println("\n  To update git identity:")
	fmt.Println("    git config --global user.email \"you@company.com\"")
	fmt.Println("    git config --global user.name \"Your Name\"")
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

	downloadURL := fmt.Sprintf("%s/dist/%s", RepoURL, binaryName)
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
	fmt.Println("✅ Update complete. New version:", Version)
	if !reconfigure {
		fmt.Println("   Run 'ai-trailer configure' to refresh hooks/files if needed.")
	}
}
