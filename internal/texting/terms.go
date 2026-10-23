package texting

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The text message terms page (brief §5.7): a built-in page at
// news.acme.io/sms-terms, generated from the SMS rules. Its text can be
// edited, and it's versioned: consent records point to the version that
// was live. Forms & pages lists it with the preference center.

// SmsTermsContent is the page's text, section by section.
type SmsTermsContent struct {
	Intro     string `json:"intro" maxLength:"1000"`
	Program   string `json:"program" maxLength:"500"`
	Frequency string `json:"frequency" maxLength:"300"`
	Cost      string `json:"cost" maxLength:"300"`
	Help      string `json:"help" maxLength:"500"`
	OptOut    string `json:"optOut" maxLength:"500"`
	Senders   string `json:"senders" maxLength:"500"`
	Privacy   string `json:"privacy" maxLength:"500"`
	Carriers  string `json:"carriers" maxLength:"300"`
	// PrivacyURL links the privacy policy.
	PrivacyURL string `json:"privacyUrl,omitempty" maxLength:"500"`
	// HelpNumber and HelpEmail are shown as links in the Help section.
	HelpNumber string `json:"helpNumber,omitempty" maxLength:"40"`
	HelpEmail  string `json:"helpEmail,omitempty" maxLength:"200"`
	// Address is the sender's postal address, at the foot of the page.
	Address string `json:"address,omitempty" maxLength:"300"`
}

// SmsTerms is the terms page in Forms & pages.
type SmsTerms struct {
	Version     int             `json:"version"`
	PublishedAt *time.Time      `json:"publishedAt,omitempty"`
	PublishedBy string          `json:"publishedBy,omitempty"`
	Content     SmsTermsContent `json:"content"`
	// Generated is what the rules would write now: the editor's "Reset".
	Generated SmsTermsContent `json:"generated"`
	URL       string          `json:"url"`
	CanEdit   bool            `json:"canEdit"`
}

// TermsURL is the terms page's address: on the workspace's own domain when
// it has one.
func (s *Service) TermsURL(ctx context.Context, ws *store.Workspace) string {
	if s.PageBase != nil {
		if b := s.PageBase(ctx, ws); b != "" {
			return strings.TrimRight(b, "/") + "/sms-terms"
		}
	}
	return strings.TrimRight(s.URLs.Base(), "/") + "/p/" + ws.Slug + "/sms-terms"
}

// generatedTerms writes the page from the rules and the routes.
func (s *Service) generatedTerms(ctx context.Context, ws *store.Workspace) (SmsTermsContent, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsTermsContent{}, err
	}
	brand := strings.TrimSuffix(strings.TrimSpace(g.rules.BrandPrefix), ":")
	if brand == "" {
		brand = brandOf(ws)
	}
	c := SmsTermsContent{
		Intro:     "These terms cover the texts you agree to when you tick the SMS box on one of our forms. Email and texts are separate: one never signs you up for the other.",
		Program:   brand + " texts: updates from " + ws.Name + ".",
		Frequency: "Message frequency varies.",
		Cost:      "Message and data rates may apply.",
		OptOut:    "Text STOP at any time. We'll confirm once and send nothing more.",
		Privacy:   "We never sell or share your number for marketing.",
		Carriers:  "Carriers aren't liable for delayed or undelivered messages.",
	}
	for _, id := range g.order {
		if sn := g.senders[id]; sn.Kind != "sender_id" {
			c.HelpNumber = senderLabel(sn)
			break
		}
	}
	if c.HelpNumber != "" {
		c.Help = "Text HELP to " + c.HelpNumber + "."
	} else {
		c.Help = "Text HELP to the number that texted you."
	}
	parts := []string{}
	for _, r := range g.routes {
		sn, ok := g.senders[r.SenderID]
		if !ok {
			continue
		}
		parts = append(parts, senderLabel(sn)+" in "+termsRegion(r.Countries))
	}
	if len(parts) > 0 {
		c.Senders = strings.Join(parts, "; ") + "."
	}
	return c, nil
}

