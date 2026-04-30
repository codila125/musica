# MUSICA

MUSICA is a premium-feeling, keyboard-first cassette deck for your music library. It turns your terminal into a retro hi-fi with instant browsing, crisp search, and a queue that behaves like a real tape.

Built for Navidrome and Jellyfin. Tuned for speed. Designed to feel like hardware.

## Why MUSICA

- **Cassette-deck experience**: polished retro visuals, animated tape motion, and a living UI.
- **Keyboard-first flow**: play, queue, replay, and navigate with zero mouse friction.
- **Queue that makes sense**: add tracks to play next, not “sometime later.”
- **Fast discovery**: browse recent tracks or search by track/album/artist.
- **Multi-server ready**: switch servers on the fly.

## Highlights

- Tabs: `BROWSE`, `SEARCH`, `QUEUE`
- Smooth playback timeline and on-screen audio bars
- Works in any modern terminal

## Install (Homebrew)

```bash
brew tap codile125/tap
brew install musica
```

Verify and run:

```bash
musica help
musica setup
musica
```

Update:

```bash
brew update
brew upgrade musica
```

Uninstall:

```bash
brew uninstall musica
brew untap codile125/tap
```

## First-time setup

```bash
musica setup
```

Then launch:

```bash
musica
```

If no servers are configured:

```text
No servers configured. Run: musica setup
```

## Commands

```bash
musica setup
musica list
musica remove <server-name>
musica --server <server-name>
```

## Keybindings (essentials)

- Global
  - `tab` / `shift+tab`: switch tabs
  - `ctrl+h`: help
  - `ctrl+s`: switch server
  - `ctrl+q`: quit

- Playback
  - `p` / `enter`: play/pause
  - `n`: next track
  - `m`: previous track
  - `r`: replay
  - `q`: queue after current

- Browse
  - `j/k` or arrows: move
  - `←/→`: previous/next page

- Search
  - `enter`: search (input) / play (results)
  - `esc`: back to input
  - `←/→` or `h/l`: switch category

## Troubleshooting

### mpv/libmpv not found

```bash
brew install mpv
```

macOS fallback path:

```bash
export DYLD_FALLBACK_LIBRARY_PATH="$(brew --prefix mpv)/lib${DYLD_FALLBACK_LIBRARY_PATH:+:$DYLD_FALLBACK_LIBRARY_PATH}"
```

Linux fallback path:

```bash
export LD_LIBRARY_PATH="<mpv-lib-dir>${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
```

## Docs

- Developer guide: `docs/developer.md`
- CI and quality gates: `docs/ci.md`
- Reliability notes: `docs/reliability.md`
- Config and validation: `docs/config.md`
- Operations/UX hardening: `docs/operations.md`
