# ADR-0005: Outbox and Idempotency for Domain Events

## Status
Accepted

## Context
Eventos podem divergir da transacao de negocio sem padrao de consistencia.

## Decision
Toda mutacao que publica evento deve usar outbox transacional + idempotency key.
Consumidores devem ser idempotentes.

## Consequences
- Positivas: consistencia entre estado persistido e eventos publicados.
- Negativas: maior complexidade operacional em publisher/consumer.
