#!/usr/bin/env bash
# Generate .bazelrc.user with Nix store paths for X11 development
# This script must be run from within the Nix environment (nix develop)
#
# Usage: ./scripts/gen-bazelrc-user.sh

set -e

BAZELRC_USER=".bazelrc.user"

# Check we're in a Nix environment
if ! which nix >/dev/null 2>&1 || [ -z "$IN_NIX_SHELL" ] && [ -z "$NIX_BUILD_TOP" ]; then
    echo "Warning: Not in a Nix environment. Run 'nix develop' first." >&2
fi

echo "Generating $BAZELRC_USER..."

cat > "$BAZELRC_USER" << 'EOF'
# Auto-generated Nix X11 paths for Bazel CGO compilation
# These paths are machine-specific and should not be committed
# Regenerate with: ./scripts/gen-bazelrc-user.sh

EOF

# X11 libraries that need include paths (-dev packages)
X11_LIBS=(
    libx11
    libxext
    libxfixes
    libxrandr
    libxrender
    libxcursor
    libxinerama
    libxi
    libxxf86vm
    libglvnd
    alsa-lib
)

echo "# Include paths for compilation" >> "$BAZELRC_USER"

for lib in "${X11_LIBS[@]}"; do
    # Find the -dev package
    path=$(find /nix/store -maxdepth 1 -name "*-${lib}*-dev" -type d 2>/dev/null | head -1)
    if [ -n "$path" ] && [ -d "$path/include" ]; then
        echo "build --copt=-I${path}/include" >> "$BAZELRC_USER"
    fi
done

# xorgproto doesn't have a -dev suffix
xorgproto=$(find /nix/store -maxdepth 1 -name "*xorgproto*" -type d 2>/dev/null | grep -v "\.drv$" | head -1)
if [ -n "$xorgproto" ] && [ -d "$xorgproto/include" ]; then
    echo "build --copt=-I${xorgproto}/include" >> "$BAZELRC_USER"
fi

echo "" >> "$BAZELRC_USER"
echo "# Library paths for linking" >> "$BAZELRC_USER"

for lib in "${X11_LIBS[@]}"; do
    # Find the runtime library package (not -dev, not -doc)
    path=$(find /nix/store -maxdepth 1 -type d -name "*-${lib}-[0-9]*" 2>/dev/null | grep -v '\-dev$' | grep -v '\-doc$' | head -1)
    if [ -n "$path" ] && [ -d "$path/lib" ]; then
        echo "build --linkopt=-L${path}/lib" >> "$BAZELRC_USER"
    fi
done

echo "Generated $BAZELRC_USER with $(grep -c "^build" "$BAZELRC_USER") entries."
echo ""
echo "Contents:"
cat "$BAZELRC_USER"
