package digests

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// GmailClip is the size from which Gmail clips an email (102 KB).
const GmailClip = 102 * 1024

// DigestRenderInput is what the composer draws: which posts preview the
// Feed items block, its settings as they are in the inspector, and the
// subject and preheader as typed.
type DigestRenderInput struct {
	// Preview: next (the next issue as it stands), last (the last sent
	// issue), one or six (the newest posts that pass the filters).
	Preview   string           `json:"preview" enum:"next,last,one,six"`
	Block     *DigestFeedBlock `json:"block,omitempty"`
	Subject   string           `json:"subject,omitempty" maxLength:"998"`
	Preheader string           `json:"preheader,omitempty" maxLength:"500"`
	// MJML is the template as on the canvas (else the saved one).
	MJML string `json:"mjml,omitempty" maxLength:"2000000"`
	// Sizes measures the issue with 1 to 8 posts (slower: it compiles each).
	Sizes bool `json:"sizes,omitempty"`
}

// DigestSize is the email's size with a number of posts.
type DigestSize struct {
	Posts int `json:"posts"`
	Bytes int `json:"bytes"`
}

// DigestBlockChecks are the checks of the Feed items block.
type DigestBlockChecks struct {
	// MaxBytes is the issue's size with the block's maximum of posts.
	MaxPosts int `json:"maxPosts"`
	MaxBytes int `json:"maxBytes"`
	Limit    int `json:"limit"`
	// Images counts the preview's images, AltFromTitle those whose alt text
	// is the post's title, Fallback the posts without an image.
	Images       int `json:"images"`
	AltFromTitle int `json:"altFromTitle"`
	Fallback     int `json:"fallback"`
	// Links counts the block's links (each post's title and button).
	Links int `json:"links"`
	// FitPosts is how many posts fit before Gmail clips.
	FitPosts int `json:"fitPosts"`
	// Bytes is the size of the issue the preview shows (the next issue as it stands).
	Bytes int `json:"bytes"`
}

// DigestRender is the Feed items block drawn with a preview's posts.
type DigestRender struct {
	Preview string `json:"preview"`
	// Number is the issue's number; Count how many posts the preview has.
	Number int `json:"number"`
	Count  int `json:"count"`
	// MJML is the block's sections for the canvas.
	MJML      string            `json:"mjml"`
	Items     []DigestFeedItem  `json:"items"`
	Subject   string            `json:"subject"`
	Preheader string            `json:"preheader"`
	Subjects  []DigestSubject   `json:"subjects"`
	Sizes     []DigestSize      `json:"sizes"`
	Checks    DigestBlockChecks `json:"checks"`
	// SendAt is the next issue's send time.
	SendAt *time.Time `json:"sendAt,omitempty"`
	// Tokens are the feed tokens' values for the next issue (the token menu).
	Tokens []DigestToken `json:"tokens"`
	// StarterSubject is the Blog digest starter's subject ("Use the starter's subject").
	StarterSubject string `json:"starterSubject"`
}

// DigestToken is a feed token with its value for the next issue.
type DigestToken struct {
	Token string `json:"token"`
	Value string `json:"value"`
}

