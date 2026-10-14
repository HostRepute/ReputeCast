package audience

// Public self-service: signups, double opt-in confirmations, the preference
// center and unsubscribes (spec 5.12, 7.12). These act for the contact, so
// instead of audit entries every change writes consent events carrying the
// evidence a regulator or mailbox provider asks for: IP, browser, page and,
// for opt-ins, the exact consent text and its version. They run inside the
// caller's transaction so the form's own bookkeeping commits with them.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/store"
)

// Consent sources written by the public pages. Form signups use
// "form:<form slug>" (segments.SourceLabel reads it as "Signup form · …").
const (
	SourceSubscribePage   = "subscribe_page"
	SourceConfirmLink     = "confirm_link"
	SourcePreferences     = "preference_center"
	SourceUnsubscribePage = "unsubscribe_page"
	SourceOneClick        = "one_click"
)

// ErrContactGone is returned when a link points at a contact that was
// erased or belongs to another workspace.
var ErrContactGone = errors.New("audience: contact not found")

// FieldError is a value the contact must correct, worded for them.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }

// PublicEvidence is what the contact did and from where.
type PublicEvidence struct {
	// Source names the page or form ("form:newsletter-signup", "one_click").
	Source    string
	IP        *netip.Addr
	UserAgent string
	// ConsentText is the exact sentence the contact agreed to; ConsentVersion
	// is its version ("v3").
	ConsentText    string
	ConsentVersion string
	// Note goes on the consent events.
	Note string
	At   time.Time
}

