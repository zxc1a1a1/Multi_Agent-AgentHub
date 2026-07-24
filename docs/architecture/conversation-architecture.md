# Conversation Architecture

**Status:** Active entry document; see the [PDR](../pdr/AgentHub_2.0_PDR.md) and the [Active Contract index](../contracts/README.md) for authoritative detail.

AgentHub is a multi-conversation IM system. Complete conversation history is a platform persistence fact; model and Agent context is selected and bounded by the internal Context Manager.

Conversation participants are not a fixed Agent list. CodeAgent and WebAgent are built-in stable reference Agents; eligible A2A-conforming Agents may be registered dynamically as a 2.0 Target/In Progress capability. Planner, Context Manager and Synthesizer are not conversation Agents.

Direct Mode invokes one eligible Agent. Manual Multi-Agent Mode uses only the user-selected Agent set, and Auto Mode selects from the authorized Registry catalog. Both multi-Agent modes require an LLM-generated PlanVersion, local validation and explicit user confirmation before dispatch.
