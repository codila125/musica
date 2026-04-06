# Developer Guide

This document is for contributors and maintainers.

## Project layout

```text
cmd/main.go
  -> internal/tui            (presentation + event loop)
  -> internal/app            (orchestration/controllers)
  -> internal/api/*          (Navidrome/Jellyfin adapters)
  -> internal/player         (mpv integration + test backend)
  -> internal/tui/views      (tab view models + renderers)
  -> internal/telemetry      (structured events/counters/timing)
```

Notable modules:

- `internal/app/coordinator.go`: server switch/connect orchestration
- `internal/app/playback_controller.go`: playback command orchestration
- `internal/tui/view_adapter.go`: centralized view lifecycle/update wiring
- `internal/tui/state.go`: app state machine + transition validation

## Build tags strategy

- Tests/CI: `-tags testmpv`
- Runtime/release: `-tags nocgo`

Why this matters:

- `nocgo` runtime path loads `libmpv` during startup.
- Loader env vars must be set before process start.
- Do not rely on in-app env changes for `libmpv` discovery.

## Local development

Run app directly:

```bash
go build -o musica ./cmd
./musica
```

Or use dev helper:

```bash
./run.sh
```

`run.sh` is for local development only and not part of the production/Homebrew user flow.

## Testing and quality

Run tests (no native mpv required):

```bash
go test -tags=testmpv ./...
```

Install analysis tools once:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Run full local CI gates:

```bash
make ci
```

Run release-oriented gates (includes nocgo smoke build):

```bash
make ci-release
```

## Release workflow (summary)

1. Ensure `make ci-release` passes.
2. Tag and publish GitHub release (`vX.Y.Z`).
3. Update Homebrew formula tarball URL + sha256.
4. Validate Homebrew install on macOS and Linux.

Reference docs:

- `docs/production-homebrew.md`
- `Formula/musica.rb`
