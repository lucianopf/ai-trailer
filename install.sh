#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════
# ai-trailer — One-line installer (macOS, Linux, WSL)
# ═══════════════════════════════════════════════════════════════════════
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
#
# Or locally:
#   chmod +x install.sh && ./install.sh
# ═══════════════════════════════════════════════════════════════════════
set -euo pipefail

REPO="lucianopf/ai-trailer"
INSTALL_DIR="${HOME}/.local/bin"

# ── Colors ───────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${CYAN}ℹ${NC}  $*"; }
ok()    { echo -e "${GREEN}✓${NC}  $*"; }
warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
err()   { echo -e "${RED}✗${NC}  $*"; }

# ── Detect platform ─────────────────────────────────────────────────
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *) err "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) err "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# ── Main ────────────────────────────────────────────────────────────
main() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║           ai-trailer — One-Line Installer                ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""

    PLATFORM=$(detect_platform)
    BINARY_SRC="ai-trailer-${PLATFORM}"

    # Windows → .exe
    if [[ "$PLATFORM" == windows-* ]]; then
        BINARY_SRC="${BINARY_SRC}.exe"
    fi

    info "Platform: ${PLATFORM}"

    # Create temp dir
    TMPDIR=$(mktemp -d)
    trap "rm -rf ${TMPDIR}" EXIT

    # ── Clone repo (shallow) to get pre-built binaries ────────────
    info "Cloning ai-trailer (shallow)..."

    CLONE_ARGS=(--depth 1 --filter=blob:none --sparse)
    if git clone "${CLONE_ARGS[@]}" "git@github.com:${REPO}.git" "${TMPDIR}/repo" 2>/dev/null; then
        ok "Cloned via SSH"
    elif git clone "${CLONE_ARGS[@]}" "https://github.com/${REPO}.git" "${TMPDIR}/repo" 2>/dev/null; then
        ok "Cloned via HTTPS"
    elif command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
        info "Using gh CLI..."
        gh repo clone "${REPO}" "${TMPDIR}/repo" -- --depth 1 --filter=blob:none --sparse
        ok "Cloned via gh CLI"
    else
        err ""
        err "  Cannot clone the repo. Make sure you have access to ${REPO}."
        err ""
        err "  Try one of these first:"
        err "    • gh auth login"
        err "    • ssh-add ~/.ssh/id_ed25519  (or your SSH key)"
        err "    • git clone git@github.com:${REPO}.git"
        err ""
        err "  Then run this installer again."
        exit 1
    fi

    cd "${TMPDIR}/repo"
    git sparse-checkout set dist 2>/dev/null || true

    # Find the binary
    BINARY_PATH="${TMPDIR}/repo/dist/${BINARY_SRC}"
    if [[ ! -f "${BINARY_PATH}" ]]; then
        err "Binary not found: dist/${BINARY_SRC}"
        err "Available binaries:"
        ls -1 "${TMPDIR}/repo/dist/" 2>/dev/null || echo "  (none)"
        exit 1
    fi
    ok "Found: dist/${BINARY_SRC}"

    # ── macOS: strip quarantine ──────────────────────────────────
    if [[ "$(uname -s)" == "Darwin" ]]; then
        info "macOS detected — stripping quarantine attribute..."
        xattr -d com.apple.quarantine "${BINARY_PATH}" 2>/dev/null || true
        xattr -cr "${BINARY_PATH}" 2>/dev/null || true
        ok "Quarantine removed — no more Gatekeeper prompt!"
    fi

    # Make executable
    chmod +x "${BINARY_PATH}"

    # Install
    mkdir -p "${INSTALL_DIR}"
    FINAL_NAME="ai-trailer"
    if [[ "$PLATFORM" == windows-* ]]; then
        FINAL_NAME="ai-trailer.exe"
    fi

    cp "${BINARY_PATH}" "${INSTALL_DIR}/${FINAL_NAME}"
    ok "Installed to ${INSTALL_DIR}/${FINAL_NAME}"

    # Check if install dir is in PATH
    if ! echo "${PATH}" | tr ':' '\n' | grep -qxF "${INSTALL_DIR}"; then
        warn ""
        warn "  ${INSTALL_DIR} is not in your PATH!"
        warn "  Add this to your shell config:"
        echo ""
        echo -e "    ${GREEN}export PATH=\"\${HOME}/.local/bin:\${PATH}\"${NC}"
        echo ""
        case "${SHELL}" in
            */zsh)  echo "  (add to ~/.zshrc)" ;;
            */bash) echo "  (add to ~/.bashrc)" ;;
            */fish) echo "  (run: fish_add_path ~/.local/bin)" ;;
        esac
    fi

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   Installation complete!                                ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "  Next step — configure your AI tools:"
    echo ""
    echo -e "    ${CYAN}ai-trailer configure${NC}"
    echo ""
    echo "  Or configure everything automatically:"
    echo ""
    echo -e "    ${CYAN}ai-trailer configure --all${NC}"
    echo ""
}

main "$@"
