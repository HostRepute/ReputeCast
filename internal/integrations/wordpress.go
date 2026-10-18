package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// KnownPluginVersion is the WordPress plugin release this version of
// ReputeCast knows. With the update check off, "Update available" compares
// with it.
const KnownPluginVersion = "1.3.0"

// SiteSilentAfter is when a site that hasn't reported in needs attention.
const SiteSilentAfter = 24 * time.Hour

// DefaultCheckTimeout bounds "Check now".
const DefaultCheckTimeout = 10 * time.Second

func (s *Service) checkTimeout() time.Duration {
	if s.CheckTimeout > 0 {
		return s.CheckTimeout
	}
	return DefaultCheckTimeout
}

// seconds writes a duration as the check says it: "10 s", "0.5 s".
func seconds(d time.Duration) string {
	if d%time.Second == 0 {
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
	return fmt.Sprintf("%.1f s", d.Seconds())
}

// WordPressCategory is a post category of a site, as the plugin reports it.
type WordPressCategory struct {
	ID   string `json:"id"`
	Name string `json:"name" maxLength:"200"`
	Slug string `json:"slug" maxLength:"200"`
}

// WordPressFormPlace is where a form is used on a site.
type WordPressFormPlace struct {
	Title string `json:"title" maxLength:"200" example:"Join the beta"`
	URL   string `json:"url,omitempty" maxLength:"500"`
}

// WordPressFormUse is how a site uses one signup form.
type WordPressFormUse struct {
	Form   string               `json:"form" maxLength:"40" example:"frm_8c2e1a" doc:"The form's id (frm_…)"`
	Kind   string               `json:"kind" enum:"block,shortcode,widget"`
	Places []WordPressFormPlace `json:"places" maxItems:"100"`
}

// WordPressReportInput is what the plugin reports: when it connects, every
// 12 hours and with every post.
type WordPressReportInput struct {
	SiteURL          string              `json:"siteUrl" maxLength:"300" example:"https://blog.acme.io"`
	Name             string              `json:"name,omitempty" maxLength:"200"`
	PluginVersion    string              `json:"pluginVersion" maxLength:"20" example:"1.2.0"`
	WordPressVersion string              `json:"wordpressVersion,omitempty" maxLength:"20" example:"6.9.2"`
	PHPVersion       string              `json:"phpVersion,omitempty" maxLength:"20" example:"8.3"`
	FeedURL          string              `json:"feedUrl,omitempty" maxLength:"500"`
	LanguagePlugin   string              `json:"languagePlugin,omitempty" enum:"wpml,polylang,"`
	Categories       []WordPressCategory `json:"categories,omitempty" maxItems:"500"`
	Forms            []WordPressFormUse  `json:"forms,omitempty" maxItems:"100"`
	// WooCommerce is WooCommerce on the site, when it's active (plugin 1.4).
	WooCommerce *WordPressWooCommerce `json:"woocommerce,omitempty"`
}

// WordPressWooCommerce is WooCommerce on a site: its version, and whether
// the plugin's checkout consent and abandoned-checkout reporting are on.
type WordPressWooCommerce struct {
	Version   string `json:"version" maxLength:"20" example:"10.2.1"`
	Consent   bool   `json:"consent" doc:"The consent checkbox is at the checkout"`
	Checkouts bool   `json:"checkouts" doc:"Abandoned checkouts are reported"`
}

// WordPressReportResult tells the plugin where it reported to.
type WordPressReportResult struct {
	Workspace     string `json:"workspace"`
	WorkspaceName string `json:"workspaceName"`
	Site          string `json:"site" doc:"The site's address as ReputeCast lists it"`
	// LatestPlugin is the newest plugin release ReputeCast knows of.
	LatestPlugin string `json:"latestPlugin"`
}

func semverParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	out := []int{}
	for _, p := range strings.SplitN(v, ".", 3) {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	for len(out) < 3 {
		out = append(out, 0)
	}
	return out
}

// newerVersion reports whether a is a later version than b.
func newerVersion(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	x, y := semverParts(a), semverParts(b)
	for i := range 3 {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// LatestPlugin returns the newest WordPress plugin version: from the daily
// update check when it ran, else the one this release knows.
func (s *Service) LatestPlugin(ctx context.Context) string {
	if r, err := s.Q.IntegrationReleaseGet(ctx, KindWordPress); err == nil && newerVersion(r.Version, KnownPluginVersion) {
		return r.Version
	}
	return KnownPluginVersion
}

func (s *Service) callingSite(ctx context.Context) (*api.IntegrationKey, error) {
	ik := api.IntegrationFrom(ctx)
	if ik == nil || ik.Kind != KindWordPress {
		return nil, api.Err(http.StatusForbidden, api.CodePermissionDenied, "Only a WordPress site's key calls this.")
	}
	return ik, nil
}

// Report keeps what a site's plugin reports. The first report connects
// the site.
func (s *Service) Report(ctx context.Context, ws *store.Workspace, in WordPressReportInput) (WordPressReportResult, error) {
	ik, err := s.callingSite(ctx)
	if err != nil {
		return WordPressReportResult{}, err
	}
	u, err := url.Parse(strings.TrimSpace(in.SiteURL))
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return WordPressReportResult{}, api.Invalid("body.siteUrl", "Send the site's address, like https://blog.acme.io.")
	}
	siteURL := strings.TrimRight(u.Scheme+"://"+u.Host+u.EscapedPath(), "/")
	host, _ := siteHost(siteURL)
	noteTargetJSON(ctx, CallTarget{Kind: "report", Version: in.PluginVersion, Endpoint: host})
	if other, err := s.Q.WordPressSiteElsewhere(ctx, store.WordPressSiteElsewhereParams{SiteUrl: siteURL, IntegrationID: ik.ID}); err == nil {
		return WordPressReportResult{}, api.Conflict(fmt.Sprintf("%s is connected to %s. A site connects to one workspace: disconnect it there first.", host, other.WorkspaceName))
	}
	site, err := s.Q.WordPressSite(ctx, ik.ID)
	if err != nil {
		return WordPressReportResult{}, api.MapError(err, "Site")
	}
	cats := in.Categories
	if cats == nil {
		cats = []WordPressCategory{}
	}
	formsUsed := in.Forms
	if formsUsed == nil {
		formsUsed = []WordPressFormUse{}
	}
	for i := range formsUsed {
		if formsUsed[i].Places == nil {
			formsUsed[i].Places = []WordPressFormPlace{}
		}
	}
	first := site.FirstReportAt == nil
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WordPressSiteReport(ctx, store.WordPressSiteReportParams{IntegrationID: ik.ID, SiteUrl: siteURL, Name: clip(in.Name, 200),
			PluginVersion: clip(in.PluginVersion, 20), WpVersion: clip(in.WordPressVersion, 20), PhpVersion: clip(in.PHPVersion, 20),
			FeedUrl: clip(in.FeedURL, 500), LanguagePlugin: in.LanguagePlugin, Categories: jsonOf(cats), Forms: jsonOf(formsUsed), At: s.now()}); err != nil {
			return err
		}
		woo := WordPressWooCommerce{}
		if in.WooCommerce != nil {
			woo = *in.WooCommerce
		}
		if err := q.CommerceWordPressWoo(ctx, store.CommerceWordPressWooParams{IntegrationID: ik.ID, Version: clip(woo.Version, 20), Consent: woo.Consent,
			Checkouts: woo.Checkouts}); err != nil {
			return err
		}
		if !first {
			return nil
		}
		// The plugin knows the real address: the site is listed by it.
		if host != "" && host != site.Address {
			if err := q.IntegrationSetLabel(ctx, store.IntegrationSetLabelParams{ID: ik.ID, Label: host}); err != nil && !isUnique(err) {
				return err
			}
			if err := q.IntegrationKeySetName(ctx, store.IntegrationKeySetNameParams{IntegrationID: &ik.ID, Name: "WordPress · " + host}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "wordpress.site_connected", TargetType: "integration", TargetID: ik.ID.String(),
			TargetName: "WordPress · " + host, After: map[string]any{"site": siteURL, "plugin": in.PluginVersion, "wordpress": in.WordPressVersion, "php": in.PHPVersion},
			Metadata: s.keyMeta(ctx, map[string]any{"summary": fmt.Sprintf("plugin %s reported in · WordPress %s · PHP %s", in.PluginVersion, in.WordPressVersion, in.PHPVersion)})})
	})
	if err != nil {
		return WordPressReportResult{}, err
	}
	return WordPressReportResult{Workspace: ws.Slug, WorkspaceName: ws.Name, Site: host, LatestPlugin: s.LatestPlugin(ctx)}, nil
}

