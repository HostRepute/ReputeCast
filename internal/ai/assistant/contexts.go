package assistant

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIThreadContextInput attaches a resource to a thread. The current screen
// is only suggested: nothing is shared until the person attaches it and
// confirms sharing (SharingReviewed).
type AIThreadContextInput struct {
	Kind            string    `json:"kind" enum:"campaign,report"`
	ID              uuid.UUID `json:"id" format:"uuid"`
	SharingReviewed bool      `json:"sharingReviewed" doc:"The person reviewed what is shared with the model and confirmed it"`
}

// AttachContext attaches a resource the person may read to one of their
// threads; a report's numbers are its fixed snapshot, taken now. Attaching
// it again changes nothing.
func (s *Service) AttachContext(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID, in AIThreadContextInput) (AIThread, error) {
	if !in.SharingReviewed {
		return AIThread{}, problem(ctx, http.StatusUnprocessableEntity, codeSharingReview, mSharingReview, nil)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		t, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread)
		if err != nil {
			return err
		}
		have, err := q.AIAssistantContexts(ctx, thread)
		if err != nil {
			return err
		}
		for _, c := range have {
			if c.Kind == in.Kind && c.TargetID == in.ID {
				return nil
			}
		}
		if len(have) >= maxContexts {
			return problem(ctx, http.StatusConflict, codeContextsFull, mContextsFull, i18n.Args{"max": maxContexts})
		}
		a, err := s.AI.AssistantAttachment(ctx, q, ws, fresh, in.Kind, in.ID, true)
		if err != nil {
			return err
		}
		id := uuid.Must(uuid.NewV7())
		var snapshot []byte
		if len(a.Snapshot) > 0 {
			if snapshot, err = s.Box.Seal(a.Snapshot, contextAD(id)); err != nil {
				return err
			}
		}
		if _, err := q.AIAssistantContextInsert(ctx, store.AIAssistantContextInsertParams{ID: id, ThreadID: thread, Kind: in.Kind, TargetID: in.ID,
			Revision: a.Revision, SnapshotSealed: snapshot, SharedAt: s.now()}); err != nil {
			return err
		}
		if err := s.touch(ctx, q, t); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, "ai_assistant_context.add",
			map[string]any{"kind": in.Kind, "target_id": in.ID, "revision": a.Revision, "sharing_policy": editorial.SharingPolicy}))
	})
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, thread)
}

// DetachContext removes a resource from a thread: later turns no longer
// read it.
func (s *Service) DetachContext(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread, id uuid.UUID) (AIThread, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		t, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread)
		if err != nil {
			return err
		}
		gone, err := q.AIAssistantContextDelete(ctx, store.AIAssistantContextDeleteParams{ThreadID: thread, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Attached resource")
		}
		if err != nil {
			return err
		}
		if err := s.touch(ctx, q, t); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, "ai_assistant_context.remove", map[string]any{"kind": gone.Kind, "target_id": gone.TargetID}))
	})
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, thread)
}

// touch records activity on a locked thread: its content is kept 30 days
// from now.
func (s *Service) touch(ctx context.Context, q *store.Queries, t store.AiAssistantThread) error {
	now := s.now()
	return q.AIAssistantThreadSave(ctx, store.AIAssistantThreadSaveParams{TitleSealed: t.TitleSealed, ModelID: t.ModelID, Now: now, ExpiresAt: now.Add(Retention),
		WorkspaceID: t.WorkspaceID, ID: t.ID})
}
