package imports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/store"
)

// Row classes: every staged row lands in exactly one.
const (
	classImport            = "import"
	classDuplicate         = "duplicate"          // an existing contact
	classAlreadySuppressed = "already_suppressed" // on the shared list already
	classSuppress          = "suppress"           // the source says unsubscribed, bounced or complained
	classInvalid           = "invalid"
	classSkipStatus        = "skip_status" // its status is mapped to Skip
	classRole              = "role"
	classDisposable        = "disposable"
	classTypo              = "typo"
	classStale             = "stale"
	classTrap              = "trap"
)

// ProblemNoMX is the invalid reason for domains that can't receive mail.
const ProblemNoMX = "no MX record"

// mappedRow is a staged row read through the field mapping.
type mappedRow struct {
	RowNo  int32
	Raw    string // the email as it was in the source
	Email  string // normalized
	Status string
	Data   map[string]string
	Tags   []string
	Eng    Engagement
	List   string
}

// existingContact is what ReputeCast already has for an address.
type existingContact struct {
	ID         uuid.UUID
	Email      string
	FirstName  string
	LastName   string
	Company    string
	Country    string
	Timezone   string
	Language   string
	Consent    string
	Tier       string
	Tags       []string
	Attributes map[string]any
	Source     string
	CreatedAt  time.Time
	LastOpenAt *time.Time
	Lists      []string
}

type existingSuppression struct {
	Reason string
	At     time.Time
}

// verdict is a row's class with details.
type verdict struct {
	Class   string
	Problem string // invalid
	Fixed   string // typo: the corrected address
	Contact *existingContact
	Supp    *existingSuppression
	// ByService: a verification service put the row in its class; Agree:
	// it found what ReputeCast found (X14).
	ByService bool
	Agree     bool
}

// batch is a page of staged rows with the lookups they need.
type batch struct {
	rows       []mappedRow
	contacts   map[string]*existingContact
	suppressed map[string]*existingSuppression
	mx         map[string]MXResult
}

// fieldSources finds which source fields hold the email and the status.
func fieldSources(fm FieldMap) (email, status string) {
	for _, f := range fm.Fields {
		switch f.Target {
		case TargetEmail:
			email = f.Source
		case TargetStatus:
			status = f.Source
		}
	}
	return email, status
}

// loadBatch reads staged rows after row number `after` and looks up the
// contacts, suppressions and MX answers they need.
func (s *Service) loadBatch(ctx context.Context, ws uuid.UUID, imp store.Import, fm FieldMap, after int32, limit int32) (batch, error) {
	rows, err := s.Q.ImportRowsAfter(ctx, store.ImportRowsAfterParams{ImportID: imp.ID, After: after, Lim: limit})
	if err != nil {
		return batch{}, err
	}
	emailField, statusField := fieldSources(fm)
	b := batch{rows: make([]mappedRow, 0, len(rows)), contacts: map[string]*existingContact{}, suppressed: map[string]*existingSuppression{}}
	var emails []string
	domains := map[string]bool{}
	for _, r := range rows {
		m := mappedRow{RowNo: r.RowNo, Status: r.Status, Tags: r.Tags, List: r.ListRef}
		_ = json.Unmarshal(r.Data, &m.Data)
		if m.Data == nil {
			m.Data = map[string]string{}
		}
		_ = json.Unmarshal(r.Engagement, &m.Eng)
		m.Raw = r.Email
		if emailField != "" {
			if v, ok := m.Data[emailField]; ok {
				m.Raw = v
			}
		}
		if statusField != "" {
			m.Status = NormalizeStatus(m.Data[statusField])
		}
		if m.Tags == nil {
			m.Tags = []string{}
		}
		e, problem := emailaddr.Check(m.Raw)
		if problem == "" {
			m.Email = e
			emails = append(emails, e)
			domains[emailaddr.Domain(e)] = true
		}
		b.rows = append(b.rows, m)
	}
	if len(emails) == 0 {
		return b, nil
	}
	crows, err := s.Pool.Query(ctx, `SELECT c.id, c.email, c.first_name, c.last_name, c.company, c.country, c.timezone, c.language, c.consent, c.tier,
		c.tags, c.attributes, c.source, c.created_at, c.last_open_at,
		coalesce((SELECT array_agg(l.name ORDER BY l.created_at) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id
		          WHERE lm.contact_id = c.id AND lm.status <> 'unsubscribed' AND l.deleted_at IS NULL), '{}')
		FROM contacts c WHERE c.workspace_id = $1 AND c.email = ANY ($2) AND c.erased_at IS NULL`, ws, emails)
	if err != nil {
		return b, err
	}
	for crows.Next() {
		var c existingContact
		var attrs json.RawMessage
		if err := crows.Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName, &c.Company, &c.Country, &c.Timezone, &c.Language, &c.Consent, &c.Tier,
			&c.Tags, &attrs, &c.Source, &c.CreatedAt, &c.LastOpenAt, &c.Lists); err != nil {
			crows.Close()
			return b, err
		}
		_ = json.Unmarshal(attrs, &c.Attributes)
		if c.Attributes == nil {
			c.Attributes = map[string]any{}
		}
		b.contacts[c.Email] = &c
	}
	crows.Close()
	srows, err := s.Pool.Query(ctx, `SELECT email, reason, created_at FROM suppressions WHERE workspace_id = $1 AND email = ANY ($2)`, ws, emails)
	if err != nil {
		return b, err
	}
	for srows.Next() {
		var email string
		var sup existingSuppression
		if err := srows.Scan(&email, &sup.Reason, &sup.At); err != nil {
			srows.Close()
			return b, err
		}
		b.suppressed[email] = &sup
	}
	srows.Close()
	if s.MX != nil {
		list := make([]string, 0, len(domains))
		for d := range domains {
			list = append(list, d)
		}
		if b.mx, err = s.MX.Check(ctx, list); err != nil {
			return b, err
		}
	}
	return b, nil
}