// Render draws the Feed items block and the subject with a preview's
// posts, for the composer in digest mode.
func (s *Service) Render(ctx context.Context, ws *store.Workspace, id uuid.UUID, in DigestRenderInput) (DigestRender, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestRender{}, api.MapError(err, "Digest")
	}
	d := r.Digest
	tpl, err := s.template(ctx, d)
	if err != nil {
		return DigestRender{}, err
	}
	src := in.MJML
	subject, preheader := in.Subject, in.Preheader
	if tpl != nil {
		if src == "" {
			src = tpl.Mjml
		}
		if subject == "" {
			subject = tpl.Subject
		}
		if preheader == "" {
			preheader = tpl.Preheader
		}
	}
	block, _ := BlockOf(src)
	if in.Block != nil {
		block = in.Block.normalized()
	}
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return DigestRender{}, err
	}
	number, err := s.Q.DigestNextNumber(ctx, d.ID)
	if err != nil {
		return DigestRender{}, err
	}
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	out := DigestRender{Preview: in.Preview, Number: int(number), Items: []DigestFeedItem{}, Sizes: []DigestSize{}, Tokens: []DigestToken{}, StarterSubject: StarterSubject}
	// The preview's posts.
	var posts []item
	pool := []item{}
	for _, it := range all {
		if it.Status == "new" || it.Status == "sent" {
			pool = append(pool, it)
		}
	}
	switch in.Preview {
	case "last":
		if last, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil {
			out.Number = int(*last.Number)
			rows, err := s.Q.DigestIssueItems(ctx, last.ID)
			if err != nil {
				return DigestRender{}, err
			}
			for _, row := range rows {
				for _, it := range all {
					if it.ID == row.ID {
						posts = append(posts, it)
					}
				}
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return DigestRender{}, err
		}
	case "one":
		posts = pool[:min(1, len(pool))]
	case "six":
		posts = pool[:min(6, len(pool))]
	default:
		posts = newItems(all)
		if open, err := s.Q.DigestIssueOpen(ctx, d.ID); err == nil {
			out.SendAt = open.SendAt
			if open.Status != "collecting" && open.Status != "preparing" {
				rows, err := s.Q.DigestIssueItems(ctx, open.ID)
				if err != nil {
					return DigestRender{}, err
				}
				posts = posts[:0]
				for _, row := range rows {
					for _, it := range all {
						if it.ID == row.ID {
							posts = append(posts, it)
						}
					}
				}
			}
		}
	}
	bis := make([]BlockItem, 0, len(posts))
	for _, it := range posts {
		bis = append(bis, blockItem(it.DigestItemsRow))
	}
	shown := bis
	if in.Preview == "next" {
		shown, _ = block.Select(bis)
	} else if len(shown) > block.Max && in.Preview != "six" {
		shown = shown[:block.Max]
	}
	for i := range shown {
		shown[i] = block.withImage(shown[i])
		for _, it := range posts {
			if it.ID.String() == shown[i].ID {
				out.Items = append(out.Items, feedItem(it, block))
			}
		}
	}
	out.Count = len(shown)
	out.MJML = RenderBlock(block, shown, loc)
	data := IssueData{FeedTitle: d.FeedTitle, FeedURL: d.FeedUrl, Number: out.Number, Items: shown, Loc: loc}
	if out.SendAt != nil {
		data.Date = *out.SendAt
	}
	out.Subject, _ = RenderIssue(subject, data, false)
	out.Preheader, _ = RenderIssue(preheader, data, false)
	out.Subject, out.Preheader = strings.TrimSpace(out.Subject), strings.TrimSpace(out.Preheader)
	out.Subjects = subjects(subject, d, all, block, out.Number, loc)
	// Checks: images, alt text, links.
	for _, it := range shown {
		if it.Image != "" {
			out.Checks.Images++
		}
		if it.AltFromTitle {
			out.Checks.AltFromTitle++
		}
		if it.Fallback {
			out.Checks.Fallback++
		}
		if it.URL != "" {
			if block.has(FieldTitle) {
				out.Checks.Links++
			}
			if block.has(FieldButton) && block.Layout != "list" {
				out.Checks.Links++
			}
		}
	}
	// Sizes with 1 to 8 posts (Gmail clips at 102 KB).
	out.Checks.Limit, out.Checks.MaxPosts = GmailClip, block.Max
	if src != "" && strings.Contains(src, "<mj-body") {
		sample := []BlockItem{}
		for _, it := range pool {
			sample = append(sample, block.withImage(blockItem(it.DigestItemsRow)))
		}
		if len(sample) > 0 {
			// The size with the most posts the block sends is always measured;
			// the whole series only when asked.
			from, to := block.Max, block.Max
			if in.Sizes {
				from, to = 1, max(8, block.Max)
			}
			out.Checks.Bytes = issueSize(src, shown, data, loc)
			for n := from; n <= to; n++ {
				items := make([]BlockItem, 0, n)
				for i := 0; i < n; i++ {
					items = append(items, sample[i%len(sample)])
				}
				size := issueSize(src, items, data, loc)
				if in.Sizes {
					out.Sizes = append(out.Sizes, DigestSize{Posts: n, Bytes: size})
				}
				if n == block.Max {
					out.Checks.MaxBytes = size
				}
				if size <= GmailClip {
					out.Checks.FitPosts = n
				}
			}
		}
	}
	// The token menu's values for the next issue.
	nextData := data
	if in.Preview != "next" {
		nextPosts := newItems(all)
		items := make([]BlockItem, 0, len(nextPosts))
		for _, it := range nextPosts {
			items = append(items, blockItem(it.DigestItemsRow))
		}
		nextShown, _ := block.Select(items)
		nextData = IssueData{FeedTitle: d.FeedTitle, FeedURL: d.FeedUrl, Number: int(number), Items: nextShown, Loc: loc}
		if open, err := s.Q.DigestIssueOpen(ctx, d.ID); err == nil && open.SendAt != nil {
			nextData.Date = *open.SendAt
		}
	}
	for _, tok := range []string{"{{ feed.title }}", "{{ issue.number }}", "{{ issue.date }}", "{{ issue.items.size }}", "{{ issue.items.first.title }}"} {
		v, _ := RenderIssue(tok, nextData, false)
		out.Tokens = append(out.Tokens, DigestToken{Token: tok, Value: v})
	}
	return out, nil
}