func (e PublicEvidence) at() time.Time {
	if e.At.IsZero() {
		return time.Now()
	}
	return e.At
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func consentEvent(ctx context.Context, q *store.Queries, contact uuid.UUID, list *uuid.UUID, action, detail string, ev PublicEvidence, withConsent bool) error {
	p := store.SignupInsertConsentParams{
		ID: uuid.Must(uuid.NewV7()), ContactID: contact, ListID: list, Action: action, Source: ev.Source, Ip: ev.IP,
		UserAgent: clip(ev.UserAgent, 500), Detail: detail, Note: ev.Note, OccurredAt: ev.at(),
	}
	if withConsent {
		p.ConsentText, p.ConsentVersion = ev.ConsentText, ev.ConsentVersion
	}
	return q.SignupInsertConsent(ctx, p)
}

// SignupInput is a subscription request from a form or the hosted page.
type SignupInput struct {
	Email     string
	FirstName string
	LastName  string
	Company   string
	// Fields are custom field values by key, checked against their type.
	Fields map[string]string
	// Lists to join; each must belong to the workspace.
	Lists []uuid.UUID
	// DoubleOptIn keeps the memberships pending until the confirmation link
	// is clicked.
	DoubleOptIn bool
	Evidence    PublicEvidence
}

// SignupResult says what happened. Public pages show the same message in
// every case, so a form never reveals who is already subscribed.
type SignupResult struct {
	ContactID uuid.UUID
	Email     string
	FirstName string
	Created   bool
	// Pending lists wait for the confirmation click (double opt-in, or a
	// returning contact who had unsubscribed).
	Pending []uuid.UUID
	// Subscribed lists were joined now (single opt-in).
	Subscribed []uuid.UUID
	// Already lists the contact was subscribed to before.
	Already []uuid.UUID
	// Blocked means the address is suppressed for a reason other than an
	// unsubscribe (hard bounce, complaint, spam trap…): nothing changed and
	// no email may go out.
	Blocked bool
}

// NeedsConfirmation reports whether a confirmation email should go out.
func (r SignupResult) NeedsConfirmation() bool { return !r.Blocked && len(r.Pending) > 0 }

// Signup adds or updates a contact from a public form inside tx. A new
// contact gets the form's fields; an existing contact only gets the fields
// it left empty, because anyone can type anyone's address into a form. A
// contact who unsubscribed before must confirm by email even when the form
// is single opt-in.
func (s *Service) Signup(ctx context.Context, tx pgx.Tx, ws *store.Workspace, in SignupInput) (SignupResult, error) {
	email, problem := emailaddr.Check(in.Email)
	if problem != "" {
		return SignupResult{}, &FieldError{Field: "email", Message: "That doesn't look like a complete email address."}
	}
	if len(in.Lists) == 0 {
		return SignupResult{}, &FieldError{Field: "lists", Message: "Pick at least one newsletter."}
	}
	q := store.New(tx)
	for _, l := range in.Lists {
		if _, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: l}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return SignupResult{}, &FieldError{Field: "lists", Message: "One of those newsletters isn't available any more. Reload the page and pick again."}
			}
			return SignupResult{}, err
		}
	}
	attrs, err := s.signupFields(ctx, q, ws, in.Fields)
	if err != nil {
		return SignupResult{}, err
	}
	out := SignupResult{Email: email, Pending: []uuid.UUID{}, Subscribed: []uuid.UUID{}, Already: []uuid.UUID{}}
	double := in.DoubleOptIn
	if sup, err := q.SignupSuppression(ctx, store.SignupSuppressionParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		if sup.Reason != "unsubscribe" {
			out.Blocked = true
			return out, nil
		}
		double = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	at := in.Evidence.at()
	first, last, company := clip(strings.TrimSpace(in.FirstName), 100), clip(strings.TrimSpace(in.LastName), 100), clip(strings.TrimSpace(in.Company), 200)
	c, err := q.SignupContactByEmail(ctx, store.SignupContactByEmailParams{WorkspaceID: ws.ID, Email: email})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		consent := "single"
		if double {
			consent = "pending"
		}
		c, err = q.SignupInsertContact(ctx, store.SignupInsertContactParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: email, FirstName: first, LastName: last, Company: company,
			MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), Consent: consent, Source: in.Evidence.Source,
			Attributes: attrs, RoleAddress: emailaddr.IsRole(email), At: at,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			// Another submission for the same address created it a moment ago.
			c, err = q.SignupContactByEmail(ctx, store.SignupContactByEmailParams{WorkspaceID: ws.ID, Email: email})
		} else if err == nil {
			out.Created = true
		}
		if err != nil {
			return out, err
		}
	case err != nil:
		return out, err
	}
	out.ContactID, out.FirstName = c.ID, c.FirstName
	if !out.Created {
		switch c.Status {
		case "bounced", "complained", "suppressed":
			out.Blocked = true
			return out, nil
		case "unsubscribed":
			double = true
		}
		if err := q.SignupFillContact(ctx, store.SignupFillContactParams{ID: c.ID, FirstName: first, LastName: last, Company: company, Attributes: attrs, At: at}); err != nil {
			return out, err
		}
		if out.FirstName == "" {
			out.FirstName = first
		}
	}
	members, err := q.SignupMemberships(ctx, c.ID)
	if err != nil {
		return out, err
	}
	status := map[uuid.UUID]string{}
	for _, m := range members {
		status[m.ListID] = m.Status
	}
	seen := map[uuid.UUID]bool{}
	for _, l := range in.Lists {
		if seen[l] {
			continue
		}
		seen[l] = true
		lid := l
		st := status[l]
		if st == "subscribed" && c.Status == "active" {
			out.Already = append(out.Already, l)
			continue
		}
		if double {
			if st != "pending" {
				if err := q.SignupSetMembership(ctx, store.SignupSetMembershipParams{ListID: l, ContactID: c.ID, Status: "pending", Source: in.Evidence.Source, At: at}); err != nil {
					return out, err
				}
			}
			ev := in.Evidence
			if ev.Note == "" {
				ev.Note = "waiting for the confirmation click"
			}
			if err := consentEvent(ctx, q, c.ID, &lid, "subscribed", "", ev, true); err != nil {
				return out, err
			}
			out.Pending = append(out.Pending, l)
			continue
		}
		if err := q.SignupSetMembership(ctx, store.SignupSetMembershipParams{ListID: l, ContactID: c.ID, Status: "subscribed", Source: in.Evidence.Source, At: at}); err != nil {
			return out, err
		}
		action := "subscribed"
		if st == "unsubscribed" {
			action = "resubscribed"
		}
		if err := consentEvent(ctx, q, c.ID, &lid, action, "", in.Evidence, true); err != nil {
			return out, err
		}
		out.Subscribed = append(out.Subscribed, l)
	}
	if !out.Created && !double && len(out.Subscribed) > 0 && c.Consent == "pending" {
		// Waiting for re-permission and now signed up again: that is consent.
		if err := q.SignupSetContactState(ctx, store.SignupSetContactStateParams{ID: c.ID, Status: "active", Consent: "single", At: at}); err != nil {
			return out, err
		}
	}
	if err := q.SignupTouchContact(ctx, store.SignupTouchContactParams{ID: c.ID, At: at}); err != nil {
		return out, err
	}
	return out, nil
}

