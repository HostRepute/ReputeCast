package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// ComposerSenderOption is a sender the campaign can come from.
type ComposerSenderOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Domain   string `json:"domain"`
	Verified bool   `json:"verified"`
}

// ComposerAudienceOption is a list or segment with its size.
type ComposerAudienceOption struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count *int64 `json:"count,omitempty"`
}

// ComposerTestRecipient is a workspace member offered for test emails.
type ComposerTestRecipient struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ComposerContext is the workspace data the composer needs once.
type ComposerContext struct {
	Senders         []ComposerSenderOption        `json:"senders"`
	DefaultSenderID string                        `json:"defaultSenderId,omitempty"`
	Lists           []ComposerAudienceOption      `json:"lists"`
	Segments        []ComposerAudienceOption      `json:"segments"`
	Tokens          []liquid.PersonalizationToken `json:"tokens"`
	Members         []ComposerTestRecipient       `json:"members"`
	BroadcastRoute  ComposerRoute                 `json:"broadcastRoute"`
	// PostalAddress is the workspace's address for footers, if set.
	PostalAddress string `json:"postalAddress"`
	// PublicURL is where images and links point once an email leaves ReputeCast.
	PublicURL     string   `json:"publicUrl"`
	SenderDomains []string `json:"senderDomains"`
}