// issueSize is the compiled size of a template with posts.
func issueSize(src string, items []BlockItem, data IssueData, loc *time.Location) int {
	data.Items = items
	expanded := ExpandBlock(src, items, loc)
	rendered, err := RenderIssue(expanded, data, true)
	if err != nil {
		rendered = expanded
	}
	res := mjml.Compile(rendered)
	return len(res.HTML)
}

// DigestPreview is the next issue as a reader would get it now.
type DigestPreview struct {
	Number    int    `json:"number"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	Posts     int    `json:"posts"`
	// Empty: there are no new posts yet; the preview uses the newest ones.
	Empty bool `json:"empty"`
}

// Preview renders the next issue as it would be built now, for a reader
// with no first name (fallbacks show). It changes nothing.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (DigestPreview, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestPreview{}, api.MapError(err, "Digest")
	}
	d := r.Digest
	tpl, err := s.template(ctx, d)
	if err != nil {
		return DigestPreview{}, err
	}
	if tpl == nil {
		return DigestPreview{}, api.Conflict(mIssueNoTemplate.Text(i18n.LangFrom(ctx), nil))
	}
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return DigestPreview{}, err
	}
	out := DigestPreview{}
	posts := []BlockItem{}
	for _, it := range newItems(all) {
		posts = append(posts, blockItem(it.DigestItemsRow))
	}
	if len(posts) == 0 {
		out.Empty = true
		for _, it := range all {
			if it.Status == "sent" || it.Status == "new" {
				posts = append(posts, blockItem(it.DigestItemsRow))
			}
		}
	}
	number, err := s.Q.DigestNextNumber(ctx, d.ID)
	if err != nil {
		return DigestPreview{}, err
	}
	var sendAt *time.Time
	if open, err := s.Q.DigestIssueOpen(ctx, d.ID); err == nil {
		sendAt = open.SendAt
	}
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	b, err := build(d, *tpl, posts, int(number), sendAt, loc)
	if err != nil {
		return DigestPreview{}, api.Invalid("body", mTemplateRender.Text(i18n.LangFrom(ctx), i18n.Args{"detail": err.Error()}))
	}
	links := map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"}
	vars := liquid.Vars("", "", "", nil, links)
	html, err := liquid.Render(b.HTML, vars)
	if err != nil {
		html = b.HTML
	}
	subject, _ := liquid.Render(b.Subject, vars)
	preheader, _ := liquid.Render(b.Preheader, vars)
	out.Number, out.Subject, out.Preheader, out.HTML, out.Posts = int(number), subject, preheader, html, len(b.Items)
	return out, nil
}
