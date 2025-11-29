# tts-cached

Generic caching front-end for the Piper CLI TTS engine. Accepts text over HTTP, normalizes and hashes it (with voice ID), caches WAV outputs on disk, enforces a size cap with LRU eviction, plays audio asynchronously, and gracefully shuts down on signals.

## Features
- HTTP API: `POST /tts` with `{"text":"..."}` and `GET /healthz`.
- Disk cache keyed by `sha256(VOICE_ID + "::" + normalizedText)`, modtime-based eviction after size cap.
- Calls local Piper executable; writes `.tmp` then atomically renames to avoid partial cache entries.
- Playback via external command (default `aplay`) without blocking HTTP responses.
- Graceful shutdown on SIGINT/SIGTERM.

## Configuration
Flags override env; env serves as defaults. Key settings:
- `-piper-model` / `PIPER_MODEL` (required): path to the ONNX model.
- `-piper-exec` / `PIPER_EXEC` (default `/usr/local/bin/piper`).
- `-piper-flags` / `PIPER_FLAGS`: extra Piper CLI args (space-separated).
- `-cache-dir` / `CACHE_DIR` (default `/var/cache/tts-cached`).
- `-listen-addr` / `LISTEN_ADDR` (default `127.0.0.1:4410`).
- `-play-cmd` / `PLAY_CMD` (default `/usr/bin/aplay`), `-play-args` / `PLAY_ARGS`.
- `-voice-id` / `VOICE_ID` (default `default`).
- `-cache-max-bytes` / `CACHE_MAX_BYTES` (default `536870912`).

Example:
```bash
PIPER_MODEL=/opt/piper/en_US.onnx \
CACHE_DIR=/var/cache/tts-cached \
LISTEN_ADDR=127.0.0.1:4410 \
./bin/tts-cached -piper-flags="-s 0"
```

## Build & Run
- Default build: `make build` (CGO disabled). Binary at `bin/tts-cached`.
- Cross-build arm64: `make build-linux-arm64` (CGO disabled).
- Run locally: `./bin/tts-cached -piper-model /path/to/model.onnx`.
- Tests: `make test`.

## HTTP Usage
```bash
curl -X POST http://127.0.0.1:4410/tts \
  -H "Content-Type: application/json" \
  -d '{"text":"hello world"}'
```
Responses:
- Cache miss: `{"status":"cache_miss","file":"<key>.wav"}`
- Cache hit: `{"status":"cache_hit","file":"<key>.wav"}`

Health:
```bash
curl http://127.0.0.1:4410/healthz
```

## Systemd Example
```ini
[Unit]
Description=Generic TTS cache front-end for Piper
After=network.target

[Service]
ExecStart=/usr/local/bin/tts-cached
Environment=PIPER_EXEC=/usr/local/bin/piper
Environment=PIPER_MODEL=/opt/piper/en_US.onnx
Environment=PIPER_FLAGS=-s 0
Environment=CACHE_DIR=/var/cache/tts-cached
Environment=LISTEN_ADDR=127.0.0.1:4410
Environment=CACHE_MAX_BYTES=536870912
Environment=VOICE_ID=default
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
Ensure the service user can execute Piper and write to `CACHE_DIR`.
