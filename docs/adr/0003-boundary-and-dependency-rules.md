# ADR-0003: Boundary and Dependency Rules

## Status
Accepted

## Context
Sem regras explicitas de dependencia, o monolito modular tende a acoplamento e regressao arquitetural.

## Decision
Definir e impor boundaries por modulo:
- Cross-module apenas via interface publica ou evento.
- Proibido importar internals de outro modulo.
- Fluxo obrigatorio: `delivery -> application -> domain -> infrastructure`.

## Consequences
- Positivas: escalabilidade organizacional e menor risco de "big ball of mud".
- Negativas: exige mais disciplina e revisao arquitetural.
