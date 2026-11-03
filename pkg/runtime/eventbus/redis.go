package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Frozen consumer groups of event-bus.md section 4.2. One group per duty; each
// group keeps its own PEL so a backlog in projection cannot stall signal (EB-GRP-01).
const (
	GroupProjection = "projection"
	GroupSignal     = "signal"
	GroupAudit      = "audit"
)

// Frozen consumerKey values (the idempotency/cursor key of each group). These
// match the SQL side: assistant_message.v1 already exists in the gateway store,
// run_signal.v1 mirrors run_signals, audit.v1 is the new accounting key.
const (
	ConsumerKeyProjection = "assistant_message.v1"
	ConsumerKeySignal     = "run_signal.v1"
	ConsumerKeyAudit      = "audit.v1"
)

// DefaultStreamField is the single XADD field name carrying the envelope JSON
// (section 4.3: "只使用 data 一个字段"). Secondary index fields are forbidden, so
// observability must come from XLEN/lag rather than a duplicated runId/seq column.
const DefaultStreamField = "data"

const (
	DefaultBlock        = 500 * time.Millisecond
	DefaultClaimIdle    = 5 * time.Second
	DefaultReadCount    = 100
	DefaultClaimPeriod  = time.Second
	DefaultGroupStart   = "0"
	busyGroupErrorToken = "BUSYGROUP"
)

// ConsumerKeyForGroup returns the frozen consumerKey of a known group, or ""
// when the group is not part of the contract.
func ConsumerKeyForGroup(group string) string {
	switch group {
	case GroupProjection:
		return ConsumerKeyProjection
	case GroupSignal:
		return ConsumerKeySignal
	case GroupAudit:
		return ConsumerKeyAudit
	default:
		return ""
	}
}

// TransportError marks a middleware failure: Redis unreachable, a command error,
// a write that never landed. It classifies as store_error and keeps the wrapped
// cause for logs only; a DLQ entry never takes text from it (EB-DEL-09).
type TransportError struct {
	Op  string
	Err error
}

func (e *TransportError) Error() string {
	if e == nil {
		return "eventbus: transport error"
	}
	return "eventbus: " + e.Op + " failed: " + e.Err.Error()
}

// Unwrap keeps errors.Is working against redis.Nil, context.DeadlineExceeded and
// friends without exposing their text to metrics or DLQ rows.
func (e *TransportError) Unwrap() error { return e.Err }

// ErrorClassOf recognises a transport failure as store_error.
func classifyTransport(err error) bool {
	var transport *TransportError
	return errors.As(err, &transport)
}

// RedisConfig is the whole configuration surface of the adapter: address,
// credentials by reference, stream/group/consumer identity, trim bound, claim
// window and block time. Nothing here may carry a literal DSN into a log line.
type RedisConfig struct {
	// Addr, Password and DB are the connection. Password is read from an
	// environment variable by the caller; this package never reads .env.
	Addr     string
	Password string
	DB       int

	// Env selects the frozen shared keys agenthub:{env}:run:events and
	// agenthub:{env}:dlq. StreamKey/DLQStreamKey override it for tests only.
	Env          string
	StreamKey    string
	DLQStreamKey string
	StreamField  string

	// Group is one of projection|signal|audit. Consumer is the instance name
	// (EB-GRP-02) and ConsumerKey the group's idempotency key; ConsumerKey
	// defaults from Group.
	Group       string
	Consumer    string
	ConsumerKey string

	// GroupStart is the XGROUP CREATE start id. "0" is the default so a group
	// created after events already flowed does not silently skip them; SQL
	// replay stays authoritative either way (EB-INV-03).
	GroupStart string

	// MaxLen and DLQMaxLen bound the two streams with XADD MAXLEN ~. A zero
	// value falls back to the contract's recommended bound, never to "unlimited".
	MaxLen    int64
	DLQMaxLen int64

	// StreamTTL and DLQStreamTTL keep both keys inside the repository rule that
	// forbids business keys without an expiry (docs/contracts/redis-usage.md §6).
	// SQL outbox stays authoritative either way.
	StreamTTL    time.Duration
	DLQStreamTTL time.Duration

	// ClaimMinIdle is the XAUTOCLAIM idle threshold; ClaimPeriod is how often
	// the reclaim pass runs.
	ClaimMinIdle time.Duration
	ClaimPeriod  time.Duration

	// Block is the XREADGROUP BLOCK duration. It must stay positive, because
	// BLOCK 0 means "block forever" in Redis and would make the loop
	// unkillable by a slow ticker.
	Block time.Duration
	// Count is the per-round batch size for reads, claims and pending scans.
	Count int64

	// MaxAttempts bounds consumer-side re-delivery before the DLQ (EB-DEL-04).
	MaxAttempts int

	// QueueSize bounds the per-run in-flight queue of the adapter.
	QueueSize int

	// DisableDedupe turns off the process-local duplicate guard. It is on by
	// default so a re-delivered eventId cannot produce a second business effect
	// inside one process; EB-IDM-06 forbids treating that map as the only dedup
	// source, which is why the durable table belongs to R2-05 and this adapter
	// must stay unwired until then.
	DisableDedupe bool

	// Hooks receives publish/consume/ack/retry/dlq/pending notifications.
	Hooks *MetricsHooks

	// Now supplies timestamps for DLQ firstSeen/lastSeen (default time.Now).
	Now func() time.Time

	// Client overrides the built client; used by tests that share one pool or
	// need a deliberately broken option set.
	Client redis.UniversalClient
}

