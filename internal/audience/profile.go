package audience

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactProfileStats is the headline row of the profile (last 90 days).
type ContactProfileStats struct {
	Emails90d  int64 `json:"emails90d"`
	HumanOpens int64 `json:"humanOpens" doc:"Emails opened by a person"`
	// HumanOpenRate is in percent of emails.
	HumanOpenRate   *float64   `json:"humanOpenRate,omitempty"`
	Clicks          int64      `json:"clicks"`
	LastOpenAt      *time.Time `json:"lastOpenAt,omitempty"`
	LastOpenMachine bool       `json:"lastOpenMachine"`
}

// ContactProfileField is one row of the Fields card.
type ContactProfileField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	// Custom fields show their key and type ("mrr custom · number").
	Custom  bool   `json:"custom"`
	Type    string `json:"type" enum:"text,number,date,boolean,select,builtin"`
	Display string `json:"display" enum:",currency"`
}

// ContactConsentEvent is one consent event of a list.
type ContactConsentEvent struct {
	Action         string `json:"action"`
	Source         string `json:"source"`
	SourceLabel    string `json:"sourceLabel"`
	Detail         string `json:"detail"`
	Note           string `json:"note"`
	IP             string `json:"ip,omitempty"`
	Actor          string `json:"actor,omitempty"`
	ConsentVersion string `json:"consentVersion,omitempty"`
	// ConsentText is the exact text the contact agreed to, when it was
	// recorded ("Also send me the Weekly Digest" at a product checkout).
	ConsentText string `json:"consentText,omitempty"`
	// Language is the language the consent text was shown in ("es").
	Language string    `json:"language,omitempty"`
	At       time.Time `json:"at"`
}

// ContactListConsent is the consent state and history of one list.
type ContactListConsent struct {
	ListID      string                `json:"listId"`
	Name        string                `json:"name"`
	Status      string                `json:"status" enum:"subscribed,pending,unsubscribed,none"`
	DoubleOptIn bool                  `json:"doubleOptIn"`
	Events      []ContactConsentEvent `json:"events"`
}

// ContactProfilePreferences is what the subscriber chose in the preference center.
type ContactProfilePreferences struct {
	Frequency   string     `json:"frequency" enum:"all,weekly,monthly"`
	PausedUntil *time.Time `json:"pausedUntil,omitempty"`
	FewerEmails bool       `json:"fewerEmails"`
}

// ContactProfileSuppression says the address is on the suppression list.
type ContactProfileSuppression struct {
	ID     string    `json:"id"`
	Reason string    `json:"reason"`
	Scope  string    `json:"scope" enum:"all,marketing"`
	At     time.Time `json:"at"`
}

// ContactProfile is the contact profile screen.
type ContactProfile struct {
	ID              string                     `json:"id"`
	Email           string                     `json:"email"`
	FirstName       string                     `json:"firstName"`
	LastName        string                     `json:"lastName"`
	Name            string                     `json:"name"`
	Tier            string                     `json:"tier" enum:"active,cooling,at_risk,sunset"`
	Consent         string                     `json:"consent" enum:"double,single,pending"`
	MailboxProvider string                     `json:"mailboxProvider"`
	MailboxLabel    string                     `json:"mailboxLabel"`
	JoinedAt        time.Time                  `json:"joinedAt"`
	Source          string                     `json:"source"`
	SourceLabel     string                     `json:"sourceLabel"`
	RoleAddress     bool                       `json:"roleAddress"`
	Suppression     *ContactProfileSuppression `json:"suppression,omitempty"`
	Stats           ContactProfileStats        `json:"stats"`
	Fields          []ContactProfileField      `json:"fields"`
	Tags            []string                   `json:"tags"`
	Lists           []ContactListConsent       `json:"lists"`
	Preferences     ContactProfilePreferences  `json:"preferences"`
	ActivityTotal   int64                      `json:"activityTotal"`
	// Language is the contact's language with where it came from.
	Language ContactLanguage `json:"language"`
}

