#!/usr/bin/env bash
#
# setup.sh - Complete CodingGame development environment setup
#
# This is the one-stop script for setting up everything you need to develop CodingGame:
#   1. Nix package manager with flakes support
#   2. direnv for automatic environment loading (optional)
#   3. Bazel and Go (via Nix)
#   4. beads issue tracking system
#
# Usage:
#   ./scripts/setup.sh                 # Full automated setup
#   ./scripts/setup.sh --skip-direnv   # Skip direnv installation
#   ./scripts/setup.sh --skip-beads    # Skip beads setup
#   ./scripts/setup.sh --beads-only    # Only set up beads
#   ./scripts/setup.sh --verify        # Only verify existing setup
#
# What this script does:
#   - Installs Nix with flakes enabled (if needed)
#   - Installs direnv for auto-loading (optional)
#   - Verifies Bazel and Go are available
#   - Installs and configures beads issue tracker
#   - Provides clear next steps
#
# Requirements:
#   - curl (for downloading installers)
#   - Linux, macOS, or WSL2
#
# For more: https://github.com/tedks/CodingGame

set -euo pipefail

# ============================================================================
# Configuration and Argument Parsing
# ============================================================================

SKIP_DIRENV=false
SKIP_BEADS=false
BEADS_ONLY=false
VERIFY_ONLY=false
NO_DAEMON=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-direnv)
            SKIP_DIRENV=true
            shift
            ;;
        --skip-beads)
            SKIP_BEADS=true
            shift
            ;;
        --beads-only)
            BEADS_ONLY=true
            shift
            ;;
        --no-daemon)
            NO_DAEMON="1"
            export BEADS_NO_DAEMON=1
            shift
            ;;
        --verify)
            VERIFY_ONLY=true
            shift
            ;;
        -h|--help)
            cat << EOF
CodingGame Development Environment Setup

Usage: $0 [options]

Options:
  --skip-direnv    Skip direnv installation
  --skip-beads     Skip beads setup
  --beads-only     Only set up beads (skip Nix/Bazel)
  --no-daemon      Use no-daemon mode for beads (web Claude Code)
  --verify         Only verify existing setup
  -h, --help       Show this help message

Examples:
  $0                        # Full setup (recommended)
  $0 --skip-direnv          # Setup without direnv
  $0 --beads-only           # Only install beads
  $0 --verify               # Check current setup
EOF
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# ============================================================================
# Colors and Output Functions
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

header() {
    echo ""
    echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}${BOLD}  $1${NC}"
    echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# ============================================================================
# Environment Detection
# ============================================================================

detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        warn "Windows detected. Nix requires WSL2."
        warn "Please install WSL2 and run this script from within WSL."
        error "Native Windows is not supported. Use WSL2."
    else
        echo "unknown"
    fi
}

check_repository() {
    if [[ ! -f "DESIGN.md" ]] || [[ ! -f "flake.nix" ]]; then
        error "Please run this script from the CodingGame repository root"
    fi
}

# ============================================================================
# Nix Installation
# ============================================================================

install_nix() {
    header "Step 1/4: Nix Package Manager"

    if command -v nix &> /dev/null; then
        NIX_VERSION=$(nix --version 2>/dev/null | head -1 || echo "unknown")
        success "Nix is already installed: $NIX_VERSION"

        # Check if flakes are enabled
        if nix eval --expr '1 + 1' 2>/dev/null | grep -q '2'; then
            success "Nix flakes are enabled"
        else
            warn "Nix flakes are not enabled"
            info "Enabling flakes..."
            mkdir -p ~/.config/nix
            if ! grep -q "experimental-features.*flakes" ~/.config/nix/nix.conf 2>/dev/null; then
                echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
                success "Flakes enabled in ~/.config/nix/nix.conf"
            fi
        fi
        return
    fi

    info "Nix is not installed. Installing..."
    echo ""
    warn "This will download and install Nix from Determinate Systems"
    warn "Repository: https://github.com/DeterminateSystems/nix-installer"
    warn "This installer is recommended and includes flakes support"
    echo ""
    warn "Press Ctrl+C within 5 seconds to cancel..."
    sleep 5

    info "Downloading Nix installer..."
    if curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install; then
        success "Nix installed successfully"

        # Source nix for current session
        if [[ -f "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh" ]]; then
            . "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh"
        fi

        info "Nix has been added to your PATH"
    else
        error "Failed to install Nix. Please install manually: https://nixos.org/download.html"
    fi

    if ! command -v nix &> /dev/null; then
        error "Nix installation completed but 'nix' command not found. Try restarting your terminal."
    fi
}

