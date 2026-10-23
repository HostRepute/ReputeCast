package earn

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// UpgradePage is the upgrade page as readers see it (Earn › Plans &
// offers › Edit page): the publisher's copy and the page's choices.
type UpgradePage struct {
	Headline string `json:"headline" maxLength:"120"`
	Intro    string `json:"intro" maxLength:"600"`
	// Order lists the tiers in the page's order; Off are public tiers left
	// off the page. Hidden and archived tiers never show.
	Order         []string `json:"order" maxItems:"20"`
	Off           []string `json:"off" maxItems:"20"`
	DefaultPeriod string   `json:"defaultPeriod" enum:"month,year"`
	// Highlight is the tier put forward, with its label ("Most popular").
	Highlight      string `json:"highlight" maxLength:"40"`
	HighlightLabel string `json:"highlightLabel" maxLength:"30"`
	// The free card: its list, its line ("the Weekly Digest, every
	// Tuesday") and its text.
	FreeList string            `json:"freeList" maxLength:"40"`
	FreeLine string            `json:"freeLine" maxLength:"120"`
	FreeText string            `json:"freeText" maxLength:"300"`
	FAQ      []UpgradeFAQ      `json:"faq" maxItems:"8"`
	Welcome  string            `json:"welcome" maxLength:"400" doc:"The welcome page after checkout; {{ first_name }} and {{ next_issue }} are filled in."`
	Buttons  map[string]string `json:"buttons,omitempty" doc:"A tier's button when it has no trial (\"Become a founding member\"), by tier id."`
}

// UpgradeFAQ is one question on the upgrade page.
type UpgradeFAQ struct {
	Q string `json:"q" maxLength:"160"`
	A string `json:"a" maxLength:"800"`
}

func (s *Service) upgradePage(ctx context.Context, ws *store.Workspace) (UpgradePage, error) {
	return s.upgradePageTx(ctx, s.Q, ws)
}

// upgradePageIn is the page with what was never set in a language: the
// page's own, as its readers read it.
func (s *Service) upgradePageIn(ctx context.Context, ws *store.Workspace, l i18n.Lang) (UpgradePage, error) {
	return s.upgradePageTxIn(ctx, s.Q, ws, l)
}

// upgradePageTx reads the page, filling what was never set with the
// defaults (the headline names the first list), in English.
func (s *Service) upgradePageTx(ctx context.Context, q *store.Queries, ws *store.Workspace) (UpgradePage, error) {
	return s.upgradePageTxIn(ctx, q, ws, defaultLang)
}

func (s *Service) upgradePageTxIn(ctx context.Context, q *store.Queries, ws *store.Workspace, l i18n.Lang) (UpgradePage, error) {
	st, err := s.settingsRow(ctx, q, ws.ID)
	if err != nil {
		return UpgradePage{}, err
	}
	var p UpgradePage
	_ = json.Unmarshal(st.UpgradePage, &p)
	if p.DefaultPeriod == "" {
		p.DefaultPeriod = "year"
	}
	if p.Order == nil {
		p.Order = []string{}
	}
	if p.Off == nil {
		p.Off = []string{}
	}
	if p.FAQ == nil {
		p.FAQ = []UpgradeFAQ{}
	}
	if p.FreeList == "" || p.Headline == "" {
		lists, err := q.EarnLists(ctx, ws.ID)
		if err == nil && len(lists) > 0 {
			best := lists[0]
			for _, l := range lists {
				if l.Subscribers > best.Subscribers {
					best = l
				}
			}
			if p.FreeList == "" {
				p.FreeList = best.ID.String()
			}
			if p.Headline == "" {
				p.Headline = msgs.M("page.defaultHeadline").Text(l, map[string]any{"name": best.Name})
			}
			if p.FreeLine == "" {
				p.FreeLine = msgs.M("page.defaultFreeLine").Text(l, map[string]any{"name": best.Name})
			}
		}
	}
	if p.Headline == "" {
		p.Headline = msgs.M("page.defaultHeadline").Text(l, map[string]any{"name": ws.Name})
	}
	if p.Welcome == "" {
		p.Welcome = msgs.M("page.defaultWelcome").Text(l, nil)
	}
	return p, nil
}

// EarnPageTier is a tier as the page editor lists it.
type EarnPageTier struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility" enum:"public,hidden,archived"`
	Shown      bool   `json:"shown"`
	TrialDays  int    `json:"trialDays"`
}

// EarnUpgradeEditor is the upgrade page editor's data.
type EarnUpgradeEditor struct {
	Page        UpgradePage    `json:"page"`
	Tiers       []EarnPageTier `json:"tiers"`
	Lists       []EarnListRef  `json:"lists"`
	URL         string         `json:"url"`
	Host        string         `json:"host"`
	Live        bool           `json:"live"`
	PublishedAt *string        `json:"publishedAt,omitempty"`
	PublishedBy string         `json:"publishedBy,omitempty"`
	// TestCheckout: a test key is connected, so "Preview with test
	// checkout" works.
	TestCheckout    bool     `json:"testCheckout"`
	TestCheckoutURL string   `json:"testCheckoutUrl,omitempty"`
	CanEdit         bool     `json:"canEdit"`
	Helpers         []string `json:"helpers"`
}

