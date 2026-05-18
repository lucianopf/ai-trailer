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
elif [ -n "${GEMINI_MODEL:-}" ]; then
    TOOL="gemini-cli"
    MODEL="$GEMINI_MODEL"
elif [ -n "${WINDSURF_EXTENSION_VERSION:-}" ]; then
    TOOL="windsurf"
    MODEL=""
elif [ -n "${CURSOR_TRACE_ID:-}" ]; then
    TOOL="cursor"
    _sf="$HOME/.ai-trailer/cursor-model"
    if [ -f "$_sf" ]; then
        MODEL=$(tr -d '[:space:]' < "$_sf")
    fi
else
    # ── Codex: detect from its native Co-authored-by trailer ──────────
    # Codex CLI injects "Co-authored-by: Codex <model>" before the hook runs.
    _codex_line=$(grep -i "Co-authored-by: Codex <" "$COMMIT_MSG_FILE" 2>/dev/null | head -1)
    if [ -n "$_codex_line" ]; then
        TOOL="codex"
        MODEL=$(echo "$_codex_line" | sed 's/.*<\([^>]*\)>/\1/')
    # ── Copilot: detect from its native Co-authored-by trailer ────────
    # GitHub Copilot CLI injects "Co-authored-by: Copilot <...>" before the hook.
    # Model is read from VS Code's state DB (sqlite3): the key
    # chat.currentLanguageModel.panel.copilotcli stores e.g. "copilotcli/claude-opus-4.6".
    elif grep -qi "Co-authored-by: Copilot <" "$COMMIT_MSG_FILE" 2>/dev/null; then
        TOOL="github-copilot"
        if command -v sqlite3 &>/dev/null; then
            for _db in \
                "$HOME/Library/Application Support/Code/User/globalStorage/state.vscdb" \
                "$HOME/.config/Code/User/globalStorage/state.vscdb" \
                "$HOME/.config/Code - Insiders/User/globalStorage/state.vscdb"; do
                if [ -f "$_db" ]; then
                    _raw=$(sqlite3 "$_db" \
                        "SELECT value FROM ItemTable WHERE key='chat.currentLanguageModel.panel.copilotcli';" \
                        2>/dev/null)
                    if [ -n "$_raw" ]; then
                        MODEL=$(echo "$_raw" | sed 's|.*/||')
                        break
                    fi
                fi
            done
        fi
    # ── OpenCode: detect by active process + recent DB activity ──────────
    # Both conditions required to avoid false positives when another tool
    # commits shortly after an OpenCode session ends.
    elif pgrep -q "opencode" 2>/dev/null \
         && [ -n "$(find "$HOME/.local/share/opencode/opencode.db" -mmin -10 -type f 2>/dev/null)" ] \
         && command -v sqlite3 &>/dev/null; then
        TOOL="opencode"
        _oc_model_json=$(sqlite3 "$HOME/.local/share/opencode/opencode.db" \
            "SELECT model FROM session ORDER BY time_updated DESC LIMIT 1;" \
            2>/dev/null)
        if [ -n "$_oc_model_json" ]; then
            MODEL=$(echo "$_oc_model_json" | sed 's/.*"id":"\([^"]*\)".*/\1/')
        fi
    else
        # ── Session file fallback (for Codex versions without native trailer) ──
        _sf="$HOME/.ai-trailer/codex-model"
        if [ -n "$(find "$_sf" -mmin -60 -type f 2>/dev/null)" ]; then
            TOOL="codex"
            MODEL=$(tr -d '[:space:]' < "$_sf")
        fi
    fi
fi

if [ -z "$TOOL" ]; then
    exit 0
fi

# ── Co-author trailer mapping ──────────────────────────────────────────
case "$TOOL" in
    claude-code)    COAUTHOR="Co-authored-by: Claude <noreply@anthropic.com>" ;;
    hermes)         COAUTHOR="Co-authored-by: Hermes Agent <noreply@nousresearch.com>" ;;
    opencode)       COAUTHOR="Co-authored-by: OpenCode <noreply@opencode.ai>" ;;
    gemini-cli)     COAUTHOR="Co-authored-by: Gemini <noreply@google.com>" ;;
    windsurf)       COAUTHOR="Co-authored-by: Windsurf <noreply@codeium.com>" ;;
    cursor)         COAUTHOR="Co-authored-by: Cursor <noreply@cursor.sh>" ;;
    codex)          COAUTHOR="Co-authored-by: OpenAI Codex <noreply@openai.com>" ;;
    github-copilot) COAUTHOR="Co-authored-by: GitHub Copilot <noreply@github.com>" ;;
    *)              exit 0 ;;
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
