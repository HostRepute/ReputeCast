package agency

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Search across clients finds campaigns, automations, templates, forms and
// library items by name, in the clients the person works in. Contacts are
// found only by an exact email address, only in clients where the person
// holds subscribers:view, and every such search is recorded in the audit
// log: who searched, and for which address.

// AgencySearchResults are the results grouped by client, then the library.
type AgencySearchResults struct {
	Query string `json:"query"`
	// Contact is true when the query was an email address: contacts were
	// searched by exact address, and the search was recorded.
	Contact bool                `json:"contact"`
	Groups  []AgencySearchGroup `json:"groups"`
	Library []AgencySearchHit   `json:"library"`
	Total   int                 `json:"total"`
	// Searched counts the clients searched for contacts (where you can see
	// contacts), and Skipped the ones you can't.
	Searched int `json:"searched"`
	Skipped  int `json:"skipped"`
}

// AgencySearchGroup is one client's results.
type AgencySearchGroup struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Hits      []AgencySearchHit  `json:"hits"`
}

// AgencySearchHit is one result.
type AgencySearchHit struct {
	Kind string `json:"kind" enum:"campaign,automation,template,section,form,contact,library"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status is the campaign's, automation's or form's; LibraryKind and
	// Version describe a library item.
	Status      string `json:"status,omitempty"`
	LibraryKind string `json:"libraryKind,omitempty"`
	Version     int    `json:"version,omitempty"`
	// Contact details: lists, and when they subscribed or left.
	Lists          []string   `json:"lists,omitempty"`
	LeftLists      []string   `json:"leftLists,omitempty"`
	SubscribedAt   *time.Time `json:"subscribedAt,omitempty"`
	UnsubscribedAt *time.Time `json:"unsubscribedAt,omitempty"`
}

// looksLikeEmail is a query that names one address.
func looksLikeEmail(q string) bool {
	at := strings.Index(q, "@")
	return at > 0 && at < len(q)-1 && !strings.ContainsAny(q, " ,;") && strings.Contains(q[at:], ".")
}

// Search searches the caller's clients.
func (s *Service) Search(ctx context.Context, query string) (AgencySearchResults, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencySearchResults{}, err
	}
	q := strings.TrimSpace(query)
	out := AgencySearchResults{Query: q, Groups: []AgencySearchGroup{}, Library: []AgencySearchHit{}}
	if len([]rune(q)) < 2 {
		return out, api.Invalid("query.q", mSearchShort.Text(i18n.LangFrom(ctx), nil))
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	ids := set.ids()
	groups := map[uuid.UUID]*AgencySearchGroup{}
	order := []uuid.UUID{}
	group := func(id uuid.UUID) *AgencySearchGroup {
		if g, ok := groups[id]; ok {
			return g
		}
		w, _ := set.find(id)
		g := &AgencySearchGroup{Workspace: wsRef(w), Hits: []AgencySearchHit{}}
		groups[id] = g
		order = append(order, id)
		return g
	}
	if looksLikeEmail(q) {
		out.Contact = true
		// Only where the person can see contacts.
		allowed := []uuid.UUID{}
		for _, id := range ids {
			if set.permsIn(id).Has(authz.SubscribersView) {
				allowed = append(allowed, id)
			}
		}
		out.Searched, out.Skipped = len(allowed), len(ids)-len(allowed)
		rows, err := s.Q.AgencySearchContacts(ctx, store.AgencySearchContactsParams{WorkspaceIds: allowed, Email: strings.ToLower(q)})
		if err != nil {
			return out, err
		}
		found := map[uuid.UUID]bool{}
		for _, r := range rows {
			g := group(r.WorkspaceID)
			g.Hits = append(g.Hits, AgencySearchHit{Kind: "contact", ID: r.ID.String(), Name: r.Email, Lists: r.Lists, LeftLists: r.LeftLists,
				SubscribedAt: unixTime(r.SubscribedUnix), UnsubscribedAt: unixTime(r.UnsubscribedUnix)})
			found[r.WorkspaceID] = true
		}
		if err := s.recordContactSearch(ctx, set, q, allowed, found); err != nil {
			return out, err
		}
	} else {
		pattern := "%" + escapeLike(q) + "%"
		rows, err := s.Q.AgencySearchNames(ctx, store.AgencySearchNamesParams{WorkspaceIds: ids, Pattern: pattern})
		if err != nil {
			return out, err
		}
		// Copies of library items show once, under the library.
		copies := map[uuid.UUID]bool{}
		checked := map[uuid.UUID]bool{}
		for _, r := range rows {
			if (r.Kind != "section" && r.Kind != "template" && r.Kind != "automation") || checked[r.WorkspaceID] {
				continue
			}
			checked[r.WorkspaceID] = true
			cs, err := s.Q.AgencyCopiesOfWorkspace(ctx, r.WorkspaceID)
			if err != nil {
				return out, err
			}
			for _, c := range cs {
				copies[c.TargetID] = true
			}
		}
		for _, r := range rows {
			if copies[r.ID] {
				continue
			}
			g := group(r.WorkspaceID)
			hit := AgencySearchHit{Kind: r.Kind, ID: r.ID.String(), Name: r.Name}
			if r.Status != "saved" {
				hit.Status = r.Status
			}
			g.Hits = append(g.Hits, hit)
		}
		lib, err := s.Q.AgencySearchLibrary(ctx, pattern)
		if err != nil {
			return out, err
		}
		for _, r := range lib {
			out.Library = append(out.Library, AgencySearchHit{Kind: "library", ID: r.ID.String(), Name: r.Name, LibraryKind: r.Kind, Version: int(r.Version)})
		}
	}
	// Clients in the order of their best hit: the most recently changed first.
	for _, id := range order {
		g := groups[id]
		out.Groups = append(out.Groups, *g)
		out.Total += len(g.Hits)
	}
	out.Total += len(out.Library)
	return out, nil
}

// recordContactSearch writes the search into the audit log: once for the
// agency (the instance's log), and in each client where the address was
// found, so the client's own log shows it.
func (s *Service) recordContactSearch(ctx context.Context, set workspaceSet, address string, searched []uuid.UUID, found map[uuid.UUID]bool) error {
	names := []string{}
	for _, id := range searched {
		if found[id] {
			w, _ := set.find(id)
			names = append(names, w.Name)
		}
	}
	if err := s.Audit.Log(ctx, audit.Event{Action: "agency.contact_searched", TargetType: "contact_search", TargetName: address,
		After:    map[string]any{"query": address, "workspaces_searched": len(searched), "found_in": names},
		Metadata: map[string]any{"permission": string(authz.SubscribersView), "summary": "contact search across clients"}}); err != nil {
		return err
	}
	for _, id := range searched {
		if !found[id] {
			continue
		}
		wsID := id
		if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "agency.contact_searched", TargetType: "contact_search", TargetName: address,
			After:    map[string]any{"query": address, "found": true},
			Metadata: map[string]any{"permission": string(authz.SubscribersView), "summary": "searched from the agency overview"}}); err != nil {
			return err
		}
	}
	return nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
