GO ?= go

.PHONY: build contract-test format format-check integration-test openapi-check smoke-targets-check test test-race vet verify

build:
	$(GO) build ./...

contract-test:
	python3 -m unittest discover -s tests -p 'test_*.py'

openapi-check:
	python3 scripts/sync_openapi_legacy_contracts.py \
		--inventory docs/contracts/laravel-routes.json \
		--consumer-map docs/contracts/consumer-route-map.json \
		--openapi api/openapi.yaml \
		--check
	python3 scripts/generate_openapi_server_contracts.py \
		--openapi api/openapi.yaml \
		--output internal/openapicontract/generated_routes.go \
		--check

smoke-targets-check:
	python3 scripts/build_web_smoke_targets.py --check
	python3 scripts/build_cli_smoke_targets.py --check

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

verify: format-check vet test test-race contract-test openapi-check smoke-targets-check build
