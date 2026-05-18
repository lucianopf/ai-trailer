# ai-trailer Hook Redesign: Env Var Detection — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the multi-layer prepare-commit-msg hook (process tree walk + per-tool /tmp files + 3-layer model detection) with a simple env-var-first design: env vars for confirmed tools, session files in `~/.ai-trailer/` for Codex/Cursor.

**Architecture:** The hook checks env vars in priority order (CLAUDE_MODEL → HERMES_SESSION → OPENCODE_MODEL → GEMINI_MODEL → CURSOR_TRACE_ID → codex session file). For Codex, the session file must be newer than 60 minutes (written by a PreToolUse hook installed by `ai-trailer configure`). No process tree walking. `ai-trailer configure` becomes simpler: installs the global hook + Codex PreToolUse hook. New `ai-trailer test` command dry-runs detection in the current env.

**Tech Stack:** Go 1.22+, bash (hook script embedded in Go), `go test ./...`

**Spec:** `docs/2026-05-18-env-var-hook-redesign.md`

---

## File Map

| File | Action | What changes |
|------|--------|--------------|
| `internal/config/hook_script.go` | Rewrite | 270 lines → ~80 lines. Remove `tool_from_process_name`, `detect_parent_process_tool`, `detect_env_tool`, `detect_tool`, 3-layer `detect_model`. Replace with env var priority chain + session file fallback for Codex/Cursor. |
| `internal/config/hook_script_test.go` | Rewrite | Remove fake `ps` binary / process tree tests. Add 9 env-var-based tests from spec. |
| `internal/detect/detect.go` | Modify | Remove `InstrumentTempFile` and `InstrumentSetup` fields from `Tool` struct and all entries. |
| `instrument.go` | Modify | Remove `installClaudeStatusLine()`, `installOpenCodePlugin()`. Update `installCodexHooks()` to write `~/.ai-trailer/codex-model` (not `/tmp/codex-current-model`). Add `installCursorHooks()` stub. |
| `main.go` | Modify | `cmdConfigure`: remove per-tool instrumentation loop (`HasInstrument` block). Install Codex/Cursor session hooks. Add `cmdTest`. Update `printUsage`. Register `test` in switch. |

---

## Task 1: Rewrite hook_script_test.go

**Files:**
- Modify: `internal/config/hook_script_test.go`

The new tests set env vars directly — no fake `ps` binary, no `AI_TRAILER_TEST_PROCESS_TREE`. Use a temp `HOME` dir for session file tests.

- [ ] **Step 1: Write the new test file**

Replace `internal/config/hook_script_test.go` entirely with:

