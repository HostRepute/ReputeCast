package audience

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
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
	// JoinedAfter and JoinedBefore are dates (2026-09-01) in the workspace's
	// time zone, both included.
	JoinedAfter  string `json:"joinedAfter,omitempty" maxLength:"10"`
	JoinedBefore string `json:"joinedBefore,omitempty" maxLength:"10"`
	// Import is an import's id: the contacts it brought in.
	Import string `json:"import,omitempty" maxLength:"40"`
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
	// The phone and SMS consent (channels.go).
	ChannelFilter
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

// addNumbered appends a condition whose placeholders are numbered already,
// from the next argument on.
func (f *filterSQL) addNumbered(cond string, vals ...any) {
	f.args = append(f.args, vals...)
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
	if q := strings.ToLower(strings.TrimSpace(f.Q)); q != "" {
		switch {
		case emailaddr.Valid(q):
			// A whole address: the unique index answers (review 2026-11 AU-2).
			out.add("c.email = ?", emailaddr.Normalize(q))
		case strings.Contains(q, "@"):
			// Part of an address: the trigram index on email.
			out.add("c.email LIKE ?", "%"+db.EscapeLike(q)+"%")
		default:
			// Part of an address or a name through the trigram indexes; tags
			// are stored lower-case (cleanTags), so the search is too.
			like := "%" + db.EscapeLike(q) + "%"
			out.add("(c.email LIKE ? OR lower(c.first_name || ' ' || c.last_name) LIKE ? OR c.tags @> ARRAY[?::text])", like, like, q)
		}
	}
	if f.List != "" {
		id, err := uuid.Parse(f.List)
		if err != nil {
			return api.Invalid("list", say(ctx, mFilterList, nil))
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
	if t := strings.ToLower(strings.TrimSpace(f.Tag)); t != "" {
		out.add("c.tags @> ARRAY[?::text]", t)
	}
	if f.JoinedAfter != "" || f.JoinedBefore != "" {
		// Days in the workspace's time zone (review 2026-11 AU-10).
		loc := time.UTC
		var tz string
		if err := s.Pool.QueryRow(ctx, `SELECT timezone FROM workspaces WHERE id = $1`, wsID).Scan(&tz); err == nil {
			if l, err := time.LoadLocation(tz); err == nil {
				loc = l
			}
		}
		for _, d := range []struct {
			v, loc string
			cond   string
			next   bool
		}{{f.JoinedAfter, "joinedAfter", "c.created_at >= ?", false}, {f.JoinedBefore, "joinedBefore", "c.created_at < ?", true}} {
			if d.v == "" {
				continue
			}
			day, err := time.ParseInLocation("2006-01-02", d.v, loc)
			if err != nil {
				return api.Invalid(d.loc, say(ctx, mJoinedAfter, nil))
			}
			if d.next {
				day = day.AddDate(0, 0, 1)
			}
			out.add(d.cond, day)
		}
	}
	if f.Import != "" {
		id, err := uuid.Parse(f.Import)
		if err != nil {
			return api.Invalid("import", say(ctx, mFilterImport, nil))
		}
		out.add("c.import_id = ?", id)
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
	if err := s.storeFilter(ctx, out, f.StoreFilter); err != nil {
		return err
	}
	if err := channelFilter(ctx, out, f.ChannelFilter); err != nil {
		return err
	}
	if f.Segment != "" {
		id, err := uuid.Parse(f.Segment)
		if err != nil {
			return api.Invalid("segment", say(ctx, mFilterSegment, nil))
		}
		cond, args, err := s.Segments.Condition(ctx, wsID, id, len(out.args)+1)
		if err != nil {
			return segments.Problem(ctx, err)
		}
		out.addNumbered(cond, args...)
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

// TierName is how a tier reads in the UI, in English.
func TierName(t string) string { return TierNameIn(i18n.Default, t) }

// TierNameIn is how a tier reads in a reader's language: "At risk".
func TierNameIn(l i18n.Lang, t string) string { return lookup(l, "tier."+t, t) }

// ConsentName is how a consent state reads in the UI, in English.
func ConsentName(c string) string { return ConsentNameIn(i18n.Default, c) }

// ConsentNameIn is how a consent state reads in a reader's language:
// "Double opt-in".
func ConsentNameIn(l i18n.Lang, c string) string { return lookup(l, "consent."+c, c) }

// MailboxLabel is "Outlook" or "Other · Google Workspace".
func MailboxLabel(provider, detail string) string {
	name := emailaddr.MailboxName(provider)
	if provider == emailaddr.Other && detail != "" {
		return name + " · " + detail
	}
	return name
}