// signupFields checks custom field values from a form against their types.
func (s *Service) signupFields(ctx context.Context, q *store.Queries, ws *store.Workspace, fields map[string]string) (json.RawMessage, error) {
	if len(fields) == 0 {
		return json.RawMessage(`{}`), nil
	}
	defs, err := q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	byKey := map[string]store.CustomField{}
	for _, f := range defs {
		byKey[f.Key] = f
	}
	attrs := map[string]any{}
	for k, v := range fields {
		f, ok := byKey[k]
		if !ok {
			continue // a field removed from the workspace since the page loaded
		}
		val, err := customValue(f, v)
		if err != nil {
			return nil, &FieldError{Field: k, Message: publicFieldMessage(f)}
		}
		if val != nil {
			attrs[k] = val
		}
	}
	return json.Marshal(attrs)
}

// publicFieldMessage explains a custom field's format to a visitor.
func publicFieldMessage(f store.CustomField) string {
	switch f.Type {
	case "number":
		return "Enter a number."
	case "date":
		return "Enter a date like 2026-10-30."
	case "boolean":
		return "Answer yes or no."
	case "select":
		var opts []string
		_ = json.Unmarshal(f.Options, &opts)
		if len(opts) > 0 {
			return "Pick one of: " + strings.Join(opts, ", ") + "."
		}
	}
	return "Keep it under 500 characters."
}

// RecordOptInSent notes that the confirmation email went out (or was sent
// again as a reminder) for the contact's pending lists.
func (s *Service) RecordOptInSent(ctx context.Context, q *store.Queries, contactID uuid.UUID, lists []uuid.UUID, ev PublicEvidence) error {
	if len(lists) == 0 {
		return consentEvent(ctx, q, contactID, nil, "optin_sent", "", ev, false)
	}
	for _, l := range lists {
		lid := l
		if err := consentEvent(ctx, q, contactID, &lid, "optin_sent", "", ev, false); err != nil {
			return err
		}
	}
	return nil
}

// ConfirmResult is the outcome of a confirmation click.
type ConfirmResult struct {
	ContactID uuid.UUID
	Email     string
	FirstName string
	// Confirmed lists moved from pending to subscribed now.
	Confirmed []uuid.UUID
	// Already means nothing was pending: the link was clicked before.
	Already bool
	// Blocked means the address is suppressed (bounce, complaint…).
	Blocked bool
}

