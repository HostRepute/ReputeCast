package publicpages

// The shared sponsor report (design expansion phase 6, X17): a read-only
// link for a sponsor (/r/{token}) with one booking's results, or every
// booking of a sponsor with the issues to come. No contact data, never
// indexed. Campaign reports' links share the path: their tokens go on to
// the web app.

import (
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sponsors"
	"github.com/hostrepute/reputecast/internal/store"
)

// reportTile is one number of the report.
type reportTile struct{ Label, Value, Sub string }

// reportLink is a link's share of the clicks.
type reportLink struct {
	Label, Count, Share string
	Pct                 int
}

// reportCreative is the block as sent.
type reportCreative struct {
	Label, Initials, Headline, Text, Button, Logo, When string
}

// everyRow is one booking of a sponsor's every-booking report.
type everyRow struct {
	Title, Line, Sub string
	Final, Coming    bool
}

// sponsorReportBody is the shared sponsor report in one of its states:
// report (one booking), every (a sponsor's bookings) or gone.
type sponsorReportBody struct {
	State     string
	Title     string
	Tag       string
	Coming    bool
	Sub       string
	Tiles     []reportTile
	NoteTitle string
	NoteText  string
	ByLink    string
	Links     []reportLink
	LinksNote template.HTML
	Creative  *reportCreative
	CreativeH string
	HowLabel  string
	HowText   string
	Rows      []everyRow
	EveryNote string
	GoneText  string
	Footer    string
}

func (h *Handler) sharedReportRoutes(r chi.Router) {
	r.Get("/r/{token}", h.sharedReport)
}

// spaOr404 hands a path the public pages don't answer to the web app.
func (h *Handler) spaOr404(w http.ResponseWriter, r *http.Request) {
	if h.SPA != nil {
		h.SPA.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

// siteFor is a workspace's pages on the instance's address.
func (h *Handler) siteFor(ctx context.Context, ws *store.Workspace) (*site, error) {
	set, err := h.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return nil, err
	}
	return &site{ws: ws, set: set, canon: h.Forms.Links(ws, set), links: forms.LinksOn(h.URLs.Base(), ws.Slug, false)}, nil
}

func (h *Handler) sharedReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if h.Sponsors == nil || !h.Sponsors.IsShareToken(ctx, token) {
		h.spaOr404(w, r)
		return
	}
	rep, err := h.Sponsors.Shared(ctx, token)
	status := http.StatusOK
	var body sponsorReportBody
	ws := rep.Workspace
	switch {
	case errors.Is(err, sponsors.ErrShareGone):
		sh, serr := h.Q.SponsorShareByToken(ctx, authn.HashToken(token))
		if serr != nil {
			h.spaOr404(w, r)
			return
		}
		if ws, serr = h.Q.GetWorkspace(ctx, sh.WorkspaceID); serr != nil {
			h.spaOr404(w, r)
			return
		}
		status = http.StatusGone
		body = sponsorReportBody{State: "gone", Title: sponsors.Text("shared.goneTitle", nil), GoneText: sponsors.Text("shared.goneText", i18n.Args{"publisher": ws.Name})}
	case err != nil:
		h.serverError(w, r, nil, err)
		return
	default:
		body = h.sponsorReportView(&ws, rep)
	}
	s, err := h.siteFor(ctx, &ws)
	if err != nil {
		h.serverError(w, r, nil, err)
		return
	}
	vw := h.earnView(s, h.earnTitle(s, body.Title), body)
	// Private: never cached, never indexed, never sent as a referrer.
	h.render(w, r, s, "sponsorreport", status, vw, csp{}, true)
}

// reportDay is "Tue 7 Oct 2026" in the workspace's time zone.
func reportDay(t time.Time, ws *store.Workspace) string {
	return t.In(wsLocation(ws)).Format("Mon 2 Jan 2006")
}

func wsLocation(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// lowerFirst writes "After the first story" as "after the first story".
func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 || len(s) <= n {
		return s
	}
	next, _ := utf8.DecodeRuneInString(s[n:])
	if unicode.IsUpper(r) && unicode.IsLower(next) {
		return string(unicode.ToLower(r)) + s[n:]
	}
	return s
}

