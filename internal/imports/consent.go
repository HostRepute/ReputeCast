package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Consent handling of each source status.
const (
	HandleConfirmed         = "confirmed"
	HandlePending           = "pending"
	HandleSkip              = "skip"
	HandleSuppressUnsub     = "suppress_unsubscribe"
	HandleSuppressBounce    = "suppress_bounce"
	HandleSuppressComplaint = "suppress_complaint"
)

// ImportConsentMap is imports.consent_map: what each source status becomes.
// Unsubscribed, cleaned and complained are locked to suppressions.
type ImportConsentMap struct {
	Subscribed    string `json:"subscribed" enum:"confirmed,pending,skip"`
	Pending       string `json:"pending" enum:"pending,skip"`
	Transactional string `json:"transactional" enum:"skip,pending"`
}

func parseConsentMap(raw json.RawMessage, opts ImportOptions) ImportConsentMap {
	m := ImportConsentMap{Subscribed: HandleConfirmed, Pending: HandlePending, Transactional: HandleSkip}
	if opts.Evidence == "repermission" {
		m.Subscribed = HandlePending
	}
	_ = json.Unmarshal(raw, &m)
	if m.Subscribed == "" {
		m.Subscribed = HandleConfirmed
	}
	if m.Pending == "" {
		m.Pending = HandlePending
	}
	if m.Transactional == "" {
		m.Transactional = HandleSkip
	}
	return m
}

// handling returns what happens to a row with this source status.
func (m ImportConsentMap) handling(status string) string {
	switch status {
	case StatusSubscribed:
		return m.Subscribed
	case StatusPending:
		return m.Pending
	case StatusTransactional:
		return m.Transactional
	case StatusCleaned:
		return HandleSuppressBounce
	case StatusComplained:
		return HandleSuppressComplaint
	}
	return HandleSuppressUnsub
}

// ImportConsentStatus is one row of the consent table.
type ImportConsentStatus struct {
	Status   string        `json:"status"`
	Label    string        `json:"label" doc:"The source's own word: subscribed, cleaned, archived with \"abuse\" complaint"`
	Note     string        `json:"note,omitempty"`
	Tone     string        `json:"tone" enum:"ok,warn,bad,neutral"`
	Contacts int64         `json:"contacts"`
	Handling string        `json:"handling"`
	Options  []ImportFieldOption `json:"options"`
	Locked   bool          `json:"locked"`
	Effect   string        `json:"effect"`
}

// ImportConsentMapping is the "Consent" step.
type ImportConsentMapping struct {
	Statuses      []ImportConsentStatus `json:"statuses"`
	SuppressCount int64           `json:"suppressCount"`
	Evidence      string          `json:"evidence" enum:"trust,repermission"`
	// EvidenceSource names what the source's opt-in proof is ("OPTIN_TIME and OPTIN_IP").
	EvidenceSource   string `json:"evidenceSource"`
	DoubleOptIn      bool   `json:"doubleOptIn"`
	PullSuppressions bool   `json:"pullSuppressions"`
	RecordHistory    bool   `json:"recordHistory"`
	// HistoryExample is how the consent history entry will read.
	HistoryExample string `json:"historyExample"`
	// Providers are the providers new suppressions are pushed to.
	Providers []string `json:"providers"`
	// InSync is how many addresses are already shared with the providers.
	InSync int64 `json:"inSync"`
}

var subscribedOptions = []ImportFieldOption{
	{HandleConfirmed, "Subscribed · confirmed, keep opt-in date and IP as evidence"},
	{HandlePending, "Subscribed · pending, send a re-permission email"},
	{HandleSkip, "Skip"},
}

