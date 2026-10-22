package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Consent conflict codes: an action never adds someone back to a list they
// left, and never adds a suppressed address.
const (
	CodeUnsubscribed = "unsubscribed"
	CodeSuppressed   = "suppressed"
)

// ContactUpsertInput creates or updates a contact, matched by email.
type ContactUpsertInput struct {
	Email     string  `json:"email" minLength:"3" maxLength:"254" example:"rob@example.net"`
	FirstName *string `json:"firstName,omitempty" maxLength:"100"`
	LastName  *string `json:"lastName,omitempty" maxLength:"100"`
	Company   *string `json:"company,omitempty" maxLength:"200"`
	// Language is a BCP 47 code ("es-MX"); "Spanish" and "es_MX" are
	// accepted and normalized.
	Language *string `json:"language,omitempty" maxLength:"40"`
	// Fields sets custom fields by key; null clears one.
	Fields map[string]any `json:"fields,omitempty"`
	// Tags are added to the contact's tags.
	Tags []string `json:"tags,omitempty" maxItems:"50"`
	// Lists are list ids to add the contact to.
	Lists []string `json:"lists,omitempty" maxItems:"20"`
	// Consent: single adds them to the lists now; double sends a
	// confirmation email first. Without it, each list's own setting applies.
	Consent string `json:"consent,omitempty" enum:"single,double"`
}

