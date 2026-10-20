package agency

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// The shared library holds saved sections, campaign templates, automation
// templates, transactional templates and SMS texts, each versioned. Items
// come from any client ("Add to Northwind library"); using one in a client
// makes a copy there. Copies aren't linked: when the item changes, the copy
// shows "Library has a newer version", with the diff and "Update this copy".

// Library item kinds and where their copies go.
const (
	KindSection       = "section"
	KindTemplate      = "template"
	KindAutomation    = "automation"
	KindTransactional = "transactional"
	KindSMS           = "sms"
)

// LibraryContent is one version of an item.
type LibraryContent struct {
	// Sections and campaign templates.
	MJML  string          `json:"mjml,omitempty"`
	HTML  string          `json:"html,omitempty"`
	Thumb json.RawMessage `json:"thumb,omitempty"`
	// Automation templates.
	Automation *TplAutomation `json:"automation,omitempty"`
	// Transactional templates.
	Subject   string                                `json:"subject,omitempty"`
	Preheader string                                `json:"preheader,omitempty"`
	PlainText string                                `json:"plainText,omitempty"`
	Variables []transactional.TransactionalVariable `json:"variables,omitempty"`
	// SMS texts.
	Body string `json:"body,omitempty"`
}

func parseLibraryContent(raw json.RawMessage) LibraryContent {
	var c LibraryContent
	_ = json.Unmarshal(raw, &c)
	return c
}

// AgencyLibraryItem is an item of the shared library.
type AgencyLibraryItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind" enum:"section,template,automation,transactional,sms"`
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	// Clients counts the clients with a copy; Behind the copies on an older version.
	Clients int `json:"clients"`
	Copies  int `json:"copies"`
	Behind  int `json:"behind"`
	// Fields are the Liquid fields it needs, with their fallbacks.
	Fields []AgencyLibraryField `json:"fields"`
	// Thumb is the gallery wireframe (sections and templates); Body an SMS text.
	Thumb json.RawMessage `json:"thumb,omitempty"`
	Body  string          `json:"body,omitempty"`
	// Steps counts an automation template's steps.
	Steps int `json:"steps,omitempty"`
}

// AgencyLibraryField is a Liquid field an item needs.
type AgencyLibraryField struct {
	Key      string `json:"key"`
	Fallback string `json:"fallback,omitempty"`
}

// AgencyLibraryView is the Library page's shared library.
type AgencyLibraryView struct {
	Items  []AgencyLibraryItem `json:"items"`
	Counts map[string]int      `json:"counts"`
	Total  int                 `json:"total"`
	// CanEdit: you hold agency:library (add, change and remove items).
	CanEdit bool     `json:"canEdit"`
	Reason  string   `json:"reason,omitempty"`
	Helpers []string `json:"helpers"`
}

