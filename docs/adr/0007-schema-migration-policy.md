# ADR-0007: Schema Migration Policy

## Status
Accepted

## Context
Mudancas destrutivas sem governanca aumentam risco de indisponibilidade e perda de dados.

## Decision
Politica additive-first por padrao.
Mudanca destrutiva exige ADR, plano de rollback e janela de manutencao.

## Consequences
- Positivas: menor risco operacional e maior previsibilidade.
- Negativas: transicoes de schema podem ocorrer em mais de uma release.
