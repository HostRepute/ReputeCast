package website

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// WebsiteTurnOnInput turns the website on, with the home layout and the
// members' settings of the turn-on page.
type WebsiteTurnOnInput struct {
	Layout   string              `json:"layout" enum:"featured,list,grid"`
	SigninOn bool                `json:"signinOn"`
	Lists    []WebsiteSignupList `json:"lists,omitempty" maxItems:"50"`
}

// WebsiteTurnOffInput turns the website off: the site's address, typed.
type WebsiteTurnOffInput struct {
	Confirm string `json:"confirm" maxLength:"300"`
}

// TurnOnArgs runs turning a website on: tags from the archived lists, a
// post for every archived campaign, the redirects of their old addresses.
type TurnOnArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
}

// Kind implements river.JobArgs.
func (TurnOnArgs) Kind() string { return "website.turn_on" }

// InsertOpts runs one at a time per workspace.
func (TurnOnArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 10, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// TurnOn starts turning the website on. A certificate that fails stops it;
// a domain that isn't verified yet runs the site at the instance's address
// until it is.
func (s *Service) TurnOn(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteTurnOnInput) (WebsiteOverview, error) {
	addr := s.address(ctx, ws, true)
	if addr.Certificate.Status == "failing" {
		return WebsiteOverview{}, api.Err(http.StatusConflict, "certificate_failing", tr(ctx, "site.certificateFailing", map[string]any{"domain": addr.Domain}))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		if c.row.Status != StatusOff {
			return api.Conflict(tr(ctx, "site.alreadyOn", nil))
		}
		switch in.Layout {
		case LayoutFeatured, LayoutList, LayoutGrid:
			c.row.HomeLayout = in.Layout
		case "":
		default:
			return api.Invalid("body.layout", tr(ctx, "home.layoutInvalid", nil))
		}
		c.row.SigninOn = in.SigninOn
		if in.Lists != nil {
			c.SignupLists = in.Lists
		}
		if err := s.save(ctx, q, ws, c, userID(p)); err != nil {
			return err
		}
		now := s.now()
		progress, _ := json.Marshal(WebsiteProgress{StartedAt: &now, Pages: "waiting"})
		if err := q.WebsiteSetStatus(ctx, store.WebsiteSetStatusParams{WorkspaceID: ws.ID, Status: StatusTurningOn, Progress: progress, At: now, Actor: userID(p)}); err != nil {
			return err
		}
		facts, _ := s.archiveFacts(ctx, ws)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.turned_on", TargetType: "website", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"status": StatusOff}, After: map[string]any{"status": StatusTurningOn, "address": addr.Host, "layout": c.row.HomeLayout,
				"signin_on": c.row.SigninOn, "archived_campaigns": facts.Issues, "tags": facts.Tags}})
	})
	if err != nil {
		return WebsiteOverview{}, err
	}
	if j := s.jobs(); j != nil {
		if _, err := j.Insert(ctx, TurnOnArgs{WorkspaceID: ws.ID}, nil); err != nil {
			return WebsiteOverview{}, err
		}
	} else if err := s.RunTurnOn(ctx, ws.ID); err != nil {
		return WebsiteOverview{}, err
	}
	return s.Overview(ctx, ws, p)
}

