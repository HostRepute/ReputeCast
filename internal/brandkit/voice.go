package brandkit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// BrandVoiceContent describes style, not an instruction to perform actions.
type BrandVoiceContent struct {
	Tone           string `json:"tone" maxLength:"1000"`
	Address        string `json:"address" maxLength:"500"`
	SentenceLength string `json:"sentenceLength" maxLength:"500"`
	PreferredWords string `json:"preferredWords" maxLength:"1000"`
	AvoidedWords   string `json:"avoidedWords" maxLength:"1000"`
	Examples       string `json:"examples" maxLength:"2000"`
}

// Validate bounds internal model output too, not only HTTP input.
func (v *BrandVoiceContent) Validate(ctx context.Context) error {
	for _, f := range []struct {
		name  string
		value *string
		limit int
	}{
		{"tone", &v.Tone, 1000}, {"address", &v.Address, 500}, {"sentenceLength", &v.SentenceLength, 500},
		{"preferredWords", &v.PreferredWords, 1000}, {"avoidedWords", &v.AvoidedWords, 1000}, {"examples", &v.Examples, 2000},
	} {
		*f.value = strings.TrimSpace(*f.value)
		if !utf8.ValidString(*f.value) || utf8.RuneCountInString(*f.value) > f.limit || strings.ContainsRune(*f.value, 0) {
			return api.Invalid("body.content."+f.name, say(ctx, msgs.M("voice.invalid")))
		}
	}
	if v.Tone == "" {
		return api.Invalid("body.content.tone", say(ctx, msgs.M("voice.toneRequired")))
	}
	return nil
}

type BrandVoice struct {
	Revision  int32             `json:"revision"`
	Content   BrandVoiceContent `json:"content"`
	Learned   bool              `json:"learned"`
	CreatedBy string            `json:"createdBy"`
	CreatedAt *time.Time        `json:"createdAt,omitempty"`
}

type BrandVoiceSaveInput struct {
	BaseRevision int32             `json:"baseRevision" minimum:"0"`
	Content      BrandVoiceContent `json:"content"`
	// A saved suggestion's ID, not a client assertion of AI provenance.
	SuggestionID string `json:"suggestionId,omitempty" format:"uuid"`
}

type BrandVoiceHistory struct {
	Items          []BrandVoice `json:"items"`
	BeforeRevision int32        `json:"beforeRevision,omitempty"`
}

func (s *Service) Voice(ctx context.Context, ws *store.Workspace) (BrandVoice, error) {
	return voice(ctx, s.Q, ws.ID)
}

func voice(ctx context.Context, q *store.Queries, ws uuid.UUID) (BrandVoice, error) {
	r, err := q.BrandVoiceLatest(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return BrandVoice{}, nil
	}
	if err != nil {
		return BrandVoice{}, err
	}
	v := BrandVoice{Revision: r.Revision, Learned: r.RequestID != nil, CreatedBy: r.CreatedByName, CreatedAt: &r.CreatedAt}
	err = json.Unmarshal(r.Content, &v.Content)
	return v, err
}

func (s *Service) VoiceHistory(ctx context.Context, ws *store.Workspace, before int32) (BrandVoiceHistory, error) {
	out := BrandVoiceHistory{Items: []BrandVoice{}}
	rows, err := s.Q.BrandVoiceHistory(ctx, store.BrandVoiceHistoryParams{WorkspaceID: ws.ID, BeforeRevision: before})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		v := BrandVoice{Revision: r.Revision, Learned: r.RequestID != nil, CreatedBy: r.CreatedByName, CreatedAt: &r.CreatedAt}
		if err := json.Unmarshal(r.Content, &v.Content); err != nil {
			return out, err
		}
		out.Items = append(out.Items, v)
	}
	if len(rows) == 20 {
		out.BeforeRevision = rows[len(rows)-1].Revision
	}
	return out, nil
}

