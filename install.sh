#!/usr/bin/env bash
# ai-trailer — One-line installer (macOS, Linux, WSL)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
set -euo pipefail

REPO="lucianopf/ai-trailer"
INSTALL_DIR="${HOME}/.local/bin"
RELEASES_URL="https://github.com/${REPO}/releases/latest/download"

# ── Colors ───────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${CYAN}ℹ${NC}  $*"; }
ok()    { echo -e "${GREEN}✓${NC}  $*"; }
warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
die()   { echo -e "${RED}✗${NC}  $*" >&2; exit 1; }

# ── Detect platform ──────────────────────────────────────────────────
detect_platform() {
    local os arch
    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *) die "Unsupported OS: $(uname -s)" ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) die "Unsupported architecture: $(uname -m)" ;;
    esac
    echo "${os}-${arch}"
}

main() {
    echo ""
    echo -e "${CYAN}  ai-trailer — installer${NC}"
    echo -e "  ─────────────────────────────────────────────────────────"
    echo ""

    local platform binary_name url tmpfile final_name

    platform=$(detect_platform)
    binary_name="ai-trailer-${platform}"
    [[ "$platform" == windows-* ]] && binary_name="${binary_name}.exe"

    url="${RELEASES_URL}/${binary_name}"
    info "Platform : ${platform}"
    info "Download : ${url}"

    tmpfile=$(mktemp)
    trap "rm -f ${tmpfile}" EXIT

    # Download
    if command -v curl &>/dev/null; then
        curl -fsSL --progress-bar -o "${tmpfile}" "${url}" || die "Download failed: ${url}"
    elif command -v wget &>/dev/null; then
        wget -q --show-progress -O "${tmpfile}" "${url}" || die "Download failed: ${url}"
    else
        die "Neither curl nor wget found. Install one and retry."
    fi
    ok "Downloaded $(du -h "${tmpfile}" | cut -f1)"

    # macOS: strip quarantine so Gatekeeper doesn't prompt
    if [[ "$(uname -s)" == "Darwin" ]]; then
        xattr -d com.apple.quarantine "${tmpfile}" 2>/dev/null || true
        xattr -cr "${tmpfile}" 2>/dev/null || true
    fi

    chmod +x "${tmpfile}"

    # Install
    mkdir -p "${INSTALL_DIR}"
    final_name="ai-trailer"
    [[ "$platform" == windows-* ]] && final_name="ai-trailer.exe"
    cp "${tmpfile}" "${INSTALL_DIR}/${final_name}"
    ok "Installed → ${INSTALL_DIR}/${final_name}"

    # PATH check
    if ! echo "${PATH}" | tr ':' '\n' | grep -qxF "${INSTALL_DIR}"; then
        echo ""
        warn "${INSTALL_DIR} is not in your PATH. Add this to your shell config:"
        echo ""
        echo -e "    ${GREEN}export PATH=\"\${HOME}/.local/bin:\${PATH}\"${NC}"
        echo ""
        case "${SHELL:-}" in
            */zsh)  echo "  → ~/.zshrc" ;;
            */bash) echo "  → ~/.bashrc or ~/.bash_profile" ;;
            */fish) echo "  → run: fish_add_path ~/.local/bin" ;;
        esac
        echo ""
        warn "Reload your shell before running ai-trailer."
        echo ""
    fi

    echo -e "${GREEN}  ✓ Installation complete!${NC}"
    echo ""
    echo "  Run this to set up git hooks for your AI tools:"
    echo ""
    echo -e "    ${CYAN}ai-trailer configure${NC}"
    echo ""
}

main "$@"