// UpgradeEditor is the page editor's data.
func (s *Service) UpgradeEditor(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnUpgradeEditor, error) {
	// What was never set shows as readers get it: in the workspace's
	// default language.
	page, err := s.upgradePageIn(ctx, ws, workspaceLang(ctx, s.Q, ws.ID))
	if err != nil {
		return EarnUpgradeEditor{}, err
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return EarnUpgradeEditor{}, err
	}
	out := EarnUpgradeEditor{Page: page, Tiers: []EarnPageTier{}, Lists: []EarnListRef{}, URL: s.pageURL(ctx, ws, "upgrade"), Host: s.pageHost(ctx, ws, "upgrade"),
		CanEdit: p.Can(authz.EarnEdit), Helpers: s.helpers(ctx, ws.ID, authz.EarnEdit, p.UserID), TestCheckoutURL: s.TestCheckoutURL(ctx, ws)}
	for _, t := range orderTiers(tiers, page) {
		out.Tiers = append(out.Tiers, EarnPageTier{ID: t.ID.String(), Name: t.Name, Visibility: t.Visibility, TrialDays: int(t.TrialDays),
			Shown: t.Visibility == VisibilityPublic && !slices.Contains(page.Off, t.ID.String())})
		if t.Visibility == VisibilityPublic && !slices.Contains(page.Off, t.ID.String()) {
			out.Live = true
		}
	}
	lists, err := s.Q.EarnLists(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, EarnListRef{ID: l.ID.String(), Name: l.Name})
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	if st.UpgradePublishedAt != nil {
		v := st.UpgradePublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		out.PublishedAt = &v
		if st.UpgradePublishedBy != nil {
			out.PublishedBy, _ = s.Q.EarnUserName(ctx, *st.UpgradePublishedBy)
		}
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	_, out.TestCheckout = connected(conns, "test")
	if !out.TestCheckout {
		// Without a test key, the live account takes no test card.
		out.TestCheckout = false
	}
	_, live := active(conns)
	out.Live = out.Live && live
	return out, nil
}

// orderTiers puts tiers in the page's order: the ones it lists first, the
// rest by rank.
func orderTiers(tiers []store.EarnTier, page UpgradePage) []store.EarnTier {
	pos := map[string]int{}
	for i, id := range page.Order {
		pos[id] = i
	}
	out := slices.Clone(tiers)
	slices.SortStableFunc(out, func(a, b store.EarnTier) int {
		pa, oka := pos[a.ID.String()]
		pb, okb := pos[b.ID.String()]
		switch {
		case oka && okb:
			return pa - pb
		case oka:
			return -1
		case okb:
			return 1
		}
		return int(a.AccessRank - b.AccessRank)
	})
	return out
}

// SaveUpgradePage publishes the page: readers see it at once.
func (s *Service) SaveUpgradePage(ctx context.Context, ws *store.Workspace, p *authz.Principal, in UpgradePage) (EarnUpgradeEditor, error) {
	before, err := s.upgradePage(ctx, ws)
	if err != nil {
		return EarnUpgradeEditor{}, err
	}
	in.Headline, in.Intro = strings.TrimSpace(in.Headline), strings.TrimSpace(in.Intro)
	if in.Headline == "" {
		return EarnUpgradeEditor{}, invalid(ctx, "body.headline", "page.headline", nil)
	}
	if in.DefaultPeriod != "month" {
		in.DefaultPeriod = "year"
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return EarnUpgradeEditor{}, err
	}
	known := map[string]bool{}
	for _, t := range tiers {
		known[t.ID.String()] = true
	}
	keep := func(ids []string) []string {
		out := []string{}
		for _, id := range ids {
			if known[id] && !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
		return out
	}
	in.Order, in.Off = keep(in.Order), keep(in.Off)
	if in.Highlight != "" && !known[in.Highlight] {
		in.Highlight, in.HighlightLabel = "", ""
	}
	if in.Highlight == "" {
		in.HighlightLabel = ""
	}
	faq := []UpgradeFAQ{}
	for _, f := range in.FAQ {
		if f.Q = strings.TrimSpace(f.Q); f.Q != "" {
			f.A = strings.TrimSpace(f.A)
			faq = append(faq, f)
		}
	}
	in.FAQ = faq
	for k := range in.Buttons {
		if !known[k] || strings.TrimSpace(in.Buttons[k]) == "" {
			delete(in.Buttons, k)
		}
	}
	b, _ := json.Marshal(in)
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.EarnSettingsSetUpgradePage(ctx, store.EarnSettingsSetUpgradePageParams{WorkspaceID: ws.ID, UpgradePage: b, At: &now, By: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.upgrade_page_updated", TargetType: "earn_page", TargetID: "upgrade",
			TargetName: s.pageHost(ctx, ws, "upgrade"), Before: pageAudit(before), After: pageAudit(in)})
	})
	if err != nil {
		return EarnUpgradeEditor{}, err
	}
	return s.UpgradeEditor(ctx, ws, p)
}

func pageAudit(p UpgradePage) map[string]any {
	return map[string]any{"headline": p.Headline, "intro": p.Intro, "tiers": p.Order, "off": p.Off, "default_period": p.DefaultPeriod,
		"highlight": p.Highlight, "highlight_label": p.HighlightLabel, "free_list": p.FreeList, "faq": len(p.FAQ), "welcome": p.Welcome}
}

// trialNote, buttons and prices are worded per card.

// yearlySaving is how much a year saves against twelve months, in whole
// months ("2 months free"), when it saves at least one.
func yearlySaving(monthly, yearly int64) int {
	if monthly <= 0 || yearly <= 0 || yearly >= 12*monthly {
		return 0
	}
	return int((12*monthly - yearly) / monthly)
}

func perMonth(yearly int64, currency string) string {
	return money((yearly+6)/12, currency, false)
}

var _ = fmt.Sprint
var _ = uuid.Nil
