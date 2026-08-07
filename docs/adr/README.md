# Architecture Decision Records

Architecture Decision Records (ADRs) preserve consequential technical choices
for the Laravel-to-Go migration. They complement the migration plan and
contracts; they do not replace executable tests or the route ownership matrix.

## Lifecycle

1. Copy [`template.md`](template.md) to `NNNN-short-title.md` using the next
   four-digit sequence.
2. Open it as `Proposed` and link the decision to its migration ticket.
3. Record considered alternatives, compatibility constraints, tenant and
   credential risks, deployment gates, and a bounded rollback.
4. Change the status to `Accepted` only after review. Use `Superseded by
   ADR-NNNN` rather than rewriting an accepted decision.
5. Commit the ADR with the implementation or governance slice that requires it.

## Index

| ADR | Status | Decision |
| --- | --- | --- |
| [ADR-0001](0001-strangler-migration-model.md) | Accepted | Migrate route groups through a reversible strangler model. |