// ContactListJoin is a list a contact is now on.
type ContactListJoin struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"subscribed,pending"`
}

// ContactUpsertResult is what an action did to a contact.
type ContactUpsertResult struct {
	ContactID string            `json:"contactId"`
	Email     string            `json:"email"`
	Created   bool              `json:"created"`
	Lists     []ContactListJoin `json:"lists"`
	Tags      []string          `json:"tags"`
	// ConfirmationSent: a double opt-in email went out.
	ConfirmationSent bool `json:"confirmationSent"`
}

// ContactListAddInput adds an existing contact to a list.
type ContactListAddInput struct {
	Email   string `json:"email" minLength:"3" maxLength:"254"`
	Consent string `json:"consent,omitempty" enum:"single,double"`
}

// ContactTagInput adds tags to an existing contact.
type ContactTagInput struct {
	Email string   `json:"email" minLength:"3" maxLength:"254"`
	Tags  []string `json:"tags" minItems:"1" maxItems:"50"`
}

func wsLocation(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// joinPlan is a list a contact will join.
type joinPlan struct {
	id     uuid.UUID
	name   string
	status string
}

// planJoins checks lists for a contact: they exist, the key's scope reaches
// them, and the contact didn't leave them.
func (s *Service) planJoins(ctx context.Context, ws *store.Workspace, p *authz.Principal, contact *store.Contact, email string, raw []string, consent string) ([]joinPlan, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	lists, err := s.Q.IntegrationListNames(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]store.IntegrationListNamesRow{}
	for _, l := range lists {
		byID[l.ID] = l
	}
	var out []joinPlan
	for _, r := range raw {
		id, err := uuid.Parse(strings.TrimSpace(r))
		l, ok := byID[id]
		if err != nil || !ok {
			return nil, api.Invalid("body.lists", problem(ctx, "problem.listUnknown", i18n.Args{"list": r}))
		}
		if !p.Scope.AllowsList(id) {
			return nil, api.Err(http.StatusForbidden, api.CodePermissionDenied, problem(ctx, "problem.listOutOfScope", i18n.Args{"list": l.Name}))
		}
		status := "subscribed"
		if consent == "double" || consent == "" && l.DoubleOptIn {
			status = "pending"
		}
		if contact != nil {
			m, err := s.Q.IntegrationMembership(ctx, store.IntegrationMembershipParams{ListID: id, ContactID: contact.ID})
			if err == nil && m.Status == "unsubscribed" {
				left := m.UnsubscribedAt
				if e, err := s.Q.IntegrationLastLeft(ctx, store.IntegrationLastLeftParams{ContactID: contact.ID, ListID: &id}); err == nil {
					left = &e.OccurredAt
				}
				args := i18n.Args{"email": email, "list": l.Name, "when": "earlier"}
				if left != nil {
					args["when"], args["at"] = "on", left.In(wsLocation(ws))
				}
				return nil, api.Err(http.StatusConflict, CodeUnsubscribed, problem(ctx, "problem.leftList", args)).
					WithMeta("list", l.Name, "listId", id.String(), "leftAt", left)
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			} else if err == nil && m.Status == "subscribed" {
				status = "subscribed"
			}
		}
		out = append(out, joinPlan{id: id, name: l.Name, status: status})
	}
	return out, nil
}

// refuseSuppressed answers 409 for a suppressed address being added.
func (s *Service) refuseSuppressed(ctx context.Context, ws *store.Workspace, email string) error {
	sup, err := s.Q.GetSuppressionByEmail(ctx, store.GetSuppressionByEmailParams{WorkspaceID: ws.ID, Email: email})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return api.Err(http.StatusConflict, CodeSuppressed, problem(ctx, "problem.suppressed",
		i18n.Args{"email": email, "at": sup.CreatedAt.In(wsLocation(ws)), "reason": sup.Reason, "reasonText": sup.Reason})).
		WithMeta("reason", sup.Reason, "since", sup.CreatedAt)
}

// emailProblems name emailaddr's problems for problem.notEmail.
var emailProblems = map[string]string{
	emailaddr.ProblemNoDomain: "noDomain", emailaddr.ProblemDoubleDot: "doubleDot", emailaddr.ProblemTooLong: "tooLong", emailaddr.ProblemNotAnAddress: "notAnAddress",
}

func cleanEmail(ctx context.Context, raw string) (string, error) {
	email, why := emailaddr.Check(raw)
	if why != "" {
		return "", api.Invalid("body.email", problem(ctx, "problem.notEmail", i18n.Args{"problem": emailProblems[why]}))
	}
	return email, nil
}

// UpsertContact creates or updates a contact matched by email, and adds it
// to lists without ever adding someone back to a list they left or adding a
// suppressed address.
func (s *Service) UpsertContact(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ContactUpsertInput) (ContactUpsertResult, error) {
	email, err := cleanEmail(ctx, in.Email)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	var existing *store.Contact
	if c, err := s.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		existing = &c
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ContactUpsertResult{}, err
	}
	if existing != nil && !p.Can(authz.SubscribersEdit) {
		return ContactUpsertResult{}, api.Forbidden(authz.SubscribersEdit, access.Helpers(ctx, s.Q, ws.ID, authz.SubscribersEdit))
	}
	if existing == nil && !p.Can(authz.SubscribersCreate) {
		return ContactUpsertResult{}, api.Forbidden(authz.SubscribersCreate, access.Helpers(ctx, s.Q, ws.ID, authz.SubscribersCreate))
	}
	joins, err := s.planJoins(ctx, ws, p, existing, email, in.Lists, in.Consent)
	if err != nil {
		s.noteContactCall(ctx, "upsert_contact", email, existing, in.Lists, in.Tags, in.Consent)
		return ContactUpsertResult{}, err
	}
	if existing == nil || len(joins) > 0 {
		if err := s.refuseSuppressed(ctx, ws, email); err != nil {
			s.noteContactCall(ctx, "upsert_contact", email, existing, in.Lists, in.Tags, in.Consent)
			return ContactUpsertResult{}, err
		}
	}
	language, err := normalizeLanguage(in.Language)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	attrs, err := s.fieldValues(ctx, ws, existing, in.Fields)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	out := ContactUpsertResult{Email: email, Created: existing == nil, Lists: []ContactListJoin{}}
	var contactID uuid.UUID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var before map[string]any
		if existing == nil {
			consent := "single"
			for _, j := range joins {
				if j.status == "pending" {
					consent = "pending"
				}
			}
			row, err := q.IntegrationContactInsert(ctx, store.IntegrationContactInsertParams{
				ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: email, FirstName: str(in.FirstName), LastName: str(in.LastName), Company: str(in.Company),
				Language: language, MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), Consent: consent, Source: "api",
				Tags: cleanTags(in.Tags, nil), Attributes: attrs, RoleAddress: emailaddr.IsRole(email), Now: s.now(),
			})
			if err != nil {
				return err
			}
			contactID = row.ID
			out.Tags = cleanTags(in.Tags, nil)
			if language != "" {
				if err := audience.RecordLanguageSource(ctx, q, contactID, audience.LanguageFromAPI, languageSetter(ctx), s.now(), nil); err != nil {
					return err
				}
			}
		} else {
			contactID = existing.ID
			before = contactSnapshot(*existing)
			lang := existing.Language
			if in.Language != nil {
				lang = language
			}
			out.Tags = cleanTags(in.Tags, existing.Tags)
			if err := q.IntegrationContactUpdate(ctx, store.IntegrationContactUpdateParams{ID: existing.ID, WorkspaceID: ws.ID,
				FirstName: strOr(in.FirstName, existing.FirstName), LastName: strOr(in.LastName, existing.LastName), Company: strOr(in.Company, existing.Company),
				Language: lang, Tags: out.Tags, Attributes: attrs, Now: s.now()}); err != nil {
				return err
			}
			if lang != existing.Language {
				if err := audience.RecordLanguageSource(ctx, q, contactID, audience.LanguageFromAPI, languageSetter(ctx), s.now(), nil); err != nil {
					return err
				}
			}
		}
		if err := s.join(ctx, q, contactID, joins, &out); err != nil {
			return err
		}
		action := "contact.create"
		if existing != nil {
			action = "contact.update"
		}
		after := map[string]any{"email": email, "lists": listNames(joins), "tags": out.Tags}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "contact", TargetID: contactID.String(), TargetName: email,
			Before: before, After: after, Metadata: s.keyMeta(ctx, map[string]any{"summary": "through " + integrationName(ctx)})})
	})
	if err != nil {
		return ContactUpsertResult{}, err
	}
	out.ContactID = contactID.String()
	s.noteContactCall(ctx, "upsert_contact", email, existing, in.Lists, in.Tags, in.Consent)
	api.NoteCall(ctx, noteContact, contactID.String())
	out.ConfirmationSent = s.confirm(ctx, ws, contactID, out.Lists)
	return out, nil
}

// join adds the contact to its lists and records consent.
func (s *Service) join(ctx context.Context, q *store.Queries, contactID uuid.UUID, joins []joinPlan, out *ContactUpsertResult) error {
	for _, j := range joins {
		if err := q.IntegrationJoinList(ctx, store.IntegrationJoinListParams{ListID: j.id, ContactID: contactID, Status: j.status, Source: "api", Now: s.now()}); err != nil {
			return err
		}
		action := "subscribed"
		if j.status == "pending" {
			action = "optin_sent"
		}
		lid := j.id
		if err := q.IntegrationConsentEvent(ctx, store.IntegrationConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: contactID, ListID: &lid,
			Action: action, Source: "api", Detail: integrationName(ctx), Note: consentNote(ctx, j.status), OccurredAt: s.now()}); err != nil {
			return err
		}
		out.Lists = append(out.Lists, ContactListJoin{ID: j.id.String(), Name: j.name, Status: j.status})
	}
	return nil
}

func consentNote(ctx context.Context, status string) string {
	n := "added through " + integrationName(ctx)
	if status == "pending" {
		n += "; waiting for the confirmation click"
	}
	return n
}

// confirm sends the double opt-in email when a list is pending.
func (s *Service) confirm(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, lists []ContactListJoin) bool {
	pending := false
	for _, l := range lists {
		if l.Status == "pending" {
			pending = true
		}
	}
	if !pending || s.Confirm == nil {
		return false
	}
	n, err := s.Confirm.SendConfirmations(ctx, ws, []uuid.UUID{contactID})
	return err == nil && n > 0
}

// AddToList adds an existing contact to a list, single or double opt-in.
func (s *Service) AddToList(ctx context.Context, ws *store.Workspace, p *authz.Principal, listID string, in ContactListAddInput) (ContactUpsertResult, error) {
	email, err := cleanEmail(ctx, in.Email)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	c, err := s.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: email})
	if errors.Is(err, pgx.ErrNoRows) {
		s.noteContactCall(ctx, "add_to_list", email, nil, []string{listID}, nil, in.Consent)
		return ContactUpsertResult{}, api.Err(http.StatusNotFound, api.CodeNotFound, problem(ctx, "problem.noContact", i18n.Args{"email": email}))
	}
	if err != nil {
		return ContactUpsertResult{}, err
	}
	joins, err := s.planJoins(ctx, ws, p, &c, email, []string{listID}, in.Consent)
	if err == nil {
		err = s.refuseSuppressed(ctx, ws, email)
	}
	s.noteContactCall(ctx, "add_to_list", email, &c, []string{listID}, nil, in.Consent)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	api.NoteCall(ctx, noteContact, c.ID.String())
	out := ContactUpsertResult{ContactID: c.ID.String(), Email: email, Lists: []ContactListJoin{}, Tags: nonNil(c.Tags)}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.join(ctx, q, c.ID, joins, &out); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.list_added", TargetType: "contact", TargetID: c.ID.String(), TargetName: email,
			After: map[string]any{"lists": listNames(joins)}, Metadata: s.keyMeta(ctx, map[string]any{"summary": "through " + integrationName(ctx)})})
	})
	if err != nil {
		return ContactUpsertResult{}, err
	}
	out.ConfirmationSent = s.confirm(ctx, ws, c.ID, out.Lists)
	return out, nil
}

// TagContact adds tags to an existing contact.
func (s *Service) TagContact(ctx context.Context, ws *store.Workspace, in ContactTagInput) (ContactUpsertResult, error) {
	email, err := cleanEmail(ctx, in.Email)
	if err != nil {
		return ContactUpsertResult{}, err
	}
	tags := cleanTags(in.Tags, nil)
	api.NoteCall(ctx, noteItem, "add_tag")
	noteUse(ctx, WorkflowUse{Item: "add_tag", Tags: cleanTags(tags, nil)})
	noteTargetJSON(ctx, CallTarget{Kind: "add_tag", Email: email, Tag: strings.Join(tags, ", ")})
	c, err := s.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: email})
	if errors.Is(err, pgx.ErrNoRows) {
		return ContactUpsertResult{}, api.Err(http.StatusNotFound, api.CodeNotFound, problem(ctx, "problem.noContact", i18n.Args{"email": email}))
	}
	if err != nil {
		return ContactUpsertResult{}, err
	}
	if len(tags) == 0 {
		return ContactUpsertResult{}, api.Invalid("body.tags", problem(ctx, "problem.noTag", nil))
	}
	api.NoteCall(ctx, noteContact, c.ID.String())
	merged := cleanTags(tags, c.Tags)
	out := ContactUpsertResult{ContactID: c.ID.String(), Email: email, Lists: []ContactListJoin{}, Tags: merged}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationContactUpdate(ctx, store.IntegrationContactUpdateParams{ID: c.ID, WorkspaceID: ws.ID, FirstName: c.FirstName, LastName: c.LastName,
			Company: c.Company, Language: c.Language, Tags: merged, Attributes: c.Attributes, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact.update", TargetType: "contact", TargetID: c.ID.String(), TargetName: email,
			Before: map[string]any{"tags": nonNil(c.Tags)}, After: map[string]any{"tags": merged},
			Metadata: s.keyMeta(ctx, map[string]any{"summary": "tagged " + strings.Join(tags, ", ") + " through " + integrationName(ctx)})})
	})
	return out, err
}

// noteContactCall describes a contact action for the call log and the
// workflow list: its lists, tags and opt-in.
func (s *Service) noteContactCall(ctx context.Context, item, email string, c *store.Contact, listIDs, tags []string, consent string) {
	names := []string{}
	if lists, err := s.Q.IntegrationListNames(ctx, api.WorkspaceFrom(ctx).ID); err == nil {
		for _, raw := range listIDs {
			for _, l := range lists {
				if l.ID.String() == strings.TrimSpace(raw) {
					names = append(names, l.Name)
				}
			}
		}
	}
	use := WorkflowUse{Item: item, Lists: names, Tags: cleanTags(tags, nil)}
	if consent == "double" || consent == "single" {
		use.OptIn = consent
	}
	noteUse(ctx, use)
	noteTargetJSON(ctx, CallTarget{Kind: item, Email: email, List: strings.Join(names, ", ")})
	if c != nil {
		api.NoteCall(ctx, noteContact, c.ID.String())
	}
}

func integrationName(ctx context.Context) string {
	if ik := api.IntegrationFrom(ctx); ik != nil {
		return ik.Name
	}
	return "the API"
}

// languageSetter names who set a contact's language through the API: the
// integration ("Zapier"), or the API key.
func languageSetter(ctx context.Context) string {
	if ik := api.IntegrationFrom(ctx); ik != nil {
		return ik.Name
	}
	if p := api.PrincipalFrom(ctx); p != nil {
		return p.Name
	}
	return ""
}

func listNames(joins []joinPlan) []string {
	out := []string{}
	for _, j := range joins {
		out = append(out, j.name)
	}
	return out
}

func contactSnapshot(c store.Contact) map[string]any {
	return map[string]any{"firstName": c.FirstName, "lastName": c.LastName, "company": c.Company, "language": c.Language, "tags": nonNil(c.Tags), "fields": c.Attributes}
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func strOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return strings.TrimSpace(*p)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// cleanTags lowercases and trims tags, merging them into existing ones.
func cleanTags(in, existing []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range append(append([]string{}, existing...), in...) {
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

// fieldValues merges custom field values into the contact's.
func (s *Service) fieldValues(ctx context.Context, ws *store.Workspace, c *store.Contact, in map[string]any) (json.RawMessage, error) {
	attrs := map[string]any{}
	if c != nil {
		_ = json.Unmarshal(c.Attributes, &attrs)
	}
	if len(in) > 0 {
		defs, err := s.Q.IntegrationCustomFields(ctx, ws.ID)
		if err != nil {
			return nil, err
		}
		byKey := map[string]store.IntegrationCustomFieldsRow{}
		for _, f := range defs {
			byKey[f.Key] = f
		}
		for k, v := range in {
			f, ok := byKey[k]
			if !ok {
				return nil, api.Invalid("body.fields."+k, problem(ctx, "problem.fieldUnknown", i18n.Args{"field": k}))
			}
			val, bad := fieldValue(f, v)
			if bad != nil {
				return nil, api.Invalid("body.fields."+k, problem(ctx, bad.key, bad.args))
			}
			if val == nil {
				delete(attrs, k)
			} else {
				attrs[k] = val
			}
		}
	}
	return json.Marshal(attrs)
}

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// badValue is why a custom field refuses a value: a problem's message and
// its arguments. The field's key starts each sentence, with a capital.
type badValue struct {
	key  string
	args i18n.Args
}

func fieldValue(f store.IntegrationCustomFieldsRow, v any) (any, *badValue) {
	if v == nil {
		return nil, nil
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return nil, nil
	}
	field := capitalize(f.Key)
	switch f.Type {
	case "number":
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, &badValue{"problem.fieldNumber", i18n.Args{"field": field, "value": s}}
		}
		return n, nil
	case "date":
		if !datePattern.MatchString(s) {
			return nil, &badValue{"problem.fieldDate", i18n.Args{"field": field}}
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return nil, &badValue{"problem.fieldRealDate", i18n.Args{"field": field, "value": s}}
		}
		return s, nil
	case "boolean":
		b, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return nil, &badValue{"problem.fieldBoolean", i18n.Args{"field": field}}
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
			// The choices are the workspace's own words, listed as data.
			return nil, &badValue{"problem.fieldSelect", i18n.Args{"field": field, "options": strings.Join(opts, ", ")}}
		}
	}
	return s, nil
}

// Lists returns the lists the key's scope reaches, for the apps' dropdowns.
func (s *Service) Lists(ctx context.Context, ws *store.Workspace, p *authz.Principal) (IntegrationListItems, error) {
	rows, err := s.Q.IntegrationListNames(ctx, ws.ID)
	if err != nil {
		return IntegrationListItems{}, err
	}
	noteTargetJSON(ctx, CallTarget{Kind: "lists"})
	out := IntegrationListItems{Items: []IntegrationListItem{}}
	for _, l := range rows {
		if p.Scope.AllowsList(l.ID) {
			out.Items = append(out.Items, IntegrationListItem{ID: l.ID.String(), Name: l.Name, DoubleOptIn: l.DoubleOptIn})
		}
	}
	return out, nil
}

// Fields returns the custom fields, for the apps' field mapping.
func (s *Service) Fields(ctx context.Context, ws *store.Workspace) (IntegrationFieldItems, error) {
	rows, err := s.Q.IntegrationCustomFields(ctx, ws.ID)
	if err != nil {
		return IntegrationFieldItems{}, err
	}
	noteTargetJSON(ctx, CallTarget{Kind: "fields"})
	out := IntegrationFieldItems{Items: []IntegrationFieldItem{}}
	for _, f := range rows {
		out.Items = append(out.Items, IntegrationFieldItem{Key: f.Key, Label: f.Label, Type: f.Type})
	}
	return out, nil
}

// IntegrationTemplateVariable is one variable of a transactional template,
// as the apps map it.
type IntegrationTemplateVariable struct {
	Name        string `json:"name" example:"amount_due"`
	Type        string `json:"type" example:"text"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive" doc:"Masked in the call log, never stored"`
	Description string `json:"description,omitempty"`
}