```go
package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runHook runs the embedded HookScript with a clean env, a temp HOME,
// and returns the final commit message content.
// envVars is a map of extra env vars to set (e.g. {"CLAUDE_MODEL": "claude-sonnet-4-6"}).
// commitSource is the second argument to the hook ("", "merge", "squash", etc.)
func runHook(t *testing.T, commitMsg string, commitSource string, envVars map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	if err := os.Mkdir(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}

	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte(commitMsg), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{msgPath}
	if commitSource != "" {
		args = append(args, commitSource)
	}

	cmd := exec.Command("bash", append([]string{hookPath}, args...)...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
	}
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook exited non-zero: %v\nstderr/stdout:\n%s", err, output)
	}

	msg, err := os.ReadFile(msgPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(msg)
}

func TestClaudeModelEnvAppendsTrailers(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertContains(t, msg, "Co-authored-by: Claude <noreply@anthropic.com>")
	assertContains(t, msg, "Ai-tool: claude-code")
	assertContains(t, msg, "Ai-model: claude-sonnet-4-6")
	assertContains(t, msg, "Ai-os:")
}

func TestNoAIEnvNoTrailer(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{})

	assertNotContains(t, msg, "Co-authored-by:")
	assertNotContains(t, msg, "Ai-tool:")
	assertNotContains(t, msg, "Ai-model:")
}

func TestMergeSourceSkipsHook(t *testing.T) {
	msg := runHook(t, "Merge branch 'main'\n", "merge", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertNotContains(t, msg, "Ai-tool:")
}

func TestSquashSourceSkipsHook(t *testing.T) {
	msg := runHook(t, "squash commit\n", "squash", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	assertNotContains(t, msg, "Ai-tool:")
}

func TestClaudeModelWithExistingCoauthor(t *testing.T) {
	initial := "subject\n\nCo-authored-by: Alice <alice@example.com>\n"
	msg := runHook(t, initial, "", map[string]string{
		"CLAUDE_MODEL": "claude-sonnet-4-6",
	})

	count := strings.Count(strings.ToLower(msg), "co-authored-by:")
	if count != 1 {
		t.Fatalf("expected 1 Co-authored-by line, got %d:\n%s", count, msg)
	}
	assertContains(t, msg, "Ai-tool: claude-code")
	assertContains(t, msg, "Ai-model: claude-sonnet-4-6")
}

func TestHermesSessionWithModel(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"HERMES_SESSION": "abc123",
		"HERMES_MODEL":   "llama-3",
	})

	assertContains(t, msg, "Co-authored-by: Hermes Agent <noreply@nousresearch.com>")
	assertContains(t, msg, "Ai-tool: hermes")
	assertContains(t, msg, "Ai-model: llama-3")
}

func TestClaudeWinsOverHermesWhenBothPresent(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CLAUDE_MODEL":   "claude-sonnet-4-6",
		"HERMES_SESSION": "abc123",
		"HERMES_MODEL":   "llama-3",
	})

	assertContains(t, msg, "Ai-tool: claude-code")
	assertNotContains(t, msg, "Ai-tool: hermes")
}

func TestCursorTraceIdWithSessionFile(t *testing.T) {
	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	aiTrailerDir := filepath.Join(homeDir, ".ai-trailer")
	if err := os.MkdirAll(aiTrailerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aiTrailerDir, "cursor-model"), []byte("claude-3.7-sonnet"), 0o644); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(HookScript), 0o755); err != nil {
		t.Fatal(err)
	}
	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgPath, []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", hookPath, msgPath)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"CURSOR_TRACE_ID=abc123",
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}

	msg, _ := os.ReadFile(msgPath)
	assertContains(t, string(msg), "Ai-tool: cursor")
	assertContains(t, string(msg), "Ai-model: claude-3.7-sonnet")
}

func TestCursorTraceIdWithoutSessionFile(t *testing.T) {
	msg := runHook(t, "subject\n", "", map[string]string{
		"CURSOR_TRACE_ID": "abc123",
	})

	assertContains(t, msg, "Ai-tool: cursor")
	assertNotContains(t, msg, "Ai-model:")
}

func TestCursorTraceIdWithEmptySessionFile(t *testing.T) {
	dir := t.TempDir()
	homeDir := filepath.Join(dir, "home")
	aiTrailerDir := filepath.Join(homeDir, ".ai-trailer")
	os.MkdirAll(aiTrailerDir, 0o755)
	os.WriteFile(filepath.Join(aiTrailerDir, "cursor-model"), []byte(""), 0o644)

	hookPath := filepath.Join(dir, "prepare-commit-msg")
	os.WriteFile(hookPath, []byte(HookScript), 0o755)
	msgPath := filepath.Join(dir, "COMMIT_EDITMSG")
	os.WriteFile(msgPath, []byte("subject\n"), 0o644)

	cmd := exec.Command("bash", hookPath, msgPath)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"CURSOR_TRACE_ID=abc123",
	}
	cmd.CombinedOutput()

	msg, _ := os.ReadFile(msgPath)
	assertContains(t, string(msg), "Ai-tool: cursor")
	assertNotContains(t, string(msg), "Ai-model:")
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q\nFull message:\n%s", s, substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
		t.Fatalf("expected %q NOT to contain %q\nFull message:\n%s", s, substr, s)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/cora/cora/ai-trailer && go test ./internal/config/... -v 2>&1 | head -60
```

Expected: multiple FAIL — old hook uses process tree / wrong env vars.

- [ ] **Step 3: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add internal/config/hook_script_test.go
git commit -m "test: rewrite hook_script_test.go for env var detection"
```

---

## Task 2: Rewrite hook_script.go

**Files:**
- Modify: `internal/config/hook_script.go`

Replace the 270-line bash script with the new ~80-line version. Remove: `tool_from_process_name`, `detect_env_tool`, `detect_parent_process_tool`, `detect_tool`, 3-layer `detect_model`. Replace with env var priority chain.

- [ ] **Step 1: Write the new hook_script.go**

Replace the entire `HookScript` constant in `internal/config/hook_script.go`:

```go
// Package config — embedded git hook script.
package config

