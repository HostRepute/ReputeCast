package integrations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// IntegrationWordPressPage is Settings › Integrations › WordPress.
type IntegrationWordPressPage struct {
	Status          string              `json:"status" enum:"not_connected,waiting,connected,attention"`
	Reasons         []IntegrationReason `json:"reasons"`
	UpdateAvailable bool                `json:"updateAvailable"`
	// ConnectedBy names who connected the sites when one person did.
	ConnectedBy  string            `json:"connectedBy,omitempty"`
	LatestPlugin string            `json:"latestPlugin"`
	Sites        []IntegrationSite `json:"sites"`
	// What the post-draft rule dialog offers.
	Templates []IntegrationOption     `json:"templates"`
	Senders   []IntegrationOption     `json:"senders"`
	Lists     []IntegrationListOption `json:"lists"`
	Members   []IntegrationOption     `json:"members"`
	CanManage bool                    `json:"canManage"`
	// CanEditForms: the caller can allow a site on a form (forms:edit).
	CanEditForms bool     `json:"canEditForms"`
	Helpers      []string `json:"helpers"`
	InstanceURL  string   `json:"instanceUrl"`
}

// IntegrationOption is a choice of a picker.
type IntegrationOption struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
}

// IntegrationListOption is a list with its size.
type IntegrationListOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Contacts int64  `json:"contacts"`
}

// IntegrationSite is a connected WordPress site.
type IntegrationSite struct {
	ID              string                 `json:"id" doc:"The connection's id"`
	Host            string                 `json:"host"`
	SiteURL         string                 `json:"siteUrl,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Status          string                 `json:"status" enum:"waiting,connected,attention"`
	UpdateAvailable bool                   `json:"updateAvailable"`
	PluginVersion   string                 `json:"pluginVersion,omitempty"`
	WPVersion       string                 `json:"wordpressVersion,omitempty"`
	PHPVersion      string                 `json:"phpVersion,omitempty"`
	FirstReportAt   *time.Time             `json:"firstReportAt,omitempty"`
	LastReportAt    *time.Time             `json:"lastReportAt,omitempty"`
	LastCheck       *WordPressCheck        `json:"lastCheck,omitempty"`
	FeedURL         string                 `json:"feedUrl,omitempty"`
	LanguagePlugin  string                 `json:"languagePlugin,omitempty"`
	Forms           []IntegrationSiteForm  `json:"forms"`
	Categories      []WordPressCategory    `json:"categories"`
	Rule            IntegrationSiteRule    `json:"rule"`
	Drafts          []IntegrationSiteDraft `json:"drafts"`
	DraftsThisWeek  int64                  `json:"draftsThisWeek"`
	Key             *IntegrationKeyView    `json:"key,omitempty"`
	OldKey          *IntegrationKeyView    `json:"oldKey,omitempty"`
	Items           []IntegrationItemState `json:"items"`
	ConnectedBy     string                 `json:"connectedBy,omitempty"`
	ConnectedAt     time.Time              `json:"connectedAt"`
	Reasons         []IntegrationReason    `json:"reasons"`
	// SignupsRejected: a form used on the site doesn't allow it.
	SignupsRejected bool `json:"signupsRejected"`
	// WooCommerce is "WooCommerce detected": the plugin's checkout consent
	// and abandoned-checkout reporting, and the store connected at the
	// site's address.
	WooCommerce *IntegrationSiteWoo `json:"woocommerce,omitempty"`
}

// IntegrationSiteWoo is WooCommerce on a WordPress site.
type IntegrationSiteWoo struct {
	Version   string `json:"version"`
	Consent   bool   `json:"consent"`
	Checkouts bool   `json:"checkouts"`
	// StoreID and StoreName are the store connected at the site's address.
	StoreID     string `json:"storeId,omitempty"`
	StoreName   string `json:"storeName,omitempty"`
	StoreStatus string `json:"storeStatus,omitempty" enum:"connecting,importing,connected,disconnected,"`
}

// IntegrationSiteForm is a signup form used on a site.
type IntegrationSiteForm struct {
	ID      string               `json:"id" doc:"The form's uuid, for links"`
	Public  string               `json:"publicId" example:"frm_8c2e1a"`
	Name    string               `json:"name"`
	Kind    string               `json:"kind" enum:"block,shortcode,widget"`
	Places  []WordPressFormPlace `json:"places"`
	Allowed bool                 `json:"allowed" doc:"The site is one of the form's allowed domains"`
	Missing bool                 `json:"missing" doc:"The site uses a form that no longer exists"`
}

// IntegrationSiteRule is a site's post-draft rule, with names.
type IntegrationSiteRule struct {
	Enabled    bool                   `json:"enabled"`
	Set        bool                   `json:"set" doc:"A rule was set up (even if it's off now)"`
	Categories []WordPressCategory    `json:"categories"`
	List       *IntegrationListOption `json:"list,omitempty"`
	ListIDs    []string               `json:"listIds"`
	Template   *IntegrationOption     `json:"template,omitempty"`
	Sender     *IntegrationOption     `json:"sender,omitempty"`
	Notify     *IntegrationOption     `json:"notify,omitempty"`
}

// IntegrationSiteDraft is a draft created from a post.
type IntegrationSiteDraft struct {
	CampaignID  string     `json:"campaignId,omitempty"`
	Title       string     `json:"title"`
	Status      string     `json:"status" doc:"The campaign's status (draft, scheduled, sent…)"`
	Audience    string     `json:"audience,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	Notified    string     `json:"notified,omitempty"`
	PostURL     string     `json:"postUrl,omitempty"`
}

