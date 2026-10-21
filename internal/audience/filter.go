package audience

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/segments"
)

// ContactFilter narrows the contacts table. Saved views store it as JSON;
// list, export and bulk actions accept it.
type ContactFilter struct {
	Q       string `json:"q,omitempty" maxLength:"200" doc:"Email, name or tag"`
	List    string `json:"list,omitempty" doc:"List id"`
	Tier    string `json:"tier,omitempty" enum:"active,cooling,at_risk,sunset"`
	Mailbox string `json:"mailbox,omitempty" enum:"gmail,outlook,yahoo,icloud,other"`
	Consent string `json:"consent,omitempty" enum:"double,single,pending"`
	Source  string `json:"source,omitempty" maxLength:"80"`
	Tag     string `json:"tag,omitempty" maxLength:"80"`
	Segment string `json:"segment,omitempty" doc:"Segment id"`
	// JoinedAfter is a date (2026-09-01).
	JoinedAfter string `json:"joinedAfter,omitempty" maxLength:"10"`
	// Language is a workspace language's code (its regional codes match),
	// "none" for contacts with no language or "not_offered" for languages
	// the workspace doesn't offer.
	Language string `json:"language,omitempty" maxLength:"16"`
	// Score, band and growth source (scoregrowth.go).
	ScoreGrowthFilter
	// Check is the latest address check's result, or never checked.
	Check string `json:"check,omitempty" enum:"valid,invalid,risky,unknown,never"`
	// Members of the website (website.go).
	WebsiteFilter
	// Paid status (Earn, paid.go).
	PaidFilter
	// Push subscribed and WhatsApp consent (X22, pushwhatsapp.go).
	PushWhatsAppFilter
	// Buyers of digital products (products.go).
	ProductsFilter
	// Store customers and what they spent (storefilter.go).
	StoreFilter
}

// Empty reports whether the filter matches everyone.
func (f ContactFilter) Empty() bool { return f == ContactFilter{} }

// filterSQL is a WHERE clause over contacts c with its arguments.
type filterSQL struct {
	where string
	args  []any
}

func (f *filterSQL) add(cond string, vals ...any) {
	for _, v := range vals {
		f.args = append(f.args, v)
		cond = strings.Replace(cond, "?", fmt.Sprintf("$%d", len(f.args)), 1)
	}
	f.where += "\n  AND " + cond
}

// filterSQL builds the WHERE clause for a filter in workspace ws, limited
// to the caller's list scope ("Editor, Beta Program list only"). It
// resolves segments to contact ids.
func (s *Service) filterSQL(ctx context.Context, wsID uuid.UUID, f ContactFilter) (filterSQL, error) {
	out := filterSQL{where: "c.workspace_id = $1 AND c.erased_at IS NULL", args: []any{wsID}}
	if err := s.applyFilter(ctx, wsID, &out, f); err != nil {
		return out, err
	}
	if p := api.PrincipalFrom(ctx); p != nil && len(p.Scope.Lists) > 0 {
		out.add("EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = ANY (?))", p.Scope.Lists)
	}
	return out, nil
}

// applyFilter appends the conditions of f to out, numbering arguments after
// the ones already there.
func (s *Service) applyFilter(ctx context.Context, wsID uuid.UUID, out *filterSQL, f ContactFilter) error {
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + escapeLike(strings.ToLower(q)) + "%"
		out.add("(c.email LIKE ? OR lower(c.first_name || ' ' || c.last_name) LIKE ? OR ? = ANY (c.tags))", like, like, q)
	}
	if f.List != "" {
		id, err := uuid.Parse(f.List)
		if err != nil {
			return api.Invalid("list", "Pick a list from this workspace.")
		}
		out.add("EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = ? AND lm.status <> 'unsubscribed')", id)
	}
	if f.Tier != "" {
		out.add("c.tier = ?", f.Tier)
	}
	if f.Mailbox != "" {
		out.add("c.mailbox_provider = ?", f.Mailbox)
	}
	if f.Consent != "" {
		out.add("c.consent = ?", f.Consent)
	}
	switch f.Source {
	case "":
	case "partner":
		// Any partner (Grow): "partner:<link>".
		out.add("c.source LIKE ?", "partner:%")
	case "referral":
		out.add("(c.source = 'referral' OR c.source LIKE ?)", "referral:%")
	default:
		out.add("c.source = ?", f.Source)
	}
	if t := strings.TrimSpace(f.Tag); t != "" {
		out.add("? = ANY (c.tags)", t)
	}
	if f.JoinedAfter != "" {
		d, err := time.Parse("2006-01-02", f.JoinedAfter)
		if err != nil {
			return api.Invalid("joinedAfter", "Use a date like 2026-09-01.")
		}
		out.add("c.created_at >= ?", d)
	}
	switch f.Check {
	case "":
	case "never":
		out.add("NOT EXISTS (SELECT 1 FROM contact_address_checks ac WHERE ac.contact_id = c.id)")
	default:
		out.add("EXISTS (SELECT 1 FROM contact_address_checks ac WHERE ac.contact_id = c.id AND ac.result = ?)", f.Check)
	}
	if err := s.scoreGrowthFilter(ctx, wsID, out, f.ScoreGrowthFilter); err != nil {
		return err
	}
	if err := s.languageFilter(ctx, wsID, out, f.Language); err != nil {
		return err
	}
	websiteFilter(out, f.WebsiteFilter)
	pushWhatsAppFilter(out, f.PushWhatsAppFilter)
	if err := s.paidFilter(ctx, wsID, out, f.PaidFilter); err != nil {
		return err
	}
	productsFilter(out, f.ProductsFilter)
	if err := s.storeFilter(out, f.StoreFilter); err != nil {
		return err
	}
	if f.Segment != "" {
		id, err := uuid.Parse(f.Segment)
		if err != nil {
			return api.Invalid("segment", "Pick a segment from this workspace.")
		}
		ids, err := s.Segments.ContactIDs(ctx, wsID, id)
		if err != nil {
			return segments.Problem(err)
		}
		out.add("c.id = ANY (?)", ids)
	}
	return nil
}

// inScope reports whether a contact is visible to the caller's list scope.
func (s *Service) inScope(ctx context.Context, contactID uuid.UUID) (bool, error) {
	p := api.PrincipalFrom(ctx)
	if p == nil || len(p.Scope.Lists) == 0 {
		return true, nil
	}
	var ok bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM list_memberships WHERE contact_id = $1 AND list_id = ANY ($2))`, contactID, p.Scope.Lists).Scan(&ok)
	return ok, err
}

// scopeAllowsList reports whether the caller may act on a list.
func scopeAllowsList(ctx context.Context, id uuid.UUID) bool {
	p := api.PrincipalFrom(ctx)
	if p == nil {
		return true
	}
	return authz.Scope.AllowsList(p.Scope, id)
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// TierName is how a tier reads in the UI.
func TierName(t string) string {
	switch t {
	case "active":
		return "Active"
	case "cooling":
		return "Cooling"
	case "at_risk":
		return "At risk"
	case "sunset":
		return "Sunset"
	}
	return t
}

// ConsentName is how a consent state reads in the UI.
func ConsentName(c string) string {
	switch c {
	case "double":
		return "Double opt-in"
	case "single":
		return "Single opt-in"
	case "pending":
		return "Imported · pending"
	}
	return c
}

// MailboxLabel is "Outlook" or "Other · Google Workspace".
func MailboxLabel(provider, detail string) string {
	name := emailaddr.MailboxName(provider)
	if provider == emailaddr.Other && detail != "" {
		return name + " · " + detail
	}
	return name
}
