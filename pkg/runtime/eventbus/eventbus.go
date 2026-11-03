// Package eventbus implements the EventBus contract frozen in
// docs/contracts/event-bus.md (schemaVersion 1) and docs/contracts/event-bus.schema.json.
//
// Two transports exist:
//
//   - memory.go: an in-process implementation used by contract tests and by
//     higher layers that need a deterministic bus without middleware.
//   - redis.go: the Redis Streams adapter (XADD / XREADGROUP / XACK / XAUTOCLAIM /
//     DLQ), which is only reachable when the operator configures an address.
//
// MySQL `execution_events` plus the outbox stay the only source of truth
// (EB-INV-01). Nothing in this package is authoritative: deleting the queue must
// never lose an event (EB-INV-03). Delivery is at-least-once (EB-INV-04); the
// "exactly one business effect" property comes from idempotency, not from the
// transport.
//
// Plan tasks: R2-02 (interface + in-memory), R2-03 (Redis Streams adapter).
// Production wiring (outbox relay v2) is R2-04 and is deliberately absent here.
package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SchemaVersion is the frozen envelope major version. A consumer that sees any
// other value must stop parsing and route the message to the DLQ with
// errorClass=unknown_schema_version; "best effort" parsing is forbidden.
const SchemaVersion = 1

const (
	// MaxEnvelopeBytes is EB-SEC-01: the serialized envelope, in UTF-8 bytes,
	// must not exceed 64KB. Oversized envelopes are rejected, never truncated.
	MaxEnvelopeBytes = 65536

	// DefaultMaxAttempts is EB-DEL-04's default retry budget.
	DefaultMaxAttempts = 5

	// MaxAttemptsCap is the hard ceiling for the retry budget (attempt is
	// bounded 1..32 by the schema, so an unbounded retry is unrepresentable).
	MaxAttemptsCap = 32

	// DefaultStreamMaxLen and DefaultDLQMaxLen are the recommended XADD MAXLEN ~	// bounds from event-bus.md section 4.1. A write without any bound is
	// forbidden (EB-STR), so an unset value falls back to these, never to 0.
	DefaultStreamMaxLen = 100_000
	DefaultDLQMaxLen    = 10_000
	// DefaultStreamTTL matches the relay's existing one-hour event-stream
	// lifetime; DefaultDLQStreamTTL keeps failures inspectable for a month.
	DefaultStreamTTL    = time.Hour
	DefaultDLQStreamTTL = 30 * 24 * time.Hour
)

// Frozen stream keys (event-bus.md section 4.1). The shared run stream carries
// every workspace of one environment, which is why there is no workspaceId in
// the key and why visibility must be resolved from SQL (EB-ISO-01).
const (
	RunStreamFormat = "agenthub:%s:run:events"
	DLQStreamFormat = "agenthub:%s:dlq"
	envPattern      = `^[a-z0-9_-]{1,32}$`
)

// ErrorClass is the closed DLQ/metric error taxonomy of EB-DEL-09. Raw error
// text, stack traces, DSNs and provider responses must never be recorded, so
// nothing but these ten values is representable.
type ErrorClass string

const (
	ErrorClassDecode                    ErrorClass = "decode_error"
	ErrorClassUnknownSchemaVersion      ErrorClass = "unknown_schema_version"
	ErrorClassTypeNotAllowed            ErrorClass = "type_not_allowed"
	ErrorClassSizeExceeded              ErrorClass = "size_exceeded"
	ErrorClassSecurityRedactionRequired ErrorClass = "security_redaction_required"
	ErrorClassHandlerError              ErrorClass = "handler_error"
	ErrorClassStoreError                ErrorClass = "store_error"
	ErrorClassIdempotencyConflict       ErrorClass = "idempotency_conflict"
	ErrorClassTimeout                   ErrorClass = "timeout"
	ErrorClassUnknown                   ErrorClass = "unknown"
)

