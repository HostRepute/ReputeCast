// Package welcomeprompt offers the Welcome series starter where a list
// starts getting subscribers (add-on transactional §6): List health after a
// list is created, the form builder when a form goes live, and the import
// summary after an import finishes. A prompt names the first list whose new
// subscribers get no welcome email: no live automation starts on it (or, on
// the form builder, on the form), and nobody dismissed it for that list.
package welcomeprompt

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Contexts where a prompt shows.
const (
	ContextHealth = "health"
	ContextForm   = "form"
	ContextImport = "import"
)

// Starter is the automation template the prompt opens.
const Starter = "welcome-series"

// Service holds the prompts' dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// WelcomePrompt is the card: "New subscribers of Product Updates get no
// welcome email yet".
type WelcomePrompt struct {
	ListID   string `json:"listId"`
	ListName string `json:"listName"`
	// CoveredLists are the lists a live automation already welcomes, for
	// "Weekly Digest already has one".
	CoveredLists []string `json:"coveredLists"`
	// Starter is the automation template "Start the welcome series" opens,
	// with the list as its trigger.
	Starter string `json:"starter" enum:"welcome-series"`
	// DraftID is an automation that isn't live yet and starts on this list:
	// the action opens it instead of making another.
	DraftID string `json:"draftId,omitempty"`
}

// WelcomePromptResult is the prompt for a screen, if any.
type WelcomePromptResult struct {
	Prompt *WelcomePrompt `json:"prompt,omitempty"`
}

type trigger struct {
	Kind   string `json:"kind"`
	ListID string `json:"listId"`
	FormID string `json:"formId"`
}

// coverage is what the workspace's automations start on.
type coverage struct {
	lists  map[uuid.UUID]bool      // a live automation starts on the list
	forms  map[uuid.UUID]bool      // a live automation starts on the form
	drafts map[uuid.UUID]uuid.UUID // list → an automation not live yet
}

func (s *Service) coverage(ctx context.Context, ws uuid.UUID) (coverage, error) {
	c := coverage{lists: map[uuid.UUID]bool{}, forms: map[uuid.UUID]bool{}, drafts: map[uuid.UUID]uuid.UUID{}}
	rows, err := s.Q.WelcomeListTriggers(ctx, ws)
	if err != nil {
		return c, err
	}
	for _, r := range rows {
		var t trigger
		if json.Unmarshal(r.Trigger, &t) != nil {
			continue
		}
		switch t.Kind {
		case "joined_list":
			id, err := uuid.Parse(t.ListID)
			if err != nil {
				continue
			}
			if r.Status == "live" {
				c.lists[id] = true
			} else if _, ok := c.drafts[id]; !ok {
				c.drafts[id] = r.ID
			}
		case "form_submitted":
			if id, err := uuid.Parse(t.FormID); err == nil && r.Status == "live" {
				c.forms[id] = true
			}
		}
	}
	return c, nil
}

// Prompt returns the prompt for a screen: List health (no id), a form, or
// a finished import.
func (s *Service) Prompt(ctx context.Context, ws *store.Workspace, kind string, id uuid.UUID) (WelcomePromptResult, error) {
	out := WelcomePromptResult{}
	lists, err := s.Q.WelcomeLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.WelcomeListsRow{}
	byName := map[string]uuid.UUID{}
	for _, l := range lists {
		byID[l.ID] = l
		if _, ok := byName[strings.ToLower(l.Name)]; !ok {
			byName[strings.ToLower(l.Name)] = l.ID
		}
	}
	cov, err := s.coverage(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	var candidates []uuid.UUID
	switch kind {
	case ContextHealth:
		for _, l := range lists {
			candidates = append(candidates, l.ID)
		}
	case ContextForm:
		f, err := s.Q.WelcomeForm(ctx, store.WelcomeFormParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return out, api.MapError(err, "Form")
		}
		if f.Status != "live" || cov.forms[f.ID] {
			return out, nil
		}
		candidates = f.ListIds
	case ContextImport:
		imp, err := s.Q.WelcomeImport(ctx, store.WelcomeImportParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return out, api.MapError(err, "Import")
		}
		if imp.Status != "done" {
			return out, nil
		}
		var opts struct {
			Lists []struct {
				Target string `json:"target"`
			} `json:"lists"`
		}
		_ = json.Unmarshal(imp.Options, &opts)
		for _, l := range opts.Lists {
			if name, ok := strings.CutPrefix(l.Target, "new:"); ok {
				if lid, ok := byName[strings.ToLower(strings.TrimSpace(name))]; ok {
					candidates = append(candidates, lid)
				}
				continue
			}
			if lid, err := uuid.Parse(l.Target); err == nil {
				candidates = append(candidates, lid)
			}
		}
	default:
		return out, api.Invalid("query.context", mContext.Text(i18n.LangFrom(ctx), nil))
	}
	dismissedIDs, err := s.Q.WelcomeDismissedLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	dismissed := map[uuid.UUID]bool{}
	for _, d := range dismissedIDs {
		dismissed[d] = true
	}
	for _, c := range candidates {
		l, ok := byID[c]
		if !ok || cov.lists[c] || dismissed[c] {
			continue
		}
		p := &WelcomePrompt{ListID: c.String(), ListName: l.Name, Starter: Starter, CoveredLists: []string{}}
		if d, ok := cov.drafts[c]; ok {
			p.DraftID = d.String()
		}
		// Oldest first, as the lists were created.
		for i := len(lists) - 1; i >= 0; i-- {
			if cov.lists[lists[i].ID] {
				p.CoveredLists = append(p.CoveredLists, lists[i].Name)
			}
		}
		out.Prompt = p
		return out, nil
	}
	return out, nil
}

// Dismiss hides the prompt for a list, for everyone in the workspace
// ("Not now").
func (s *Service) Dismiss(ctx context.Context, ws *store.Workspace, by *uuid.UUID, listID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		l, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: listID})
		if err != nil {
			return api.MapError(err, "List")
		}
		if err := q.WelcomeDismiss(ctx, store.WelcomeDismissParams{WorkspaceID: ws.ID, ListID: l.ID, DismissedBy: by, DismissedAt: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation.welcome_prompt_dismissed", TargetType: "list", TargetID: l.ID.String(), TargetName: l.Name,
			Metadata: map[string]any{"starter": Starter}})
	})
}