// IntegrationTemplateItem is a published transactional template.
type IntegrationTemplateItem struct {
	Key       string                        `json:"key" example:"payment-failed" doc:"The template's ID, for the send call"`
	Name      string                        `json:"name"`
	Version   int32                         `json:"version" doc:"The live version"`
	Variables []IntegrationTemplateVariable `json:"variables"`
}

// IntegrationTemplateItems lists the templates "Send a transactional email" can send.
type IntegrationTemplateItems struct {
	Items []IntegrationTemplateItem `json:"items"`
}

// Templates lists the published transactional templates and their
// variables, for "Send a transactional email" in the apps.
func (s *Service) Templates(ctx context.Context, ws *store.Workspace) (IntegrationTemplateItems, error) {
	rows, err := s.Q.IntegrationTransactionalTemplates(ctx, ws.ID)
	if err != nil {
		return IntegrationTemplateItems{}, err
	}
	noteTargetJSON(ctx, CallTarget{Kind: "templates"})
	out := IntegrationTemplateItems{Items: []IntegrationTemplateItem{}}
	for _, r := range rows {
		it := IntegrationTemplateItem{Key: r.Key, Name: r.Name, Variables: []IntegrationTemplateVariable{}}
		if r.LiveVersion != nil {
			it.Version = *r.LiveVersion
		}
		_ = json.Unmarshal(r.Variables, &it.Variables)
		if it.Variables == nil {
			it.Variables = []IntegrationTemplateVariable{}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// normalizeLanguage turns "es_MX", "ES" or "Spanish" into a BCP 47 code
// (audience.NormalizeLanguage). An unknown value answers 422 with the
// closest match when there is one.
func normalizeLanguage(p *string) (string, error) {
	if p == nil {
		return "", nil
	}
	return audience.NormalizeLanguage(*p, "body.language")
}