// errorClasses is the EB-DEL-09 enum, kept in one place so tests can compare it
// against the JSON Schema.
var errorClasses = []ErrorClass{
	ErrorClassDecode,
	ErrorClassUnknownSchemaVersion,
	ErrorClassTypeNotAllowed,
	ErrorClassSizeExceeded,
	ErrorClassSecurityRedactionRequired,
	ErrorClassHandlerError,
	ErrorClassStoreError,
	ErrorClassIdempotencyConflict,
	ErrorClassTimeout,
	ErrorClassUnknown,
}

// IsErrorClass reports whether class belongs to the closed enum.
func IsErrorClass(class ErrorClass) bool {
	for _, known := range errorClasses {
		if known == class {
			return true
		}
	}
	return false
}

// ErrorClasses returns a copy of the closed errorClass enum.
func ErrorClasses() []ErrorClass {
	out := make([]ErrorClass, len(errorClasses))
	copy(out, errorClasses)
	return out
}

// allowedTypes is the EB-TYPE-01 allowlist: the internal (lower_snake) event
// names of pkg/runtime/agui/event_types.go. Public UPPER_SNAKE names (EB-TYPE-02)
// and translator dot names (EB-TYPE-03) are not queue protocol and are rejected.
// TestEnvelopeAllowlistCoversInternalTypes pins this set against both the agui
// package and the contract schema, so the list cannot drift silently.
var allowedTypes = []string{
	"run_started",
	"run_finished",
	"run_error",
	"message_start",
	"message_delta",
	"message_end",
	"state_update",
	"activity_snapshot",
	"tool_call_start",
	"tool_call_args",
	"tool_call_end",
	"agent_turn_started",
	"agent_turn_content",
	"agent_turn_finished",
}

// AllowedTypes returns a copy of the frozen event type allowlist.
func AllowedTypes() []string {
	out := make([]string, len(allowedTypes))
	copy(out, allowedTypes)
	return out
}

// IsAllowedType reports whether typ may appear as an envelope type.
func IsAllowedType(typ string) bool {
	for _, known := range allowedTypes {
		if known == typ {
			return true
		}
	}
	return false
}

// forbiddenPayloadKeys is EB-SEC-02: prompts, tokens, secrets, DSNs, cookies
// and inline blobs may never enter the queue. The schema enforces the top-level
// keys; deeper nesting is covered by publisher redaction plus the secret scan
// (event-bus.md section 12).
var forbiddenPayloadKeys = []string{
	"prompt", "prompts", "systemPrompt",
	"token", "tokens", "apiKey", "accessToken", "refreshToken",
	"clientSecret", "secret", "secrets", "password",
	"credential", "credentials", "privateKey",
	"dsn", "databaseUrl", "cookie", "authorization", "headers", "blob",
}

// ForbiddenPayloadKeys returns a copy of the EB-SEC-02 top-level key blocklist.
func ForbiddenPayloadKeys() []string {
	out := make([]string, len(forbiddenPayloadKeys))
	copy(out, forbiddenPayloadKeys)
	return out
}

var (
	eventIDPattern      = regexp.MustCompile(`^evt_[A-Za-z0-9_-]{1,80}_[0-9]{1,19}_[0-9]{1,10}$`)
	runIDPattern        = regexp.MustCompile(`^run_[A-Za-z0-9_-]{1,64}$`)
	conversationPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	scopedIDPattern     = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	traceIDPattern      = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)
	idempotencyPattern  = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,80}:[0-9]{1,19}:[0-9]{1,10}$`)
	utcTimestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`)
	consumerKeyPattern  = regexp.MustCompile(`^[a-z0-9_.:-]{1,128}$`)
	streamKeyPattern    = regexp.MustCompile(`^agenthub:[a-z0-9_-]{1,32}:run:events$`)
	streamEntryPattern  = regexp.MustCompile(`^[0-9]{1,19}-[0-9]{1,10}$`)
	envRegexp           = regexp.MustCompile(envPattern)

	// envelopeFields is the closed field set of EB-ENV-05
	// (schema additionalProperties:false). An envelope carrying any other key,
	// or missing any of these 13, is malformed.
	envelopeFields = []string{
		"eventId", "schemaVersion", "type", "runId", "conversationId", "taskId",
		"seq", "subIndex", "occurredAt", "traceId", "idempotencyKey", "attempt",
		"payload",
	}

	// ErrClosedBus is returned by Publish once Close has been called.
	ErrClosedBus = errors.New("eventbus: bus is closed")
)