// RunTurnOn makes the posts, tags and redirects of a website being turned
// on, then puts it on. It resumes where it stopped.
func (s *Service) RunTurnOn(ctx context.Context, wsID uuid.UUID) error {
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	c, err := s.load(ctx, s.Q, &ws)
	if err != nil {
		return err
	}
	if c.row.Status != StatusTurningOn {
		return nil
	}
	var pr WebsiteProgress
	_ = json.Unmarshal(c.row.Progress, &pr)
	set, err := s.Forms.LoadSettings(ctx, &ws)
	if err != nil {
		return err
	}
	if err := s.Forms.SyncArchive(ctx, &ws, set); err != nil {
		return err
	}
	lists, err := s.Forms.ArchiveLists(ctx, s.Q, &ws, set)
	if err != nil {
		return err
	}
	issues, err := s.Q.WebsiteArchiveCampaigns(ctx, store.WebsiteArchiveCampaignsParams{WorkspaceID: ws.ID, Lists: lists})
	if err != nil {
		return err
	}
	saveProgress := func() error {
		b, _ := json.Marshal(pr)
		return s.Q.WebsiteSetProgress(ctx, store.WebsiteSetProgressParams{WorkspaceID: ws.ID, Progress: b})
	}
	// 1. Tags: each list with archived campaigns.
	tagOf := map[uuid.UUID]uuid.UUID{}
	withIssues := []uuid.UUID{}
	for _, is := range issues {
		if is.ListID != nil && !slices.Contains(withIssues, *is.ListID) {
			withIssues = append(withIssues, *is.ListID)
		}
	}
	pr.Tags.Total, pr.Tags.Names = int64(len(withIssues)), []string{}
	names := map[uuid.UUID]string{}
	if pl, err := s.Audience.PublicLists(ctx, s.Q, &ws); err == nil {
		for _, l := range pl {
			names[l.ID] = l.Name
		}
	}
	for _, list := range withIssues {
		id, err := s.tagForList(ctx, &ws, list, names[list])
		if err != nil {
			return err
		}
		tagOf[list] = id
		pr.Tags.Names = append(pr.Tags.Names, names[list])
	}
	pr.Tags.Done = pr.Tags.Total
	pr.Posts.Total, pr.Redirects.Total = int64(len(issues)), int64(len(issues))
	if err := saveProgress(); err != nil {
		return err
	}
	// 2. Posts and 3. redirects, a batch at a time.
	pr.Posts.Done, pr.Redirects.Done = 0, 0
	for i, is := range issues {
		if err := s.postForIssue(ctx, &ws, is, tagOf); err != nil {
			return err
		}
		pr.Posts.Done++
		pr.Redirects.Done++
		if (i+1)%25 == 0 {
			if err := saveProgress(); err != nil {
				return err
			}
		}
	}
	now := s.now()
	pr.Pages, pr.FinishedAt = "done", &now
	b, _ := json.Marshal(pr)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.lock(ctx, q, &ws)
		if err != nil {
			return err
		}
		if cur.row.Status != StatusTurningOn {
			return nil
		}
		// The header gets the tags made from lists, up to its six links.
		if len(cur.Header) == 0 {
			for _, list := range withIssues {
				if len(cur.Header) >= MaxHeaderLinks {
					break
				}
				cur.Header = append(cur.Header, WebsiteNavLink{Kind: NavTag, Ref: tagOf[list].String()})
			}
			if err := s.save(ctx, q, &ws, cur, nil); err != nil {
				return err
			}
		}
		if err := q.WebsiteSetStatus(ctx, store.WebsiteSetStatusParams{WorkspaceID: ws.ID, Status: StatusOn, Progress: b, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.live", TargetType: "website", TargetID: ws.ID.String(), TargetName: ws.Name,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"}, Before: map[string]any{"status": StatusTurningOn},
			After: map[string]any{"status": StatusOn, "posts": pr.Posts.Done, "redirects": pr.Redirects.Done, "tags": pr.Tags.Names}})
	})
}

// tagForList is the tag made from a list (made once).
func (s *Service) tagForList(ctx context.Context, ws *store.Workspace, list uuid.UUID, name string) (uuid.UUID, error) {
	if t, err := s.Q.WebsiteTagByList(ctx, store.WebsiteTagByListParams{WorkspaceID: ws.ID, ListID: &list}); err == nil {
		return t.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if name == "" {
		name = "Newsletter"
	}
	slug := slugify(name)
	if slug == "" {
		slug = "newsletter"
	}
	base := slug
	for i := 2; ; i++ {
		taken, err := s.Q.WebsiteTagSlugTaken(ctx, store.WebsiteTagSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: uuid.Nil})
		if err != nil {
			return uuid.Nil, err
		}
		if !taken {
			break
		}
		slug = base + "-" + itoa(i)
	}
	id := newID()
	err := s.Q.WebsiteTagInsert(ctx, store.WebsiteTagInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Slug: slug, ListID: &list, At: s.now()})
	return id, err
}

func itoa(i int) string {
	b := []byte{}
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if len(b) == 0 {
		return "0"
	}
	return string(b)
}

