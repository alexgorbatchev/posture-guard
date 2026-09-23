# Default recipe: list available tasks
default:
    @just --list

# Build compiled binary strictly into bin/
build:
    @mkdir -p bin
    go build -o bin/posture-guard ./cmd/posture-guard

# Run in human-facing interactive mode
run *args:
    go run ./cmd/posture-guard {{args}}

# Run in agent-facing token-conservative mode
run-ai *args:
    AGENT=1 go run ./cmd/posture-guard {{args}}

# Run all unit and integration tests
test:
    go test ./...

# Check module hygiene and static analysis
lint:
    go mod tidy -diff
    go vet ./...

# Clean build artifacts
clean:
    rm -rf bin/