// WordPressForm is a signup form as the plugin's block lists it.
type WordPressForm struct {
	ID     string `json:"id" example:"frm_8c2e1a"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,live,paused"`
	Kind   string `json:"kind"`
	// SiteAllowed: the calling site is one of the form's allowed domains.
	// The block warns the editor when it isn't.
	SiteAllowed bool   `json:"siteAllowed"`
	Shortcode   string `json:"shortcode" example:"[reputecast_form id=\"frm_8c2e1a\"]"`
	Script      string `json:"script" doc:"The form's public script, as embeds load it"`
}

// WordPressFormList is the workspace's signup forms.
type WordPressFormList struct {
	Items []WordPressForm `json:"items"`
}

// Forms lists the workspace's signup forms for the block and the shortcode.
// The key reads forms only: never contacts.
func (s *Service) Forms(ctx context.Context, ws *store.Workspace) (WordPressFormList, error) {
	ik, err := s.callingSite(ctx)
	if err != nil {
		return WordPressFormList{}, err
	}
	noteTargetJSON(ctx, CallTarget{Kind: "forms"})
	host := ""
	if site, err := s.Q.WordPressSite(ctx, ik.ID); err == nil {
		host = siteBareHost(firstNonEmpty(site.SiteUrl, site.Address))
	}
	rows, err := s.Q.WordPressForms(ctx, ws.ID)
	if err != nil {
		return WordPressFormList{}, err
	}
	out := WordPressFormList{Items: []WordPressForm{}}
	for _, f := range rows {
		out.Items = append(out.Items, WordPressForm{ID: f.PublicID, Name: f.Name, Status: f.Status, Kind: f.Kind,
			SiteAllowed: host != "" && forms.OriginAllowed(host, f.AllowedOrigins, nil),
			Shortcode:   fmt.Sprintf("[reputecast_form id=%q]", f.PublicID),
			Script:      s.base() + "/p/" + url.PathEscape(ws.Slug) + "/forms/" + url.PathEscape(f.PublicID) + ".js"})
	}
	return out, nil
}