// HookScript is the bash script installed as prepare-commit-msg.
// Detection priority: env vars (confirmed tools) → session files (Codex/Cursor).
const HookScript = `#!/usr/bin/env bash
# prepare-commit-msg — AI Tool Git Trailer Hook
# Installed by: ai-trailer CLI
set -euo pipefail

COMMIT_MSG_FILE="$1"
COMMIT_SOURCE="${2:-}"

case "$COMMIT_SOURCE" in merge|squash) exit 0 ;; esac

TOOL=""
MODEL=""

# ── Env var detection (tools with guaranteed model in subprocess) ──────
if [ -n "${CLAUDE_MODEL:-}" ]; then
    TOOL="claude-code"
    MODEL="$CLAUDE_MODEL"
elif [ -n "${HERMES_SESSION:-}" ]; then
    TOOL="hermes"
    MODEL="${HERMES_MODEL:-}"
elif [ -n "${OPENCODE_MODEL:-}" ]; then
    TOOL="opencode"
    MODEL="$OPENCODE_MODEL"
elif [ -n "${GEMINI_MODEL:-}" ]; then
    TOOL="gemini-cli"
    MODEL="$GEMINI_MODEL"
elif [ -n "${CURSOR_TRACE_ID:-}" ]; then
    TOOL="cursor"
    _sf="$HOME/.ai-trailer/cursor-model"
    if [ -f "$_sf" ]; then
        MODEL=$(cat "$_sf" | tr -d '[:space:]')
    fi
else
    # ── Session file fallback for Codex (file newer than 60 min = active) ──
    _sf="$HOME/.ai-trailer/codex-model"
    if [ -n "$(find "$_sf" -mmin -60 -type f 2>/dev/null)" ]; then
        TOOL="codex"
        MODEL=$(cat "$_sf" | tr -d '[:space:]')
    fi
fi

[ -z "$TOOL" ] && exit 0

# ── Co-author trailer mapping ──────────────────────────────────────────
case "$TOOL" in
    claude-code) COAUTHOR="Co-authored-by: Claude <noreply@anthropic.com>" ;;
    hermes)      COAUTHOR="Co-authored-by: Hermes Agent <noreply@nousresearch.com>" ;;
    opencode)    COAUTHOR="Co-authored-by: OpenCode <noreply@opencode.ai>" ;;
    gemini-cli)  COAUTHOR="Co-authored-by: Gemini <noreply@google.com>" ;;
    cursor)      COAUTHOR="Co-authored-by: Cursor <noreply@cursor.sh>" ;;
    codex)       COAUTHOR="Co-authored-by: OpenAI Codex <noreply@openai.com>" ;;
    *)           exit 0 ;;
esac

# ── Append trailers (idempotent) ───────────────────────────────────────
if ! grep -qi "Co-authored-by:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "\n%s\n" "$COAUTHOR" >> "$COMMIT_MSG_FILE"
fi

if ! grep -qi "Ai-tool:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "Ai-tool: %s\n" "$TOOL" >> "$COMMIT_MSG_FILE"
fi

if [ -n "$MODEL" ] && ! grep -qi "Ai-model:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "Ai-model: %s\n" "$MODEL" >> "$COMMIT_MSG_FILE"
fi

if ! grep -qi "Ai-os:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    case "$(uname -s)" in
        Linux)
            if grep -qi 'microsoft\|wsl' /proc/version 2>/dev/null; then
                AI_OS="wsl"
            else
                AI_OS="linux"
            fi ;;
        Darwin) AI_OS="macos" ;;
        MINGW*|MSYS*|CYGWIN*) AI_OS="windows" ;;
        *) AI_OS="$(uname -s | tr '[:upper:]' '[:lower:]')" ;;
    esac
    printf "Ai-os: %s\n" "$AI_OS" >> "$COMMIT_MSG_FILE"
fi

if command -v ai-trailer &>/dev/null; then
    REPO=$(git rev-parse --show-toplevel 2>/dev/null || echo "unknown")
    COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "pending")
    ai-trailer record-commit "$TOOL" "$COAUTHOR" "$REPO" "$COMMIT_HASH" 2>/dev/null || true
fi
`
```

- [ ] **Step 2: Run tests**

```bash
cd /Users/cora/cora/ai-trailer && go test ./internal/config/... -v 2>&1
```

Expected: all tests PASS (TestCursorTraceIdWithSessionFile, TestHermesSessionWithModel, etc.)

- [ ] **Step 3: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add internal/config/hook_script.go
git commit -m "feat: simplify prepare-commit-msg hook to env var + session file detection"
```