// getContact loads a contact visible to the caller.
func (s *Service) getContact(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (store.Contact, error) {
	c, err := q.GetContact(ctx, store.GetContactParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return c, api.MapError(err, "Contact")
	}
	ok, err := s.inScope(ctx, id)
	if err != nil {
		return c, err
	}
	if !ok {
		return c, api.NotFound("Contact")
	}
	return c, nil
}

// Profile returns everything the profile screen shows.
func (s *Service) Profile(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ContactProfile, error) {
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return ContactProfile{}, err
	}
	out := ContactProfile{
		ID: c.ID.String(), Email: c.Email, FirstName: c.FirstName, LastName: c.LastName, Name: strings.TrimSpace(c.FirstName + " " + c.LastName),
		Tier: c.Tier, Consent: c.Consent, MailboxProvider: c.MailboxProvider, MailboxLabel: MailboxLabel(c.MailboxProvider, c.MailboxDetail),
		JoinedAt: c.CreatedAt, Source: c.Source, SourceLabel: segments.SourceLabel(c.Source), RoleAddress: c.RoleAddress,
		Tags: c.Tags, Fields: []ContactProfileField{}, Lists: []ContactListConsent{}, Preferences: ContactProfilePreferences{Frequency: "all"},
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if sup, err := s.Q.GetSuppressionByEmail(ctx, store.GetSuppressionByEmailParams{WorkspaceID: ws.ID, Email: c.Email}); err == nil {
		out.Suppression = &ContactProfileSuppression{ID: sup.ID.String(), Reason: sup.Reason, Scope: sup.Scope, At: sup.CreatedAt}
	}
	eng, err := s.Q.ContactEngagementSince(ctx, store.ContactEngagementSinceParams{ContactID: &c.ID, Since: s.now().AddDate(0, 0, -90)})
	if err != nil {
		return out, err
	}
	out.Stats = ContactProfileStats{Emails90d: eng.Emails, HumanOpens: eng.HumanOpened, Clicks: eng.Clicked, LastOpenAt: c.LastOpenAt, LastOpenMachine: c.LastOpenMachine}
	if eng.Emails > 0 {
		r := math.Round(float64(eng.HumanOpened)/float64(eng.Emails)*1000) / 10
		out.Stats.HumanOpenRate = &r
	}
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Fields = profileFields(c, fields)
	if err := s.listConsent(ctx, ws, c, &out); err != nil {
		return out, err
	}
	if p, err := s.Q.GetContactPreferences(ctx, c.ID); err == nil {
		out.Preferences.Frequency = p.Frequency
		if p.PausedUntil != nil && p.PausedUntil.After(s.now()) {
			out.Preferences.PausedUntil = p.PausedUntil
		}
	}
	if out.Preferences.FewerEmails, err = s.Q.ContactChoseFewer(ctx, &c.ID); err != nil {
		return out, err
	}
	if out.ActivityTotal, err = s.Q.ContactActivityCount(ctx, &c.ID); err != nil {
		return out, err
	}
	if out.Language, err = s.contactLanguage(ctx, c); err != nil {
		return out, err
	}
	return out, nil
}

// profileFields lists built-in fields, then custom fields, then language.
func profileFields(c store.Contact, custom []store.CustomField) []ContactProfileField {
	out := []ContactProfileField{
		{Key: "first_name", Label: "First name", Value: c.FirstName, Type: "builtin"},
		{Key: "last_name", Label: "Last name", Value: c.LastName, Type: "builtin"},
		{Key: "company", Label: "Company", Value: c.Company, Type: "builtin"},
	}
	loc := []string{}
	if c.Country != "" {
		loc = append(loc, CountryName(c.Country))
	}
	if c.Timezone != "" {
		loc = append(loc, c.Timezone)
	}
	out = append(out, ContactProfileField{Key: "country_timezone", Label: "Country · timezone", Value: strings.Join(loc, " · "), Type: "builtin"})
	var attrs map[string]any
	_ = json.Unmarshal(c.Attributes, &attrs)
	for _, f := range custom {
		out = append(out, ContactProfileField{Key: f.Key, Label: f.Key, Value: attrString(attrs[f.Key]), Custom: true, Type: f.Type, Display: f.Display})
	}
	return append(out, ContactProfileField{Key: "language", Label: "Language", Value: c.Language, Type: "builtin"})
}

func attrString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// listConsent builds the "Consent per list" card: every list of the
// workspace with this contact's status and the events behind it.
func (s *Service) listConsent(ctx context.Context, ws *store.Workspace, c store.Contact, out *ContactProfile) error {
	lists, err := s.Q.ListListsWithCounts(ctx, ws.ID)
	if err != nil {
		return err
	}
	members, err := s.Q.ContactMemberships(ctx, c.ID)
	if err != nil {
		return err
	}
	events, err := s.Q.ContactConsentEvents(ctx, c.ID)
	if err != nil {
		return err
	}
	status := map[uuid.UUID]string{}
	for _, m := range members {
		status[m.ListID] = m.Status
	}
	byList := map[uuid.UUID][]ContactConsentEvent{}
	for _, e := range events {
		if e.ListID == nil {
			continue
		}
		v := ContactConsentEvent{Action: e.Action, Source: e.Source, SourceLabel: consentSourceLabel(e.Source), Detail: e.Detail, Note: e.Note,
			Actor: e.ActorName, ConsentVersion: e.ConsentVersion, ConsentText: e.ConsentText, Language: e.Language, At: e.OccurredAt}
		if e.Ip != nil {
			v.IP = e.Ip.String()
		}
		byList[*e.ListID] = append(byList[*e.ListID], v)
	}
	for _, l := range lists {
		st := status[l.ID]
		if st == "" {
			st = "none"
		}
		ev := byList[l.ID]
		if ev == nil {
			ev = []ContactConsentEvent{}
		}
		out.Lists = append(out.Lists, ContactListConsent{ListID: l.ID.String(), Name: l.Name, Status: st, DoubleOptIn: l.DoubleOptIn, Events: ev})
	}
	return nil
}

// consentSourceLabel names where a consent event came from.
func consentSourceLabel(src string) string {
	switch src {
	case "preference_center":
		return "the preference center"
	case "one_click":
		return "List-Unsubscribe one-click"
	case "repermission":
		return "re-permission email"
	case "team":
		return "the team"
	case "api":
		return "the API"
	}
	if strings.HasPrefix(src, "import:") || strings.HasPrefix(src, "form:") || strings.HasPrefix(src, "product_checkout:") {
		return segments.SourceLabel(src)
	}
	return src
}

// ContactUpdate changes fields and tags. Omitted fields stay as they are;
// a custom field set to null is cleared.
type ContactUpdate struct {
	FirstName *string        `json:"firstName,omitempty" maxLength:"100"`
	LastName  *string        `json:"lastName,omitempty" maxLength:"100"`
	Company   *string        `json:"company,omitempty" maxLength:"200"`
	Country   *string        `json:"country,omitempty" maxLength:"2"`
	Timezone  *string        `json:"timezone,omitempty" maxLength:"64"`
	Language  *string        `json:"language,omitempty" maxLength:"40" doc:"A BCP 47 code such as es-MX; \"Spanish\" and \"es_MX\" are normalized. Empty clears it."`
	Fields    map[string]any `json:"fields,omitempty" doc:"Custom field key → value; null clears it"`
	Tags      []string       `json:"tags,omitempty" maxItems:"50"`
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// customValue checks a value against a custom field's type and returns
// what to store (nil clears).
func customValue(f store.CustomField, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	s := strings.TrimSpace(attrString(v))
	if s == "" {
		return nil, nil
	}
	switch f.Type {
	case "number":
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("%s is a number field; %q isn't a number", f.Key, s)
		}
		return n, nil
	case "date":
		if !datePattern.MatchString(s) {
			return nil, fmt.Errorf("%s is a date field; use a date like 2026-10-30", f.Key)
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return nil, fmt.Errorf("%s: %q isn't a real date", f.Key, s)
		}
		return s, nil
	case "boolean":
		b, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return nil, fmt.Errorf("%s is a yes/no field; use true or false", f.Key)
		}
		return b, nil
	case "select":
		var opts []string
		_ = json.Unmarshal(f.Options, &opts)
		if len(opts) > 0 {
			for _, o := range opts {
				if o == s {
					return s, nil
				}
			}
			return nil, fmt.Errorf("%s must be one of %s", f.Key, strings.Join(opts, ", "))
		}
	}
	if len(s) > 500 {
		return nil, fmt.Errorf("%s is limited to 500 characters", f.Key)
	}
	return s, nil
}

