# ADR-0001: Modular Monolith

## Status
Accepted

## Context
O projeto nasce com equipe pequena e ambiente on-prem local. Precisamos de simplicidade operacional sem perder capacidade de escalar.

## Decision
Adotar modular monolith com boundaries explicitos por dominio em `internal/modules`.

## Consequences
- Positivas: deploy simples, menor overhead operacional, transacoes locais.
- Negativas: exige disciplina arquitetural para evitar acoplamento indevido.
