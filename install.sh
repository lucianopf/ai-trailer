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
BRANCH="master"
RAW_BASE="https://raw.githubusercontent.com/${REPO}/${BRANCH}"
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
    BINARY_NAME="ai-trailer-${PLATFORM}"

    # Windows → .exe
    if [[ "$PLATFORM" == windows-* ]]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    DOWNLOAD_URL="${RAW_BASE}/dist/${BINARY_NAME}"

    info "Platform: ${PLATFORM}"
    info "Downloading: ${DOWNLOAD_URL}"

    # Create temp dir
    TMPDIR=$(mktemp -d)
    trap "rm -rf ${TMPDIR}" EXIT

    # Download
    if command -v curl &>/dev/null; then
        curl -fsSL "${DOWNLOAD_URL}" -o "${TMPDIR}/${BINARY_NAME}"
    elif command -v wget &>/dev/null; then
        wget -q "${DOWNLOAD_URL}" -O "${TMPDIR}/${BINARY_NAME}"
    else
        err "Neither curl nor wget found. Install one and retry."
        exit 1
    fi

    # Verify download
    if [[ ! -s "${TMPDIR}/${BINARY_NAME}" ]]; then
        err "Download failed or empty file."
        err "URL: ${DOWNLOAD_URL}"
        exit 1
    fi

    # ── macOS: strip quarantine ──────────────────────────────────
    if [[ "$(uname -s)" == "Darwin" ]]; then
        info "macOS detected — stripping quarantine attribute..."
        xattr -d com.apple.quarantine "${TMPDIR}/${BINARY_NAME}" 2>/dev/null || true
        # Also try the recursive flag (belt and suspenders)
        xattr -cr "${TMPDIR}/${BINARY_NAME}" 2>/dev/null || true
        ok "Quarantine removed — no more Gatekeeper prompt!"
    fi

    # Make executable
    chmod +x "${TMPDIR}/${BINARY_NAME}"

    # Install
    mkdir -p "${INSTALL_DIR}"
    FINAL_NAME="ai-trailer"
    if [[ "$PLATFORM" == windows-* ]]; then
        FINAL_NAME="ai-trailer.exe"
    fi

    mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${FINAL_NAME}"
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
