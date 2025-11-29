# Repository Guidelines

## Project Structure & Responsibilities
- Go module with service entrypoint at `cmd/tts-cached/main.go`; supporting packages under `internal/` (config, server/handlers, cache, piperclient, audio).
- Cache artifacts live on disk (default `/var/lib/tts-cached`); never commit generated `.wav` or `.tmp` files.
- Tests co-located with code (`*_test.go`); keep fixtures small and in-memory where possible.

## Build, Test, and Development Commands
- `go mod init github.com/venkytv/tts-cached` then `go mod tidy` to set up the module.
- `go fmt ./...` format; `go vet ./...` static checks.
- `go test ./...` run unit tests; add `-race` when feasible on dev machines.
- `go build ./cmd/tts-cached` produce the binary; `go run ./cmd/tts-cached` for local runs.

## Coding Style & Naming Conventions
- Standard Go style; rely on `gofmt` (tabs, 4-space display) and idiomatic naming (`CamelCase` for exported, `lowerCamel` for internal).
- Keep packages focused and small; avoid cross-package globals.
- Prefer context-aware functions and explicit timeouts for I/O (HTTP to Piper, playback commands).
- Log with the standard `log` package; INFO for flow events, ERROR for failures.

## Testing Guidelines
- Use the standard library `testing` package; table-driven tests where useful.
- Stub external calls (Piper HTTP, `exec.Command`) via interfaces so tests stay fast and hermetic.
- Name tests `TestFunctionality` and keep temporary files under `t.TempDir()`.

## Commit & Pull Request Guidelines
- Commit messages: short imperative summaries (e.g., `Add cache eviction`); detailed commit message; group related changes; format multi-line messages with a blank line after the summary; use bullet points for details; line length ~72 chars. IMPORTANT: Make sure the comments are formatted in this way:
    ```
    Short summary line

    - Detail point one
    - Detail point two
    - Long lines should be wrapped appropriately to maintain readability
      and wrapped at around 72 characters.
    ```
  - IMPORTANT: Multi-line commits messages are NOT created using `-m` for each line. Use a `-m` for the summary and a single `-m` with newlines for the details.
- PRs: include purpose, key implementation notes, config/env vars touched, and test evidence (`go test ./...`). Add logs/output only when it clarifies behaviour.

## Security & Configuration Tips
- Environment-driven configuration with defaults: `PIPER_EXEC`, `PIPER_MODEL` (required), `PIPER_FLAGS`, `CACHE_DIR`, `LISTEN_ADDR`, `PLAY_CMD`, `PLAY_ARGS`, `VOICE_ID`, `CACHE_MAX_BYTES`.
- Ensure `CACHE_DIR` is writable by the service user; avoid world-writable permissions.
- Piper executable is assumed local; when using shared models/configs, keep permissions limited to the service user.

## Operational Notes
- Service should handle SIGINT/SIGTERM with graceful HTTP shutdown; avoid blocking on playback.
- Cache enforcement must ignore partial `.tmp` files and delete oldest `.wav` first when over `CACHE_MAX_BYTES`.
