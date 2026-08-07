GO ?= go

.PHONY: build format format-check integration-test test test-race vet verify

build:
	$(GO) build ./...

format:
	$(GO) fmt ./...

format-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l .; echo "Go sources required formatting." >&2; exit 1)

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

integration-test:
	./scripts/integration-test.sh

vet:
	$(GO) vet ./...

verify: format-check vet test test-race build