func (c RedisConfig) normalized() (RedisConfig, error) {
	if c.Env == "" {
		c.Env = "dev"
	}
	if err := ValidateEnv(c.Env); err != nil {
		return c, err
	}
	if c.StreamKey == "" {
		c.StreamKey = StreamKeyFor(c.Env)
	}
	if c.DLQStreamKey == "" {
		c.DLQStreamKey = DLQStreamKeyFor(c.Env)
	}
	if !streamKeyPattern.MatchString(c.StreamKey) {
		return c, fmt.Errorf("eventbus: stream key %q does not match the frozen pattern", c.StreamKey)
	}
	if c.StreamField == "" {
		c.StreamField = DefaultStreamField
	}
	if c.Group == "" {
		c.Group = GroupProjection
	}
	if c.ConsumerKey == "" {
		c.ConsumerKey = ConsumerKeyForGroup(c.Group)
	}
	if c.ConsumerKey == "" {
		if !consumerKeyPattern.MatchString(c.Group) {
			return c, fmt.Errorf("eventbus: group %q needs an explicit consumerKey", c.Group)
		}
		c.ConsumerKey = c.Group
	}
	if !consumerKeyPattern.MatchString(c.ConsumerKey) {
		return c, fmt.Errorf("eventbus: consumerKey %q is outside the schema pattern", c.ConsumerKey)
	}
	if c.Consumer == "" {
		c.Consumer = "eventbus-local"
	}
	if c.GroupStart == "" {
		c.GroupStart = DefaultGroupStart
	}
	if c.MaxLen <= 0 {
		c.MaxLen = DefaultStreamMaxLen
	}
	if c.DLQMaxLen <= 0 {
		c.DLQMaxLen = DefaultDLQMaxLen
	}
	if c.StreamTTL <= 0 {
		c.StreamTTL = DefaultStreamTTL
	}
	if c.DLQStreamTTL <= 0 {
		c.DLQStreamTTL = DefaultDLQStreamTTL
	}
	if c.ClaimMinIdle <= 0 {
		c.ClaimMinIdle = DefaultClaimIdle
	}
	if c.ClaimPeriod <= 0 {
		c.ClaimPeriod = DefaultClaimPeriod
	}
	if c.Block <= 0 {
		c.Block = DefaultBlock
	}
	if c.Count <= 0 {
		c.Count = DefaultReadCount
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = DefaultMaxAttempts
	}
	if c.MaxAttempts > MaxAttemptsCap {
		c.MaxAttempts = MaxAttemptsCap
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 256
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Addr == "" && c.Client == nil {
		return c, errors.New("eventbus: RedisConfig requires Addr or Client")
	}
	return c, nil
}

// newClient builds the go-redis client for the configuration. Timeouts stay at
// the library defaults unless the caller sets them.
func (c RedisConfig) newClient() redis.UniversalClient {
	if c.Client != nil {
		return c.Client
	}
	return redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})
}