func fieldsOf(raw json.RawMessage) []AgencyLibraryField {
	out := []AgencyLibraryField{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// libraryFields lists the Liquid fields content prints, leaving out the
// links ReputeCast fills in itself.
func libraryFields(srcs ...string) []AgencyLibraryField {
	out := []AgencyLibraryField{}
	seen := map[string]bool{}
	for _, src := range srcs {
		for _, u := range liquid.Uses(src) {
			if seen[u.Key] || slices.ContainsFunc(liquid.SystemFields, func(f liquid.PersonalizationToken) bool { return f.Key == u.Key }) {
				continue
			}
			seen[u.Key] = true
			out = append(out, AgencyLibraryField{Key: u.Key, Fallback: u.Fallback})
		}
	}
	return out
}

// Library lists the shared library, most used first.
func (s *Service) Library(ctx context.Context) (AgencyLibraryView, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyLibraryView{}, err
	}
	rows, err := s.Q.AgencyLibraryItems(ctx)
	if err != nil {
		return AgencyLibraryView{}, err
	}
	out := AgencyLibraryView{Items: []AgencyLibraryItem{}, Counts: map[string]int{}, CanEdit: me.Can(authz.AgencyLibrary), Helpers: s.managerNames(ctx, 2)}
	if !out.CanEdit {
		out.Reason = s.denied(ctx, authz.AgencyLibrary).Detail
	}
	for _, k := range []string{KindSection, KindTemplate, KindAutomation, KindTransactional, KindSMS} {
		out.Counts[k] = 0
	}
	for _, r := range rows {
		item := AgencyLibraryItem{ID: r.ID.String(), Kind: r.Kind, Name: r.Name, Version: int(r.Version), UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName,
			Clients: int(r.Clients), Copies: int(r.Copies), Behind: int(r.Behind), Fields: fieldsOf(r.Fields)}
		if v, err := s.Q.AgencyLibraryVersion(ctx, store.AgencyLibraryVersionParams{ItemID: r.ID, Version: r.Version}); err == nil {
			c := parseLibraryContent(v.Content)
			item.Thumb, item.Body = c.Thumb, c.Body
			if c.Automation != nil {
				if g, err := automations.ParseGraph(c.Automation.Graph); err == nil {
					item.Steps = len(g.Nodes)
				}
			}
		}
		out.Items = append(out.Items, item)
		out.Counts[r.Kind]++
	}
	// Most used first: the copies made, then the latest change.
	slices.SortStableFunc(out.Items, func(a, b AgencyLibraryItem) int {
		if a.Copies != b.Copies {
			return b.Copies - a.Copies
		}
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	out.Total = len(out.Items)
	return out, nil
}

// AgencyLibraryDetail is one item: its preview, versions, the fields it
// needs, and the copies in clients with the diff of the ones behind.
type AgencyLibraryDetail struct {
	Item AgencyLibraryItem `json:"item"`
	// Preview is the item's HTML (sections and templates) or text (SMS).
	Preview  string                 `json:"preview,omitempty"`
	Subject  string                 `json:"subject,omitempty"`
	Versions []AgencyLibraryVersion `json:"versions"`
	Copies   []AgencyLibraryCopy    `json:"copies"`
	// Diff compares the oldest version a copy is on with the latest.
	Diff *AgencyLibraryDiff `json:"diff,omitempty"`
	// Clients are where you can copy it.
	Clients []AgencyCopyTarget `json:"clients"`
	CanEdit bool               `json:"canEdit"`
}

// AgencyLibraryVersion is one version of an item.
type AgencyLibraryVersion struct {
	Version int       `json:"version"`
	At      time.Time `json:"at"`
	By      string    `json:"by,omitempty"`
	Note    string    `json:"note,omitempty"`
}

// AgencyLibraryCopy is a copy in a client.
type AgencyLibraryCopy struct {
	ID        string             `json:"id"`
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Version   int                `json:"version"`
	Behind    bool               `json:"behind"`
	CopiedAt  time.Time          `json:"copiedAt"`
	// CanUpdate: you can edit it there and it can be updated in place.
	CanUpdate bool   `json:"canUpdate"`
	Reason    string `json:"reason,omitempty"`
}

// AgencyLibraryDiff is what changed between two versions.
type AgencyLibraryDiff struct {
	From  int        `json:"from"`
	To    int        `json:"to"`
	Lines []DiffLine `json:"lines"`
}

// DiffLine is a removed (-) or added (+) line.
type DiffLine struct {
	Op   string `json:"op" enum:"del,add,ctx"`
	Text string `json:"text"`
}

// AgencyCopyTarget is a client an item can be copied to.
type AgencyCopyTarget struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	// CanCopy: you can edit that kind of content there.
	CanCopy bool   `json:"canCopy"`
	Reason  string `json:"reason,omitempty"`
}

// editPermission is the permission a copy needs in the client.
func editPermission(kind string) authz.Permission {
	switch kind {
	case KindAutomation:
		return authz.AutomationsEdit
	case KindTransactional:
		return authz.TransactionalEdit
	case KindSMS:
		return authz.CampaignsEdit
	}
	return authz.TemplatesEdit
}

// viewPermission is what adding an item from a workspace needs there.
func viewPermission(kind string) authz.Permission {
	switch kind {
	case KindAutomation:
		return authz.AutomationsView
	case KindTransactional:
		return authz.TransactionalView
	case KindSMS:
		return authz.CampaignsView
	}
	return authz.TemplatesView
}

// LibraryItem returns one item with its versions and copies.
func (s *Service) LibraryItem(ctx context.Context, id string) (AgencyLibraryDetail, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyLibraryDetail{}, err
	}
	l := i18n.LangFrom(ctx)
	iid, err := uuid.Parse(id)
	if err != nil {
		return AgencyLibraryDetail{}, api.NotFound("Library item")
	}
	r, err := s.Q.AgencyLibraryItem(ctx, iid)
	if err != nil {
		return AgencyLibraryDetail{}, api.MapError(err, "Library item")
	}
	out := AgencyLibraryDetail{Item: AgencyLibraryItem{ID: r.ID.String(), Kind: r.Kind, Name: r.Name, Version: int(r.Version), UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName,
		Fields: fieldsOf(r.Fields)}, Versions: []AgencyLibraryVersion{}, Copies: []AgencyLibraryCopy{}, Clients: []AgencyCopyTarget{}, CanEdit: me.Can(authz.AgencyLibrary)}
	versions, err := s.Q.AgencyLibraryVersions(ctx, iid)
	if err != nil {
		return out, err
	}
	summaries := map[int][]string{}
	for _, v := range versions {
		out.Versions = append(out.Versions, AgencyLibraryVersion{Version: int(v.Version), At: v.CreatedAt, By: v.CreatedByName, Note: v.Note})
		summaries[int(v.Version)] = v.Summary
		if v.Version == r.Version {
			c := parseLibraryContent(v.Content)
			out.Item.Thumb, out.Item.Body, out.Preview, out.Subject = c.Thumb, c.Body, c.HTML, c.Subject
			if r.Kind == KindSMS {
				out.Preview = c.Body
			}
			if c.Automation != nil {
				if g, err := automations.ParseGraph(c.Automation.Graph); err == nil {
					out.Item.Steps = len(g.Nodes)
				}
			}
		}
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	copies, err := s.Q.AgencyLibraryCopies(ctx, iid)
	if err != nil {
		return out, err
	}
	oldest := int(r.Version)
	clients := map[uuid.UUID]bool{}
	for _, c := range copies {
		w, ok := set.find(c.WorkspaceID)
		if !ok {
			continue
		}
		clients[c.WorkspaceID] = true
		cp := AgencyLibraryCopy{ID: c.ID.String(), Workspace: wsRef(w), Version: int(c.Version), Behind: c.Version < r.Version, CopiedAt: c.CopiedAt}
		if cp.Behind {
			cp.CanUpdate, cp.Reason = s.canUpdateCopy(l, r.Kind, set.permsIn(w.ID))
			oldest = min(oldest, cp.Version)
		}
		out.Copies = append(out.Copies, cp)
	}
	out.Item.Clients, out.Item.Copies = len(clients), len(out.Copies)
	for _, c := range out.Copies {
		if c.Behind {
			out.Item.Behind++
		}
	}
	if oldest < int(r.Version) {
		out.Diff = &AgencyLibraryDiff{From: oldest, To: int(r.Version), Lines: diffLines(summaries[oldest], summaries[int(r.Version)])}
	}
	for _, w := range set.visible {
		if w.Kind != "client" {
			continue
		}
		t := AgencyCopyTarget{Workspace: wsRef(w), CanCopy: set.permsIn(w.ID).Has(editPermission(r.Kind))}
		if !t.CanCopy {
			t.Reason = mCopyNeeds.Text(l, i18n.Args{"permission": string(editPermission(r.Kind)), "workspace": w.Name})
		}
		if r.Kind == KindSMS {
			t.CanCopy, t.Reason = false, mSMSFromComposer.Text(l, nil)
		}
		out.Clients = append(out.Clients, t)
	}
	return out, nil
}

func (s *Service) canUpdateCopy(l i18n.Lang, kind string, perms authz.Set) (bool, string) {
	switch kind {
	case KindAutomation:
		return false, mFlowNotInPlace.Text(l, nil)
	case KindSMS:
		return false, mSMSFromComposer.Text(l, nil)
	}
	if !perms.Has(editPermission(kind)) {
		return false, mCopyNeedsHere.Text(l, i18n.Args{"permission": string(editPermission(kind))})
	}
	return true, ""
}

// diffLines lists what was removed and added between two summaries, in
// the order of the older one, each removal next to what replaced it.
func diffLines(from, to []string) []DiffLine {
	out := []DiffLine{}
	removed := []string{}
	for _, l := range from {
		if !slices.Contains(to, l) {
			removed = append(removed, l)
		}
	}
	added := []string{}
	for _, l := range to {
		if !slices.Contains(from, l) {
			added = append(added, l)
		}
	}
	// Pair a removal with the addition about the same thing ("Button: …").
	used := make([]bool, len(added))
	for _, r := range removed {
		out = append(out, DiffLine{Op: "del", Text: r})
		label := labelOf(r)
		for i, a := range added {
			if !used[i] && label != "" && labelOf(a) == label {
				out = append(out, DiffLine{Op: "add", Text: a})
				used[i] = true
				break
			}
		}
	}
	for i, a := range added {
		if !used[i] {
			out = append(out, DiffLine{Op: "add", Text: a})
		}
	}
	return out
}

func labelOf(line string) string {
	if i := strings.Index(line, ":"); i > 0 {
		return line[:i]
	}
	return ""
}

var (
	tagRe   = regexp.MustCompile(`(?s)<mj-(button|image|text)\b([^>]*)>(.*?)</mj-(?:button|text)>|<mj-image\b([^>]*)/?>`)
	attrRe  = regexp.MustCompile(`([a-z-]+)="([^"]*)"`)
	stripRe = regexp.MustCompile(`(?s)<[^>]+>`)
	spaceRe = regexp.MustCompile(`\s+`)
	bgRe    = regexp.MustCompile(`<mj-(?:section|hero|wrapper|body)\b[^>]*background-color="([^"]+)"`)
)

// summaryLines are the lines the diff between versions compares: buttons,
// images and the first words of each text, and background colours, for
// MJML; steps for automations; subject and text for the others.
func summaryLines(kind string, c LibraryContent) []string {
	out := []string{}
	add := func(s string) {
		s = strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
		if s != "" && !slices.Contains(out, s) && len(out) < 16 {
			out = append(out, s)
		}
	}
	switch kind {
	case KindSMS:
		add("Text: " + c.Body)
		return out
	case KindAutomation:
		if c.Automation == nil {
			return out
		}
		g, err := automations.ParseGraph(c.Automation.Graph)
		if err != nil {
			return out
		}
		for _, n := range g.Nodes {
			label := n.Data.Label
			if label == "" {
				label = n.Type
			}
			add(strings.ToUpper(n.Type[:1]) + n.Type[1:] + ": " + label)
		}
		for _, e := range c.Automation.Emails {
			add("Email subject: " + e.Subject)
		}
		return out
	case KindTransactional:
		add("Subject: " + c.Subject)
	}
	for _, m := range bgRe.FindAllStringSubmatch(c.MJML, -1) {
		add("Background: " + m[1])
	}
	for _, m := range tagRe.FindAllStringSubmatch(c.MJML, -1) {
		switch m[1] {
		case "button":
			add("Button: " + stripRe.ReplaceAllString(m[3], ""))
		case "text":
			t := strings.TrimSpace(stripRe.ReplaceAllString(m[3], " "))
			if r := []rune(spaceRe.ReplaceAllString(t, " ")); len(r) > 60 {
				t = string(r[:60]) + "…"
			}
			add("Text: " + t)
		default:
			attrs := m[2] + m[4]
			alt := ""
			for _, a := range attrRe.FindAllStringSubmatch(attrs, -1) {
				if a[1] == "alt" {
					alt = a[2]
				}
			}
			if alt != "" {
				add("Image: " + alt)
			}
		}
	}
	return out
}

// AgencyAddToLibrary adds a workspace's item to the library, or saves it
// as a new version of the library item it was copied from.
type AgencyAddToLibrary struct {
	Kind string `json:"kind" enum:"section,template,automation,transactional"`
	ID   string `json:"id" format:"uuid" doc:"The saved section, campaign template, automation or transactional template"`
	// Name defaults to the item's own name.
	Name string `json:"name,omitempty" maxLength:"120"`
	// NewVersionOf saves it as a new version of this library item.
	NewVersionOf string `json:"newVersionOf,omitempty" format:"uuid"`
	Note         string `json:"note,omitempty" maxLength:"200"`
}

// AgencyLibraryAdded says where it went.
type AgencyLibraryAdded struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// AddToLibrary is "Add to Northwind library" in a client workspace. It
// needs agency:library and the item's view permission in that workspace.
func (s *Service) AddToLibrary(ctx context.Context, ws *store.Workspace, in AgencyAddToLibrary) (AgencyLibraryAdded, error) {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return AgencyLibraryAdded{}, err
	}
	l := i18n.LangFrom(ctx)
	p := api.PrincipalFrom(ctx)
	if !p.Can(viewPermission(in.Kind)) {
		return AgencyLibraryAdded{}, api.Forbidden(viewPermission(in.Kind), nil)
	}
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return AgencyLibraryAdded{}, api.NotFound("Item")
	}
	content, name, err := s.readItem(ctx, ws.ID, in.Kind, id)
	if err != nil {
		return AgencyLibraryAdded{}, err
	}
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	}
	raw, _ := json.Marshal(content)
	srcs := []string{content.MJML, content.Subject, content.Body}
	if content.Automation != nil {
		for _, e := range content.Automation.Emails {
			srcs = append(srcs, e.MJML, e.Subject)
		}
	}
	fields, _ := json.Marshal(libraryFields(srcs...))
	summary := summaryLines(in.Kind, content)
	now := s.now()
	out := AgencyLibraryAdded{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if in.NewVersionOf != "" {
			iid, err := uuid.Parse(in.NewVersionOf)
			if err != nil {
				return api.NotFound("Library item")
			}
			item, err := q.AgencyLibraryItem(ctx, iid)
			if err != nil {
				return api.MapError(err, "Library item")
			}
			if item.Kind != in.Kind {
				return api.Invalid("body.newVersionOf", mKindMismatch.Text(l, nil))
			}
			v := item.Version + 1
			if err := q.AgencyLibraryInsertVersion(ctx, store.AgencyLibraryInsertVersionParams{ItemID: iid, Version: v, Content: raw, Summary: summary,
				Note: strings.TrimSpace(in.Note), SourceWorkspaceID: &ws.ID, CreatedBy: &me.UserID, At: now}); err != nil {
				return err
			}
			if _, err := q.AgencyLibraryBump(ctx, store.AgencyLibraryBumpParams{ID: iid, Version: v, Name: item.Name, Fields: fields, UpdatedBy: &me.UserID, At: now}); err != nil {
				return err
			}
			// The copy it came from is now on the new version.
			if cp, err := q.AgencyLibraryCopyOf(ctx, store.AgencyLibraryCopyOfParams{TargetKind: targetKind(in.Kind), TargetID: id, WorkspaceID: ws.ID}); err == nil && cp.ItemID == iid {
				if _, err := q.AgencyLibraryUpdateCopy(ctx, store.AgencyLibraryUpdateCopyParams{ID: cp.ID, Version: v, At: now}); err != nil {
					return err
				}
			}
			out = AgencyLibraryAdded{ID: iid.String(), Version: int(v)}
			return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.library_item_changed", TargetType: "agency_library_item", TargetID: iid.String(), TargetName: item.Name,
				Before: map[string]any{"version": item.Version}, After: map[string]any{"version": v, "from": ws.Name, "note": strings.TrimSpace(in.Note)},
				Metadata: map[string]any{"permission": string(authz.AgencyLibrary), "kind": in.Kind}})
		}
		if taken, err := q.AgencyLibraryNameTaken(ctx, store.AgencyLibraryNameTakenParams{Kind: in.Kind, Name: name, ExcludeID: uuid.Nil}); err != nil {
			return err
		} else if taken {
			return api.Conflict(mLibraryNameTaken.Text(l, i18n.Args{"name": name}))
		}
		iid := uuid.Must(uuid.NewV7())
		if err := q.AgencyLibraryInsertItem(ctx, store.AgencyLibraryInsertItemParams{ID: iid, Kind: in.Kind, Name: name, Fields: fields, CreatedBy: &me.UserID, At: now}); err != nil {
			return err
		}
		if err := q.AgencyLibraryInsertVersion(ctx, store.AgencyLibraryInsertVersionParams{ItemID: iid, Version: 1, Content: raw, Summary: summary,
			Note: strings.TrimSpace(in.Note), SourceWorkspaceID: &ws.ID, CreatedBy: &me.UserID, At: now}); err != nil {
			return err
		}
		// The item it came from is its first copy.
		if err := q.AgencyLibraryInsertCopy(ctx, store.AgencyLibraryInsertCopyParams{ID: uuid.Must(uuid.NewV7()), ItemID: iid, Version: 1, WorkspaceID: ws.ID,
			TargetKind: targetKind(in.Kind), TargetID: id, CopiedBy: &me.UserID, At: now}); err != nil && !strings.Contains(err.Error(), "agency_library_copies_target_key") {
			return err
		}
		out = AgencyLibraryAdded{ID: iid.String(), Version: 1}
		wsID := ws.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.library_item_added", TargetType: "agency_library_item", TargetID: iid.String(), TargetName: name,
			After: map[string]any{"kind": in.Kind, "from": ws.Name}, Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
	return out, err
}

