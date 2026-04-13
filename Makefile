SHELL := /bin/sh

.PHONY: up down logs test test-watch

up:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d

down:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env down

logs:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env logs -f

# Frontend tests — must run from the app directory so vitest picks up the
# correct vite.config / vitest.config and its React plugin transform pipeline.
# Running `vitest run` from the repo root uses the global npx vitest which
# cannot resolve the Vite plugin chain and causes STACK_TRACE_ERROR on some
# tests. Always use `make test` or `cd frontend/apps/web && npx vitest run`.
test:
	cd frontend/apps/web && npx vitest run

test-watch:
	cd frontend/apps/web && npx vitest
