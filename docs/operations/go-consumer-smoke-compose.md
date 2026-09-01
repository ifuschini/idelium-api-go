# Isolated Go consumer-smoke Compose profile

`compose.smoke.yml` starts a pinned MariaDB instance and the Go API on a
dedicated network. It does not share volumes, networks, or credentials with
the Laravel Compose projects.

## Start

Provide disposable values in the shell (or an ignored local env file):

```sh
export SMOKE_DB_PASSWORD='disposable-db-password'
export SMOKE_DB_ROOT_PASSWORD='disposable-root-password'
docker compose -f compose.smoke.yml up -d --build
```

The smoke gateway is exposed on `http://127.0.0.1:18080` by default. It removes
the public `/api` prefix before forwarding to Go. Readiness can be checked with:

```sh
curl --fail http://127.0.0.1:18080/health/live
curl --fail http://127.0.0.1:18080/health/ready
```

The image starts against an existing Idelium-compatible schema. The profile
intentionally does not invent migrations or credentials: load a disposable
schema and synthetic tenant fixtures before running consumer smoke checks.

## Stop and clean up

```sh
docker compose -f compose.smoke.yml down --volumes --remove-orphans
```

Never point this profile at a production database or real customer secrets.