func targetKind(kind string) string {
	switch kind {
	case KindSection, KindTemplate:
		return "template"
	}
	return kind
}

// readItem reads a workspace's item as library content.
func (s *Service) readItem(ctx context.Context, wsID uuid.UUID, kind string, id uuid.UUID) (LibraryContent, string, error) {
	switch kind {
	case KindSection, KindTemplate:
		t, err := s.Q.AgencyGetTemplate(ctx, store.AgencyGetTemplateParams{ID: id, WorkspaceID: wsID})
		if err != nil {
			return LibraryContent{}, "", api.MapError(err, "Template")
		}
		if (kind == KindSection) != (t.Kind == "section") {
			return LibraryContent{}, "", api.NotFound("Template")
		}
		return LibraryContent{MJML: t.Mjml, HTML: t.Html, Thumb: t.Thumb}, t.Name, nil
	case KindAutomation:
		a, err := s.Q.AgencyGetAutomation(ctx, store.AgencyGetAutomationParams{ID: id, WorkspaceID: wsID})
		if err != nil {
			return LibraryContent{}, "", api.MapError(err, "Automation")
		}
		ws, err := s.Q.GetWorkspace(ctx, wsID)
		if err != nil {
			return LibraryContent{}, "", err
		}
		c, err := s.capture(ctx, s.Q, ws)
		if err != nil {
			return LibraryContent{}, "", err
		}
		for _, f := range c.Automations {
			if f.Name == a.Name {
				ta := f
				return LibraryContent{Automation: &ta}, a.Name, nil
			}
		}
		return LibraryContent{}, "", api.NotFound("Automation")
	case KindTransactional:
		t, err := s.Q.AgencyGetTransactionalTemplate(ctx, store.AgencyGetTransactionalTemplateParams{ID: id, WorkspaceID: wsID})
		if err != nil {
			return LibraryContent{}, "", api.MapError(err, "Transactional template")
		}
		var vars []transactional.TransactionalVariable
		_ = json.Unmarshal(t.Variables, &vars)
		return LibraryContent{Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml, HTML: t.Html, PlainText: t.PlainText, Variables: vars}, t.Name, nil
	}
	return LibraryContent{}, "", api.Invalid("body.kind", mKindMismatch.Text(i18n.LangFrom(ctx), nil))
}

