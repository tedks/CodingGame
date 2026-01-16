#!/usr/bin/env bash
#
# setup-dev-env.sh - Install Nix development environment for CodingGame
#
# This script sets up the complete development environment including:
#   - Nix package manager with flakes support
#   - direnv for automatic environment loading (optional)
#   - Verification of the Nix + Bazel setup
#
# Usage:
#   ./scripts/setup-dev-env.sh
#   ./scripts/setup-dev-env.sh --skip-direnv    # Skip direnv installation
#   ./scripts/setup-dev-env.sh --uninstall      # Remove Nix (use with caution)
#
# What this script does:
#   1. Detects your operating system
#   2. Installs Nix with flakes enabled (if not already installed)
#   3. Installs direnv for automatic environment loading (optional)
#   4. Verifies that bazel and go are available in the Nix environment
#   5. Provides next steps for development
#
# Requirements:
#   - curl (for downloading installers)
#   - Linux, macOS, or WSL2 (Nix does not run natively on Windows)
#
# For more information: https://nixos.org

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
SKIP_DIRENV=false
UNINSTALL=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-direnv)
            SKIP_DIRENV=true
            shift
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--skip-direnv] [--uninstall]"
            echo ""
            echo "Options:"
            echo "  --skip-direnv    Skip direnv installation"
            echo "  --uninstall      Uninstall Nix (use with caution)"
            echo "  -h, --help       Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

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

# Detect OS
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

# Check if we're in the repository root
check_repository() {
    if [[ ! -f "DESIGN.md" ]] || [[ ! -f "flake.nix" ]]; then
        error "Please run this script from the CodingGame repository root"
    fi
}

# Uninstall Nix (use with caution)
uninstall_nix() {
    header "Uninstalling Nix"

    warn "This will remove Nix and all packages installed with it."
    warn "This action cannot be undone."
    echo ""
    read -p "Are you sure you want to continue? (type 'yes' to confirm): " confirm

    if [[ "$confirm" != "yes" ]]; then
        info "Uninstall cancelled"
        exit 0
    fi

    if [[ ! -f "/nix/receipt.json" ]] && [[ ! -d "/nix" ]]; then
        info "Nix is not installed"
        exit 0
    fi

    info "Uninstalling Nix..."

    # Use official uninstall if available
    if [[ -f "/nix/receipt.json" ]]; then
        # Determinate Systems installer has an uninstall command
        if command -v nix &> /dev/null; then
            sudo /nix/nix-installer uninstall || true
        fi
    fi

    # Fallback: Manual cleanup
    sudo rm -rf /nix

    # Remove from PATH in shell configs
    for rc in ~/.bashrc ~/.zshrc ~/.profile; do
        if [[ -f "$rc" ]]; then
            sed -i.bak '/nix/d' "$rc" 2>/dev/null || true
        fi
    done

    success "Nix has been uninstalled"
    info "You may need to restart your terminal"
    exit 0
}

# Install Nix
install_nix() {
    header "Installing Nix Package Manager"

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
    warn "This installer is recommended by Nix and includes flakes support"
    echo ""
    warn "Press Ctrl+C within 5 seconds to cancel..."
    sleep 5

    # Use Determinate Systems installer (includes flakes by default)
    info "Downloading Nix installer..."
    if curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install; then
        success "Nix installed successfully"

        # Source nix for current session
        if [[ -f "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh" ]]; then
            . "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh"
        fi

        info "Nix has been added to your PATH"
        info "You may need to restart your terminal or run:"
        echo "  source /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh"
    else
        error "Failed to install Nix. Please install manually: https://nixos.org/download.html"
    fi

    # Verify installation
    if ! command -v nix &> /dev/null; then
        error "Nix installation completed but 'nix' command not found. Try restarting your terminal."
    fi
}

# Install direnv
install_direnv() {
    if [[ "$SKIP_DIRENV" == true ]]; then
        info "Skipping direnv installation (--skip-direnv flag set)"
        return
    fi

    header "Installing direnv (Optional)"

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

# Check if direnv hook is configured
check_direnv_hook() {
    local shell_rc=""

    if [[ -n "${BASH_VERSION:-}" ]]; then
        shell_rc="$HOME/.bashrc"
    elif [[ -n "${ZSH_VERSION:-}" ]]; then
        shell_rc="$HOME/.zshrc"
    else
        # Try to detect shell from $SHELL
        case "$SHELL" in
            */bash)
                shell_rc="$HOME/.bashrc"
                ;;
            */zsh)
                shell_rc="$HOME/.zshrc"
                ;;
            */fish)
                shell_rc="$HOME/.config/fish/config.fish"
                ;;
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

# Setup direnv hook
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

# Verify Nix environment
verify_environment() {
    header "Verifying Development Environment"

    info "Entering Nix development shell to verify tools..."
    echo ""

    # Create a temporary script to run inside nix develop
    local verify_script=$(mktemp)
    cat > "$verify_script" << 'EOF'
#!/usr/bin/env bash
set -e

echo "Checking Go..."
if command -v go &> /dev/null; then
    go_version=$(go version)
    echo "  ✓ $go_version"
else
    echo "  ✗ Go not found"
    exit 1
fi

echo "Checking Bazel..."
if command -v bazel &> /dev/null; then
    bazel_version=$(bazel version 2>&1 | head -1)
    echo "  ✓ $bazel_version"
else
    echo "  ✗ Bazel not found"
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
echo "All tools are available in the Nix environment!"
EOF
    chmod +x "$verify_script"

    # Run verification in nix develop
    if nix develop --command bash "$verify_script"; then
        rm "$verify_script"
        success "Environment verification passed"
    else
        rm "$verify_script"
        error "Environment verification failed"
    fi
}

# Allow direnv for this project
allow_direnv() {
    if [[ "$SKIP_DIRENV" == true ]]; then
        return
    fi

    if ! command -v direnv &> /dev/null; then
        return
    fi

    header "Configuring direnv for CodingGame"

    if [[ -f ".envrc" ]]; then
        info "Found .envrc file"

        # Check if already allowed
        if direnv status 2>/dev/null | grep -q "Found RC allowed true"; then
            success "direnv is already allowed for this project"
        else
            info "Allowing direnv for this project..."
            direnv allow
            success "direnv allowed - environment will load automatically"
        fi
    else
        warn ".envrc file not found - direnv will not work"
        warn "This is unusual. Check that you're in the CodingGame repository root."
    fi
}

# Print next steps
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

    echo ""
    echo -e "${BOLD}Learn more:${NC}"
    echo "  • BUILD.md - Detailed build instructions"
    echo "  • AGENTS.md - Agent workflow and guidelines"
    echo "  • DESIGN.md - Complete design specification"
    echo ""
    echo -e "${BOLD}Quick tips:${NC}"
    echo "  • Run 'bazel help' for Bazel commands"
    echo "  • Run 'nix flake show' to see available outputs"
    echo "  • Check NIX_BAZEL_QUICKREF.md for common tasks"
    echo ""
}

# Main execution
main() {
    header "CodingGame Development Environment Setup"

    # Handle uninstall
    if [[ "$UNINSTALL" == true ]]; then
        uninstall_nix
    fi

    # Check we're in the right place
    check_repository

    # Detect OS
    local os=$(detect_os)
    info "Detected OS: $os"

    # Install Nix
    install_nix

    # Install direnv (optional)
    install_direnv

    # Verify environment
    verify_environment

    # Allow direnv for this project
    allow_direnv

    # Print next steps
    print_next_steps
}

# Run main
main
