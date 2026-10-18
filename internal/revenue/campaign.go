package revenue

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// CampaignRevenue is a campaign report's Revenue section: the upgrades and
// conversions credited to it.
type CampaignRevenue struct {
	// Locked: the person can't see revenue (earn:view); Helpers can.
	Locked   bool       `json:"locked"`
	Helpers  []string   `json:"helpers"`
	Name     string     `json:"name"`
	SentAt   *time.Time `json:"sentAt,omitempty"`
	Currency string     `json:"currency"`
	// Pending: trials are still running, so new MRR counts only those who
	// paid at once; TrialsEnd is when the last of them ends.
	Pending    bool       `json:"pending"`
	TrialsEnd  *time.Time `json:"trialsEnd,omitempty"`
	Upgrades   int64      `json:"upgrades"`
	PaidAtOnce int64      `json:"paidAtOnce"`
	Trials     int64      `json:"trials"`
	// Collected is what the first payments came to, by amount ("6 ×
	// $64.00 and 3 × $200.00"), before tax.
	CollectedCents int64             `json:"collectedCents"`
	Collected      []CampaignPayment `json:"collected"`
	NewMRRCents    int64             `json:"newMrrCents"`
	// PotentialMRRCents is new MRR if every trial becomes paid.
	PotentialMRRCents int64 `json:"potentialMrrCents"`
	UpgradeClicks     int64 `json:"upgradeClicks"`
	Checkouts         int64 `json:"checkouts"`
	Delivered         int64 `json:"delivered"`
	// UpgradedRate is upgrades ÷ delivered.
	UpgradedRate *float64 `json:"upgradedRate,omitempty"`
	// What the upgrades are: paid at once and trials, by tier and period.
	PaidLines  []CampaignUpgradeLine `json:"paidLines"`
	TrialLines []CampaignUpgradeLine `json:"trialLines"`
	// HasUpgradeLinks: the email links to the upgrade page.
	HasUpgradeLinks bool                `json:"hasUpgradeLinks"`
	Conversions     CampaignConversions `json:"conversions"`
	// Window is the attribution window in days.
	Window int `json:"window"`
}

// CampaignPayment is first payments of one amount.
type CampaignPayment struct {
	Count       int64 `json:"count"`
	AmountCents int64 `json:"amountCents"`
}

// CampaignUpgradeLine is upgrades of one tier and period (and offer).
type CampaignUpgradeLine struct {
	Tier   string `json:"tier"`
	Period string `json:"period"`
	Offer  string `json:"offer,omitempty"`
	Count  int64  `json:"count"`
}

// CampaignConversions is the product conversions credited to the email.
type CampaignConversions struct {
	Count      int64  `json:"count"`
	ValueCents int64  `json:"valueCents"`
	Currency   string `json:"currency"`
	ByToken    int64  `json:"byToken"`
	ByEmail    int64  `json:"byEmail"`
	// Name, TypicalCents and Sender describe a single kind of conversion.
	Name         string `json:"name,omitempty"`
	TypicalCents int64  `json:"typicalCents,omitempty"`
	Sender       string `json:"sender,omitempty"`
	// Domains are the click token's domains.
	Domains []string `json:"domains"`
}