// Confirm completes double opt-in for every pending list of a contact
// inside tx. The consent text the contact agreed to at signup is copied
// onto the confirmation, so each confirmed list carries its own evidence.
func (s *Service) Confirm(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, ev PublicEvidence) (ConfirmResult, error) {
	q := store.New(tx)
	c, err := q.SignupContactByID(ctx, store.SignupContactByIDParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmResult{}, ErrContactGone
	}
	if err != nil {
		return ConfirmResult{}, err
	}
	out := ConfirmResult{ContactID: c.ID, Email: c.Email, FirstName: c.FirstName, Confirmed: []uuid.UUID{}}
	switch c.Status {
	case "bounced", "complained", "suppressed":
		out.Blocked = true
		return out, nil
	}
	if sup, err := q.SignupSuppression(ctx, store.SignupSuppressionParams{WorkspaceID: ws.ID, Email: c.Email}); err == nil && sup.Reason != "unsubscribe" {
		out.Blocked = true
		return out, nil
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	at := ev.at()
	lists, err := q.SignupConfirmPending(ctx, store.SignupConfirmPendingParams{ContactID: c.ID, At: at})
	if err != nil {
		return out, err
	}
	if len(lists) == 0 {
		out.Already = true
		return out, nil
	}
	slices.SortFunc(lists, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	out.Confirmed = lists
	if _, err := q.SignupLiftUnsubscribe(ctx, store.SignupLiftUnsubscribeParams{WorkspaceID: ws.ID, Email: c.Email}); err != nil {
		return out, err
	}
	if err := q.SignupSetContactState(ctx, store.SignupSetContactStateParams{ID: c.ID, Status: "active", Consent: "double", At: at}); err != nil {
		return out, err
	}
	if ev.ConsentText == "" {
		if prev, err := q.SignupLastConsent(ctx, store.SignupLastConsentParams{ContactID: c.ID, Action: "subscribed"}); err == nil {
			ev.ConsentText, ev.ConsentVersion = prev.ConsentText, prev.ConsentVersion
		}
	}
	for _, l := range lists {
		lid := l
		if err := consentEvent(ctx, q, c.ID, &lid, "confirmed", "", ev, true); err != nil {
			return out, err
		}
	}
	return out, nil
}

// PublicConfirmation is when and from where a contact last confirmed.
type PublicConfirmation struct {
	At time.Time
	IP *netip.Addr
}

// PublicContact is what the public pages show about the contact a link
// was signed for.
type PublicContact struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	Status    string
	Consent   string
	// Lists maps list ids to the membership status (subscribed, pending, unsubscribed).
	Lists       map[uuid.UUID]string
	Frequency   string
	PausedUntil *time.Time
	Confirmed   *PublicConfirmation
	// LastName and Attributes personalize "view online".
	LastName   string
	Attributes map[string]any
}

// FindPublicContact returns the contact behind an address, for "email me
// a link to my preferences". ok is false for unknown or erased addresses.
func (s *Service) FindPublicContact(ctx context.Context, q *store.Queries, ws *store.Workspace, email string) (uuid.UUID, bool, error) {
	e, problem := emailaddr.Check(email)
	if problem != "" {
		return uuid.Nil, false, nil
	}
	c, err := q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: e})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return c.ID, true, nil
}

// Subscribed reports whether the contact gets list l.
func (p PublicContact) Subscribed(l uuid.UUID) bool {
	return p.Lists[l] == "subscribed" && p.Status == "active"
}