# ============================================================================
# direnv Installation
# ============================================================================

install_direnv() {
    if [[ "$SKIP_DIRENV" == true ]]; then
        info "Skipping direnv installation (--skip-direnv flag set)"
        return
    fi

    header "Step 2/4: direnv (Optional)"

    if command -v direnv &> /dev/null; then
        DIRENV_VERSION=$(direnv version 2>/dev/null || echo "unknown")
        success "direnv is already installed: $DIRENV_VERSION"
        check_direnv_hook
        return
    fi

    echo ""
    info "direnv automatically loads the Nix environment when you cd into the project"
    echo ""
    read -p "Install direnv? [Y/n]: " install_choice

    if [[ "$install_choice" =~ ^[Nn] ]]; then
        info "Skipping direnv installation"
        return
    fi

    local os=$(detect_os)

    if [[ "$os" == "macos" ]]; then
        if command -v brew &> /dev/null; then
            info "Installing direnv via Homebrew..."
            brew install direnv
        else
            warn "Homebrew not found. Installing direnv via Nix..."
            nix-env -iA nixpkgs.direnv
        fi
    elif [[ "$os" == "linux" ]]; then
        info "Installing direnv via Nix..."
        nix-env -iA nixpkgs.direnv
    fi

    if command -v direnv &> /dev/null; then
        success "direnv installed successfully"
        setup_direnv_hook
    else
        warn "Failed to install direnv. You can install it manually later."
    fi
}

check_direnv_hook() {
    local shell_rc=""

    if [[ -n "${BASH_VERSION:-}" ]]; then
        shell_rc="$HOME/.bashrc"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_rc="$HOME/.zshrc"
    else
        case "$SHELL" in
            */bash) shell_rc="$HOME/.bashrc" ;;
            */zsh) shell_rc="$HOME/.zshrc" ;;
            */fish) shell_rc="$HOME/.config/fish/config.fish" ;;
        esac
    fi

    if [[ -n "$shell_rc" ]] && [[ -f "$shell_rc" ]]; then
        if grep -q "direnv hook" "$shell_rc"; then
            success "direnv hook is configured in $shell_rc"
        else
            warn "direnv hook is NOT configured in $shell_rc"
            setup_direnv_hook
        fi
    fi
}

setup_direnv_hook() {
    info "Setting up direnv hook..."

    local shell_rc=""
    local hook_line=""

    if [[ -n "${BASH_VERSION:-}" ]]; then
        shell_rc="$HOME/.bashrc"
        hook_line='eval "$(direnv hook bash)"'
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_rc="$HOME/.zshrc"
        hook_line='eval "$(direnv hook zsh)"'
    else
        case "$SHELL" in
            */bash)
                shell_rc="$HOME/.bashrc"
                hook_line='eval "$(direnv hook bash)"'
                ;;
            */zsh)
                shell_rc="$HOME/.zshrc"
                hook_line='eval "$(direnv hook zsh)"'
                ;;
            */fish)
                shell_rc="$HOME/.config/fish/config.fish"
                hook_line='direnv hook fish | source'
                ;;
            *)
                warn "Unknown shell: $SHELL"
                warn "Please add direnv hook manually. See: https://direnv.net/docs/hook.html"
                return
                ;;
        esac
    fi

    if [[ -n "$shell_rc" ]]; then
        echo "" >> "$shell_rc"
        echo "# direnv hook (added by CodingGame setup)" >> "$shell_rc"
        echo "$hook_line" >> "$shell_rc"
        success "Added direnv hook to $shell_rc"
        warn "Run 'source $shell_rc' or restart your terminal for changes to take effect"
    fi
}

allow_direnv() {
    if [[ "$SKIP_DIRENV" == true ]]; then
        return
    fi

    if ! command -v direnv &> /dev/null; then
        return
    fi

    if [[ -f ".envrc" ]]; then
        info "Found .envrc file"
        if direnv status 2>/dev/null | grep -q "Found RC allowed true"; then
            success "direnv is already allowed for this project"
        else
            info "Allowing direnv for this project..."
            direnv allow
            success "direnv allowed - environment will load automatically"
        fi
    fi
}

