# Process Boundaries

**Status:** Active entry document; canonical rules are in the [PDR](../pdr/AgentHub_2.0_PDR.md) and [project architecture Contract](../contracts/project-architecture.md).

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
```

- Frontend does not call Orchestrator or Agents directly.
- Gateway does not dispatch Agents or call a Provider directly.
- Orchestrator is the only dispatch entry and owns the internal Planner, Validator, Confirmation Gate, Context Manager, Registry and Synthesizer.
- These internal modules are not Agents and must not be registered as such.
- CodeAgent and WebAgent are built-in stable reference Agents; additional conforming Agents are a dynamic registration Target/In Progress capability.
- Direct has no multi-Agent Plan. Manual Multi-Agent and Auto require an LLM-generated, validated and user-confirmed exact PlanVersion.
