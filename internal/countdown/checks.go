package countdown

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// Checks of emails with Countdown blocks: the composer warns about one
// that already ended, Send Readiness blocks a send that starts after a
// countdown ends (and warns when it ends within the hour), and publishing
// a flow warns when a countdown ends within 30 days, after which the flow
// keeps sending an image that only shows its end.

// IssueEnded is the check of a countdown that has ended (the composer's
// warning, Send Readiness's block); IssueSoon Send Readiness's warning of
// one that ends within an hour of the send's start.
const (
	IssueEnded = "content.countdown_ended"
	IssueSoon  = "content.countdown_soon"
)

// Lookahead windows.
const (
	soonAfterStart = time.Hour
	flowWithin     = 30 * 24 * time.Hour
)

// tokenRe finds a countdown image's token in an address (an image token is
// at least 67 bytes, 90 characters).
var tokenRe = regexp.MustCompile(`/t/cd/([A-Za-z0-9_-]{80,})`)

// Found is a countdown an email shows.
type Found struct {
	// ID is the countdown's; uuid.Nil for a block whose image wasn't made
	// yet (only its settings are known).
	ID       uuid.UUID
	Settings CountdownSettings
	EndsAt   time.Time
	// Path is the block's place from mj-body, and Section its top-level
	// section (-1 when the countdown was found in HTML only).
	Path    []int
	Section int
}

// Find lists the countdowns that content (MJML or HTML) shows: by their
// image's token, and by the settings of blocks whose image isn't made yet,
// soonest first. Tokens of another workspace count for nothing.
func (s *Service) Find(ctx context.Context, ws *store.Workspace, contents ...string) []Found {
	type place struct {
		path    []int
		section int
	}
	var ids []uuid.UUID
	places := map[uuid.UUID]place{}
	var drafts []Found
	idOf := func(token string) uuid.UUID {
		c, err := s.Links.Verify(token, Kind)
		if err != nil || c.WorkspaceID != ws.ID || c.MessageID == uuid.Nil {
			return uuid.Nil
		}
		if !slices.Contains(ids, c.MessageID) {
			ids = append(ids, c.MessageID)
		}
		return c.MessageID
	}
	for _, src := range contents {
		if !strings.Contains(src, "/t/cd/") && !strings.Contains(src, mjml.AttrCountdown) {
			continue
		}
		for _, m := range tokenRe.FindAllStringSubmatch(src, -1) {
			idOf(m[1])
		}
		if !strings.Contains(src, mjml.AttrCountdown) {
			continue
		}
		root, err := mjml.Parse(src)
		if err != nil {
			continue
		}
		body := mjml.Body(root)
		if body == nil {
			continue
		}
		body.Walk(func(n *mjml.Node, path []int) {
			raw, ok := n.Get(mjml.AttrCountdown)
			if !ok || len(path) == 0 {
				return
			}
			at := place{path: slices.Clone(path), section: path[0]}
			if m := tokenRe.FindStringSubmatch(n.Attr("src")); m != nil {
				if id := idOf(m[1]); id != uuid.Nil {
					places[id] = at
					return
				}
			}
			var in CountdownSettings
			if json.Unmarshal([]byte(raw), &in) != nil {
				return
			}
			if set, endsAt, err := Normalize(ctx, in); err == nil {
				drafts = append(drafts, Found{Settings: set, EndsAt: endsAt, Path: at.path, Section: at.section})
			}
		})
	}
	out := make([]Found, 0, len(ids)+len(drafts))
	if len(ids) > 0 {
		rows, err := s.Q.CountdownsByIDs(ctx, store.CountdownsByIDsParams{WorkspaceID: ws.ID, Ids: ids})
		if err != nil {
			s.log().WarnContext(ctx, "countdowns of an email", "err", err)
		}
		for _, r := range rows {
			f := Found{ID: r.ID, EndsAt: r.EndsAt, Path: []int{}, Section: -1}
			if json.Unmarshal(r.Settings, &f.Settings) != nil {
				continue
			}
			if p, ok := places[r.ID]; ok {
				f.Path, f.Section = p.path, p.section
			}
			out = append(out, f)
		}
	}
	out = append(out, drafts...)
	slices.SortStableFunc(out, func(a, b Found) int { return a.EndsAt.Compare(b.EndsAt) })
	return out
}

