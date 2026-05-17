// Package config — embedded git hook script.
package config

// HookScript is the bash script installed as a prepare-commit-msg git hook.
// It detects which AI tool is making the commit by checking parent processes
// and environment variables, then appends Co-authored-by / Ai-tool / Ai-os /
// Ai-model trailers.
//
// Model detection priority:
//  0. Well-known temp files (per-tool instrumentation — 100% precise)
//  1. Manual override (~/.ai-trailer/models)
//  2. Config file defaults (fallback)
const HookScript = `#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────
# AI Tool Git Trailer Hook (prepare-commit-msg)
# Installed by: ai-trailer CLI
# Detects AI tool and appends Co-authored-by trailer.
# ──────────────────────────────────────────────────────────────────────
set -euo pipefail

COMMIT_MSG_FILE="$1"
COMMIT_SOURCE="${2:-}"

# Skip for merges and squashes
case "$COMMIT_SOURCE" in merge|squash) exit 0 ;; esac

# ── Tool Detection ───────────────────────────────────────────────────

tool_from_process_name() {
    local pname="${1:-}"
    local pbase="${pname##*/}"
    case "$pbase" in
        claude|claude-code)       echo "claude-code";      return 0 ;;
        codex|codex-cli)          echo "codex";            return 0 ;;
        hermes)                   echo "hermes";           return 0 ;;
        opencode)                 echo "opencode";         return 0 ;;
        kilocode)                 echo "kilocode";         return 0 ;;
        cursor|Cursor)            echo "cursor";           return 0 ;;
        copilot|github-copilot|gh-copilot) echo "github-copilot"; return 0 ;;
        gemini|gemini-cli)        echo "gemini-cli";       return 0 ;;
        aider|aider-chat)         echo "aider";            return 0 ;;
        continue)                 echo "continue";         return 0 ;;
        cody|cody-agent)          echo "cody";             return 0 ;;
        windsurf|codeium)         echo "windsurf";         return 0 ;;
        q|amazon-q)               echo "amazon-q";         return 0 ;;
        tabnine)                  echo "tabnine";          return 0 ;;
        coderabbit)               echo "coderabbit";       return 0 ;;
        *)                        return 1 ;;
    esac
}

detect_env_tool() {
    if env | grep -qE '^(CLAUDE_CODE_SIMPLE|CLAUDE_CODE_USE_BEDROCK|CLAUDE_CODE_USE_VERTEX|CLAUDE_CODE_USE_FOUNDRY)='; then
        echo "claude-code"; return 0
    fi
    if env | grep -qE '^(HERMES_|_HERMES_)'; then
        echo "hermes"; return 0
    fi
    if env | grep -qE '^COPILOT_|^GITHUB_COPILOT'; then
        echo "github-copilot"; return 0
    fi
    if env | grep -qE '^AIDER_'; then
        echo "aider"; return 0
    fi
    return 1
}

detect_parent_process_tool() {
    # Linux/WSL: walk /proc parent chain
    if [ -f /proc/self/stat ]; then
        local ppid=$(awk '{print $4}' /proc/self/stat)
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=""
            [ -f "/proc/$ppid/comm" ] && pname=$(tr -d '\0' < "/proc/$ppid/comm" 2>/dev/null || true)
            if tool_from_process_name "$pname"; then return 0; fi
            ppid=$(awk '{print $4}' "/proc/$ppid/stat" 2>/dev/null || echo 1)
        done
    fi

    # macOS: walk ps parent chain
    if [ "$(uname -s)" = "Darwin" ]; then
        local ppid=$PPID
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=$(ps -o comm= -p "$ppid" 2>/dev/null || true)
            if tool_from_process_name "$pname"; then return 0; fi
            ppid=$(ps -o ppid= -p "$ppid" 2>/dev/null | tr -d ' ' || echo 1)
        done
    fi
    return 1
}

detect_tool() {
    detect_parent_process_tool && return 0
    detect_env_tool && return 0
    return 1
}

TOOL=$(detect_tool) || true
[ -z "$TOOL" ] && exit 0

# ── Trailer Mapping ──────────────────────────────────────────────────

case "$TOOL" in
    claude-code)     TRAILER="Co-authored-by: Claude <noreply@anthropic.com>" ;;
    codex)           TRAILER="Co-authored-by: OpenAI Codex <noreply@openai.com>" ;;
    gemini-cli)      TRAILER="Co-authored-by: Gemini <noreply@google.com>" ;;
    github-copilot)  TRAILER="Co-authored-by: GitHub Copilot <noreply@github.com>" ;;
    hermes)          TRAILER="Co-authored-by: Hermes Agent <noreply@nousresearch.com>" ;;
    opencode)        TRAILER="Co-authored-by: OpenCode <noreply@opencode.ai>" ;;
    kilocode)        TRAILER="Co-authored-by: KiloCode <noreply@kilocode.ai>" ;;
    aider)           TRAILER="Co-authored-by: Aider <noreply@aider.chat>" ;;
    continue)        TRAILER="Co-authored-by: Continue <noreply@continue.dev>" ;;
    cody)            TRAILER="Co-authored-by: Cody <noreply@sourcegraph.com>" ;;
    cursor)          TRAILER="Co-authored-by: Cursor <noreply@cursor.sh>" ;;
    windsurf)        TRAILER="Co-authored-by: Windsurf <noreply@codeium.com>" ;;
    amazon-q)        TRAILER="Co-authored-by: Amazon Q <noreply@aws.amazon.com>" ;;
    tabnine)         TRAILER="Co-authored-by: Tabnine <noreply@tabnine.com>" ;;
    coderabbit)      TRAILER="Co-authored-by: CodeRabbit <noreply@coderabbit.ai>" ;;
    *)               exit 0 ;;
esac

# ── Append Trailers (each checked independently) ─────────────────────

HAS_COAUTHOR=0
if grep -qi "Co-authored-by:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    HAS_COAUTHOR=1
fi

if [ "$HAS_COAUTHOR" -eq 0 ]; then
    printf "\n%s\n" "$TRAILER" >> "$COMMIT_MSG_FILE"
fi

if ! grep -qi "Ai-tool:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "Ai-tool: %s\n" "$TOOL" >> "$COMMIT_MSG_FILE"
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

# ── Model Detection ─────────────────────────────────────────────────
#
# Priority:
#   Layer 0 — Agent-side instrumentation temp files (100% precise)
#   Layer 1 — Manual override (~/.ai-trailer/models)
#   Layer 2 — Config file defaults (best-effort fallback)

detect_model() {
    # ── Layer 0: Well-known temp files written by per-tool instrumentation ──
    case "$TOOL" in
        claude-code|kilocode)
            # Status line script writes model after every assistant message
            if [ -f /tmp/claude-current-model ]; then
                cat /tmp/claude-current-model && return 0
            fi
            ;;
        hermes)
            # Session JSON updated every turn — latest by mtime
            local latest
            latest=$(ls -t "$HOME/.hermes/sessions/session_"*.json 2>/dev/null | head -1)
            if [ -n "$latest" ] && [ -f "$latest" ]; then
                python3 -c "import json; print(json.load(open('$latest')).get('model',''))" 2>/dev/null && return 0
            fi
            ;;
        codex)
            # PreToolUse hook writes model on every tool call
            if [ -f /tmp/codex-current-model ]; then
                cat /tmp/codex-current-model && return 0
            fi
            ;;
        opencode)
            # Plugin writes model on session.updated events
            if [ -f /tmp/opencode-current-model ]; then
                cat /tmp/opencode-current-model && return 0
            fi
            ;;
    esac

    # ── Layer 1: Manual override (~/.ai-trailer/models) ──
    local override_file="$HOME/.ai-trailer/models"
    if [ -f "$override_file" ]; then
        local override_model
        override_model=$(grep "^${TOOL}=" "$override_file" 2>/dev/null | head -1 | cut -d= -f2-)
        if [ -n "$override_model" ]; then
            echo "$override_model" && return 0
        fi
    fi

    # ── Layer 2: Config file defaults (best-effort) ──
    case "$TOOL" in
        claude-code|kilocode)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.claude/settings.json" 2>/dev/null && return 0
            grep -oP '"model"\s*:\s*"\K[^"]+' .claude/settings.json 2>/dev/null && return 0
            ;;
        hermes)
            grep -oP '^\s*default:\s*\K\S+' "$HOME/.hermes/config.yaml" 2>/dev/null && return 0
            ;;
        codex)
            grep -oP '^\s*model\s*=\s*"\K[^"]+' "$HOME/.codex/config.toml" 2>/dev/null && return 0
            ;;
        opencode)
            for cfg in "$HOME/.config/opencode/opencode.json" "opencode.json"; do
                [ -f "$cfg" ] && grep -oP '"model"\s*:\s*"\K[^"]+' "$cfg" 2>/dev/null && return 0
            done
            ;;
        gemini-cli)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.gemini/settings.json" 2>/dev/null && return 0
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.config/gemini/settings.json" 2>/dev/null && return 0
            ;;
        aider)
            grep -oP '^\s*model:\s*\K\S+' .aider.conf.yml 2>/dev/null && return 0
            grep -oP '^\s*model:\s*\K\S+' "$HOME/.aider.conf.yml" 2>/dev/null && return 0
            ;;
        cursor)
            for settings in \
                "$HOME/Library/Application Support/Cursor/User/settings.json" \
                "$HOME/.config/Cursor/User/settings.json" \
                "$HOME/.cursor/settings.json" \
                "$HOME/AppData/Roaming/Cursor/User/settings.json"; do
                if [ -f "$settings" ]; then
                    grep -oP '"cursor\.(chat\.)?[Mm]odel"\s*:\s*"\K[^"]+' "$settings" 2>/dev/null | head -1 && return 0
                fi
            done
            ;;
        github-copilot)
            grep -oP 'model:\s*\K\S+' "$HOME/.config/gh/config.yml" 2>/dev/null && return 0
            ;;
        cody)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.cody/config.json" 2>/dev/null && return 0
            ;;
        windsurf)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.windsurf/settings.json" 2>/dev/null && return 0
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.codeium/config.json" 2>/dev/null && return 0
            ;;
        continue)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.continue/config.json" 2>/dev/null && return 0
            ;;
        amazon-q)
            grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.aws/q/config.json" 2>/dev/null && return 0
            ;;
    esac

    echo "unknown"
}

AI_MODEL=$(detect_model)
if ! grep -qi "Ai-model:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "Ai-model: %s\n" "$AI_MODEL" >> "$COMMIT_MSG_FILE"
fi

# ── Record event (local only) ───────────────────────────────────────
if command -v ai-trailer &>/dev/null; then
    REPO=$(git rev-parse --show-toplevel 2>/dev/null || echo "unknown")
    COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "pending")
    ai-trailer record-commit "$TOOL" "$TRAILER" "$REPO" "$COMMIT_HASH" 2>/dev/null || true
fi
`
