# LLM Provider Review Checklist

## Boundary

- [ ] Provider adapter is behind an authorized backend use-case policy.
- [ ] Gateway/browser does not call Provider or receive Provider credential.
- [ ] Provider SDK types do not leak into domain/API/Event/Artifact contracts.
- [ ] Provider state is not treated as Conversation truth.

## Registry and policy

- [ ] Provider and model are registered and enabled.
- [ ] use case is one of the stable AgentHub 2.0 values.
- [ ] required capabilities match the selected and fallback models.
- [ ] prompt template and policy versions are explicit.

## Structured output

- [ ] local JSON/domain validation is mandatory.
- [ ] Planner output cannot bypass Plan Validator.
- [ ] invalid output has bounded repair/failure behavior.

## Streaming and reliability

- [ ] timeout and cancellation propagate.
- [ ] retry is classified and bounded.
- [ ] fallback is capability/privacy compatible.
- [ ] visible streaming is not duplicated by a full retry.
- [ ] terminal stream event is unique.

## Security and observability

- [ ] configuration contains a secret reference only.
- [ ] credentials, prompts, content, and private reasoning are redacted.
- [ ] provider/model/use case/request ID/usage/latency are normalized.
- [ ] errors are safe and correlated.

## Testing

- [ ] deterministic Mock Provider.
- [ ] disabled/capability mismatch.
- [ ] structured output validation.
- [ ] retry/fallback.
- [ ] cancellation/stream interruption.
- [ ] provider state loss/switch.
- [ ] redaction.
