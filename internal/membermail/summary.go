package membermail

import (
	"html"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
)

// The sending summary (M57 G-14): a member who can view reports asked for a
// weekly or monthly email of a workspace's campaigns, and, as far as their
// role lets them see, its audience and sender health. internal/reports
// counts the numbers; this writes them in the member's language.

var (
	mSummarySubject     = msgs.M("summary.subject")
	mSummaryTitle       = msgs.M("summary.title")
	mSummaryPeriod      = msgs.M("summary.period")
	mSummaryWeek        = msgs.M("summary.week")
	mSummaryCampaigns   = msgs.M("summary.campaigns")
	mSummaryStarted     = msgs.M("summary.started")
	mSummaryCollecting  = msgs.M("summary.collecting")
	mSummaryNone        = msgs.M("summary.noCampaigns")
	mSummaryColTotal    = msgs.M("summary.colTotal")
	mSummaryColRate     = msgs.M("summary.colRate")
	mSummarySent        = msgs.M("summary.sent")
	mSummaryDelivered   = msgs.M("summary.delivered")
	mSummaryOpens       = msgs.M("summary.humanOpens")
	mSummaryClicks      = msgs.M("summary.clicks")
	mSummaryUnsubs      = msgs.M("summary.unsubscribes")
	mSummaryBounces     = msgs.M("summary.bounces")
	mSummaryComplaints  = msgs.M("summary.complaints")
	mSummaryNotReported = msgs.M("summary.notReported")
	mSummaryComplNote   = msgs.M("summary.complaintsNote")
	mSummaryTop         = msgs.M("summary.top")
	mSummaryColCampaign = msgs.M("summary.colCampaign")
	mSummaryOpenReport  = msgs.M("summary.openReport")
	mSummaryTopLine     = msgs.M("summary.topLine")
	mSummaryOpenAll     = msgs.M("summary.openCampaigns")
	mSummaryAudience    = msgs.M("summary.audience")
	mSummaryColPeople   = msgs.M("summary.colSubscribers")
	mSummaryJoined      = msgs.M("summary.joined")
	mSummaryDeparted    = msgs.M("summary.departed")
	mSummaryNet         = msgs.M("summary.net")
	mSummarySubscribers = msgs.M("summary.subscribers")
	mSummaryHealth      = msgs.M("summary.health")
	mSummaryHealthScore = msgs.M("summary.healthScore")
	mSummaryOpenDeliv   = msgs.M("summary.openDeliverability")
	mSummaryLine        = msgs.M("summary.line")
	mSummaryLineRate    = msgs.M("summary.lineRate")
	mSummaryWhy         = msgs.M("summary.why")
	mSummaryWhyText     = msgs.M("summary.whyText")
)

// ReportSummaryEmail is one member's sending summary of a workspace.
type ReportSummaryEmail struct {
	Workspace string
	// Frequency is weekly or monthly.
	Frequency string
	// From and To bound the period in the member's time zone: [From, To).
	From, To  time.Time
	Campaigns SummaryCampaigns
	// Top are the most opened campaigns, at most five.
	Top []SummaryCampaign
	// Audience and Health are nil for members whose role doesn't show them.
	Audience *SummaryAudience
	Health   *SummaryHealth
	Links    SummaryLinks
}

// SummaryCampaigns are the totals of the campaigns started in the period.
type SummaryCampaigns struct {
	// Started campaigns, of which Collecting are still within the 72
	// hours that opens and clicks keep counting.
	Started, Collecting                                        int
	Sent, Delivered, HumanOpens, Clicks, Unsubscribes, Bounces int64
	// Complaints and their rate of delivered count only the mailbox
	// providers that report complaints; Reported is false when none did.
	Complaints    int64
	ComplaintRate float64
	Reported      bool
}

// SummaryCampaign is one of the most opened campaigns.
type SummaryCampaign struct {
	Name                     string
	Sent, HumanOpens, Clicks int64
	// Link opens the campaign's report.
	Link string
}

