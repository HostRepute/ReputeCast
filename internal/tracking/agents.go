package tracking

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/useragent"
)

// The user agents of machines that open and click email are a list the
// instance keeps (tracking_user_agents, review AN-2), as its networks are:
// the built-in ones come with releases, and instance admins add the ones
// they meet, and the words of people's devices a rule takes for machines.
// The classifier reads the list from memory, refreshed every few minutes;
// without it, the built-in list.

type agentCache struct {
	mu   sync.RWMutex
	at   time.Time
	list *useragent.List
}

// agents returns the instance's user agent list, read again when it is
// older than netRefresh; nil (the built-in list) when it can't be read.
func (s *Service) agents(ctx context.Context) *useragent.List {
	s.uas.mu.RLock()
	list, at := s.uas.list, s.uas.at
	s.uas.mu.RUnlock()
	if list != nil && time.Since(at) < netRefresh {
		return list
	}
	rows, err := s.Q.TrackingUserAgentList(ctx)
	if err != nil {
		s.log().WarnContext(ctx, "read tracking user agents", "err", err)
		return list
	}
	ps := make([]useragent.Pattern, 0, len(rows))
	for _, r := range rows {
		ps = append(ps, useragent.Pattern{Text: r.Pattern, Kind: r.Kind, Name: r.Name})
	}
	fresh := useragent.NewList(ps)
	s.uas.mu.Lock()
	s.uas.list, s.uas.at = fresh, time.Now()
	s.uas.mu.Unlock()
	return fresh
}

// forgetAgents makes the next lookup read the list again.
func (s *Service) forgetAgents() {
	s.uas.mu.Lock()
	s.uas.at = time.Time{}
	s.uas.mu.Unlock()
}

