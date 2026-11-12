package coupons

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dependents"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

var (
	slugRe   = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)
	prefixRe = regexp.MustCompile(`^[A-Z0-9-]{0,12}$`)
)

// CouponPoolStore is the store a pool's codes are created in.
type CouponPoolStore struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Platform  string `json:"platform" enum:"shopify,woocommerce"`
	Domain    string `json:"domain"`
	Connected bool   `json:"connected"`
}

// CouponPoolListItem is one row of Campaigns › Discount codes. Codes are
// never in it.
type CouponPoolListItem struct {
	ID     string           `json:"id" format:"uuid"`
	Name   string           `json:"name"`
	Slug   string           `json:"slug" doc:"The name templates use: {{ coupons.<slug>.code }}"`
	Source string           `json:"source" enum:"upload,shopify,woocommerce"`
	Store  *CouponPoolStore `json:"store,omitempty"`
	Status string           `json:"status" enum:"active,archived"`
	Prefix string           `json:"prefix"`
	// Total are the codes the pool holds (not removed); Free nobody got.
	Total        int64  `json:"total"`
	Free         int64  `json:"free"`
	Sent         int64  `json:"sent"`
	Redeemed     int64  `json:"redeemed"`
	RevenueCents int64  `json:"revenueCents" doc:"Orders that used the pool's codes: refunds taken off, cancelled and test orders left out"`
	Currency     string `json:"currency"`
	LowAt        int    `json:"lowAt"`
	Low          bool   `json:"low" doc:"Free codes are under the low mark"`
	// Settings say what a store pool's discount takes off, or where an
	// upload pool's codes apply.
	Settings      CouponPoolSettings `json:"settings"`
	OnePerContact bool               `json:"onePerContact"`
	// Running is the request to create codes in the store under way.
	Running   *CouponBatch `json:"running,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
}

// CouponPoolList is the pools of a workspace, active first.
type CouponPoolList struct {
	Items []CouponPoolListItem `json:"items"`
	// Helpers are members who can manage discount codes (who to ask).
	Helpers []string `json:"helpers"`
}

// CouponBatch is an upload or a request to create codes.
type CouponBatch struct {
	ID   string `json:"id" format:"uuid"`
	Kind string `json:"kind" enum:"upload,generate"`
	// Reason: the pool's first codes, a member asking for more, or the
	// hourly refill (Automation names the live flow it refilled for).
	Reason     string `json:"reason" enum:"pool,request,refill"`
	Automation string `json:"automation,omitempty"`
	By         string `json:"by,omitempty" doc:"Who asked, by name"`
	Requested  int    `json:"requested"`
	Created    int    `json:"created"`
	// Failed are codes the store refused or the pool had already.
	Failed int    `json:"failed"`
	Status string `json:"status" enum:"running,done,failed"`
	// Error says why the batch failed, or why a running one waits to try
	// again: unreachable, not_granted, read_only, not_connected,
	// unsupported, store_gone, archived or ended.
	Error string `json:"error,omitempty"`
	// WaitingUntil: the store asked ReputeCast to slow down; it goes on then.
	WaitingUntil *time.Time `json:"waitingUntil,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

// CouponPoolUse is a campaign, automation email or template that shows
// the pool's codes.
type CouponPoolUse struct {
	Kind string `json:"kind" enum:"campaign,automation,template"`
	// ID is the campaign's, the automation's (with EmailID) or the
	// template's.
	ID   string `json:"id"`
	Name string `json:"name"`
	// EmailID and EmailName are the automation's email.
	EmailID   string `json:"emailId,omitempty"`
	EmailName string `json:"emailName,omitempty"`
	// Status is the campaign's or the automation's.
	Status string `json:"status,omitempty"`
	// Lists and Recipients are a campaign's audience.
	Lists      string `json:"lists,omitempty"`
	Recipients int64  `json:"recipients"`
	// Block: the Discount code block shows the code (else Liquid does).
	Block bool `json:"block"`
	// Live: an automation that's live (archiving the pool asks first).
	Live bool `json:"live"`
}

// CouponPoolWaiting is a flow whose emails wait for one of the pool's codes.
type CouponPoolWaiting struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Emails int64  `json:"emails"`
}

