package forms

// The public archive (spec 7.12): every sent campaign of the archived lists
// gets a page. Rows in archive_issues are created the first time the
// archive is read after a send (slugs are stable from then on) and carry
// the publisher's edits: a title or excerpt for the archive, the list an
// issue is filed under, an issue label ("#141", "special") and hiding.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// ArchivePageSize is how many issues an archive page lists.
const ArchivePageSize = 10

// ArchiveLists are the lists whose sent campaigns the archive publishes:
// the ones picked on the Archive tab, or every public list.
func (s *Service) ArchiveLists(ctx context.Context, q *store.Queries, ws *store.Workspace, set *Settings) ([]uuid.UUID, error) {
	lists, err := s.Audience.PublicLists(ctx, q, ws)
	if err != nil {
		return nil, err
	}
	picked := set.Pages.Archive.Lists
	out := []uuid.UUID{}
	for _, l := range lists {
		if len(picked) > 0 {
			if slices.Contains(picked, l.ID.String()) {
				out = append(out, l.ID)
			}
		} else if l.Public {
			out = append(out, l.ID)
		}
	}
	return out, nil
}

func idTexts(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

var issueNumber = regexp.MustCompile(`#(\d{1,6})\s*$`)

// IssueLabel is the label a new archive entry gets from its campaign's name:
// "Weekly Digest #141" is issue "#141".
func IssueLabel(name string) string {
	if m := issueNumber.FindStringSubmatch(name); m != nil {
		return "#" + m[1]
	}
	return ""
}

// SyncArchive files the sent campaigns of the archived lists that have no
// archive entry yet. It is cheap when nothing is missing, and serialized per
// workspace so two requests never hand out the same slug.
func (s *Service) SyncArchive(ctx context.Context, ws *store.Workspace, set *Settings) error {
	lists, err := s.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil || len(lists) == 0 {
		return err
	}
	missing, err := s.Q.ArchiveMissingIssues(ctx, store.ArchiveMissingIssuesParams{WorkspaceID: ws.ID, Lists: idTexts(lists)})
	if err != nil || len(missing) == 0 {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ArchiveLock(ctx, ws.ID); err != nil {
			return err
		}
		missing, err := q.ArchiveMissingIssues(ctx, store.ArchiveMissingIssuesParams{WorkspaceID: ws.ID, Lists: idTexts(lists)})
		if err != nil {
			return err
		}
		for _, m := range missing {
			title := RenderTitle(m.Subject)
			if title == "" {
				title = m.Name
			}
			slug, err := s.uniqueIssueSlug(ctx, q, ws, title)
			if err != nil {
				return err
			}
			var aud struct {
				Lists []uuid.UUID `json:"lists"`
			}
			_ = json.Unmarshal(m.Audience, &aud)
			var list *uuid.UUID
			for _, l := range aud.Lists {
				if slices.Contains(lists, l) {
					id := l
					list = &id
					break
				}
			}
			if err := q.ArchiveInsertIssue(ctx, store.ArchiveInsertIssueParams{CampaignID: m.ID, WorkspaceID: ws.ID, Slug: slug, ListID: list,
				Label: IssueLabel(m.Name), At: s.now()}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) uniqueIssueSlug(ctx context.Context, q *store.Queries, ws *store.Workspace, title string) (string, error) {
	base := Slugify(title, 80)
	if base == "" {
		base = "issue"
	}
	slug := base
	for i := 2; ; i++ {
		taken, err := q.ArchiveSlugTaken(ctx, store.ArchiveSlugTakenParams{WorkspaceID: ws.ID, Slug: slug})
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// RenderTitle renders a subject for a reader without a contact: Liquid
// tokens get their fallbacks ("{{ first_name | default: "there" }}").
func RenderTitle(subject string) string {
	out, err := liquid.Render(subject, liquid.Vars("", "", "", nil, nil))
	if err != nil {
		return strings.TrimSpace(subject)
	}
	return strings.Join(strings.Fields(out), " ")
}

// Excerpt shortens a plain-text body to about n characters at a word boundary.
func Excerpt(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	r := []rune(text)[:n]
	cut := string(r)
	if i := strings.LastIndexByte(cut, ' '); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:-–—") + "…"
}

// ArchiveEntry is an issue as the archive lists it.
type ArchiveEntry struct {
	CampaignID uuid.UUID
	Slug       string
	ListID     *uuid.UUID
	ListName   string
	// Title and Excerpt are the publisher's when set, else the subject and
	// the preheader (or the start of the text).
	Title, Excerpt, Label string
	Subject               string
	// TitleOverride and ExcerptOverride are what the publisher wrote.
	TitleOverride, ExcerptOverride string
	Hidden                         bool
	SentAt, UpdatedAt              time.Time
}

func issueItem(e ArchiveEntry, links PageLinks) ArchiveIssueItem {
	item := ArchiveIssueItem{CampaignID: e.CampaignID.String(), Slug: e.Slug, URL: links.Issue(e.Slug), ListName: e.ListName, Title: e.Title,
		Excerpt: e.Excerpt, Label: e.Label, Subject: e.Subject, Hidden: e.Hidden, SentAt: e.SentAt, TitleOverride: e.TitleOverride, ExcerptOverride: e.ExcerptOverride}
	if e.ListID != nil {
		item.ListID = e.ListID.String()
	}
	return item
}

func entryOf(r store.ArchiveIssuesRow) ArchiveEntry {
	e := ArchiveEntry{CampaignID: r.CampaignID, Slug: r.Slug, ListID: r.ListID, ListName: r.ListName, Label: r.Label, Hidden: r.Hidden,
		Subject: RenderTitle(r.Subject), TitleOverride: r.Title, ExcerptOverride: r.Excerpt, SentAt: r.SentAt, UpdatedAt: r.UpdatedAt}
	e.Title = strings.TrimSpace(r.Title)
	if e.Title == "" {
		e.Title = e.Subject
	}
	if e.Title == "" {
		e.Title = r.Name
	}
	e.Excerpt = strings.TrimSpace(r.Excerpt)
	if e.Excerpt == "" {
		e.Excerpt = RenderTitle(r.Preheader)
	}
	if e.Excerpt == "" {
		e.Excerpt = Excerpt(r.PlainText, 180)
	}
	return e
}

// ArchiveQuery filters the archive.
type ArchiveQuery struct {
	Search string
	List   *uuid.UUID
	// Page counts from 1.
	Page          int
	PageSize      int
	IncludeHidden bool
}

// likePattern escapes a search for ILIKE.
func likePattern(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// Archive lists the archive's issues, newest first, with the total.
func (s *Service) Archive(ctx context.Context, ws *store.Workspace, set *Settings, aq ArchiveQuery) ([]ArchiveEntry, int64, error) {
	if err := s.SyncArchive(ctx, ws, set); err != nil {
		return nil, 0, err
	}
	lists, err := s.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil {
		return nil, 0, err
	}
	if aq.PageSize <= 0 {
		aq.PageSize = ArchivePageSize
	}
	if aq.Page < 1 {
		aq.Page = 1
	}
	pattern := likePattern(aq.Search)
	total, err := s.Q.ArchiveIssueCount(ctx, store.ArchiveIssueCountParams{WorkspaceID: ws.ID, Lists: lists, IncludeHidden: aq.IncludeHidden, ListID: aq.List, Q: pattern})
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.Q.ArchiveIssues(ctx, store.ArchiveIssuesParams{WorkspaceID: ws.ID, Lists: lists, IncludeHidden: aq.IncludeHidden, ListID: aq.List, Q: pattern,
		Off: int32(min((aq.Page-1)*aq.PageSize, 1<<30)), Lim: int32(aq.PageSize)})
	if err != nil {
		return nil, 0, err
	}
	out := make([]ArchiveEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entryOf(r))
	}
	return out, total, nil
}

// ArchiveIssueItem is an issue on the Archive tab.
type ArchiveIssueItem struct {
	CampaignID string    `json:"campaignId"`
	Slug       string    `json:"slug"`
	URL        string    `json:"url"`
	ListID     string    `json:"listId"`
	ListName   string    `json:"listName"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	Label      string    `json:"label"`
	Subject    string    `json:"subject"`
	Hidden     bool      `json:"hidden"`
	SentAt     time.Time `json:"sentAt"`
	// TitleOverride and ExcerptOverride are what the publisher wrote; empty
	// uses the subject and the preheader.
	TitleOverride   string `json:"titleOverride"`
	ExcerptOverride string `json:"excerptOverride"`
}

// ArchiveListCount is how many published issues a list has.
type ArchiveListCount struct {
	ListID string `json:"listId"`
	Name   string `json:"name"`
	Issues int64  `json:"issues"`
}

// ArchiveIssuePage is a page of the Archive tab's issues.
type ArchiveIssuePage struct {
	Issues   []ArchiveIssueItem `json:"issues"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	// Published counts the issues readers see; Lists breaks them down.
	Published int64              `json:"published"`
	Lists     []ArchiveListCount `json:"lists"`
}

// ArchiveIssues lists the archive for the Archive tab, hidden issues included.
func (s *Service) ArchiveIssues(ctx context.Context, ws *store.Workspace, search, list string, page int) (ArchiveIssuePage, error) {
	set, err := s.settings(ctx, ws)
	if err != nil {
		return ArchiveIssuePage{}, err
	}
	aq := ArchiveQuery{Search: search, Page: page, PageSize: 10, IncludeHidden: true}
	if list != "" {
		id, err := uuid.Parse(list)
		if err != nil {
			return ArchiveIssuePage{}, api.Invalid("list", "Pick a list of this workspace.")
		}
		aq.List = &id
	}
	entries, total, err := s.Archive(ctx, ws, set, aq)
	if err != nil {
		return ArchiveIssuePage{}, err
	}
	links := s.Links(ws, set)
	out := ArchiveIssuePage{Issues: []ArchiveIssueItem{}, Total: total, Page: max(page, 1), PageSize: aq.PageSize, Lists: []ArchiveListCount{}}
	for _, e := range entries {
		out.Issues = append(out.Issues, issueItem(e, links))
	}
	lists, err := s.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil {
		return out, err
	}
	counts, err := s.Q.ArchiveListCounts(ctx, store.ArchiveListCountsParams{WorkspaceID: ws.ID, Lists: lists})
	if err != nil {
		return out, err
	}
	all, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	names := map[uuid.UUID]string{}
	for _, l := range all {
		names[l.ID] = l.Name
	}
	byList := map[uuid.UUID]int64{}
	for _, c := range counts {
		byList[c.ListID] = c.Issues
		out.Published += c.Issues
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, ArchiveListCount{ListID: l.String(), Name: names[l], Issues: byList[l]})
	}
	return out, nil
}

// ArchiveIssueInput edits an issue on the Archive tab.
type ArchiveIssueInput struct {
	Hidden  bool   `json:"hidden" doc:"Hide the issue from the archive, the feed and the sitemap"`
	Title   string `json:"title" maxLength:"200" doc:"Empty uses the subject"`
	Excerpt string `json:"excerpt" maxLength:"500" doc:"Empty uses the preheader"`
	Label   string `json:"label" maxLength:"40" example:"#141"`
	ListID  string `json:"listId" doc:"The list the issue is filed under"`
}

// UpdateArchiveIssue saves an issue's archive settings.
func (s *Service) UpdateArchiveIssue(ctx context.Context, ws *store.Workspace, campaignID string, in ArchiveIssueInput) (ArchiveIssueItem, error) {
	cid, err := uuid.Parse(campaignID)
	if err != nil {
		return ArchiveIssueItem{}, api.NotFound("Archive issue")
	}
	set, err := s.settings(ctx, ws)
	if err != nil {
		return ArchiveIssueItem{}, err
	}
	lists, err := s.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil {
		return ArchiveIssueItem{}, err
	}
	list, err := uuid.Parse(in.ListID)
	if err != nil || !slices.Contains(lists, list) {
		return ArchiveIssueItem{}, api.Invalid("body.listId", "File the issue under one of the archived lists.")
	}
	var after store.ArchiveIssue
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.ArchiveIssueGet(ctx, store.ArchiveIssueGetParams{WorkspaceID: ws.ID, CampaignID: cid})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Archive issue")
		}
		if err != nil {
			return err
		}
		after, err = q.ArchiveUpdateIssue(ctx, store.ArchiveUpdateIssueParams{WorkspaceID: ws.ID, CampaignID: cid, Hidden: in.Hidden,
			Title: strings.TrimSpace(in.Title), Excerpt: strings.TrimSpace(in.Excerpt), Label: strings.TrimSpace(in.Label), ListID: &list,
			UpdatedBy: actor(ctx), At: s.now()})
		if err != nil {
			return err
		}
		snap := func(a store.ArchiveIssue) map[string]any {
			m := map[string]any{"hidden": a.Hidden, "title": a.Title, "excerpt": a.Excerpt, "label": a.Label}
			if a.ListID != nil {
				m["listId"] = a.ListID.String()
			}
			return m
		}
		action := "archive.update"
		if before.Hidden != after.Hidden {
			action = "archive.show"
			if after.Hidden {
				action = "archive.hide"
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: cid.String(), TargetName: after.Slug,
			Before: snap(before), After: snap(after)})
	})
	if err != nil {
		return ArchiveIssueItem{}, err
	}
	row, err := s.Q.ArchiveIssueOne(ctx, store.ArchiveIssueOneParams{WorkspaceID: ws.ID, CampaignID: cid})
	if err != nil {
		return ArchiveIssueItem{}, api.MapError(err, "Archive issue")
	}
	return issueItem(entryOf(store.ArchiveIssuesRow(row)), s.Links(ws, set)), nil
}