// RedisPublisher writes contract envelopes into the shared run stream.
type RedisPublisher struct {
	client redis.UniversalClient
	cfg    RedisConfig
	owned  bool
	closed bool
	mu     sync.Mutex
}

// NewRedisPublisher connects using cfg.Addr/Password/DB.
func NewRedisPublisher(cfg RedisConfig) (*RedisPublisher, error) {
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	return &RedisPublisher{client: normalized.newClient(), cfg: normalized, owned: cfg.Client == nil}, nil
}

// NewRedisPublisherWithClient reuses a caller-owned client, so a relay and a
// consumer can share one pool. The client is not closed by Close in that case.
func NewRedisPublisherWithClient(cfg RedisConfig, client redis.UniversalClient) (*RedisPublisher, error) {
	if client == nil {
		return nil, errors.New("eventbus: a Redis client is required")
	}
	cfg.Client = client
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	return &RedisPublisher{client: client, cfg: normalized}, nil
}

// StreamKey exposes the frozen key this publisher writes.
func (p *RedisPublisher) StreamKey() string { return p.cfg.StreamKey }

// Publish validates the envelope, then XADDs it with MAXLEN ~ (section 4.3).
// A failure leaves the SQL outbox pending (EB-DEL-07); this call never retries
// into the queue and never bypasses the outbox.
func (p *RedisPublisher) Publish(ctx context.Context, envelope Envelope) error {
	if err := envelope.ValidateForPublish(); err != nil {
		p.cfg.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	frame, err := envelope.Encode()
	if err != nil {
		p.cfg.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		err := ErrClosedBus
		p.cfg.Hooks.publish(envelope, OutcomeError, err)
		return err
	}
	pipe := p.client.TxPipeline()
	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: p.cfg.StreamKey,
		MaxLen: p.cfg.MaxLen,
		Approx: true, // the "~" matcher: MAXLEN ~ N, never a bare MAXLEN
		Values: map[string]any{p.cfg.StreamField: string(frame)},
	})
	pipe.Expire(ctx, p.cfg.StreamKey, p.cfg.StreamTTL)
	if _, err = pipe.Exec(ctx); err != nil {
		transport := &TransportError{Op: "xadd", Err: err}
		p.cfg.Hooks.publish(envelope, OutcomeError, transport)
		return transport
	}
	p.cfg.Hooks.publish(envelope, OutcomeOK, nil)
	return nil
}

// Ping checks reachability so a caller can decide to keep the outbox pending.
func (p *RedisPublisher) Ping(ctx context.Context) error {
	if err := p.client.Ping(ctx).Err(); err != nil {
		return &TransportError{Op: "ping", Err: err}
	}
	return nil
}

// StreamLength reports XLEN of the run stream, the lag signal the contract
// prefers over a secondary index (section 4.3).
func (p *RedisPublisher) StreamLength(ctx context.Context) (int64, error) {
	length, err := p.client.XLen(ctx, p.cfg.StreamKey).Result()
	if err != nil {
		return 0, &TransportError{Op: "xlen", Err: err}
	}
	return length, nil
}

// Close releases a client this publisher created. It is idempotent.
func (p *RedisPublisher) Close(_ context.Context) error {
	p.mu.Lock()
	already := p.closed
	p.closed = true
	p.mu.Unlock()
	if already || !p.owned {
		return nil
	}
	return p.client.Close()
}

// RedisConsumer is one consumer group's reader: XREADGROUP + XACK + XAUTOCLAIM,
// with a bounded retry budget and a metadata-only DLQ.
type RedisConsumer struct {
	client redis.UniversalClient
	cfg    RedisConfig
	owned  bool

	mu        sync.Mutex
	groupDone bool
	seen      map[string]struct{}
	dispatch  *runDispatcher
	running   bool
	closed    bool
	closeOnce sync.Once
	quit      chan struct{}

	wg sync.WaitGroup
}

