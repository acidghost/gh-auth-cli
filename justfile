program := 'gh-auth-cli'

version := 'SNAPSHOT-'+`git describe --tags --always --dirty 2>/dev/null || printf 'unknown'`
commit_sha := `(git rev-parse HEAD 2>/dev/null || printf 'unknown') | tr -d '\n'`
build_time := `date -u '+%Y-%m-%d_%H:%M:%S'`

ldflags := '-s -w -X main.buildVersion='+version \
        +' -X main.buildCommit='+commit_sha \
        +' -X main.buildDate='+build_time

install_prefix := `go env GOBIN`

goos := if os() == 'macos' { 'darwin' } else { os() }
goarch := if arch() == 'aarch64' { 'arm64' } else if arch() == 'x86_64' { 'amd64' } else { arch() }

alias b := build
alias r := run

# Build binaries for macOS arm64 and Linux arm64/amd64.
build-all: (build 'darwin' 'arm64') (build 'linux' 'arm64') (build 'linux' 'amd64')

# Build a binary for the selected OS and architecture (defaults to the host).
build os=goos arch=goarch: build-dir
    CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} \
        go build \
            -ldflags '{{ldflags}}' \
            -o build/{{program}}-{{os}}-{{arch}}

# Create the build output directory.
build-dir:
    mkdir -p build

# Build and run the host binary with the supplied arguments.
run *args: build
    ./build/{{program}}-{{goos}}-{{goarch}} {{args}}

# Tidy Go modules and refresh vendored dependencies.
vendor:
    go mod tidy
    go mod vendor

# Format Go source files.
fmt:
    golangci-lint fmt

# Run the Go linter.
lint:
    golangci-lint run

# Run all Go tests.
test:
    go test ./...

# Validate the GoReleaser configuration.
release-check:
    goreleaser check

# Build all release archives locally without publishing.
release-snapshot:
    goreleaser release --snapshot --clean --skip=publish

# Publish a tagged release (requires GITHUB_TOKEN).
release:
    goreleaser release --clean

# Build and install the host binary into GOBIN.
install: build
    cp -v './build/{{program}}-{{goos}}-{{goarch}}' "{{install_prefix}}/{{program}}"

# Remove the build output directory.
clean:
    rm -rf build

# List available recipes.
help:
    @just --list