// VoiceSources checks all selected campaigns in one query. A partial match
// fails closed; IDs never silently disappear from the person's selection.
func VoiceSources(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, ids []uuid.UUID) ([]store.BrandVoiceSourcesRow, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID || !p.Can(authz.AIUse) {
		return nil, api.Forbidden(authz.AIUse, nil)
	}
	if !p.Can(authz.CampaignsView) {
		return nil, api.Forbidden(authz.CampaignsView, nil)
	}
	if len(ids) < 1 || len(ids) > 10 {
		return nil, api.Invalid("body.campaignIds", say(ctx, msgs.M("voice.selection")))
	}
	domains := make([]string, 0, len(p.Scope.SenderDomains))
	for _, d := range p.Scope.SenderDomains {
		domains = append(domains, strings.ToLower(d))
	}
	rows, err := q.BrandVoiceSources(ctx, store.BrandVoiceSourcesParams{WorkspaceID: ws.ID, Ids: ids, ScopeLists: access.ScopeLists(p), ScopeDomains: domains})
	if err != nil {
		return nil, err
	}
	if len(rows) != len(ids) {
		return nil, api.Invalid("body.campaignIds", say(ctx, msgs.M("voice.selection")))
	}
	return rows, nil
}

// ActorRow reads and locks a person's active membership, role and account
// (BrandVoiceActor). A role change that commits while the read waits for
// the membership's lock makes it find no row under READ COMMITTED: the
// updated membership is checked again against the role row read first, and
// its new role_id doesn't match. So a read that finds nothing reads once
// more, with the new role; a person who really left still finds nothing.
func ActorRow(ctx context.Context, q *store.Queries, ws, user uuid.UUID) (store.BrandVoiceActorRow, error) {
	r, err := q.BrandVoiceActor(ctx, store.BrandVoiceActorParams{WorkspaceID: ws, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		r, err = q.BrandVoiceActor(ctx, store.BrandVoiceActorParams{WorkspaceID: ws, UserID: user})
	}
	return r, err
}

// VoiceActor refreshes human permissions. In a transaction its row locks
// fence membership and role changes until the voice decision commits.
func VoiceActor(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal) (*authz.Principal, error) {
	if !p.IsHuman() || p.WorkspaceID != ws.ID {
		return nil, api.Forbidden(authz.AIUse, nil)
	}
	r, err := ActorRow(ctx, q, ws.ID, p.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, api.Forbidden(authz.AIUse, nil)
	}
	if err != nil {
		return nil, err
	}
	fresh := *p
	fresh.Permissions = authz.Effective(r.Key, r.Builtin, r.Permissions)
	fresh.Scope = authz.Scope{}
	if err := json.Unmarshal(r.Scope, &fresh.Scope); err != nil {
		return nil, err
	}
	return &fresh, nil
}

func (s *Service) SaveVoice(ctx context.Context, ws *store.Workspace, p *authz.Principal, in BrandVoiceSaveInput) (BrandVoice, error) {
	if !p.Can(authz.TemplatesEdit) || p.WorkspaceID != ws.ID {
		return BrandVoice{}, api.Forbidden(authz.TemplatesEdit, nil)
	}
	if err := in.Content.Validate(ctx); err != nil {
		return BrandVoice{}, err
	}
	var out BrandVoice
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.BrandVoiceLock(ctx, ws.ID); err != nil {
			return err
		}
		if p.IsHuman() {
			fresh, err := VoiceActor(ctx, q, ws, p)
			if err != nil {
				return err
			}
			p = fresh
			if !p.Can(authz.TemplatesEdit) {
				return api.Forbidden(authz.TemplatesEdit, nil)
			}
		}
		sources := []uuid.UUID{}
		var request *uuid.UUID
		outcome := "accepted"
		if in.SuggestionID != "" {
			id, err := uuid.Parse(in.SuggestionID)
			if err != nil {
				return api.NotFound("Suggestion")
			}
			if !p.IsHuman() || !p.Can(authz.AIUse) {
				return api.Forbidden(authz.AIUse, nil)
			}
			// A repeated acceptance returns the revision originally saved.
			if rev, err := q.BrandVoiceAccepted(ctx, store.BrandVoiceAcceptedParams{WorkspaceID: ws.ID, RequestID: &id}); err == nil {
				r, err := q.BrandVoiceRevision(ctx, store.BrandVoiceRevisionParams{WorkspaceID: ws.ID, Revision: rev})
				if err != nil {
					return err
				}
				if r.CreatedBy == nil || *r.CreatedBy != p.UserID {
					return api.NotFound("Suggestion")
				}
				if _, err := VoiceSources(ctx, q, ws, p, r.SourceCampaigns); err != nil {
					return err
				}
				out = BrandVoice{Revision: r.Revision, Learned: true, CreatedBy: r.CreatedByName, CreatedAt: &r.CreatedAt}
				if err := json.Unmarshal(r.Content, &out.Content); err != nil {
					return err
				}
				if out.Content != in.Content || in.BaseRevision != rev-1 {
					return api.Conflict(say(ctx, msgs.M("voice.changed")))
				}
				return nil
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			r, err := q.BrandVoiceSuggestion(ctx, store.BrandVoiceSuggestionParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Now: s.now()})
			if errors.Is(err, pgx.ErrNoRows) {
				return api.Conflict(say(ctx, msgs.M("voice.gone")))
			}
			if err != nil {
				return err
			}
			if in.BaseRevision != r.BaseRevision {
				return api.Conflict(say(ctx, msgs.M("voice.changed")))
			}
			if _, err := VoiceSources(ctx, q, ws, p, r.SourceCampaigns); err != nil {
				return err
			}
			var suggested BrandVoiceContent
			if err := json.Unmarshal(r.Content, &suggested); err != nil {
				return err
			}
			if suggested != in.Content {
				outcome = "accepted_edits"
			}
			sources, request = r.SourceCampaigns, &id
		}
		before, err := voice(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if before.Revision != in.BaseRevision {
			return api.Conflict(say(ctx, msgs.M("voice.changed")))
		}
		content, err := json.Marshal(in.Content)
		if err != nil {
			return err
		}
		now := s.now()
		out = BrandVoice{Revision: before.Revision + 1, Content: in.Content, Learned: request != nil, CreatedBy: p.Name, CreatedAt: &now}
		if err := q.BrandVoiceInsert(ctx, store.BrandVoiceInsertParams{WorkspaceID: ws.ID, Revision: out.Revision, Content: content, SourceCampaigns: sources, RequestID: request, CreatedBy: actor(p), CreatedAt: now}); err != nil {
			return err
		}
		if request != nil {
			if _, err := q.BrandVoiceSuggestionDelete(ctx, store.BrandVoiceSuggestionDeleteParams{ID: *request, WorkspaceID: ws.ID, UserID: p.UserID}); err != nil {
				return err
			}
			if _, err := q.AIRequestSetOutcome(ctx, store.AIRequestSetOutcomeParams{ID: *request, WorkspaceID: ws.ID, UserID: actor(p), Outcome: outcome, DecidedAt: &now}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "brand_voice.save", TargetType: "brand_voice", TargetID: ws.ID.String(),
			Before: map[string]any{"revision": before.Revision}, After: map[string]any{"revision": out.Revision}, Metadata: map[string]any{"request_id": request, "source_count": len(sources)}})
	})
	return out, err
}

func (s *Service) RejectVoice(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := VoiceActor(ctx, q, ws, p)
		if err != nil {
			return err
		}
		if !fresh.Can(authz.AIUse) {
			return api.Forbidden(authz.AIUse, nil)
		}
		n, err := q.BrandVoiceSuggestionDelete(ctx, store.BrandVoiceSuggestionDeleteParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID})
		if err != nil || n == 0 {
			return err
		}
		now := s.now()
		if _, err := q.AIRequestSetOutcome(ctx, store.AIRequestSetOutcomeParams{ID: id, WorkspaceID: ws.ID, UserID: actor(p), Outcome: "rejected", DecidedAt: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "brand_voice.reject", TargetType: "ai_request", TargetID: id.String()})
	})
}
