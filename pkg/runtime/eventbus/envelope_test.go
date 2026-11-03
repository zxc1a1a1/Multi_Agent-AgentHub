package eventbus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/agui"
)

// testClock is a fixed UTC instant so DLQ timestamps and occurredAt assertions
// are reproducible.
var testClock = time.Date(2026, 11, 2, 9, 30, 0, 0, time.UTC)

func testPayload(t *testing.T, fields map[string]any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return encoded
}

// newTestEnvelope builds a contract-conforming envelope through the public
// constructor, so every test also exercises the derivation rules.
func newTestEnvelope(t *testing.T, runID string, seq int64, subIndex int) Envelope {
	t.Helper()
	env, err := NewEnvelope(EnvelopeParams{
		Type:           "run_finished",
		RunID:          runID,
		ConversationID: "conv_test_1",
		TaskID:         "",
		Seq:            seq,
		SubIndex:       subIndex,
		OccurredAt:     testClock,
		TraceID:        "trace_test_1",
		Payload:        map[string]any{"type": "run_finished", "runId": runID, "status": "completed"},
	})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	return env
}

// repoFile locates a file relative to the repository root by walking up from the
// test working directory, so contract fixtures stay in one place.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, rel)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skipf("SKIP contract fixture %q is not reachable from the test working directory", rel)
	return ""
}

