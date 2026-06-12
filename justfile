# Directories gofmt should walk (excludes .agentic, vendored skills, etc.)
fmt_dirs := "cmd internal"

# Check formatting (exits 1 if any files need formatting)
fmt:
    @if [ -n "$(gofmt -l {{fmt_dirs}})" ]; then gofmt -l {{fmt_dirs}}; exit 1; fi

# Fix formatting
fmt-fix:
    gofmt -w {{fmt_dirs}}

# Run go vet
vet:
    go vet ./...

# Run unit tests
test:
    go test ./...

# Run a single test by name regex, optionally narrowed to a package
# Usage: just test-one TestResolveSystemUser
#        just test-one TestResolveSystemUser ./internal/cli
test-one name pkg="./...":
    go test {{pkg}} -run '{{name}}' -count=1 -v

# Run integration tests
test-integration:
    go test -tags=integration ./...

# Run all tests
test-all: test test-integration

# Build the binary
build:
    mkdir -p bin
    go build -o bin/worktree-manager ./cmd/worktree-manager

# Run the CLI
run *args:
    go run ./cmd/worktree-manager {{args}}

# Tidy dependencies
tidy:
    go mod tidy

# Fast pre-push check
gate: fmt vet test

# Full check
gate-expensive: gate test-integration
