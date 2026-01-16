# Nix + Bazel Quick Reference

Quick reference for common Nix and Bazel commands for CodingGame development.

## Essential Commands

### Environment

```bash
# Enter Nix shell
nix develop

# With direnv (automatic)
direnv allow

# Exit Nix shell
exit

# Update flake inputs
nix flake update
```

### Build

```bash
# Build everything
bazel build //...

# Build main binary
bazel build //:codinggame

# Build specific package
bazel build //internal/tile

# Release build (optimized)
bazel build --config=release //:codinggame

# Debug build (with symbols)
bazel build --config=debug //:codinggame
```

### Run

```bash
# Run the game
bazel run //:codinggame

# Run with arguments
bazel run //:codinggame -- /path/to/project

# Run release build
bazel run --config=release //:codinggame
```

### Test

```bash
# Run all tests
bazel test //...

# Run specific test
bazel test //internal/tile:tile_test

# Verbose test output
bazel test //... --test_output=all

# Run tests with race detector
bazel test //... --features=race

# Run specific test with verbose output
bazel test //internal/claude:claude_test --test_output=streamed
```

### Clean

```bash
# Clean build artifacts
bazel clean

# Clean everything (including downloaded deps)
bazel clean --expunge

# Nix garbage collection
nix-collect-garbage -d
```

### Maintenance

```bash
# Update Go dependencies from go.mod
bazel run //:gazelle-update-repos

# Regenerate BUILD.bazel files
bazel run //:gazelle

# Format Bazel files
buildifier -r .

# Check Bazel file formatting
buildifier -mode=check -r .
```

## File Structure

```
CodingGame/
├── flake.nix              # Nix flake (environment definition)
├── .envrc                 # direnv configuration
├── WORKSPACE              # Bazel workspace (dependencies)
├── BUILD.bazel            # Root build file
├── .bazelrc               # Bazel configuration
├── .bazelrc.user          # User-specific config (gitignored)
├── .bazelversion          # Bazel version pin
├── go.mod                 # Go modules (also used by Bazel)
├── go.sum                 # Go checksums
└── internal/
    ├── game/BUILD.bazel
    ├── tile/BUILD.bazel
    ├── claude/BUILD.bazel
    ├── mapview/BUILD.bazel
    └── resources/BUILD.bazel
```

## Workflow Examples

### Adding a New Go File

```bash
# 1. Create the file
vim internal/newpackage/newfile.go

# 2. Regenerate BUILD.bazel
bazel run //:gazelle

# 3. Build and test
bazel build //internal/newpackage
bazel test //internal/newpackage:newpackage_test
```

### Adding a New Dependency

```bash
# 1. Add to go.mod
go get github.com/some/package

# 2. Update Bazel deps
bazel run //:gazelle-update-repos

# 3. Regenerate BUILD files
bazel run //:gazelle

# 4. Build to verify
bazel build //...
```

### Debugging Build Issues

```bash
# Verbose build output
bazel build //... --verbose_failures

# Show all commands being run
bazel build //... --subcommands

# Clean and rebuild
bazel clean && bazel build //...

# Check dependency tree
bazel query 'deps(//:codinggame)'

# Find why a file is being rebuilt
bazel build //:codinggame --explain=explain.txt --verbose_explanations
```

### Optimizing Build Performance

```bash
# Enable disk cache (add to .bazelrc.user)
echo "build --disk_cache=~/.cache/bazel" >> .bazelrc.user

# Use all CPU cores
echo "build --jobs=auto" >> .bazelrc.user

# Show build performance
bazel build //... --profile=profile.json
bazel analyze-profile profile.json
```

## Nix Shell Customization

### Custom Environment Variables

Edit `flake.nix` shellHook:

```nix
shellHook = ''
  export MY_VAR="value"
  export PATH="$PWD/scripts:$PATH"
'';
```

### Additional Tools

Edit `flake.nix` buildInputs:

```nix
buildInputs = buildInputs ++ [
  pkgs.ripgrep
  pkgs.fd
];
```

## Bazel Configuration Tips

### .bazelrc.user Examples

```bash
# Faster builds
build --jobs=auto
build --disk_cache=~/.cache/bazel

# Always show test output
test --test_output=streamed

# Use Nix config by default (if in Nix environment)
build --config=nix

# Colored output
common --color=yes

# Build with debug symbols
build --copt=-g
```

## Troubleshooting

### Bazel not found

```bash
# Make sure you're in Nix shell
nix develop
```

### Build fails with "cannot find package"

```bash
# Regenerate dependencies
bazel run //:gazelle-update-repos
bazel run //:gazelle
```

### Nix flake evaluation fails

```bash
# Update flake lock
nix flake update

# Check flake
nix flake check
```

### Slow initial build

This is normal. Bazel downloads and caches dependencies on first build. Subsequent builds are much faster.

### X11 errors on Linux

Make sure you're in the Nix shell, which provides X11 libraries automatically:

```bash
nix develop
bazel build //:codinggame
```

## Advanced

### Query Targets

```bash
# List all targets
bazel query //...

# List all tests
bazel query 'tests(//...)'

# Show dependencies of a target
bazel query 'deps(//:codinggame)'

# Show reverse dependencies
bazel query 'rdeps(//..., //internal/tile)'
```

### Remote Execution / Caching (Team Setup)

```bash
# Add to .bazelrc.user
build --remote_cache=https://cache.example.com
build --remote_executor=grpc://executor.example.com
```

### Build Event Protocol (BEP)

```bash
# Generate build events
bazel build //... --build_event_json_file=events.json

# Analyze events
cat events.json | jq '.testSummary'
```

## Resources

- [Bazel User Guide](https://bazel.build/docs)
- [rules_go](https://github.com/bazelbuild/rules_go)
- [Gazelle](https://github.com/bazelbuild/bazel-gazelle)
- [Nix Manual](https://nixos.org/manual/nix/stable/)
- [direnv](https://direnv.net/)

---

**Pro Tip**: Use `direnv` for automatic environment loading. Just `cd` into the project and everything is set up!
