// Package config — embedded git hook script.
package config

// HookScript is the bash script installed as a prepare-commit-msg git hook.
// It detects which AI tool is making the commit by checking environment
// variables and parent processes, then appends the appropriate
// Co-authored-by trailer.
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

# ── Detection ────────────────────────────────────────────────────────

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
    # Use only explicit runtime markers as a fallback. Broad prefixes can be
    # inherited by other tools and cause false attribution.
    if env | grep -qE '^(CLAUDE_CODE_SIMPLE|CLAUDE_CODE_USE_BEDROCK|CLAUDE_CODE_USE_VERTEX|CLAUDE_CODE_USE_FOUNDRY)='; then
        echo "claude-code"; return 0
    fi
    if env | grep -qE '^(HERMES_|_HERMES_)'; then
        echo "hermes"; return 0
    fi
    if env | grep -qE '^COPILOT_|^GITHUB_COPILOT'; then
        echo "github-copilot"; return 0
    fi
    if env | grep -qE '^CURSOR_'; then
        echo "cursor"; return 0
    fi
    if env | grep -qE '^AIDER_'; then
        echo "aider"; return 0
    fi
    return 1
}

detect_parent_process_tool() {
    # Strategy 1: Walk parent process tree (Linux/WSL)
    if [ -f /proc/self/stat ]; then
        local ppid=$(awk '{print $4}' /proc/self/stat)
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=""
            [ -f "/proc/$ppid/comm" ] && pname=$(tr -d '\0' < "/proc/$ppid/comm" 2>/dev/null || true)
            local tool=""
            if tool=$(tool_from_process_name "$pname"); then
                echo "$tool"; return 0
            fi
            ppid=$(awk '{print $4}' "/proc/$ppid/stat" 2>/dev/null || echo 1)
        done
    fi

    # Strategy 2: macOS parent process walk
    if [ "$(uname -s)" = "Darwin" ]; then
        local ppid=$PPID
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=$(ps -o comm= -p "$ppid" 2>/dev/null || true)
            local tool=""
            if tool=$(tool_from_process_name "$pname"); then
                echo "$tool"; return 0
            fi
            ppid=$(ps -o ppid= -p "$ppid" 2>/dev/null | tr -d ' ' || echo 1)
        done
    fi
    return 1
}

detect_tool() {
    # Prefer the actual parent process tree. Env vars are often inherited
    # across shells and nested tools, so they are only a fallback.
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

# ── Append Trailers ───────────────────────────────────────────────────

# Check if Co-authored-by already exists (written by the AI tool itself)
HAS_COAUTHOR=0
if grep -qi "Co-authored-by:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    HAS_COAUTHOR=1
fi

# Append Co-authored-by if not already present
if [ "$HAS_COAUTHOR" -eq 0 ]; then
    printf "\n%s\n" "$TRAILER" >> "$COMMIT_MSG_FILE"
fi

# Always append Ai-tool if missing (tools often write Co-authored-by but not Ai-*)
if ! grep -qi "Ai-tool:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    printf "Ai-tool: %s\n" "$TOOL" >> "$COMMIT_MSG_FILE"
fi

# Detect OS and append Ai-os if missing
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

# ── Detect model ────────────────────────────────────────────────────
detect_model() {
    case "$TOOL" in
        claude-code|kilocode)
            # ~/.claude/settings.json → model field
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.claude/settings.json" 2>/dev/null || true)
            # Also check project-level settings
            [ -z "$MODEL" ] && MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' .claude/settings.json 2>/dev/null || true)
            ;;
        hermes)
            # ~/.hermes/config.yaml → model.default
            MODEL=$(grep -oP '^\s*default:\s*\K\S+' "$HOME/.hermes/config.yaml" 2>/dev/null || true)
            ;;
        codex)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.codex/config.json" 2>/dev/null || true)
            ;;
        opencode)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.opencode/config.json" 2>/dev/null || true)
            [ -z "$MODEL" ] && MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' .opencode/config.json 2>/dev/null || true)
            ;;
        gemini-cli)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.gemini/settings.json" 2>/dev/null || true)
            [ -z "$MODEL" ] && MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.config/gemini/settings.json" 2>/dev/null || true)
            ;;
        github-copilot|copilot)
            # Copilot stores model in VS Code / Cursor settings.json
            for settings in \
                "$HOME/Library/Application Support/Code/User/settings.json" \
                "$HOME/Library/Application Support/Cursor/User/settings.json" \
                "$HOME/.vscode-server/data/Machine/settings.json" \
                "$HOME/.config/Code/User/settings.json" \
                "$HOME/.config/Cursor/User/settings.json" \
                "$HOME/AppData/Roaming/Code/User/settings.json"
            do
                if [ -f "$settings" ]; then
                    MODEL=$(grep -oP '"github\.copilot\.(chat|selectedCompletion|advanced)\.?\w*[Mm]odel"\s*:\s*"\K[^"]+' "$settings" 2>/dev/null | head -1 || true)
                    [ -n "$MODEL" ] && break
                fi
            done
            ;;
        aider)
            MODEL=$(grep -oP '^\s*model:\s*\K\S+' .aider.conf.yml 2>/dev/null || true)
            [ -z "$MODEL" ] && MODEL=$(grep -oP '^\s*model:\s*\K\S+' "$HOME/.aider.conf.yml" 2>/dev/null || true)
            ;;
        cody)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.cody/config.json" 2>/dev/null || true)
            ;;
        windsurf)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.windsurf/settings.json" 2>/dev/null || true)
            [ -z "$MODEL" ] && MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.codeium/config.json" 2>/dev/null || true)
            ;;
        cursor)
            MODEL="Sonnet (default)"
            ;;
        continue)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.continue/config.json" 2>/dev/null || true)
            ;;
        amazon-q)
            MODEL=$(grep -oP '"model"\s*:\s*"\K[^"]+' "$HOME/.aws/q/config.json" 2>/dev/null || true)
            ;;
        *)
            MODEL=""
            ;;
    esac
    [ -z "$MODEL" ] && MODEL="unknown"
    echo "$MODEL"
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