# ============================================================================
# Verify Development Environment (Nix + Bazel + Go)
# ============================================================================

verify_environment() {
    header "Step 3/4: Verify Development Tools"

    info "Checking development tools in Nix environment..."
    echo ""

    # Create verification script
    local verify_script=$(mktemp)
    cat > "$verify_script" << 'EOF'
#!/usr/bin/env bash
set -e

echo "Checking Nix..."
if command -v nix &> /dev/null; then
    nix_version=$(nix --version 2>&1 | head -1)
    echo "  ✓ $nix_version"
else
    echo "  ✗ Nix not found"
    exit 1
fi

echo "Checking Go..."
if command -v go &> /dev/null; then
    go_version=$(go version)
    echo "  ✓ $go_version"
else
    echo "  ✗ Go not found in Nix environment"
    exit 1
fi

echo "Checking Bazel..."
if command -v bazel &> /dev/null; then
    bazel_version=$(bazel version 2>&1 | head -1)
    echo "  ✓ $bazel_version"
else
    echo "  ✗ Bazel not found in Nix environment"
    exit 1
fi

echo "Checking git..."
if command -v git &> /dev/null; then
    git_version=$(git --version)
    echo "  ✓ $git_version"
else
    echo "  ✗ git not found"
    exit 1
fi

echo ""
echo "All development tools are available!"
EOF
    chmod +x "$verify_script"

    # Run verification
    if nix develop --command bash "$verify_script"; then
        rm "$verify_script"
        success "Development environment verified"
        return 0
    else
        rm "$verify_script"
        error "Development environment verification failed"
        return 1
    fi
}

# ============================================================================
# beads Installation and Setup
# ============================================================================

install_beads() {
    if [[ "$SKIP_BEADS" == true ]]; then
        info "Skipping beads setup (--skip-beads flag set)"
        return
    fi

    header "Step 4/4: beads Issue Tracking"

    # Check if bd is installed
    info "Checking for bd CLI..."
    if command -v bd &> /dev/null; then
        BD_VERSION=$(bd --version 2>/dev/null | head -1 || echo "unknown")
        success "bd is already installed: $BD_VERSION"
    else
        info "Installing bd CLI..."
        warn "About to download and execute installation script from GitHub"
        warn "Repository: https://github.com/steveyegge/beads"
        warn "Press Ctrl+C within 3 seconds to cancel..."
        sleep 3

        if curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash; then
            success "bd installed successfully"
        else
            error "Failed to install bd. Please install manually: https://github.com/steveyegge/beads"
        fi

        if ! command -v bd &> /dev/null; then
            error "Failed to verify bd installation. Check your PATH or install manually."
        fi
    fi

    # Initialize beads if needed
    if [[ -d ".beads" ]]; then
        success "beads already initialized in this repository"
    else
        info "Initializing beads..."
        if bd init; then
            success "beads initialized"
        else
            error "Failed to initialize beads"
        fi
    fi

    # Install git hooks
    info "Installing git hooks..."
    if bd hooks install; then
        success "Git hooks installed"
    else
        warn "Failed to install git hooks (continuing anyway)"
    fi

    # Configure git merge driver
    info "Configuring git merge driver for beads..."
    if ! git config --get merge.beads.driver &> /dev/null; then
        git config merge.beads.driver "bd merge %O %A %B %L %P"
        git config merge.beads.name "Beads JSONL merge driver"
        success "Git merge driver configured"
    else
        success "Git merge driver already configured"
    fi

    # Claude Code integration
    if [[ -d "$HOME/.claude" ]]; then
        info "Claude Code detected, setting up integration..."
        if bd setup claude; then
            success "Claude Code integration configured"
            warn "Restart Claude Code for hook changes to take effect"
        else
            warn "Failed to set up Claude Code integration (continuing anyway)"
        fi
    fi

    # Run doctor
    info "Running bd doctor to verify setup..."
    if bd doctor; then
        success "beads setup verified"
    else
        warn "Some issues detected - check output above"
    fi

    if [[ -n "$NO_DAEMON" ]]; then
        success "beads configured in no-daemon mode for web Claude Code"
    fi
}

# ============================================================================
# Verification Only Mode
# ============================================================================