// AgencyCopyInput copies a library item into a client.
type AgencyCopyInput struct {
	WorkspaceID string `json:"workspaceId" format:"uuid"`
}

// AgencyCopyResult is the copy made.
type AgencyCopyResult struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Kind      string             `json:"kind"`
	TargetID  string             `json:"targetId"`
	Name      string             `json:"name"`
}

// CopyToClient makes a copy of a library item in a client. It needs the
// edit permission of that kind of content there (templates:edit,
// automations:edit, transactional:edit).
func (s *Service) CopyToClient(ctx context.Context, itemID string, in AgencyCopyInput) (AgencyCopyResult, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyCopyResult{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyCopyResult{}, err
	}
	wsID, err := uuid.Parse(in.WorkspaceID)
	if err != nil {
		return AgencyCopyResult{}, api.NotFound("Client")
	}
	w, ok := set.find(wsID)
	if !ok {
		return AgencyCopyResult{}, api.NotFound("Client")
	}
	return s.copyItem(ctx, me, set, w, itemID)
}

func (s *Service) copyItem(ctx context.Context, me *Staff, set workspaceSet, w store.AgencyWorkspacesRow, itemID string) (AgencyCopyResult, error) {
	l := i18n.LangFrom(ctx)
	iid, err := uuid.Parse(itemID)
	if err != nil {
		return AgencyCopyResult{}, api.NotFound("Library item")
	}
	item, err := s.Q.AgencyLibraryItem(ctx, iid)
	if err != nil {
		return AgencyCopyResult{}, api.MapError(err, "Library item")
	}
	if item.Kind == KindSMS {
		return AgencyCopyResult{}, api.Conflict(mSMSFromComposer.Text(l, nil))
	}
	perm := editPermission(item.Kind)
	if !set.permsIn(w.ID).Has(perm) {
		p := api.Err(http.StatusForbidden, api.CodePermissionDenied, mCopyNeeds.Text(l, i18n.Args{"permission": string(perm), "workspace": w.Name}))
		p.Permission = string(perm)
		return AgencyCopyResult{}, p
	}
	v, err := s.Q.AgencyLibraryVersion(ctx, store.AgencyLibraryVersionParams{ItemID: iid, Version: item.Version})
	if err != nil {
		return AgencyCopyResult{}, err
	}
	c := parseLibraryContent(v.Content)
	ws, err := s.Q.GetWorkspace(ctx, w.ID)
	if err != nil {
		return AgencyCopyResult{}, err
	}
	now := s.now()
	out := AgencyCopyResult{Workspace: wsRef(w), Kind: item.Kind}
	if item.Kind == KindTransactional {
		d := s.deps()
		if d.Transactional == nil {
			return out, api.Err(http.StatusServiceUnavailable, api.CodeUnavailable, "Transactional templates aren't available on this server.")
		}
		p := s.principalIn(me, set, w.ID)
		t, err := d.Transactional.CreateTemplate(ctx, &ws, p, transactional.TransactionalTemplateCreate{Name: item.Name, Subject: c.Subject, Preheader: c.Preheader,
			EditorMode: "visual", MJML: c.MJML, HTML: c.HTML, PlainText: c.PlainText, Variables: c.Variables,
			Origin: map[string]any{"source": "agency_library", "item": iid.String(), "version": item.Version}})
		if err != nil {
			return out, err
		}
		tid := uuid.MustParse(t.ID)
		out.TargetID, out.Name = t.ID, t.Name
		return out, db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			return s.recordCopy(ctx, store.New(tx), item, w, "transactional", tid, me, now)
		})
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		switch item.Kind {
		case KindSection, KindTemplate:
			kind := item.Kind
			taken := func(n string) (bool, error) {
				return q.EmailTemplateNameTaken(ctx, store.EmailTemplateNameTakenParams{WorkspaceID: w.ID, Kind: kind, Name: n, ID: uuid.Nil})
			}
			name := item.Name
			if clash, err := taken(name); err != nil {
				return err
			} else if clash {
				if name, err = renamedLibraryName(item.Name, taken); err != nil {
					return err
				}
			}
			id, err := insertTemplate(ctx, q, w.ID, kind, name, TplItem{Name: name, MJML: c.MJML, HTML: c.HTML, Thumb: c.Thumb}, &me.UserID, now)
			if err != nil {
				return err
			}
			out.TargetID, out.Name = id.String(), name
			return s.recordCopy(ctx, q, item, w, "template", id, me, now)
		case KindAutomation:
			if c.Automation == nil {
				return api.Conflict(mEmptyItem.Text(l, nil))
			}
			taken := func(n string) (bool, error) {
				return q.AgencyAutomationNameTaken(ctx, store.AgencyAutomationNameTakenParams{WorkspaceID: w.ID, Name: n})
			}
			name := item.Name
			if clash, err := taken(name); err != nil {
				return err
			} else if clash {
				if name, err = renamedLibraryName(item.Name, taken); err != nil {
					return err
				}
			}
			lists, err := q.AgencyCaptureLists(ctx, w.ID)
			if err != nil {
				return err
			}
			listIDs := map[string]string{}
			for _, li := range lists {
				listIDs[li.Name] = li.ID.String()
			}
			id, err := s.insertAutomation(ctx, q, w.ID, name, *c.Automation, listIDs, map[string]string{}, &me.UserID, "copied from the agency library")
			if err != nil {
				return err
			}
			out.TargetID, out.Name = id.String(), name
			return s.recordCopy(ctx, q, item, w, "automation", id, me, now)
		}
		return api.Conflict(mEmptyItem.Text(l, nil))
	})
	return out, err
}

