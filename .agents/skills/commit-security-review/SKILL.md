---
name: commit-security-review
description: "AgentHub 2.0 pre-commit security and repository hygiene skill for staged-diff review, secrets, credentials, private data, Agent registration, Tool permissions, Artifact/Preview safety, dependencies, and generated files."
---

# commit-security-review

## Purpose

Use this Skill before committing, packaging, or handing off AgentHub changes.

## Read first

```text
/project-architecture
/security-boundary-contract
/testing-review-contract
/code-style-and-conventions
```

Primary sources:

```text
docs/contracts/commit-security-review.md
docs/contracts/commit-security-review.schema.json
docs/contracts/security-boundaries.md
```

## Required review flow

```text
working tree
-> staged file inventory
-> diff and whitespace
-> secret/private-data scan
-> trust-boundary review
-> dependency/generated-file review
-> tests and evidence
-> commit decision
```

## Blocking findings

- credential, token, private key, session cookie, or encrypted credential payload;
- `.env`, local database, uploaded private content, model/cache directory, or dependency tree;
- browser-visible provider or Agent credential;
- unvalidated remote Agent URL, redirect, DNS target, or AgentCard;
- Tool permission implicitly granted by Agent registration or Plan confirmation;
- unconfirmed Plan execution;
- cross-Conversation context access;
- Artifact overwrite without version check;
- generated code executing in the main application origin;
- secret/private prompt/private reasoning in log, event, fixture, or debug bundle;
- unexplained dependency or lockfile change;
- binary/large generated output unrelated to the requested artifact.

## Non-negotiable rules

- Review the actual staged diff, not only the working tree.
- Use `git diff --cached --check`.
- Do not stage or unstage files without explicit user intent.
- Do not rewrite history, force-push, or bypass hooks unless explicitly authorized.
- A security scan result is evidence, not a substitute for domain-boundary review.
- Redacted examples use unmistakably fake placeholders.
- A blocked commit report names the file and finding without repeating the secret value.

## Completion checklist

- [ ] staged inventory matches requested scope.
- [ ] diff/whitespace check passed.
- [ ] secret and private-data review completed.
- [ ] Agent/Registry/Provider/Tool/Artifact/Preview boundaries were checked when affected.
- [ ] dependency and generated files were reviewed.
- [ ] tests and validation evidence are recorded.
- [ ] commit is approved, conditionally approved, or blocked with an explicit reason.
