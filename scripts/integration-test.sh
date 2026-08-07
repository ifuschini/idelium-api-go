#!/bin/sh
set -eu

cleanup() {
    docker compose -f compose.test.yml down --volumes
}

trap cleanup EXIT INT TERM
docker compose -f compose.test.yml up --abort-on-container-exit --exit-code-from test
