# ADR-0004: Auth v1 Local RBAC with SSO-ready Interface

## Status
Accepted

## Context
Precisamos entregar rapido com seguranca, mantendo caminho para SSO corporativo futuro.

## Decision
Adotar autenticacao local na v1 com RBAC backend e interface de provider de identidade desacoplada para integrar SSO/LDAP depois.

## Consequences
- Positivas: menor complexidade inicial e rollout mais rapido.
- Negativas: migracao futura para SSO requer etapa planejada de integracao.