// NewRedisConsumer connects using cfg.Addr/Password/DB.
func NewRedisConsumer(cfg RedisConfig) (*RedisConsumer, error) {
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	return newRedisConsumer(normalized, normalized.newClient(), cfg.Client == nil), nil
}

// NewRedisConsumerWithClient reuses a caller-owned client.
func NewRedisConsumerWithClient(cfg RedisConfig, client redis.UniversalClient) (*RedisConsumer, error) {
	if client == nil {
		return nil, errors.New("eventbus: a Redis client is required")
	}
	cfg.Client = client
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	return newRedisConsumer(normalized, client, false), nil
}

func newRedisConsumer(cfg RedisConfig, client redis.UniversalClient, owned bool) *RedisConsumer {
	return &RedisConsumer{
		client: client,
		cfg:    cfg,
		owned:  owned,
		seen:   make(map[string]struct{}),
		quit:   make(chan struct{}),
	}
}

// StreamKey, Group and ConsumerKey expose the frozen identity of this consumer.
func (c *RedisConsumer) StreamKey() string    { return c.cfg.StreamKey }
func (c *RedisConsumer) Group() string        { return c.cfg.Group }
func (c *RedisConsumer) ConsumerKey() string  { return c.cfg.ConsumerKey }
func (c *RedisConsumer) DLQStreamKey() string { return c.cfg.DLQStreamKey }

// EnsureGroup creates the consumer group with mkstream, idempotently: a
// BUSYGROUP reply means another instance (or a previous start) already did it
// and is treated as success (EB-GRP-03).
func (c *RedisConsumer) EnsureGroup(ctx context.Context) error {
	c.mu.Lock()
	ready := c.groupDone
	c.mu.Unlock()
	if ready {
		return nil
	}
	err := c.client.XGroupCreateMkStream(ctx, c.cfg.StreamKey, c.cfg.Group, c.cfg.GroupStart).Err()
	if err != nil && !isBusyGroup(err) {
		return &TransportError{Op: "xgroup_create", Err: err}
	}
	c.mu.Lock()
	c.groupDone = true
	c.mu.Unlock()
	return nil
}

func isBusyGroup(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), busyGroupErrorToken)
}

// Consume runs the read loop plus the periodic reclaim pass until ctx is
// cancelled or Close is called. Same-run deliveries are serial (EB-ORD-01);
// different runs share the loop's parallelism (EB-ORD-06).
func (c *RedisConsumer) Consume(ctx context.Context, handler Handler) error {
	if handler == nil {
		return errors.New("eventbus: Consume requires a handler")
	}
	if err := c.EnsureGroup(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	if c.closed || c.running {
		c.mu.Unlock()
		if c.closed {
			return ErrClosedBus
		}
		return errors.New("eventbus: Consume is already running for this consumer")
	}
	c.running = true
	dispatcher := newRunDispatcher(c.cfg.QueueSize)
	c.dispatch = dispatcher
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.readLoop(ctx, dispatcher, handler)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.claimLoop(ctx, dispatcher, handler)
	}()

	var reason error
	select {
	case <-ctx.Done():
		reason = ctx.Err()
	case <-c.quit:
	}
	// Wait for in-flight handlers before returning: a consumer that stops while
	// a handler is still running would ack after the fact.
	dispatcher.stop()
	c.wg.Wait()

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	return reason
}

// readLoop performs XREADGROUP ... BLOCK ... STREAMS key > and hands every
// entry to the owning run's worker.
func (c *RedisConsumer) readLoop(ctx context.Context, dispatcher *runDispatcher, handler Handler) {
	streams := []string{c.cfg.StreamKey, ">"}
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		c.mu.Lock()
		stopped := c.closed
		c.mu.Unlock()
		if stopped {
			return
		}
		res, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.Group,
			Consumer: c.cfg.Consumer,
			Streams:  streams,
			Count:    c.cfg.Count,
			Block:    c.cfg.Block,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			// A transport error is not a message error: nothing was acked, so no
			// event is lost; the loop retries after the block window (EB-DEL-07).
			c.cfg.Hooks.consume(nil, OutcomeError, &TransportError{Op: "xreadgroup", Err: err})
			if !sleepOrDone(ctx, c.cfg.Block) {
				return
			}
			continue
		}
		for _, stream := range res {
			for _, entry := range stream.Messages {
				c.accept(ctx, dispatcher, handler, entry, false, 1)
			}
		}
	}
}