// SummaryAudience is the newsletter's subscribers over the period.
type SummaryAudience struct {
	Joined, Departed int64
	// Subscribers are the lists' subscribed members when the email is written.
	Subscribers int64
}

// SummaryHealth is the workspace's sender health score.
type SummaryHealth struct {
	Score int
	// Label is Good, Fair or Poor (the Overview's word).
	Label string
}

// SummaryLinks are the screens the email links to.
type SummaryLinks struct {
	Campaigns, Deliverability string
	// Preferences is where the member turns the summary off.
	Preferences string
}

const (
	sumH2   = `<h2 style="margin:24px 0 8px;font-size:16px;line-height:1.3;font-weight:700;color:#0B1220">`
	sumP    = `<p style="margin:0 0 12px;font-size:14px;line-height:1.5">`
	sumMute = `<p style="margin:8px 0 12px;font-size:14px;line-height:1.5;color:#5B6472">`
	sumLink = `<a style="color:#0F766E" href="`
	sumTbl  = `<table cellpadding="0" cellspacing="0" border="0" style="width:100%;border-collapse:collapse;font-size:14px;line-height:1.4">`
	sumHead = `<th scope="col" style="padding:6px 0 6px 12px;border-bottom:1px solid #E3E6EB;text-align:right;font-size:14px;font-weight:600;color:#5B6472;white-space:nowrap">`
	sumLeft = `<th scope="col" style="padding:6px 0;border-bottom:1px solid #E3E6EB;text-align:left;font-size:14px;font-weight:600;color:#5B6472">`
	sumRow  = `<th scope="row" style="padding:8px 0;border-bottom:1px solid #EEF0F3;text-align:left;font-size:14px;font-weight:400">`
	sumNum  = `<td style="padding:8px 0 8px 12px;border-bottom:1px solid #EEF0F3;text-align:right;font-size:14px;white-space:nowrap">`
)

