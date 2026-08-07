GO ?= go

.PHONY: build contract-test format format-check integration-test test test-race vet verify

build:
	$(GO) build ./...

contract-test:
	python3 -m unittest discover -s tests -p 'test_*.py'

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

verify: format-check vet test test-race contract-test build
