// Package main — instrument.go: per-tool session file setup for Codex/Cursor.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// installCodexHooks creates ~/.codex/hooks.json with a PreToolUse hook
// and ~/.codex/hooks/ai-trailer-model-dump.sh that writes the active model to
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