// EB-ENV-05 / section 2.1: the envelope field set is exactly the 13 frozen
// names, all required, with no additional properties.
func TestEnvelopeRequiredFields(t *testing.T) {
	want := []string{
		"eventId", "schemaVersion", "type", "runId", "conversationId", "taskId",
		"seq", "subIndex", "occurredAt", "traceId", "idempotencyKey", "attempt",
		"payload",
	}
	got := EnvelopeFields()
	if len(got) != 13 {
		t.Fatalf("contract freezes 13 fields, implementation declares %d", len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("field %d: want %q, got %q", i, want[i], got[i])
		}
	}

	env := newTestEnvelope(t, "run_fields", 1, 0)
	encoded, err := env.Encode()
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &asMap); err != nil {
		t.Fatal(err)
	}
	if len(asMap) != 13 {
		t.Fatalf("encoded envelope has %d keys, want 13: %v", len(asMap), keysOf(asMap))
	}
	for _, field := range want {
		if _, ok := asMap[field]; !ok {
			t.Fatalf("encoded envelope is missing required field %q", field)
		}
	}

	// A field without a value must serialize as the empty string, not disappear.
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	if value, ok := raw["taskId"]; !ok || value != "" {
		t.Fatalf("taskId must be present and empty for a task-unrelated event, got %#v (present=%v)", value, ok)
	}

	// Every missing required key is refused, one at a time.
	for _, field := range want {
		stripped := stripJSONField(t, encoded, field)
		if _, err := DecodeEnvelope(stripped); err == nil {
			t.Fatalf("dropping required field %q must be rejected", field)
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func stripJSONField(t *testing.T, data []byte, field string) []byte {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	delete(obj, field)
	encoded, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	return encoded
}

func addJSONField(t *testing.T, data []byte, field string, value any) []byte {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal extra field: %v", err)
	}
	obj[field] = encoded
	out, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	return out
}

// EB-ENV-01 / EB-IDM-01: eventId is a deterministic function of
// (runId, seq, subIndex). A relay that crashes and republishes must produce the
// same id, otherwise consumer dedup cannot work.
func TestEnvelopeEventIdIsDeterministic(t *testing.T) {
	want := "evt_run_order_7_2"
	if got := DeriveEventID("run_order", 7, 2); got != want {
		t.Fatalf("DeriveEventID = %q, want %q", got, want)
	}
	first, err := NewEnvelope(EnvelopeParams{
		Type: "message_delta", RunID: "run_order", ConversationID: "conv_a", TaskID: "task_a",
		Seq: 7, SubIndex: 2, OccurredAt: testClock,
		Payload: map[string]any{"type": "message_delta", "messageId": "msg_1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEnvelope(EnvelopeParams{
		Type: "message_delta", RunID: "run_order", ConversationID: "conv_a", TaskID: "task_a",
		Seq: 7, SubIndex: 2, OccurredAt: testClock.Add(time.Hour),
		Payload: map[string]any{"type": "message_delta", "messageId": "msg_1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.EventID != second.EventID || first.EventID != want {
		t.Fatalf("two publishers of the same logical event must agree: %q vs %q", first.EventID, second.EventID)
	}

	copied := first
	copied.EventID = "evt_" + randomishID()
	if err := copied.Validate(); err == nil {
		t.Fatal("a randomly derived eventId must be rejected (EB-ENV-01 forbids UUIDs)")
	} else if class := ErrorClassOf(err); class != ErrorClassDecode {
		t.Fatalf("bad eventId class = %q, want %q", class, ErrorClassDecode)
	}

	// The identity triple must agree between eventId and idempotencyKey
	// (EB-ENV-02: "二者不一致时 publisher 必须拒绝发布").
	skewed := first
	skewed.IdempotencyKey = DeriveIdempotencyKey(skewed.RunID, skewed.Seq+1, skewed.SubIndex)
	if err := skewed.Validate(); err == nil {
		t.Fatal("idempotencyKey inconsistent with eventId must be refused")
	}

	badPatterns := []string{"", "run", "run_bad id", "run_" + strings.Repeat("x", 70), "RUN_upper"}
	for _, runID := range badPatterns {
		env := first
		env.RunID = runID
		if err := env.Validate(); err == nil {
			t.Fatalf("runId %q must not satisfy ^run_[A-Za-z0-9_-]{1,64}$", runID)
		}
	}
}

func randomishID() string {
	// Not crypto-random: the point is only that it differs from the derived form.
	return fmt.Sprintf("uuid_%d", time.Now().UnixNano())
}

// EB-ENV-02: idempotencyKey == "<runId>:<seq>:<subIndex>", structurally the same
// as run_signals.idempotency_key and the projection cursor.
func TestIdempotencyKeyDerivation(t *testing.T) {
	if got := DeriveIdempotencyKey("run_a", 12, 0); got != "run_a:12:0" {
		t.Fatalf("DeriveIdempotencyKey = %q", got)
	}
	env := newTestEnvelope(t, "run_a", 12, 0)
	if env.IdempotencyKey != "run_a:12:0" {
		t.Fatalf("envelope idempotencyKey = %q", env.IdempotencyKey)
	}
	for _, bad := range []string{"", "run_a:12", "run_a:0:0", "run_a:12:", "run_a::0", strings.Repeat("r", 90) + ":1:0"} {
		copied := env
		copied.IdempotencyKey = bad
		if err := copied.Validate(); err == nil {
			t.Fatalf("idempotencyKey %q must be rejected", bad)
		}
	}
}

// section 2.1: subIndex participates in both derived identities, so two shards
// of one seq never collapse into the same dedup key.
func TestEnvelopeSubIndexParticipationInIdempotencyKey(t *testing.T) {
	zero := newTestEnvelope(t, "run_shard", 4, 0)
	one := newTestEnvelope(t, "run_shard", 4, 1)
	if zero.EventID == one.EventID {
		t.Fatalf("subIndex must participate in eventId: %q", zero.EventID)
	}
	if zero.IdempotencyKey == one.IdempotencyKey {
		t.Fatalf("subIndex must participate in idempotencyKey: %q", zero.IdempotencyKey)
	}
	if one.EventID != "evt_run_shard_4_1" || one.IdempotencyKey != "run_shard:4:1" {
		t.Fatalf("derived forms wrong: %q / %q", one.EventID, one.IdempotencyKey)
	}
	negative := one
	negative.SubIndex = -1
	if err := negative.Validate(); err == nil {
		t.Fatal("subIndex must be >= 0")
	}
}

// section 2.1: taskId is written as the empty string when the event has no A2A
// task; the key may never be omitted.
func TestEnvelopeTaskIdEmptyWhenIrrelevant(t *testing.T) {
	env := newTestEnvelope(t, "run_task", 1, 0)
	if env.TaskID != "" {
		t.Fatalf("taskId should default to empty, got %q", env.TaskID)
	}
	encoded, err := env.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"taskId":""`) {
		t.Fatalf("encoded envelope must keep an empty taskId key, got %s", encoded)
	}
	withTask := env
	withTask.TaskID = "task_0123"
	withTask.EventID = DeriveEventID(withTask.RunID, withTask.Seq, withTask.SubIndex)
	if err := withTask.Validate(); err != nil {
		t.Fatalf("a well formed taskId must be accepted: %v", err)
	}
	for _, bad := range []string{"task 1", "task_" + strings.Repeat("t", 70), "task_!"} {
		copied := withTask
		copied.TaskID = bad
		if err := copied.Validate(); err == nil {
			t.Fatalf("taskId %q must be rejected", bad)
		}
	}
}

// EB-ENV-04: occurredAt is UTC RFC3339 with a Z suffix, never a local offset.
func TestEnvelopeOccurredAtIsUtcRfc3339(t *testing.T) {
	located := time.FixedZone("CST", 8*3600)
	inShanghai := time.Date(2026, 11, 2, 17, 30, 0, 123456789, located)
	formatted := FormatOccurredAt(inShanghai)
	if !utcTimestampPattern.MatchString(formatted) {
		t.Fatalf("FormatOccurredAt produced %q, outside the schema pattern", formatted)
	}
	if !strings.HasSuffix(formatted, "Z") {
		t.Fatalf("FormatOccurredAt must render UTC, got %q", formatted)
	}
	if !strings.HasPrefix(formatted, "2026-11-02T09:30:00") {
		t.Fatalf("FormatOccurredAt did not normalize the zone: %q", formatted)
	}

	env := newTestEnvelope(t, "run_time", 1, 0)
	for _, bad := range []string{
		"2026-11-02T09:30:00+08:00",
		"2026-11-02 09:30:00",
		"2026-11-02T09:30:00",
		"2026-11-02T09:30:00.1234567890Z",
		"",
	} {
		copied := env
		copied.OccurredAt = bad
		if err := copied.Validate(); err == nil {
			t.Fatalf("occurredAt %q must be rejected", bad)
		} else if class := ErrorClassOf(err); class != ErrorClassDecode {
			t.Fatalf("occurredAt %q class = %q", bad, class)
		}
	}
}

// section 2.1: traceId may be empty (the relay has no trace source today) but
// inventing one is forbidden, and an invalid shape is refused.
func TestEnvelopeTraceIdEmptyIsValidButShaped(t *testing.T) {
	env := newTestEnvelope(t, "run_trace", 1, 0)
	cleared := env
	cleared.TraceID = ""
	if err := cleared.Validate(); err != nil {
		t.Fatalf("empty traceId must be accepted: %v", err)
	}
	for _, bad := range []string{"trace with space", "trace!" + strings.Repeat("x", 130)} {
		copied := env
		copied.TraceID = bad
		if err := copied.Validate(); err == nil {
			t.Fatalf("traceId %q must be rejected", bad)
		}
	}
}

// EB-TYPE-01 / EB-SEC-07: the allowlist is exactly the internal lower_snake
// names, matching both the agui package and the frozen schema enum.
func TestEnvelopeAllowlistCoversInternalTypes(t *testing.T) {
	ours := map[string]bool{}
	for _, typ := range AllowedTypes() {
		ours[typ] = true
	}
	fromCode := map[string]bool{}
	for internal := range agui.InternalToPublic {
		fromCode[internal] = true
	}
	if len(fromCode) != len(ours) {
		t.Fatalf("allowlist has %d types, pkg/runtime/agui declares %d", len(ours), len(fromCode))
	}
	for internal := range fromCode {
		if !ours[internal] {
			t.Fatalf("agui internal type %q is missing from the allowlist", internal)
		}
	}

	// Cross-check against the contract schema enum so the docs and code cannot
	// drift apart silently.
	path := repoFile(t, filepath.Join("docs", "contracts", "event-bus.schema.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties struct {
			Type struct {
				Enum []string `json:"enum"`
			} `json:"type"`
		} `json:"properties"`
		Defs struct {
			DLQEntry struct {
				Required   []string `json:"required"`
				Properties struct {
					ErrorClass struct {
						Enum []string `json:"enum"`
					} `json:"errorClass"`
				} `json:"properties"`
			} `json:"dlqEntry"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if len(schema.Required) != 13 {
		t.Fatalf("schema freezes %d required fields, want 13", len(schema.Required))
	}
	for i := range schema.Required {
		if schema.Required[i] != EnvelopeFields()[i] {
			t.Fatalf("field %d: schema %q vs implementation %q", i, schema.Required[i], EnvelopeFields()[i])
		}
	}
	if len(schema.Properties.Type.Enum) != len(ours) {
		t.Fatalf("schema enum has %d types, implementation has %d", len(schema.Properties.Type.Enum), len(ours))
	}
	for _, typ := range schema.Properties.Type.Enum {
		if !ours[typ] {
			t.Fatalf("schema type %q is missing from the implementation allowlist", typ)
		}
	}

	// EB-DEL-09 / $defs.dlqEntry: closed errorClass enum and the frozen
	// metadata-only required key set.
	schemaClasses := map[string]bool{}
	for _, class := range schema.Defs.DLQEntry.Properties.ErrorClass.Enum {
		schemaClasses[class] = true
	}
	implemented := map[string]bool{}
	for _, class := range ErrorClasses() {
		implemented[string(class)] = true
	}
	if len(schemaClasses) != len(implemented) {
		t.Fatalf("schema declares %d errorClasses, implementation %d", len(schemaClasses), len(implemented))
	}
	for class := range schemaClasses {
		if !implemented[class] {
			t.Fatalf("schema errorClass %q is missing from the implementation", class)
		}
	}
	if strings.Join(schema.Defs.DLQEntry.Required, ",") != "eventId,consumerKey,runId,seq,type,attempts,errorClass,firstSeen,lastSeen" {
		t.Fatalf("unexpected dlqEntry required set: %v", schema.Defs.DLQEntry.Required)
	}

	// Every allowlisted type must be publishable with a minimal payload.
	for _, typ := range AllowedTypes() {
		params := EnvelopeParams{
			Type: typ, RunID: "run_all", ConversationID: "conv_all", Seq: 1,
			OccurredAt: testClock, Payload: map[string]any{"type": typ},
		}
		switch typ {
		case "agent_turn_started", "agent_turn_content", "agent_turn_finished":
			params.Payload = map[string]any{"type": typ, "turnIndex": 1}
		case "message_start", "message_delta", "message_end":
			params.Payload = map[string]any{"type": typ, "messageId": "msg_1"}
		}
		if _, err := NewEnvelope(params); err != nil {
			t.Fatalf("allowlisted type %q must build: %v", typ, err)
		}
	}
}

// EB-TYPE-01/02/03/04: public UPPER_SNAKE names, translator dot names and the
// grpcapi "message" fallback are all outside the queue protocol.
func TestEnvelopeTypeNotAllowed(t *testing.T) {
	env := newTestEnvelope(t, "run_type", 1, 0)
	for _, typ := range []string{
		"RUN_FINISHED", "AGENT_TURN_CONTENT", "STATE_UPDATE",
		"artifact.delta", "message.end", "tool.call", "tool.result", "state.delta", "thinking",
		"message", "run.Finished", "", "run_finished ",
	} {
		copied := env
		copied.Type = typ
		err := copied.Validate()
		if err == nil {
			t.Fatalf("type %q must be refused by the allowlist", typ)
		}
		if class := ErrorClassOf(err); class != ErrorClassTypeNotAllowed {
			t.Fatalf("type %q class = %q, want %q", typ, class, ErrorClassTypeNotAllowed)
		}
	}
}

// EB-ENV-06: when the payload carries its own type it must equal the envelope
// type; JSON Schema cannot compare two positions, so this is an implementation
// rule with a dedicated test.
func TestPayloadTypeMismatchIsDecodeError(t *testing.T) {
	env := newTestEnvelope(t, "run_mismatch", 1, 0)
	if err := env.Validate(); err != nil {
		t.Fatal(err)
	}
	copied := env
	copied.Payload = testPayload(t, map[string]any{"type": "run_started", "status": "completed"})
	err := copied.Validate()
	if err == nil {
		t.Fatal("payload.type differing from envelope type must be refused")
	}
	if class := ErrorClassOf(err); class != ErrorClassDecode {
		t.Fatalf("class = %q, want %q", class, ErrorClassDecode)
	}
	// Payloads without a type key stay legal: today's run_finished/state_update
	// writers only emit {status, reason} (event-bus.md section 2.3).
	typeless := env
	typeless.Payload = testPayload(t, map[string]any{"status": "completed", "reason": "done"})
	if err := typeless.Validate(); err != nil {
		t.Fatalf("payload without type must stay valid: %v", err)
	}
	// Non-object payloads are malformed.
	for _, raw := range []string{`"just a string"`, `[1,2]`, `42`, `null`} {
		nonObject := env
		nonObject.Payload = json.RawMessage(raw)
		if err := nonObject.Validate(); err == nil {
			t.Fatalf("payload %s must be refused", raw)
		}
	}
}

// EB-SEC-01: an oversized envelope is rejected, never truncated.
func TestOversizedEnvelopeIsRejectedNotTruncated(t *testing.T) {
	env := newTestEnvelope(t, "run_big", 1, 0)
	env.Payload = testPayload(t, map[string]any{"type": "run_finished", "status": strings.Repeat("x", MaxEnvelopeBytes)})
	env.EventID = DeriveEventID(env.RunID, env.Seq, env.SubIndex)
	err := env.Validate()
	if err == nil {
		t.Fatal("an envelope above 64KB must be refused")
	}
	if class := ErrorClassOf(err); class != ErrorClassSizeExceeded {
		t.Fatalf("class = %q, want %q", class, ErrorClassSizeExceeded)
	}
	if len(env.MustEncode()) <= MaxEnvelopeBytes {
		t.Fatal("test fixture is not oversized, the assertion above is vacuous")
	}

	big := newTestEnvelope(t, "run_frame", 1, 0)
	frame := big.MustEncode()
	// The frame guard fires before parsing, so an oversized read cannot be
	// mistaken for a decodable envelope.
	padded := addJSONField(t, frame, "unknownField", strings.Repeat("y", MaxEnvelopeBytes))
	if _, err := DecodeEnvelope(padded); err == nil {
		t.Fatal("oversized frame must be refused")
	} else if class := ErrorClassOf(err); class != ErrorClassSizeExceeded {
		t.Fatalf("oversized frame class = %q, want %q", class, ErrorClassSizeExceeded)
	}

	// A small envelope still round-trips, proving the check is a byte budget and
	// not an accidental "reject everything".
	fits := newTestEnvelope(t, "run_fit", 1, 0)
	if len(fits.MustEncode()) > MaxEnvelopeBytes {
		t.Fatalf("baseline fixture must be small, got %d bytes", len(fits.MustEncode()))
	}
	if _, err := DecodeEnvelope(fits.MustEncode()); err != nil {
		t.Fatalf("baseline fixture must decode: %v", err)
	}
}

// EB-SEC-02: prompts, tokens, secrets, DSNs and blobs may never enter the queue.
func TestPayloadCarriesNoSecrets(t *testing.T) {
	env := newTestEnvelope(t, "run_secret", 1, 0)
	for _, key := range ForbiddenPayloadKeys() {
		copied := env
		copied.Payload = testPayload(t, map[string]any{"type": "run_finished", key: "anything"})
		err := copied.Validate()
		if err == nil {
			t.Fatalf("payload key %q must be blocked", key)
		}
		if class := ErrorClassOf(err); class != ErrorClassSecurityRedactionRequired {
			t.Fatalf("payload key %q class = %q", key, class)
		}
	}
	// A legitimate payload with similar-looking but allowed keys still passes.
	ok := env
	ok.Payload = testPayload(t, map[string]any{"type": "run_finished", "status": "completed", "artifactId": "art_abcdef12"})
	if err := ok.Validate(); err != nil {
		t.Fatalf("clean payload rejected: %v", err)
	}
}

// EB-SEC-06: run_error may only carry the redacted SafeError projection.
func TestRunErrorPayloadIsRedacted(t *testing.T) {
	build := func(payload map[string]any) error {
		env, err := NewEnvelope(EnvelopeParams{
			Type: "run_error", RunID: "run_err", ConversationID: "conv_e", Seq: 1,
			OccurredAt: testClock, Payload: payload,
		})
		if err != nil {
			return err
		}
		return env.Validate()
	}
	if err := build(map[string]any{"type": "run_error", "error": map[string]any{"code": "ORCHESTRATOR_AGENT_UNAVAILABLE", "message": "agent unavailable"}}); err != nil {
		t.Fatalf("redacted SafeError must be accepted: %v", err)
	}
	for _, forbidden := range []string{"stackTrace", "rawError"} {
		err := build(map[string]any{"type": "run_error", forbidden: "goroutine 1 [running]"})
		if err == nil {
			t.Fatalf("run_error payload must not carry %q", forbidden)
		}
		if class := ErrorClassOf(err); class != ErrorClassSecurityRedactionRequired {
			t.Fatalf("%q class = %q", forbidden, class)
		}
	}
}

// Schema allOf conditions: AGENT_TURN needs turnIndex, text stream needs messageId.
func TestConditionalPayloadRequirements(t *testing.T) {
	_, err := NewEnvelope(EnvelopeParams{
		Type: "agent_turn_content", RunID: "run_turn", ConversationID: "conv_t", Seq: 1,
		OccurredAt: testClock, Payload: map[string]any{"type": "agent_turn_content", "delta": "hi"},
	})
	if err == nil {
		t.Fatal("agent_turn_content without turnIndex must be refused")
	}
	if class := ErrorClassOf(err); class != ErrorClassDecode {
		t.Fatalf("class = %q", class)
	}
	// turnIndex 0 is a real value (the first turn) and must survive.
	if _, err := NewEnvelope(EnvelopeParams{
		Type: "agent_turn_content", RunID: "run_turn", ConversationID: "conv_t", Seq: 1,
		OccurredAt: testClock, Payload: map[string]any{"type": "agent_turn_content", "turnIndex": 0},
	}); err != nil {
		t.Fatalf("turnIndex 0 must be accepted: %v", err)
	}
	if _, err := NewEnvelope(EnvelopeParams{
		Type: "message_delta", RunID: "run_msg", ConversationID: "conv_m", Seq: 1,
		OccurredAt: testClock, Payload: map[string]any{"type": "message_delta", "delta": "hi"},
	}); err == nil {
		t.Fatal("message_delta without messageId must be refused")
	}
}

// EB-IDM: the publisher writes attempt=1 and never claims a higher count; the
// retry budget belongs to the consumer/adapter, so it must stay bounded.
func TestAttemptCountedByConsumerNotPublisher(t *testing.T) {
	env := newTestEnvelope(t, "run_attempt", 1, 0)
	if env.Attempt != 1 {
		t.Fatalf("a fresh envelope must carry attempt=1, got %d", env.Attempt)
	}
	second := env
	second.Attempt = 2
	if err := second.ValidateForPublish(); err == nil {
		t.Fatal("a publisher must not announce attempt=2")
	}
	if err := second.Validate(); err != nil {
		t.Fatalf("attempt=2 is a legal consumer-side value: %v", err)
	}
	over := env
	over.Attempt = MaxAttemptsCap + 1
	if err := over.Validate(); err == nil {
		t.Fatalf("attempt above %d must be refused", MaxAttemptsCap)
	}
	zero := env
	zero.Attempt = 0
	if err := zero.Validate(); err == nil {
		t.Fatal("attempt=0 must be refused (schema minimum 1)")
	}
	if err := env.ValidateForPublish(); err != nil {
		t.Fatalf("attempt=1 must be publishable: %v", err)
	}
}

// EB-DEL-08 / EB-SEC-04: a DLQ entry is metadata only. Nothing in the serialized
// entry may echo the payload.
func TestDlqEntryCarriesNoPayload(t *testing.T) {
	env := newTestEnvelope(t, "run_dlq", 3, 0)
	env.Payload = testPayload(t, map[string]any{
		"type": "run_finished", "status": "completed", "artifactId": "art_secretish",
		"canary": "payload-must-never-reach-the-dlq",
	})
	entry := NewDLQEntry(env, "assistant_message.v1", 5, ErrorClassHandlerError, StreamKeyFor("dev"), "1730500000000-0", testClock)
	if err := entry.Validate(); err != nil {
		t.Fatalf("entry must validate: %v", err)
	}
	encoded, err := entry.Encode()
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	// The entry legitimately mirrors eventId/consumerKey/runId/seq/type/attempts/
	// errorClass/timestamps; everything payload-shaped must be absent.
	for _, leak := range []string{"payload", "canary", "payload-must-never-reach-the-dlq", "completed", "artifactId", "art_secretish", "status"} {
		if strings.Contains(text, leak) {
			t.Fatalf("DLQ entry leaked %q into %s", leak, text)
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, key := range DLQFields() {
		allowed[key] = true
	}
	for key := range fields {
		if !allowed[key] {
			t.Fatalf("DLQ entry carries an undefined field %q (schema additionalProperties:false)", key)
		}
	}
	for _, required := range []string{"eventId", "consumerKey", "runId", "seq", "type", "attempts", "errorClass", "firstSeen", "lastSeen"} {
		if _, ok := fields[required]; !ok {
			t.Fatalf("DLQ entry is missing required field %q", required)
		}
	}
	if entry.ErrorClass != ErrorClassHandlerError || entry.Attempts != 5 {
		t.Fatalf("entry fields lost information: %#v", entry)
	}
	if entry.StreamKey != "agenthub:dev:run:events" || entry.StreamEntryID != "1730500000000-0" {
		t.Fatalf("optional operational fields wrong: %#v", entry)
	}

	// An unknown errorClass is not representable: it degrades to "unknown".
	fallback := NewDLQEntry(env, "audit.v1", 1, ErrorClass("provider_http_500"), "", "", testClock)
	if fallback.ErrorClass != ErrorClassUnknown {
		t.Fatalf("errorClass must degrade to unknown, got %q", fallback.ErrorClass)
	}
	// consumerKey must stay inside the schema pattern.
	bad := fallback
	bad.ConsumerKey = "Bad Consumer"
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid consumerKey must be refused")
	}
	// attempts can never be recorded as unbounded.
	clamped := NewDLQEntry(env, "audit.v1", 999, ErrorClassHandlerError, "", "", testClock)
	if clamped.Attempts > MaxAttemptsCap {
		t.Fatalf("attempts must be clamped to the retry budget, got %d", clamped.Attempts)
	}
}

// section 4.1: the frozen stream keys and the env label shape.
func TestStreamKeysFollowContract(t *testing.T) {
	if got := StreamKeyFor("dev"); got != "agenthub:dev:run:events" {
		t.Fatalf("run stream key = %q", got)
	}
	if got := DLQStreamKeyFor("prod"); got != "agenthub:prod:dlq" {
		t.Fatalf("dlq stream key = %q", got)
	}
	if !streamKeyPattern.MatchString(StreamKeyFor("dev")) {
		t.Fatal("run stream key must satisfy the schema pattern")
	}
	for _, bad := range []string{"", "Dev", "dev/prod", strings.Repeat("d", 33)} {
		if err := ValidateEnv(bad); err == nil {
			t.Fatalf("env %q must be rejected", bad)
		}
	}
	if err := ValidateEnv("dev"); err != nil {
		t.Fatalf("dev must be accepted: %v", err)
	}
}

// The envelope must survive a transport round trip byte-for-byte in meaning.
func TestEnvelopeEncodeDecodeRoundTrip(t *testing.T) {
	for _, typ := range AllowedTypes() {
		params := EnvelopeParams{
			Type: typ, RunID: "run_rt", ConversationID: "conv_rt", TaskID: "task_rt", Seq: 9,
			SubIndex: 1, OccurredAt: testClock, TraceID: "trace_rt",
			Payload: map[string]any{"type": typ, "turnIndex": 2, "messageId": "msg_rt", "status": "completed"},
		}
		env, err := NewEnvelope(params)
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		frame, err := env.Encode()
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		decoded, err := DecodeEnvelope(frame)
		if err != nil {
			t.Fatalf("%s decode: %v", typ, err)
		}
		if decoded.EventID != env.EventID || decoded.Seq != env.Seq || decoded.SubIndex != env.SubIndex ||
			decoded.Type != env.Type || decoded.TraceID != env.TraceID || decoded.ConversationID != env.ConversationID {
			t.Fatalf("%s: round trip lost fields: %#v", typ, decoded)
		}
		if string(decoded.Payload) != string(env.Payload) {
			t.Fatalf("%s: payload changed: %s vs %s", typ, decoded.Payload, env.Payload)
		}
	}
}

// DecodeEnvelope must refuse an unknown envelope key (EB-ENV-05) while still
// parsing today's known set.
func TestDecodeRejectsUnknownEnvelopeField(t *testing.T) {
	env := newTestEnvelope(t, "run_extra", 1, 0)
	extended := addJSONField(t, env.MustEncode(), "workspaceId", "ws_1")
	if _, err := DecodeEnvelope(extended); err == nil {
		t.Fatal("an extra envelope field must be refused at publish time")
	} else if class := ErrorClassOf(err); class != ErrorClassDecode {
		t.Fatalf("class = %q", class)
	}
	// A completely undecodable frame is a decode_error, not a panic.
	for _, garbage := range []string{"", "{", "not json", "[]", "null"} {
		if _, err := DecodeEnvelope([]byte(garbage)); err == nil {
			t.Fatalf("frame %q must be refused", garbage)
		} else if class := ErrorClassOf(err); class != ErrorClassDecode && class != ErrorClassSizeExceeded {
			t.Fatalf("frame %q class = %q", garbage, class)
		}
	}
}

// Consumers must refuse an unknown schemaVersion instead of guessing (section 14).
func TestEnvelopeUnknownSchemaVersion(t *testing.T) {
	env := newTestEnvelope(t, "run_version", 1, 0)
	copied := env
	copied.SchemaVersion = 2
	err := copied.Validate()
	if err == nil {
		t.Fatal("schemaVersion 2 must not be parsed by a version 1 consumer")
	}
	if class := ErrorClassOf(err); class != ErrorClassUnknownSchemaVersion {
		t.Fatalf("class = %q, want %q", class, ErrorClassUnknownSchemaVersion)
	}
	if SchemaVersion != 1 {
		t.Fatalf("schemaVersion is frozen at 1, got %d", SchemaVersion)
	}
}

// EB-ENV-03: seq starts at 1 and is the ordering key.
func TestEnvelopeSeqMustStartAtOne(t *testing.T) {
	env := newTestEnvelope(t, "run_seq", 1, 0)
	if env.Seq != 1 {
		t.Fatalf("seq %d", env.Seq)
	}
	zero := env
	zero.Seq = 0
	if err := zero.Validate(); err == nil {
		t.Fatal("seq=0 must be refused")
	}
	negative := env
	negative.Seq = -1
	if err := negative.Validate(); err == nil {
		t.Fatal("negative seq must be refused")
	}
}

// The contract testdata directory is the acceptance fixture of R2-01; the
// implementation must agree with it on every example.
func TestEnvelopeSchemaExamplesMatchContract(t *testing.T) {
	dir := repoFile(t, filepath.Join("docs", "contracts", "testdata", "event-bus"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}
	var valid, invalid int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		shouldFail := strings.Contains(entry.Name(), ".should_fail.")
		if shouldFail {
			invalid++
		} else {
			valid++
		}
		env, err := DecodeEnvelope(data)
		if shouldFail {
			if err == nil {
				t.Fatalf("counter-example %s must be rejected, but it validated as %#v", entry.Name(), env)
			}
			continue
		}
		if err != nil {
			t.Fatalf("positive example %s must validate: %v", entry.Name(), err)
		}
		if env.Attempt != 1 {
			t.Fatalf("positive example %s should carry attempt=1", entry.Name())
		}
	}
	if valid != 14 || invalid != 7 {
		t.Fatalf("fixture set changed shape: %d positive, %d negative (want 14/7)", valid, invalid)
	}
}