// zone is where a countdown ends.
func (f Found) zone() *time.Location {
	if loc, err := time.LoadLocation(f.Settings.TimeZone); err == nil {
		return loc
	}
	return time.UTC
}

// ComposerIssues warns about countdowns of a campaign's MJML that have
// already ended (campaigns.MoreChecks).
func (s *Service) ComposerIssues(ctx context.Context, ws *store.Workspace, _ uuid.UUID, src string) []contentcheck.ContentCheckIssue {
	if !strings.Contains(src, "/t/cd/") && !strings.Contains(src, mjml.AttrCountdown) {
		return nil
	}
	now := s.now()
	l := i18n.LangFrom(ctx)
	var out []contentcheck.ContentCheckIssue
	for _, f := range s.Find(ctx, ws, src) {
		if f.EndsAt.After(now) {
			continue
		}
		title, tag := keep("check.ended", nil), keep("check.tag", nil)
		detail := keep("check.endedDetail", i18n.Args{"at": f.EndsAt.In(f.zone()), "mode": f.Settings.Ended.Mode})
		out = append(out, contentcheck.ContentCheckIssue{ID: IssueEnded, Severity: contentcheck.Warn, Group: "Content",
			Title: title.In(l), Detail: detail.In(l), Tag: tag.In(l), Path: f.Path, Section: f.Section, Fixes: []contentcheck.ContentCheckFix{},
			TitleText: title, DetailText: []i18n.Text{detail}, TagText: tag})
	}
	return out
}

// ReadinessRow is a Send Readiness row about a campaign's countdowns; its
// texts are kept messages, written in each reader's language.
type ReadinessRow struct {
	Key, Status               string // block or warn
	Title, Detail, Label, Ack i18n.Text
	// Link opens the campaign in the composer (a workspace path).
	LinkLabel i18n.Text
	Href      string
}

// ReadinessRows checks the countdowns of a campaign's HTML against when its
// send starts: one that has ended by then blocks the send; one that ends
// within an hour of it warns. at writes a moment as Send Readiness keeps
// it, in the workspace's time zone.
func (s *Service) ReadinessRows(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, html string, start time.Time, at func(time.Time) any) []ReadinessRow {
	if !strings.Contains(html, "/t/cd/") {
		return nil
	}
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	var blocked, soon *Found
	found := s.Find(ctx, ws, html)
	for i := range found {
		f := &found[i]
		switch {
		case !f.EndsAt.After(start) && blocked == nil:
			blocked = f
		case f.EndsAt.After(start) && f.EndsAt.Sub(start) <= soonAfterStart && soon == nil:
			soon = f
		}
	}
	href := "/campaigns/" + campaignID.String() + "/edit"
	var out []ReadinessRow
	if blocked != nil {
		out = append(out, ReadinessRow{Key: IssueEnded, Status: "block", Title: keep("ready.ended", nil), Label: keep("ready.label", nil),
			Detail:    keep("ready.endedDetail", i18n.Args{"end": at(blocked.EndsAt.In(loc)), "start": at(start.In(loc)), "mode": blocked.Settings.Ended.Mode}),
			LinkLabel: keep("ready.open", nil), Href: href})
	}
	if soon != nil {
		out = append(out, ReadinessRow{Key: IssueSoon, Status: "warn", Title: keep("ready.soon", nil), Label: keep("ready.label", nil),
			Detail: keep("ready.soonDetail", i18n.Args{"end": at(soon.EndsAt.In(loc)), "start": at(start.In(loc)), "mode": soon.Settings.Ended.Mode}),
			Ack:    keep("ready.soonAck", nil), LinkLabel: keep("ready.open", nil), Href: href})
	}
	return out
}

// FlowWarnings are a flow's publish badges, in the request's language: one
// for each countdown of its emails that ends within 30 days, after which
// the flow keeps sending it.
func (s *Service) FlowWarnings(ctx context.Context, ws *store.Workspace, contents ...string) []string {
	now := s.now()
	l := i18n.LangFrom(ctx)
	var out []string
	seen := map[string]bool{}
	for _, f := range s.Find(ctx, ws, contents...) {
		if f.EndsAt.Sub(now) > flowWithin {
			continue
		}
		label := msgs.M("flow.ends").Text(l, i18n.Args{"date": l.Region.ShortDate(f.EndsAt.In(f.zone())), "mode": f.Settings.Ended.Mode})
		if !seen[label] {
			seen[label] = true
			out = append(out, label)
		}
	}
	return out
}
