package imports

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Phone numbers and SMS consent in imports (SMS, design expansion phase 8,
// brief §3.3): a column can map to the phone, and SMS consent comes in only
// with its evidence columns, its date, source and text. Without them the
// number is imported and SMS consent stays Not subscribed. Imports never
// text anyone: their consent is old news to the person.

// The phone and SMS consent targets of Map fields.
const (
	TargetPhone            = "phone"
	TargetSmsConsent       = "sms_consent"
	TargetSmsConsentAt     = "sms_consent_at"
	TargetSmsConsentSource = "sms_consent_source"
	TargetSmsConsentText   = "sms_consent_text"
)

func init() {
	builtinTargets = append(builtinTargets,
		struct{ key, label, typ string }{TargetPhone, "phone (mobile number)", "phone"},
		struct{ key, label, typ string }{TargetSmsConsent, "SMS consent (yes or no)", "sms consent"},
		struct{ key, label, typ string }{TargetSmsConsentAt, "SMS consent evidence: date", "sms consent"},
		struct{ key, label, typ string }{TargetSmsConsentSource, "SMS consent evidence: source", "sms consent"},
		struct{ key, label, typ string }{TargetSmsConsentText, "SMS consent evidence: text", "sms consent"},
	)
	for k, v := range map[string]string{
		"smsconsent": TargetSmsConsent, "smsoptin": TargetSmsConsent, "textconsent": TargetSmsConsent,
		"smsconsentdate": TargetSmsConsentAt, "smsconsentat": TargetSmsConsentAt, "smsoptindate": TargetSmsConsentAt,
		"smsconsentsource": TargetSmsConsentSource, "smsoptinsource": TargetSmsConsentSource,
		"smsconsenttext": TargetSmsConsentText, "smsoptintext": TargetSmsConsentText,
	} {
		aliases[k] = v
	}
}

// Texts records imported numbers and SMS consent (internal/texting, which
// the app wires in); nil imports neither.
type Texts interface {
	// Import records a row's number on the contact and, with its
	// evidence, SMS consent on the list, in the run's transaction.
	Import(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID, listID uuid.UUID, row TextRow) (TextOutcome, error)
	// DefaultCountry is the country numbers without a country code are
	// read in when the row has no country column: the workspace's first
	// routed country, or "".
	DefaultCountry(ctx context.Context, ws *store.Workspace) string
}

// TextRow is a row's number and SMS consent columns.
type TextRow struct {
	Phone   string
	Country string
	// Consent is the SMS consent column; At, Source and Text its evidence.
	Consent bool
	At      *time.Time
	Source  string
	Text    string
	// ImportID and File name the import, kept with the consent record.
	ImportID uuid.UUID
	File     string
	// Overwrite replaces a number the contact has (the conflict rule).
	Overwrite bool
}

// TextOutcome is what a row's texts came to.
type TextOutcome string

// Text outcomes, counted in ImportStats.Texts.
const (
	TextNumber     TextOutcome = "number"        // the number, no SMS consent asked for
	TextConsent    TextOutcome = "consent"       // the number and SMS consent with its evidence
	TextNoEvidence TextOutcome = "no_evidence"   // SMS consent without its evidence: Not subscribed
	TextInvalid    TextOutcome = "invalid_phone" // not a phone number
)

// SMSEvidenceMapped reports whether SMS consent can come in: the consent
// column and its date, source and text are all mapped.
func (fm FieldMap) SMSEvidenceMapped() (consent, evidence bool) {
	got := map[string]bool{}
	for _, f := range fm.Fields {
		got[f.Target] = true
	}
	return got[TargetSmsConsent], got[TargetSmsConsent] && got[TargetSmsConsentAt] && got[TargetSmsConsentSource] && got[TargetSmsConsentText]
}

// ImportSmsConsent is the Consent step's SMS line: SMS consent comes in
// only with its date, source and text.
type ImportSmsConsent struct {
	// ConsentMapped: a column maps to SMS consent; EvidenceMapped: its
	// date, source and text columns too.
	ConsentMapped  bool `json:"consentMapped"`
	EvidenceMapped bool `json:"evidenceMapped"`
	// Missing are the evidence columns still to map: date, source, text.
	Missing []string `json:"missing"`
}

// smsConsentOf is the SMS line of a mapping; nil when no column maps to
// the phone.
func smsConsentOf(fm FieldMap) *ImportSmsConsent {
	got := map[string]bool{}
	for _, f := range fm.Fields {
		got[f.Target] = true
	}
	if !got[TargetPhone] {
		return nil
	}
	out := &ImportSmsConsent{Missing: []string{}}
	out.ConsentMapped, out.EvidenceMapped = fm.SMSEvidenceMapped()
	for _, m := range []struct{ target, name string }{{TargetSmsConsentAt, "date"}, {TargetSmsConsentSource, "source"}, {TargetSmsConsentText, "text"}} {
		if !got[m.target] {
			out.Missing = append(out.Missing, m.name)
		}
	}
	return out
}

// textRow reads a row's number and SMS consent columns; false when the row
// has no number.
func textRow(r mappedRow, fm FieldMap) (TextRow, bool) {
	var tr TextRow
	for _, f := range fm.Fields {
		v := strings.TrimSpace(r.Data[f.Source])
		switch f.Target {
		case TargetPhone:
			tr.Phone = v
		case "country":
			tr.Country = strings.ToUpper(v)
		case TargetSmsConsent:
			tr.Consent = truthy(v)
		case TargetSmsConsentAt:
			tr.At = parseTime(v)
		case TargetSmsConsentSource:
			tr.Source = v
		case TargetSmsConsentText:
			tr.Text = v
		}
	}
	return tr, tr.Phone != ""
}

// truthy reads a yes/no column: yes, y, true, 1, x, subscribed, opted in.
func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "y", "true", "1", "x", "subscribed", "opted_in", "opted in", "optin", "opt-in":
		return true
	}
	return false
}

// importTexts records a row's number and SMS consent, and counts what
// came of them.
func (s *Service) importTexts(ctx context.Context, tx pgx.Tx, st *runState, contact, list uuid.UUID, r mappedRow) error {
	if s.Texts == nil {
		return nil
	}
	tr, ok := textRow(r, st.fm)
	if !ok {
		return nil
	}
	if tr.Country == "" {
		if st.textCountry == nil {
			c := s.Texts.DefaultCountry(ctx, &st.ws)
			st.textCountry = &c
		}
		tr.Country = *st.textCountry
	}
	tr.ImportID, tr.File = st.imp.ID, firstNonEmpty(st.imp.FileName, SourceName(st.imp.Source))
	tr.Overwrite = st.opts.OnConflict == "overwrite"
	out, err := s.Texts.Import(ctx, tx, &st.ws, contact, list, tr)
	if err != nil || out == "" {
		return err
	}
	if st.stats.Texts == nil {
		st.stats.Texts = map[string]int64{}
	}
	st.stats.Texts[string(out)]++
	return nil
}
