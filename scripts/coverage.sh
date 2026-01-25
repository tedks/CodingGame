#!/usr/bin/env bash
# Generate Go test coverage report
# Usage: ./scripts/coverage.sh [--html] [--summary]
#
# Outputs:
#   coverage.out      - Raw coverage profile
#   coverage.txt      - Function-level coverage report
#   coverage.html     - HTML report (if --html specified)
#
# Requires: xvfb-run for GUI tests (apt-get install xvfb)

set -euo pipefail

COVERAGE_OUT="${COVERAGE_OUT:-coverage.out}"
COVERAGE_TXT="${COVERAGE_TXT:-coverage.txt}"
COVERAGE_HTML="${COVERAGE_HTML:-coverage.html}"
GENERATE_HTML=false
SUMMARY_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --html) GENERATE_HTML=true; shift ;;
        --summary) SUMMARY_ONLY=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Detect if we need xvfb (no DISPLAY and xvfb-run available)
RUN_PREFIX=""
if [[ -z "${DISPLAY:-}" ]] && [[ -z "${WAYLAND_DISPLAY:-}" ]]; then
    if command -v xvfb-run &> /dev/null; then
        RUN_PREFIX="xvfb-run -a -s \"-screen 0 1024x768x24 -ac\""
        echo "Running with xvfb (headless mode)"
    else
        echo "Warning: No display available and xvfb-run not found."
        echo "GUI package tests will be skipped."
    fi
fi

# All packages to test
# Note: Some packages require display (game, mapview, tile, ui, testutil, systemtest)
# We run them all and let xvfb handle it, or they'll skip gracefully
ALL_PACKAGES="./internal/..."

echo "Generating coverage profile..."
if [[ -n "$RUN_PREFIX" ]]; then
    eval "$RUN_PREFIX go test -coverprofile=\"$COVERAGE_OUT\" -coverpkg=\"$ALL_PACKAGES\" \"$ALL_PACKAGES\""
else
    go test -coverprofile="$COVERAGE_OUT" -coverpkg="$ALL_PACKAGES" "$ALL_PACKAGES"
fi

echo ""
echo "Generating coverage report..."
go tool cover -func="$COVERAGE_OUT" > "$COVERAGE_TXT"

if [[ "$SUMMARY_ONLY" == "true" ]]; then
    # Just output total
    grep "^total:" "$COVERAGE_TXT"
else
    # Show files with low coverage (< 50%)
    echo ""
    echo "=== Low Coverage Files (< 50%) ==="
    awk -F'\t' '$NF ~ /%/ {
        gsub(/%/, "", $NF);
        if ($NF+0 < 50) print $0
    }' "$COVERAGE_TXT" | head -30

    echo ""
    echo "=== Coverage Summary ==="
    grep "^total:" "$COVERAGE_TXT"
fi

if [[ "$GENERATE_HTML" == "true" ]]; then
    echo ""
    echo "Generating HTML report: $COVERAGE_HTML"
    go tool cover -html="$COVERAGE_OUT" -o "$COVERAGE_HTML"
fi

# Output for CI parsing
TOTAL=$(grep "^total:" "$COVERAGE_TXT" | awk '{print $NF}')
echo ""
echo "COVERAGE_TOTAL=$TOTAL"