// postForIssue makes the post of an archived campaign, published at its
// send, at the archive's address under /posts, with its old address
// redirecting. Nothing goes behind a paywall: every post is public.
func (s *Service) postForIssue(ctx context.Context, ws *store.Workspace, is store.WebsiteArchiveCampaignsRow, tagOf map[uuid.UUID]uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsitePostByCampaign(ctx, &is.CampaignID)
		if err == nil {
			// Made already (an earlier run, or Publish to web). A post
			// someone kept off the web stays off.
			if !cur.WebOn && cur.Status == PostDraft {
				return nil
			}
			if cur.Status != PostPublished && cur.Status != PostUnpublished {
				if err := s.publishNow(ctx, tx, ws, cur, nil, map[string]any{"name": "ReputeCast", "source": "turning the website on", "at": is.SentAt}, ""); err != nil {
					return err
				}
			}
			return q.WebsiteRedirectInsert(ctx, store.WebsiteRedirectInsertParams{WorkspaceID: ws.ID, Path: "archive/" + is.Slug, PostID: cur.ID, At: s.now()})
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		id := newID()
		title := strings.TrimSpace(is.Title)
		if title == "" {
			title = forms.RenderTitle(is.Subject)
		}
		if title == "" {
			title = is.Name
		}
		excerpt := strings.TrimSpace(is.Excerpt)
		if excerpt == "" {
			excerpt = forms.RenderTitle(is.Preheader)
		}
		if len([]rune(excerpt)) > 200 {
			excerpt = forms.Excerpt(excerpt, 195)
		}
		slug := is.Slug
		if !slugRe.MatchString(slug) {
			slug = slugify(slug)
		}
		if taken, err := q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: id}); err != nil {
			return err
		} else if taken || slug == "" {
			slug = s.freeSlug(ctx, ws, id, title)
		}
		var mjmlSrc string
		_ = tx.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, is.CampaignID).Scan(&mjmlSrc)
		cover, alt := firstImage(mjmlSrc)
		if err := q.WebsitePostInsert(ctx, store.WebsitePostInsertParams{ID: id, WorkspaceID: ws.ID, Kind: KindCampaign, CampaignID: &is.CampaignID, WebOn: true,
			Status: PostDraft, PublishMode: WhenWithSend, Title: title, TitleEdited: strings.TrimSpace(is.Title) != "", Slug: slug, Excerpt: excerpt,
			ExcerptEdited: strings.TrimSpace(is.Excerpt) != "", CoverUrl: cover, CoverAlt: alt, BylineName: "", Visibility: "public", Tiers: []uuid.UUID{},
			At: s.now()}); err != nil {
			return err
		}
		if is.ListID != nil {
			if tag, ok := tagOf[*is.ListID]; ok {
				if err := q.WebsitePostAddTag(ctx, store.WebsitePostAddTagParams{PostID: id, TagID: tag, Position: 0}); err != nil {
					return err
				}
			}
		}
		post, err := q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		return s.publishNow(ctx, tx, ws, post, nil, map[string]any{"name": "ReputeCast", "source": "turning the website on", "at": is.SentAt}, "")
	})
}

// firstImage is an email's first image and its alt text: a post's cover by
// default. A logo or an icon (a header's image, or one under 240 px wide)
// isn't a cover.
func firstImage(src string) (string, string) {
	root, err := mjml.Parse(src)
	if err != nil {
		return "", ""
	}
	body := mjml.Body(root)
	if body == nil {
		return "", ""
	}
	var url, alt string
	body.Walk(func(n *mjml.Node, _ []int) {
		if url == "" && n.Tag == "mj-image" && n.Marker != "header-logo" && !narrowImage(n.Attr("width")) {
			if src := n.Attr("src"); strings.HasPrefix(src, "http") || strings.HasPrefix(src, "/media/") {
				url, alt = src, n.Attr("alt")
			}
		}
	})
	return url, alt
}

// narrowImage reports an image width under 240 px ("96px"): a logo or an icon.
func narrowImage(width string) bool {
	w := strings.TrimSuffix(strings.TrimSpace(width), "px")
	n, err := strconv.Atoi(w)
	return err == nil && n > 0 && n < 240
}

// TurnOff turns the website off: the home page, web-only posts, sign-in,
// the account page and the paywall go offline; the archive comes back.
// Every post and setting is kept for next time.
func (s *Service) TurnOff(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WebsiteTurnOffInput) (WebsiteOverview, error) {
	addr := s.address(ctx, ws, false)
	want := addr.Host
	if addr.Domain != "" {
		want = addr.Domain
	}
	if !strings.EqualFold(strings.TrimSpace(in.Confirm), want) {
		return WebsiteOverview{}, api.Invalid("body.confirm", tr(ctx, "site.confirmMismatch", map[string]any{"address": want}))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.lock(ctx, q, ws)
		if err != nil {
			return err
		}
		if c.row.Status == StatusOff {
			return api.Conflict(tr(ctx, "site.alreadyOff", nil))
		}
		counts, err := s.counts(ctx, ws)
		if err != nil {
			return err
		}
		members, err := s.membersFacts(ctx, ws, c)
		if err != nil {
			return err
		}
		if err := q.WebsiteSetStatus(ctx, store.WebsiteSetStatusParams{WorkspaceID: ws.ID, Status: StatusOff, Progress: []byte("{}"), At: s.now(), Actor: userID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.turned_off", TargetType: "website", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"status": c.row.Status}, After: map[string]any{"status": StatusOff, "web_only_posts_offline": counts.WebOnly,
				"paid_members_kept": members.Paid}})
	})
	if err != nil {
		return WebsiteOverview{}, err
	}
	return s.Overview(ctx, ws, p)
}

// turnOnWorker runs TurnOnArgs.
type turnOnWorker struct {
	river.WorkerDefaults[TurnOnArgs]
	s *Service
}

func (w *turnOnWorker) Work(ctx context.Context, j *river.Job[TurnOnArgs]) error {
	return w.s.RunTurnOn(ctx, j.Args.WorkspaceID)
}

// Timeout gives a large archive time.
func (w *turnOnWorker) Timeout(*river.Job[TurnOnArgs]) time.Duration { return 30 * time.Minute }