// TrackingUserAgent is one entry of the user agent list.
type TrackingUserAgent struct {
	// Pattern is found in a machine's user agent (any text), or is a word
	// of a person's device (letters and digits).
	Pattern string `json:"pattern" example:"acmegate-linkcheck"`
	// Kind: scanner (a mail-security scanner or link checker), script,
	// headless (a headless browser), preview (a link preview), or person (a
	// person's device a rule would take for a machine).
	Kind string `json:"kind" enum:"scanner,script,headless,preview,person"`
	// Name is the product or who runs it; a script, headless browser or
	// link preview may have none.
	Name string `json:"name"`
	// Source: builtin (comes with ReputeCast) or admin (added here).
	Source        string    `json:"source" enum:"builtin,admin"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedByName string    `json:"createdByName,omitempty"`
}

// TrackingUserAgentList lists the user agents.
type TrackingUserAgentList struct {
	Items []TrackingUserAgent `json:"items"`
}

// TrackingUserAgentInput adds a user agent.
type TrackingUserAgentInput struct {
	Pattern string `json:"pattern" minLength:"3" maxLength:"80" doc:"Text found in the user agent: AcmeGate-LinkCheck. A person's device: one word, KioskBot"`
	// Kind is scanner for a machine (a scanner, link checker or script),
	// person for a person's device or app a rule takes for a machine.
	Kind string `json:"kind" enum:"scanner,person"`
	Name string `json:"name" minLength:"1" maxLength:"80" doc:"Who or what it is: our mail gateway, lobby kiosks"`
	Note string `json:"note,omitempty" maxLength:"300"`
}

// UserAgents lists the instance's user agents.
func (s *Service) UserAgents(ctx context.Context) (TrackingUserAgentList, error) {
	rows, err := s.Q.TrackingUserAgentList(ctx)
	if err != nil {
		return TrackingUserAgentList{}, err
	}
	out := TrackingUserAgentList{Items: make([]TrackingUserAgent, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, TrackingUserAgent{Pattern: r.Pattern, Kind: r.Kind, Name: r.Name, Source: r.Source, Note: r.Note, CreatedAt: r.CreatedAt,
			CreatedByName: r.CreatedByName})
	}
	return out, nil
}

// AddUserAgent adds a user agent to the list (audited). A machine's
// pattern found in the user agent of a browser or mail client people read
// email with is refused: it would take them for machines.
func (s *Service) AddUserAgent(ctx context.Context, in TrackingUserAgentInput) (TrackingUserAgent, error) {
	l := i18n.LangFrom(ctx)
	pattern := strings.ToLower(strings.TrimSpace(in.Pattern))
	name, note := strings.TrimSpace(in.Name), strings.TrimSpace(in.Note)
	switch {
	case len([]rune(pattern)) < 3:
		return TrackingUserAgent{}, api.Invalid("body.pattern", msgs.M("problem.agentShort").Text(l, nil))
	case in.Kind == useragent.KindPerson && !useragent.Word(pattern):
		return TrackingUserAgent{}, api.Invalid("body.pattern", msgs.M("problem.agentWord").Text(l, nil))
	case in.Kind != useragent.KindPerson && useragent.TakesPeople(pattern):
		return TrackingUserAgent{}, api.Invalid("body.pattern", msgs.M("problem.agentPeople").Text(l, i18n.Args{"pattern": pattern}))
	case name == "":
		return TrackingUserAgent{}, api.Invalid("body.name", msgs.M("problem.agentName").Text(l, nil))
	}
	var by *uuid.UUID
	if pr := api.PrincipalFrom(ctx); pr != nil && pr.UserID != uuid.Nil {
		u := pr.UserID
		by = &u
	}
	var out TrackingUserAgent
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.TrackingUserAgentAdd(ctx, store.TrackingUserAgentAddParams{Pattern: pattern, Kind: in.Kind, Name: name, Note: note, CreatedBy: by})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Conflict(msgs.M("problem.agentListed").Text(l, i18n.Args{"pattern": pattern}))
		}
		if err != nil {
			return err
		}
		out = TrackingUserAgent{Pattern: row.Pattern, Kind: in.Kind, Name: name, Source: "admin", Note: note, CreatedAt: row.CreatedAt}
		if p := api.PrincipalFrom(ctx); p != nil {
			out.CreatedByName = p.Name
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "tracking_user_agent.create", TargetType: "tracking_user_agent", TargetID: pattern, TargetName: name,
			After: map[string]any{"pattern": pattern, "kind": in.Kind, "name": name, "note": note}})
	})
	if err != nil {
		return TrackingUserAgent{}, err
	}
	s.forgetAgents()
	return out, nil
}

// RemoveUserAgent takes a user agent an admin added off the list (audited).
func (s *Service) RemoveUserAgent(ctx context.Context, pattern string) error {
	l := i18n.LangFrom(ctx)
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.TrackingUserAgentDelete(ctx, pattern)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Err(http.StatusNotFound, api.CodeNotFound, msgs.M("problem.agentNotRemovable").Text(l, i18n.Args{"pattern": pattern}))
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "tracking_user_agent.delete", TargetType: "tracking_user_agent", TargetID: row.Pattern, TargetName: row.Name,
			Before: map[string]any{"pattern": row.Pattern, "kind": row.Kind, "name": row.Name, "note": row.Note}})
	})
	if err != nil {
		return err
	}
	s.forgetAgents()
	return nil
}

// registerUserAgents adds the instance admin's operations on the user
// agents (RegisterNetworks).
func registerUserAgents(h huma.API, s *Service, tags []string) {
	base := "/admin/tracking-user-agents"
	api.Get(h, api.Op{ID: "list-tracking-user-agents", Path: base, Summary: "User agents of machines whose opens and clicks aren't people's",
		Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[TrackingUserAgentList], error) {
			out, err := s.UserAgents(ctx)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-tracking-user-agent", Path: base, Summary: "Add the user agent of a machine, or of a person's device a rule takes for one (audited)",
		Description: "A machine's pattern is found anywhere in a user agent: its opens and clicks count as a machine's. A person's is one word of a device's " +
			"user agent that a rule would take for a machine's (a word ending in “bot”): its opens and clicks count as a person's again.",
		Tags: tags, Access: api.InstanceAdmin, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body TrackingUserAgentInput }) (*api.Body[TrackingUserAgent], error) {
			out, err := s.AddUserAgent(ctx, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-tracking-user-agent", Path: base, Summary: "Remove a user agent an admin added (audited)", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			Pattern string `query:"pattern" required:"true" maxLength:"80" doc:"The pattern, as listed: acmegate-linkcheck"`
		}) (*api.Empty, error) {
			return api.NoContent, s.RemoveUserAgent(ctx, in.Pattern)
		})
}
