package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/store"
)

// What internal/ai/assistant uses of this package (B3): the model a thread
// uses, its budget and the editorial projections of what a person attaches.
// The assistant never reaches a connection, a key or a price but through
// these.

// AssistantModel is a model an assistant thread uses, with its account and
// the workspace's AI settings as they were read.
type AssistantModel struct{ c *connection }

// ID is the model's id.
func (m *AssistantModel) ID() uuid.UUID { return m.c.ModelID }

// Personal reports whether the model is on its owner's own key.
func (m *AssistantModel) Personal() bool { return m.c.personal() }

// AIAssistantModel is a model the assistant can use, as the model picker
// shows it: its account, prices and whether a thread can use it now.
type AIAssistantModel struct {
	ID           uuid.UUID    `json:"id"`
	AccountID    uuid.UUID    `json:"accountId"`
	Provider     string       `json:"provider" enum:"anthropic,openai,google,azure,compatible"`
	ProviderName string       `json:"providerName"`
	Model        string       `json:"model"`
	ModelName    string       `json:"modelName"`
	ImageInput   bool         `json:"imageInput"`
	Prices       AIPricesView `json:"prices"`
	// Personal: the model is on the caller's own key.
	Personal bool `json:"personal"`
	// Tools is its tool-calling check; the assistant needs one that passed.
	Tools AIAgentCapability `json:"tools"`
	// Usable: a thread can use it now; Reason says why it can't.
	Usable bool   `json:"usable"`
	Reason string `json:"reason,omitempty" enum:"disconnected,key_rejected,tools_unchecked,personal_keys_off"`
}

// AIAssistantModelList is the models the caller may pick for a thread.
type AIAssistantModelList struct {
	Models []AIAssistantModel `json:"models"`
	// DefaultID is the model a new thread uses: the caller's own pick for
	// the assistant, or else the workspace's.
	DefaultID *uuid.UUID `json:"defaultId,omitempty" format:"uuid"`
}

// assistantView is how the picker shows a model.
func (s *Service) assistantView(ctx context.Context, q *store.Queries, c *connection) (AIAssistantModel, error) {
	tools, err := agentCapability(ctx, q, c)
	if err != nil {
		return AIAssistantModel{}, err
	}
	v := AIAssistantModel{ID: c.ModelID, AccountID: c.AccountID, Provider: c.Provider, ProviderName: c.providerName(), Model: c.Model, ModelName: c.modelName(),
		ImageInput: c.ImageInput, Prices: AIPricesView{Input: c.InputPrice, Output: c.OutputPrice, Known: c.PriceKnown || c.Demo, From: c.PricesFrom},
		Personal: c.personal(), Tools: tools}
	switch {
	case !c.connected():
		v.Reason = "disconnected"
	case c.Status == "key_rejected":
		v.Reason = "key_rejected"
	case c.personal() && !c.PersonalKeys:
		v.Reason = "personal_keys_off"
	case tools.State != "supported":
		v.Reason = "tools_unchecked"
	}
	v.Usable = v.Reason == ""
	return v, nil
}

// AssistantModels lists the workspace's models and the caller's own, each
// with whether a thread can use it.
func (s *Service) AssistantModels(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIAssistantModelList, error) {
	out := AIAssistantModelList{Models: []AIAssistantModel{}}
	m, err := s.models(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	list := m.list
	if user := actorID(p); user != nil {
		ids, err := s.Q.AIPersonalModels(ctx, store.AIPersonalModelsParams{WorkspaceID: ws.ID, UserID: user})
		if err != nil {
			return out, err
		}
		for _, id := range ids {
			c, err := s.binding(ctx, s.Q, ws.ID, &id)
			if err != nil {
				return out, err
			}
			if c != nil {
				list = append(list, c)
			}
		}
		if c, err := s.loadFor(ctx, s.Q, ws.ID, user, FeatureAssistant); err != nil {
			return out, err
		} else if c != nil {
			out.DefaultID = &c.ModelID
		}
	}
	for _, c := range list {
		v, err := s.assistantView(ctx, s.Q, c)
		if err != nil {
			return out, err
		}
		out.Models = append(out.Models, v)
	}
	return out, nil
}

// ResolveAssistantModel is the model a thread of user uses: the one named,
// or else their own pick for the assistant, or the workspace's. Another
// person's personal model isn't there; a model the assistant can't use
// (disconnected, its key rejected, its tool check not passed, personal keys
// off) is refused with the reason.
func (s *Service) ResolveAssistantModel(ctx context.Context, q *store.Queries, ws, user uuid.UUID, model *uuid.UUID) (*AssistantModel, error) {
	id := uuid.Nil
	if model != nil {
		id = *model
	} else {
		pick, ok, err := featureModel(ctx, q, ws, &user, FeatureAssistant)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, problem(ctx, http.StatusConflict, codeNotConnected, mNotConnectedYet, nil)
		}
		id = pick
	}
	c, err := s.binding(ctx, q, ws, &id)
	if err != nil {
		return nil, err
	}
	if c == nil || (c.personal() && !c.ownedBy(&user)) {
		return nil, api.NotFound("AI model")
	}
	if c.personal() && !c.PersonalKeys {
		return nil, personalKeysOff(ctx)
	}
	if err := suits(ctx, q, c, FeatureAssistant); err != nil {
		return nil, err
	}
	return &AssistantModel{c}, nil
}

