.PHONY: fmt fmt-check vet staticcheck govulncheck test test-race ci

fmt:
	gofmt -w .

fmt-check:
	@test -z "$(shell gofmt -l .)" || (echo "Run 'make fmt' to format code" && gofmt -l . && exit 1)

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

govulncheck:
	govulncheck ./...

test:
	go test -tags=testmpv ./...

test-race:
	go test -race -tags=testmpv ./internal/app ./internal/api ./internal/tui ./internal/tui/views

ci: fmt-check vet staticcheck govulncheck test test-race
