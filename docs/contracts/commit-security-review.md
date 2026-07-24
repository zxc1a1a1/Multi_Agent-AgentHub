# Commit Security Review Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

This Contract defines the pre-commit security and repository hygiene review for AgentHub changes.

The review examines the actual staged change and affected trust boundaries.

## 2. Inputs

```text
git status --short
git diff --cached --name-status
git diff --cached
git diff --cached --check
dependency/lockfile diff
test and scan evidence
```

Review does not stage, unstage, amend, rewrite history, or bypass hooks without explicit user authorization.

## 3. File inventory

Classify staged files:

```text
source
test
Contract/Schema/OpenAPI/ADR
migration/query
configuration/example
dependency/lockfile
generated artifact
binary/large file
secret/private-data risk
legacy migration
```

Unexpected scope is a blocker until resolved.

## 4. Prohibited sensitive content

Do not commit:

- API keys, access/refresh tokens, cookies, private keys, certificates with private material;
- AgentCredential encrypted payload or key material;
- `.env` with values;
- local SQLite database containing user data;
- provider raw request/response containing private content;
- uploaded Attachment/Artifact content unless it is an approved fixture;
- private prompt or private model reasoning;
- production URL containing credential/query secret;
- personal/private data not required by the repository.

Redacted examples use obvious placeholders such as:

```text
example-token
REPLACE_ME
${ENV_VAR}
```

## 5. Agent and network review

When Agent registration or A2A changes:

- URL, redirect, DNS, and private-network policy;
- AgentCard validation and size limits;
- credential storage and forwarding;
- AgentVersion and historical snapshot;
- enable/health/authorization eligibility;
- no implicit Tool permission;
- safe unknown output fallback.

## 6. Planning and context review

Block:

- AgentInvocation before exact PlanVersion confirmation;
- Manual Mode adding unselected Agent;
- stale Plan execution;
- silent Agent replacement;
- cross-Conversation context;
- full-history forwarding without policy;
- sensitive content included outside the approved projection.

## 7. Artifact and preview review

Block:

- ArtifactVersion overwrite;
- missing `baseVersion` conflict check;
- path traversal or unrestricted file write;
- unknown Artifact/project auto-execution;
- generated HTML/code in main application origin;
- browser access to app/provider/Agent credential;
- unrestricted preview network or parent-window access.

## 8. Provider and Tool review

- Provider secret remains backend-only;
- structured output is locally validated;
- visible stream cannot be duplicated by retry;
- Tool schema/permission/timeout/output limits exist;
- Plan confirmation is not treated as high-risk Tool approval;
- Tool result is untrusted.

## 9. Dependency and generated-file review

Every dependency or lockfile change has:

```text
reason
owner/import site
license/security consideration
version source
test evidence
rollback
```

Do not commit by default:

```text
node_modules
vendor unless repository policy requires it
build/dist binaries
coverage
temporary logs
model/cache files
IDE state
local databases
unrequested generated archives
```

## 10. Verdict

```text
APPROVED
APPROVED_WITH_FOLLOW_UP
BLOCKED
```

A blocked report includes:

```text
file
finding category
safe description
required action
```

Never repeat the secret value.

## 11. Required evidence

- staged inventory;
- whitespace result;
- secret/private-data review;
- affected domain security checks;
- dependency/generated-file review;
- tests/scans run;
- unrun checks and reason;
- verdict.