// Context returns senders, lists, segments, tokens, team and the route.
func (s *Service) Context(ctx context.Context, ws *store.Workspace, p *authz.Principal) (ComposerContext, error) {
	out := ComposerContext{Senders: []ComposerSenderOption{}, Lists: []ComposerAudienceOption{}, Segments: []ComposerAudienceOption{}, Tokens: []liquid.PersonalizationToken{}, Members: []ComposerTestRecipient{},
		SenderDomains: []string{}, PostalAddress: templates.WorkspaceAddress(ws), PublicURL: s.URLs.Base()}
	senders, err := s.Q.ListComposerSenders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	seenDomain := map[string]bool{}
	for _, sn := range senders {
		if !p.Scope.AllowsSenderDomain(sn.Domain) {
			continue
		}
		out.Senders = append(out.Senders, ComposerSenderOption{ID: sn.ID.String(), Name: sn.Name, Email: sn.Email, Domain: sn.Domain, Verified: sn.DomainStatus == "verified"})
		if !seenDomain[sn.Domain] {
			seenDomain[sn.Domain] = true
			out.SenderDomains = append(out.SenderDomains, sn.Domain)
		}
	}
	if def, err := s.Q.ComposerDefaultSender(ctx, ws.ID); err == nil {
		out.DefaultSenderID = def.SenderID
	}
	lists, err := s.Q.ComposerAudienceLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		if !p.Scope.AllowsList(l.ID) {
			continue
		}
		n := l.Subscribers
		out.Lists = append(out.Lists, ComposerAudienceOption{ID: l.ID.String(), Name: l.Name, Count: &n})
	}
	segs, err := s.Q.ComposerAudienceSegments(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sg := range segs {
		out.Segments = append(out.Segments, ComposerAudienceOption{ID: sg.ID.String(), Name: sg.Name, Count: sg.CachedCount})
	}
	if out.Tokens, err = s.tokens(ctx, ws.ID); err != nil {
		return out, err
	}
	members, err := s.Q.ComposerTestRecipients(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, m := range members {
		out.Members = append(out.Members, ComposerTestRecipient{Name: m.Name, Email: m.Email})
	}
	if out.BroadcastRoute, err = s.broadcastRoute(ctx, ws.ID); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) tokens(ctx context.Context, wsID uuid.UUID) ([]liquid.PersonalizationToken, error) {
	fields, err := s.Q.ComposerCustomFields(ctx, wsID)
	if err != nil {
		return nil, err
	}
	custom := make([]liquid.CustomField, 0, len(fields))
	for _, f := range fields {
		custom = append(custom, liquid.CustomField{Key: f.Key, Label: f.Label, Type: f.Type})
	}
	return liquid.Catalog(custom), nil
}

// ComposerAudienceCount is the live count of an audience definition.
type ComposerAudienceCount struct {
	Count int64 `json:"count"`
	// Approximate is true when segments are involved: their last cached
	// count is used, so overlaps may be counted twice until Send Readiness
	// computes the final number.
	Approximate bool `json:"approximate"`
}

// CountAudience counts the recipients of a definition.
func (s *Service) CountAudience(ctx context.Context, ws *store.Workspace, p *authz.Principal, def ComposerAudience) (ComposerAudienceCount, error) {
	def = normalizeAudience(def)
	if err := s.checkAudience(ctx, s.Q, ws, p, def); err != nil {
		return ComposerAudienceCount{}, err
	}
	return s.countAudience(ctx, s.Q, ws.ID, def)
}

func (s *Service) countAudience(ctx context.Context, q *store.Queries, wsID uuid.UUID, def ComposerAudience) (ComposerAudienceCount, error) {
	if def.empty() {
		return ComposerAudienceCount{}, nil
	}
	ids := func(in []string) []uuid.UUID {
		out := make([]uuid.UUID, 0, len(in))
		for _, s := range in {
			if u, err := uuid.Parse(s); err == nil {
				out = append(out, u)
			}
		}
		return out
	}
	var out ComposerAudienceCount
	if len(def.Lists) > 0 {
		n, err := q.ComposerCountListAudience(ctx, store.ComposerCountListAudienceParams{WorkspaceID: wsID, Lists: ids(def.Lists), ExcludeLists: ids(def.ExcludeLists)})
		if err != nil {
			return out, err
		}
		out.Count = n
	}
	if len(def.Segments) > 0 {
		segs, err := q.ComposerAudienceSegments(ctx, wsID)
		if err != nil {
			return out, err
		}
		want := map[string]bool{}
		for _, id := range def.Segments {
			want[id] = true
		}
		for _, sg := range segs {
			if want[sg.ID.String()] && sg.CachedCount != nil {
				out.Count += *sg.CachedCount
			}
		}
		out.Approximate = true
	}
	return out, nil
}

// ComposerPreviewRequest asks for the email as one recipient sees it. Subject,
// preheader and HTML may carry unsaved edits.
type ComposerPreviewRequest struct {
	ContactID string  `json:"contactId,omitempty" format:"uuid"`
	Sample    string  `json:"sample,omitempty" enum:"contact,no_first_name,random" default:"contact"`
	Variant   string  `json:"variant,omitempty" enum:"A,B,C"`
	Subject   *string `json:"subject,omitempty" maxLength:"998"`
	Preheader *string `json:"preheader,omitempty" maxLength:"500"`
	HTML      *string `json:"html,omitempty" maxLength:"2500000"`
	// Language previews a language version. Empty: the source, or, when
	// previewing as a contact, the version that contact gets.
	Language string `json:"language,omitempty" maxLength:"16"`
	// Editing is the version the subject, preheader and HTML above belong
	// to (the one the composer has open; empty is the source). Previewing as
	// a contact who gets another version shows that version as saved.
	Editing string `json:"editing,omitempty" maxLength:"16"`
}

// ComposerPreviewContact is who the preview is personalized for.
type ComposerPreviewContact struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Label describes the choice ("first_name set", "fallback "there"").
	Label string `json:"label"`
	// Language is the contact's language code ("es-MX"), empty when unset.
	Language string `json:"language"`
}

