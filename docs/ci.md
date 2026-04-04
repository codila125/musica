# CI Quality Gates

This project uses CI gates to enforce baseline code quality and reliability.

## Gates

- `gofmt` formatting check
- `go vet`
- `staticcheck`
- `govulncheck`
- unit tests using `testmpv` tag
- race tests on core packages using `testmpv` tag

## Local run

Install tools once:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Run full local CI pipeline:

```bash
make ci
```

## Why `testmpv`?

CI does not require native libmpv. The `testmpv` build tag uses the in-repo test backend so tests can run deterministically without system mpv dependencies.