// claimLoop reclaims messages that sat idle in the pending list longer than
// ClaimMinIdle, which is how a crashed consumer's work is picked up (EB-DEL-03).
func (c *RedisConsumer) claimLoop(ctx context.Context, dispatcher *runDispatcher, handler Handler) {
	ticker := time.NewTicker(c.cfg.ClaimPeriod)
	defer ticker.Stop()
	// Reclaim once immediately: a restarted instance should pick up its own
	// unacked work before reading new entries.
	if _, err := c.ClaimOnce(ctx, dispatcher, handler); err != nil && ctx.Err() == nil {
		c.cfg.Hooks.consume(nil, OutcomeError, err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := c.ClaimOnce(ctx, dispatcher, handler); err != nil && ctx.Err() == nil {
				c.cfg.Hooks.consume(nil, OutcomeError, err)
			}
		}
	}
}

// ClaimOnce runs one reclaim pass: it reads the pending list to learn each
// entry's delivery count, dead-letters whatever has spent its budget, then
// XAUTOCLAIMs the idle remainder back to this consumer for re-delivery.
// It returns the number of entries re-delivered to the handler.
func (c *RedisConsumer) ClaimOnce(ctx context.Context, dispatcher *runDispatcher, handler Handler) (int, error) {
	if err := c.EnsureGroup(ctx); err != nil {
		return 0, err
	}
	pending, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.cfg.StreamKey,
		Group:  c.cfg.Group,
		Start:  "-",
		End:    "+",
		Count:  c.cfg.Count,
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, &TransportError{Op: "xpending", Err: err}
	}
	deliveries := make(map[string]int64, len(pending))
	for _, item := range pending {
		deliveries[item.ID] = item.RetryCount
	}
	if summary, err := c.client.XPending(ctx, c.cfg.StreamKey, c.cfg.Group).Result(); err == nil {
		c.cfg.Hooks.pending(c.cfg.Group, summary.Count)
	}

	reclaimed, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.cfg.StreamKey,
		Group:    c.cfg.Group,
		Consumer: c.cfg.Consumer,
		MinIdle:  c.cfg.ClaimMinIdle,
		Start:    "0",
		Count:    c.cfg.Count,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, &TransportError{Op: "xautoclaim", Err: err}
	}
	delivered := 0
	for _, entry := range reclaimed {
		// XAUTOCLAIM already incremented the delivery counter; the pending scan
		// above ran before the claim, so RetryCount is the previous delivery.
		attempt := int(deliveries[entry.ID]) + 1
		if attempt < 2 {
			attempt = 2
		}
		if attempt > c.cfg.MaxAttempts {
			// The budget is spent: dead-letter (metadata only) and ack so the
			// PEL cannot grow forever (EB-DEL-04).
			if err := c.dropToDlq(ctx, c.cfg.StreamField, entry.ID, entry.Values, ErrorClassHandlerError, attempt); err != nil {
				return delivered, err
			}
			continue
		}
		if c.dispatchThrough(ctx, dispatcher, handler, entry, true, attempt) {
			delivered++
		}
	}
	return delivered, nil
}

// accept decodes one stream entry and either dead-letters it immediately or
// hands it to the per-run worker.
func (c *RedisConsumer) accept(ctx context.Context, dispatcher *runDispatcher, handler Handler, entry redis.XMessage, redelivered bool, attempt int) {
	c.dispatchThrough(ctx, dispatcher, handler, entry, redelivered, attempt)
}

