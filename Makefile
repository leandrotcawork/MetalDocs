SHELL := /bin/sh

.PHONY: up down logs

up:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d

down:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env down

logs:
	docker compose -f deploy/compose/docker-compose.yml --env-file .env logs -f
