{
  description = "CodingGame - Strategy game interface for Claude Code";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # X11 libraries needed for Ebitengine on Linux
        x11Libs = with pkgs; [
          xorg.libX11
          xorg.libXrandr
          xorg.libXcursor
          xorg.libXinerama
          xorg.libXi
          xorg.libXxf86vm
          libGL
          alsa-lib
        ];

        # Build inputs
        buildInputs = [
          pkgs.go_1_24
          pkgs.bazel_7
          pkgs.buildifier
          pkgs.buildozer
          pkgs.pkg-config
        ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux x11Libs;

        # Development tools
        nativeBuildInputs = with pkgs; [
          gopls
          gotools
          go-tools
          delve
          git
        ];

      in
      {
        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = buildInputs ++ nativeBuildInputs;

          shellHook = ''
            echo "🎮 CodingGame Development Environment"
            echo "======================================"
            echo ""
            echo "Available commands:"
            echo "  bazel build //...        - Build all targets"
            echo "  bazel test //...         - Run all tests"
            echo "  bazel run //:codinggame  - Run the game"
            echo "  go build                 - Build with Go modules"
            echo ""
            echo "Tools:"
            echo "  Go:        $(go version | cut -d' ' -f3)"
            echo "  Bazel:     $(bazel version | head -1)"
            echo ""

            # Set up environment for X11 on Linux
            ${pkgs.lib.optionalString pkgs.stdenv.isLinux ''
              export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath x11Libs}:$LD_LIBRARY_PATH"
            ''}

            # Bazel configuration
            export BAZEL_USE_CPP_ONLY_TOOLCHAIN=1
          '';

          # Environment variables
          CGO_ENABLED = "1";
        };

        # Package definition for building CodingGame
        packages.default = pkgs.buildGoModule {
          pname = "codinggame";
          version = "0.1.0-phase1";
          src = ./.;

          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # Will be updated after first build

          buildInputs = x11Libs;
          nativeBuildInputs = [ pkgs.pkg-config ];

          meta = with pkgs.lib; {
            description = "Strategy game interface for Claude Code";
            homepage = "https://github.com/tedks/CodingGame";
            license = licenses.mit;
            maintainers = [];
          };
        };

        # Formatter for Nix files
        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
