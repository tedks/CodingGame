workspace(name = "com_github_tedks_codinggame")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Go rules
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "f4a9314518ca6acfa16cc4ab43b0b8ce1e4ea64b81c38d8a3772883f153346b8",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.50.1/rules_go-v0.50.1.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.50.1/rules_go-v0.50.1.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "b760f7fe75173886007f7c2e616a21241208f3d90e8657dc65d36a771e916b6a",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.1/bazel-gazelle-v0.39.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.1/bazel-gazelle-v0.39.1.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

# Go dependencies
go_rules_dependencies()

go_register_toolchains(version = "1.24.12")

gazelle_dependencies()

# External Go dependencies
load("@bazel_gazelle//:deps.bzl", "go_repository")

go_repository(
    name = "com_github_hajimehoshi_ebiten_v2",
    importpath = "github.com/hajimehoshi/ebiten/v2",
    sum = "h1:WuNgM24uJxwdLZLqM8SXLAGVBof/45udRjo2tJoTpM0=",
    version = "v2.9.7",
)

go_repository(
    name = "com_github_ebitengine_gomobile",
    importpath = "github.com/ebitengine/gomobile",
    sum = "h1:JqpCb/KkQf3XbLqG5tnT4wY5PmzTvYmO/iQUfOdwZH8=",
    version = "v0.0.0-20250923094054-ea854a63cce1",
)

go_repository(
    name = "com_github_ebitengine_purego",
    importpath = "github.com/ebitengine/purego",
    sum = "h1:mh0zpKBIXDceC63hpvPuGLiJ8ZAa3DfrFTudmfi8A4k=",
    version = "v0.9.0",
)

go_repository(
    name = "com_github_ebitengine_hideconsole",
    importpath = "github.com/ebitengine/hideconsole",
    sum = "h1:5J4U0kXF+pv/DhiXt5/lTz0eO5ogJ1iXb8Yj1yReDqE=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_jezek_xgb",
    importpath = "github.com/jezek/xgb",
    sum = "h1:bE/r8ZZtSv7l9gk6nU0mYx51aXrvnyb44892TwSaqS4=",
    version = "v1.1.1",
)

go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    sum = "h1:Auo3B8PjA9JN//iI2vWWHI5s9LdxVwP6Uik8LFmSjz4=",
    version = "v0.17.0",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    sum = "h1:YtRBw5Z4ePy5kjWXPGvSRBP4Dg21Hqk/oBvD0kSw/hw=",
    version = "v0.36.0",
)
