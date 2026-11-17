# AzerMind Go client (experimental)

This is an incremental Go replacement for the Node bridge. The current Lua addon and the Node bridge are unchanged. **Do not run the Go and Node clients at the same time**: both read/write the same transport slots and state files.

## Supported so far

- Reads the existing `bridge/config.json` and the Lua SavedVariables fallback.
- Receives the existing pixel-strip framing from `bridge/capture.ps1` (Windows), `capture_mac.py` (macOS) or `capture_x11.py` (Linux). Reassembles multipart records.
- Writes `AzerMind_Inbox`, reply slots, ACK, SIG, action and presence WAV files; creates missing slots idempotently with `--install-slots`.
- Runs Claude, Codex, Grok, Agy and Hermes CLIs with configurable model, reasoning effort, permissions, project folders, timeout and basic session resume. Streams progress and publishes final replies.
- Handles multiple chats, queuing, cancellation, deduplication and state persistence, including legacy numeric handled-ID format.
- Exposes model choices from local Codex cache and compatibility defaults. Supports game-side default model configuration and on-demand Grok/Agy model discovery.

## Remaining parity work (not production-ready)

- Full transcript restoration after addon data resets, map routes, macros, Wowhead reference enrichment and exact system-prompt parity.
- Full provider model-catalog caching, precise CLI event/tool/permission parsers for every agent and compatibility with all Windows npm shims.
- Complete installer/migration (the Go `--install-slots` only creates transport files), standalone packaging of capture scripts and addon resources, Windows-in-game tests, release automation and zero-Node first-time setup.
- Restart/supervision and robust ownership locking between the Node and Go bridge.

## Build and test

Requires Go 1.22+:

```sh
cd client-go
go test ./...
go test -race ./...             # Linux/macOS
go vet ./...
GOOS=windows GOARCH=amd64 go build -o AzerMind.exe ./cmd/azermind
```

Run from the repository root (first install the addon and slots using the existing setup flow):

```sh
./client-go/AzerMind.exe --config bridge/config.json
./client-go/AzerMind.exe --config bridge/config.json --project path/to/project
./client-go/AzerMind.exe --config bridge/config.json --once
./client-go/AzerMind.exe --config bridge/config.json --inject "Hello" --agent codex
./client-go/AzerMind.exe --config bridge/config.json --install-slots
```

The Go binary invokes the configured agent CLIs, not Node.js. Capturing still requires the platform capture scripts from `bridge/`. `state.json`, `Inbox.lua` and slot filenames intentionally remain compatible with the Node implementation. Back up your state/transcripts before switching clients, and keep the Node implementation as the rollback path.

Tests use fake WoW folders and a mocked CLI; passing tests do **not** demonstrate that every real agent and the game client work on Windows.
