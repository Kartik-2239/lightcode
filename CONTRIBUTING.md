# Contributing

Contributions are welcome!

## Getting started

1. Fork the repository and clone your fork.
2. Install Go **1.25+**.
3. Build and run from source:
   ```bash
   go run ./cmd/lightcode/main.go
   ```

## Before opening a PR

- `go build ./...` passes.
- `go vet ./...` is clean.
- `go test ./...` passes and add tests for any new behavior.

## Reporting issues

Open an issue and specify:
- what happened?
- what was the expected behavior?
- your operating system and lightcode version
