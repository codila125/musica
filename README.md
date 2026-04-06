# MUSICA

Retro terminal music player for Navidrome and Jellyfin.

MUSICA is a keyboard-first TUI app with cassette-deck vibes. It runs in your terminal, lets you browse and search your library, and manage a playback queue without a mouse.

## Features

- Retro terminal UI with `BROWSE`, `SEARCH`, and `QUEUE` tabs
- Works with both Navidrome and Jellyfin
- Fast keyboard navigation and queue-based playback
- Runtime server switching
- Clean first-run flow with guided setup

## Install

### Homebrew (recommended)

```bash
brew tap <you>/tap
brew install musica
```

### From source

Prerequisites:

- Go `1.26+`
- `mpv` + `libmpv`

Build:

```bash
go build -o musica ./cmd
```

## First-time setup

Run:

```bash
musica setup
```

Then start the player:

```bash
musica
```

If no servers are configured yet, MUSICA exits cleanly and shows:

```text
No servers configured. Run: musica setup
```

## Basic commands

```bash
musica setup
musica list
musica remove <server-name>
musica --server <server-name>
```

## Keybindings

- Global
  - `tab` / `shift+tab`: switch tabs
  - `s`: switch server
  - `ctrl+q` / `ctrl+c`: quit
- Browse
  - `j/k` or arrows: move
  - `enter` / `p`: play/pause selected track
  - `q`: add selected track to queue
  - `r`: refresh recent tracks
- Search
  - `enter`: search (input mode) / play (results mode)
  - `left/right` or `h/l`: switch category
  - `p`: play/pause selected track
  - `q`: queue selected track
  - `esc`: back to input mode
- Queue
  - `j/k` or arrows: move
  - `enter` / `p`: play/pause selected queue track

## Troubleshooting

### `libmpv` load error on startup

Install `mpv` first, then ensure your loader path includes mpv libs.

On macOS (Homebrew):

```bash
export DYLD_FALLBACK_LIBRARY_PATH="$(brew --prefix mpv)/lib${DYLD_FALLBACK_LIBRARY_PATH:+:$DYLD_FALLBACK_LIBRARY_PATH}"
```

On Linux:

```bash
export LD_LIBRARY_PATH="<mpv-lib-dir>${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
```

If you are using `run.sh` locally, it already tries to detect and set these paths automatically.

## Documentation

- Developer guide: `docs/developer.md`
- CI and quality gates: `docs/ci.md`
- Reliability notes: `docs/reliability.md`
- Config and validation: `docs/config.md`
- Operations/UX hardening: `docs/operations.md`
