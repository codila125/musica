# Reliability & Fault Injection (Phase 12)

This project includes fault-injection style tests for high-risk interaction paths.

## Covered scenarios

- rapid switch key spam while switching is already in progress
- server switch auth failure transition + user-facing status mapping
- recovery flow after network failure (error state -> successful switch -> ready)
- coordinator propagation of typed connector failures

## Run locally

```bash
go test -tags=testmpv ./internal/tui ./internal/app
```

## Why this matters

These tests target non-happy-path behavior that often causes production regressions:

- race-prone repeated user actions
- transient backend/network errors
- state-machine recovery correctness
