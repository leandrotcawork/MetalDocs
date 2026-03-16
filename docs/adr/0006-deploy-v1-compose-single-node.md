# ADR-0006: Deploy v1 with Docker Compose Single-node

## Status
Accepted

## Context
Ambiente inicial e servidor local `192.168.0.3`.

## Decision
Padronizar deploy v1 em Docker Compose single-node.
Criar runbooks de deploy, smoke e rollback antes de escala.

## Consequences
- Positivas: simplicidade operacional e entrega rapida.
- Negativas: limites de elasticidade horizontal no curto prazo.
