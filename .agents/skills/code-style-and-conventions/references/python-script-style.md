# Python Script Style

Python is used by default for:

```text
evaluation
dataset preparation
offline analysis
migration/support scripts
bounded developer automation
```

Rules:

- use type hints for public functions;
- use `pathlib`;
- validate CLI input;
- use deterministic seeds when relevant;
- do not embed secrets;
- do not write outside the requested directory without explicit flags;
- record dependency versions in `pyproject.toml` or approved requirements file;
- do not introduce a core online AgentHub service in Python without an approved ADR;
- tests use temporary directories and local fixtures.