// ReportSummary writes the sending summary: an HTML part with tables and a
// plain-text part with the same numbers.
func ReportSummary(l i18n.Lang, d ReportSummaryEmail) mailer.Message {
	r := l.Region
	freq := i18n.Args{"frequency": d.Frequency}
	var period string
	if d.Frequency == "monthly" {
		period = r.MonthYear(d.From, true)
	} else {
		period = mSummaryWeek.Text(l, i18n.Args{"from": r.Date(d.From), "to": r.Date(d.To.AddDate(0, 0, -1))})
	}
	head := i18n.Args{"workspace": d.Workspace, "period": period}
	var text, body strings.Builder
	title := mSummaryTitle.Text(l, freq)
	text.WriteString(title + "\n" + mSummaryPeriod.Text(l, head) + "\n")
	body.WriteString(`<h1 style="margin:0 0 4px;font-size:20px;line-height:1.3;font-weight:700;color:#0B1220">` + html.EscapeString(title) + `</h1>`)
	body.WriteString(`<p style="margin:0 0 4px;font-size:14px;line-height:1.5;color:#5B6472">` + mSummaryPeriod.HTML(l, head, nil) + `</p>`)

	// Campaigns.
	c := d.Campaigns
	heading := mSummaryCampaigns.Text(l, nil)
	text.WriteString("\n" + heading + "\n")
	body.WriteString(sumH2 + html.EscapeString(heading) + `</h2>`)
	if c.Started == 0 {
		line := mSummaryNone.Text(l, nil)
		text.WriteString(line + "\n")
		body.WriteString(sumP + html.EscapeString(line) + `</p>`)
	} else {
		line := mSummaryStarted.Text(l, i18n.Args{"count": c.Started})
		if c.Collecting > 0 {
			line += " · " + mSummaryCollecting.Text(l, i18n.Args{"count": c.Collecting})
		}
		text.WriteString(line + "\n")
		body.WriteString(sumP + html.EscapeString(line) + `</p>`)
		body.WriteString(sumTbl + `<tr><td style="padding:6px 0;border-bottom:1px solid #E3E6EB"></td>` +
			sumHead + mSummaryColTotal.HTML(l, nil, nil) + `</th>` + sumHead + mSummaryColRate.HTML(l, nil, nil) + `</th></tr>`)
		row := func(label i18n.Msg, n int64, rate string) {
			name := label.Text(l, nil)
			if rate == "" {
				text.WriteString(mSummaryLine.Text(l, i18n.Args{"label": name, "value": r.Int(n)}) + "\n")
			} else {
				text.WriteString(mSummaryLineRate.Text(l, i18n.Args{"label": name, "value": r.Int(n), "rate": rate}) + "\n")
			}
			body.WriteString(`<tr>` + sumRow + html.EscapeString(name) + `</th>` + sumNum + html.EscapeString(r.Int(n)) + `</td>` + sumNum + html.EscapeString(rate) + `</td></tr>`)
		}
		pct := func(n, of int64, digits int) string {
			if of <= 0 {
				return ""
			}
			return r.Percent(float64(n)/float64(of)*100, digits)
		}
		row(mSummarySent, c.Sent, "")
		row(mSummaryDelivered, c.Delivered, pct(c.Delivered, c.Sent, 1))
		row(mSummaryOpens, c.HumanOpens, pct(c.HumanOpens, c.Delivered, 1))
		row(mSummaryClicks, c.Clicks, pct(c.Clicks, c.Delivered, 1))
		row(mSummaryUnsubs, c.Unsubscribes, pct(c.Unsubscribes, c.Delivered, 2))
		row(mSummaryBounces, c.Bounces, pct(c.Bounces, c.Sent, 2))
		if c.Reported {
			row(mSummaryComplaints, c.Complaints, r.Percent(c.ComplaintRate, 2))
		} else {
			name, na := mSummaryComplaints.Text(l, nil), mSummaryNotReported.Text(l, nil)
			text.WriteString(mSummaryLine.Text(l, i18n.Args{"label": name, "value": na}) + "\n")
			body.WriteString(`<tr>` + sumRow + html.EscapeString(name) + `</th>` + sumNum + `</td>` + sumNum + html.EscapeString(na) + `</td></tr>`)
		}
		body.WriteString(`</table>`)
		note := mSummaryComplNote.Text(l, nil)
		text.WriteString(note + "\n")
		body.WriteString(sumMute + html.EscapeString(note) + `</p>`)

		if len(d.Top) > 0 {
			top := mSummaryTop.Text(l, nil)
			text.WriteString("\n" + top + "\n")
			body.WriteString(sumH2 + html.EscapeString(top) + `</h2>` + sumTbl + `<tr>` + sumLeft + mSummaryColCampaign.HTML(l, nil, nil) + `</th>` +
				sumHead + mSummarySent.HTML(l, nil, nil) + `</th>` + sumHead + mSummaryOpens.HTML(l, nil, nil) + `</th>` + sumHead + mSummaryClicks.HTML(l, nil, nil) + `</th></tr>`)
			for _, t := range d.Top {
				open := mSummaryOpenReport.Text(l, i18n.Args{"campaign": t.Name})
				text.WriteString("• " + mSummaryTopLine.Text(l, i18n.Args{"campaign": t.Name, "sent": r.Int(t.Sent), "opens": r.Int(t.HumanOpens), "clicks": r.Int(t.Clicks)}) + "\n")
				text.WriteString("  " + mLinkLine.Text(l, i18n.Args{"label": open, "url": t.Link}) + "\n")
				// The name opens the report; its accessible name says so and
				// starts with what the reader sees.
				body.WriteString(`<tr>` + sumRow + sumLink + html.EscapeString(t.Link) + `" aria-label="` + html.EscapeString(open) + `">` + html.EscapeString(t.Name) + `</a></th>` +
					sumNum + html.EscapeString(r.Int(t.Sent)) + `</td>` + sumNum + html.EscapeString(r.Int(t.HumanOpens)) + `</td>` + sumNum + html.EscapeString(r.Int(t.Clicks)) + `</td></tr>`)
			}
			body.WriteString(`</table>`)
		}
	}
	if d.Links.Campaigns != "" {
		label := mSummaryOpenAll.Text(l, nil)
		text.WriteString(mLinkLine.Text(l, i18n.Args{"label": label, "url": d.Links.Campaigns}) + "\n")
		body.WriteString(`<p style="margin:12px 0 0;font-size:14px">` + sumLink + html.EscapeString(d.Links.Campaigns) + `">` + html.EscapeString(label) + `</a></p>`)
	}

	// Audience.
	if a := d.Audience; a != nil {
		heading := mSummaryAudience.Text(l, nil)
		text.WriteString("\n" + heading + "\n")
		body.WriteString(sumH2 + html.EscapeString(heading) + `</h2>` + sumTbl + `<tr><td style="padding:6px 0;border-bottom:1px solid #E3E6EB"></td>` +
			sumHead + mSummaryColPeople.HTML(l, nil, nil) + `</th></tr>`)
		net := r.Int(a.Joined - a.Departed)
		if a.Joined > a.Departed {
			net = "+" + net
		}
		for _, x := range []struct {
			label i18n.Msg
			value string
		}{{mSummaryJoined, r.Int(a.Joined)}, {mSummaryDeparted, r.Int(a.Departed)}, {mSummaryNet, net}, {mSummarySubscribers, r.Int(a.Subscribers)}} {
			name := x.label.Text(l, nil)
			text.WriteString(mSummaryLine.Text(l, i18n.Args{"label": name, "value": x.value}) + "\n")
			body.WriteString(`<tr>` + sumRow + html.EscapeString(name) + `</th>` + sumNum + html.EscapeString(x.value) + `</td></tr>`)
		}
		body.WriteString(`</table>`)
	}

	// Sender health.
	if h := d.Health; h != nil {
		heading := mSummaryHealth.Text(l, nil)
		score := i18n.Args{"score": r.Int(int64(h.Score)), "label": h.Label}
		text.WriteString("\n" + mSummaryLine.Text(l, i18n.Args{"label": heading, "value": mSummaryHealthScore.Text(l, score)}) + "\n")
		body.WriteString(sumH2 + html.EscapeString(heading) + `</h2>` + sumP + mSummaryHealthScore.HTML(l, score, nil) + `</p>`)
		if d.Links.Deliverability != "" {
			label := mSummaryOpenDeliv.Text(l, nil)
			text.WriteString(mLinkLine.Text(l, i18n.Args{"label": label, "url": d.Links.Deliverability}) + "\n")
			body.WriteString(`<p style="margin:0;font-size:14px">` + sumLink + html.EscapeString(d.Links.Deliverability) + `">` + html.EscapeString(label) + `</a></p>`)
		}
	}

	// Why it came, with the place to turn it off.
	ws := i18n.Args{"workspace": d.Workspace, "url": d.Links.Preferences}
	text.WriteString("\n—\n" + mSummaryWhyText.Text(l, ws) + "\n")
	body.WriteString(`<p style="margin:24px 0 0;font-size:14px;line-height:1.5;color:#5B6472">` + mSummaryWhy.HTML(l, ws, i18n.Tags{
		"link": func(inner string) string {
			return sumLink + html.EscapeString(d.Links.Preferences) + `">` + inner + `</a>`
		},
	}) + `</p>`)
	return mailer.Message{Kind: "report_summary", Subject: mSummarySubject.Text(l, i18n.Args{"workspace": d.Workspace, "frequency": d.Frequency}),
		Text: text.String(), HTML: body.String(), Lang: l}
}