---

## Task 3: Remove InstrumentTempFile/InstrumentSetup from detect.go

**Files:**
- Modify: `internal/detect/detect.go`

These fields are no longer used by the hook or configure command.

- [ ] **Step 1: Remove fields from Tool struct**

In `internal/detect/detect.go`, remove these two lines from the `Tool` struct (lines 26-27):

```go
	InstrumentTempFile string   // Well-known temp file for model tracking (empty if none)
	InstrumentSetup    string   // Description of how to set up model instrumentation
```

- [ ] **Step 2: Remove field values from all tool entries**

Remove `InstrumentTempFile` and `InstrumentSetup` values from these tool entries:
- `claude-code` (lines 44-45)
- Second `codex` entry (lines 83-84)
- `hermes` (lines 109-110)
- `opencode` (lines 123-124)
- `kilocode` (lines 136-137)

After removal the `claude-code` entry looks like:
```go
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
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
cd /Users/cora/cora/ai-trailer && go build ./... 2>&1
```

Expected: compile error in `main.go` referencing `r.Tool.InstrumentTempFile` — this is expected, we'll fix it in Task 5.

- [ ] **Step 4: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add internal/detect/detect.go
git commit -m "refactor: remove InstrumentTempFile/InstrumentSetup from Tool struct"
```

---

## Task 4: Update instrument.go

**Files:**
- Modify: `instrument.go`

Remove `installClaudeStatusLine()` and `installOpenCodePlugin()`. Update `installCodexHooks()` to write to `~/.ai-trailer/codex-model`. Add `installCursorHooks()`.

- [ ] **Step 1: Replace instrument.go**

Replace the entire file:

```go
// Package main — instrument.go: per-tool session file setup for Codex/Cursor.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// installCodexHooks creates ~/.codex/hooks.json with a PreToolUse hook
// and ~/.codex/hooks/model-dump.sh that writes the active model to
// ~/.ai-trailer/codex-model on every tool call.
func installCodexHooks() error {
	home, _ := os.UserHomeDir()
	hooksDir := filepath.Join(home, ".codex", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks dir: %w", err)
	}
	aiTrailerDir := filepath.Join(home, ".ai-trailer")
	if err := os.MkdirAll(aiTrailerDir, 0755); err != nil {
		return fmt.Errorf("creating ~/.ai-trailer: %w", err)
	}

	modelFile := filepath.Join(aiTrailerDir, "codex-model")
	dumpScript := fmt.Sprintf(`#!/bin/bash
# ai-trailer Codex model tracker
python3 -c "
import json, sys
data = json.load(sys.stdin)
model = data.get('model', '')
if model:
    open('%s', 'w').write(model)
" 2>/dev/null || true
`, modelFile)

	dumpPath := filepath.Join(hooksDir, "ai-trailer-model-dump.sh")
	if err := os.WriteFile(dumpPath, []byte(dumpScript), 0755); err != nil {
		return fmt.Errorf("writing model-dump.sh: %w", err)
	}

	hooksJSON := fmt.Sprintf(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash %s"
          }
        ]
      }
    ]
  }
}
`, dumpPath)

	hooksPath := filepath.Join(home, ".codex", "hooks.json")
	if err := os.WriteFile(hooksPath, []byte(hooksJSON), 0644); err != nil {
		return fmt.Errorf("writing hooks.json: %w", err)
	}

	return nil
}

// installCursorHooks is a placeholder — Cursor IDE's hook mechanism needs
// empirical verification before implementation.
// See: docs/2026-05-18-env-var-hook-redesign.md (what this spec doesn't cover)
func installCursorHooks() error {
	fmt.Println("  ℹ  Cursor: PreToolUse hook support not yet verified.")
	fmt.Println("     Session file: ~/.ai-trailer/cursor-model")
	fmt.Println("     See: https://github.com/lucianopf/ai-trailer for updates.")
	return nil
}
```