verify_only() {
    header "Verifying CodingGame Development Setup"

    local all_ok=true

    echo "Checking Nix..."
    if command -v nix &> /dev/null; then
        nix_version=$(nix --version 2>&1 | head -1)
        success "$nix_version"
    else
        warn "Nix not found"
        all_ok=false
    fi

    echo "Checking direnv..."
    if command -v direnv &> /dev/null; then
        direnv_version=$(direnv version 2>/dev/null || echo "unknown")
        success "direnv $direnv_version"
    else
        info "direnv not installed (optional)"
    fi

    echo "Checking bd (beads)..."
    if command -v bd &> /dev/null; then
        bd_version=$(bd --version 2>/dev/null | head -1 || echo "unknown")
        success "$bd_version"
    else
        warn "bd not found"
        all_ok=false
    fi

    echo ""
    echo "Checking Nix environment tools..."

    # Create verification script
    local verify_script=$(mktemp)
    cat > "$verify_script" << 'EOF'
#!/usr/bin/env bash
all_ok=true

if command -v go &> /dev/null; then
    echo "  ✓ $(go version)"
else
    echo "  ✗ Go not found in Nix environment"
    all_ok=false
fi

if command -v bazel &> /dev/null; then
    echo "  ✓ $(bazel version 2>&1 | head -1)"
else
    echo "  ✗ Bazel not found in Nix environment"
    all_ok=false
fi

if [[ "$all_ok" == false ]]; then
    exit 1
fi
EOF
    chmod +x "$verify_script"

    if nix develop --command bash "$verify_script"; then
        rm "$verify_script"
    else
        rm "$verify_script"
        all_ok=false
    fi

    echo ""
    if [[ "$all_ok" == true ]]; then
        success "All tools verified! Your setup is complete."
        return 0
    else
        warn "Some tools are missing. Run './scripts/setup.sh' to install them."
        return 1
    fi
}

# ============================================================================
# Final Instructions
# ============================================================================

print_next_steps() {
    header "Setup Complete! 🎉"

    echo ""
    echo "Your CodingGame development environment is ready!"
    echo ""

    if command -v direnv &> /dev/null && [[ -f ".envrc" ]]; then
        echo -e "${BOLD}With direnv (recommended):${NC}"
        echo "  1. Restart your terminal (or run: source ~/.bashrc or ~/.zshrc)"
        echo "  2. cd into the CodingGame directory"
        echo "  3. Environment loads automatically! ✨"
        echo ""
        echo -e "${BOLD}Build and run:${NC}"
        echo "  bazel build //..."
        echo "  bazel test //..."
        echo "  bazel run //:codinggame"
    else
        echo -e "${BOLD}Manual Nix environment:${NC}"
        echo "  1. Enter the development environment:"
        echo "     nix develop"
        echo ""
        echo "  2. Build and run:"
        echo "     bazel build //..."
        echo "     bazel test //..."
        echo "     bazel run //:codinggame"
    fi

    if ! [[ "$SKIP_BEADS" == true ]] && command -v bd &> /dev/null; then
        echo ""
        echo -e "${BOLD}beads issue tracking:${NC}"
        echo "  bd ready              # Find available work"
        echo "  bd show <id>          # View issue details"
        echo "  bd update <id> --status in_progress"
        echo "  bd close <id>         # Complete work"
    fi

    echo ""
    echo -e "${BOLD}Learn more:${NC}"
    echo "  • BUILD.md - Detailed build instructions"
    echo "  • AGENTS.md - Agent workflow and guidelines"
    echo "  • DESIGN.md - Complete design specification"
    echo ""
    echo -e "${BOLD}Quick tips:${NC}"
    echo "  • Run './scripts/setup.sh --verify' to check your setup"
    echo "  • Run 'bazel help' for Bazel commands"
    echo "  • Run 'nix flake show' to see available outputs"
    echo ""
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    header "CodingGame Development Environment Setup"

    # Check we're in the right place
    check_repository

    # Verify only mode
    if [[ "$VERIFY_ONLY" == true ]]; then
        verify_only
        exit $?
    fi

    # Detect OS
    local os=$(detect_os)
    info "Detected OS: $os"

    # beads-only mode
    if [[ "$BEADS_ONLY" == true ]]; then
        install_beads
        echo ""
        success "beads setup complete!"
        exit 0
    fi

    # Full setup
    install_nix
    install_direnv
    verify_environment
    allow_direnv
    install_beads
    print_next_steps
}

# Run main
main