// renamedLibraryName is a copy's name when the client has one already:
// "Holiday hero (snow) (library)".
func renamedLibraryName(name string, taken func(string) (bool, error)) (string, error) {
	cand := name + " (library)"
	for i := 2; i < 100; i++ {
		t, err := taken(cand)
		if err != nil {
			return "", err
		}
		if !t {
			return cand, nil
		}
		cand = name + " (library " + itoa(i) + ")"
	}
	return cand, nil
}

func (s *Service) recordCopy(ctx context.Context, q *store.Queries, item store.AgencyLibraryItemRow, w store.AgencyWorkspacesRow, target string, targetID uuid.UUID, me *Staff, now time.Time) error {
	if err := q.AgencyLibraryInsertCopy(ctx, store.AgencyLibraryInsertCopyParams{ID: uuid.Must(uuid.NewV7()), ItemID: item.ID, Version: item.Version, WorkspaceID: w.ID,
		TargetKind: target, TargetID: targetID, CopiedBy: &me.UserID, At: now}); err != nil {
		return err
	}
	wsID := w.ID
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.library_item_copied", TargetType: target, TargetID: targetID.String(), TargetName: item.Name,
		After: map[string]any{"item": item.Name, "version": item.Version, "kind": item.Kind}, Metadata: map[string]any{"permission": string(editPermission(item.Kind))}})
}