// Consent returns the consent step: counts per source status and the
// handling of each.
func (s *Service) Consent(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportConsentMapping, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportConsentMapping{}, err
	}
	opts := parseOptions(imp.Options)
	m := parseConsentMap(imp.ConsentMap, opts)
	counts := map[string]int64{}
	rows, err := s.Q.ImportStatusCounts(ctx, imp.ID)
	if err != nil {
		return ImportConsentMapping{}, err
	}
	for _, r := range rows {
		counts[r.Status] += r.N
	}
	conn := parseConnection(imp.Connection)
	name := SourceName(imp.Source)
	providers, inSync, err := s.providerNames(ctx, ws)
	if err != nil {
		return ImportConsentMapping{}, err
	}
	out := ImportConsentMapping{Statuses: []ImportConsentStatus{}, Evidence: opts.Evidence, PullSuppressions: opts.PullSuppressions, RecordHistory: opts.RecordHistory,
		Providers: providers, InSync: inSync}
	if m.Subscribed == HandlePending {
		out.Evidence = "repermission"
	} else {
		out.Evidence = "trust"
	}
	for _, l := range conn.Lists {
		if l.DoubleOptIn {
			out.DoubleOptIn = true
		}
	}
	evidence := ""
	for _, f := range s.effectiveMap(imp, map[string]store.CustomField{}).Fields {
		if f.Target == TargetEvidence {
			evidence = f.Source
			if f.Source == "OPTIN" {
				evidence = "OPTIN_TIME and OPTIN_IP"
			}
		}
	}
	out.EvidenceSource = evidence
	statuses := []string{StatusSubscribed, StatusUnsubscribed, StatusCleaned, StatusPending, StatusTransactional, StatusComplained}
	push := pushText(providers)
	for _, st := range statuses {
		n := counts[st]
		// API importers show every status their tool has; files only what they contain.
		if n == 0 && isFile(imp.Source) && st != StatusSubscribed {
			continue
		}
		row := ImportConsentStatus{Status: st, Label: st, Contacts: n, Handling: m.handling(st)}
		switch st {
		case StatusSubscribed:
			row.Tone, row.Options = "ok", subscribedOptions
			if out.DoubleOptIn {
				since := ""
				for _, l := range conn.Lists {
					if l.Since != "" {
						since = " since " + l.Since
					}
				}
				row.Note = "double opt-in was on in this audience" + since
			}
		case StatusUnsubscribed:
			row.Tone, row.Locked = "bad", true
			row.Options = []ImportFieldOption{{HandleSuppressUnsub, "Suppressed · unsubscribed (locked)"}}
			row.Effect = "Broadcasts and Automations blocked; System mail allowed. Can't be changed by an import."
		case StatusCleaned:
			row.Tone, row.Locked = "bad", true
			row.Note = "hard bounced in " + name
			row.Options = []ImportFieldOption{{HandleSuppressBounce, "Suppressed · hard bounce (locked)"}}
			row.Effect = "All streams blocked" + push
		case StatusPending:
			row.Tone, row.Note = "warn", "never confirmed double opt-in"
			row.Options = []ImportFieldOption{{HandlePending, "Subscribed · pending, send a re-permission email"}, {HandleSkip, "Skip"}}
		case StatusTransactional:
			row.Tone = "neutral"
			row.Options = []ImportFieldOption{{HandleSkip, "Skip"}, {HandlePending, "Subscribed · pending"}}
		case StatusComplained:
			row.Tone, row.Locked = "bad", true
			if imp.Source == "mailchimp" {
				row.Label = `archived with "abuse" complaint`
			}
			row.Options = []ImportFieldOption{{HandleSuppressComplaint, "Suppressed · complaint (locked)"}}
			row.Effect = "All streams blocked"
		}
		if row.Effect == "" {
			switch row.Handling {
			case HandleConfirmed:
				row.Effect = "Gets Broadcasts and Automations right away"
			case HandlePending:
				row.Effect = "No marketing mail until they confirm"
			case HandleSkip:
				row.Effect = "Not imported"
			}
		}
		if row.Locked {
			out.SuppressCount += n
		}
		out.Statuses = append(out.Statuses, row)
	}
	out.HistoryExample, err = s.historyExample(ctx, ws, imp, m)
	return out, err
}

func pushText(providers []string) string {
	if len(providers) == 0 {
		return ""
	}
	return ", pushed to " + joinAnd(providers)
}

func joinAnd(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// providerNames lists the workspace's providers (short names) and how many
// suppressed addresses they already share.
func (s *Service) providerNames(ctx context.Context, ws *store.Workspace) ([]string, int64, error) {
	rows, err := s.Q.SuppressionSyncStates(ctx, ws.ID)
	if err != nil {
		return nil, 0, err
	}
	out := []string{}
	var pulled int64
	for _, r := range rows {
		out = append(out, shortProvider(r.Name, r.Adapter))
		pulled += r.Pulled
	}
	return out, pulled, nil
}

// historyExample is how the consent history entry of the first row reads:
// "Imported from Mailchimp by Linh Nguyen, 12 Oct 2026, status subscribed,
// evidence OPTIN_TIME 2025-03-02".
func (s *Service) historyExample(ctx context.Context, ws *store.Workspace, imp store.Import, m ImportConsentMap) (string, error) {
	rows, err := s.Q.ImportRowsAfter(ctx, store.ImportRowsAfterParams{ImportID: imp.ID, After: 0, Lim: 50})
	if err != nil {
		return "", err
	}
	who := ""
	if u := api.UserFrom(ctx); u != nil {
		who = " by " + u.Name
	}
	day := s.now().In(location(ws)).Format("2 Jan 2006")
	for _, r := range rows {
		if r.Status != StatusSubscribed {
			continue
		}
		ex := fmt.Sprintf("Imported from %s%s, %s, status %s", SourceName(imp.Source), who, day, r.Status)
		var e Engagement
		_ = json.Unmarshal(r.Engagement, &e)
		if e.OptinAt != nil && m.Subscribed == HandleConfirmed {
			ex += ", evidence OPTIN_TIME " + e.OptinAt.UTC().Format("2006-01-02")
		}
		return ex, nil
	}
	return fmt.Sprintf("Imported from %s%s, %s", SourceName(imp.Source), who, day), nil
}

// ImportSaveConsentInput is the consent step's choices.
type ImportSaveConsentInput struct {
	Map              ImportConsentMap `json:"map"`
	Evidence         string     `json:"evidence" enum:"trust,repermission"`
	PullSuppressions bool       `json:"pullSuppressions"`
	RecordHistory    bool       `json:"recordHistory"`
}

// SaveConsent stores the consent mapping and moves on to the preview.
func (s *Service) SaveConsent(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ImportSaveConsentInput) (ImportConsentMapping, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportConsentMapping{}, err
	}
	if err := editable(imp); err != nil {
		return ImportConsentMapping{}, err
	}
	m := in.Map
	if in.Evidence == "repermission" && m.Subscribed == HandleConfirmed {
		m.Subscribed = HandlePending
	}
	opts := parseOptions(imp.Options)
	opts.Evidence, opts.PullSuppressions, opts.RecordHistory = in.Evidence, in.PullSuppressions, in.RecordHistory
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetImportConsentMap(ctx, store.SetImportConsentMapParams{ID: imp.ID, ConsentMap: mustJSON(m)}); err != nil {
			return err
		}
		if err := q.SetImportOptions(ctx, store.SetImportOptionsParams{ID: imp.ID, Options: mustJSON(opts)}); err != nil {
			return err
		}
		if err := q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: "preview"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.map_consent", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: SourceName(imp.Source), After: in})
	})
	if err != nil {
		return ImportConsentMapping{}, err
	}
	return s.Consent(ctx, ws, id)
}