func (c *RedisConsumer) dispatchThrough(ctx context.Context, dispatcher *runDispatcher, handler Handler, entry redis.XMessage, redelivered bool, attempt int) bool {
	raw, ok := entry.Values[c.cfg.StreamField]
	if !ok {
		raw = nil
	}
	frame, err := frameBytes(raw)
	if err != nil {
		// The entry does not even carry a frame: metadata-only DLQ, then ack.
		c.dlqUndecodable(ctx, entry, ErrorClassDecode)
		return false
	}
	env, err := DecodeEnvelope(frame)
	if err != nil {
		class := ErrorClassOf(err)
		if class == ErrorClassUnknown {
			class = ErrorClassDecode
		}
		// EB-DEL-06: malformed frames go straight to the DLQ on the first
		// sighting, because retrying cannot fix a bad envelope.
		c.dlqPartial(ctx, entry, frame, class)
		return false
	}
	msg := &Message{
		Envelope:    env,
		TransportID: entry.ID,
		Attempt:     attempt,
		Group:       c.cfg.Group,
		ConsumerKey: c.cfg.ConsumerKey,
		Consumer:    c.cfg.Consumer,
		Redelivered: redelivered,
	}
	// Keep the envelope's own attempt honest: the consumer-side counter wins.
	msg.Envelope.Attempt = attempt
	if err := dispatcher.dispatch(ctx, dispatchTask{ctx: ctx, msg: msg, run: func(runCtx context.Context, m *Message) {
		c.deliver(runCtx, m, handler)
	}}); err != nil {
		if errors.Is(err, ErrClosedBus) || ctx.Err() != nil {
			return false
		}
		c.cfg.Hooks.consume(msg, OutcomeError, err)
		return false
	}
	return true
}

// deliver runs the handler for one delivery and applies the ack rule: ack only
// after the handler reported a committed transaction (EB-DEL-02), never after a
// failure (EB-DEL-03).
func (c *RedisConsumer) deliver(ctx context.Context, msg *Message, handler Handler) {
	if !c.cfg.DisableDedupe && c.alreadyProcessed(msg) {
		// EB-IDM-03: a duplicate of a completed event is acked as-is and the
		// handler is not re-entered.
		c.cfg.Hooks.consume(msg, OutcomeDuplicate, nil)
		if err := c.Ack(ctx, msg); err != nil {
			return
		}
		c.cfg.Hooks.ack(msg, OutcomeOK)
		return
	}
	err := callHandler(ctx, msg, handler)
	if err == nil {
		c.markProcessed(msg)
		if ackErr := c.Ack(ctx, msg); ackErr != nil {
			// The business effect happened but the PEL still holds the entry;
			// the next claim re-delivers it and the dedupe guard absorbs it.
			c.cfg.Hooks.ack(msg, OutcomeError)
			return
		}
		c.cfg.Hooks.ack(msg, OutcomeOK)
		return
	}
	if ctx.Err() != nil {
		// Cancelled mid-handler: leave it unacked, exactly like a crash.
		return
	}
	if nackErr := c.Nack(ctx, msg, err); nackErr != nil && !errors.Is(nackErr, ErrClosedBus) {
		c.cfg.Hooks.consume(msg, OutcomeError, nackErr)
		return
	}
	c.cfg.Hooks.consume(msg, OutcomeFailure, err)
}

// Ack performs XACK for one delivery.
func (c *RedisConsumer) Ack(ctx context.Context, msg *Message) error {
	if msg == nil || msg.TransportID == "" {
		return errors.New("eventbus: Ack requires a delivered message")
	}
	if _, err := c.client.XAck(ctx, c.cfg.StreamKey, c.cfg.Group, msg.TransportID).Result(); err != nil {
		return &TransportError{Op: "xack", Err: err}
	}
	return nil
}

// Nack reports a failed delivery. While budget remains it only records a retry
// and deliberately performs no XACK; once the budget is spent the entry is
// dead-lettered (metadata only) and then acked.
func (c *RedisConsumer) Nack(ctx context.Context, msg *Message, cause error) error {
	if msg == nil {
		return errors.New("eventbus: Nack requires a message")
	}
	class := ErrorClassForDelivery(cause)
	if msg.Attempt >= c.cfg.MaxAttempts {
		entry := NewDLQEntry(msg.Envelope, c.cfg.ConsumerKey, msg.Attempt, class, c.cfg.StreamKey, msg.TransportID, c.cfg.Now())
		if err := c.writeDlq(ctx, entry); err != nil {
			return err
		}
		c.cfg.Hooks.dlq(entry)
		c.cfg.Hooks.retry(msg, class)
		return c.Ack(ctx, msg)
	}
	c.cfg.Hooks.retry(msg, class)
	return nil
}