// LoadPublicContact reads a contact for a public page. Erased contacts and
// contacts of other workspaces are ErrContactGone.
func (s *Service) LoadPublicContact(ctx context.Context, q *store.Queries, ws *store.Workspace, contactID uuid.UUID) (PublicContact, error) {
	c, err := q.GetContact(ctx, store.GetContactParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicContact{}, ErrContactGone
	}
	if err != nil {
		return PublicContact{}, err
	}
	out := PublicContact{ID: c.ID, Email: c.Email, FirstName: c.FirstName, LastName: c.LastName, Status: c.Status, Consent: c.Consent,
		Lists: map[uuid.UUID]string{}, Frequency: "all", Attributes: map[string]any{}}
	_ = json.Unmarshal(c.Attributes, &out.Attributes)
	members, err := q.SignupMemberships(ctx, c.ID)
	if err != nil {
		return out, err
	}
	for _, m := range members {
		out.Lists[m.ListID] = m.Status
	}
	if p, err := q.SignupPreferences(ctx, c.ID); err == nil {
		out.Frequency, out.PausedUntil = p.Frequency, p.PausedUntil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if last, err := q.SignupLastConsent(ctx, store.SignupLastConsentParams{ContactID: c.ID, Action: "confirmed"}); err == nil {
		out.Confirmed = &PublicConfirmation{At: last.OccurredAt, IP: last.Ip}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	return out, nil
}

// PublicList is a list a public page can offer.
type PublicList struct {
	ID          uuid.UUID
	Name        string
	Description string
	// Public lists are offered in the preference center by default.
	Public     bool
	Subscribed int64
}

// PublicLists returns the workspace's lists with their size, oldest first.
func (s *Service) PublicLists(ctx context.Context, q *store.Queries, ws *store.Workspace) ([]PublicList, error) {
	rows, err := q.SignupPublicLists(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]PublicList, 0, len(rows))
	for _, r := range rows {
		out = append(out, PublicList{ID: r.ID, Name: r.Name, Description: r.Description, Public: r.Public, Subscribed: r.Subscribed})
	}
	return out, nil
}

// PreferencesInput is a preference center submission.
type PreferencesInput struct {
	// Shown are the lists the page offered; Subscribed the ones left checked.
	Shown      []uuid.UUID
	Subscribed []uuid.UUID
	// Frequency is all, weekly or monthly; empty keeps it.
	Frequency string
	// PauseDays pauses every list for 30 or 90 days; 0 resumes; -1 keeps it.
	PauseDays int
	// FirstName changes the first name when set.
	FirstName *string
}

// PreferencesResult lists what changed.
type PreferencesResult struct {
	Joined      []uuid.UUID
	Left        []uuid.UUID
	Frequency   string
	PausedUntil *time.Time
	Changed     bool
}

// Frequencies offered in the preference center.
var Frequencies = []string{"all", "weekly", "monthly"}

// SavePreferences applies a preference center submission inside tx. The
// signed link proves the contact reads this mailbox, so joining a list here
// needs no further confirmation, even after an unsubscribe.
func (s *Service) SavePreferences(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, in PreferencesInput, ev PublicEvidence) (PreferencesResult, error) {
	q := store.New(tx)
	c, err := q.SignupContactByID(ctx, store.SignupContactByIDParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return PreferencesResult{}, ErrContactGone
	}
	if err != nil {
		return PreferencesResult{}, err
	}
	out := PreferencesResult{Joined: []uuid.UUID{}, Left: []uuid.UUID{}}
	if in.Frequency != "" && !slices.Contains(Frequencies, in.Frequency) {
		return out, &FieldError{Field: "frequency", Message: "Pick how often you'd like emails."}
	}
	at := ev.at()
	members, err := q.SignupMemberships(ctx, c.ID)
	if err != nil {
		return out, err
	}
	status := map[uuid.UUID]string{}
	for _, m := range members {
		status[m.ListID] = m.Status
	}
	want := map[uuid.UUID]bool{}
	for _, l := range in.Subscribed {
		want[l] = true
	}
	blocked := c.Status == "bounced" || c.Status == "complained" || c.Status == "suppressed"
	for _, l := range in.Shown {
		lid := l
		st := status[l]
		switch {
		case want[l] && st != "subscribed" && !blocked:
			if _, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: l}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				return out, err
			}
			if err := q.SignupSetMembership(ctx, store.SignupSetMembershipParams{ListID: l, ContactID: c.ID, Status: "subscribed", Source: ev.Source, At: at}); err != nil {
				return out, err
			}
			action := "subscribed"
			if st == "unsubscribed" || c.Status == "unsubscribed" {
				action = "resubscribed"
			}
			if err := consentEvent(ctx, q, c.ID, &lid, action, "", ev, false); err != nil {
				return out, err
			}
			out.Joined = append(out.Joined, l)
		case !want[l] && (st == "subscribed" || st == "pending"):
			if err := q.SignupSetMembership(ctx, store.SignupSetMembershipParams{ListID: l, ContactID: c.ID, Status: "unsubscribed", Source: ev.Source, At: at}); err != nil {
				return out, err
			}
			if err := consentEvent(ctx, q, c.ID, &lid, "unsubscribed", "", ev, false); err != nil {
				return out, err
			}
			out.Left = append(out.Left, l)
		}
	}
	if len(out.Joined) > 0 && c.Status == "unsubscribed" {
		if _, err := q.SignupLiftUnsubscribe(ctx, store.SignupLiftUnsubscribeParams{WorkspaceID: ws.ID, Email: c.Email}); err != nil {
			return out, err
		}
		consent := c.Consent
		if consent == "pending" {
			consent = "double"
		}
		if err := q.SignupSetContactState(ctx, store.SignupSetContactStateParams{ID: c.ID, Status: "active", Consent: consent, At: at}); err != nil {
			return out, err
		}
	}
	prefs := store.SignupUpsertPreferencesParams{ContactID: c.ID, Frequency: "all", At: at}
	if p, err := q.SignupPreferences(ctx, c.ID); err == nil {
		prefs.Frequency, prefs.PausedUntil = p.Frequency, p.PausedUntil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.Frequency, out.PausedUntil = prefs.Frequency, prefs.PausedUntil
	prefsChanged := false
	if in.Frequency != "" && in.Frequency != prefs.Frequency {
		prefs.Frequency = in.Frequency
		prefsChanged = true
		if err := consentEvent(ctx, q, c.ID, nil, "frequency_changed", frequencyWords(in.Frequency), ev, false); err != nil {
			return out, err
		}
	}
	paused := prefs.PausedUntil != nil && prefs.PausedUntil.After(at)
	switch {
	case in.PauseDays > 0:
		until := at.AddDate(0, 0, in.PauseDays)
		prefs.PausedUntil = &until
		prefsChanged = true
		if err := consentEvent(ctx, q, c.ID, nil, "paused", fmt.Sprintf("for %d days, until %s", in.PauseDays, until.Format("2 Jan 2006")), ev, false); err != nil {
			return out, err
		}
	case in.PauseDays == 0 && paused:
		prefs.PausedUntil = nil
		prefsChanged = true
		if err := consentEvent(ctx, q, c.ID, nil, "paused", "pause lifted", ev, false); err != nil {
			return out, err
		}
	}
	if prefsChanged {
		if err := q.SignupUpsertPreferences(ctx, prefs); err != nil {
			return out, err
		}
		out.Frequency, out.PausedUntil = prefs.Frequency, prefs.PausedUntil
	}
	nameChanged := false
	if in.FirstName != nil {
		name := clip(strings.TrimSpace(*in.FirstName), 100)
		if name != c.FirstName {
			if err := q.SignupSetFirstName(ctx, store.SignupSetFirstNameParams{ID: c.ID, FirstName: name, At: at}); err != nil {
				return out, err
			}
			nameChanged = true
		}
	}
	out.Changed = len(out.Joined)+len(out.Left) > 0 || prefsChanged || nameChanged
	if out.Changed {
		if err := q.SignupTouchContact(ctx, store.SignupTouchContactParams{ID: c.ID, At: at}); err != nil {
			return out, err
		}
	}
	return out, nil
}

