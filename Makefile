GO ?= go

.PHONY: baseline-migration-check build cli-smoke contract-test dev-down dev-up docker-default-image-switch-check format format-check integration-test laravel-readonly-maintenance-check laravel-schema-freeze-check openapi-check operations-handover-check rollback-rehearsal-check smoke-targets-check staging-cutover-check test test-race vet verify

dev-up:
	@test -f ../idelium-api/ideliumapi/Dockerfile || (echo "Laravel source must be checked out beside this repository at ../idelium-api." >&2; exit 1)
	@umask 077; mkdir -p secrets; if [ ! -s secrets/dev_db_password ]; then od -An -N32 -tx1 /dev/urandom | tr -d ' \n' > secrets/dev_db_password; fi; chmod 0444 secrets/dev_db_password
	docker compose -f docker-compose.yml up --build -d

dev-down:
	docker compose -f docker-compose.yml down --remove-orphans

baseline-migration-check:
	python3 scripts/build_go_baseline_migration.py \
		--source-dir ../idelium-api/database/migrations \
		--output-json docs/contracts/go-baseline-migration.json \
		--output-markdown docs/contracts/go-baseline-migration.md \
		--output-embedded-json internal/migrations/baseline_manifest.json \
		--check

laravel-schema-freeze-check:
	python3 scripts/check_laravel_schema_freeze.py --check

laravel-readonly-maintenance-check:
	python3 scripts/build_laravel_readonly_maintenance.py --check

docker-default-image-switch-check:
	python3 scripts/build_docker_default_image_switch.py --check

rollback-rehearsal-check:
	python3 scripts/build_rollback_rehearsal.py --check

operations-handover-check:
	python3 scripts/build_operations_handover.py --check

build:
	$(GO) build ./...

cli-smoke:
	python3 scripts/run_cli_smoke.py --owner go --mode configuration-read

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

staging-cutover-check:
	python3 scripts/build_staging_route_cutover.py --check

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

verify: format-check vet test test-race contract-test baseline-migration-check laravel-schema-freeze-check openapi-check smoke-targets-check staging-cutover-check laravel-readonly-maintenance-check docker-default-image-switch-check rollback-rehearsal-check operations-handover-check build