// classify decides what happens to one row.
func classify(r mappedRow, b batch, cm ImportConsentMap, now time.Time) verdict {
	handling := cm.handling(r.Status)
	if r.Email == "" {
		_, problem := emailaddr.Check(r.Raw)
		return verdict{Class: classInvalid, Problem: problem}
	}
	switch handling {
	case HandleSuppressUnsub, HandleSuppressBounce, HandleSuppressComplaint:
		return verdict{Class: classSuppress}
	case HandleSkip:
		return verdict{Class: classSkipStatus}
	}
	if sup := b.suppressed[r.Email]; sup != nil {
		return verdict{Class: classAlreadySuppressed, Supp: sup, Contact: b.contacts[r.Email]}
	}
	if c := b.contacts[r.Email]; c != nil {
		return verdict{Class: classDuplicate, Contact: c}
	}
	domain := emailaddr.Domain(r.Email)
	switch {
	case emailaddr.TrapLike(r.Email):
		return verdict{Class: classTrap}
	case emailaddr.IsDisposable(domain):
		return verdict{Class: classDisposable}
	}
	if fixed, ok := emailaddr.TypoFix(domain); ok {
		return verdict{Class: classTypo, Fixed: emailaddr.Local(r.Email) + "@" + fixed}
	}
	if emailaddr.IsRole(r.Email) {
		return verdict{Class: classRole}
	}
	if mx, ok := b.mx[domain]; ok && !mx.HasMX {
		return verdict{Class: classInvalid, Problem: ProblemNoMX}
	}
	if stale(r.Eng, now) {
		return verdict{Class: classStale}
	}
	return verdict{Class: classImport}
}

// stale: never engaged and untouched for two years (Mailchimp member
// rating 1, no change since): most likely to be dead or recycled into a trap.
func stale(e Engagement, now time.Time) bool {
	if e.Rating == 0 || e.Rating > 1 {
		return false
	}
	cutoff := now.AddDate(-2, 0, 0)
	if e.LastOpen != nil && e.LastOpen.After(cutoff) {
		return false
	}
	if e.LastClick != nil && e.LastClick.After(cutoff) {
		return false
	}
	return e.Changed != nil && e.Changed.Before(cutoff)
}

// isRisky reports the risky classes.
func isRisky(class string) bool {
	switch class {
	case classRole, classDisposable, classTypo, classStale, classTrap, classCatchAll, classAbuse, classUnknown:
		return true
	}
	return false
}

// riskyAction is what the person chose for a risky class.
func riskyAction(o ImportRiskyActions, class string) string {
	switch class {
	case classRole:
		return o.Role
	case classDisposable:
		return o.Disposable
	case classTypo:
		return o.Typo
	case classStale:
		return o.Stale
	case classTrap:
		return o.Trap
	// A verification service's risky rows are imported, tagged, unless the
	// person skips them.
	case classCatchAll:
		return firstNonEmpty(o.CatchAll, "import")
	case classAbuse:
		return firstNonEmpty(o.Abuse, "import")
	case classUnknown:
		return firstNonEmpty(o.Unknown, "import")
	}
	return ""
}
