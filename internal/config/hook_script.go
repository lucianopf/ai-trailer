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

detect_tool() {
    # Strategy 1: Check environment variables (most reliable)
    # Claude Code
    if env | grep -qE '^CLAUDE_CODE_'; then
        echo "claude-code"; return 0
    fi
    # Hermes Agent
    if env | grep -qE '^(HERMES_|_HERMES_)'; then
        echo "hermes"; return 0
    fi
    # GitHub Copilot
    if env | grep -qE '^COPILOT_|^GITHUB_COPILOT'; then
        echo "github-copilot"; return 0
    fi
    # Gemini
    if env | grep -qE '^(GOOGLE_API_KEY|GEMINI_API_KEY)'; then
        # Only if running under gemini process (checked in walk below)
        :
    fi
    # Amazon Q
    if env | grep -qE '^AWS_PROFILE'; then
        :
    fi
    # Cursor
    if env | grep -qE '^CURSOR_'; then
        echo "cursor"; return 0
    fi
    # Aider
    if env | grep -qE '^AIDER_'; then
        echo "aider"; return 0
    fi

    # Strategy 2: Walk parent process tree (Linux/WSL)
    if [ -f /proc/self/stat ]; then
        local ppid=$(awk '{print $4}' /proc/self/stat)
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=""
            [ -f "/proc/$ppid/comm" ] && pname=$(tr -d '\0' < "/proc/$ppid/comm" 2>/dev/null || true)
            case "$pname" in
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
                node|python|python3|bash|sh|zsh|git|dash|tmux) ;; # Keep walking
                *) ;; # Unknown, keep walking
            esac
            # Check env of this process for tools that share env vars
            if [ -f "/proc/$ppid/environ" ]; then
                local penv=$(tr '\0' '\n' < "/proc/$ppid/environ" 2>/dev/null || true)
                if echo "$penv" | grep -qE '^CLAUDE_CODE_';         then echo "claude-code";      return 0; fi
                if echo "$penv" | grep -qE '^(HERMES_|_HERMES_)';   then echo "hermes";           return 0; fi
                if echo "$penv" | grep -qE '^COPILOT_';             then echo "github-copilot";   return 0; fi
                if echo "$penv" | grep -qE '^AIDER_';               then echo "aider";            return 0; fi
                # Gemini needs process name + API key
                if echo "$penv" | grep -qE '^GEMINI_API_KEY'; then
                    case "$pname" in gemini|gemini-cli) echo "gemini-cli"; return 0 ;; esac
                fi
                # Amazon Q
                if echo "$penv" | grep -qE '^AWS_PROFILE'; then
                    case "$pname" in q|amazon-q) echo "amazon-q"; return 0 ;; esac
                fi
            fi
            ppid=$(awk '{print $4}' "/proc/$ppid/stat" 2>/dev/null || echo 1)
        done
    fi

    # Strategy 3: macOS parent process walk
    if [ "$(uname -s)" = "Darwin" ]; then
        local ppid=$PPID
        for _ in $(seq 1 10); do
            [ "$ppid" -le 1 ] && break
            local pname=$(ps -o comm= -p "$ppid" 2>/dev/null || true)
            case "$pname" in
                claude|claude-code)       echo "claude-code";      return 0 ;;
                codex|codex-cli)          echo "codex";            return 0 ;;
                hermes)                   echo "hermes";           return 0 ;;
                opencode)                 echo "opencode";         return 0 ;;
                kilocode)                 echo "kilocode";         return 0 ;;
                cursor|Cursor)            echo "cursor";           return 0 ;;
                copilot|github-copilot)   echo "github-copilot";   return 0 ;;
                gemini|gemini-cli)        echo "gemini-cli";       return 0 ;;
                aider|aider-chat)         echo "aider";            return 0 ;;
                continue)                 echo "continue";         return 0 ;;
                cody|cody-agent)          echo "cody";             return 0 ;;
                windsurf|codeium)         echo "windsurf";         return 0 ;;
                q|amazon-q)               echo "amazon-q";         return 0 ;;
                tabnine)                  echo "tabnine";          return 0 ;;
                coderabbit)               echo "coderabbit";       return 0 ;;
                node|python|python3|bash|sh|zsh|git) ;;
                *) ;;
            esac
            ppid=$(ps -o ppid= -p "$ppid" 2>/dev/null | tr -d ' ' || echo 1)
        done
    fi

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

# ── Append Trailer ───────────────────────────────────────────────────

# Skip if trailer already present
if grep -qF "$TRAILER" "$COMMIT_MSG_FILE" 2>/dev/null; then
    exit 0
fi

# Skip if any Co-authored-by already present (avoid duplicates)
if grep -qi "Co-authored-by:" "$COMMIT_MSG_FILE" 2>/dev/null; then
    exit 0
fi

# Append
printf "\n%s\n" "$TRAILER" >> "$COMMIT_MSG_FILE"

# ── Record event (local only) ───────────────────────────────────────
if command -v ai-trailer &>/dev/null; then
    REPO=$(git rev-parse --show-toplevel 2>/dev/null || echo "unknown")
    COMMIT_HASH=$(git rev-parse HEAD 2>/dev/null || echo "pending")
    ai-trailer record-commit "$TOOL" "$TRAILER" "$REPO" "$COMMIT_HASH" 2>/dev/null || true
fi
`