// dropToDlq dead-letters a pending entry identified through the pending list.
func (c *RedisConsumer) dropToDlq(ctx context.Context, field, id string, values map[string]any, class ErrorClass, attempts int) error {
	entry := c.buildDlqEntry(values, id, class, attempts)
	if err := c.writeDlq(ctx, entry); err != nil {
		return err
	}
	c.cfg.Hooks.dlq(entry)
	if _, err := c.client.XAck(ctx, c.cfg.StreamKey, c.cfg.Group, id).Result(); err != nil {
		return &TransportError{Op: "xack", Err: err}
	}
	return nil
}

func (c *RedisConsumer) dlqUndecodable(ctx context.Context, entry redis.XMessage, class ErrorClass) {
	dlq := c.buildDlqEntry(entry.Values, entry.ID, class, 1)
	if err := c.writeDlq(ctx, dlq); err != nil {
		c.cfg.Hooks.consume(nil, OutcomeError, err)
		return
	}
	c.cfg.Hooks.dlq(dlq)
	if _, err := c.client.XAck(ctx, c.cfg.StreamKey, c.cfg.Group, entry.ID).Result(); err != nil {
		c.cfg.Hooks.consume(nil, OutcomeError, &TransportError{Op: "xack", Err: err})
	}
}

func (c *RedisConsumer) dlqPartial(ctx context.Context, entry redis.XMessage, frame []byte, class ErrorClass) {
	_ = frame
	dlq := c.buildDlqEntry(entry.Values, entry.ID, class, 1)
	if err := c.writeDlq(ctx, dlq); err != nil {
		c.cfg.Hooks.consume(nil, OutcomeError, err)
		return
	}
	c.cfg.Hooks.dlq(dlq)
	if _, err := c.client.XAck(ctx, c.cfg.StreamKey, c.cfg.Group, entry.ID).Result(); err != nil {
		c.cfg.Hooks.consume(nil, OutcomeError, &TransportError{Op: "xack", Err: err})
	}
}

// buildDlqEntry recovers whatever identity metadata the frame still carries.
// A value is kept only when it satisfies its own schema pattern, so the DLQ
// stream never stores a fabricated or invalid identifier, and no payload byte
// is ever copied (EB-SEC-04).
func (c *RedisConsumer) buildDlqEntry(values map[string]any, transportID string, class ErrorClass, attempts int) DLQEntry {
	var env Envelope
	if raw, ok := values[c.cfg.StreamField]; ok {
		if frame, err := frameBytes(raw); err == nil {
			env = partialMetadata(frame)
		}
	}
	if !IsAllowedType(env.Type) {
		// The type of a malformed frame is not queue metadata: drop it rather
		// than write a value the DLQ schema forbids.
		env.Type = ""
	}
	return NewDLQEntry(env, c.cfg.ConsumerKey, attempts, class, c.cfg.StreamKey, transportID, c.cfg.Now())
}

func (c *RedisConsumer) writeDlq(ctx context.Context, entry DLQEntry) error {
	frame, err := entry.Encode()
	if err != nil {
		return err
	}
	dlqPipe := c.client.TxPipeline()
	dlqPipe.XAdd(ctx, &redis.XAddArgs{
		Stream: c.cfg.DLQStreamKey,
		MaxLen: c.cfg.DLQMaxLen,
		Approx: true,
		Values: map[string]any{c.cfg.StreamField: string(frame)},
	})
	dlqPipe.Expire(ctx, c.cfg.DLQStreamKey, c.cfg.DLQStreamTTL)
	if _, err := dlqPipe.Exec(ctx); err != nil {
		return &TransportError{Op: "xadd_dlq", Err: err}
	}
	return nil
}

// PendingEntry is one row of the consumer group pending list.
type PendingEntry struct {
	ID         string
	Consumer   string
	Idle       time.Duration
	Deliveries int64
}