- [ ] **Step 2: Verify compile (still expect errors from main.go)**

```bash
cd /Users/cora/cora/ai-trailer && go build ./... 2>&1
```

- [ ] **Step 3: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add instrument.go
git commit -m "refactor: update instrument.go — remove status line/plugin setup, write session files to ~/.ai-trailer/"
```

---

## Task 5: Update cmdConfigure and add cmdTest in main.go

**Files:**
- Modify: `main.go`

Three changes: (1) fix compile error from removed `InstrumentTempFile` field, (2) replace the per-tool instrumentation block with Codex/Cursor session hook install, (3) add `cmdTest` and register it in the switch.

- [ ] **Step 1: Fix cmdConfigure — replace the instrumentation block**

In `main.go`, find the `cmdConfigure` function. Remove the `HasInstrument` field from `menuItem` struct (line ~175) and the `// ── Per-tool instrumentation ──` block (lines ~227-255). Replace the instrumentation block with:

```go
	// ── Session file hooks for tools without guaranteed env vars ──
	fmt.Println("\n🔧 Setting up session file hooks for Codex/Cursor...")
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
			installCursorHooks()
		default:
			// Env var detection — no extra setup needed
		}
	}
```

Also, in the `menuItem` struct and `showInteractiveMenu` call, remove the `HasInstrument` field:

Find in `main.go`:
```go
		menuItems = append(menuItems, menuItem{
			ToolName:      r.Tool.Name,
			ToolID:        r.Tool.ID,
			Trailer:       r.Tool.Trailer,
			Selected:      r.Found,
			Detected:      r.Found,
			IsNative:      r.Tool.NativeTrailer,
			HasInstrument: r.Tool.InstrumentTempFile != "",
		})
```

Replace with:
```go
		menuItems = append(menuItems, menuItem{
			ToolName: r.Tool.Name,
			ToolID:   r.Tool.ID,
			Trailer:  r.Tool.Trailer,
			Selected: r.Found,
			Detected: r.Found,
			IsNative: r.Tool.NativeTrailer,
		})
```

- [ ] **Step 2: Find and remove HasInstrument from menuItem struct**

The `menuItem` struct is in one of the TUI files (`tui_common.go` or `tui_other.go`). Check:

```bash
grep -n "HasInstrument\|menuItem" /Users/cora/cora/ai-trailer/tui_common.go /Users/cora/cora/ai-trailer/tui_linux.go /Users/cora/cora/ai-trailer/tui_other.go 2>/dev/null
```

Remove the `HasInstrument bool` field from the `menuItem` struct definition in whichever file contains it.

- [ ] **Step 3: Add cmdTest function**

Add `"time"` to the existing imports block in `main.go`, then add this function before the final comment:

```go
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
	fmt.Printf("    Ai-os: %s\n", config.Platform())
}
```

- [ ] **Step 4: Register cmdTest in main switch**

In `main()`, add before `case "help"`:
```go
	case "test":
		cmdTest(args)
```

- [ ] **Step 5: Build to verify no errors**

```bash
cd /Users/cora/cora/ai-trailer && go build ./... 2>&1
```

Expected: clean build.

- [ ] **Step 6: Run all tests**

```bash
cd /Users/cora/cora/ai-trailer && go test ./... 2>&1
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add main.go tui_common.go tui_linux.go tui_other.go
git commit -m "feat: update cmdConfigure for session hooks; add ai-trailer test command"
```

---

## Task 6: Update printUsage

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace the printUsage function**

Find the `printUsage()` function in `main.go` and replace:

```go
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
```

- [ ] **Step 2: Build and run**

```bash
cd /Users/cora/cora/ai-trailer && go build -o /tmp/ai-trailer-new . && /tmp/ai-trailer-new help
```

Expected: updated usage text, `test` command listed.

- [ ] **Step 3: Smoke test the test command**

```bash
CLAUDE_MODEL=claude-sonnet-4-6 /tmp/ai-trailer-new test
```