// EnvelopeFields returns a copy of the 13 required envelope field names.
func EnvelopeFields() []string {
	out := make([]string, len(envelopeFields))
	copy(out, envelopeFields)
	return out
}

// Envelope is the frozen queue envelope of event-bus.md section 2.1. All 13
// fields are required; fields without a value carry the empty string rather
// than being omitted, so consumers have exactly one parse path.
type Envelope struct {
	// EventID is derived: evt_<runId>_<seq>_<subIndex>. Random UUIDs are
	// forbidden (EB-ENV-01) so a re-publish after a relay crash keeps the id.
	EventID string `json:"eventId"`
	// SchemaVersion is always SchemaVersion in this build.
	SchemaVersion int `json:"schemaVersion"`
	// Type must come from the allowlist (EB-TYPE-01).
	Type string `json:"type"`
	// RunID is the ordering key (EB-ORD-01) and partition key.
	RunID string `json:"runId"`
	// ConversationID scopes the message; authorization checks still go to SQL.
	ConversationID string `json:"conversationId"`
	// TaskID is the A2A task, or "" when the event is unrelated to a task.
	TaskID string `json:"taskId"`
	// Seq is the monotonic per-run sequence starting at 1.
	Seq int64 `json:"seq"`
	// SubIndex is the shard index inside one seq, starting at 0.
	SubIndex int `json:"subIndex"`
	// OccurredAt is when the event happened inside the Orchestrator
	// transaction (UTC RFC3339), not when it was published.
	OccurredAt string `json:"occurredAt"`
	// TraceID is the cross-layer trace, or "" (relay has no trace source yet;
	// inventing one is forbidden by section 2.1).
	TraceID string `json:"traceId"`
	// IdempotencyKey is derived: <runId>:<seq>:<subIndex> (EB-ENV-02).
	IdempotencyKey string `json:"idempotencyKey"`
	// Attempt is written as 1 by the publisher; only the consumer/adapter
	// counts re-deliveries, so the counter has a single writer.
	Attempt int `json:"attempt"`
	// Payload is the agui.InternalStreamEvent JSON, unmodified and unwrapped.
	Payload json.RawMessage `json:"payload"`
}

// EnvelopeParams describes the inputs needed to build a conforming envelope.
// eventId, idempotencyKey, schemaVersion and attempt are derived, never
// supplied, so a caller cannot publish an inconsistent identity.
type EnvelopeParams struct {
	Type           string
	RunID          string
	ConversationID string
	TaskID         string
	Seq            int64
	SubIndex       int
	OccurredAt     time.Time
	TraceID        string
	// Payload accepts json.RawMessage, a string of JSON, or any value that
	// encoding/json can marshal into an object.
	Payload any
}

// DeriveEventID is EB-ENV-01: eventId is a deterministic function of
// (runId, seq, subIndex). Two publishers racing on the same logical event must
// produce the same id, otherwise idempotency cannot work.
func DeriveEventID(runID string, seq int64, subIndex int) string {
	return fmt.Sprintf("evt_%s_%d_%d", runID, seq, subIndex)
}

// DeriveIdempotencyKey is EB-ENV-02: <runId>:<seq>:<subIndex>, structurally
// identical to run_signals.idempotency_key and the projection cursor.
func DeriveIdempotencyKey(runID string, seq int64, subIndex int) string {
	return fmt.Sprintf("%s:%d:%d", runID, seq, subIndex)
}

// FormatOccurredAt renders a timestamp as the UTC RFC3339 form the schema
// pattern requires (EB-ENV-04). Local offsets are never emitted.
func FormatOccurredAt(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.999999999Z")
}

