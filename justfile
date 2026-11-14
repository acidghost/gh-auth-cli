program := 'gh-auth-cli'

install_prefix := `go env GOBIN`

goos := if os() == 'macos' { 'darwin' } else { os() }
goarch := if arch() == 'aarch64' { 'arm64' } else if arch() == 'x86_64' { 'amd64' } else { arch() }

alias b := build
alias r := run

# Build binaries for all GoReleaser targets into dist/.
build-all:
    goreleaser build --snapshot --clean

# Build a binary for the selected OS and architecture (defaults to the host).
build os=goos arch=goarch: build-dir
    GOOS={{os}} GOARCH={{arch}} goreleaser build --snapshot --single-target --clean --output build/{{program}}

# Create the build output directory.
build-dir:
    mkdir -p build

# Build and run the host binary with the supplied arguments.
run *args: build
    ./build/{{program}} {{args}}

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
    cp -v './build/{{program}}' "{{install_prefix}}/{{program}}"

# Remove local build and release output directories.
clean:
    rm -rf build dist

# List available recipes.
help:
    @just --list
