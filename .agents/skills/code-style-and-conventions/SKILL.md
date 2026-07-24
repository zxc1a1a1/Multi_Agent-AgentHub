---
name: code-style-and-conventions
description: "AgentHub 2.0 engineering conventions for Go platform services, React/TypeScript frontend, Python evaluation scripts, contracts, errors, tests, naming, and generated changes."
---

# code-style-and-conventions

## Purpose

Use this Skill when writing or reviewing Go, TypeScript/React, Python helper scripts, SQL, JSON/YAML, Markdown Contracts, naming, errors, tests, comments, or generated code.

## Read first

```text
/project-architecture
/ai-collaboration-workflow
/testing-review-contract
/commit-security-review
```

Primary source:

```text
docs/contracts/code-style.md
```

References:

```text
references/go-style.md
references/typescript-react-style.md
references/python-script-style.md
references/testing-and-errors.md
references/review-checklist.md
```

## Language boundary

```text
Go          core online services, Agent Runtime, protocol/storage adapters
TypeScript  React frontend, AG-UI reducer, Artifact/Sandpack workspace
Python      evaluation, datasets, analysis, bounded development scripts
SQL         migrations and typed queries through approved tooling
```

A new core online service in Python or another language requires an approved ADR.

## Non-negotiable rules

- Optimize for correctness and readability before abstraction.
- Keep changes small and domain-bounded.
- External input is `unknown`/untrusted until validated.
- Context, cancellation, timeout, and errors propagate across I/O boundaries.
- Do not introduce parallel domain/protocol types when official or active Contract types exist.
- Do not add new work to legacy `server/**` or root `agents/**` unless the task is an explicit migration.
- Do not add an abstraction used by one call site unless it enforces a real boundary or test seam.
- Logs contain stable IDs and safe errors, not credentials, private prompts, content dumps, or private reasoning.
- Comments explain invariants and tradeoffs, not obvious syntax.
- Generated code is formatted, tested, and free of placeholders.
- Never claim an unexecuted command or test passed.

## Completion checklist

- [ ] Language and module boundary is correct.
- [ ] Formatting/typecheck/lint appropriate to the language was run.
- [ ] I/O has cancellation/timeout and safe error handling.
- [ ] validation occurs at the trust boundary.
- [ ] concurrency has ownership and race coverage.
- [ ] tests cover negative behavior.
- [ ] no stale AgentHub terminology or retired Skill path was introduced.
- [ ] no secret or generated cache/binary was added.