// NewEnvelope builds and validates an envelope, deriving eventId,
// idempotencyKey, schemaVersion and attempt=1.
func NewEnvelope(params EnvelopeParams) (Envelope, error) {
	payload, err := normalizePayload(params.Payload)
	if err != nil {
		return Envelope{}, err
	}
	env := Envelope{
		EventID:        DeriveEventID(params.RunID, params.Seq, params.SubIndex),
		SchemaVersion:  SchemaVersion,
		Type:           params.Type,
		RunID:          params.RunID,
		ConversationID: params.ConversationID,
		TaskID:         params.TaskID,
		Seq:            params.Seq,
		SubIndex:       params.SubIndex,
		OccurredAt:     FormatOccurredAt(params.OccurredAt),
		TraceID:        params.TraceID,
		IdempotencyKey: DeriveIdempotencyKey(params.RunID, params.Seq, params.SubIndex),
		Attempt:        1,
		Payload:        payload,
	}
	if err := env.Validate(); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func normalizePayload(raw any) (json.RawMessage, error) {
	switch v := raw.(type) {
	case nil:
		return nil, &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload is required"}
	case json.RawMessage:
		if len(v) == 0 {
			return nil, &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload is required"}
		}
		return v, nil
	case []byte:
		return normalizePayload(json.RawMessage(v))
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload is required"}
		}
		return json.RawMessage(v), nil
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload is not serializable"}
		}
		return encoded, nil
	}
}

// EnvelopeError carries a contract rule violation together with the closed
// ErrorClass it maps to, so a DLQ entry never has to embed raw error text.
type EnvelopeError struct {
	Class  ErrorClass
	Field  string
	Reason string
}

func (e *EnvelopeError) Error() string {
	if e == nil {
		return "eventbus: unknown envelope error"
	}
	if e.Field == "" {
		return "eventbus: envelope rejected (" + string(e.Class) + "): " + e.Reason
	}
	return "eventbus: envelope rejected (" + string(e.Class) + ") at " + e.Field + ": " + e.Reason
}

// ErrorClassOf maps any error to the closed EB-DEL-09 taxonomy. Anything that
// is not explicitly classified becomes "unknown"; the original text is dropped
// on purpose so it cannot leak into a DLQ entry or a metric label.
func ErrorClassOf(err error) ErrorClass {
	if err == nil {
		return ErrorClassUnknown
	}
	var envErr *EnvelopeError
	if errors.As(err, &envErr) {
		return envErr.Class
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorClassTimeout
	case errors.Is(err, ErrClosedBus):
		return ErrorClassStoreError
	default:
		return ErrorClassUnknown
	}
}