// UpdateCopy brings a client's copy to the library's latest version. It
// replaces the copy's content; a flow can't be replaced in place.
func (s *Service) UpdateCopy(ctx context.Context, copyID string) error {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	cid, err := uuid.Parse(copyID)
	if err != nil {
		return api.NotFound("Copy")
	}
	cp, err := s.Q.AgencyLibraryCopy(ctx, cid)
	if err != nil {
		return api.MapError(err, "Copy")
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return err
	}
	w, ok := set.find(cp.WorkspaceID)
	if !ok {
		return api.NotFound("Copy")
	}
	item, err := s.Q.AgencyLibraryItem(ctx, cp.ItemID)
	if err != nil {
		return api.MapError(err, "Library item")
	}
	if ok, reason := s.canUpdateCopy(l, item.Kind, set.permsIn(w.ID)); !ok {
		return api.Conflict(reason)
	}
	v, err := s.Q.AgencyLibraryVersion(ctx, store.AgencyLibraryVersionParams{ItemID: item.ID, Version: item.Version})
	if err != nil {
		return err
	}
	c := parseLibraryContent(v.Content)
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		switch item.Kind {
		case KindSection, KindTemplate:
			thumb := c.Thumb
			if len(thumb) == 0 {
				thumb, _ = json.Marshal(mjml.Thumb(c.MJML))
			}
			html := c.HTML
			if item.Kind == KindSection {
				html = ""
			}
			if n, err := q.AgencyUpdateTemplateContent(ctx, store.AgencyUpdateTemplateContentParams{ID: cp.TargetID, WorkspaceID: w.ID, Mjml: c.MJML, Html: html,
				Thumb: thumb, SizeBytes: int32(len(html)), UpdatedBy: &me.UserID, At: now}); err != nil {
				return err
			} else if n == 0 {
				return api.Conflict(mCopyGone.Text(l, i18n.Args{"workspace": w.Name}))
			}
		case KindTransactional:
			if _, err := tx.Exec(ctx, `UPDATE transactional_templates SET subject = $3, preheader = $4, mjml = $5, html = $6, plain_text = $7,
				draft_revision = draft_revision + 1, draft_saved_at = $8, draft_saved_by = $9, updated_at = $8
				WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, cp.TargetID, w.ID, c.Subject, c.Preheader, c.MJML, c.HTML, c.PlainText, now, me.UserID); err != nil {
				return err
			}
		}
		if _, err := q.AgencyLibraryUpdateCopy(ctx, store.AgencyLibraryUpdateCopyParams{ID: cid, Version: item.Version, At: now}); err != nil {
			return err
		}
		wsID := w.ID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "agency.library_copy_updated", TargetType: cp.TargetKind, TargetID: cp.TargetID.String(), TargetName: item.Name,
			Before: map[string]any{"version": cp.Version}, After: map[string]any{"version": item.Version}, Metadata: map[string]any{"permission": string(editPermission(item.Kind))}})
	})
}

// AgencyLibraryEdit renames an item or publishes a client's copy as its
// next version.
type AgencyLibraryEdit struct {
	Name string `json:"name" minLength:"1" maxLength:"120"`
	// FromCopy publishes that copy's current content as a new version.
	FromCopy string `json:"fromCopy,omitempty" format:"uuid"`
	Note     string `json:"note,omitempty" maxLength:"200"`
}

// EditLibraryItem renames an item, or makes a copy's content its next version.
func (s *Service) EditLibraryItem(ctx context.Context, itemID string, in AgencyLibraryEdit) error {
	me, _, err := s.require(ctx, authz.AgencyLibrary)
	if err != nil {
		return err
	}
	l := i18n.LangFrom(ctx)
	iid, err := uuid.Parse(itemID)
	if err != nil {
		return api.NotFound("Library item")
	}
	item, err := s.Q.AgencyLibraryItem(ctx, iid)
	if err != nil {
		return api.MapError(err, "Library item")
	}
	name := strings.TrimSpace(in.Name)
	if taken, err := s.Q.AgencyLibraryNameTaken(ctx, store.AgencyLibraryNameTakenParams{Kind: item.Kind, Name: name, ExcludeID: iid}); err != nil {
		return err
	} else if taken {
		return api.Invalid("body.name", mLibraryNameTaken.Text(l, i18n.Args{"name": name}))
	}
	if in.FromCopy != "" {
		cid, err := uuid.Parse(in.FromCopy)
		if err != nil {
			return api.NotFound("Copy")
		}
		cp, err := s.Q.AgencyLibraryCopy(ctx, cid)
		if err != nil || cp.ItemID != iid {
			return api.NotFound("Copy")
		}
		ws, err := s.Q.GetWorkspace(ctx, cp.WorkspaceID)
		if err != nil {
			return err
		}
		// Publishing a copy as the next version: through the workspace,
		// with the content read from it.
		content, _, err := s.readItem(ctx, ws.ID, item.Kind, cp.TargetID)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(content)
		fields, _ := json.Marshal(libraryFields(content.MJML, content.Subject, content.Body))
		now := s.now()
		v := item.Version + 1
		return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.AgencyLibraryInsertVersion(ctx, store.AgencyLibraryInsertVersionParams{ItemID: iid, Version: v, Content: raw, Summary: summaryLines(item.Kind, content),
				Note: strings.TrimSpace(in.Note), SourceWorkspaceID: &ws.ID, CreatedBy: &me.UserID, At: now}); err != nil {
				return err
			}
			if _, err := q.AgencyLibraryBump(ctx, store.AgencyLibraryBumpParams{ID: iid, Version: v, Name: name, Fields: fields, UpdatedBy: &me.UserID, At: now}); err != nil {
				return err
			}
			if _, err := q.AgencyLibraryUpdateCopy(ctx, store.AgencyLibraryUpdateCopyParams{ID: cid, Version: v, At: now}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.library_item_changed", TargetType: "agency_library_item", TargetID: iid.String(), TargetName: name,
				Before: map[string]any{"version": item.Version, "name": item.Name}, After: map[string]any{"version": v, "name": name, "from": ws.Name, "note": strings.TrimSpace(in.Note)},
				Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
		})
	}
	if name == item.Name {
		return nil
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyLibraryBump(ctx, store.AgencyLibraryBumpParams{ID: iid, Version: item.Version, Name: name, Fields: item.Fields, UpdatedBy: &me.UserID, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.library_item_changed", TargetType: "agency_library_item", TargetID: iid.String(), TargetName: name,
			Before: map[string]any{"name": item.Name}, After: map[string]any{"name": name}, Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
}

// RemoveLibraryItem takes an item out of the library. Copies in clients
// stay: they were never linked.
func (s *Service) RemoveLibraryItem(ctx context.Context, itemID string) error {
	if _, _, err := s.require(ctx, authz.AgencyLibrary); err != nil {
		return err
	}
	iid, err := uuid.Parse(itemID)
	if err != nil {
		return api.NotFound("Library item")
	}
	item, err := s.Q.AgencyLibraryItem(ctx, iid)
	if err != nil {
		return api.MapError(err, "Library item")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AgencyLibraryDelete(ctx, iid); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "agency.library_item_removed", TargetType: "agency_library_item", TargetID: iid.String(), TargetName: item.Name,
			Before: map[string]any{"kind": item.Kind, "version": item.Version}, Metadata: map[string]any{"permission": string(authz.AgencyLibrary)}})
	})
}
