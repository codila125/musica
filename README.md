# MUSICA :: Retro Terminal Deck

```text
╔═══════════════════════════════════════════════════════════════╗
║  MUSICA                                                       ║
║  TUI music player for Navidrome + Jellyfin                    ║
║  Cassette vibes. Keyboard first. No mouse required.           ║
╚═══════════════════════════════════════════════════════════════╝
```

MUSICA is a terminal-based music player with a retro cassette UI, multi-server support,
and a clean architecture that separates UI, app orchestration, and API integrations.

## Features

- Retro TUI with `BROWSE`, `SEARCH`, and `QUEUE` tabs
- Works with both Navidrome and Jellyfin servers
- Runtime server switching (with playback stop + data refetch)
- Adaptive track table layout (song name prioritized on narrow terminals)
- Async request cancellation + stale-response protection
- Typed API error model and state-machine guarded transitions
- Test backend for mpv (`testmpv`) so CI can run without native `libmpv`

## Quick Start

### 1) Prerequisites

- Go `1.26+`
- `mpv` + `libmpv` (for normal runtime playback)

### 2) Build

```bash
go build -o musica ./cmd
```

### 3) Setup a server

```bash
./musica setup
```

### 4) Run

```bash
./musica
```

Use a specific server:

```bash
./musica --server my-server
```

## CLI Commands

```bash
musica setup
musica list
musica remove <server-name>
musica --server <server-name>
```

## Keybindings (TUI)

- Global
  - `tab` / `shift+tab`: switch main tabs
  - `s`: switch server
  - `ctrl+q` / `ctrl+c`: quit

- Browse
  - `j/k` or arrows: move
  - `enter` / `p`: play/pause selected track
  - `q`: add selected track to queue
  - `r`: refresh recent tracks

- Search
  - `enter`: search (input mode) / play (results mode)
  - `left/right` (or `h/l`): switch category
  - `p`: play/pause selected track
  - `q`: queue selected track
  - `esc`: back to input mode

- Queue
  - `j/k` or arrows: move
  - `enter` / `p`: play/pause selected queue track

## Architecture

```text
cmd/main.go
  -> internal/tui            (presentation + event loop)
  -> internal/app            (orchestration/controllers)
  -> internal/api/*          (Navidrome/Jellyfin adapters)
  -> internal/player         (mpv integration + test backend)
  -> internal/tui/views      (tab view models + renderers)
  -> internal/telemetry      (structured events/counters/timing)
```

### Notable modules

- `internal/app/coordinator.go`: server switch/connect orchestration
- `internal/app/playback_controller.go`: playback command orchestration
- `internal/tui/view_adapter.go`: centralized view lifecycle/update wiring
- `internal/tui/state.go`: app state machine + transition validation

## Development

### Run tests without native mpv

```bash
go test -tags=testmpv ./...
```

### Full local quality gates

Install once:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Run:

```bash
make ci
```

This runs formatting checks, vet, static analysis, vuln checks, unit tests, and race tests.

## CI

GitHub Actions workflow is in:

- `.github/workflows/ci.yml`

Quality gates are documented in:

- `docs/ci.md`

Reliability/fault-injection notes are in:

- `docs/reliability.md`

Config/security validation notes are in:

- `docs/config.md`

Operations and UX hardening notes are in:

- `docs/operations.md`

## License

No license file is currently present in this repository.