// CouponPoolTokens are the Liquid a template shows the pool's code with.
type CouponPoolTokens struct {
	Code string `json:"code" example:"{{ coupons.welcome15.code }}"`
	URL  string `json:"url" example:"{{ coupons.welcome15.url }}"`
}

// CouponPool is a pool's page.
type CouponPool struct {
	CouponPoolListItem
	Removed int64            `json:"removed"`
	Tokens  CouponPoolTokens `json:"tokens"`
	// SampleURL is the code's link with the sample code previews show
	// ("" without an address).
	SampleURL string `json:"sampleUrl"`
	// Orders that used the pool's codes; FirstSentAt when it first gave one.
	Orders      int64      `json:"orders"`
	FirstSentAt *time.Time `json:"firstSentAt,omitempty"`
	CreatedBy   string     `json:"createdBy" doc:"Who created the pool, by name"`
	// Batch is the latest request to create codes (running or not);
	// Batches every upload and request, newest first.
	Batch   *CouponBatch    `json:"batch,omitempty"`
	Batches []CouponBatch   `json:"batches"`
	UsedIn  []CouponPoolUse `json:"usedIn"`
	// Refill: a live automation uses the store pool, so the hourly job
	// keeps it topped up; RefillCheckedAt is when it last looked.
	Refill          bool       `json:"refill"`
	RefillCheckedAt *time.Time `json:"refillCheckedAt,omitempty"`
	// Waiting are the flows whose emails wait for a code from the pool.
	Waiting []CouponPoolWaiting `json:"waiting"`
	// Helpers are members who can manage discount codes (who to ask).
	Helpers   []string  `json:"helpers"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CouponPoolInput creates a pool.
type CouponPoolInput struct {
	Name          string             `json:"name" maxLength:"200"`
	Slug          string             `json:"slug" maxLength:"64" doc:"1–32 lowercase letters, digits or _; never changes"`
	Source        string             `json:"source" enum:"upload,shopify,woocommerce"`
	StoreID       string             `json:"storeId,omitempty" maxLength:"40"`
	Settings      CouponPoolSettings `json:"settings,omitempty"`
	Prefix        string             `json:"prefix,omitempty" maxLength:"40"`
	OnePerContact *bool              `json:"onePerContact,omitempty" doc:"Default true"`
	LowAt         *int               `json:"lowAt,omitempty" doc:"Default 200"`
}

// CouponPoolChange changes a pool; missing fields stay.
type CouponPoolChange struct {
	Name          *string             `json:"name,omitempty" maxLength:"200"`
	Settings      *CouponPoolSettings `json:"settings,omitempty"`
	Prefix        *string             `json:"prefix,omitempty" maxLength:"40"`
	OnePerContact *bool               `json:"onePerContact,omitempty"`
	LowAt         *int                `json:"lowAt,omitempty"`
}

// CouponSource is a store as a source of codes for the New pool dialog.
type CouponSource struct {
	StoreID  string `json:"storeId"`
	Platform string `json:"platform" enum:"shopify,woocommerce"`
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	Currency string `json:"currency"`
	Can      bool   `json:"can"`
	// Reason and Detail say why a store can't create codes.
	Reason string `json:"reason,omitempty" enum:"not_connected,not_granted,unsupported,read_only,"`
	Detail string `json:"detail,omitempty"`
}

// CouponSourceList is the stores a pool can create codes in.
type CouponSourceList struct {
	Items []CouponSource `json:"items"`
}

// counts are a pool's numbers.
type counts struct {
	total, free, sent, redeemed, removed, revenue, orders int64
	currency                                              string
	firstSent                                             *time.Time
}

func (s *Service) pools(ctx context.Context, ws uuid.UUID, id *uuid.UUID, slugs []string) ([]store.CouponPoolsRow, error) {
	return s.Q.CouponPools(ctx, store.CouponPoolsParams{WorkspaceID: ws, ID: id, Slugs: slugs})
}

func (s *Service) pool(ctx context.Context, ws uuid.UUID, id uuid.UUID) (store.CouponPoolsRow, error) {
	rows, err := s.pools(ctx, ws, &id, nil)
	if err != nil {
		return store.CouponPoolsRow{}, err
	}
	if len(rows) == 0 {
		return store.CouponPoolsRow{}, notFound(ctx)
	}
	return rows[0], nil
}

func (s *Service) counts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]counts, error) {
	out := map[uuid.UUID]counts{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.CouponPoolCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.PoolID] = counts{total: r.Total, free: r.Free, sent: r.Sent, redeemed: r.Redeemed, removed: r.Removed, firstSent: unixTime(r.FirstSentUnix)}
	}
	rev, err := s.Q.CouponPoolRevenue(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rev {
		c := out[r.PoolID]
		c.revenue, c.currency, c.orders = r.RevenueCents, r.Currency, r.Orders
		out[r.PoolID] = c
	}
	return out, nil
}

func deref(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}

// unixTime is a moment from seconds since 1970 (nil for 0).
func unixTime(sec int64) *time.Time {
	if sec == 0 {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

func listItem(p store.CouponPoolsRow, c counts, running *CouponBatch) CouponPoolListItem {
	it := CouponPoolListItem{ID: p.ID.String(), Name: p.Name, Slug: p.Slug, Source: p.Source, Status: p.Status, Prefix: p.Prefix,
		Total: c.total, Free: c.free, Sent: c.sent, Redeemed: c.redeemed, RevenueCents: c.revenue, Currency: c.currency,
		LowAt: int(p.LowAt), Low: p.Status == StatusActive && c.free < int64(p.LowAt), Settings: settingsOf(p.Settings), OnePerContact: p.OnePerContact,
		Running: running, CreatedAt: p.CreatedAt}
	if p.StoreID != nil {
		it.Store = &CouponPoolStore{ID: p.StoreID.String(), Name: p.StoreName, Platform: p.StorePlatform, Domain: storeHost(p), Connected: p.StoreStatus == "connected"}
	}
	if it.Currency == "" {
		it.Currency = p.StoreCurrency
	}
	return it
}

// storeHost is the storefront's host (its own domain when it has one).
func storeHost(p store.CouponPoolsRow) string {
	if p.StorePublicDomain != "" {
		return p.StorePublicDomain
	}
	return p.StoreDomain
}

// List is the workspace's pools with their numbers.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (CouponPoolList, error) {
	out := CouponPoolList{Items: []CouponPoolListItem{}, Helpers: s.helpers(ctx, ws.ID)}
	rows, err := s.pools(ctx, ws.ID, nil, nil)
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	cs, err := s.counts(ctx, ids)
	if err != nil {
		return out, err
	}
	running, err := s.Q.CouponBatchesRunning(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byPool := make(map[uuid.UUID]*CouponBatch, len(running))
	for _, b := range running {
		byPool[b.PoolID] = batchView(b, "", "")
	}
	for _, r := range rows {
		out.Items = append(out.Items, listItem(r, cs[r.ID], byPool[r.ID]))
	}
	return out, nil
}

// Get is a pool's page: its numbers, settings, latest batch and what uses it.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CouponPool, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponPool{}, err
	}
	cs, err := s.counts(ctx, []uuid.UUID{p.ID})
	if err != nil {
		return CouponPool{}, err
	}
	c := cs[p.ID]
	out := CouponPool{Removed: c.removed, UpdatedAt: p.UpdatedAt, Orders: c.orders, FirstSentAt: c.firstSent, CreatedBy: p.CreatedByName,
		Tokens:    CouponPoolTokens{Code: "{{ " + Root + "." + p.Slug + ".code }}", URL: "{{ " + Root + "." + p.Slug + ".url }}"},
		SampleURL: codeURL(p, sample(p.Prefix)), RefillCheckedAt: p.RefillCheckedAt, Batches: []CouponBatch{}, UsedIn: []CouponPoolUse{},
		Helpers: s.helpers(ctx, ws.ID)}
	batches, err := s.Q.CouponBatches(ctx, p.ID)
	if err != nil {
		return CouponPool{}, err
	}
	var running *CouponBatch
	for _, b := range batches {
		v := batchView(store.CouponBatch{ID: b.ID, Kind: b.Kind, Reason: b.Reason, Requested: b.Requested, Created: b.Created, Failed: b.Failed,
			Status: b.Status, Error: b.Error, WaitingUntil: b.WaitingUntil, CreatedAt: b.CreatedAt, FinishedAt: b.FinishedAt}, b.CreatedByName, b.AutomationName)
		out.Batches = append(out.Batches, *v)
		if b.Kind == "generate" && out.Batch == nil {
			out.Batch = v
		}
		if b.Kind == "generate" && b.Status == "running" {
			running = v
		}
	}
	out.CouponPoolListItem = listItem(p, c, running)
	if out.UsedIn, err = s.usedIn(ctx, ws.ID, p.Slug); err != nil {
		return CouponPool{}, err
	}
	var flows []uuid.UUID
	for _, u := range out.UsedIn {
		if u.Kind == dependents.KindAutomation {
			if id, err := uuid.Parse(u.ID); err == nil && !slices.Contains(flows, id) {
				flows = append(flows, id)
			}
		}
		out.Refill = out.Refill || (u.Live && p.Source != SourceUpload)
	}
	out.Waiting = []CouponPoolWaiting{}
	if len(flows) > 0 {
		held, err := s.Q.CouponHeldRuns(ctx, store.CouponHeldRunsParams{WorkspaceID: ws.ID, Ids: flows})
		if err != nil {
			return CouponPool{}, err
		}
		for _, h := range held {
			out.Waiting = append(out.Waiting, CouponPoolWaiting{ID: h.ID.String(), Name: h.Name, Emails: h.Runs})
		}
	}
	return out, nil
}

func batchView(b store.CouponBatch, by, automation string) *CouponBatch {
	return &CouponBatch{ID: b.ID.String(), Kind: b.Kind, Reason: b.Reason, Automation: automation, By: by, Requested: int(b.Requested), Created: int(b.Created),
		Failed: int(b.Failed), Status: b.Status, Error: b.Error, WaitingUntil: b.WaitingUntil, CreatedAt: b.CreatedAt, FinishedAt: b.FinishedAt}
}

// usedIn are the campaigns, automation emails and templates that show a
// pool's codes: campaigns with their audience, emails with their flow (a
// live flow's marked), saved templates.
func (s *Service) usedIn(ctx context.Context, ws uuid.UUID, slug string) ([]CouponPoolUse, error) {
	uses, err := dependents.Coupon(ctx, s.Pool, ws, slug)
	if err != nil {
		return nil, err
	}
	var campaignIDs []uuid.UUID
	out := []CouponPoolUse{}
	var templates []CouponPoolUse
	for _, u := range uses {
		switch u.Kind {
		case dependents.KindCampaign:
			if id, err := uuid.Parse(u.ID); err == nil {
				campaignIDs = append(campaignIDs, id)
			}
		case dependents.KindTemplate:
			templates = append(templates, CouponPoolUse{Kind: u.Kind, ID: u.ID, Name: u.Name})
		}
	}
	if len(campaignIDs) > 0 {
		rows, err := s.Q.CouponUsedCampaigns(ctx, store.CouponUsedCampaignsParams{WorkspaceID: ws, Ids: campaignIDs})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, CouponPoolUse{Kind: dependents.KindCampaign, ID: r.ID.String(), Name: r.Name, Status: r.Status, Lists: r.ListNames,
				Recipients: deref(r.AudienceCount), Block: usesBlock(r.Content, slug)})
		}
	}
	live, err := s.liveAutomations(ctx, ws, slug)
	if err != nil {
		return nil, err
	}
	emails, err := s.Q.CouponUsedAutomationEmails(ctx, store.CouponUsedAutomationEmailsParams{WorkspaceID: ws, Needle: slug})
	if err != nil {
		return nil, err
	}
	for _, e := range emails {
		if !names(e.Content, slug) {
			continue
		}
		_, isLive := live[e.AutomationID.String()]
		out = append(out, CouponPoolUse{Kind: dependents.KindAutomation, ID: e.AutomationID.String(), Name: e.AutomationName, EmailID: e.ID.String(),
			EmailName: e.Name, Status: e.AutomationStatus, Block: usesBlock(e.Content, slug), Live: isLive && e.AutomationStatus == "live"})
	}
	return append(out, templates...), nil
}

// names reports whether content shows the pool's codes: Liquid reading
// coupons.<slug> or a Discount code block set to it.
func names(content, slug string) bool {
	return usesBlock(content, slug) || contains(usedSlugs(content), slug)
}

// usesBlock reports whether content has a Discount code block for the pool.
func usesBlock(content, slug string) bool {
	if !strings.Contains(content, "rc:coupon") {
		return false
	}
	for _, m := range blockRe.FindAllStringSubmatch(content, -1) {
		if parseBlock([]byte(m[1])).Pool == slug {
			return true
		}
	}
	return false
}

// blockRe finds Discount code blocks' settings in content.
var blockRe = regexp.MustCompile(`<!--\s*rc:coupon\s+(\{[^>]*?\})\s*-->`)

// liveAutomations are the live automations whose live version sends an
// email showing the pool's codes, by id, with their names.
func (s *Service) liveAutomations(ctx context.Context, ws uuid.UUID, slug string) (map[string]string, error) {
	rows, err := s.Q.CouponLiveAutomationEmails(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		if names(r.Content, slug) {
			out[r.ID.String()] = r.Name
		}
	}
	return out, nil
}

// Sources are the workspace's stores and whether each can create codes.
func (s *Service) SourceList(ctx context.Context, ws *store.Workspace) (CouponSourceList, error) {
	out := CouponSourceList{Items: []CouponSource{}}
	if s.Sources == nil {
		return out, nil
	}
	src, err := s.Sources(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, x := range src {
		out.Items = append(out.Items, CouponSource{StoreID: x.StoreID.String(), Platform: x.Platform, Name: x.Name, Domain: x.Domain, Currency: x.Currency,
			Can: x.Can, Reason: x.Reason, Detail: x.Detail})
	}
	return out, nil
}

// draft is a pool's settings once checked.
type draft struct {
	name, slug, source, prefix string
	storeID                    *uuid.UUID
	settings                   CouponPoolSettings
	onePerContact              bool
	lowAt                      int
}

// checkSettings checks a pool's settings for its source.
func (s *Service) checkSettings(ctx context.Context, source string, set CouponPoolSettings) (CouponPoolSettings, error) {
	if source == SourceUpload {
		out := CouponPoolSettings{ApplyURL: strings.TrimSpace(set.ApplyURL), CollectionIDs: []string{}}
		if out.ApplyURL != "" {
			u, err := url.Parse(strings.ReplaceAll(out.ApplyURL, "{code}", "CODE"))
			if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || !strings.Contains(out.ApplyURL, "{code}") || len(out.ApplyURL) > 500 {
				return out, invalid(ctx, "body.settings.applyUrl", mPoolApplyURL, i18n.Args{"placeholder": "{code}"})
			}
		}
		return out, nil
	}
	out := CouponPoolSettings{Kind: set.Kind, Value: set.Value, MinimumCents: set.MinimumCents, EndsDays: set.EndsDays, CollectionIDs: []string{}}
	switch out.Kind {
	case KindPercentage:
		if out.Value < 1 || out.Value > 100 {
			return out, invalid(ctx, "body.settings.value", mPoolPercent, nil)
		}
	case KindFixedAmount:
		if out.Value < 1 || out.Value > 100_000_000 {
			return out, invalid(ctx, "body.settings.value", mPoolAmount, nil)
		}
	case KindFreeShipping:
		out.Value = 0
	default:
		return out, invalid(ctx, "body.settings.kind", mPoolKindMissing, nil)
	}
	if out.MinimumCents < 0 || out.MinimumCents > 100_000_000 {
		return out, invalid(ctx, "body.settings.minimumCents", mPoolMinimum, nil)
	}
	if out.EndsDays < 0 || out.EndsDays > 365 {
		return out, invalid(ctx, "body.settings.endsDays", mPoolEndsDays, nil)
	}
	for _, c := range set.CollectionIDs {
		if c = strings.TrimSpace(c); c != "" && len(c) <= 100 && !slices.Contains(out.CollectionIDs, c) {
			out.CollectionIDs = append(out.CollectionIDs, c)
		}
	}
	if len(out.CollectionIDs) > 50 {
		return out, invalid(ctx, "body.settings.collectionIds", mPoolCollections, nil)
	}
	return out, nil
}

func (s *Service) checkCommon(ctx context.Context, d *draft) error {
	d.name = strings.TrimSpace(d.name)
	switch {
	case d.name == "":
		return invalid(ctx, "body.name", mPoolNameMissing, nil)
	case len([]rune(d.name)) > 80:
		return invalid(ctx, "body.name", mPoolNameLong, nil)
	}
	d.prefix = strings.ToUpper(strings.TrimSpace(d.prefix))
	if !prefixRe.MatchString(d.prefix) {
		return invalid(ctx, "body.prefix", mPoolPrefixFormat, i18n.Args{"example": "WELCOME-"})
	}
	if d.lowAt < 0 || d.lowAt > 1_000_000 {
		return invalid(ctx, "body.lowAt", mPoolLowAt, nil)
	}
	return nil
}

// snapshot is a pool as the audit log keeps it.
func snapshot(d draft) map[string]any {
	out := map[string]any{"name": d.name, "slug": d.slug, "source": d.source, "prefix": d.prefix, "onePerContact": d.onePerContact, "lowAt": d.lowAt,
		"settings": d.settings}
	if d.storeID != nil {
		out["storeId"] = d.storeID.String()
	}
	return out
}

func draftOf(p store.CouponPoolsRow) draft {
	return draft{name: p.Name, slug: p.Slug, source: p.Source, prefix: p.Prefix, storeID: p.StoreID, settings: settingsOf(p.Settings),
		onePerContact: p.OnePerContact, lowAt: int(p.LowAt)}
}

// Create adds a pool.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in CouponPoolInput) (CouponPool, error) {
	d := draft{name: in.Name, slug: strings.TrimSpace(in.Slug), source: in.Source, prefix: in.Prefix, onePerContact: true, lowAt: 200}
	if in.OnePerContact != nil {
		d.onePerContact = *in.OnePerContact
	}
	if in.LowAt != nil {
		d.lowAt = *in.LowAt
	}
	if err := s.checkCommon(ctx, &d); err != nil {
		return CouponPool{}, err
	}
	if !slugRe.MatchString(d.slug) {
		return CouponPool{}, invalid(ctx, "body.slug", mPoolSlugFormat, i18n.Args{"example": "welcome15"})
	}
	storeID := strings.TrimSpace(in.StoreID)
	if d.source == SourceUpload {
		if storeID != "" {
			return CouponPool{}, invalid(ctx, "body.storeId", mPoolUploadStore, nil)
		}
	} else {
		if storeID == "" {
			return CouponPool{}, invalid(ctx, "body.storeId", mPoolStoreMissing, nil)
		}
		id, err := uuid.Parse(storeID)
		if err != nil {
			return CouponPool{}, invalid(ctx, "body.storeId", mPoolStoreUnknown, nil)
		}
		src, err := s.source(ctx, ws, id)
		if err != nil {
			return CouponPool{}, err
		}
		if src.Platform != d.source {
			return CouponPool{}, invalid(ctx, "body.storeId", mPoolStorePlatform, i18n.Args{"store": src.Name, "platform": platformName(d.source)})
		}
		if !src.Can {
			return CouponPool{}, api.Err(http.StatusConflict, "store_cannot_create", src.Detail).WithMeta("reason", src.Reason)
		}
		d.storeID = &id
	}
	set, err := s.checkSettings(ctx, d.source, in.Settings)
	if err != nil {
		return CouponPool{}, err
	}
	d.settings = set
	raw, _ := json.Marshal(set)
	id := newID()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.CouponPoolInsert(ctx, store.CouponPoolInsertParams{ID: id, WorkspaceID: ws.ID, Name: d.name, Slug: d.slug, Source: d.source, StoreID: d.storeID,
			Settings: raw, Prefix: d.prefix, OnePerContact: d.onePerContact, LowAt: int32(d.lowAt), CreatedBy: actorUser(api.PrincipalFrom(ctx)), Now: s.now()}); err != nil {
			if isUnique(err) {
				return invalid(ctx, "body.slug", mPoolSlugTaken, i18n.Args{"slug": d.slug})
			}
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.created", TargetType: "coupon_pool", TargetID: id.String(),
			TargetName: d.name, After: snapshot(d)})
	})
	if err != nil {
		return CouponPool{}, err
	}
	return s.Get(ctx, ws, id)
}

// source is one store of the workspace as a source of codes.
func (s *Service) source(ctx context.Context, ws *store.Workspace, id uuid.UUID) (StoreSource, error) {
	if s.Sources != nil {
		src, err := s.Sources(ctx, ws)
		if err != nil {
			return StoreSource{}, err
		}
		for _, x := range src {
			if x.StoreID == id {
				return x, nil
			}
		}
	}
	return StoreSource{}, invalid(ctx, "body.storeId", mPoolStoreUnknown, nil)
}

func platformName(p string) string {
	if p == SourceWoo {
		return "WooCommerce"
	}
	return "Shopify"
}

func isUnique(err error) bool { return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505") }

// Update changes a pool. A store pool's discount can't change once it was
// created in the store.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, id uuid.UUID, in CouponPoolChange) (CouponPool, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponPool{}, err
	}
	before := draftOf(p)
	d := before
	if in.Name != nil {
		d.name = *in.Name
	}
	if in.Prefix != nil {
		d.prefix = *in.Prefix
	}
	if in.OnePerContact != nil {
		d.onePerContact = *in.OnePerContact
	}
	if in.LowAt != nil {
		d.lowAt = *in.LowAt
	}
	if err := s.checkCommon(ctx, &d); err != nil {
		return CouponPool{}, err
	}
	if in.Settings != nil {
		set, err := s.checkSettings(ctx, d.source, *in.Settings)
		if err != nil {
			return CouponPool{}, err
		}
		if d.source != SourceUpload && p.ExternalDiscountID != "" && !sameSettings(set, before.settings) {
			return CouponPool{}, conflict(ctx, "discount_created", mPoolLocked, i18n.Args{"store": p.StoreName})
		}
		d.settings = set
	}
	raw, _ := json.Marshal(d.settings)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CouponPoolUpdate(ctx, store.CouponPoolUpdateParams{ID: id, WorkspaceID: ws.ID, Name: d.name, Settings: raw, Prefix: d.prefix,
			OnePerContact: d.onePerContact, LowAt: int32(d.lowAt), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.updated", TargetType: "coupon_pool", TargetID: id.String(),
			TargetName: d.name, Before: snapshot(before), After: snapshot(d)})
	})
	if err != nil {
		return CouponPool{}, err
	}
	return s.Get(ctx, ws, id)
}

func sameSettings(a, b CouponPoolSettings) bool {
	return a.Kind == b.Kind && a.Value == b.Value && a.MinimumCents == b.MinimumCents && a.EndsDays == b.EndsDays && slices.Equal(a.CollectionIDs, b.CollectionIDs)
}

// Archive stops a pool from giving codes. When live automations send its
// codes it answers 409 unless confirm.
func (s *Service) Archive(ctx context.Context, ws *store.Workspace, id uuid.UUID, confirm bool) (CouponPool, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponPool{}, err
	}
	if p.Status == StatusArchived {
		return s.Get(ctx, ws, id)
	}
	live, err := s.liveAutomations(ctx, ws.ID, p.Slug)
	if err != nil {
		return CouponPool{}, err
	}
	if len(live) > 0 && !confirm {
		names := make([]string, 0, len(live))
		for _, n := range live {
			names = append(names, n)
		}
		slices.Sort(names)
		l := i18n.LangFrom(ctx)
		return CouponPool{}, api.Err(http.StatusConflict, "pool_in_use", mPoolArchiveLive.Text(l, i18n.Args{"flows": i18n.AndList(l.Locale, names),
			"count": len(names), "name": p.Name})).WithMeta("automations", names)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.CouponPoolArchive(ctx, store.CouponPoolArchiveParams{ID: id, WorkspaceID: ws.ID, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.archived", TargetType: "coupon_pool", TargetID: id.String(),
			TargetName: p.Name, Before: map[string]any{"status": p.Status}, After: map[string]any{"status": StatusArchived}, Metadata: map[string]any{"liveAutomations": len(live)}})
	})
	if err != nil {
		return CouponPool{}, err
	}
	return s.Get(ctx, ws, id)
}

// helpers are up to three members who can manage discount codes.
func (s *Service) helpers(ctx context.Context, ws uuid.UUID) []string {
	out := []string{}
	rows, err := s.Q.CommerceMembers(ctx, ws)
	if err != nil {
		return out
	}
	for _, r := range rows {
		if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.CouponsManage) {
			continue
		}
		if name := strings.TrimSpace(r.Name); name != "" {
			out = append(out, name)
		}
		if len(out) == 3 {
			break
		}
	}
	return out
}