Expected:
```
🧪 ai-trailer test — dry run (current env)
───────────────────────────────────────────────────────
  Env detected:  CLAUDE_MODEL=claude-sonnet-4-6

  Would append:
    Co-authored-by: Claude <noreply@anthropic.com>
    Ai-tool: claude-code
    Ai-model: claude-sonnet-4-6
    Ai-os: macos
```

- [ ] **Step 4: Commit**

```bash
cd /Users/cora/cora/ai-trailer
git add main.go
git commit -m "docs: update printUsage to reflect env var detection + test command"
```

---

## Task 7: Run full test suite and build

**Files:** none (verification only)

- [ ] **Step 1: Run all tests**

```bash
cd /Users/cora/cora/ai-trailer && go test ./... -v 2>&1
```

Expected: all tests pass, including:
- `TestClaudeModelEnvAppendsTrailers` ✓
- `TestNoAIEnvNoTrailer` ✓
- `TestMergeSourceSkipsHook` ✓
- `TestSquashSourceSkipsHook` ✓
- `TestClaudeModelWithExistingCoauthor` ✓
- `TestHermesSessionWithModel` ✓
- `TestClaudeWinsOverHermesWhenBothPresent` ✓
- `TestCursorTraceIdWithSessionFile` ✓
- `TestCursorTraceIdWithoutSessionFile` ✓
- `TestCursorTraceIdWithEmptySessionFile` ✓

- [ ] **Step 2: Build release binary**

```bash
cd /Users/cora/cora/ai-trailer && go build -o /tmp/ai-trailer-new . && /tmp/ai-trailer-new version
```

Expected: `ai-trailer v0.5.1` (or whatever current version)

- [ ] **Step 3: End-to-end smoke test — Claude Code**

Run inside a Claude Code session:
```bash
cd /tmp && mkdir smoke-test && cd smoke-test && git init
git config user.email "test@test.com" && git config user.name "Test"
touch test.txt && git add test.txt
CLAUDE_MODEL=claude-sonnet-4-6 GIT_HOOKS_PATH=$(dirname $(which git)) git commit -m "test commit"
git log -1 --format="%B"
```

Expected in log:
```
test commit

Co-authored-by: Claude <noreply@anthropic.com>
Ai-tool: claude-code
Ai-model: claude-sonnet-4-6
Ai-os: macos
```

Or, if the hook is already installed globally, simply make a commit inside a Claude Code session and check `git log -1 --format="%B"`.

- [ ] **Step 4: End-to-end smoke test — no AI env**

```bash
cd /tmp/smoke-test
git commit --allow-empty -m "test outside AI session"
git log -1 --format="%B"
```

Expected: just `test outside AI session` with no trailers.

- [ ] **Step 5: Final commit if anything was adjusted**

```bash
cd /Users/cora/cora/ai-trailer
git log --oneline -8
```

Verify all 5 commits from this plan are present:
1. `test: rewrite hook_script_test.go for env var detection`
2. `feat: simplify prepare-commit-msg hook to env var + session file detection`
3. `refactor: remove InstrumentTempFile/InstrumentSetup from Tool struct`
4. `refactor: update instrument.go — remove status line/plugin setup, write session files to ~/.ai-trailer/`
5. `feat: update cmdConfigure for session hooks; add ai-trailer test command`
6. `docs: update printUsage to reflect env var detection + test command`

---

## Notes for implementer

**Duplicate Codex entry in detect.go:** The `Tools` slice in `detect.go` has Codex defined twice (lines 48-58 and 71-85). The first has no `InstrumentTempFile`; the second does. After removing `InstrumentTempFile`, both entries are identical. Dedup by removing the first Codex entry (lines 48-58) and keeping the second (which was the one with the session file field). Verify `DetectAll()` still returns one Codex result.

**`tui_common.go` / `tui_other.go`:** The `menuItem` struct and `HasInstrument` field are in one of the TUI files. Find it with grep before editing. Do not break the TUI — just remove the `HasInstrument bool` field and its usage in the render function (likely a `✓` indicator next to tool names).

**Codex env var:** The spec notes that which env vars Codex exposes in its subprocess is unverified. The session file approach (60-minute window) is the correct implementation. After `ai-trailer configure` installs the Codex PreToolUse hook, test with `ai-trailer test` after a Codex tool call to confirm the session file is fresh.

**`time` import:** `main.go` already imports several packages. Add `"time"` to the existing import block — don't create a new import block.