// UpdateContact changes a contact's fields and tags.
func (s *Service) UpdateContact(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ContactUpdate) (ContactProfile, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.getContact(ctx, q, ws, id)
		if err != nil {
			return err
		}
		p := store.UpdateContactFieldsParams{WorkspaceID: ws.ID, ID: id, FirstName: c.FirstName, LastName: c.LastName, Company: c.Company,
			Country: c.Country, Timezone: c.Timezone, Language: c.Language, Tags: c.Tags, Attributes: c.Attributes}
		set := func(dst *string, v *string) {
			if v != nil {
				*dst = strings.TrimSpace(*v)
			}
		}
		set(&p.FirstName, in.FirstName)
		set(&p.LastName, in.LastName)
		set(&p.Company, in.Company)
		if in.Language != nil {
			code, err := NormalizeLanguage(*in.Language, "body.language")
			if err != nil {
				return err
			}
			p.Language = code
		}
		if in.Country != nil {
			p.Country = strings.ToUpper(strings.TrimSpace(*in.Country))
		}
		if in.Timezone != nil {
			tz := strings.TrimSpace(*in.Timezone)
			if tz != "" {
				if _, err := time.LoadLocation(tz); err != nil {
					return api.Invalid("body.timezone", fmt.Sprintf("%q isn't a timezone. Use a name like America/Chicago.", tz))
				}
			}
			p.Timezone = tz
		}
		if in.Tags != nil {
			p.Tags = cleanTags(in.Tags)
		}
		if len(in.Fields) > 0 {
			defs, err := q.ListCustomFields(ctx, ws.ID)
			if err != nil {
				return err
			}
			byKey := map[string]store.CustomField{}
			for _, f := range defs {
				byKey[f.Key] = f
			}
			attrs := map[string]any{}
			_ = json.Unmarshal(c.Attributes, &attrs)
			for k, v := range in.Fields {
				f, ok := byKey[k]
				if !ok {
					return api.Invalid("body.fields."+k, fmt.Sprintf("%s isn't a custom field. Create it first under Fields.", k))
				}
				val, err := customValue(f, v)
				if err != nil {
					return api.Invalid("body.fields."+k, capitalize(err.Error())+".")
				}
				if val == nil {
					delete(attrs, k)
				} else {
					attrs[k] = val
				}
			}
			p.Attributes, _ = json.Marshal(attrs)
		}
		after, err := q.UpdateContactFields(ctx, p)
		if err != nil {
			return err
		}
		if after.Language != c.Language {
			source, name, by := languageSourceOf(ctx)
			if err := RecordLanguageSource(ctx, q, id, source, name, s.now(), by); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: "contact.update", TargetType: "contact", TargetID: id.String(), TargetName: c.Email,
			Before: contactFieldsSnapshot(c), After: contactFieldsSnapshot(after),
		})
	})
	if err != nil {
		return ContactProfile{}, err
	}
	return s.Profile(ctx, ws, id)
}