// reportPct is a rate with one decimal ("1.3%", "44.8%").
func reportPct(v float64) string {
	if v == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", v)
}

// reportCount is a count with thousands separators ("31,299").
func reportCount(n int64) string {
	s := fmt.Sprint(n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// monoHTML escapes a sentence and sets the given parts in monospace.
func monoHTML(text string, parts ...string) template.HTML {
	out := html.EscapeString(text)
	for _, p := range parts {
		if p == "" {
			continue
		}
		e := html.EscapeString(p)
		out = strings.Replace(out, e, `<span class="mono">`+e+`</span>`, 1)
	}
	return template.HTML(out) //nolint:gosec // escaped above
}

// sponsorReportView builds the page of a resolved link.
func (h *Handler) sponsorReportView(ws *store.Workspace, rep sponsors.SharedReport) sponsorReportBody {
	b := sponsorReportBody{Title: sponsors.Text("shared.title", nil), HowLabel: sponsors.Text("shared.howLabel", nil), HowText: sponsors.Text("shared.howText", nil)}
	if rep.ExpiresAt != nil {
		b.Footer = sponsors.Text("shared.shared", i18n.Args{"publisher": ws.Name, "date": rep.ExpiresAt.In(wsLocation(ws)).Format("2 Jan 2006")})
	} else {
		b.Footer = sponsors.Text("shared.sharedNoEnd", i18n.Args{"publisher": ws.Name})
	}
	if rep.Every {
		b.State = "every"
		list := ""
		if len(rep.Reports) > 0 {
			list = rep.Reports[0].ListName
		}
		if list != "" {
			b.Sub = sponsors.Text("shared.everySub", i18n.Args{"sponsor": rep.Sponsor, "list": list})
		} else {
			b.Sub = sponsors.Text("shared.everySubNoList", i18n.Args{"sponsor": rep.Sponsor})
		}
		total := len(rep.Reports) + len(rep.Upcoming)
		b.Tag = sponsors.Text("shared.sentOf", i18n.Args{"sent": len(rep.Reports), "total": total})
		for _, r := range rep.Reports {
			row := everyRow{Title: sponsors.Text("shared.issueRow", i18n.Args{"issue": r.Issue, "date": r.IssueOn.In(wsLocation(ws)).Format("Mon 2 Jan")}), Final: r.Final}
			if r.Final {
				row.Line = sponsors.Text("shared.finalRow", i18n.Args{"delivered": reportCount(r.Delivered)})
			} else {
				row.Line = sponsors.Text("shared.still", nil)
			}
			row.Sub = sponsors.Text("shared.rowClicks", i18n.Args{"clickers": r.UniqueClickers, "rate": reportPct(r.ClickRate)})
			b.Rows = append(b.Rows, row)
		}
		for _, u := range rep.Upcoming {
			b.Rows = append(b.Rows, everyRow{Title: sponsors.Text("shared.issueRow", i18n.Args{"issue": u.Number, "date": u.IssueOn.In(wsLocation(ws)).Format("Mon 2 Jan")}),
				Line: sponsors.Text("shared.comingUp", nil), Sub: sponsors.Text("shared.comingUpSub", nil), Coming: true})
		}
		b.EveryNote = sponsors.Text("shared.everyNote", nil)
		return b
	}
	r := rep.Reports[0]
	b.State = "report"
	b.Sub = sponsors.Text("shared.sub", i18n.Args{"list": r.ListName, "issue": r.Issue, "date": reportDay(r.IssueOn, ws), "sponsor": r.Sponsor})
	tiles := []reportTile{{Label: sponsors.Text("shared.delivered", nil), Value: reportCount(r.Delivered), Sub: sponsors.Text("shared.deliveredSub", nil)}}
	if r.Final {
		b.Tag = sponsors.Text("shared.final", nil)
		tiles = append(tiles,
			reportTile{Label: sponsors.Text("shared.opens", nil), Value: reportCount(r.HumanOpens), Sub: sponsors.Text("shared.opensSub", i18n.Args{"rate": reportPct(r.HumanOpenRate)})},
			reportTile{Label: sponsors.Text("shared.clickers", nil), Value: reportCount(r.UniqueClickers), Sub: sponsors.Text("shared.clickersSub", i18n.Args{"sponsor": r.Sponsor})},
			reportTile{Label: sponsors.Text("shared.clickRate", nil), Value: reportPct(r.ClickRate), Sub: sponsors.Text("shared.clickRateSub", nil)},
			reportTile{Label: sponsors.Text("shared.cto", nil), Value: reportPct(r.ClickToOpen), Sub: sponsors.Text("shared.ctoSub", nil)})
		parts := make([]string, 0, len(r.Links))
		for _, l := range r.Links {
			parts = append(parts, sponsors.Text("shared.linkCount", i18n.Args{"kind": l.Kind, "count": l.Clicks}))
		}
		sub := strings.Join(parts, " · ")
		if sub == "" {
			sub = sponsors.Text("shared.clicksNone", nil)
		}
		tiles = append(tiles, reportTile{Label: sponsors.Text("shared.clicks", nil), Value: reportCount(r.Clicks), Sub: sub})
	} else {
		b.Tag, b.Coming = sponsors.Text("shared.still", nil), true
		tiles = append(tiles,
			reportTile{Label: sponsors.Text("shared.opensSoFar", nil), Value: reportCount(r.HumanOpens), Sub: sponsors.Text("shared.opensSub", i18n.Args{"rate": reportPct(r.HumanOpenRate)})},
			reportTile{Label: sponsors.Text("shared.clickersSoFar", nil), Value: reportCount(r.UniqueClickers), Sub: sponsors.Text("shared.clickersSubSoFar", i18n.Args{"rate": reportPct(r.ClickRate)})})
		b.NoteTitle = sponsors.Text("shared.comingTitle", nil)
		if r.FinalAt != nil {
			b.NoteText = sponsors.Text("shared.comingText", i18n.Args{"date": r.FinalAt.In(wsLocation(ws)).Format("Mon 2 Jan, 15:04")})
		}
	}
	b.Tiles = tiles
	if r.Final && len(r.Links) > 0 {
		b.ByLink = sponsors.Text("shared.byLink", nil)
		for _, l := range r.Links {
			label := sponsors.Text("shared.link.other", i18n.Args{"label": l.Label})
			switch l.Kind {
			case "button":
				label = sponsors.Text("shared.link.button", i18n.Args{"label": l.Label})
			case "logo":
				label = sponsors.Text("shared.link.logo", nil)
			}
			b.Links = append(b.Links, reportLink{Label: label, Count: reportCount(l.Clicks), Share: fmt.Sprintf("%.0f%%", l.Share), Pct: int(l.Share + 0.5)})
		}
		if r.LinkHost != "" {
			utm := ""
			if r.LinkCampaign != "" {
				utm = "utm_campaign=" + r.LinkCampaign
			}
			args := i18n.Args{"count": len(r.Links), "host": r.LinkHost, "utm": utm}
			if utm != "" {
				b.LinksNote = monoHTML(sponsors.Text("shared.linksGo", args), r.LinkHost, utm)
			} else {
				b.LinksNote = monoHTML(sponsors.Text("shared.linksGoNoUTM", args), r.LinkHost)
			}
		}
	}
	c := r.Creative
	if c.Headline != "" || c.Text != "" || c.Button != "" {
		when := ""
		if r.SentAt != nil {
			when = sponsors.Text("shared.creativeWhen", i18n.Args{"when": r.SentAt.In(wsLocation(ws)).Format("Mon 2 Jan, 15:04"), "placement": lowerFirst(r.Placement)})
		}
		b.CreativeH = sponsors.Text("shared.creative", nil)
		b.Creative = &reportCreative{Label: r.Label, Initials: r.Initials, Headline: c.Headline, Text: c.Text, Button: c.Button, Logo: c.Logo, When: when}
	}
	return b
}
