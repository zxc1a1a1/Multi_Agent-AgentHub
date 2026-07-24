# AgentHub 2.0 PR Review Checklist

## Scope and source

- [ ] Current user scope is respected.
- [ ] Active AgentHub 2.0 Contracts were identified.
- [ ] Legacy documents were not used as higher authority.
- [ ] Unrelated refactors and dependencies were excluded.

## Architecture

- [ ] Frontend calls Gateway only.
- [ ] Gateway does not call Agents directly.
- [ ] Orchestrator is the dispatch boundary.
- [ ] Planner, Context Manager, Synthesizer, and Preview are not Agents.
- [ ] Dynamic registered Agents remain supported.

## Planning and context

- [ ] Direct Mode remains plan-free.
- [ ] multi-Agent execution requires exact PlanVersion confirmation.
- [ ] Manual Mode cannot add an unselected Agent.
- [ ] Replan requires a new version and confirmation.
- [ ] Conversation context is isolated and bounded.

## Agent, Artifact, and preview

- [ ] Registry eligibility and Agent snapshots are enforced.
- [ ] Agent credentials and Tool permissions remain separate.
- [ ] ArtifactVersion is immutable.
- [ ] `baseVersion` conflicts are handled.
- [ ] unknown Artifact/project content cannot execute.
- [ ] Preview is sandboxed and bound to an exact version.

## Evidence

- [ ] Contract/schema/API changes agree.
- [ ] deterministic tests cover positive and negative paths.
- [ ] concurrency/race tests were used when required.
- [ ] logs/errors are redacted.
- [ ] migration, compatibility, and rollback are documented.