// ComposerTokenValue is what one token printed.
type ComposerTokenValue struct {
	Token        string `json:"token" example:"{{ first_name | default: \"there\" }}"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	FallbackUsed bool   `json:"fallbackUsed"`
	PerRecipient bool   `json:"perRecipient" doc:"A link filled in per recipient at send time"`
}

// ComposerPreview is the rendered email for one recipient.
type ComposerPreview struct {
	Subject   string                  `json:"subject"`
	Preheader string                  `json:"preheader"`
	HTML      string                  `json:"html"`
	Text      string                  `json:"text"`
	FromName  string                  `json:"fromName"`
	FromEmail string                  `json:"fromEmail"`
	Contact   *ComposerPreviewContact `json:"contact,omitempty"`
	Tokens    []ComposerTokenValue    `json:"tokens"`
	// MissingFirstName counts audience contacts with no first name (they see the fallback).
	MissingFirstName *int64                 `json:"missingFirstName,omitempty"`
	Problems         []liquid.LiquidProblem `json:"problems"`
	RenderedAt       time.Time              `json:"renderedAt"`
	// Language is the language version shown, and LanguageReason why when
	// it was picked for the contact (match, no_language, not_offered,
	// missing, draft, held). Empty without content languages.
	Language       string `json:"language"`
	LanguageReason string `json:"languageReason"`
}

var systemLinks = map[string]bool{"unsubscribe_url": true, "preferences_url": true, "view_in_browser_url": true}

// Preview renders the campaign for a contact (or a sample).
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerPreviewRequest) (ComposerPreview, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerPreview{}, api.MapError(err, "Campaign")
	}
	subject, preheader, html, abRaw := c.Subject, c.Preheader, c.Html, c.AbTest
	contact, attrs, err := s.previewContact(ctx, ws, p, c.Audience, in)
	if err != nil {
		return ComposerPreview{}, err
	}
	src := s.sourceLanguage(ctx, ws, id)
	lang, reason := src, ""
	switch {
	case in.Language != "":
		lang = in.Language
	case src != "" && contact != nil && contact.ID != "":
		if l, r, err := s.Translations.Choose(ctx, ws, id, contact.Language); err == nil {
			lang, reason = l, r
		}
	}
	if src != "" && lang != src {
		t, err := s.Translations.Content(ctx, ws, id, lang)
		if err != nil {
			return ComposerPreview{}, err
		}
		subject, preheader, html, abRaw = t.Subject, t.Preheader, t.HTML, t.ABTest
	}
	if ab := abFromDB(abRaw); ab != nil && in.Variant != "" && in.Variant != "A" {
		for _, v := range ab.Variants {
			if v.Key == in.Variant {
				if v.Subject != "" {
					subject = v.Subject
				}
				if v.Preheader != "" {
					preheader = v.Preheader
				}
				if v.HTML != "" {
					html = v.HTML
				}
			}
		}
	}
	// Unsaved edits belong to the version the composer has open.
	edited := in.Editing
	if edited == "" {
		edited = src
	}
	if src == "" || lang == edited {
		if in.Subject != nil {
			subject = *in.Subject
		}
		if in.Preheader != nil {
			preheader = *in.Preheader
		}
		if in.HTML != nil {
			html = *in.HTML
		}
	}
	links := map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"}
	first, last := "", ""
	if contact != nil {
		first, last = contactNames(contact.Name)
	}
	email := ""
	if contact != nil {
		email = contact.Email
	}
	vars := liquid.Vars(email, first, last, attrs, links)
	out := ComposerPreview{Contact: contact, Tokens: []ComposerTokenValue{}, Problems: []liquid.LiquidProblem{}, RenderedAt: s.now(), FromName: c.SenderName, FromEmail: c.SenderEmail,
		Language: lang, LanguageReason: reason}
	render := func(src string) string {
		r, err := liquid.Render(src, vars)
		if err != nil {
			out.Problems = append(out.Problems, liquid.LiquidProblem{Message: err.Error()})
			return src
		}
		return r
	}
	out.Subject = render(subject)
	out.Preheader = render(preheader)
	out.HTML = media.Absolutize(render(html), s.URLs.Base())
	out.Text = contentcheck.PlainText(out.HTML)
	for _, u := range liquid.Uses(subject + "\n" + preheader + "\n" + html) {
		tv := ComposerTokenValue{Key: u.Key, Token: tokenText(u), PerRecipient: systemLinks[u.Key]}
		if tv.PerRecipient {
			tv.Value = "per-recipient link"
		} else {
			v := fmt.Sprint(lookup(vars, u.Key))
			if strings.TrimSpace(v) == "" || v == "<nil>" {
				tv.Value, tv.FallbackUsed = u.Fallback, true
			} else {
				tv.Value = v
			}
		}
		out.Tokens = append(out.Tokens, tv)
	}
	if lists := audienceLists(c.Audience); len(lists) > 0 && p.Can(authz.SubscribersView) {
		if n, err := s.Q.ComposerCountMissingField(ctx, store.ComposerCountMissingFieldParams{WorkspaceID: ws.ID, Lists: lists}); err == nil {
			out.MissingFirstName = &n
		}
	}
	return out, nil
}

func tokenText(u liquid.Use) string {
	if u.HasFallback {
		return fmt.Sprintf("{{ %s | default: %q }}", u.Key, u.Fallback)
	}
	return "{{ " + u.Key + " }}"
}

func lookup(vars map[string]any, key string) any {
	parts := strings.Split(key, ".")
	var cur any = vars
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

func contactNames(full string) (string, string) {
	first, last, _ := strings.Cut(full, " ")
	return first, last
}

func audienceLists(raw json.RawMessage) []uuid.UUID {
	a := audienceFromDB(raw)
	out := []uuid.UUID{}
	for _, id := range a.Lists {
		if u, err := uuid.Parse(id); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// previewContact picks the contact to personalize for. Without
// subscribers:view only the synthetic samples are available.
func (s *Service) previewContact(ctx context.Context, ws *store.Workspace, p *authz.Principal, audience json.RawMessage, in ComposerPreviewRequest) (*ComposerPreviewContact, map[string]any, error) {
	if in.Sample == "no_first_name" {
		return &ComposerPreviewContact{Name: "", Email: "someone@example.com", Label: "Contact without first_name · fallbacks"}, map[string]any{}, nil
	}
	if !p.Can(authz.SubscribersView) {
		if in.ContactID != "" || in.Sample == "random" {
			return nil, nil, api.Forbidden(authz.SubscribersView, s.helpers(ctx, ws.ID, authz.SubscribersView))
		}
		return &ComposerPreviewContact{Name: "Sam Rivera", Email: "sam@example.com", Label: "Sample contact"}, map[string]any{}, nil
	}
	var row store.ComposerPreviewContactByIDRow
	var err error
	switch {
	case in.ContactID != "":
		cid, perr := uuid.Parse(in.ContactID)
		if perr != nil {
			return nil, nil, api.NotFound("Contact")
		}
		row, err = s.Q.ComposerPreviewContactByID(ctx, store.ComposerPreviewContactByIDParams{WorkspaceID: ws.ID, ID: cid})
	case in.Sample == "random":
		var r store.ComposerRandomContactRow
		r, err = s.Q.ComposerRandomContact(ctx, ws.ID)
		row = store.ComposerPreviewContactByIDRow(r)
	default:
		var rows []store.ComposerPreviewContactsRow
		rows, err = s.Q.ComposerPreviewContacts(ctx, store.ComposerPreviewContactsParams{WorkspaceID: ws.ID, Query: "", Lists: audienceLists(audience), Lim: 1})
		if err == nil && len(rows) == 0 {
			err = pgx.ErrNoRows
		}
		if err == nil {
			row = store.ComposerPreviewContactByIDRow(rows[0])
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if in.ContactID != "" {
			return nil, nil, api.NotFound("Contact")
		}
		return &ComposerPreviewContact{Name: "Sam Rivera", Email: "sam@example.com", Label: "Sample contact · no contacts in this audience yet"}, map[string]any{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	attrs := map[string]any{}
	_ = json.Unmarshal(row.Attributes, &attrs)
	name := strings.TrimSpace(row.FirstName + " " + row.LastName)
	label := "first_name set"
	if row.FirstName == "" {
		label = "no first_name · fallback"
	}
	return &ComposerPreviewContact{ID: row.ID.String(), Name: name, Email: row.Email, Label: label, Language: row.Language}, attrs, nil
}

// ComposerPreviewContactList is who the preview can be personalized for.
type ComposerPreviewContactList struct {
	Items []ComposerPreviewContact `json:"items"`
}

// PreviewContacts searches contacts of the audience.
func (s *Service) PreviewContacts(ctx context.Context, ws *store.Workspace, id uuid.UUID, query string) (ComposerPreviewContactList, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerPreviewContactList{}, api.MapError(err, "Campaign")
	}
	rows, err := s.Q.ComposerPreviewContacts(ctx, store.ComposerPreviewContactsParams{WorkspaceID: ws.ID, Query: strings.TrimSpace(query), Lists: audienceLists(c.Audience), Lim: 20})
	if err != nil {
		return ComposerPreviewContactList{}, err
	}
	out := ComposerPreviewContactList{Items: make([]ComposerPreviewContact, 0, len(rows))}
	for _, r := range rows {
		label := "first_name set"
		if r.FirstName == "" {
			label = "no first_name"
		}
		out.Items = append(out.Items, ComposerPreviewContact{ID: r.ID.String(), Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Email: r.Email, Label: label, Language: r.Language})
	}
	return out, nil
}

// ComposerTestSend sends the saved content to a few addresses.
type ComposerTestSend struct {
	To        []string `json:"to" minItems:"1" maxItems:"10"`
	Variant   string   `json:"variant,omitempty" enum:"A,B,C"`
	ContactID string   `json:"contactId,omitempty" format:"uuid" doc:"Personalize as this contact"`
	// Language sends one language version (the source when empty);
	// EachLanguage sends one email per language that would be sent.
	Language     string `json:"language,omitempty" maxLength:"16"`
	EachLanguage bool   `json:"eachLanguage,omitempty"`
}

// ComposerTestSendResult says where the test went.
type ComposerTestSendResult struct {
	Sent     []string `json:"sent"`
	Provider string   `json:"provider"`
	// Languages are the versions sent to each address.
	Languages []string `json:"languages"`
}

// SendTest delivers test emails through the System route. Content Writers
// and other members without campaigns:send can only send to themselves.
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerTestSend) (ComposerTestSendResult, error) {
	to := make([]string, 0, len(in.To))
	seen := map[string]bool{}
	for _, raw := range in.To {
		addr, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return ComposerTestSendResult{}, api.Invalid("body.to", fmt.Sprintf("“%s” isn't an email address.", raw))
		}
		e := strings.ToLower(addr.Address)
		if seen[e] {
			continue
		}
		seen[e] = true
		if !strings.EqualFold(e, p.Email) && !p.Can(authz.CampaignsSend) {
			return ComposerTestSendResult{}, api.Forbidden(authz.CampaignsSend, s.helpers(ctx, ws.ID, authz.CampaignsSend)).WithMeta("reason", "Without campaigns:send, test emails can only go to your own address.")
		}
		to = append(to, e)
	}
	langs := []string{in.Language}
	if in.EachLanguage && s.Translations != nil {
		ready, err := s.Translations.Ready(ctx, ws, id)
		if err != nil {
			return ComposerTestSendResult{}, err
		}
		langs = ready
	}
	if len(langs) == 0 || (in.Language == "" && !in.EachLanguage) {
		langs = []string{s.sourceLanguage(ctx, ws, id)}
	}
	links := map[string]string{
		"#unsubscribe":     s.URLs.Unsubscribe(ws.Slug, "test"),
		"#preferences":     s.URLs.Preferences(ws.Slug, "test"),
		"#view-in-browser": s.URLs.ViewOnline(ws.Slug, "test"),
	}
	type testEmail struct{ subject, html string }
	emails := make([]testEmail, 0, len(langs))
	for _, lang := range langs {
		pv, err := s.Preview(ctx, ws, p, id, ComposerPreviewRequest{ContactID: in.ContactID, Variant: in.Variant, Language: lang})
		if err != nil {
			return ComposerTestSendResult{}, err
		}
		html := pv.HTML
		for k, v := range links {
			html = strings.ReplaceAll(html, `href="`+k+`"`, `href="`+v+`"`)
		}
		emails = append(emails, testEmail{pv.Subject, html})
	}
	sender := s.mailer()
	if !sender.SystemAvailable(ctx, &ws.ID) {
		return ComposerTestSendResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable,
			"Test emails go out through the System route, and no sending provider can deliver them yet. Connect one in Sending providers, or check the email in Preview.")
	}
	provider := ""
	for _, addr := range to {
		for _, e := range emails {
			name, err := sender.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: addr, Subject: "[Test] " + e.subject, HTML: e.html, Text: contentcheck.PlainText(e.html), Kind: "campaign_test"})
			if err != nil {
				if errors.Is(err, mailer.ErrNoSystemProvider) {
					return ComposerTestSendResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable,
						"Test emails go out through the System route, and no sending provider can deliver them yet. Connect one in Sending providers, or check the email in Preview.")
				}
				return ComposerTestSendResult{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, "The sending provider refused the test email: "+err.Error())
			}
			provider = name
		}
	}
	sent := []string{}
	for _, l := range langs {
		if l != "" {
			sent = append(sent, l)
		}
	}
	after := map[string]any{"to": to, "variant": in.Variant, "provider": provider}
	if len(sent) > 0 {
		after["languages"] = sent
	}
	err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.test_send", TargetType: "campaign", TargetID: id.String(), After: after})
	return ComposerTestSendResult{Sent: to, Provider: provider, Languages: sent}, err
}

func (s *Service) mailer() mailer.Sender {
	if s.Mail != nil {
		if m := s.Mail(); m != nil {
			return m
		}
	}
	return mailer.Unavailable{}
}

// ComposerChecksRequest is content to check (unsaved edits included).
type ComposerChecksRequest struct {
	Subject    string `json:"subject,omitempty" maxLength:"998"`
	Preheader  string `json:"preheader,omitempty" maxLength:"500"`
	MJML       string `json:"mjml,omitempty" maxLength:"2000000"`
	HTML       string `json:"html,omitempty" maxLength:"2500000"`
	ProbeLinks bool   `json:"probeLinks,omitempty" doc:"Also request every link (slower; Send Readiness does it)"`
	// Language is the language version checked: subject heuristics look for
	// English phrases only in English.
	Language string `json:"language,omitempty" maxLength:"16"`
}

// ComposerChecks is a content report plus what the composer shows around it.
type ComposerChecks struct {
	contentcheck.ContentCheckReport
	// FooterRemovedAt is when the version without a footer was saved, if the
	// footer existed in an earlier version.
	FooterRemovedAt *time.Time `json:"footerRemovedAt,omitempty"`
	// Lint lists MJML layout warnings (image wider than its column…).
	Lint []mjml.MJMLProblem `json:"lint"`
}

// Checks runs the content checks on the composer's current content.
func (s *Service) Checks(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ComposerChecksRequest) (ComposerChecks, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerChecks{}, api.MapError(err, "Campaign")
	}
	tokens, err := s.tokens(ctx, ws.ID)
	if err != nil {
		return ComposerChecks{}, err
	}
	fields := make([]string, 0, len(tokens))
	for _, t := range tokens {
		fields = append(fields, t.Key)
	}
	var domains []string
	if c.SenderEmail != "" {
		if _, d, ok := strings.Cut(c.SenderEmail, "@"); ok {
			domains = append(domains, d)
		}
	}
	checker := s.Checker
	if checker == nil {
		checker = &contentcheck.Checker{}
	}
	var settings ComposerContentSettings
	_ = json.Unmarshal(c.ContentSettings, &settings)
	html := in.HTML
	if settings.Minify && html != "" {
		html = contentcheck.Minify(html)
	}
	r, err := checker.Check(ctx, contentcheck.Input{Subject: in.Subject, Preheader: in.Preheader, MJML: in.MJML, HTML: html, OwnDomains: domains,
		Address: templates.WorkspaceAddress(ws), Fields: fields, ProbeLinks: in.ProbeLinks, Language: in.Language})
	if err != nil {
		return ComposerChecks{}, err
	}
	out := ComposerChecks{ContentCheckReport: r, Lint: mjml.Lint(in.MJML)}
	for _, is := range r.Issues {
		if is.ID == "compliance.footer" {
			out.FooterRemovedAt = s.footerRemovedAt(ctx, id)
		}
	}
	return out, nil
}

// footerRemovedAt finds when the footer disappeared: the first version
// without one after a version that had it.
func (s *Service) footerRemovedAt(ctx context.Context, id uuid.UUID) *time.Time {
	rows, err := s.Q.ComposerVersionSources(ctx, store.ComposerVersionSourcesParams{CampaignID: id, Lim: 50})
	if err != nil {
		return nil
	}
	var removed *time.Time
	for _, r := range rows { // newest first
		secs, err := mjml.Outline(r.Mjml)
		if err != nil {
			continue
		}
		has := false
		for _, sec := range secs {
			if sec.Label == "footer" {
				has = true
				break
			}
		}
		if has {
			return removed
		}
		t := r.SavedAt
		removed = &t
	}
	return nil
}