// WordPressPage builds the WordPress sites page.
func (s *Service) WordPressPage(ctx context.Context, ws *store.Workspace, p *authz.Principal) (IntegrationWordPressPage, error) {
	out := IntegrationWordPressPage{Status: StatusNotConnected, Reasons: []IntegrationReason{}, Sites: []IntegrationSite{}, Templates: []IntegrationOption{},
		Senders: []IntegrationOption{}, Lists: []IntegrationListOption{}, Members: []IntegrationOption{}, LatestPlugin: s.LatestPlugin(ctx),
		CanManage: p.Can(authz.IntegrationsManage), CanEditForms: p.Can(authz.FormsEdit), Helpers: s.helpers(ctx, ws, authz.IntegrationsManage), InstanceURL: s.base()}
	conns, err := s.active(ctx, ws)
	if err != nil {
		return out, err
	}
	lists, err := s.Q.IntegrationListSizes(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, IntegrationListOption{ID: l.ID.String(), Name: l.Name, Contacts: l.Contacts})
	}
	if ts, err := s.Q.IntegrationTemplates(ctx, ws.ID); err == nil {
		for _, t := range ts {
			out.Templates = append(out.Templates, IntegrationOption{ID: t.ID.String(), Name: t.Name})
		}
	}
	if ss, err := s.Q.IntegrationSenders(ctx, ws.ID); err == nil {
		for _, x := range ss {
			out.Senders = append(out.Senders, IntegrationOption{ID: x.ID.String(), Name: x.Name, Detail: x.Email})
		}
	}
	if ms, err := s.Q.IntegrationMembers(ctx, ws.ID); err == nil {
		for _, m := range ms {
			out.Members = append(out.Members, IntegrationOption{ID: m.ID.String(), Name: m.Name, Detail: m.RoleName})
		}
	}
	formRows, err := s.Q.WordPressForms(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	connectors := map[string]bool{}
	waiting := 0
	for _, l := range conns {
		if l.conn.Kind != KindWordPress || l.site == nil {
			continue
		}
		site := s.siteView(ctx, ws, l, out, formRows)
		connectors[site.ConnectedBy] = true
		if site.Status == StatusWaiting {
			waiting++
		}
		out.Reasons = append(out.Reasons, site.Reasons...)
		out.UpdateAvailable = out.UpdateAvailable || site.UpdateAvailable
		out.Sites = append(out.Sites, site)
	}
	switch {
	case len(out.Sites) == 0:
	case len(out.Reasons) > 0:
		out.Status = StatusAttention
	case waiting == len(out.Sites):
		out.Status = StatusWaiting
	default:
		out.Status = StatusConnected
	}
	if len(connectors) == 1 {
		for name := range connectors {
			out.ConnectedBy = name
		}
	}
	return out, nil
}