// Pending reports the group's pending list: depth plus the entry ids, their
// owner and delivery counts. This is the single source of the retry budget, so
// there is no second counter anywhere (EB: attempt is counted by the consumer).
func (c *RedisConsumer) Pending(ctx context.Context, count int64) (int64, []PendingEntry, error) {
	if err := c.EnsureGroup(ctx); err != nil {
		return 0, nil, err
	}
	if count <= 0 {
		count = c.cfg.Count
	}
	summary, err := c.client.XPending(ctx, c.cfg.StreamKey, c.cfg.Group).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, nil, &TransportError{Op: "xpending", Err: err}
	}
	rows, err := c.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.cfg.StreamKey,
		Group:  c.cfg.Group,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, nil, &TransportError{Op: "xpending", Err: err}
	}
	entries := make([]PendingEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, PendingEntry{ID: row.ID, Consumer: row.Consumer, Idle: row.Idle, Deliveries: row.RetryCount})
	}
	c.cfg.Hooks.pending(c.cfg.Group, summary.Count)
	return summary.Count, entries, nil
}

// StreamLength reports XLEN of the run stream.
func (c *RedisConsumer) StreamLength(ctx context.Context) (int64, error) {
	length, err := c.client.XLen(ctx, c.cfg.StreamKey).Result()
	if err != nil {
		return 0, &TransportError{Op: "xlen", Err: err}
	}
	return length, nil
}

// IsConsuming reports whether a Consume loop owns this consumer.
func (c *RedisConsumer) IsConsuming() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// Close stops the loops and releases a client this consumer created.
func (c *RedisConsumer) Close(_ context.Context) error {
	var dispatcher *runDispatcher
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		dispatcher = c.dispatch
		c.mu.Unlock()
		close(c.quit)
	})
	if dispatcher != nil {
		dispatcher.stop()
	}
	c.wg.Wait()
	if c.owned {
		return c.client.Close()
	}
	return nil
}

// sleepOrDone waits up to d, returning false when ctx was cancelled first.
func sleepOrDone(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		d = time.Millisecond
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// frameBytes accepts the two representations XADD may hand back: []byte when
// the client preserves the raw payload, or string when it decodes the field.
// Any other type is a protocol violation, never a payload to coerce.
func frameBytes(v any) ([]byte, error) {
	switch raw := v.(type) {
	case nil:
		return nil, errors.New("eventbus: stream entry has no data field")
	case []byte:
		return raw, nil
	case string:
		return []byte(raw), nil
	default:
		return nil, fmt.Errorf("eventbus: stream field has unsupported type %T", v)
	}
}

// alreadyProcessed and markProcessed implement the process-local duplicate
// guard. It is a cache, not the dedup authority: the durable
// (consumerKey,eventId) table is R2-05.
func (c *RedisConsumer) alreadyProcessed(msg *Message) bool {
	if msg == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.seen[msg.Envelope.EventID]
	return ok
}

func (c *RedisConsumer) markProcessed(msg *Message) {
	if msg == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seen == nil {
		c.seen = make(map[string]struct{})
	}
	c.seen[msg.Envelope.EventID] = struct{}{}
}

// partialMetadata recovers only the identity fields a malformed frame still
// carries. Each value must satisfy its own schema pattern, so the DLQ never
// stores fabricated identifiers; no payload byte is copied (EB-SEC-04).
func partialMetadata(frame []byte) Envelope {
	var raw struct {
		EventID    string `json:"eventId"`
		RunID      string `json:"runId"`
		Seq        int64  `json:"seq"`
		Type       string `json:"type"`
		OccurredAt string `json:"occurredAt"`
	}
	if err := json.Unmarshal(frame, &raw); err != nil {
		return Envelope{}
	}
	var env Envelope
	if eventIDPattern.MatchString(raw.EventID) && len(raw.EventID) <= 160 {
		env.EventID = raw.EventID
	}
	if runIDPattern.MatchString(raw.RunID) && len(raw.RunID) <= 64 {
		env.RunID = raw.RunID
	}
	if raw.Seq > 0 {
		env.Seq = raw.Seq
	}
	if IsAllowedType(raw.Type) {
		env.Type = raw.Type
	}
	if _, err := time.Parse(time.RFC3339, raw.OccurredAt); err == nil {
		env.OccurredAt = raw.OccurredAt
	}
	return env
}