func frequencyWords(f string) string {
	switch f {
	case "weekly":
		return "weekly roundup"
	case "monthly":
		return "monthly best-of"
	}
	return "every issue"
}

type UnsubscribeResult struct {
	ContactID uuid.UUID
	Email     string
	FirstName string
	// Lists the contact left now.
	Lists []uuid.UUID
	// Already means the contact had unsubscribed before (a repeated
	// one-click POST, a second visit).
	Already bool
}

// Unsubscribe removes the contact from every list and suppresses Broadcasts
// and Automations to the address (System mail still goes through), inside
// tx. campaignID names the email the link came from, if any.
func (s *Service) Unsubscribe(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, campaignID *uuid.UUID, ev PublicEvidence) (UnsubscribeResult, error) {
	q := store.New(tx)
	c, err := q.SignupContactByID(ctx, store.SignupContactByIDParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return UnsubscribeResult{}, ErrContactGone
	}
	if err != nil {
		return UnsubscribeResult{}, err
	}
	at := ev.at()
	out := UnsubscribeResult{ContactID: c.ID, Email: c.Email, FirstName: c.FirstName}
	lists, err := q.SignupUnsubscribeAll(ctx, store.SignupUnsubscribeAllParams{ContactID: c.ID, At: at})
	if err != nil {
		return out, err
	}
	slices.SortFunc(lists, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	out.Lists = lists
	if len(lists) == 0 && c.Status == "unsubscribed" {
		out.Already = true
		return out, nil
	}
	if c.Status == "active" {
		if err := q.SignupSetContactState(ctx, store.SignupSetContactStateParams{ID: c.ID, Status: "unsubscribed", Consent: c.Consent, At: at}); err != nil {
			return out, err
		}
	}
	where := "Unsubscribe page"
	if ev.Source == SourceOneClick {
		where = "List-Unsubscribe one-click"
	}
	if err := q.SignupAddUnsubscribe(ctx, store.SignupAddUnsubscribeParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: c.Email, Source: ev.Source, Detail: ev.Note, CampaignID: campaignID, Context: where, At: at,
	}); err != nil {
		return out, err
	}
	if len(lists) == 0 {
		return out, consentEvent(ctx, q, c.ID, nil, "unsubscribed", "", ev, false)
	}
	for _, l := range lists {
		lid := l
		if err := consentEvent(ctx, q, c.ID, &lid, "unsubscribed", "", ev, false); err != nil {
			return out, err
		}
	}
	return out, nil
}