func (s *Service) siteView(ctx context.Context, ws *store.Workspace, l loaded, page IntegrationWordPressPage, formRows []store.WordPressFormsRow) IntegrationSite {
	st := l.site
	host := siteBareHost(firstNonEmpty(st.SiteUrl, st.Address))
	v := IntegrationSite{ID: l.conn.ID.String(), Host: l.conn.Label, SiteURL: st.SiteUrl, Name: st.Name, Status: l.status(), PluginVersion: st.PluginVersion,
		WPVersion: st.WpVersion, PHPVersion: st.PhpVersion, FirstReportAt: st.FirstReportAt, LastReportAt: st.LastReportAt, FeedURL: st.FeedUrl,
		LanguagePlugin: st.LanguagePlugin, Forms: []IntegrationSiteForm{}, Categories: []WordPressCategory{}, Drafts: []IntegrationSiteDraft{},
		ConnectedBy: l.by, ConnectedAt: l.conn.ConnectedAt, Reasons: l.rs, Items: s.itemStates(ctx, ws, l)}
	v.UpdateAvailable = st.PluginVersion != "" && newerVersion(page.LatestPlugin, st.PluginVersion)
	if st.WoocommerceVersion != "" {
		v.WooCommerce = &IntegrationSiteWoo{Version: st.WoocommerceVersion, Consent: st.WoocommerceConsent, Checkouts: st.WoocommerceCheckouts}
		if r, err := s.Q.CommerceStoreForSite(ctx, store.CommerceStoreForSiteParams{WorkspaceID: ws.ID, Host: host}); err == nil {
			v.WooCommerce.StoreID, v.WooCommerce.StoreName, v.WooCommerce.StoreStatus = r.ID.String(), r.Name, r.Status
		}
	}
	if c := lastCheck(st); c != nil {
		v.LastCheck = c
	}
	_ = json.Unmarshal(st.Categories, &v.Categories)
	if v.Categories == nil {
		v.Categories = []WordPressCategory{}
	}
	var used []WordPressFormUse
	_ = json.Unmarshal(st.Forms, &used)
	for _, u := range used {
		f := IntegrationSiteForm{Public: u.Form, Kind: u.Kind, Places: u.Places, Missing: true}
		if f.Places == nil {
			f.Places = []WordPressFormPlace{}
		}
		for _, row := range formRows {
			if row.PublicID == u.Form {
				f.ID, f.Name, f.Missing = row.ID.String(), row.Name, false
				f.Allowed = forms.OriginAllowed(host, row.AllowedOrigins, nil)
			}
		}
		if !f.Missing && !f.Allowed {
			v.SignupsRejected = true
		}
		v.Forms = append(v.Forms, f)
	}
	rule := parseRule(st.Rule)
	v.Rule = IntegrationSiteRule{Enabled: st.RuleEnabled, Set: len(rule.ListIDs) > 0, Categories: []WordPressCategory{}, ListIDs: rule.ListIDs}
	for _, slug := range rule.Categories {
		c := WordPressCategory{Slug: slug, Name: slug}
		for _, k := range v.Categories {
			if k.Slug == slug {
				c = k
			}
		}
		v.Rule.Categories = append(v.Rule.Categories, c)
	}
	for _, l := range page.Lists {
		if len(rule.ListIDs) > 0 && l.ID == rule.ListIDs[0] {
			lc := l
			v.Rule.List = &lc
		}
	}
	for _, t := range page.Templates {
		if t.ID == rule.TemplateID {
			tc := t
			v.Rule.Template = &tc
		}
	}
	for _, x := range page.Senders {
		if x.ID == rule.SenderID {
			xc := x
			v.Rule.Sender = &xc
		}
	}
	for _, m := range page.Members {
		if m.ID == rule.NotifyUserID {
			mc := m
			v.Rule.Notify = &mc
		}
	}
	if drafts, err := s.Q.WordPressDrafts(ctx, store.WordPressDraftsParams{IntegrationID: l.conn.ID, Lim: 3}); err == nil {
		for _, d := range drafts {
			dv := IntegrationSiteDraft{Title: d.PostTitle, Status: d.CampaignStatus, Audience: d.AudienceName, PublishedAt: d.PublishedAt, CreatedAt: d.CreatedAt,
				Notified: d.NotifiedName, PostURL: d.PostUrl}
			if d.CampaignID != nil && d.CampaignStatus != "" {
				dv.CampaignID = d.CampaignID.String()
			}
			v.Drafts = append(v.Drafts, dv)
		}
	}
	if n, err := s.Q.WordPressDraftsWeek(ctx, store.WordPressDraftsWeekParams{IntegrationID: l.conn.ID, Since: s.now().Add(-7 * 24 * time.Hour)}); err == nil {
		v.DraftsThisWeek = n
	}
	lists, _ := access.ListNames(ctx, s.Q, ws.ID)
	cur, old := current(l.keys, s.now())
	if cur != nil {
		k := keyView(*cur, lists, s.now())
		v.Key = &k
	}
	if old != nil {
		k := keyView(*old, lists, s.now())
		v.OldKey = &k
	}
	return v
}
