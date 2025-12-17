# Agent Guidelines for Sebaschtian the Fish

## Commands
- **Build**: `go build ./cmd/fish`, `go build ./cmd/sounds`, `go build ./cmd/balena-monitor`
- **Test (All)**: `go test ./...`
- **Test (Single)**: `go test -v -run TestName ./pkg/package`
- **Lint**: `go vet ./...`

## Code Style & Conventions
- **Language**: Go 1.24+. Use standard `gofmt` formatting.
- **Logging**: Use `log/slog` exclusively. DO NOT use `fmt.Print` or `log` package directly.
- **Telemetry**: Use `pkg/telemetry` to initialize tracing/metrics. Pass `context.Context` for tracing.
- **Error Handling**: Explicit `if err != nil`. Wrap errors with `fmt.Errorf("...: %w", err)`.
- **Imports**: Group stdlib first, then 3rd-party, then local (`github.com/wachiwi/sebaschtian-the-fish/...`).
- **Configuration**: Use environment variables.
- **Structure**: `cmd/` for entrypoints, `pkg/` for library code.
- **Platform**: Handles Linux (RPi) and Darwin (macOS) via build tags (`//go:build linux`, `//go:build darwin`).