func contactFieldsSnapshot(c store.Contact) map[string]any {
	return map[string]any{"firstName": c.FirstName, "lastName": c.LastName, "company": c.Company, "country": c.Country,
		"timezone": c.Timezone, "language": c.Language, "tags": c.Tags, "fields": c.Attributes}
}

func cleanTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range in {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" || seen[t] || len(t) > 60 {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// NewContact is the "Add contact" dialog.
type NewContact struct {
	Email     string   `json:"email" minLength:"3" maxLength:"254"`
	FirstName string   `json:"firstName,omitempty" maxLength:"100"`
	LastName  string   `json:"lastName,omitempty" maxLength:"100"`
	Lists     []string `json:"lists" maxItems:"50" doc:"List ids"`
	// Consent: single adds them as subscribed now; double sends a
	// confirmation email and keeps them pending until they click it.
	Consent string   `json:"consent" enum:"single,double"`
	Tags    []string `json:"tags,omitempty" maxItems:"50"`
	// Language is a BCP 47 code ("es-MX"); "Spanish" and "es_MX" are
	// normalized, and a value that isn't a language answers 422 with the
	// closest match.
	Language string `json:"language,omitempty" maxLength:"40"`
}

type NewContactResult struct {
	ID string `json:"id"`
	// ConfirmationSent is true when a double opt-in email went out.
	ConfirmationSent bool   `json:"confirmationSent"`
	ConfirmURL       string `json:"confirmUrl,omitempty" doc:"Shown when no System provider could send the confirmation"`
}

// CreateContact adds one contact from the app.
func (s *Service) CreateContact(ctx context.Context, ws *store.Workspace, in NewContact) (NewContactResult, error) {
	email, problem := emailaddr.Check(in.Email)
	if problem != "" {
		return NewContactResult{}, api.Invalid("body.email", fmt.Sprintf("That isn't an email address (%s).", problem))
	}
	if len(in.Lists) == 0 {
		return NewContactResult{}, api.Invalid("body.lists", "Pick at least one list.")
	}
	language, lerr := NormalizeLanguage(in.Language, "body.language")
	if lerr != nil {
		return NewContactResult{}, lerr
	}
	if sup, err := s.Q.GetSuppressionByEmail(ctx, store.GetSuppressionByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		return NewContactResult{}, api.Conflict(fmt.Sprintf("%s is on the suppression list (%s). Remove it there first if they gave you a new opt-in.", email, reasonName(sup.Reason)))
	}
	if _, err := s.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		return NewContactResult{}, api.Conflict(fmt.Sprintf("%s is already a contact. Open their profile to change lists.", email))
	}
	listIDs := make([]uuid.UUID, 0, len(in.Lists))
	for _, l := range in.Lists {
		id, err := uuid.Parse(l)
		if err != nil {
			return NewContactResult{}, api.Invalid("body.lists", "Pick lists from this workspace.")
		}
		if _, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return NewContactResult{}, api.Invalid("body.lists", "Pick lists from this workspace.")
		}
		if !scopeAllowsList(ctx, id) {
			return NewContactResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, "Your role is limited to other lists.")
		}
		listIDs = append(listIDs, id)
	}
	double := in.Consent == "double"
	consent, status := "single", "subscribed"
	if double {
		consent, status = "pending", "pending"
	}
	domain := emailaddr.Domain(email)
	var c store.Contact
	actor := actorID(ctx)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		c, err = q.CreateContact(ctx, store.CreateContactParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: email, FirstName: strings.TrimSpace(in.FirstName), LastName: strings.TrimSpace(in.LastName),
			MailboxProvider: emailaddr.MailboxProvider(domain), Consent: consent, Source: "manual", Tags: cleanTags(in.Tags),
			Attributes: json.RawMessage(`{}`), RoleAddress: emailaddr.IsRole(email),
		})
		if err != nil {
			return err
		}
		if language != "" {
			source, name, by := languageSourceOf(ctx)
			if err := q.ContactSetLanguage(ctx, store.ContactSetLanguageParams{ID: c.ID, Language: language, Source: source, SourceName: name, At: s.now(), SetBy: by}); err != nil {
				return err
			}
		}
		for _, l := range listIDs {
			if err := q.UpsertMembership(ctx, store.UpsertMembershipParams{ListID: l, ContactID: c.ID, Status: status, Source: "team"}); err != nil {
				return err
			}
			action := "subscribed"
			if double {
				action = "optin_sent"
			}
			lid := l
			if err := q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, ListID: &lid, Action: action, Source: "team", ActorID: actor, OccurredAt: s.now()}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.create", TargetType: "contact", TargetID: c.ID.String(), TargetName: email,
			After: map[string]any{"email": email, "lists": in.Lists, "consent": in.Consent}})
	})
	if err != nil {
		return NewContactResult{}, err
	}
	out := NewContactResult{ID: c.ID.String()}
	if double {
		sent, url := s.sendConfirmation(ctx, ws, c)
		out.ConfirmationSent = sent
		if !sent {
			out.ConfirmURL = url
		}
	}
	return out, nil
}

