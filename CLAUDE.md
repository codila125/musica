# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

MUSICA: keyboard-first terminal (Bubble Tea TUI) music player for Navidrome and Jellyfin servers. Go module `github.com/codila125/musica`, distributed via Homebrew (`Formula/musica.rb`).

## Build tags — critical

Two mutually exclusive build tags gate the `player` package's mpv backend:

- `testmpv`: fake/test player backend, no native libmpv needed. Use for all dev/test/lint work.
- `nocgo`: real backend, loads native `libmpv` at startup via cgo. Use only for release builds.

```bash
go build -o musica ./cmd              # local dev binary (default backend)
go test -tags=testmpv ./...           # tests — always use this tag
go vet -tags testmpv ./...
staticcheck -tags testmpv ./...
go build -trimpath -tags nocgo -o musica ./cmd   # release binary
```

Do not rely on in-app env changes for libmpv discovery — loader env vars (`DYLD_FALLBACK_LIBRARY_PATH` / `LD_LIBRARY_PATH`) must be set before process start.

## Common commands

```bash
make fmt            # gofmt -w .
make fmt-check      # verify formatting (CI gate)
make vet            # go vet -tags testmpv ./...
make staticcheck    # requires: go install honnef.co/go/tools/cmd/staticcheck@latest
make govulncheck    # requires: go install golang.org/x/vuln/cmd/govulncheck@latest
make test           # go test -tags=testmpv ./...
make test-race      # race tests, scoped to internal/app internal/api internal/tui internal/tui/views
make ci             # fmt-check vet staticcheck govulncheck test test-race
make ci-release     # ci + nocgo smoke build
./run.sh            # local dev helper (not part of production/Homebrew flow)
```

Run a single test: `go test -tags=testmpv ./internal/app -run TestName`

golangci-lint config (`.golangci.yml`) enables only: govet, staticcheck, ineffassign, typecheck, gosimple, unused.

## Architecture

```
cmd/main.go
  -> internal/tui            presentation + Bubble Tea event loop
  -> internal/app            orchestration/controllers
  -> internal/api/*          Navidrome/Jellyfin adapters behind api.Client
  -> internal/player         mpv integration + test backend
  -> internal/tui/views      tab view models + renderers (BROWSE/SEARCH/QUEUE)
  -> internal/telemetry      structured events/counters/timing
```

Key files:
- `internal/api/client.go`: the `api.Client` interface both `navidrome` and `jellyfin` adapters implement — this is the seam between server backends and the rest of the app.
- `internal/api/errors.go`: typed errors (`api.Error` with `Kind`: auth/network/config/unknown) via `api.Wrap` — callers use `api.KindOf(err)` to branch, not string matching.
- `internal/app/coordinator.go`: server switch/connect orchestration; `Connector` interface is the injection point for testing without real network calls (`defaultConnect` is the real implementation).
- `internal/app/playback_controller.go`: playback command orchestration.
- `internal/tui/state.go`: app state machine (`booting/loading/ready/switching_server/error`) with explicit `canTransition` validation — don't bypass it when adding new states/transitions.
- `internal/tui/view_adapter.go`: centralized view lifecycle/update wiring.
- `internal/models/models.go`: shared domain types (Track/Album/Artist/Playlist/SearchResult) used across api, app, and tui/views.
- `internal/config/config.go` + `loader.go` + `validate.go`: YAML config (`~/.config/musica` style), multi-server support, `ServerConfig.Redacted()` for safe logging of credentials.

Config CLI subcommands (`cmd/main.go`, `cmd/setup.go`): `setup`, `list`, `remove <name>`, `--server <name>`, default help text is in `printUsage()`.

## Code conventions (Go)

Style
- `gofmt` is non-negotiable; CI fails on unformatted code (`make fmt-check`). Run `make fmt` before committing.
- Package = one clear responsibility, short lowercase name, no underscores (`api`, `player`, `config`). Never add a `utils`/`helpers`/`common` grab-bag package.
- DRY: don't copy-paste the same logic across functions or files. If a block appears twice with the same meaning (not just coincidentally similar text), extract it into a shared helper the first time you touch either copy. Don't extract mid-refactor abstractions for code that merely looks similar but represents different domain concepts — that's coincidental duplication, not a DRY violation.
- Default to unexported; export only what another package genuinely needs to call.
- A file growing past ~500 lines (`tui/tui.go`, `player/player.go`) is a signal to split by responsibility next time you're in there — not a size to imitate in new files.
- Errors: wrap with `%w`, never silently discard (`_ = err` only when truly inert, and say why in a comment). Use the existing `api.Error` / `api.Wrap` / `api.KindOf` pattern for anything a caller must branch on by category — don't invent a second error scheme alongside it.
- Concurrency: guard shared state with a `sync.Mutex` declared next to the fields it protects (see `player.Player`); prefer channels/`context` cancellation over ad-hoc goroutine signaling flags.
- `context.Context` is always the first parameter and always propagated through I/O (network, mpv calls); never stored on a struct.
- No package-level mutable state, no `init()` side effects beyond constants.
- Comments explain non-obvious "why" only (a workaround, an invariant, a constraint) — never restate what the code already says.

Modularity / dependency injection
- Depend on interfaces owned by the consumer, not the implementer: `api.Client` is declared in `internal/api`, consumed by `internal/app`/`internal/tui`, and implemented by `navidrome`/`jellyfin`. New server backends implement the interface; callers never import a concrete adapter package directly.
- Inject collaborators through constructors (`NewCoordinator(servers, connector)`); never reach for a package-level singleton. `Connector` / `ConnectorFunc` in `coordinator.go` is the template for making any external call swappable in tests.
- One interface, one real implementation, one test fake. Don't add a factory or registry layer unless a task actually introduces a second real implementation.

## Testing conventions (TDD)

This project is developed test-first. For any new behavior or bug fix:
1. Write a failing test in the same package, exercising the behavior you're about to add/fix.
2. Run it — confirm it fails for the expected reason, not a compile error.
3. Write the minimum code to make it pass.
4. Refactor with the test green. Don't add new behavior and refactor in the same step.
5. No behavior change ships without a test that would have failed on the old code.

Conventions
- Test files are `_test.go`, colocated with the code under test, same package (white-box) unless the intent is specifically to test only the public API from outside.
- Table-driven tests are the default shape for anything with more than one case: a `[]struct{ name string; ... }` slice plus `for _, tt := range cases { t.Run(tt.name, func(t *testing.T) { ... }) }`.
- No mocking framework, hand-written fakes/stubs only — the function-adapter pattern (`ConnectorFunc`) or a fake `api.Client`. This matches the existing codebase and keeps tests readable without indirection.
- Every test always builds under `-tags=testmpv`; there is no CGO-free way to exercise the real mpv path, so never gate a test behind `nocgo` or real network/hardware.
- Cover the failure path, not just the happy path — see `coordinator_fault_test.go` / `fault_injection_test.go` as the template: inject an error at a boundary (connector, client) and assert the state machine/UI reacts correctly.
- One assertion concept per test; prefer several small tests over one test asserting many unrelated things.
- Race-sensitive packages (`player`, `app`, `tui`) must stay clean under `-race` (`make test-race`) — run it before calling a change in these packages done.
