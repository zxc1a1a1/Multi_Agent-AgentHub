// Package repository provides sqlc-backed implementations of the domain
// repository interfaces. It maps between clean domain types and sqlc
// generated types, and converts SQLite errors to domain sentinel errors.
package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/db"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/internal/domain"
)

// Compile-time interface checks.
var (
	_ domain.ConversationRepository = (*ConversationRepo)(nil)
	_ domain.MessageRepository      = (*MessageRepo)(nil)
	_ domain.RunRepository          = (*RunRepo)(nil)
	_ domain.EventRepository        = (*EventRepo)(nil)
	_ domain.RunStepRepository      = (*RunStepRepo)(nil)
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

const timeLayout = time.RFC3339

func fmtTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeLayout, s)
	return t
}

func parseNullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(timeLayout, ns.String)
	if err != nil {
		return nil
	}
	return &t
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed") {
		return domain.ErrConflict
	}
	if strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy") {
		return domain.ErrDatabaseBusy
	}
	return domain.ErrInternal
}

func rowAffected(n int64, execErr error) error {
	if execErr != nil {
		return mapDBError(execErr)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

// ---------------------------------------------------------------------------
// sqlc -> domain conversion helpers
// ---------------------------------------------------------------------------

// convRow is a private intermediate struct that all sqlc conversation row
// types map to before being converted to domain.Conversation.
type convRow struct {
	id, userID, title, mode, responseMode, status string
	version                                       int64
	pinned                                        int64
	lastMessageAt, pinnedAt, archivedAt, deletedAt sql.NullString
	createdAt, updatedAt                           string
}

func (r convRow) toDomain() domain.Conversation {
	c := domain.Conversation{
		ID:            r.id,
		UserID:        r.userID,
		Title:         r.title,
		Mode:          r.mode,
		ResponseMode:  r.responseMode,
		Status:        r.status,
		Version:       r.version,
		LastMessageAt: parseNullTime(r.lastMessageAt),
		ArchivedAt:    parseNullTime(r.archivedAt),
		DeletedAt:     parseNullTime(r.deletedAt),
		CreatedAt:     parseTime(r.createdAt),
		UpdatedAt:     parseTime(r.updatedAt),
	}
	// Derive pinned state: prefer explicit pinned int, fall back to pinned_at.
	if r.pinned != 0 || (r.pinnedAt.Valid && r.pinnedAt.String != "") {
		c.Pinned = true
		c.PinnedAt = parseNullTime(r.pinnedAt)
	}
	return c
}

func convFromCreateConversationRow(r db.CreateConversationRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, 0, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}
func convFromGetConversationForUserRow(r db.GetConversationForUserRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, 0, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}
func convFromListConversationsForUserRow(r db.ListConversationsForUserRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, 0, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}
func convFromSetConversationArchivedRow(r db.SetConversationArchivedRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, 0, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}
func convFromSetConversationPinnedRow(r db.SetConversationPinnedRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, r.Pinned, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}
func convFromUpdateConversationVersionRow(r db.UpdateConversationVersionRow) convRow {
	return convRow{r.ID, r.UserID, r.Title, r.Mode, r.ResponseMode, r.Status,
		r.Version, 0, r.LastMessageAt, r.PinnedAt, r.ArchivedAt, r.DeletedAt,
		r.CreatedAt, r.UpdatedAt}
}

// msgRow is a private intermediate for sqlc message row types.
type msgRow struct {
	id, conversationID, messageID, role, senderType, senderName, agentName string
	runID, stepID, clientMessageID, contentJSON, replyToMessageID, deletedAt sql.NullString
	content, status                                                         string
	errorCode, errorMessage                                                 string
	sequence                                                                int64
	createdAt, updatedAt                                                    string
}

func (r msgRow) toDomain() domain.Message {
	return domain.Message{
		ID:               r.id,
		ConversationID:   r.conversationID,
		RunID:            r.runID.String,
		StepID:           r.stepID.String,
		MessageID:        r.messageID,
		Role:             r.role,
		SenderType:       r.senderType,
		SenderName:       r.senderName,
		AgentName:        r.agentName,
		ClientMessageID:  r.clientMessageID.String,
		Content:          r.content,
		ContentJSON:      r.contentJSON.String,
		ReplyToMessageID: r.replyToMessageID.String,
		Status:           r.status,
		Sequence:         r.sequence,
		ErrorCode:        r.errorCode,
		ErrorMessage:     r.errorMessage,
		CreatedAt:        parseTime(r.createdAt),
		UpdatedAt:        parseTime(r.updatedAt),
		DeletedAt:        parseNullTime(r.deletedAt),
	}
}

func msgFromCreateMessageRow(r db.CreateMessageRow) msgRow {
	return msgRow{r.ID, r.ConversationID, r.MessageID, r.Role, r.SenderType, r.SenderName, r.AgentName,
		r.RunID, r.StepID, r.ClientMessageID, r.ContentJson, r.ReplyToMessageID, r.DeletedAt,
		r.Content, r.Status, r.ErrorCode, r.ErrorMessage,
		r.Sequence, r.CreatedAt, r.UpdatedAt}
}
func msgFromGetMessageByClientIDRow(r db.GetMessageByClientIDRow) msgRow {
	return msgRow{r.ID, r.ConversationID, r.MessageID, r.Role, r.SenderType, r.SenderName, r.AgentName,
		r.RunID, r.StepID, r.ClientMessageID, r.ContentJson, r.ReplyToMessageID, r.DeletedAt,
		r.Content, r.Status, r.ErrorCode, r.ErrorMessage,
		r.Sequence, r.CreatedAt, r.UpdatedAt}
}
func msgFromListMessagesRow(r db.ListMessagesRow) msgRow {
	return msgRow{r.ID, r.ConversationID, r.MessageID, r.Role, r.SenderType, r.SenderName, r.AgentName,
		r.RunID, r.StepID, r.ClientMessageID, r.ContentJson, r.ReplyToMessageID, r.DeletedAt,
		r.Content, r.Status, r.ErrorCode, r.ErrorMessage,
		r.Sequence, r.CreatedAt, r.UpdatedAt}
}

// runRow is a private intermediate for sqlc run row types.
type runRow struct {
	id, conversationID, mode, status, errorCode, startedAt       string
	triggerMessageID, planID, confirmedPlanVersion, contextSnapshotID sql.NullString
	finishedAt, createdAt, updatedAt                                  sql.NullString
}

func (r runRow) toDomain() domain.Run {
	return domain.Run{
		ID:                   r.id,
		ConversationID:       r.conversationID,
		TriggerMessageID:     r.triggerMessageID.String,
		Mode:                 r.mode,
		Status:               r.status,
		PlanID:               r.planID.String,
		ConfirmedPlanVersion: r.confirmedPlanVersion.String,
		ContextSnapshotID:    r.contextSnapshotID.String,
		ErrorCode:            r.errorCode,
		StartedAt:            parseTime(r.startedAt),
		FinishedAt:           parseNullTime(r.finishedAt),
		CreatedAt:            parseNullTime(r.createdAt),
		UpdatedAt:            parseNullTime(r.updatedAt),
	}
}

func runFromCreateRunRow(r db.CreateRunRow) runRow {
	return runRow{r.ID, r.ConversationID, r.Mode, r.Status, r.ErrorCode,
		r.StartedAt, r.TriggerMessageID, r.PlanID, r.ConfirmedPlanVersion,
		r.ContextSnapshotID, r.FinishedAt, r.CreatedAt, r.UpdatedAt}
}
func runFromGetRunRow(r db.GetRunRow) runRow {
	return runRow{r.ID, r.ConversationID, r.Mode, r.Status, r.ErrorCode,
		r.StartedAt, r.TriggerMessageID, r.PlanID, r.ConfirmedPlanVersion,
		r.ContextSnapshotID, r.FinishedAt, r.CreatedAt, r.UpdatedAt}
}
func runFromListRunsByConversationRow(r db.ListRunsByConversationRow) runRow {
	return runRow{r.ID, r.ConversationID, r.Mode, r.Status, r.ErrorCode,
		r.StartedAt, r.TriggerMessageID, r.PlanID, r.ConfirmedPlanVersion,
		r.ContextSnapshotID, r.FinishedAt, r.CreatedAt, r.UpdatedAt}
}

type stepRow struct {
	id, runID, conversationID, taskID, agentName, capabilityID, status string
	stepIndex                                                          int64
	startedAt, finishedAt                                              sql.NullString
	errorCode, errorMessage                                            string
}

func (r stepRow) toDomain() domain.RunStep {
	return domain.RunStep{
		ID:             r.id,
		RunID:          r.runID,
		ConversationID: r.conversationID,
		TaskID:         r.taskID,
		StepIndex:      int(r.stepIndex),
		AgentName:      r.agentName,
		CapabilityID:   r.capabilityID,
		Status:         r.status,
		StartedAt:      parseNullTime(r.startedAt),
		FinishedAt:     parseNullTime(r.finishedAt),
		ErrorCode:      r.errorCode,
		ErrorMessage:   r.errorMessage,
	}
}

func stepFromListRunStepsByRunRow(r db.ListRunStepsByRunRow) stepRow {
	return stepRow{r.ID, r.RunID, r.ConversationID, r.TaskID, r.AgentName, r.CapabilityID, r.Status,
		r.StepIndex, r.StartedAt, r.FinishedAt, r.ErrorCode, r.ErrorMessage}
}

// ---------------------------------------------------------------------------
// ConversationRepo
// ---------------------------------------------------------------------------

// ConversationRepo implements domain.ConversationRepository using sqlc.
type ConversationRepo struct {
	q *db.Queries
}

// NewConversationRepo creates a ConversationRepo backed by sqlc Queries.
func NewConversationRepo(q *db.Queries) *ConversationRepo {
	return &ConversationRepo{q: q}
}

func (r *ConversationRepo) Create(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.ID == "" || input.UserID == "" || !domain.ValidMode(input.Mode) || !domain.ValidResponseMode(input.ResponseMode) {
		return nil, domain.ErrInvalidArgument
	}
	now := time.Now().UTC()
	row, err := r.q.CreateConversation(ctx, db.CreateConversationParams{
		ID:           input.ID,
		UserID:       input.UserID,
		Title:        input.Title,
		Mode:         input.Mode,
		ResponseMode: input.ResponseMode,
		Status:       "active",
		CreatedAt:    fmtTime(now),
		UpdatedAt:    fmtTime(now),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	c := convFromCreateConversationRow(row).toDomain()
	return &c, nil
}

func (r *ConversationRepo) Get(ctx context.Context, userID, id string) (*domain.Conversation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.q.GetConversationForUser(ctx, db.GetConversationForUserParams{ID: id, UserID: userID})
	if err != nil {
		return nil, mapDBError(err)
	}
	c := convFromGetConversationForUserRow(row).toDomain()
	return &c, nil
}

func (r *ConversationRepo) List(ctx context.Context, userID string, p domain.Pagination) ([]domain.Conversation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, domain.ErrInvalidArgument
	}
	rows, err := r.q.ListConversationsForUser(ctx, db.ListConversationsForUserParams{
		UserID: userID,
		Limit:  p.Limit,
		Offset: p.Offset,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]domain.Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, convFromListConversationsForUserRow(row).toDomain())
	}
	return out, nil
}

func (r *ConversationRepo) Update(ctx context.Context, userID, id string, version int64, title string) (*domain.Conversation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if version < 1 {
		return nil, domain.ErrInvalidArgument
	}
	// Fetch current values so we don't overwrite mode/response_mode.
	current, err := r.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	// If the caller's version is stale, return conflict immediately rather
	// than letting the SQL layer map 0-rows to not-found.
	if current.Version != version {
		return nil, domain.ErrConflict
	}
	now := time.Now().UTC()
	row, err := r.q.UpdateConversationVersion(ctx, db.UpdateConversationVersionParams{
		Title:        title,
		Mode:         current.Mode,
		ResponseMode: current.ResponseMode,
		UpdatedAt:    fmtTime(now),
		ID:           id,
		UserID:       userID,
		Version:      version,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	c := convFromUpdateConversationVersionRow(row).toDomain()
	return &c, nil
}

func (r *ConversationRepo) SetArchived(ctx context.Context, userID, id string, archive bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	var archivedAt sql.NullString
	if archive {
		archivedAt = sql.NullString{String: fmtTime(now), Valid: true}
	}
	_, err := r.q.SetConversationArchived(ctx, db.SetConversationArchivedParams{
		ArchivedAt: archivedAt,
		UpdatedAt:  fmtTime(now),
		ID:         id,
		UserID:     userID,
	})
	return mapDBError(err)
}

func (r *ConversationRepo) SetPinned(ctx context.Context, userID, id string, pin bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	pinnedInt := int64(0)
	var pinnedAt sql.NullString
	if pin {
		pinnedInt = 1
		pinnedAt = sql.NullString{String: fmtTime(now), Valid: true}
	}
	_, err := r.q.SetConversationPinned(ctx, db.SetConversationPinnedParams{
		Pinned:    pinnedInt,
		PinnedAt:  pinnedAt,
		UpdatedAt: fmtTime(now),
		ID:        id,
		UserID:    userID,
	})
	return mapDBError(err)
}

func (r *ConversationRepo) SoftDelete(ctx context.Context, userID, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	n, err := r.q.SoftDeleteConversation(ctx, db.SoftDeleteConversationParams{
		DeletedAt: sql.NullString{String: fmtTime(now), Valid: true},
		UpdatedAt: fmtTime(now),
		ID:        id,
		UserID:    userID,
	})
	return rowAffected(n, err)
}

// ---------------------------------------------------------------------------
// MessageRepo
// ---------------------------------------------------------------------------

// MessageRepo implements domain.MessageRepository using sqlc.
type MessageRepo struct {
	q  *db.Queries
	db *sql.DB // for transactional idempotency
}

// NewMessageRepo creates a MessageRepo backed by sqlc Queries.
func NewMessageRepo(q *db.Queries, underlying *sql.DB) *MessageRepo {
	return &MessageRepo{q: q, db: underlying}
}

func (r *MessageRepo) Create(ctx context.Context, input domain.CreateMessageInput) (*domain.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.ConversationID == "" || input.SenderType == "" {
		return nil, domain.ErrInvalidArgument
	}
	if input.ContentJSON != "" {
		if len(input.ContentJSON) > domain.MaxEventPayloadBytes || !json.Valid([]byte(input.ContentJSON)) {
			return nil, domain.ErrInvalidArgument
		}
	}

	// Begin transaction for idempotency and sequence allocation.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer tx.Rollback()
	txq := r.q.WithTx(tx)

	// Idempotency check.
	if input.ClientMessageID != "" {
		existing, err := txq.GetMessageByClientID(ctx, db.GetMessageByClientIDParams{
			ConversationID:  input.ConversationID,
			ClientMessageID: nullStr(input.ClientMessageID),
		})
		if err == nil {
			_ = tx.Rollback()
			m := msgFromGetMessageByClientIDRow(existing).toDomain()
			return &m, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, mapDBError(err)
		}
	}

	// Allocate next sequence.
	nextSeq, err := txq.NextMessageSequence(ctx, input.ConversationID)
	if err != nil {
		return nil, mapDBError(err)
	}

	now := time.Now().UTC()
	id := input.ID
	if id == "" {
		id = newID()
	}
	status := input.Status
	if status == "" {
		status = "completed"
	}

	row, err := txq.CreateMessage(ctx, db.CreateMessageParams{
		ID:               id,
		ConversationID:   input.ConversationID,
		RunID:            nullStr(input.RunID),
		StepID:           nullStr(input.StepID),
		MessageID:        input.MessageID,
		Role:             input.Role,
		SenderType:       input.SenderType,
		SenderName:       input.SenderName,
		AgentName:        input.AgentName,
		Content:          input.Content,
		ContentJson:      nullStr(input.ContentJSON),
		ReplyToMessageID: nullStr(input.ReplyToMessageID),
		Status:           status,
		Sequence:         nextSeq,
		ErrorCode:        input.ErrorCode,
		ErrorMessage:     input.ErrorMessage,
		CreatedAt:        fmtTime(now),
		UpdatedAt:        fmtTime(now),
		ClientMessageID:  nullStr(input.ClientMessageID),
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	// Update conversation last_message_at (best-effort, via tx).
	_, err = txq.TouchConversation(ctx, db.TouchConversationParams{
		LastMessageAt: sql.NullString{String: fmtTime(now), Valid: true},
		UpdatedAt:     fmtTime(now),
		ID:            input.ConversationID,
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, mapDBError(err)
	}

	m := msgFromCreateMessageRow(row).toDomain()
	return &m, nil
}

func (r *MessageRepo) GetByClientID(ctx context.Context, conversationID, clientMessageID string) (*domain.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.q.GetMessageByClientID(ctx, db.GetMessageByClientIDParams{
		ConversationID:  conversationID,
		ClientMessageID: nullStr(clientMessageID),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	m := msgFromGetMessageByClientIDRow(row).toDomain()
	return &m, nil
}

func (r *MessageRepo) List(ctx context.Context, conversationID string, p domain.Pagination) ([]domain.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if conversationID == "" {
		return nil, domain.ErrInvalidArgument
	}
	rows, err := r.q.ListMessages(ctx, db.ListMessagesParams{
		ConversationID: conversationID,
		Limit:          p.Limit,
		Offset:         p.Offset,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, msgFromListMessagesRow(row).toDomain())
	}
	return out, nil
}

func (r *MessageRepo) UpdateContentAndStatus(ctx context.Context, id, content, status, errorCode, errorMessage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n, err := r.q.UpdateMessageContentStatus(ctx, db.UpdateMessageContentStatusParams{
		Content:      content,
		Status:       status,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		UpdatedAt:    fmtTime(time.Now().UTC()),
		ID:           id,
	})
	return rowAffected(n, err)
}

func (r *MessageRepo) SoftDelete(ctx context.Context, id, conversationID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	n, err := r.q.SoftDeleteMessage(ctx, db.SoftDeleteMessageParams{
		DeletedAt:      sql.NullString{String: fmtTime(now), Valid: true},
		UpdatedAt:      fmtTime(now),
		ID:             id,
		ConversationID: conversationID,
	})
	return rowAffected(n, err)
}

// ---------------------------------------------------------------------------
// RunRepo
// ---------------------------------------------------------------------------

// RunRepo implements domain.RunRepository using sqlc.
type RunRepo struct {
	q *db.Queries
}

// NewRunRepo creates a RunRepo backed by sqlc Queries.
func NewRunRepo(q *db.Queries) *RunRepo {
	return &RunRepo{q: q}
}

func (r *RunRepo) Create(ctx context.Context, input domain.CreateRunInput) (*domain.Run, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.ID == "" || input.ConversationID == "" || !domain.ValidMode(input.Mode) {
		return nil, domain.ErrInvalidArgument
	}
	status := input.Status
	if status == "" {
		status = "pending"
	}
	if !domain.ValidRunStatus(status) {
		return nil, domain.ErrInvalidArgument
	}
	now := time.Now().UTC()
	row, err := r.q.CreateRun(ctx, db.CreateRunParams{
		ID:                   input.ID,
		ConversationID:       input.ConversationID,
		TriggerMessageID:     nullStr(input.TriggerMessageID),
		Mode:                 input.Mode,
		Status:               status,
		PlanID:               nullStr(input.PlanID),
		ConfirmedPlanVersion: nullStr(input.ConfirmedPlanVersion),
		ContextSnapshotID:    nullStr(input.ContextSnapshotID),
		StartedAt:            fmtTime(now),
		UpdatedAt:            sql.NullString{String: fmtTime(now), Valid: true},
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	run := runFromCreateRunRow(row).toDomain()
	return &run, nil
}

func (r *RunRepo) Get(ctx context.Context, id string) (*domain.Run, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.q.GetRun(ctx, id)
	if err != nil {
		return nil, mapDBError(err)
	}
	run := runFromGetRunRow(row).toDomain()
	return &run, nil
}

func (r *RunRepo) List(ctx context.Context, conversationID string, p domain.Pagination) ([]domain.Run, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	rows, err := r.q.ListRunsByConversation(ctx, db.ListRunsByConversationParams{
		ConversationID: conversationID,
		Limit:          p.Limit,
		Offset:         p.Offset,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, runFromListRunsByConversationRow(row).toDomain())
	}
	return out, nil
}

func (r *RunRepo) CompareAndSetStatus(ctx context.Context, id, from, to, errorCode, errorMessage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !domain.ValidRunTransition(from, to) {
		return domain.ErrInvalidArgument
	}
	now := time.Now().UTC()
	var finishedAt sql.NullString
	if domain.TerminalRunStatus(to) {
		finishedAt = sql.NullString{String: fmtTime(now), Valid: true}
	}
	n, err := r.q.CompareAndSetRunStatus(ctx, db.CompareAndSetRunStatusParams{
		Status:       to,
		FinishedAt:   finishedAt,
		UpdatedAt:    sql.NullString{String: fmtTime(now), Valid: true},
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		ID:           id,
		Status_2:     from,
	})
	if err != nil {
		return mapDBError(err)
	}
	if n == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (r *RunRepo) Cancel(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	run, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	return r.CompareAndSetStatus(ctx, id, run.Status, "canceled", "", "")
}

// ---------------------------------------------------------------------------
// EventRepo
// ---------------------------------------------------------------------------

// EventRepo implements domain.EventRepository using sqlc.
type EventRepo struct {
	q  *db.Queries
	db *sql.DB // for transactional sequence allocation
}

// NewEventRepo creates an EventRepo backed by sqlc Queries.
func NewEventRepo(q *db.Queries, underlying *sql.DB) *EventRepo {
	return &EventRepo{q: q, db: underlying}
}

func (r *EventRepo) Append(ctx context.Context, input domain.CreateEventInput) (*domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.EventType == "" || input.RunID == "" || input.ConversationID == "" {
		return nil, domain.ErrInvalidArgument
	}
	payload := input.Payload
	if payload == "" {
		payload = "{}"
	}
	if len(payload) > domain.MaxEventPayloadBytes || !json.Valid([]byte(payload)) {
		return nil, domain.ErrInvalidArgument
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer tx.Rollback()
	txq := r.q.WithTx(tx)

	nextSeq, err := txq.NextEventSequence(ctx, input.RunID)
	if err != nil {
		return nil, mapDBError(err)
	}

	id := input.ID
	if id == "" {
		id = newID()
	}
	now := time.Now().UTC()

	row, err := txq.CreateEvent(ctx, db.CreateEventParams{
		ID:             id,
		ConversationID: input.ConversationID,
		RunID:          input.RunID,
		InvocationID:   input.InvocationID,
		EventType:      input.EventType,
		Payload:        payload,
		Sequence:       nextSeq,
		CreatedAt:      fmtTime(now),
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, mapDBError(err)
	}

	return &domain.Event{
		ID:             row.ID,
		ConversationID: row.ConversationID,
		RunID:          row.RunID,
		InvocationID:   row.InvocationID,
		EventType:      row.EventType,
		Payload:        row.Payload,
		Sequence:       row.Sequence,
		CreatedAt:      parseTime(row.CreatedAt),
	}, nil
}

func (r *EventRepo) ListAfter(ctx context.Context, runID string, after, limit int64) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.q.ListEventsAfter(ctx, db.ListEventsAfterParams{
		RunID:    runID,
		Sequence: after,
		Limit:    limit,
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Event{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			RunID:          row.RunID,
			InvocationID:   row.InvocationID,
			EventType:      row.EventType,
			Payload:        row.Payload,
			Sequence:       row.Sequence,
			CreatedAt:      parseTime(row.CreatedAt),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// RunStepRepo
// ---------------------------------------------------------------------------

// RunStepRepo implements domain.RunStepRepository using sqlc.
type RunStepRepo struct {
	q *db.Queries
}

// NewRunStepRepo creates a RunStepRepo backed by sqlc Queries.
func NewRunStepRepo(q *db.Queries) *RunStepRepo {
	return &RunStepRepo{q: q}
}

func (r *RunStepRepo) Create(ctx context.Context, input domain.CreateRunStepInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if input.ID == "" || input.RunID == "" || input.ConversationID == "" {
		return domain.ErrInvalidArgument
	}
	now := time.Now().UTC()
	err := r.q.CreateRunStep(ctx, db.CreateRunStepParams{
		ID:             input.ID,
		RunID:          input.RunID,
		ConversationID: input.ConversationID,
		TaskID:         input.TaskID,
		StepIndex:      int64(input.StepIndex),
		AgentName:      input.AgentName,
		CapabilityID:   input.CapabilityID,
		Status:         input.Status,
		StartedAt:      sql.NullString{String: fmtTime(now), Valid: true},
	})
	return mapDBError(err)
}

func (r *RunStepRepo) ListByRun(ctx context.Context, runID string) ([]domain.RunStep, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.q.ListRunStepsByRun(ctx, runID)
	if err != nil {
		return nil, mapDBError(err)
	}
	result := make([]domain.RunStep, 0, len(rows))
	for _, row := range rows {
		result = append(result, stepFromListRunStepsByRunRow(row).toDomain())
	}
	return result, nil
}

func (r *RunStepRepo) UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	n, err := r.q.UpdateRunStepStatus(ctx, db.UpdateRunStepStatusParams{
		Status:       status,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		FinishedAt:   sql.NullString{String: fmtTime(now), Valid: true},
		ID:           id,
	})
	return rowAffected(n, err)
}
