# Operations & UX Hardening (Phases 14 and 18)

## Phase 14: Operational Readiness

- player startup now shows actionable hint when `libmpv`/headers are missing
- graceful signal handling (`SIGINT`, `SIGTERM`) added in `cmd/main.go`
  - on signal: stop playback and close player cleanly

## Phase 18: UX Hardening

- search `esc` now cancels in-flight request and leaves loading state immediately
- queue cursor is clamped safely when queue size changes to avoid out-of-range behavior
- tests added for these recovery/interaction behaviors

## Local verification

```bash
go test -tags=testmpv ./internal/tui ./internal/tui/views
```
