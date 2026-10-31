package reports

import (
	"context"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Insights are worked out from the report's own numbers (Report board):
// what stands out and what to do about it. They describe numbers; they
// never guess causes the numbers don't show.

// ReportInsight is one insight with its action.
type ReportInsight struct {
	Kind string `json:"kind" enum:"blocked,mailbox_bounces,top_link"`
	Tone string `json:"tone" enum:"bad,warn,info"`
	// Title and Detail are in the reader's language.
	Title  string `json:"title"`
	Detail string `json:"detail"`
	// Action is what to do: check_delisting (Deliverability's blocklist
	// checks), open_snds (Microsoft's Smart Network Data Services),
	// open_postmaster (Google Postmaster Tools), show_link (the link on the
	// click map); empty without one.
	Action      string `json:"action,omitempty" enum:"check_delisting,open_snds,open_postmaster,show_link"`
	ActionLabel string `json:"actionLabel,omitempty"`
	// Link is an outside address for the action (SNDS, Postmaster Tools).
	Link string `json:"link,omitempty"`
	// Position is the link's position on the click map (show_link).
	Position *int `json:"position,omitempty"`
	// Variant is the A/B variant whose map the link is on, for a test on
	// content (show_link); empty otherwise.
	Variant string `json:"variant,omitempty"`
	// detailNoWarmup is Detail without the sending provider's warm-up day,
	// for a read-only link that doesn't share the warm-up state (leaveOut);
	// empty when Detail doesn't mention it.
	detailNoWarmup string
}

// Outside tools the insights open.
const (
	sndsURL       = "https://sendersupport.olc.protection.outlook.com/snds/"
	postmasterURL = "https://postmaster.google.com/"
)

// Thresholds of the insights.
const (
	// A mailbox provider's bounce rate stands out at twice the other
	// providers' together, with enough bounces to mean something.
	bounceOutlier    = 2.0
	bounceOutlierMin = 20
	// A link stands out when it drew this share of human clicks.
	topLinkShare = 30.0
)

// codeRe finds a mailbox provider's own code in a bounce's code (Outlook's
// S3150, Gmail's 5.7.26 reasons are in the diagnostic): letters then digits.
var codeRe = regexp.MustCompile(`\b([A-Z]{1,3}\d{3,5})\b`)

func (s *Service) insights(ctx context.Context, l i18n.Lang, ws *store.Workspace, c store.ReportCampaignRow, out *CampaignReport) ([]ReportInsight, error) {
	ins := []ReportInsight{}
	// Blocks for the sender's reputation: the mailbox provider that
	// blocked most, through which provider and address.
	cid := c.ID
	blocked, err := s.Q.ReportBlockedBounces(ctx, store.ReportBlockedBouncesParams{CampaignID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	if len(blocked) > 0 {
		var total int64
		for _, b := range blocked {
			total += b.N
		}
		top := blocked[0]
		code := ""
		for _, row := range out.Bounces.Rows {
			if row.Type != "blocked" {
				continue
			}
			for _, cc := range row.Codes {
				if m := codeRe.FindStringSubmatch(cc.Code); m != nil {
					code = m[1]
				}
			}
		}
		args := i18n.Args{"n": total, "count": formatCount(l, total), "mailbox": mailbox.Name(mailboxKey(top.MailboxProvider)), "code": orNone(code)}
		in := ReportInsight{Kind: "blocked", Tone: "bad", Title: mInsightBlocked.Text(l, args), Action: "check_delisting", ActionLabel: mInsightCheckDelisting.Text(l, nil)}
		short := provstate.Short(top.ProviderName, top.Adapter)
		day := "none"
		for _, col := range out.Providers.Columns {
			if top.ProviderID != nil && col.ProviderID == top.ProviderID.String() && col.WarmupDay != nil {
				day = formatCount(l, int64(*col.WarmupDay))
			}
		}
		ip := "none"
		if len(top.Ips) == 1 {
			ip = top.Ips[0]
		}
		detail := i18n.Args{"all": yesNo(top.N == total), "n": total, "count": formatCount(l, total), "provider": short, "ip": ip, "day": day}
		in.Detail = mInsightBlockedDetail.Text(l, detail)
		if day != "none" {
			detail["day"] = "none"
			in.detailNoWarmup = mInsightBlockedDetail.Text(l, detail)
		}
		ins = append(ins, in)
	}
	if in := mailboxBouncesInsight(l, out.Mailboxes); in != nil {
		ins = append(ins, *in)
	}
	if in := topLinkInsight(l, out.ClickMap); in != nil {
		ins = append(ins, *in)
	}
	return ins, nil
}

// mailboxBouncesInsight is a mailbox provider bouncing far more than the
// others together, with what to open about it: Microsoft SNDS for Outlook,
// Google Postmaster Tools for Gmail.
func mailboxBouncesInsight(l i18n.Lang, mailboxes []ReportMailboxRow) *ReportInsight {
	type rate struct {
		key, name   string
		v           float64
		sent, bounc int64
	}
	var rates []rate
	for _, m := range mailboxes {
		if m.BounceRate == nil {
			continue
		}
		rates = append(rates, rate{key: m.Key, name: m.Name, v: *m.BounceRate, sent: m.Sent, bounc: int64(math.Round(*m.BounceRate / 100 * float64(m.Sent)))})
	}
	if len(rates) < 2 {
		return nil
	}
	slices.SortStableFunc(rates, func(a, b rate) int {
		switch {
		case a.v > b.v:
			return -1
		case a.v < b.v:
			return 1
		}
		return 0
	})
	top := rates[0]
	// The other mailbox providers together.
	var sent, bounced int64
	for _, r := range rates[1:] {
		sent += r.sent
		bounced += r.bounc
	}
	others := 0.0
	if sent > 0 {
		others = float64(bounced) / float64(sent) * 100
	}
	if others <= 0 || top.v < bounceOutlier*others || top.bounc < bounceOutlierMin {
		return nil
	}
	times := math.Round(top.v / others)
	in := ReportInsight{Kind: "mailbox_bounces", Tone: "warn",
		Title: mInsightMailboxBounces.Text(l, i18n.Args{"mailbox": top.name, "rate": formatRate(l, top.v), "times": int(times),
			"others": formatRate(l, round(others, 1))}),
		Detail: mInsightMailboxBouncesDetail.Text(l, i18n.Args{"n": top.bounc, "bounced": formatCount(l, top.bounc), "sent": formatCount(l, top.sent)})}
	switch top.key {
	case "outlook":
		in.Action, in.ActionLabel, in.Link = "open_snds", mInsightOpenSNDS.Text(l, nil), sndsURL
		in.Detail += " " + mInsightSNDSDetail.Text(l, nil)
	case "gmail":
		in.Action, in.ActionLabel, in.Link = "open_postmaster", mInsightOpenPostmaster.Text(l, nil), postmasterURL
	}
	return &in
}

// topLinkInsight is a link that drew much of its email's human clicks, to
// show on the click map. A test on content has an email per variant (the
// map is PerVariant): each variant's links are measured against its own
// clicks, and the insight names the variant whose map it is on.
func topLinkInsight(l i18n.Lang, cm ReportClickMap) *ReportInsight {
	type email struct {
		variant string
		links   []ReportLink
	}
	emails := []email{{links: cm.Links}}
	if cm.PerVariant {
		emails = emails[:0]
		for _, v := range cm.Variants {
			emails = append(emails, email{variant: v.Key, links: v.Links})
		}
	}
	var best *ReportLink
	variant := ""
	for _, e := range emails {
		var top *ReportLink
		for i := range e.links {
			if lk := &e.links[i]; top == nil || lk.UniqueClicks > top.UniqueClicks {
				top = lk
			}
		}
		if top == nil || top.Share < topLinkShare || top.UniqueClicks == 0 {
			continue
		}
		if best == nil || top.Share > best.Share || (top.Share == best.Share && top.UniqueClicks > best.UniqueClicks) {
			best, variant = top, e.variant
		}
	}
	if best == nil {
		return nil
	}
	pos := best.Position
	label := best.Label
	if label == "" {
		label = strings.TrimPrefix(strings.TrimPrefix(best.URL, "https://"), "http://")
	}
	args := i18n.Args{"link": label, "share": formatShare(l, best.Share), "n": best.UniqueClicks, "people": formatCount(l, best.UniqueClicks), "variant": variant}
	in := ReportInsight{Kind: "top_link", Tone: "info", Title: mInsightTopLink.Text(l, args), Detail: mInsightTopLinkDetail.Text(l, args),
		Action: "show_link", ActionLabel: mInsightShowLink.Text(l, nil), Position: &pos, Variant: variant}
	if variant != "" {
		in.Title, in.Detail = mInsightTopLinkVariant.Text(l, args), mInsightTopLinkVariantDetail.Text(l, args)
	}
	return &in
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func formatCount(l i18n.Lang, n int64) string { return l.Region.Int(n) }

func formatRate(l i18n.Lang, v float64) string { return l.Region.Percent(v, 1) }

func formatShare(l i18n.Lang, v float64) string { return l.Region.Percent(math.Round(v), 0) }
