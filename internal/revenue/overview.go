package revenue

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/store"
)

// OverviewQuery is the overview's period.
type OverviewQuery struct {
	Period string `query:"period" enum:"30d,90d,12m" default:"30d" doc:"The last 30 days, 90 days or 12 months"`
}

// day is 24 hours.
const day = 24 * time.Hour

// tooFew is the fewest free subscribers a source needs for its rate to
// mean something.
const tooFew = 500

// window is the period's start (the start of its first day in the
// workspace's time zone) and end (now).
func (s *Service) window(ws *store.Workspace, period string) (time.Time, time.Time) {
	now := s.now()
	today := startOfDay(now, location(ws))
	switch period {
	case "90d":
		return today.AddDate(0, 0, -90), now
	case "12m":
		return today.AddDate(-1, 0, 0), now
	}
	return today.AddDate(0, 0, -30), now
}

// Overview is Earn › Overview: MRR and its movement, paid members by tier,
// trials, conversions, cancellations, free-to-paid by source, upgrades by
// email, what needs a person and the latest events. A card that fails is
// named in Failed and the rest still answers.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal, in OverviewQuery) (RevenueOverview, error) {
	period := in.Period
	if period == "" {
		period = "30d"
	}
	from, to := s.window(ws, period)
	out := RevenueOverview{Period: period, From: from, To: to, Currency: "usd", Tiers: []RevenueTierRow{}, Needs: []RevenueNeed{}, Activity: []RevenueActivity{},
		Failed: []string{}, Helpers: []string{}, Chart: RevenueChart{Points: []RevenuePoint{}, Marks: []RevenueMark{}},
		Conversions: RevenueConversions{ByEmail: []RevenueConversionRow{}, Latest: []RevenueConversionLine{}}, Cancellations: RevenueCancellations{Reasons: []RevenueReason{}},
		FreeToPaid: RevenueFreeToPaid{Rows: []RevenueSourceRow{}}, Upgrades: RevenueUpgrades{Rows: []RevenueUpgradeRow{}, Window: 7}}
	out.CanEdit, out.CanManage = p.Can(authz.EarnEdit), p.Can(authz.EarnManage)
	out.CanPayments, out.CanSponsors = p.Can(authz.PaymentsManage), p.Can(authz.SponsorsManage)
	if !out.CanEdit && !out.CanManage && !out.CanPayments && !out.CanSponsors {
		out.ReadOnly = true
		out.Helpers = s.helpers(ctx, ws.ID, p.UserID, 3, authz.EarnEdit, authz.EarnManage, authz.PaymentsManage, authz.SponsorsManage)
	}
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil {
		if st.Currency != "" {
			out.Currency = st.Currency
		}
		out.Upgrades.Window = int(st.AttributionDays)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	// Data begins with the live connection, or with older subscriptions
	// found in the provider's history (an import keeps their start).
	conn, err := s.Q.RevenueLiveConnection(ctx, ws.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	since := s.now()
	if err == nil {
		out.Account = conn.AccountName
		c := conn.CreatedAt
		if conn.ConnectedAt != nil {
			c = *conn.ConnectedAt
		}
		out.ConnectedAt, since = &c, c
	}
	if l, err := s.Q.RevenueLedgerSince(ctx, ws.ID); err == nil && l.Changes > 0 && l.FirstAt.Before(since) {
		out.HadHistory = out.ConnectedAt != nil && l.FirstAt.Before(out.ConnectedAt.Add(-day))
		since = l.FirstAt
	}
	out.DataSince = &since
	if to.Sub(since) < 30*day {
		out.TooEarly = true
		shown := startOfDay(since, location(ws)).AddDate(0, 0, 30)
		out.ShownFrom = &shown
		if since.After(from) {
			from = since
			out.From = since
		}
	}
	fail := func(card string, err error) {
		s.log().ErrorContext(ctx, "revenue overview", "card", card, "workspace", ws.ID, "err", err)
		out.Failed = append(out.Failed, card)
	}
	if err := s.movement(ctx, ws, from, to, &out); err != nil {
		fail("movement", err)
	}
	if !out.TooEarly {
		if err := s.chart(ctx, ws, &out); err != nil {
			fail("chart", err)
		}
	}
	if err := s.tiers(ctx, ws, &out); err != nil {
		fail("tiers", err)
	}
	if err := s.trials(ctx, ws, from, to, &out); err != nil {
		fail("trials", err)
	}
	if err := s.conversions(ctx, ws, from, to, &out.Conversions, out.Currency); err != nil {
		fail("conversions", err)
	}
	if err := s.cancellations(ctx, ws, from, to, &out); err != nil {
		fail("cancellations", err)
	}
	if err := s.freeToPaid(ctx, ws, from, to, &out); err != nil {
		fail("freeToPaid", err)
	}
	if err := s.upgrades(ctx, ws, from, to, &out); err != nil {
		fail("upgrades", err)
	}
	if err := s.otherIncome(ctx, ws, from, to, &out); err != nil {
		fail("income", err)
	}
	if err := s.needs(ctx, ws, &out); err != nil {
		fail("needs", err)
	}
	if err := s.activity(ctx, ws, &out); err != nil {
		fail("activity", err)
	}
	return out, nil
}

// movement is MRR at the start and end of a window and the changes between.
func (s *Service) movement(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	now, err := s.Q.RevenueMRRNow(ctx, ws.ID)
	if err != nil {
		return err
	}
	deltas, err := s.Q.RevenueDeltaSince(ctx, store.RevenueDeltaSinceParams{WorkspaceID: ws.ID, Points: []time.Time{from}})
	if err != nil {
		return err
	}
	startArr, startMembers := now.Arr, now.Members
	if len(deltas) == 1 {
		startArr -= deltas[0].Arr
		startMembers -= deltas[0].Members
	}
	moves, err := s.Q.RevenueMovement(ctx, store.RevenueMovementParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	m := RevenueMovement{FromCents: mrr(startArr), FromMembers: startMembers, ToCents: mrr(now.Arr), ToMembers: now.Members}
	for _, r := range moves {
		mv := RevenueMove{Cents: mrr(r.Arr), Count: r.Subscriptions}
		switch r.Kind {
		case "new":
			m.New = mv
		case "reactivation":
			m.Reactivated = mv
		case "expansion":
			m.Expansion = mv
		case "contraction":
			m.Contraction = mv
		case "churn":
			m.Churned = mv
		}
	}
	out.Movement = m
	t := &out.Tiles
	t.MRRCents, t.ARRCents = mrr(now.Arr), now.Arr
	t.MRRDeltaCents = mrr(now.Arr) - mrr(startArr)
	t.MRRDeltaPct = pct(t.MRRDeltaCents, mrr(startArr))
	t.Members, t.MembersDelta = now.Members, now.Members-startMembers
	t.New, t.Churned = m.New.Count+m.Reactivated.Count, m.Churned.Count
	t.ChurnBase = startMembers
	if !out.TooEarly {
		t.ChurnRate = pct(m.Churned.Count, startMembers)
	}
	return nil
}

// chart is MRR over the last 12 months on the same day each month, with
// the move and the signup offers that started marked.
func (s *Service) chart(ctx context.Context, ws *store.Workspace, out *RevenueOverview) error {
	loc := location(ws)
	now := s.now()
	today := startOfDay(now, loc)
	points := make([]time.Time, 0, 15)
	for i := 12; i >= 1; i-- {
		points = append(points, today.AddDate(0, -i, 0))
	}
	var moved *store.RevenueMovedRow
	if m, err := s.Q.RevenueMoved(ctx, ws.ID); err == nil {
		if m.At.After(points[0]) && m.At.Before(now) {
			moved = &m
			points = append(points, m.At)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	points = append(points, now)
	sort.Slice(points, func(i, j int) bool { return points[i].Before(points[j]) })
	cur, err := s.Q.RevenueMRRNow(ctx, ws.ID)
	if err != nil {
		return err
	}
	deltas, err := s.Q.RevenueDeltaSince(ctx, store.RevenueDeltaSinceParams{WorkspaceID: ws.ID, Points: points})
	if err != nil {
		return err
	}
	c := RevenueChart{Points: []RevenuePoint{}, Marks: []RevenueMark{}, Since: points[0]}
	for _, d := range deltas {
		pt := RevenuePoint{At: d.At, MRRCents: mrr(cur.Arr - d.Arr)}
		if moved != nil && d.At.Before(moved.At) {
			pt.History = true
		}
		c.Points = append(c.Points, pt)
	}
	if moved != nil {
		at := moved.At
		c.HistoryUntil = &at
		c.Marks = append(c.Marks, RevenueMark{N: 1, Kind: "moved", At: at, Name: moved.Source})
	}
	offers, err := s.Q.RevenueOfferStarts(ctx, store.RevenueOfferStartsParams{WorkspaceID: ws.ID, FromAt: points[0], ToAt: now})
	if err != nil {
		return err
	}
	for _, o := range offers {
		c.Marks = append(c.Marks, RevenueMark{N: len(c.Marks) + 1, Kind: "offer", At: o.StartedAt, Name: o.Name})
	}
	if n := len(c.Points); n > 0 {
		c.StartCents, c.EndCents = c.Points[0].MRRCents, c.Points[n-1].MRRCents
		c.DeltaCents = c.EndCents - c.StartCents
		c.DeltaPct = pct(c.DeltaCents, c.StartCents)
	}
	out.Chart = c
	return nil
}

// tiers is paying members and MRR by tier and period today, with the
// offers some of them are on.
func (s *Service) tiers(ctx context.Context, ws *store.Workspace, out *RevenueOverview) error {
	rows, err := s.Q.RevenueByTier(ctx, ws.ID)
	if err != nil {
		return err
	}
	all, err := s.Q.RevenueTiers(ctx, ws.ID)
	if err != nil {
		return err
	}
	prices, err := s.Q.RevenueTierPrices(ctx, ws.ID)
	if err != nil {
		return err
	}
	priceOf := map[string]int64{}
	for _, p := range prices {
		priceOf[p.TierID.String()+":"+p.Period] = p.AmountCents
	}
	index := map[string]int{}
	for _, r := range rows {
		id := ""
		if r.TierID != nil {
			id = r.TierID.String()
		}
		key := id + ":" + r.Period
		i, ok := index[key]
		if !ok {
			row := RevenueTierRow{TierID: id, Tier: r.TierName, Period: r.Period, Offers: []RevenueTierOffer{}, Includes: []string{}, PriceCents: priceOf[key]}
			if row.PriceCents == 0 {
				row.PriceCents = r.AmountCents
			}
			for _, t := range all {
				if t.AccessRank < r.AccessRank && t.Visibility != "archived" {
					row.Includes = append(row.Includes, t.Name)
				}
			}
			out.Tiers = append(out.Tiers, row)
			i = len(out.Tiers) - 1
			index[key] = i
		}
		row := &out.Tiers[i]
		row.Members += r.Members
		row.MRRCents += r.Arr
		if r.Offer != "" {
			row.Offers = append(row.Offers, RevenueTierOffer{Name: r.Offer, Members: r.Members, PaidCents: r.PaidCents, Duration: offerDuration(r.Duration), Months: int(r.Months)})
		}
	}
	for i := range out.Tiers {
		out.Tiers[i].MRRCents = mrr(out.Tiers[i].MRRCents)
	}
	return nil
}

// offerDuration words a provider's discount duration as an offer's.
func offerDuration(d string) string {
	switch d {
	case "repeating", "months":
		return "months"
	case "forever":
		return "forever"
	}
	return "once"
}

// trials is the window's trials and what became of them.
func (s *Service) trials(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	soon := s.now().Add(7 * day)
	r, err := s.Q.RevenueTrials(ctx, store.RevenueTrialsParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to, Soon: soon})
	if err != nil {
		return err
	}
	t := RevenueTrials{Started: r.Started, Trialing: r.Trialing, BecamePaid: r.BecamePaid, Canceled: r.Canceled, Ended: r.BecamePaid + r.Canceled, EndingSoon: r.EndingSoon}
	if r.EndingSoon > 0 {
		top, err := s.Q.RevenueTrialsEndingTop(ctx, store.RevenueTrialsEndingTopParams{WorkspaceID: ws.ID, Soon: soon})
		if err != nil {
			return err
		}
		if len(top) == 1 && top[0].SourceCampaignID != nil {
			t.EndingFrom = &RevenueCredit{Kind: "campaign", ID: top[0].SourceCampaignID.String(), Name: top[0].Name, Count: top[0].Trials}
		}
	}
	out.Trials = t
	out.Tiles.Trialing = r.Trialing
	return nil
}

// conversionSender is who sends conversions, from the key's name: "Acme
// Cloud · conversions" is Acme Cloud.
func conversionSender(key string) string {
	k := strings.TrimSpace(key)
	for _, suffix := range []string{" · conversions", " - conversions", " conversions"} {
		if len(k) > len(suffix) && strings.EqualFold(k[len(k)-len(suffix):], suffix) {
			return strings.TrimSpace(k[:len(k)-len(suffix)])
		}
	}
	return k
}

// conversions is the window's product conversions, by the email credited.
func (s *Service) conversions(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueConversions, currency string) error {
	out.Currency = currency
	rows, err := s.Q.RevenueConversionsSummary(ctx, store.RevenueConversionsSummaryParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	index := map[string]int{}
	for _, r := range rows {
		out.Total += r.Conversions
		if !strings.EqualFold(r.Currency, currency) {
			out.OtherCurrency += r.Conversions
			continue
		}
		out.Count += r.Conversions
		out.ValueCents += r.ValueCents
		var kind, id string
		switch {
		case r.CampaignID != nil:
			kind, id = "campaign", r.CampaignID.String()
		case r.AutomationID != nil:
			kind, id = "automation", r.AutomationID.String()
		default:
			out.Uncredited.Count += r.Conversions
			out.Uncredited.Cents += r.ValueCents
			continue
		}
		i, ok := index[id]
		if !ok {
			out.ByEmail = append(out.ByEmail, RevenueConversionRow{Kind: kind, ID: id, Name: r.CreditedName})
			i = len(out.ByEmail) - 1
			index[id] = i
		}
		row := &out.ByEmail[i]
		row.Count += r.Conversions
		row.ValueCents += r.ValueCents
		switch r.Matched {
		case "click_token":
			row.ByToken += r.Conversions
		case "email":
			row.ByEmail += r.Conversions
		}
	}
	sort.Slice(out.ByEmail, func(i, j int) bool { return out.ByEmail[i].Count > out.ByEmail[j].Count })
	kinds, err := s.Q.RevenueConversionKinds(ctx, store.RevenueConversionKindsParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	if len(kinds) == 1 {
		out.Name, out.TypicalCents, out.Sender = kinds[0].Name, kinds[0].ValueCents, conversionSender(kinds[0].KeyName)
	} else if len(kinds) > 1 {
		names := map[string]bool{}
		for _, k := range kinds {
			names[k.Name] = true
		}
		if len(names) == 1 {
			out.Name = kinds[0].Name
		}
	}
	first, err := s.Q.RevenueConversionsFirst(ctx, ws.ID)
	if err != nil {
		return err
	}
	if first.Conversions > 0 && first.FirstAt.After(from) {
		f := first.FirstAt
		out.Since = &f
	}
	latest, err := s.Q.RevenueConversions(ctx, store.RevenueConversionsParams{WorkspaceID: ws.ID, Lim: 3})
	if err != nil {
		return err
	}
	for _, c := range latest {
		if c.Test {
			continue
		}
		l := RevenueConversionLine{At: c.ReceivedAt, ValueCents: c.ValueCents, Currency: c.Currency, Matched: c.Matched, Credited: c.CreditedName}
		if c.CampaignID != nil {
			l.CampaignID = c.CampaignID.String()
		}
		out.Latest = append(out.Latest, l)
	}
	return nil
}

// cancellations is the window's visits to the cancel page.
func (s *Service) cancellations(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	sum, err := s.Q.RevenueCancelSummary(ctx, store.RevenueCancelSummaryParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	c := RevenueCancellations{Visits: sum.Visits, Kept: sum.Kept, Continued: sum.Continued, Undecided: sum.Undecided, Reasons: []RevenueReason{}}
	reasons, err := s.Q.RevenueCancelReasons(ctx, store.RevenueCancelReasonsParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	counts := map[string]int64{}
	for _, r := range reasons {
		counts[r.Reason] = r.Visits
	}
	for _, k := range []string{"not_reading", "too_expensive", "not_for_me", "switching", "other", ""} {
		if counts[k] > 0 {
			key := k
			if key == "" {
				key = "none"
			}
			c.Reasons = append(c.Reasons, RevenueReason{Reason: key, Count: counts[k]})
		}
	}
	if kept, err := s.Q.RevenueKeptWith(ctx, store.RevenueKeptWithParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to}); err != nil {
		return err
	} else if len(kept) == 1 {
		c.KeptOffer, c.KeptTier = kept[0].Offer, kept[0].Tier
	}
	if o, err := s.Q.RevenueRetentionOffer(ctx, ws.ID); err != nil {
		return err
	} else if len(o) == 1 {
		c.OfferID = o[0].ID.String()
		if c.KeptOffer == "" {
			c.KeptOffer = o[0].Name
		}
	}
	out.Cancellations = c
	return nil
}

// standardKinds are how readers usually join; other kinds (referrals,
// recommendations) keep their own row however small.
var standardKinds = map[string]bool{growth.KindForm: true, growth.KindSubscribePage: true, growth.KindImport: true, growth.KindCarried: true,
	growth.KindAPIKey: true, growth.KindIntegration: true, growth.KindPerson: true, growth.KindPreferenceCenter: true, growth.KindUnknown: true}

// freeToPaid is the window's free-to-paid conversion by how readers first
// joined: the four biggest usual sources, every other kind, and the rest
// summed up.
func (s *Service) freeToPaid(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	rows, err := s.Q.RevenueFreeToPaid(ctx, store.RevenueFreeToPaidParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	names := map[string]growth.GrowthSource{}
	if s.Growth != nil {
		if list, err := s.Growth.Sources(ctx, ws); err == nil {
			for _, it := range list.Items {
				names[it.Source.Key] = it.Source
			}
		}
	}
	var std, own []RevenueSourceRow
	for _, r := range rows {
		key := growth.SourceKey(r.SourceKind, r.SourceID)
		row := RevenueSourceRow{Key: key, Kind: r.SourceKind, AtStart: r.AtStart, Joined: r.Joined, Base: r.AtStart + r.Joined, Paid: r.Paid}
		if src, ok := names[key]; ok {
			row.Name, row.FormKind = src.Name, src.FormKind
		}
		if row.Base == 0 && row.Paid == 0 {
			continue
		}
		if standardKinds[r.SourceKind] {
			std = append(std, row)
		} else {
			own = append(own, row)
		}
	}
	sort.SliceStable(std, func(i, j int) bool { return std[i].Base > std[j].Base })
	ftp := RevenueFreeToPaid{Rows: []RevenueSourceRow{}, Total: RevenueSourceRow{Key: "total", Kind: "total"}}
	other := RevenueSourceRow{Key: "other", Kind: "other"}
	for i, r := range std {
		if i < 4 {
			ftp.Rows = append(ftp.Rows, r)
			continue
		}
		other.AtStart += r.AtStart
		other.Joined += r.Joined
		other.Base += r.Base
		other.Paid += r.Paid
		other.Others++
	}
	if other.Others > 0 {
		ftp.Rows = append(ftp.Rows, other)
	}
	sort.SliceStable(own, func(i, j int) bool { return own[i].Base > own[j].Base })
	ftp.Rows = append(ftp.Rows, own...)
	for i := range ftp.Rows {
		r := &ftp.Rows[i]
		r.TooFew = r.Base < tooFew
		if !r.TooFew {
			r.Rate = pct(r.Paid, r.Base)
		}
		ftp.Total.AtStart += r.AtStart
		ftp.Total.Joined += r.Joined
		ftp.Total.Base += r.Base
		ftp.Total.Paid += r.Paid
	}
	ftp.Total.Rate = pct(ftp.Total.Paid, ftp.Total.Base)
	all, err := s.Q.RevenueNewContacts(ctx, store.RevenueNewContactsParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	ftp.All = all
	ftp.Direct = max(all-ftp.Total.Paid, 0)
	out.FreeToPaid = ftp
	out.Tiles.FreeToPaid, out.Tiles.FreeBase = ftp.Total.Paid, ftp.Total.Base
	if !out.TooEarly {
		out.Tiles.FreeToPaidRate = ftp.Total.Rate
	}
	return nil
}

// upgrades is the window's new paid members by the email they're credited
// to: campaigns by date, then automations, then those with no email click.
func (s *Service) upgrades(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	rows, err := s.Q.RevenueUpgradesByEmail(ctx, store.RevenueUpgradesByEmailParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	var campaigns, autos []RevenueUpgradeRow
	none := RevenueUpgradeRow{Kind: "none"}
	autoIndex := map[string]int{}
	for _, r := range rows {
		switch {
		case r.SourceKind == "campaign" && r.SourceCampaignID != nil:
			campaigns = append(campaigns, RevenueUpgradeRow{Kind: "campaign", ID: r.SourceCampaignID.String(), Name: r.Name, SentAt: r.SentAt, Members: r.Members, MRRCents: r.Arr})
		case r.SourceKind == "automation":
			id := ""
			if r.AutomationID != nil {
				id = r.AutomationID.String()
			}
			key := id + "|" + r.Name
			if i, ok := autoIndex[key]; ok {
				autos[i].Members += r.Members
				autos[i].MRRCents += r.Arr
				continue
			}
			autos = append(autos, RevenueUpgradeRow{Kind: "automation", ID: id, Name: r.Name, Members: r.Members, MRRCents: r.Arr})
			autoIndex[key] = len(autos) - 1
		default:
			none.Members += r.Members
			none.MRRCents += r.Arr
		}
	}
	sort.SliceStable(campaigns, func(i, j int) bool {
		a, b := campaigns[i].SentAt, campaigns[j].SentAt
		if a == nil || b == nil {
			return b == nil && a != nil
		}
		return a.Before(*b)
	})
	sort.SliceStable(autos, func(i, j int) bool { return autos[i].Members > autos[j].Members })
	all := append(append(campaigns, autos...), none)
	u := RevenueUpgrades{Rows: []RevenueUpgradeRow{}, Window: out.Upgrades.Window}
	for _, r := range all {
		if r.Members == 0 && r.Kind != "none" {
			continue
		}
		u.Total.Count += r.Members
		u.Total.Cents += r.MRRCents
		r.MRRCents = mrr(r.MRRCents)
		u.Rows = append(u.Rows, r)
	}
	if u.Total.Count == 0 {
		u.Rows = []RevenueUpgradeRow{}
	}
	u.Total.Cents = mrr(u.Total.Cents)
	out.Upgrades = u
	return nil
}

// otherIncome is gift sales and sponsored issues of the window (not MRR).
func (s *Service) otherIncome(ctx context.Context, ws *store.Workspace, from, to time.Time, out *RevenueOverview) error {
	g, err := s.Q.RevenueGiftIncome(ctx, store.RevenueGiftIncomeParams{WorkspaceID: ws.ID, FromAt: from, ToAt: to})
	if err != nil {
		return err
	}
	loc := location(ws)
	civil := func(t time.Time) time.Time {
		l := t.In(loc)
		return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	}
	sp, err := s.Q.RevenueSponsorIncome(ctx, store.RevenueSponsorIncomeParams{WorkspaceID: ws.ID, FromDay: civil(from), ToDay: civil(to)})
	if err != nil {
		return err
	}
	out.Tiles.GiftsCents, out.Tiles.SponsorsCents = g.Cents, sp.Cents
	return nil
}

// needs is what needs a person: failed payments, offers near their limit
// or end, sponsor creatives and payments, and member emails not published.
func (s *Service) needs(ctx context.Context, ws *store.Workspace, out *RevenueOverview) error {
	now := s.now()
	loc := location(ws)
	pastDue, err := s.Q.RevenuePastDue(ctx, ws.ID)
	if err != nil {
		return err
	}
	if pastDue > 0 {
		out.Needs = append(out.Needs, RevenueNeed{Kind: "past_due", Count: pastDue})
	}
	offers, err := s.Q.RevenueOffersToWatch(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, o := range offers {
		ending := o.EndsAt != nil && o.EndsAt.Before(now.Add(30*day))
		filling := o.RedeemLimit != nil && *o.RedeemLimit > 0 && o.Used*4 >= int64(*o.RedeemLimit)*3
		if !ending && !filling {
			continue
		}
		n := RevenueNeed{Kind: "offer", Name: o.Name, Used: o.Used, EndsAt: o.EndsAt, TargetID: o.ID.String()}
		if o.RedeemLimit != nil {
			n.Limit = int64(*o.RedeemLimit)
		}
		out.Needs = append(out.Needs, n)
	}
	today := startOfDay(now, loc)
	civilToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	books, err := s.Q.RevenueSponsorNeeds(ctx, store.RevenueSponsorNeedsParams{WorkspaceID: ws.ID, CreativeUntil: civilToday.AddDate(0, 0, 7), Today: civilToday})
	if err != nil {
		return err
	}
	noonOf := func(d *time.Time) *time.Time {
		if d == nil {
			return nil
		}
		t := time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
		return &t
	}
	for _, b := range books {
		if b.Status == "booked" && b.CreativeStatus == "waiting" && b.CreativeDue != nil && !b.CreativeDue.After(civilToday.AddDate(0, 0, 7)) && !b.IssueOn.Before(civilToday) {
			out.Needs = append(out.Needs, RevenueNeed{Kind: "creative", Issue: int(b.IssueNumber), List: b.ListName, Sponsor: b.Sponsor, Due: noonOf(b.CreativeDue),
				Overdue: b.CreativeDue.Before(civilToday), TargetID: b.ID.String()})
		}
		if b.PaymentStatus == "unpaid" && b.IssueOn.Before(civilToday) {
			out.Needs = append(out.Needs, RevenueNeed{Kind: "sponsor_payment", Issue: int(b.IssueNumber), List: b.ListName, Sponsor: b.Sponsor, Due: noonOf(b.PaymentDue),
				Overdue: b.PaymentDue != nil && b.PaymentDue.Before(civilToday), TargetID: b.ID.String()})
		}
	}
	giftsOn, err := s.Q.RevenueGiftPageOn(ctx, ws.ID)
	if err != nil {
		return err
	}
	if giftsOn {
		emails, err := s.Q.RevenueUnpublishedMemberEmails(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, e := range emails {
			if strings.HasPrefix(e.Email, "gift_") {
				out.Needs = append(out.Needs, RevenueNeed{Kind: "member_email", Template: e.Key, TargetID: e.TemplateID.String()})
			}
		}
	}
	return nil
}

// activityKinds are the paid-member events the overview lists, the latest
// of each.
var activityKinds = []string{"payment_failed", "refunded", "cancel_scheduled", "offer_applied", "gift_redeemed", "subscribed"}

// activity is the latest of each kind of paid-member event, newest first.
func (s *Service) activity(ctx context.Context, ws *store.Workspace, out *RevenueOverview) error {
	rows, err := s.Q.RevenueActivity(ctx, store.RevenueActivityParams{WorkspaceID: ws.ID, Kinds: activityKinds})
	if err != nil {
		return err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OccurredAt.After(rows[j].OccurredAt) })
	for _, r := range rows {
		var d map[string]any
		_ = json.Unmarshal(r.Data, &d)
		if d == nil {
			d = map[string]any{}
		}
		name := strings.TrimSpace(r.FirstName + " " + r.LastName)
		if name == "" {
			name = r.Email
		}
		out.Activity = append(out.Activity, RevenueActivity{Kind: r.Kind, At: r.OccurredAt, ContactID: r.ContactID.String(), Name: name, Tier: r.Tier, Period: r.Period,
			AmountCents: r.AmountCents, Currency: r.Currency, Data: d})
	}
	return nil
}

// civilDay is a calendar day as the database's date type wants it.
func civilDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
