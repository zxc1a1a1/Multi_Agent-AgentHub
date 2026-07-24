# LLM Provider Review Checklist

- [ ] use-case policy is explicit.
- [ ] provider/model is registered and enabled.
- [ ] required capabilities match.
- [ ] structured output is locally validated.
- [ ] provider SDK types do not cross adapter boundary.
- [ ] timeout, cancellation, retry, and fallback are bounded.
- [ ] no full retry duplicates visible stream content.
- [ ] provider state loss can be rebuilt from local context.
- [ ] usage/cost metadata is normalized.
- [ ] secrets, prompts, content, and private reasoning are redacted.
- [ ] deterministic Mock tests exist.
