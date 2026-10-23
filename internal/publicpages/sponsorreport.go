package publicpages

// The shared sponsor report (design expansion phase 6, X17): a read-only
// link for a sponsor (/r/{token}) with one booking's results, or every
// booking of a sponsor with the issues to come. No contact data, never
// indexed. Campaign reports' links share the path: their tokens go on to
// the web app. It reads in the page's language, like the workspace's
// other public pages, with dates and numbers in its region format.

import (
	"context"
	"errors"
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
	ws := rep.Workspace
	gone := errors.Is(err, sponsors.ErrShareGone)
	switch {
	case gone:
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
	case err != nil:
		h.serverError(w, r, nil, err)
		return
	}
	s, err := h.siteFor(ctx, &ws)
	if err != nil {
		h.serverError(w, r, nil, err)
		return
	}
	pl := h.visitorLang(w, r, s, nil)
	l := pl.Labels.Lang()
	var body sponsorReportBody
	if gone {
		body = sponsorReportBody{State: "gone", Title: sponsors.TextIn(l, "shared.goneTitle", nil), GoneText: sponsors.TextIn(l, "shared.goneText", i18n.Args{"publisher": ws.Name})}
	} else {
		body = h.sponsorReportView(&ws, rep, l)
	}
	vw := h.earnView(r, s, h.earnTitle(s, body.Title), body, pl)
	// Private: never cached, never indexed, never sent as a referrer.
	h.render(w, r, s, "sponsorreport", status, vw, csp{}, true)
}

// reportDay is "Tue 7 Oct 2026" in the workspace's time zone.
func reportDay(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	t = t.In(wsLocation(ws))
	if standardRegion(l) {
		return t.Format("Mon 2 Jan 2006")
	}
	return l.Region.WeekdayDate(t)
}

// reportShortDay is "Tue 7 Oct" in the workspace's time zone.
func reportShortDay(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	t = t.In(wsLocation(ws))
	if standardRegion(l) {
		return t.Format("Mon 2 Jan")
	}
	return l.Region.ShortDate(t)
}

// reportStamp is "Tue 7 Oct, 09:00" in the workspace's time zone.
func reportStamp(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	t = t.In(wsLocation(ws))
	if standardRegion(l) {
		return t.Format("Mon 2 Jan, 15:04")
	}
	return l.Region.ShortDate(t) + ", " + l.Region.Time(t, "")
}

// reportDate is "7 Oct 2026" in the workspace's time zone.
func reportDate(l i18n.Lang, t time.Time, ws *store.Workspace) string {
	t = t.In(wsLocation(ws))
	if standardRegion(l) {
		return t.Format("2 Jan 2006")
	}
	return l.Region.Date(t)
}

// standardRegion reports whether a language writes ReputeCast's own formats.
func standardRegion(l i18n.Lang) bool { return l.Region == "" || l.Region == i18n.Standard }

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

// reportPct is a rate with one decimal ("1.3%", "44.8%"), in the page's
// region format ("1,3%").
func reportPct(l i18n.Lang, v float64) string {
	if v == 0 {
		return l.Region.Percent(0, 0)
	}
	return l.Region.Percent(v, 1)
}

// reportCount is a count with thousands separators ("31,299", "31.299").
func reportCount(l i18n.Lang, n int64) string { return l.Region.Int(n) }

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

