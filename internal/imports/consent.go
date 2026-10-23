package imports

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
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
	Status   string              `json:"status"`
	Label    string              `json:"label" doc:"The source's own word: subscribed, cleaned, archived with \"abuse\" complaint"`
	Note     string              `json:"note,omitempty"`
	Tone     string              `json:"tone" enum:"ok,warn,bad,neutral"`
	Contacts int64               `json:"contacts"`
	Handling string              `json:"handling"`
	Options  []ImportFieldOption `json:"options"`
	Locked   bool                `json:"locked"`
	Effect   string              `json:"effect"`
}

// ImportConsentMapping is the "Consent" step.
type ImportConsentMapping struct {
	Statuses      []ImportConsentStatus `json:"statuses"`
	SuppressCount int64                 `json:"suppressCount"`
	Evidence      string                `json:"evidence" enum:"trust,repermission"`
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
	// Sms is SMS consent and its evidence columns (phase 8), when a column
	// maps to the phone.
	Sms *ImportSmsConsent `json:"sms,omitempty"`
}

// Consent returns the consent step: counts per source status and the
// handling of each, in the request's language. The statuses keep the
// source's own words (subscribed, cleaned).
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
				evidence = say(ctx, mEvidenceOptin, nil)
			}
		}
	}
	out.EvidenceSource = evidence
	statuses := []string{StatusSubscribed, StatusUnsubscribed, StatusCleaned, StatusPending, StatusTransactional, StatusComplained}
	opt := func(value string, m i18n.Msg) ImportFieldOption { return ImportFieldOption{value, say(ctx, m, nil)} }
	for _, st := range statuses {
		n := counts[st]
		// API importers show every status their tool has; files only what they contain.
		if n == 0 && isFile(imp.Source) && st != StatusSubscribed {
			continue
		}
		row := ImportConsentStatus{Status: st, Label: st, Contacts: n, Handling: m.handling(st)}
		switch st {
		case StatusSubscribed:
			row.Tone = "ok"
			row.Options = []ImportFieldOption{opt(HandleConfirmed, mConsentConfirmed), opt(HandlePending, mConsentPending), opt(HandleSkip, mSkip)}
			if out.DoubleOptIn {
				since := ""
				for _, l := range conn.Lists {
					if l.Since != "" {
						since = l.Since
					}
				}
				row.Note = say(ctx, mNoteDoubleOptIn, nil)
				if since != "" {
					row.Note = say(ctx, mNoteDoubleOptInSince, i18n.Args{"year": since})
				}
			}
		case StatusUnsubscribed:
			row.Tone, row.Locked = "bad", true
			row.Options = []ImportFieldOption{opt(HandleSuppressUnsub, mConsentSuppressUnsub)}
			row.Effect = say(ctx, mEffectUnsub, nil)
		case StatusCleaned:
			row.Tone, row.Locked = "bad", true
			row.Note = say(ctx, mNoteHardBounced, i18n.Args{"source": name})
			row.Options = []ImportFieldOption{opt(HandleSuppressBounce, mConsentSuppressBounce)}
			row.Effect = say(ctx, mEffectBlocked, nil)
			if len(providers) > 0 {
				row.Effect = say(ctx, mEffectBlockedPushed, i18n.Args{"providers": providers})
			}
		case StatusPending:
			row.Tone, row.Note = "warn", say(ctx, mNotePending, nil)
			row.Options = []ImportFieldOption{opt(HandlePending, mConsentPending), opt(HandleSkip, mSkip)}
		case StatusTransactional:
			row.Tone = "neutral"
			row.Options = []ImportFieldOption{opt(HandleSkip, mSkip), opt(HandlePending, mConsentPendingShort)}
		case StatusComplained:
			row.Tone, row.Locked = "bad", true
			if imp.Source == "mailchimp" {
				row.Label = say(ctx, mLabelAbuse, nil)
			}
			row.Options = []ImportFieldOption{opt(HandleSuppressComplaint, mConsentSuppressCompl)}
			row.Effect = say(ctx, mEffectBlocked, nil)
		}
		if row.Effect == "" {
			switch row.Handling {
			case HandleConfirmed:
				row.Effect = say(ctx, mEffectConfirmed, nil)
			case HandlePending:
				row.Effect = say(ctx, mEffectPending, nil)
			case HandleSkip:
				row.Effect = say(ctx, mEffectSkip, nil)
			}
		}
		if row.Locked {
			out.SuppressCount += n
		}
		out.Statuses = append(out.Statuses, row)
	}
	out.HistoryExample, err = s.historyExample(ctx, ws, imp, m)
	out.Sms = smsConsentOf(parseFieldMap(imp.FieldMap))
	return out, err
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

// historyExample is how the consent history entry of the first row reads,
// in the request's language: "Imported from Mailchimp by Linh Nguyen, 12
// Oct 2026, status subscribed, evidence OPTIN_TIME 2025-03-02".
func (s *Service) historyExample(ctx context.Context, ws *store.Workspace, imp store.Import, m ImportConsentMap) (string, error) {
	rows, err := s.Q.ImportRowsAfter(ctx, store.ImportRowsAfterParams{ImportID: imp.ID, After: 0, Lim: 50})
	if err != nil {
		return "", err
	}
	args := i18n.Args{"source": SourceName(imp.Source), "by": "no", "name": ""}
	if u := api.UserFrom(ctx); u != nil {
		args["by"], args["name"] = "yes", u.Name
	}
	args["day"] = dayIn(i18n.LangFrom(ctx), s.now().In(location(ws)))
	for _, r := range rows {
		if r.Status != StatusSubscribed {
			continue
		}
		args["status"] = r.Status
		var e Engagement
		_ = json.Unmarshal(r.Engagement, &e)
		if e.OptinAt != nil && m.Subscribed == HandleConfirmed {
			args["evidence"] = e.OptinAt.UTC().Format("2006-01-02")
			return say(ctx, mHistoryEvidence, args), nil
		}
		return say(ctx, mHistory, args), nil
	}
	return say(ctx, mHistoryNoRow, args), nil
}

// ImportSaveConsentInput is the consent step's choices.
type ImportSaveConsentInput struct {
	Map              ImportConsentMap `json:"map"`
	Evidence         string           `json:"evidence" enum:"trust,repermission"`
	PullSuppressions bool             `json:"pullSuppressions"`
	RecordHistory    bool             `json:"recordHistory"`
}

// SaveConsent stores the consent mapping and moves on to the preview.
func (s *Service) SaveConsent(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ImportSaveConsentInput) (ImportConsentMapping, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportConsentMapping{}, err
	}
	if err := editable(ctx, imp); err != nil {
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