// Validate checks every rule the JSON Schema cannot express on its own plus the
// schema constraints that Go types blur (EB-ENV-01..06, EB-SEC-01/02/07,
// EB-TYPE-01). It is the publisher gate: Publish refuses an invalid envelope
// instead of repairing it.
func (e Envelope) Validate() error {
	if e.SchemaVersion != SchemaVersion {
		return &EnvelopeError{Class: ErrorClassUnknownSchemaVersion, Field: "schemaVersion",
			Reason: fmt.Sprintf("expected %d, got %d", SchemaVersion, e.SchemaVersion)}
	}
	if !IsAllowedType(e.Type) {
		return &EnvelopeError{Class: ErrorClassTypeNotAllowed, Field: "type",
			Reason: "type is outside the frozen allowlist"}
	}
	if !runIDPattern.MatchString(e.RunID) || len(e.RunID) > 64 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "runId", Reason: "must match ^run_[A-Za-z0-9_-]{1,64}$"}
	}
	if !conversationPattern.MatchString(e.ConversationID) {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "conversationId",
			Reason: "must match ^[A-Za-z0-9_.:-]{1,64}$ and be non-empty"}
	}
	if e.TaskID != "" && !scopedIDPattern.MatchString(e.TaskID) {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "taskId",
			Reason: "must be empty or match ^[A-Za-z0-9_.:-]{1,64}$"}
	}
	if e.Seq < 1 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "seq", Reason: "must be >= 1"}
	}
	if e.SubIndex < 0 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "subIndex", Reason: "must be >= 0"}
	}
	if !utcTimestampPattern.MatchString(e.OccurredAt) {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "occurredAt",
			Reason: "must be UTC RFC3339 with a Z suffix (EB-ENV-04)"}
	}
	if parsed, err := time.Parse(time.RFC3339, e.OccurredAt); err != nil || parsed.Location() != time.UTC {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "occurredAt", Reason: "not a valid UTC timestamp"}
	}
	if e.TraceID != "" && !traceIDPattern.MatchString(e.TraceID) {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "traceId",
			Reason: "must be empty or match ^[A-Za-z0-9_.:-]{1,128}$"}
	}
	if !eventIDPattern.MatchString(e.EventID) || len(e.EventID) > 160 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "eventId",
			Reason: "must match ^evt_[A-Za-z0-9_-]{1,80}_[0-9]{1,19}_[0-9]{1,10}$"}
	}
	if want := DeriveEventID(e.RunID, e.Seq, e.SubIndex); e.EventID != want {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "eventId",
			Reason: "eventId must equal evt_<runId>_<seq>_<subIndex> (EB-ENV-01)"}
	}
	if !idempotencyPattern.MatchString(e.IdempotencyKey) || len(e.IdempotencyKey) > 160 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "idempotencyKey",
			Reason: "must match ^[A-Za-z0-9_.:-]{1,80}:[0-9]{1,19}:[0-9]{1,10}$"}
	}
	if want := DeriveIdempotencyKey(e.RunID, e.Seq, e.SubIndex); e.IdempotencyKey != want {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "idempotencyKey",
			Reason: "idempotencyKey must equal <runId>:<seq>:<subIndex> (EB-ENV-02)"}
	}
	if e.Attempt < 1 || e.Attempt > MaxAttemptsCap {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "attempt",
			Reason: fmt.Sprintf("must stay within 1..%d", MaxAttemptsCap)}
	}
	if err := e.validatePayload(); err != nil {
		return err
	}
	// EB-SEC-01 is a byte budget the schema cannot express.
	if size := len(e.MustEncode()); size > MaxEnvelopeBytes {
		return &EnvelopeError{Class: ErrorClassSizeExceeded, Field: "payload",
			Reason: fmt.Sprintf("serialized envelope is %d bytes, limit is %d", size, MaxEnvelopeBytes)}
	}
	return nil
}

func (e Envelope) validatePayload() error {
	if len(e.Payload) == 0 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload is required"}
	}
	// The schema pins payload to type object. json.Unmarshal would accept "null"
	// into a nil map without complaint, so the opening byte is checked too.
	if trimmed := strings.TrimLeft(string(e.Payload), " \t\r\n"); !strings.HasPrefix(trimmed, "{") {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload must be a JSON object"}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &fields); err != nil || fields == nil {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "payload", Reason: "payload must be a JSON object"}
	}
	if inner, ok := fields["type"]; ok {
		var typ string
		if err := json.Unmarshal(inner, &typ); err != nil {
			return &EnvelopeError{Class: ErrorClassDecode, Field: "payload.type", Reason: "payload.type must be a string"}
		}
		// EB-ENV-06: a payload carrying its own type must agree with the envelope.
		if typ != e.Type {
			return &EnvelopeError{Class: ErrorClassDecode, Field: "payload.type",
				Reason: "payload.type differs from envelope type"}
		}
	}
	for key := range fields {
		for _, forbidden := range forbiddenPayloadKeys {
			if key == forbidden {
				return &EnvelopeError{Class: ErrorClassSecurityRedactionRequired, Field: "payload." + key,
					Reason: "payload key is forbidden by EB-SEC-02"}
			}
		}
	}
	switch e.Type {
	case "agent_turn_started", "agent_turn_content", "agent_turn_finished":
		if _, ok := fields["turnIndex"]; !ok {
			return &EnvelopeError{Class: ErrorClassDecode, Field: "payload.turnIndex",
				Reason: "AGENT_TURN events must carry turnIndex"}
		}
	case "message_start", "message_delta", "message_end":
		if _, ok := fields["messageId"]; !ok {
			return &EnvelopeError{Class: ErrorClassDecode, Field: "payload.messageId",
				Reason: "text stream events must carry messageId"}
		}
	case "run_error":
		// EB-SEC-06: only the redacted SafeError projection may be queued.
		for _, forbidden := range []string{"stackTrace", "rawError"} {
			if _, ok := fields[forbidden]; ok {
				return &EnvelopeError{Class: ErrorClassSecurityRedactionRequired, Field: "payload." + forbidden,
					Reason: "run_error must carry the redacted SafeError only"}
			}
		}
	}
	return nil
}