// siteBareHost is a site's host without a path ("blog.acme.io").
func siteBareHost(raw string) string {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	if u, err := url.Parse(raw); err == nil {
		return strings.ToLower(u.Hostname())
	}
	return ""
}

// WordPressPost is a published post the plugin sends.
type WordPressPost struct {
	ID          string     `json:"id" maxLength:"64"`
	URL         string     `json:"url,omitempty" maxLength:"1000"`
	Title       string     `json:"title" minLength:"1" maxLength:"300"`
	Excerpt     string     `json:"excerpt,omitempty" maxLength:"2000"`
	ImageURL    string     `json:"imageUrl,omitempty" maxLength:"1000" doc:"The featured image"`
	Categories  []string   `json:"categories,omitempty" maxItems:"100" doc:"Category slugs"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	Language    string     `json:"language,omitempty" maxLength:"40"`
	// Translations (plugin 1.3.0, with WPML or Polylang) become the draft's
	// language versions, marked up to date.
	Translations []WordPressPostTranslation `json:"translations,omitempty" maxItems:"20" doc:"The post's translations (plugin 1.3.0, WPML or Polylang): they become the draft's language versions"`
}

// WordPressPostTranslation is one of a post's translations.
type WordPressPostTranslation struct {
	ID       string `json:"id" maxLength:"64"`
	Language string `json:"language" minLength:"2" maxLength:"40" example:"es"`
	URL      string `json:"url,omitempty" maxLength:"1000"`
	Title    string `json:"title" minLength:"1" maxLength:"300"`
	Excerpt  string `json:"excerpt,omitempty" maxLength:"2000"`
	ImageURL string `json:"imageUrl,omitempty" maxLength:"1000"`
}

// WordPressDraftInput asks for a campaign draft from a post.
type WordPressDraftInput struct {
	Post WordPressPost `json:"post"`
}

// WordPressDraftResult says whether a draft was created, and why not.
type WordPressDraftResult struct {
	Created bool `json:"created"`
	// Reason when nothing was created: no_rule, rule_off, category or
	// exists (the post already has a draft).
	Reason     string `json:"reason,omitempty" enum:"no_rule,rule_off,category,exists,"`
	CampaignID string `json:"campaignId,omitempty"`
	Name       string `json:"name,omitempty"`
	// Languages are the language versions the post's translations added:
	// to the new draft, or to the post's draft when it exists.
	Languages []string `json:"languages,omitempty"`
}

// WordPressRule is a site's post-draft rule.
type WordPressRule struct {
	Categories   []string `json:"categories" maxItems:"100" doc:"Category slugs"`
	ListIDs      []string `json:"listIds" maxItems:"20"`
	TemplateID   string   `json:"templateId,omitempty" maxLength:"64"`
	SenderID     string   `json:"senderId,omitempty" maxLength:"64"`
	NotifyUserID string   `json:"notifyUserId,omitempty" maxLength:"64"`
}

func parseRule(raw json.RawMessage) WordPressRule {
	var r WordPressRule
	_ = json.Unmarshal(raw, &r)
	if r.Categories == nil {
		r.Categories = []string{}
	}
	if r.ListIDs == nil {
		r.ListIDs = []string{}
	}
	return r
}

// CreateDraft turns a published post into a campaign draft when the site's
// rule says so. Drafts only: nobody gets anything until a person reviews
// and sends it through Send Readiness and approvals as usual.
func (s *Service) CreateDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WordPressDraftInput) (WordPressDraftResult, error) {
	ik, err := s.callingSite(ctx)
	if err != nil {
		return WordPressDraftResult{}, err
	}
	post := in.Post
	post.ID = strings.TrimSpace(post.ID)
	if post.ID == "" {
		return WordPressDraftResult{}, api.Invalid("body.post.id", "Send the post's id.")
	}
	noteTargetJSON(ctx, CallTarget{Kind: "draft", Title: post.Title})
	site, err := s.Q.WordPressSite(ctx, ik.ID)
	if err != nil {
		return WordPressDraftResult{}, api.MapError(err, "Site")
	}
	if s.PostReported != nil {
		s.PostReported(ctx, ik.ID)
	}
	rule := parseRule(site.Rule)
	host := siteBareHost(firstNonEmpty(site.SiteUrl, site.Address))
	if d, ok := s.postDraft(ctx, ik.ID, post); ok {
		out := WordPressDraftResult{Reason: "exists"}
		if d.CampaignID != nil {
			out.CampaignID = d.CampaignID.String()
			if len(post.Translations) > 0 {
				// A translation published after the post joins its draft.
				out.Languages = s.draftLanguages(ctx, ws, p, rule, *d.CampaignID, post, host, true)
			}
		}
		return out, nil
	}
	switch {
	case len(rule.ListIDs) == 0:
		return WordPressDraftResult{Reason: "no_rule"}, nil
	case !site.RuleEnabled:
		return WordPressDraftResult{Reason: "rule_off"}, nil
	case !slices.ContainsFunc(post.Categories, func(c string) bool { return slices.Contains(rule.Categories, c) }):
		return WordPressDraftResult{Reason: "category"}, nil
	}
	lists := []uuid.UUID{}
	for _, raw := range rule.ListIDs {
		if id, err := uuid.Parse(raw); err == nil {
			lists = append(lists, id)
		}
	}
	src, subject, preheader := s.postContent(ctx, ws, rule, post)
	compiled := mjml.Compile(src)
	htmlOut := compiled.HTML
	plain := contentcheck.PlainText(htmlOut)
	audience, _ := json.Marshal(map[string]any{"lists": lists})
	var sender *uuid.UUID
	if id, err := uuid.Parse(rule.SenderID); err == nil {
		sender = &id
	}
	var notifyUser *uuid.UUID
	if id, err := uuid.Parse(rule.NotifyUserID); err == nil {
		notifyUser = &id
	}
	campaignID := uuid.Must(uuid.NewV7())
	owner := p.UserID
	var creator *uuid.UUID
	if owner != uuid.Nil {
		creator = &owner
	}
	listName := ""
	if names, err := s.Q.IntegrationListNames(ctx, ws.ID); err == nil {
		for _, l := range names {
			if len(lists) > 0 && l.ID == lists[0] {
				listName = l.Name
			}
		}
	}
	notifiedName := ""
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationDraftInsert(ctx, store.IntegrationDraftInsertParams{ID: campaignID, WorkspaceID: ws.ID, Name: clip(post.Title, 200),
			Subject: clip(post.Title, 250), Preheader: clip(preheader, 250), SenderID: sender, Audience: audience, Mjml: src, Html: htmlOut, PlainText: plain,
			ContentHash: campaigns.ContentHash(subject, preheader, src, htmlOut, nil), Note: "draft from " + host, CreatedBy: creator, Now: s.now()}); err != nil {
			return err
		}
		if err := q.IntegrationDraftVersion(ctx, store.IntegrationDraftVersionParams{CampaignID: campaignID, Subject: clip(post.Title, 250), Preheader: clip(preheader, 250),
			Mjml: src, Html: htmlOut, PlainText: plain, SavedBy: creator, SavedAt: s.now(), Label: "created from the post on " + host, SizeBytes: int32(len(htmlOut))}); err != nil {
			return err
		}
		if err := q.WordPressDraftInsert(ctx, store.WordPressDraftInsertParams{ID: uuid.Must(uuid.NewV7()), IntegrationID: ik.ID, WorkspaceID: ws.ID, CampaignID: &campaignID,
			PostID: post.ID, PostTitle: clip(post.Title, 300), PostUrl: clip(post.URL, 1000), PublishedAt: post.PublishedAt, NotifiedUser: notifyUser, CreatedAt: s.now()}); err != nil {
			if isUnique(err) {
				return api.Conflict("This post already has a draft.")
			}
			return err
		}
		if notifyUser != nil {
			if u, err := q.GetUser(ctx, *notifyUser); err == nil {
				notifiedName = u.Name
			}
			if err := notify.ToUser(ctx, tx, *notifyUser, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindPostDraftCreated, Severity: "info",
				Title: "Post draft created",
				Body: fmt.Sprintf("“%s” from %s is a draft for %s. Review it, then send it through Send Readiness as usual.",
					clip(post.Title, 300), host, firstNonEmpty(listName, "its audience")),
				Link: fmt.Sprintf("/w/%s/campaigns/%s/edit", ws.Slug, campaignID),
				Data: map[string]any{"campaignId": campaignID.String(), "site": host, "integrationId": ik.ID.String()}}); err != nil {
				return err
			}
		}
		summary := fmt.Sprintf("draft from the post published %s", publishedWords(post.PublishedAt, ws))
		if len(rule.Categories) > 0 && listName != "" {
			summary += fmt.Sprintf(" · rule %s → %s", strings.Join(categoryNames(site.Categories, rule.Categories), ", "), listName)
		}
		if notifiedName != "" {
			summary += " · " + notifiedName + " notified"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "wordpress.draft_created", TargetType: "campaign", TargetID: campaignID.String(),
			TargetName: firstNonEmpty(listName, "Draft") + " · " + clip(post.Title, 200),
			After:      map[string]any{"post": post.URL, "subject": post.Title, "lists": rule.ListIDs, "site": host},
			Metadata:   s.keyMeta(ctx, map[string]any{"summary": summary})})
	})
	if err != nil {
		return WordPressDraftResult{}, err
	}
	return WordPressDraftResult{Created: true, CampaignID: campaignID.String(), Name: clip(post.Title, 200),
		Languages: s.draftLanguages(ctx, ws, p, rule, campaignID, post, host, false)}, nil
}

// postDraft finds the draft of a post, or of one of its translations: a
// translation published first made the draft its post now joins.
func (s *Service) postDraft(ctx context.Context, site uuid.UUID, post WordPressPost) (store.WordpressDraft, bool) {
	ids := []string{post.ID}
	for _, t := range post.Translations {
		if id := strings.TrimSpace(t.ID); id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if d, err := s.Q.WordPressDraftByPost(ctx, store.WordPressDraftByPostParams{IntegrationID: site, PostID: id}); err == nil {
			return d, true
		}
	}
	return store.WordpressDraft{}, false
}

// DraftVersion is a language version of a post's draft: a translation's
// subject, preheader and content.
type DraftVersion struct {
	Language  string
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	Label     string
}

// DraftLanguages writes a post draft's languages (plugin 1.3.0). The
// content-languages domain implements it; without it a draft has one
// language.
type DraftLanguages interface {
	// Add writes the draft in the post's language (source), when the
	// workspace offers it and the draft has no versions yet, then adds the
	// versions the workspace offers that the draft doesn't have, marked up
	// to date. It returns the languages added.
	Add(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, source string, versions []DraftVersion) ([]string, error)
}

// draftLanguages turns a post's translations into its draft's language
// versions. Joining (the draft exists), the post itself is one more
// version and the draft's language stays. A failure leaves the draft as
// it is: it is logged, and the result lists what was added.
func (s *Service) draftLanguages(ctx context.Context, ws *store.Workspace, p *authz.Principal, rule WordPressRule, campaignID uuid.UUID, post WordPressPost, host string, joining bool) []string {
	if s.Languages == nil || (len(post.Translations) == 0 && post.Language == "") {
		return nil
	}
	version := func(lang string, t WordPressPost) DraftVersion {
		src, subject, preheader := s.postContent(ctx, ws, rule, t)
		return DraftVersion{Language: lang, Subject: subject, Preheader: preheader, MJML: src, HTML: mjml.Compile(src).HTML,
			Label: "translation of the post on " + host}
	}
	versions := []DraftVersion{}
	source := post.Language
	if joining {
		versions = append(versions, version(post.Language, post))
		source = ""
	}
	for _, t := range post.Translations {
		// A translation without its own featured image shows the post's.
		versions = append(versions, version(t.Language, WordPressPost{ID: t.ID, URL: t.URL, Title: t.Title, Excerpt: t.Excerpt, ImageURL: firstNonEmpty(t.ImageURL, post.ImageURL)}))
	}
	added, err := s.Languages.Add(ctx, ws, p, campaignID, source, versions)
	if err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "wordpress: the post's translations didn't become language versions", "campaign", campaignID, "err", err)
	}
	return added
}

func publishedWords(t *time.Time, ws *store.Workspace) string {
	if t == nil {
		return "just now"
	}
	return t.In(wsLocation(ws)).Format("15:04")
}

// categoryNames turns rule slugs into the site's category names.
func categoryNames(raw json.RawMessage, slugs []string) []string {
	var cats []WordPressCategory
	_ = json.Unmarshal(raw, &cats)
	out := []string{}
	for _, slug := range slugs {
		name := slug
		for _, c := range cats {
			if c.Slug == slug {
				name = c.Name
			}
		}
		out = append(out, name)
	}
	return out
}

var postPlaceholder = regexp.MustCompile(`\{\{\s*post\.(title|excerpt|image_url|url)\s*\}\}`)

// postImage is a template's image of the post, left out for a post
// without a featured image (an mj-image needs a src).
var postImage = regexp.MustCompile(`<mj-image\b[^>]*\{\{\s*post\.image_url\s*\}\}[^>]*?(?:/>|>\s*</mj-image>)`)

// postContent fills the template's post section with the post: the
// featured image, the title, the excerpt and a "Read more" button. A
// template without {{ post.… }} placeholders gets the section at the top.
func (s *Service) postContent(ctx context.Context, ws *store.Workspace, rule WordPressRule, post WordPressPost) (string, string, string) {
	subject := clip(post.Title, 250)
	preheader := clip(strings.TrimSpace(post.Excerpt), 250)
	values := map[string]string{
		"title": html.EscapeString(post.Title), "excerpt": html.EscapeString(strings.TrimSpace(post.Excerpt)),
		"image_url": html.EscapeString(post.ImageURL), "url": html.EscapeString(post.URL),
	}
	src := ""
	if id, err := uuid.Parse(rule.TemplateID); err == nil {
		if t, err := s.Q.IntegrationTemplate(ctx, store.IntegrationTemplateParams{ID: id, WorkspaceID: ws.ID}); err == nil {
			src = t.Mjml
		}
	}
	if src == "" {
		src = `<mjml><mj-body background-color="#F3F4F6"></mj-body></mjml>`
	}
	if postPlaceholder.MatchString(src) {
		if post.ImageURL == "" {
			src = postImage.ReplaceAllString(src, "")
		}
		return postPlaceholder.ReplaceAllStringFunc(src, func(m string) string {
			return values[postPlaceholder.FindStringSubmatch(m)[1]]
		}), subject, preheader
	}
	section := postSection(values)
	if i := strings.Index(src, "<mj-body"); i >= 0 {
		if j := strings.Index(src[i:], ">"); j >= 0 {
			at := i + j + 1
			return src[:at] + section + src[at:], subject, preheader
		}
	}
	return src, subject, preheader
}

func postSection(v map[string]string) string {
	var b strings.Builder
	b.WriteString(`<mj-section background-color="#FFFFFF" padding="0"><mj-column>`)
	if v["image_url"] != "" {
		fmt.Fprintf(&b, `<mj-image src="%s" alt="%s" padding="0" href="%s"/>`, v["image_url"], v["title"], v["url"])
	}
	fmt.Fprintf(&b, `<mj-text css-class="rc-post-title" font-size="24px" font-weight="700" line-height="1.25" padding="24px 24px 8px"><h2 style="margin:0;font-size:24px">%s</h2></mj-text>`, v["title"])
	if v["excerpt"] != "" {
		fmt.Fprintf(&b, `<mj-text font-size="16px" line-height="1.55" padding="0 24px 16px">%s</mj-text>`, v["excerpt"])
	}
	if v["url"] != "" {
		fmt.Fprintf(&b, `<mj-button href="%s" background-color="#0F766E" color="#FFFFFF" align="left" padding="0 24px 24px">Read more</mj-button>`, v["url"])
	}
	b.WriteString(`</mj-column></mj-section>`)
	return b.String()
}

// WordPressRuleInput sets a site's post-draft rule.
type WordPressRuleInput struct {
	Enabled bool `json:"enabled"`
	WordPressRule
}

// SetRule changes a site's post-draft rule.
func (s *Service) SetRule(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WordPressRuleInput) (WordPressRule, error) {
	conn, _, err := s.connection(ctx, ws, id)
	if err != nil {
		return WordPressRule{}, err
	}
	if conn.Kind != KindWordPress {
		return WordPressRule{}, api.NotFound("Site")
	}
	site, err := s.Q.WordPressSite(ctx, conn.ID)
	if err != nil {
		return WordPressRule{}, api.MapError(err, "Site")
	}
	r := in.WordPressRule
	if r.Categories == nil {
		r.Categories = []string{}
	}
	if r.ListIDs == nil {
		r.ListIDs = []string{}
	}
	if in.Enabled {
		if len(r.Categories) == 0 {
			return WordPressRule{}, api.Invalid("body.categories", "Pick at least one category.")
		}
		if len(r.ListIDs) == 0 {
			return WordPressRule{}, api.Invalid("body.listIds", "Pick the audience of the drafts.")
		}
	}
	lists, err := s.Q.IntegrationListNames(ctx, ws.ID)
	if err != nil {
		return WordPressRule{}, err
	}
	for _, raw := range r.ListIDs {
		if !slices.ContainsFunc(lists, func(l store.IntegrationListNamesRow) bool { return l.ID.String() == raw }) {
			return WordPressRule{}, api.Invalid("body.listIds", "Pick lists of this workspace.")
		}
	}
	if r.TemplateID != "" {
		tid, err := uuid.Parse(r.TemplateID)
		if err != nil {
			return WordPressRule{}, api.Invalid("body.templateId", "Pick a saved template.")
		}
		if _, err := s.Q.IntegrationTemplate(ctx, store.IntegrationTemplateParams{ID: tid, WorkspaceID: ws.ID}); err != nil {
			return WordPressRule{}, api.Invalid("body.templateId", "Pick a saved template.")
		}
	}
	if r.SenderID != "" {
		senders, err := s.Q.IntegrationSenders(ctx, ws.ID)
		if err != nil {
			return WordPressRule{}, err
		}
		if !slices.ContainsFunc(senders, func(x store.IntegrationSendersRow) bool { return x.ID.String() == r.SenderID }) {
			return WordPressRule{}, api.Invalid("body.senderId", "Pick a sender of this workspace.")
		}
	}
	if r.NotifyUserID != "" {
		uid, err := uuid.Parse(r.NotifyUserID)
		if err != nil {
			return WordPressRule{}, api.Invalid("body.notifyUserId", "Pick a member of this workspace.")
		}
		if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: uid}); err != nil || m.Status != "active" {
			return WordPressRule{}, api.Invalid("body.notifyUserId", "Pick a member of this workspace.")
		}
	}
	raw, _ := json.Marshal(r)
	before := parseRule(site.Rule)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WordPressSiteSetRule(ctx, store.WordPressSiteSetRuleParams{IntegrationID: conn.ID, Enabled: in.Enabled, Rule: raw, By: &p.UserID, At: s.now()}); err != nil {
			return err
		}
		state := "off"
		if in.Enabled {
			state = "on"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "wordpress.rule_changed", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: "WordPress · " + conn.Label, Before: map[string]any{"enabled": site.RuleEnabled, "rule": before},
			After:    map[string]any{"enabled": in.Enabled, "rule": r},
			Metadata: map[string]any{"summary": fmt.Sprintf("post drafts %s · %s", state, strings.Join(categoryNames(site.Categories, r.Categories), ", ")), "permission": string(authz.IntegrationsManage)}})
	})
	return r, err
}

// WordPressCheck is the result of "Check now".
type WordPressCheck struct {
	OK        bool      `json:"ok"`
	CheckedAt time.Time `json:"checkedAt"`
	// Outcome: the site answered 200, timed out, answered another status,
	// was blocked by the SSRF guard, or couldn't be reached.
	Outcome string `json:"outcome" enum:"answered,timeout,status,blocked,unreachable"`
	// Ms is how long a site took to answer; Seconds the time limit it ran
	// out of; Status the HTTP status it answered.
	Ms      int64   `json:"ms,omitempty"`
	Seconds float64 `json:"seconds,omitempty"`
	Status  int     `json:"status,omitempty"`
	Detail  string  `json:"detail" example:"Timed out after 10 s"`
}

// CheckSite calls the site's status endpoint once, through the SSRF guard.
// ReputeCast never polls a site on a schedule.
func (s *Service) CheckSite(ctx context.Context, ws *store.Workspace, id uuid.UUID) (WordPressCheck, error) {
	conn, _, err := s.connection(ctx, ws, id)
	if err != nil {
		return WordPressCheck{}, err
	}
	if conn.Kind != KindWordPress {
		return WordPressCheck{}, api.NotFound("Site")
	}
	site, err := s.Q.WordPressSite(ctx, conn.ID)
	if err != nil {
		return WordPressCheck{}, api.MapError(err, "Site")
	}
	base := site.SiteUrl
	if base == "" {
		base = siteBase(site.Address)
	}
	hashes, err := s.Q.WordPressSiteKeyHashes(ctx, &conn.ID)
	if err != nil {
		return WordPressCheck{}, err
	}
	out := s.probe(ctx, strings.TrimRight(base, "/")+"/wp-json/reputecast/v1/status", statusHeaders(hashes, s.now()))
	out.CheckedAt = s.now()
	b, _ := json.Marshal(out)
	if err := s.Q.WordPressSiteChecked(ctx, store.WordPressSiteCheckedParams{IntegrationID: conn.ID, At: out.CheckedAt, LastCheck: b}); err != nil {
		return out, err
	}
	return out, nil
}

// StatusSignature signs a "Check now" call with one of a site's keys: the
// hex HMAC-SHA256 of "{timestamp}.status", keyed with the key's SHA-256.
// ReputeCast keeps only that hash; the plugin computes it from the key it
// holds, so it answers only the workspace it's connected to.
func StatusSignature(keyHash []byte, ts int64) string {
	m := hmac.New(sha256.New, keyHash)
	m.Write([]byte(strconv.FormatInt(ts, 10) + ".status"))
	return hex.EncodeToString(m.Sum(nil))
}

// statusHeaders sign a "Check now" call with every live key of the site
// (a rotated key keeps working for 24 hours, and the plugin may still hold
// it): X-ReputeCast-Timestamp, and the signatures comma-separated in
// X-ReputeCast-Signature.
func statusHeaders(hashes [][]byte, now time.Time) map[string]string {
	if len(hashes) == 0 {
		return nil
	}
	ts := now.Unix()
	sigs := make([]string, 0, len(hashes))
	for _, h := range hashes {
		sigs = append(sigs, StatusSignature(h, ts))
	}
	return map[string]string{webhooks.HeaderTimestamp: strconv.FormatInt(ts, 10), webhooks.HeaderSignature: strings.Join(sigs, ",")}
}

// probe GETs a site's status endpoint and describes the answer.
func (s *Service) probe(ctx context.Context, target string, headers map[string]string) WordPressCheck {
	u, err := s.policy().ValidateURL(target)
	if err != nil {
		return WordPressCheck{Outcome: "blocked", Detail: "Blocked: " + err.Error()}
	}
	timeout := s.checkTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return WordPressCheck{Outcome: "unreachable", Detail: "Invalid address"}
	}
	req.Header.Set("User-Agent", "ReputeCast-Check")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	start := s.now()
	resp, err := s.client(timeout).Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return WordPressCheck{Outcome: "timeout", Seconds: timeout.Seconds(), Detail: "Timed out after " + seconds(timeout)}
		}
		return WordPressCheck{Outcome: "unreachable", Detail: netguard.Describe(ctx, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	ms := s.now().Sub(start).Milliseconds()
	if resp.StatusCode != http.StatusOK {
		return WordPressCheck{Outcome: "status", Status: resp.StatusCode, Detail: fmt.Sprintf("Answered HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))}
	}
	return WordPressCheck{OK: true, Outcome: "answered", Ms: ms, Detail: fmt.Sprintf("Answered in %d ms", ms)}
}
