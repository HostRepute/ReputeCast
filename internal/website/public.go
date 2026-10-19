package website

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// PublicPost is a published post as the site lists it: what an anonymous
// visitor may see of it (never its gated text).
type PublicPost struct {
	ID          uuid.UUID
	Kind        string
	CampaignID  *uuid.UUID
	Slug        string
	Title       string
	Excerpt     string
	CoverURL    string
	CoverAlt    string
	PublishedAt time.Time
	UpdatedAt   time.Time
	Visibility  gate.Visibility
	// Tags are the post's tags in order; the first names it in lists.
	Tags []TagRef
	// Label is the visibility label: "Members", or a tier's ("Insider").
	Label     PostLabel
	Words     int
	HideIndex bool
	// PreviewText is the text before the free preview marker (search).
	PreviewText string
	BodyText    string
	Byline      string
	BylineColor string
}

// TagRef is a tag of a post.
type TagRef struct {
	ID   uuid.UUID
	Name string
	Slug string
}

// PostLabel is a post's visibility label on the site: Kind is "members" or
// "paid" ("" for public posts), Text says it ("Members", "Insider").
type PostLabel struct {
	Kind string
	Text string
}

// PublicQuery filters the site's lists of posts.
type PublicQuery struct {
	Tag     *uuid.UUID
	Search  string
	Exclude []uuid.UUID
	Limit   int
	Offset  int
	// IDs lists exactly these posts, in this order (featured posts).
	IDs []uuid.UUID
}

const publicColumns = `p.id, p.kind, p.campaign_id, p.slug, p.title, p.excerpt, p.cover_url, p.cover_alt,
	coalesce(p.published_at, p.created_at), p.updated_at, p.visibility, p.tiers, p.never_free, p.words, p.hide_from_search,
	p.preview_text, p.body_text, coalesce(nullif(p.byline_name, ''), u.name, ''), coalesce(u.avatar_color, '')`

// published lists published posts, newest first, with the total.
func (s *Service) published(ctx context.Context, q *store.Queries, ws *store.Workspace, f PublicQuery) ([]PublicPost, int64, error) {
	where := []string{"p.workspace_id = $1", "p.deleted_at IS NULL", "p.status = 'published'"}
	args := []any{ws.ID}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	if f.Tag != nil {
		add("EXISTS (SELECT 1 FROM website_post_tags t WHERE t.post_id = p.id AND t.tag_id = ?)", *f.Tag)
	}
	if len(f.Exclude) > 0 {
		add("NOT (p.id = ANY (?))", f.Exclude)
	}
	if len(f.IDs) > 0 {
		add("p.id = ANY (?)", f.IDs)
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		// Search reads what a visitor may read: the title, the excerpt and
		// the text before the free preview marker (all of a public post).
		like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
		add(`(p.title ILIKE ? OR p.excerpt ILIKE ? OR p.preview_text ILIKE ? OR (p.visibility = 'public' AND p.body_text ILIKE ?))`, like)
	}
	cond := strings.Join(where, " AND ")
	var total int64
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM website_posts p WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 10
	}
	order := "coalesce(p.published_at, p.created_at) DESC, p.id DESC"
	if len(f.IDs) > 0 {
		order = "array_position($" + fmt.Sprint(len(args)+1) + "::uuid[], p.id)"
		args = append(args, f.IDs)
	}
	args = append(args, limit, f.Offset)
	rows, err := s.Pool.Query(ctx, `SELECT `+publicColumns+` FROM website_posts p LEFT JOIN users u ON u.id = p.byline_user_id
		WHERE `+cond+` ORDER BY `+order+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	out, err := scanPublic(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := s.withTags(ctx, q, out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func scanPublic(rows pgx.Rows) ([]PublicPost, error) {
	defer rows.Close()
	out := []PublicPost{}
	for rows.Next() {
		var p PublicPost
		var level string
		var tiers []uuid.UUID
		var neverFree bool
		var words int32
		if err := rows.Scan(&p.ID, &p.Kind, &p.CampaignID, &p.Slug, &p.Title, &p.Excerpt, &p.CoverURL, &p.CoverAlt, &p.PublishedAt, &p.UpdatedAt,
			&level, &tiers, &neverFree, &words, &p.HideIndex, &p.PreviewText, &p.BodyText, &p.Byline, &p.BylineColor); err != nil {
			return nil, err
		}
		if tiers == nil {
			tiers = []uuid.UUID{}
		}
		p.Visibility = gate.Visibility{Level: level, Tiers: tiers, NeverFree: neverFree}
		p.Words = int(words)
		out = append(out, p)
	}
	return out, rows.Err()
}

// withTags fills the tags of posts.
func (s *Service) withTags(ctx context.Context, q *store.Queries, posts []PublicPost) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
	}
	rows, err := q.WebsitePostTags(ctx, ids)
	if err != nil {
		return err
	}
	by := map[uuid.UUID][]TagRef{}
	for _, r := range rows {
		by[r.PostID] = append(by[r.PostID], TagRef{ID: r.ID, Name: r.Name, Slug: r.Slug})
	}
	for i := range posts {
		posts[i].Tags = by[posts[i].ID]
		if posts[i].Tags == nil {
			posts[i].Tags = []TagRef{}
		}
	}
	return nil
}

// labelOf is a post's visibility label: "Members" for members-only posts,
// the label of the lowest tier that reads a paid post ("Insider").
func labelOf(v gate.Visibility, tiers []gate.Tier, w Words) PostLabel {
	switch v.Level {
	case gate.Members:
		return PostLabel{Kind: gate.Members, Text: w.Text("site.label.members", nil)}
	case gate.Paid:
		readers := gate.Readers(v, tiers)
		if len(readers) > 0 {
			return PostLabel{Kind: gate.Paid, Text: readers[0].Label}
		}
		return PostLabel{Kind: gate.Paid, Text: w.Text("site.label.paid", nil)}
	}
	return PostLabel{}
}

// IsOn reports whether a workspace's website is on.
func (s *Service) IsOn(ctx context.Context, ws *store.Workspace) (bool, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return false, err
	}
	return c.On(), nil
}