// ValidateForPublish is the publisher-side gate. Beyond Validate it insists that
// a fresh publication carries attempt=1: the retry budget belongs to the
// consumer/adapter, so allowing a publisher to claim "attempt 4" would create a
// second source of truth for the counter.
func (e Envelope) ValidateForPublish() error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.Attempt != 1 {
		return &EnvelopeError{Class: ErrorClassDecode, Field: "attempt",
			Reason: "a publisher always writes attempt=1; re-deliveries are counted by the consumer"}
	}
	return nil
}

// Encode renders the envelope as the single JSON value stored in the Redis
// `data` field (section 4.3). The output is a compact single-line UTF-8 document.
func (e Envelope) Encode() ([]byte, error) {
	encoded, err := json.Marshal(e)
	if err != nil {
		return nil, &EnvelopeError{Class: ErrorClassDecode, Reason: "envelope cannot be encoded: " + err.Error()}
	}
	return encoded, nil
}

// MustEncode is only used on validated envelopes in metrics paths, where a
// failure would signal a bug rather than malformed input.
func (e Envelope) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		return []byte(err.Error())
	}
	return encoded
}

// DecodeEnvelope parses queue bytes strictly: the field set must equal the 13
// frozen names, so a missing key (schema `required`) and an unexpected key
// (EB-ENV-05 / additionalProperties:false) are both refused before any semantic
// check runs. Consumers ignore unknown fields for forward compatibility but
// never decide on them.
func DecodeEnvelope(data []byte) (Envelope, error) {
	if len(data) == 0 {
		return Envelope{}, &EnvelopeError{Class: ErrorClassDecode, Reason: "empty frame"}
	}
	if len(data) > MaxEnvelopeBytes {
		return Envelope{}, &EnvelopeError{Class: ErrorClassSizeExceeded,
			Reason: fmt.Sprintf("frame is %d bytes, limit is %d", len(data), MaxEnvelopeBytes)}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Envelope{}, &EnvelopeError{Class: ErrorClassDecode, Reason: "frame is not a JSON object"}
	}
	for _, required := range envelopeFields {
		if _, ok := fields[required]; !ok {
			return Envelope{}, &EnvelopeError{Class: ErrorClassDecode, Field: required, Reason: "required envelope field is missing"}
		}
	}
	for key := range fields {
		found := false
		for _, known := range envelopeFields {
			if known == key {
				found = true
				break
			}
		}
		if !found {
			return Envelope{}, &EnvelopeError{Class: ErrorClassDecode, Field: key,
				Reason: "undefined envelope field (EB-ENV-05 forbids additional properties)"}
		}
	}
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, &EnvelopeError{Class: ErrorClassDecode, Reason: "envelope does not match the frozen types: " + err.Error()}
	}
	if err := env.Validate(); err != nil {
		return env, err
	}
	return env, nil
}

// Handler processes one delivered message. Returning nil means the business
// transaction has committed, which is the only moment an ack becomes legal
// (EB-DEL-02). Returning an error keeps the message unacked.
type Handler func(ctx context.Context, msg *Message) error

// Message is one delivery of an envelope. Attempt is the consumer-side delivery
// count: 1 on first delivery, incremented by the transport on re-delivery
// (XAUTOCLAIM for Redis). The publisher never touches it.
type Message struct {
	// Envelope is the decoded, validated payload carrier.
	Envelope Envelope
	// TransportID is the Redis stream entry id; empty for the in-memory bus.
	TransportID string
	// Attempt counts consumer-side deliveries, 1..MaxAttemptsCap.
	Attempt int
	// Group is the consumer group (projection|signal|audit) the delivery belongs to.
	Group string
	// ConsumerKey is the idempotency/cursor key of that group (EB-GRP table).
	ConsumerKey string
	// Consumer is the instance name inside the group (EB-GRP-02).
	Consumer string
	// Redelivered marks a message reclaimed from the pending list.
	Redelivered bool
}