// AssistantModelView is how a thread shows its model, also when the
// assistant can no longer use it; nil when it is gone or isn't user's.
func (s *Service) AssistantModelView(ctx context.Context, q *store.Queries, ws, user uuid.UUID, model *uuid.UUID) (*AIAssistantModel, error) {
	c, err := s.binding(ctx, q, ws, model)
	if err != nil || c == nil || (c.personal() && !c.ownedBy(&user)) {
		return nil, err
	}
	v, err := s.assistantView(ctx, q, c)
	return &v, err
}

// ReadyAssistant checks that a thread's model can take a request now: the
// key works, the switch and the caps allow it.
func (s *Service) ReadyAssistant(ctx context.Context, ws *store.Workspace, m *AssistantModel) error {
	_, err := s.ready(ctx, ws, m.c, FeatureAssistant)
	return err
}

// AssistantLedger is a thread's cumulative ceilings.
type AssistantLedger struct {
	ThreadID                    uuid.UUID
	TokenLimit, CostLimitMicros int64
}

// ReserveAssistant reserves one model request of a thread: against the
// monthly cap (the workspace's or the key owner's) and the thread's
// ceilings, at the model's prices. It bounds req's answer to what is left.
// The caller holds the thread's lock; id is the request's.
func (s *Service) ReserveAssistant(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, m *AssistantModel, req *llm.Request, l AssistantLedger, id uuid.UUID) error {
	t := task{feature: FeatureAssistant, req: *req}
	if err := s.reserveIn(ctx, q, ws, p, m.c, &t, &ledger{threadID: &l.ThreadID, tokenLimit: l.TokenLimit, costLimitMicros: l.CostLimitMicros}, id); err != nil {
		return err
	}
	*req = t.req
	return nil
}

// AssistantRequest is a thread's request as the activity log keeps it:
// never its content, nor the thread's title.
type AssistantRequest struct {
	ID, WorkspaceID, UserID, ThreadID uuid.UUID
	Action, Outcome                   string
	CreatedAt                         time.Time
}

// AccountAssistant settles a reservation with what the provider reported
// (or keeps it held when that is unknown) and logs the request. It reports
// whether the usage is known.
func (s *Service) AccountAssistant(ctx context.Context, q *store.Queries, r AssistantRequest, res llm.Response, callErr error) (bool, error) {
	return s.accountIn(ctx, q, store.AIRequestInsertParams{ID: r.ID, WorkspaceID: r.WorkspaceID, UserID: &r.UserID, Feature: FeatureAssistant, Action: r.Action,
		TargetKind: "assistant_thread", TargetID: &r.ThreadID, Outcome: r.Outcome, CreatedAt: r.CreatedAt}, res, callErr)
}

// AssistantProvider is how a thread's requests reach the model.
func (s *Service) AssistantProvider(m *AssistantModel) (llm.Provider, error) {
	return s.provider(m.c, FeatureAssistant)
}

// AssistantAttachment is a resource attached to a thread, as its owner may
// read it now.
type AssistantAttachment struct {
	Kind     string
	ID       uuid.UUID
	Title    string
	Revision int64
	// Snapshot is a report's fixed numbers (reports.ModelSnapshot), taken
	// when it is attached.
	Snapshot json.RawMessage
}

// AssistantAttachment checks that p may read a resource to attach and
// returns what the thread shows of it; with snapshot, a report's numbers as
// the model reads them (the agent's report rules). Each read checks the
// person's current access: a resource they can't read isn't found.
func (s *Service) AssistantAttachment(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind string, id uuid.UUID, snapshot bool) (AssistantAttachment, error) {
	out := AssistantAttachment{Kind: kind, ID: id}
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return out, err
	}
	switch kind {
	case "campaign":
		if err := agentKindAccess(fresh, "campaign", false); err != nil {
			return out, err
		}
		c, err := draftCampaign(ctx, q, ws, fresh, id)
		if err != nil {
			return out, err
		}
		out.Title, out.Revision = c.Name, int64(c.Version)
	case "report":
		if !fresh.Can(authz.ReportsView) || !fresh.Can(authz.CampaignsView) {
			return out, api.Forbidden(authz.ReportsView, nil)
		}
		c, err := reports.ModelCampaign(ctx, q, fresh, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return out, api.NotFound("Report")
		}
		if err != nil {
			return out, err
		}
		out.Title = c.Name
		if snapshot {
			snap, err := reports.ReadModelSnapshot(ctx, q, fresh, id, s.now())
			if err != nil {
				return out, agentProblem(ctx, "report_unavailable")
			}
			if out.Snapshot, err = json.Marshal(snap); err != nil {
				return out, err
			}
		}
	default:
		return out, api.NotFound("Resource")
	}
	return out, nil
}

// AsksAboutReputation reports whether a message asks about sender
// reputation, blocklists or placement: HostRepute's, not the assistant's.
func AsksAboutReputation(text string) bool { return asksAboutReputation(text) }