// Resubscribe undoes the latest unsubscribe from the confirmation page
// ("Changed your mind?"): the lists it left come back and the suppression
// it added is lifted, inside tx.
func (s *Service) Resubscribe(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, ev PublicEvidence) ([]uuid.UUID, error) {
	q := store.New(tx)
	c, err := q.SignupContactByID(ctx, store.SignupContactByIDParams{WorkspaceID: ws.ID, ID: contactID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContactGone
	}
	if err != nil {
		return nil, err
	}
	if c.Status != "unsubscribed" {
		return []uuid.UUID{}, nil
	}
	lists, err := q.SignupLastUnsubscribedLists(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	at := ev.at()
	back := []uuid.UUID{}
	for _, l := range lists {
		if _, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: l}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		if err := q.SignupSetMembership(ctx, store.SignupSetMembershipParams{ListID: l, ContactID: c.ID, Status: "subscribed", Source: ev.Source, At: at}); err != nil {
			return nil, err
		}
		lid := l
		if err := consentEvent(ctx, q, c.ID, &lid, "resubscribed", "", ev, false); err != nil {
			return nil, err
		}
		back = append(back, l)
	}
	if _, err := q.SignupLiftUnsubscribe(ctx, store.SignupLiftUnsubscribeParams{WorkspaceID: ws.ID, Email: c.Email}); err != nil {
		return nil, err
	}
	if err := q.SignupSetContactState(ctx, store.SignupSetContactStateParams{ID: c.ID, Status: "active", Consent: c.Consent, At: at}); err != nil {
		return nil, err
	}
	return back, nil
}

// PreviewContact picks the subscriber the app previews the public pages for:
// the address asked for, else whoever confirmed a subscription last, else
// the newest active contact. ok is false for a workspace without contacts.
func (s *Service) PreviewContact(ctx context.Context, q *store.Queries, ws *store.Workspace, email string) (uuid.UUID, bool, error) {
	if strings.TrimSpace(email) != "" {
		return s.FindPublicContact(ctx, q, ws, email)
	}
	id, err := q.SignupPreviewConfirmed(ctx, ws.ID)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	id, err = q.SignupPreviewLatest(ctx, ws.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	return id, err == nil, err
}

// SamplePublicContact is the example subscriber of previews in a workspace
// without contacts: on every public list.
func SamplePublicContact(lists []PublicList) PublicContact {
	pc := PublicContact{Email: "you@example.com", Status: "active", Consent: "double", Lists: map[uuid.UUID]string{}, Frequency: "all", Attributes: map[string]any{}}
	for _, l := range lists {
		if l.Public {
			pc.Lists[l.ID] = "subscribed"
		}
	}
	return pc
}

// PublicContactName is a contact's address and first name.
type PublicContactName struct {
	Email     string
	FirstName string
}

// PublicContactNames returns the addresses and first names of some contacts
// of ws (erased contacts and other workspaces' are left out), for lists of
// signups such as the confirmations waiting to be sent.
func (s *Service) PublicContactNames(ctx context.Context, q *store.Queries, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]PublicContactName, error) {
	out := map[uuid.UUID]PublicContactName{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.SignupContactNames(ctx, store.SignupContactNamesParams{WorkspaceID: ws.ID, Ids: ids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = PublicContactName{Email: r.Email, FirstName: r.FirstName}
	}
	return out, nil
}
