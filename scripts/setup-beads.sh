#!/usr/bin/env bash
#
# setup-beads.sh - Install and configure beads for this repository
#
# Beads is Steve Yegge's memory system for coding agents. It provides
# persistent, structured task tracking using Git + JSONL + SQLite.
#
# Usage:
#   ./scripts/setup-beads.sh
#   ./scripts/setup-beads.sh --no-daemon  # For web Claude Code environments
#
# What this script does:
#   1. Installs the bd CLI if not already installed
#   2. Initializes beads in the repository (if not already initialized)
#   3. Installs git hooks for automatic sync
#   4. Sets up Claude Code integration (if Claude Code is detected)
#
# For more information: https://github.com/steveyegge/beads

set -euo pipefail

# Handle --no-daemon flag for web Claude Code environments
NO_DAEMON=""
if [[ "${1:-}" == "--no-daemon" ]]; then
    NO_DAEMON="1"
    export BEADS_NO_DAEMON=1
    echo "Running in no-daemon mode (for web Claude Code environments)"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Check if we're in the repository root
if [[ ! -f "DESIGN.md" ]]; then
    error "Please run this script from the repository root"
fi

# Step 1: Check if bd is installed
info "Checking for bd CLI..."
if command -v bd &> /dev/null; then
    BD_VERSION=$(bd --version 2>/dev/null | head -1 || echo "unknown")
    info "bd is already installed: $BD_VERSION"
else
    info "Installing bd CLI..."
    curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

    # Verify installation
    if ! command -v bd &> /dev/null; then
        error "Failed to install bd. Please install manually: https://github.com/steveyegge/beads"
    fi
    info "bd installed successfully"
fi

# Step 2: Initialize beads if not already initialized
if [[ -d ".beads" ]]; then
    info "Beads already initialized in this repository"
else
    info "Initializing beads..."
    bd init
    info "Beads initialized"
fi

# Step 3: Install git hooks
info "Installing git hooks..."
bd hooks install
info "Git hooks installed"

# Step 4: Set up Claude Code integration if available
if [[ -d "$HOME/.claude" ]]; then
    info "Claude Code detected, setting up integration..."
    bd setup claude
    info "Claude Code integration configured"
    warn "Restart Claude Code for hook changes to take effect"
else
    info "Claude Code not detected, skipping integration setup"
fi

# Step 5: Run doctor to check for any remaining issues
info "Running bd doctor to verify setup..."
bd doctor || true

echo ""
info "Beads setup complete!"
echo ""
echo "Quick start:"
echo "  bd ready              # Find available work"
echo "  bd add \"task\"         # Create an issue"
echo "  bd show <id>          # View issue details"
echo "  bd close <id>         # Complete work"
echo "  bd sync               # Sync with git"
echo ""
echo "For more info: bd help"
echo ""
if [[ -n "$NO_DAEMON" ]]; then
    echo "NOTE: Running in no-daemon mode. Remember to prefix bd commands with:"
    echo "  export BEADS_NO_DAEMON=1"
else
    echo "TIP: If running in web Claude Code, use: export BEADS_NO_DAEMON=1"
fi
