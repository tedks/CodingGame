#!/usr/bin/env bash
# Generate .bazelrc.user with Nix store paths for graphics development (X11, Wayland, EGL)
# This script must be run from within the Nix environment (nix develop)
#
# Usage: ./scripts/gen-bazelrc-user.sh [--quiet]
#
# Options:
#   --quiet    Only output if changes were made

set -e

BAZELRC_USER=".bazelrc.user"
QUIET=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --quiet|-q)
            QUIET=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Check we're in a Nix environment
if [ -z "$C_INCLUDE_PATH" ] && ! command -v pkg-config >/dev/null 2>&1; then
    echo "Error: Not in a Nix environment. Run 'nix develop' first." >&2
    exit 1
fi

# Generate content to a temporary variable first
generate_content() {
    cat << 'EOF'
# Auto-generated Nix graphics paths for Bazel CGO compilation (X11, Wayland, EGL)
# These paths are machine-specific and should not be committed
# Regenerate with: ./scripts/gen-bazelrc-user.sh
# Or enter nix develop (auto-regenerates on entry)

EOF

    echo "# Include paths for compilation"

    # Use C_INCLUDE_PATH from Nix environment (most reliable)
    # This is set by the Nix shell hook and contains all -dev package includes
    if [ -n "$C_INCLUDE_PATH" ]; then
        IFS=':' read -ra INCLUDE_PATHS <<< "$C_INCLUDE_PATH"
        for path in "${INCLUDE_PATHS[@]}"; do
            if [ -n "$path" ] && [ -d "$path" ]; then
                echo "build --copt=-I${path}"
            fi
        done
    fi

    # Also add pkg-config paths as fallback (may add some duplicates, but that's OK)
    if command -v pkg-config >/dev/null 2>&1; then
        PKG_CONFIG_LIBS=(x11 xext xfixes xrandr xrender xcursor xinerama xi xxf86vm gl egl wayland-client xkbcommon alsa)
        declare -A seen_includes

        for lib in "${PKG_CONFIG_LIBS[@]}"; do
            if pkg-config --exists "$lib" 2>/dev/null; then
                cflags=$(pkg-config --cflags "$lib" 2>/dev/null || true)
                for flag in $cflags; do
                    if [[ $flag == -I* ]]; then
                        path="${flag#-I}"
                        if [ -z "${seen_includes[$path]}" ] && [ -d "$path" ]; then
                            seen_includes["$path"]=1
                            echo "build --copt=-I${path}"
                        fi
                    fi
                done
            fi
        done
    fi

    echo ""
    echo "# Library paths for linking"

    # Use pkg-config for library paths (more reliable than scanning)
    if command -v pkg-config >/dev/null 2>&1; then
        PKG_CONFIG_LIBS=(x11 xext xfixes xrandr xrender xcursor xinerama xi xxf86vm gl egl wayland-client xkbcommon alsa)
        declare -A seen_libs

        for lib in "${PKG_CONFIG_LIBS[@]}"; do
            if pkg-config --exists "$lib" 2>/dev/null; then
                libs=$(pkg-config --libs-only-L "$lib" 2>/dev/null || true)
                for flag in $libs; do
                    if [[ $flag == -L* ]]; then
                        path="${flag#-L}"
                        if [ -z "${seen_libs[$path]}" ] && [ -d "$path" ]; then
                            seen_libs["$path"]=1
                            echo "build --linkopt=-L${path}"
                        fi
                    fi
                done
            fi
        done
    fi
}

# Generate new content
NEW_CONTENT=$(generate_content)

# Check if file exists and compare hash
if [ -f "$BAZELRC_USER" ]; then
    OLD_HASH=$(md5sum "$BAZELRC_USER" 2>/dev/null | cut -d' ' -f1 || echo "")
    NEW_HASH=$(echo "$NEW_CONTENT" | md5sum | cut -d' ' -f1)

    if [ "$OLD_HASH" = "$NEW_HASH" ]; then
        # No changes needed
        if [ "$QUIET" = false ]; then
            echo "$BAZELRC_USER is up to date."
        fi
        exit 0
    fi
fi

# Write new content
echo "$NEW_CONTENT" > "$BAZELRC_USER"

# Report changes
ENTRY_COUNT=$(grep -c "^build" "$BAZELRC_USER" || echo 0)
if [ "$QUIET" = false ]; then
    echo "Generated $BAZELRC_USER with $ENTRY_COUNT entries."
    echo ""
    echo "Contents:"
    cat "$BAZELRC_USER"
else
    echo "Regenerated $BAZELRC_USER (paths updated for this machine)"
fi
