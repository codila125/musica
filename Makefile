.PHONY: fmt fmt-check vet staticcheck govulncheck test test-race build-nocgo ci ci-release

STATICCHECK ?= $(shell go env GOPATH)/bin/staticcheck
GOVULNCHECK ?= $(shell go env GOPATH)/bin/govulncheck

fmt:
	gofmt -w .

fmt-check:
	@test -z "$(shell gofmt -l .)" || (echo "Run 'make fmt' to format code" && gofmt -l . && exit 1)

vet:
	go vet -tags testmpv ./...

staticcheck:
	$(STATICCHECK) -tags testmpv ./...

govulncheck:
	$(GOVULNCHECK) -tags testmpv ./...

test:
	go test -tags=testmpv ./...

test-race:
	go test -race -tags=testmpv ./internal/app ./internal/api ./internal/tui ./internal/tui/views

build-nocgo:
	go build -trimpath -tags nocgo -o /tmp/musica-nocgo ./cmd

ci: fmt-check vet staticcheck govulncheck test test-race

ci-release: ci build-nocgo
