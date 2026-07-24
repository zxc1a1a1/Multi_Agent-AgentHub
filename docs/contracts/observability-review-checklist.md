# Observability Review Checklist

- [ ] service and Agent boundaries create coherent traces.
- [ ] request/Run/Plan/Invocation correlation exists.
- [ ] ContextSnapshot and ArtifactVersion are diagnosable.
- [ ] exact IDs are not uncontrolled metric labels.
- [ ] log schema is stable and machine-readable.
- [ ] credentials, prompts, content, and chain-of-thought are absent/redacted.
- [ ] model token/cost fields identify use case/provider/model safely.
- [ ] event replay and partial failure are diagnosable.
- [ ] Registry/A2A/Tool/Preview failure classes are separate.
- [ ] debug bundle is explicit, bounded, authorized, and audited.
- [ ] retention and deletion implications are addressed.
- [ ] tests prove redaction and trace propagation.