// TraceID exposes the trace pass-through so a consumer can keep its own spans
// joined to the run without re-decoding the envelope.
func (m *Message) TraceID() string {
	if m == nil {
		return ""
	}
	return m.Envelope.TraceID
}

// Publisher hands envelopes to one transport. Implementations validate before
// writing and never repair an envelope silently.
type Publisher interface {
	// Publish writes one envelope. It returns an *EnvelopeError for a malformed
	// envelope and transport errors otherwise; a transport error means the SQL
	// outbox stays pending (EB-DEL-07).
	Publish(ctx context.Context, envelope Envelope) error
	// Close releases transport resources. Close is idempotent.
	Close(ctx context.Context) error
}

// Consumer reads one consumer group's messages and runs a Handler per message.
type Consumer interface {
	// Consume blocks until ctx is cancelled or Close is called, dispatching
	// every delivery to handler. Messages of the same runId are handled
	// serially (EB-ORD-01); different runIds may be handled concurrently
	// (EB-ORD-06).
	Consume(ctx context.Context, handler Handler) error
	// Ack confirms a delivery after its business transaction committed.
	Ack(ctx context.Context, msg *Message) error
	// Nack reports a failed delivery: the message stays available for retry,
	// and past maxAttempts it is routed to the DLQ. Nack never acks a handler
	// that has not exhausted its budget (EB-DEL-03).
	Nack(ctx context.Context, msg *Message, cause error) error
	// Close stops the read loops and waits for in-flight handlers.
	Close(ctx context.Context) error
}

// DLQEntry is the only structure allowed in the dead-letter stream
// (EB-DEL-08 / EB-SEC-04). There is deliberately no payload field: adding one
// would be a contract change.
type DLQEntry struct {
	EventID       string     `json:"eventId"`
	ConsumerKey   string     `json:"consumerKey"`
	RunID         string     `json:"runId"`
	Seq           int64      `json:"seq"`
	Type          string     `json:"type"`
	Attempts      int        `json:"attempts"`
	ErrorClass    ErrorClass `json:"errorClass"`
	FirstSeen     string     `json:"firstSeen"`
	LastSeen      string     `json:"lastSeen"`
	StreamKey     string     `json:"streamKey,omitempty"`
	StreamEntryID string     `json:"streamEntryId,omitempty"`
}

// NewDLQEntry derives a DLQ entry from an envelope without ever copying payload
// bytes; the original timestamp seeds firstSeen/lastSeen.
func NewDLQEntry(env Envelope, consumerKey string, attempts int, class ErrorClass, streamKey, transportID string, seenAt time.Time) DLQEntry {
	if !IsErrorClass(class) {
		class = ErrorClassUnknown
	}
	// The retry budget is bounded by the schema (attempt 1..32), so an attempt
	// count can never be recorded as unbounded.
	switch {
	case attempts < 1:
		attempts = 1
	case attempts > MaxAttemptsCap:
		attempts = MaxAttemptsCap
	}
	stamp := FormatOccurredAt(seenAt)
	first := stamp
	if env.OccurredAt != "" {
		// The envelope's own event time is the earliest known sighting.
		first = env.OccurredAt
	}
	entry := DLQEntry{
		EventID:     env.EventID,
		ConsumerKey: consumerKey,
		RunID:       env.RunID,
		Seq:         env.Seq,
		Type:        env.Type,
		Attempts:    attempts,
		ErrorClass:  class,
		FirstSeen:   first,
		LastSeen:    stamp,
	}
	if streamKeyPattern.MatchString(streamKey) {
		entry.StreamKey = streamKey
	}
	if streamEntryPattern.MatchString(transportID) {
		entry.StreamEntryID = transportID
	}
	return entry
}