// sponsorReportView builds the page of a resolved link in a language.
func (h *Handler) sponsorReportView(ws *store.Workspace, rep sponsors.SharedReport, l i18n.Lang) sponsorReportBody {
	t := func(key string, args i18n.Args) string { return sponsors.TextIn(l, key, args) }
	b := sponsorReportBody{Title: t("shared.title", nil), HowLabel: t("shared.howLabel", nil), HowText: t("shared.howText", nil)}
	if rep.ExpiresAt != nil {
		b.Footer = t("shared.shared", i18n.Args{"publisher": ws.Name, "date": reportDate(l, *rep.ExpiresAt, ws)})
	} else {
		b.Footer = t("shared.sharedNoEnd", i18n.Args{"publisher": ws.Name})
	}
	if rep.Every {
		b.State = "every"
		list := ""
		if len(rep.Reports) > 0 {
			list = rep.Reports[0].ListName
		}
		if list != "" {
			b.Sub = t("shared.everySub", i18n.Args{"sponsor": rep.Sponsor, "list": list})
		} else {
			b.Sub = t("shared.everySubNoList", i18n.Args{"sponsor": rep.Sponsor})
		}
		total := len(rep.Reports) + len(rep.Upcoming)
		b.Tag = t("shared.sentOf", i18n.Args{"sent": len(rep.Reports), "total": total})
		for _, r := range rep.Reports {
			row := everyRow{Title: t("shared.issueRow", i18n.Args{"issue": r.Issue, "date": reportShortDay(l, r.IssueOn, ws)}), Final: r.Final}
			if r.Final {
				row.Line = t("shared.finalRow", i18n.Args{"delivered": reportCount(l, r.Delivered)})
			} else {
				row.Line = t("shared.still", nil)
			}
			row.Sub = t("shared.rowClicks", i18n.Args{"clickers": r.UniqueClickers, "rate": reportPct(l, r.ClickRate)})
			b.Rows = append(b.Rows, row)
		}
		for _, u := range rep.Upcoming {
			b.Rows = append(b.Rows, everyRow{Title: t("shared.issueRow", i18n.Args{"issue": u.Number, "date": reportShortDay(l, u.IssueOn, ws)}),
				Line: t("shared.comingUp", nil), Sub: t("shared.comingUpSub", nil), Coming: true})
		}
		b.EveryNote = t("shared.everyNote", nil)
		return b
	}
	r := rep.Reports[0]
	b.State = "report"
	b.Sub = t("shared.sub", i18n.Args{"list": r.ListName, "issue": r.Issue, "date": reportDay(l, r.IssueOn, ws), "sponsor": r.Sponsor})
	tiles := []reportTile{{Label: t("shared.delivered", nil), Value: reportCount(l, r.Delivered), Sub: t("shared.deliveredSub", nil)}}
	if r.Final {
		b.Tag = t("shared.final", nil)
		tiles = append(tiles,
			reportTile{Label: t("shared.opens", nil), Value: reportCount(l, r.HumanOpens), Sub: t("shared.opensSub", i18n.Args{"rate": reportPct(l, r.HumanOpenRate)})},
			reportTile{Label: t("shared.clickers", nil), Value: reportCount(l, r.UniqueClickers), Sub: t("shared.clickersSub", i18n.Args{"sponsor": r.Sponsor})},
			reportTile{Label: t("shared.clickRate", nil), Value: reportPct(l, r.ClickRate), Sub: t("shared.clickRateSub", nil)},
			reportTile{Label: t("shared.cto", nil), Value: reportPct(l, r.ClickToOpen), Sub: t("shared.ctoSub", nil)})
		parts := make([]string, 0, len(r.Links))
		for _, k := range r.Links {
			parts = append(parts, t("shared.linkCount", i18n.Args{"kind": k.Kind, "count": k.Clicks}))
		}
		sub := strings.Join(parts, " · ")
		if sub == "" {
			sub = t("shared.clicksNone", nil)
		}
		tiles = append(tiles, reportTile{Label: t("shared.clicks", nil), Value: reportCount(l, r.Clicks), Sub: sub})
	} else {
		b.Tag, b.Coming = t("shared.still", nil), true
		tiles = append(tiles,
			reportTile{Label: t("shared.opensSoFar", nil), Value: reportCount(l, r.HumanOpens), Sub: t("shared.opensSub", i18n.Args{"rate": reportPct(l, r.HumanOpenRate)})},
			reportTile{Label: t("shared.clickersSoFar", nil), Value: reportCount(l, r.UniqueClickers), Sub: t("shared.clickersSubSoFar", i18n.Args{"rate": reportPct(l, r.ClickRate)})})
		b.NoteTitle = t("shared.comingTitle", nil)
		if r.FinalAt != nil {
			b.NoteText = t("shared.comingText", i18n.Args{"date": reportStamp(l, *r.FinalAt, ws)})
		}
	}
	b.Tiles = tiles
	if r.Final && len(r.Links) > 0 {
		b.ByLink = t("shared.byLink", nil)
		for _, k := range r.Links {
			label := t("shared.link.other", i18n.Args{"label": k.Label})
			switch k.Kind {
			case "button":
				label = t("shared.link.button", i18n.Args{"label": k.Label})
			case "logo":
				label = t("shared.link.logo", nil)
			}
			b.Links = append(b.Links, reportLink{Label: label, Count: reportCount(l, k.Clicks), Share: l.Region.Percent(k.Share, 0), Pct: int(k.Share + 0.5)})
		}
		if r.LinkHost != "" {
			utm := ""
			if r.LinkCampaign != "" {
				utm = "utm_campaign=" + r.LinkCampaign
			}
			args := i18n.Args{"count": len(r.Links), "host": r.LinkHost, "utm": utm}
			if utm != "" {
				b.LinksNote = monoHTML(t("shared.linksGo", args), r.LinkHost, utm)
			} else {
				b.LinksNote = monoHTML(t("shared.linksGoNoUTM", args), r.LinkHost)
			}
		}
	}
	c := r.Creative
	if c.Headline != "" || c.Text != "" || c.Button != "" {
		when := ""
		if r.SentAt != nil {
			// English writes the placement in lower case inside the sentence
			// ("after the first story"); a translation keeps it as it is.
			placement := r.Placement
			if sponsors.EnglishText(l, "shared.creativeWhen") {
				placement = lowerFirst(placement)
			}
			when = t("shared.creativeWhen", i18n.Args{"when": reportStamp(l, *r.SentAt, ws), "placement": placement})
		}
		b.CreativeH = t("shared.creative", nil)
		b.Creative = &reportCreative{Label: r.Label, Initials: r.Initials, Headline: c.Headline, Text: c.Text, Button: c.Button, Logo: c.Logo, When: when}
	}
	return b
}
