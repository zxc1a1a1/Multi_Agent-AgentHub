# Provider Secret and Redaction Policy

Never persist or emit:

```text
API key
Authorization header
secret manager value
full private system prompt
private model reasoning
unredacted user/Attachment/Artifact content in ordinary logs
```

Allowed metadata includes bounded:

```text
provider
model
use case
request ID
token usage
latency
safe error code
```

Configuration stores secret references, not secret values.
