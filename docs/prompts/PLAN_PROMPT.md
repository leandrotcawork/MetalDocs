# Official Prompt - Planning

Use `AGENTS.md`, `docs/architecture/ARCHITECTURE_GUARDRAILS.md`, and `docs/standards/ENGINEERING_STANDARDS.md` as hard constraints.
Produce a decision-complete plan for the requested change.
Mandatory output:
1. Goal and acceptance criteria.
2. Contract impacts (OpenAPI, events, errors).
3. Implementation steps in strict order.
4. Test plan (unit/contract/integration/e2e).
5. Risks and rollback notes.
Do not propose IA/LLM features in v1.