// Campaign is the Revenue section of a campaign's report. Without
// earn:view it says who can show it.
func (s *Service) Campaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CampaignRevenue, error) {
	out := CampaignRevenue{Helpers: []string{}, Collected: []CampaignPayment{}, PaidLines: []CampaignUpgradeLine{}, TrialLines: []CampaignUpgradeLine{},
		Currency: "usd", Window: 7, Conversions: CampaignConversions{Domains: []string{}, Currency: "usd"}}
	c, err := s.Q.RevenueCampaignInfo(ctx, store.RevenueCampaignInfoParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return out, api.MapError(err, "Campaign")
	}
	out.Name = c.Name
	out.SentAt = c.StartedAt
	if !p.Can(authz.EarnView) {
		out.Locked = true
		out.Helpers = s.helpers(ctx, ws.ID, p.UserID, 3, authz.EarnView)
		return out, nil
	}
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil {
		out.Window = int(st.AttributionDays)
		if st.Currency != "" {
			out.Currency, out.Conversions.Currency = st.Currency, st.Currency
		}
		if st.ClickTokenOn && st.ClickTokenDomains != nil {
			out.Conversions.Domains = st.ClickTokenDomains
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.Delivered = c.Delivered
	rows, err := s.Q.RevenueCampaignUpgrades(ctx, store.RevenueCampaignUpgradesParams{WorkspaceID: ws.ID, CampaignID: &id})
	if err != nil {
		return out, err
	}
	paid := map[int64]int64{}
	paidLines := map[string]*CampaignUpgradeLine{}
	trialLines := map[string]*CampaignUpgradeLine{}
	now := s.now()
	for _, r := range rows {
		out.Upgrades++
		trialing := r.Status == "trialing" || (r.TrialEnd != nil && r.TrialEnd.After(now))
		key := r.Tier + "|" + r.Period + "|" + r.Offer
		if trialing {
			out.Trials++
			out.PotentialMRRCents += r.ArrIfPaid
			if out.TrialsEnd == nil || (r.TrialEnd != nil && r.TrialEnd.After(*out.TrialsEnd)) {
				out.TrialsEnd = r.TrialEnd
			}
			key = r.Tier + "|" + r.Period
			if l, ok := trialLines[key]; ok {
				l.Count++
			} else {
				trialLines[key] = &CampaignUpgradeLine{Tier: r.Tier, Period: r.Period, Count: 1}
			}
			continue
		}
		out.PaidAtOnce++
		out.NewMRRCents += r.Arr
		out.PotentialMRRCents += r.Arr
		paid[r.FirstCents]++
		out.CollectedCents += r.FirstCents
		if l, ok := paidLines[key]; ok {
			l.Count++
		} else {
			paidLines[key] = &CampaignUpgradeLine{Tier: r.Tier, Period: r.Period, Offer: r.Offer, Count: 1}
		}
	}
	out.Pending = out.Trials > 0
	out.NewMRRCents, out.PotentialMRRCents = mrr(out.NewMRRCents), mrr(out.PotentialMRRCents)
	for amount, n := range paid {
		out.Collected = append(out.Collected, CampaignPayment{Count: n, AmountCents: amount})
	}
	sort.Slice(out.Collected, func(i, j int) bool { return out.Collected[i].Count > out.Collected[j].Count })
	for _, l := range paidLines {
		out.PaidLines = append(out.PaidLines, *l)
	}
	for _, l := range trialLines {
		out.TrialLines = append(out.TrialLines, *l)
	}
	byCount := func(ls []CampaignUpgradeLine) func(i, j int) bool {
		return func(i, j int) bool {
			if ls[i].Count != ls[j].Count {
				return ls[i].Count > ls[j].Count
			}
			return ls[i].Tier+ls[i].Period < ls[j].Tier+ls[j].Period
		}
	}
	sort.Slice(out.PaidLines, byCount(out.PaidLines))
	sort.Slice(out.TrialLines, byCount(out.TrialLines))
	out.UpgradedRate = pct(out.Upgrades, out.Delivered)
	if out.UpgradeClicks, err = s.Q.RevenueCampaignClicks(ctx, id); err != nil {
		return out, err
	}
	if out.Checkouts, err = s.Q.RevenueCampaignCheckouts(ctx, store.RevenueCampaignCheckoutsParams{WorkspaceID: ws.ID, CampaignID: id.String()}); err != nil {
		return out, err
	}
	links, err := s.Q.RevenueCampaignLinkUrls(ctx, id)
	if err != nil {
		return out, err
	}
	for _, l := range links {
		if IsUpgradeLink(l) {
			out.HasUpgradeLinks = true
		}
	}
	// Clicks on upgrade links mean the email had them, whatever its links
	// table says (an email sent before link tracking, or edited since).
	out.HasUpgradeLinks = out.HasUpgradeLinks || out.UpgradeClicks > 0 || out.Checkouts > 0
	if !out.HasUpgradeLinks && strings.Contains(c.Mjml, "upgrade") {
		out.HasUpgradeLinks = strings.Contains(c.Mjml, "/upgrade") || strings.Contains(c.Mjml, "upgrade_url")
	}
	conv, err := s.Q.RevenueCampaignConversions(ctx, store.RevenueCampaignConversionsParams{WorkspaceID: ws.ID, CampaignID: &id})
	if err != nil {
		return out, err
	}
	kinds := map[string]bool{}
	for _, r := range conv {
		if !strings.EqualFold(r.Currency, out.Currency) {
			continue
		}
		cc := &out.Conversions
		cc.Count += r.Conversions
		cc.ValueCents += r.ValueCents
		switch r.Matched {
		case "click_token":
			cc.ByToken += r.Conversions
		case "email":
			cc.ByEmail += r.Conversions
		}
		kinds[r.Name+"|"+r.KeyName+"|"+strconv.FormatInt(r.ValueCents, 10)] = true
		cc.Name, cc.TypicalCents, cc.Sender = r.Name, r.ValueCents, conversionSender(r.KeyName)
	}
	if len(kinds) > 1 {
		out.Conversions.Name, out.Conversions.TypicalCents, out.Conversions.Sender = "", 0, ""
	}
	return out, nil
}