// Validate checks a DLQ entry against $defs.dlqEntry. Entries describing a
// completely undecodable frame carry empty metadata (there is no eventId to
// report); see event-bus.md section 6 EB-DEL-06 for the rule that still requires
// them to be dead-lettered, and DLQMalformedSummary for that subset.
func (e DLQEntry) Validate() error {
	if e.EventID != "" && (!eventIDPattern.MatchString(e.EventID) || len(e.EventID) > 160) {
		return fmt.Errorf("eventbus: dlq eventId is malformed")
	}
	if !consumerKeyPattern.MatchString(e.ConsumerKey) {
		return fmt.Errorf("eventbus: dlq consumerKey must match ^[a-z0-9_.:-]{1,128}$")
	}
	if e.RunID != "" && !runIDPattern.MatchString(e.RunID) {
		return fmt.Errorf("eventbus: dlq runId is malformed")
	}
	if e.Seq < 0 {
		return fmt.Errorf("eventbus: dlq seq must not be negative")
	}
	if e.Type != "" && !IsAllowedType(e.Type) {
		return fmt.Errorf("eventbus: dlq type is outside the allowlist")
	}
	if e.Attempts < 1 || e.Attempts > MaxAttemptsCap {
		return fmt.Errorf("eventbus: dlq attempts must stay within 1..%d", MaxAttemptsCap)
	}
	if !IsErrorClass(e.ErrorClass) {
		return fmt.Errorf("eventbus: dlq errorClass %q is outside the closed enum", string(e.ErrorClass))
	}
	for name, value := range map[string]string{"firstSeen": e.FirstSeen, "lastSeen": e.LastSeen} {
		if !utcTimestampPattern.MatchString(value) {
			return fmt.Errorf("eventbus: dlq %s must be UTC RFC3339", name)
		}
	}
	if e.StreamKey != "" && !streamKeyPattern.MatchString(e.StreamKey) {
		return fmt.Errorf("eventbus: dlq streamKey must match the run stream key")
	}
	if e.StreamEntryID != "" && !streamEntryPattern.MatchString(e.StreamEntryID) {
		return fmt.Errorf("eventbus: dlq streamEntryId must look like <ms>-<seq>")
	}
	return nil
}

// DLQFields returns the JSON key set a DLQ entry may carry. It is the mirror of
// $defs.dlqEntry's closed object and is what the DLQ writer serializes, so a
// payload can never ride along by accident.
func DLQFields() []string {
	return []string{"eventId", "consumerKey", "runId", "seq", "type", "attempts", "errorClass", "firstSeen", "lastSeen", "streamKey", "streamEntryId"}
}

// Encode serializes a DLQ entry as the single `data` field of the dead-letter
// stream. It refuses an entry that fails Validate.
func (e DLQEntry) Encode() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

// StreamKeyFor builds the shared run stream key of one environment.
func StreamKeyFor(env string) string {
	return fmt.Sprintf(RunStreamFormat, env)
}

// DLQStreamKeyFor builds the dead-letter stream key of one environment.
func DLQStreamKeyFor(env string) string {
	return fmt.Sprintf(DLQStreamFormat, env)
}

// ErrorClassForDelivery maps an error raised while delivering to a consumer
// handler onto the closed EB-DEL-09 taxonomy. Unlike ErrorClassOf, the caller is
// already known to be the delivery path, so an unclassified business failure is
// handler_error rather than unknown: mislabelling it would hide the retry signal
// from R2-06 metrics. A handler panic stays "unknown" on purpose, because the
// panic value is deliberately dropped.
func ErrorClassForDelivery(err error) ErrorClass {
	if err == nil {
		return ErrorClassHandlerError
	}
	var envErr *EnvelopeError
	if errors.As(err, &envErr) {
		return envErr.Class
	}
	if errors.Is(err, ErrHandlerPanicked) {
		return ErrorClassUnknown
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassTimeout
	}
	if errors.Is(err, ErrClosedBus) {
		return ErrorClassStoreError
	}
	if errors.Is(err, context.Canceled) {
		// A cancelled consumer is not a handler failure; recording it as one
		// would burn the retry budget of an event that never really failed.
		return ErrorClassTimeout
	}
	return ErrorClassHandlerError
}

// ValidateEnv rejects an environment label the frozen key pattern cannot hold.
func ValidateEnv(env string) error {
	if !envRegexp.MatchString(env) {
		return fmt.Errorf("eventbus: env %q must match %s", env, envPattern)
	}
	return nil
}