// termsRegion names countries in the page's sentence: "the US and Canada",
// "the UK, Germany, the Netherlands and Ireland".
func termsRegion(codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		n := countryName(c)
		switch c {
		case "US":
			n = "the US"
		case "GB":
			n = "the UK"
		case "NL":
			n = "the Netherlands"
		}
		names = append(names, n)
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// Terms is the terms page as Forms & pages edits it.
func (s *Service) Terms(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SmsTerms, error) {
	gen, err := s.generatedTerms(ctx, ws)
	if err != nil {
		return SmsTerms{}, err
	}
	out := SmsTerms{Content: gen, Generated: gen, URL: s.TermsURL(ctx, ws), CanEdit: p.Can(authz.FormsEdit)}
	row, err := s.Q.SmsTermsLatest(ctx, ws.ID)
	if err == nil {
		out.Version = int(row.Version)
		at := row.PublishedAt
		out.PublishedAt = &at
		_ = json.Unmarshal(row.Content, &out.Content)
		if row.PublishedBy != nil {
			if u, err := s.Q.GetUser(ctx, *row.PublishedBy); err == nil {
				out.PublishedBy = u.Name
			}
		}
	} else if !notFound(err) {
		return out, err
	}
	return out, nil
}

// PublishTerms publishes a new version of the page. Consent given from now
// on records this version. Audited.
func (s *Service) PublishTerms(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SmsTermsContent) (SmsTerms, error) {
	for field, v := range map[string]string{"program": in.Program, "cost": in.Cost, "help": in.Help, "optOut": in.OptOut} {
		if strings.TrimSpace(v) == "" {
			return SmsTerms{}, api.Invalid("body."+field, say(ctx, mCarriersAsk, nil))
		}
	}
	cur, err := s.Terms(ctx, ws, p)
	if err != nil {
		return SmsTerms{}, err
	}
	now := s.now()
	version := cur.Version + 1
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.SmsTermsInsert(ctx, store.SmsTermsInsertParams{WorkspaceID: ws.ID, Version: int32(version), Content: jsonOf(in), PublishedBy: uuidPtr(p.UserID),
			At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sms_terms.published", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"version": cur.Version}, After: map[string]any{"version": version}})
	})
	if err != nil {
		return SmsTerms{}, err
	}
	return s.Terms(ctx, ws, p)
}

// TermsPage is the public page's data: the live version, or the generated
// text until one is published.
type TermsPage struct {
	Version int
	Since   time.Time
	Content SmsTermsContent
}

// PublicTermsPage is the public page with every word written: the terms'
// own text, and the page's headings in lang (English until translated).
type PublicTermsPage struct {
	Lang, Title, Version, Intro             string
	Sections                                []PublicTermsSection
	ClosingHeading, ClosingText             string
	ManageLabel, BackLabel, BackURL, Footer string
}

// PublicTermsSection is a heading and its text; Links are words in the
// text that link somewhere (a number, an address, the privacy policy).
type PublicTermsSection struct {
	Heading, Text string
	Links         []PublicTermsLink
}

// PublicTermsLink is words to link and where.
type PublicTermsLink struct {
	Text, Href string
	NoWrap     bool
}

var (
	mTermsTitle          = msgs.M("terms.title")
	mTermsVersion        = msgs.M("terms.version")
	mTermsProgram        = msgs.M("terms.program")
	mTermsFrequency      = msgs.M("terms.frequency")
	mTermsCost           = msgs.M("terms.cost")
	mTermsHelp           = msgs.M("terms.help")
	mTermsOptOut         = msgs.M("terms.optOut")
	mTermsSenders        = msgs.M("terms.senders")
	mTermsPrivacy        = msgs.M("terms.privacy")
	mTermsCarriers       = msgs.M("terms.carriers")
	mTermsClosingHeading = msgs.M("terms.closingHeading")
	mTermsClosingNumber  = msgs.M("terms.closingNumber")
	mTermsClosing        = msgs.M("terms.closing")
	mTermsManage         = msgs.M("terms.manage")
	mTermsBack           = msgs.M("terms.back")
	mTermsFooterAddress  = msgs.M("terms.footerAddress")
	mTermsFooter         = msgs.M("terms.footer")
	mTermsPrivacyRead    = msgs.M("terms.privacyRead")
	mTermsPrivacyLink    = msgs.M("terms.privacyLink")
)

// PublicTermsPage writes the public page in lang (the pages' language);
// false when the workspace has no SMS provider (the page answers 404).
func (s *Service) PublicTermsPage(ctx context.Context, ws *store.Workspace, lang i18n.Lang, code string) (PublicTermsPage, bool) {
	tp, ok := s.PublicTerms(ctx, ws)
	if !ok {
		return PublicTermsPage{}, false
	}
	c := tp.Content
	date := tp.Since.In(location(ws.Timezone)).Format("2 Jan 2006")
	if code != "" && code != "en" {
		date = lang.Region.Date(tp.Since.In(location(ws.Timezone)))
	}
	out := PublicTermsPage{Lang: code, Title: mTermsTitle.Text(lang, nil), Version: mTermsVersion.Text(lang, i18n.Args{"version": tp.Version, "date": date}),
		Intro: c.Intro, ClosingHeading: mTermsClosingHeading.Text(lang, nil), ManageLabel: mTermsManage.Text(lang, nil)}
	number := []PublicTermsLink{}
	if c.HelpNumber != "" {
		number = append(number, PublicTermsLink{Text: c.HelpNumber, NoWrap: true})
	}
	add := func(heading i18n.Msg, text string, links ...PublicTermsLink) {
		if strings.TrimSpace(text) != "" {
			out.Sections = append(out.Sections, PublicTermsSection{Heading: heading.Text(lang, nil), Text: text, Links: links})
		}
	}
	add(mTermsProgram, c.Program)
	add(mTermsFrequency, c.Frequency)
	add(mTermsCost, c.Cost)
	helpLinks := append([]PublicTermsLink{}, number...)
	if c.HelpEmail != "" {
		helpLinks = append(helpLinks, PublicTermsLink{Text: c.HelpEmail, Href: "mailto:" + c.HelpEmail})
	}
	add(mTermsHelp, c.Help, helpLinks...)
	add(mTermsOptOut, c.OptOut)
	add(mTermsSenders, c.Senders, number...)
	privacy := c.Privacy
	var privacyLinks []PublicTermsLink
	if c.PrivacyURL != "" {
		read := mTermsPrivacyRead.Text(lang, nil)
		privacy = strings.TrimSpace(privacy + " " + read)
		privacyLinks = append(privacyLinks, PublicTermsLink{Text: mTermsPrivacyLink.Text(lang, nil), Href: c.PrivacyURL})
	}
	add(mTermsPrivacy, privacy, privacyLinks...)
	add(mTermsCarriers, c.Carriers)
	if c.HelpNumber != "" {
		out.ClosingText = mTermsClosingNumber.Text(lang, i18n.Args{"number": c.HelpNumber})
	} else {
		out.ClosingText = mTermsClosing.Text(lang, nil)
	}
	// Back to where people give SMS consent: the page of the latest consent.
	var page string
	_ = s.Pool.QueryRow(ctx, `SELECT page FROM channel_consent_events WHERE workspace_id = $1 AND channel = 'sms' AND page LIKE '%.%' ORDER BY occurred_at DESC LIMIT 1`,
		ws.ID).Scan(&page)
	if page != "" {
		if !strings.HasPrefix(page, "http") {
			page = "https://" + page
		}
		out.BackLabel, out.BackURL = mTermsBack.Text(lang, nil), page
	}
	if strings.TrimSpace(c.Address) != "" {
		out.Footer = mTermsFooterAddress.Text(lang, i18n.Args{"address": strings.TrimSuffix(strings.TrimSpace(c.Address), ".")})
	} else {
		out.Footer = mTermsFooter.Text(lang, nil)
	}
	return out, true
}

// PublicTerms is what the public page shows; false when the workspace has
// no SMS provider (the page answers 404).
func (s *Service) PublicTerms(ctx context.Context, ws *store.Workspace) (TermsPage, bool) {
	var n int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_providers WHERE workspace_id = $1 AND removed_at IS NULL`, ws.ID).Scan(&n); err != nil || n == 0 {
		return TermsPage{}, false
	}
	row, err := s.Q.SmsTermsLatest(ctx, ws.ID)
	if err == nil {
		var c SmsTermsContent
		_ = json.Unmarshal(row.Content, &c)
		return TermsPage{Version: int(row.Version), Since: row.PublishedAt, Content: c}, true
	}
	gen, err := s.generatedTerms(ctx, ws)
	if err != nil {
		return TermsPage{}, false
	}
	return TermsPage{Version: 1, Since: ws.CreatedAt, Content: gen}, true
}
