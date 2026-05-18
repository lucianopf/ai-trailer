// Package config — embedded git hook script.
package config

// HookScript is the bash script installed as prepare-commit-msg.
// Detection priority: env vars (confirmed tools) → session files (Codex/Cursor).
const HookScript = `#!/usr/bin/env bash
# prepare-commit-msg — AI Tool Git Trailer Hook
# Installed by: ai-trailer CLI
set -uo pipefail  # -e intentionally omitted: hook must never block a commit

COMMIT_MSG_FILE="$1"
COMMIT_SOURCE="${2:-}"

case "$COMMIT_SOURCE" in merge|squash) exit 0 ;; esac

TOOL=""
MODEL=""
COAUTHOR=""

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
        MODEL=$(tr -d '[:space:]' < "$_sf")
    fi
else
    # ── Session file fallback for Codex (file newer than 60 min = active) ──
    _sf="$HOME/.ai-trailer/codex-model"
    if [ -n "$(find "$_sf" -mmin -60 -type f 2>/dev/null)" ]; then
        TOOL="codex"
        MODEL=$(tr -d '[:space:]' < "$_sf")
    fi
fi

if [ -z "$TOOL" ]; then
    exit 0
fi

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
