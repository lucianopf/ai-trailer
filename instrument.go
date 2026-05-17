// Package main — instrument.go: per-tool model tracking setup.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// installClaudeStatusLine patches ~/.claude/statusline-command.sh to
// write the current model to /tmp/claude-current-model after every
// assistant message.  Works for both Claude Code and KiloCode.
func installClaudeStatusLine() error {
	home, _ := os.UserHomeDir()
	scriptPath := filepath.Join(home, ".claude", "statusline-command.sh")

	// Check if model tracking is already present
	existing, err := os.ReadFile(scriptPath)
	if err == nil && strings.Contains(string(existing), "/tmp/claude-current-model") {
		return nil // Already installed
	}

	// Ensure directory exists
	os.MkdirAll(filepath.Join(home, ".claude"), 0755)

	// Build the model-tracking preamble
	preamble := `#!/bin/bash
# ── ai-trailer model tracking ──
# Writes current model to /tmp/claude-current-model on every message.
# Managed by ai-trailer CLI — do not edit this block manually.
input=$(cat)
model=$(echo "$input" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('model',{}).get('display_name','') or d.get('model',{}).get('id','') or 'unknown')" 2>/dev/null || echo "unknown")
echo "$model" > /tmp/claude-current-model
# ── end ai-trailer block ──

`

	// If statusline script already exists, prepend our block without
	// touching the existing content.  Otherwise create a minimal script.
	var newContent string
	if err == nil && len(existing) > 0 {
		existingStr := string(existing)
		// Remove shebang if present (we'll add our own)
		existingStr = strings.TrimPrefix(existingStr, "#!/bin/bash\n")
		existingStr = strings.TrimPrefix(existingStr, "#!/usr/bin/env bash\n")
		// Remove any previous ai-trailer block
		if idx := strings.Index(existingStr, "# ── ai-trailer model tracking ──"); idx >= 0 {
			endIdx := strings.Index(existingStr[idx:], "# ── end ai-trailer block ──")
			if endIdx >= 0 {
				existingStr = existingStr[:idx] + existingStr[idx+endIdx+len("# ── end ai-trailer block ──"):]
			}
		}
		newContent = preamble + strings.TrimSpace(existingStr) + "\n"
	} else {
		newContent = preamble + "# Existing statusline content below\n"
	}

	return os.WriteFile(scriptPath, []byte(newContent), 0755)
}

// installCodexHooks creates ~/.codex/hooks.json with a PreToolUse hook
// and ~/.codex/hooks/model-dump.sh that writes the active model to
// /tmp/codex-current-model on every tool call.
func installCodexHooks() error {
	home, _ := os.UserHomeDir()
	hooksDir := filepath.Join(home, ".codex", "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Write model-dump.sh
	dumpScript := `#!/bin/bash
# ai-trailer Codex model tracker — writes model to /tmp/codex-current-model
python3 -c "
import json, sys
data = json.load(sys.stdin)
model = data.get('model', 'unknown')
open('/tmp/codex-current-model', 'w').write(model)
"
`
	dumpPath := filepath.Join(hooksDir, "model-dump.sh")
	if err := os.WriteFile(dumpPath, []byte(dumpScript), 0755); err != nil {
		return fmt.Errorf("writing model-dump.sh: %w", err)
	}

	// Write hooks.json
	hooksJSON := fmt.Sprintf(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash %s/model-dump.sh"
          }
        ]
      }
    ]
  }
}
`, hooksDir)
	hooksPath := filepath.Join(home, ".codex", "hooks.json")
	if err := os.WriteFile(hooksPath, []byte(hooksJSON), 0644); err != nil {
		return fmt.Errorf("writing hooks.json: %w", err)
	}

	// Note: user still needs to run /hooks in Codex and [t]rust the hook.
	// Future Codex versions may auto-trust via hooks.state in config.toml.
	return nil
}

// installOpenCodePlugin creates a JS plugin in ~/.config/opencode/plugins/
// that listens for session.updated events and writes the model to
// /tmp/opencode-current-model.
func installOpenCodePlugin() error {
	home, _ := os.UserHomeDir()
	pluginDir := filepath.Join(home, ".config", "opencode", "plugins")
	os.MkdirAll(pluginDir, 0755)

	pluginJS := `// ai-trailer OpenCode model tracker
// Writes current model to /tmp/opencode-current-model on session updates.
export const AiTrailerModelTracker = async ({ client }) => {
  return {
    "session.updated": async (input) => {
      try {
        // The session object should contain the current model
        const fs = require("fs");
        const model = input?.session?.model ||
                      input?.model ||
                      (await client?.session?.current?.())?.model ||
                      "unknown";
        fs.writeFileSync("/tmp/opencode-current-model", String(model));
      } catch (e) {
        // Silently ignore — don't break OpenCode
      }
    },
  };
};
`
	pluginPath := filepath.Join(pluginDir, "ai-trailer-model-tracker.js")
	return os.WriteFile(pluginPath, []byte(pluginJS), 0644)
}