// confirmURL is the double opt-in link of a contact.
func (s *Service) confirmURL(ws *store.Workspace, contactID uuid.UUID) string {
	if s.Links == nil {
		return ""
	}
	return s.URLs.Confirm(ws.Slug, s.Links.Sign(linkClaims(ws.ID, contactID, linkConfirm)))
}

// sendConfirmation emails the double opt-in link through the System stream.
// It reports whether the email went out and returns the link either way.
func (s *Service) sendConfirmation(ctx context.Context, ws *store.Workspace, c store.Contact) (bool, string) {
	url := s.confirmURL(ws, c.ID)
	if s.Mail == nil || !s.Mail.SystemAvailable(ctx, &ws.ID) {
		return false, url
	}
	name := strings.TrimSpace(c.FirstName)
	if name == "" {
		name = "there"
	}
	text := fmt.Sprintf("Hi %s,\n\nPlease confirm that you want to receive email from %s:\n\n%s\n\nIf you didn't ask for this, ignore this email and you won't hear from us.\n", name, ws.Name, url)
	if _, err := s.Mail.SendSystem(ctx, mailMessage(ws, c, "Confirm your subscription to "+ws.Name, text)); err != nil {
		return false, url
	}
	return true, url
}

// SendConfirmations emails the double opt-in link to contacts waiting for
// re-permission (imports). It returns how many went out; without a System
// provider nothing is sent and the contacts stay pending.
func (s *Service) SendConfirmations(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (int, error) {
	if s.Mail == nil || !s.Mail.SystemAvailable(ctx, &ws.ID) || len(ids) == 0 {
		return 0, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT * FROM contacts WHERE id = ANY ($1) AND erased_at IS NULL`, ids)
	if err != nil {
		return 0, err
	}
	contacts, err := pgx.CollectRows(rows, pgx.RowToStructByName[store.Contact])
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, c := range contacts {
		if ok, _ := s.sendConfirmation(ctx, ws, c); ok {
			sent++
			_ = s.Q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, Action: "optin_sent", Source: "repermission",
				Note: "confirmation sent via the System stream", OccurredAt: s.now()})
		}
	}
	return sent, nil
}

// OptInResendResult is the outcome of resending the opt-in email.
type OptInResendResult struct {
	Sent       bool   `json:"sent"`
	ConfirmURL string `json:"confirmUrl,omitempty"`
}

// ResendOptIn sends the confirmation email again and records it.
func (s *Service) ResendOptIn(ctx context.Context, ws *store.Workspace, id uuid.UUID) (OptInResendResult, error) {
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return OptInResendResult{}, err
	}
	if _, err := s.Q.GetSuppressionByEmail(ctx, store.GetSuppressionByEmailParams{WorkspaceID: ws.ID, Email: c.Email}); err == nil {
		return OptInResendResult{}, api.Conflict(c.Email + " is on the suppression list, so no opt-in email can be sent.")
	}
	sent, url := s.sendConfirmation(ctx, ws, c)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if sent {
			if err := q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, Action: "optin_sent", Source: "team",
				Note: "confirmation sent via the System stream", ActorID: actorID(ctx), OccurredAt: s.now()}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.resend_optin", TargetType: "contact", TargetID: c.ID.String(), TargetName: c.Email,
			Metadata: map[string]any{"sent": sent}})
	})
	if err != nil {
		return OptInResendResult{}, err
	}
	out := OptInResendResult{Sent: sent}
	if !sent {
		out.ConfirmURL = url
	}
	return out, nil
}

// PreferenceLink returns the preference center link of a contact ("Open
// preference center as Sam"). The link is logged.
func (s *Service) PreferenceLink(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return "", err
	}
	if s.Links == nil {
		return "", errors.New("link signing is not configured")
	}
	url := s.URLs.Preferences(ws.Slug, s.Links.Sign(linkClaims(ws.ID, c.ID, linkPreferences)))
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "contact.preference_link", TargetType: "contact", TargetID: c.ID.String(), TargetName: c.Email})
	return url, err
}

// ContactSuppressInput adds a contact to the suppression list by hand.
type ContactSuppressInput struct {
	Reason string `json:"reason" minLength:"1" maxLength:"300" doc:"Why, for the audit log and the list"`
}

// SuppressContact puts a contact's address on the shared suppression list.
func (s *Service) SuppressContact(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ContactSuppressInput) error {
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return err
	}
	res, err := s.addSuppressions(ctx, ws, []string{c.Email}, "manual", strings.TrimSpace(in.Reason))
	if err != nil {
		return err
	}
	if res.Added == 0 {
		return api.Conflict(c.Email + " is already on the suppression list.")
	}
	return nil
}
