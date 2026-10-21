package pages

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Publish checks (the builder's Checks tab and the publish dialog). They
// run on every change; warnings don't stop publishing, blocking checks
// do. The web app words them from their key and values.

// Check keys.
const (
	CheckAddress     = "address"      // the address is free (or the page's own)
	CheckHeading     = "heading"      // a heading opens the page
	CheckMoments     = "moments"      // no countdown or event in the past
	CheckProduct     = "product"      // a product block's product: picked and published
	CheckStripe      = "stripe"       // Buy buttons need a payment connection
	CheckForm        = "form"         // a signup block's form: picked and live
	CheckImages      = "images"       // images have alt text
	CheckLinks       = "links"        // links answer
	CheckVisibility  = "visibility"   // Members and Paid need the website
	CheckStoreBlocks = "store_blocks" // store products need a connected store (phase 7, X20)
)

// Check statuses.
const (
	CheckPass  = "pass"
	CheckWarn  = "warn"
	CheckBlock = "block"
)

// PageCheck is one publish check.
type PageCheck struct {
	Key    string `json:"key" enum:"address,heading,moments,product,stripe,form,images,links,visibility,store_blocks"`
	Status string `json:"status" enum:"pass,warn,block"`
	// Path is the address (/field-guide).
	Path string `json:"path,omitempty"`
	// Count is how many blocks, images or links it's about.
	Count int `json:"count,omitempty"`
	// Ref is what the check names: a product's or form's id, an image's
	// address, a link's; Name and Detail say it in words.
	Ref    string `json:"ref,omitempty"`
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`
	// Problem is a product's or form's state: missing, draft, archived,
	// paused, deleted; a link's HTTP status.
	Problem    string `json:"problem,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	// PWYW: the price is a pay-what-you-want minimum.
	PWYW bool       `json:"pwyw,omitempty"`
	By   string     `json:"by,omitempty"`
	At   *time.Time `json:"at,omitempty"`
	// Account and Mode are the payment connection's (Acme Media, live).
	Account string `json:"account,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

// checks runs a page's publish checks.
func (s *Service) checks(ctx context.Context, ws *store.Workspace, pg store.Page) ([]PageCheck, PageCounts, error) {
	u := Analyze(pg.Html)
	counts := PageCounts{Blocks: u.Blocks, Images: u.Images, ImagesWithAlt: u.Images - len(u.ImagesNoAlt), Links: len(u.Links)}
	var out []PageCheck
	add := func(c PageCheck) { out = append(out, c) }
	now := s.now()

	// The address.
	taken, err := s.Q.PageSlugTaken(ctx, store.PageSlugTakenParams{WorkspaceID: ws.ID, ID: pg.ID, Slug: pg.Slug})
	if err != nil {
		return nil, counts, err
	}
	addr := PageCheck{Key: CheckAddress, Status: CheckPass, Path: "/" + pg.Slug}
	if pg.LiveSlug != nil && *pg.LiveSlug == pg.Slug {
		addr.Detail = "yours"
	} else {
		addr.Detail = "free"
	}
	if taken || Reserved(pg.Slug) {
		addr.Status, addr.Detail = CheckBlock, "taken"
	}
	add(addr)

	// Visibility.
	if pg.Visibility != VisibilityPublic && s.Website != nil {
		if on, _ := s.Website.IsOn(ctx, ws); !on {
			add(PageCheck{Key: CheckVisibility, Status: CheckBlock, Detail: pg.Visibility})
		}
	}

	// Products and the payment connection.
	if len(u.Products) > 0 || u.BuyButtons > 0 {
		rows, err := s.Q.ProductsByIDs(ctx, store.ProductsByIDsParams{WorkspaceID: ws.ID, Ids: u.Products})
		if err != nil {
			return nil, counts, err
		}
		byID := map[uuid.UUID]store.Product{}
		for _, r := range rows {
			byID[r.ID] = r
		}
		if u.EmptyProducts > 0 {
			add(PageCheck{Key: CheckProduct, Status: CheckBlock, Problem: "missing", Count: u.EmptyProducts})
		}
		for _, id := range u.Products {
			p, ok := byID[id]
			c := PageCheck{Key: CheckProduct, Status: CheckPass, Ref: id.String()}
			switch {
			case !ok:
				c.Status, c.Problem = CheckBlock, "deleted"
			case p.Status == "draft":
				c.Status, c.Problem, c.Name = CheckBlock, "draft", p.Name
			case p.Status == "archived":
				c.Status, c.Problem, c.Name = CheckBlock, "archived", p.Name
			default:
				c.Name = liveName(p)
				c.Detail, c.PWYW = livePrice(p)
			}
			add(c)
		}
		st := PageCheck{Key: CheckStripe, Status: CheckBlock, Count: u.BuyButtons}
		if s.Selling != nil {
			sel, ok, err := s.Selling(ctx, ws)
			if err != nil {
				return nil, counts, err
			}
			if ok {
				st.Status, st.Account, st.Mode = CheckPass, sel.Account, sel.Mode
			}
		}
		add(st)
	}
	if u.Stores > 0 {
		add(PageCheck{Key: CheckStoreBlocks, Status: CheckWarn, Count: u.Stores})
	}

	// Signup forms.
	if len(u.Forms) > 0 || u.EmptyForms > 0 {
		if u.EmptyForms > 0 {
			add(PageCheck{Key: CheckForm, Status: CheckBlock, Problem: "missing", Count: u.EmptyForms})
		}
		for _, id := range u.Forms {
			c := PageCheck{Key: CheckForm, Status: CheckPass, Ref: id.String()}
			var name, status string
			var pausedAt *time.Time
			var pausedBy string
			err := s.Pool.QueryRow(ctx, `SELECT f.name, f.status, f.updated_at, coalesce(u.name, '')
				FROM forms f LEFT JOIN users u ON u.id = f.updated_by
				WHERE f.workspace_id = $1 AND f.id = $2`, ws.ID, id).Scan(&name, &status, &pausedAt, &pausedBy)
			switch {
			case err != nil:
				c.Status, c.Problem = CheckBlock, "deleted"
			case status == "paused":
				c.Status, c.Problem, c.Name, c.At, c.By = CheckBlock, "paused", name, pausedAt, pausedBy
			case status != "live":
				// A draft form shows nothing on the site.
				c.Status, c.Problem, c.Name = CheckBlock, status, name
			default:
				c.Name = name
			}
			add(c)
		}
	}

	// Countdowns and events in the past. A countdown that ended on a page
	// already published did its job (a launch); on a page not yet
	// published it's a mistake.
	past := PageCheck{Key: CheckMoments, Status: CheckPass}
	for _, e := range u.Events {
		if e.At.Before(now) {
			past.Status, past.Name, past.At, past.Detail = CheckWarn, e.Label, timePtr(e.At), "event"
		}
	}
	for _, c := range u.Countdowns {
		if c.At.Before(now) && pg.FirstPublishedAt == nil {
			past.Status, past.Name, past.At, past.Detail = CheckWarn, c.Label, timePtr(c.At), "countdown"
		}
	}
	add(past)

	// Accessibility: images and the first heading.
	if len(u.ImagesNoAlt) > 0 {
		for _, src := range u.ImagesNoAlt {
			add(PageCheck{Key: CheckImages, Status: CheckWarn, Ref: src, Name: fileName(src)})
		}
	} else if u.Images > 0 {
		add(PageCheck{Key: CheckImages, Status: CheckPass, Count: u.Images})
	}
	h := PageCheck{Key: CheckHeading, Status: CheckPass}
	if u.FirstHeading != 1 {
		h.Status = CheckWarn
	}
	add(h)

	// Links: on the site, or answering outside it.
	if len(u.Links) > 0 {
		bad := s.brokenLinks(ctx, u.Links)
		for _, l := range bad {
			add(PageCheck{Key: CheckLinks, Status: CheckWarn, Ref: l.link.Href, Name: l.link.Text, HTTPStatus: l.status})
		}
		if len(bad) == 0 {
			add(PageCheck{Key: CheckLinks, Status: CheckPass, Count: len(u.Links)})
		}
	}
	// Blocking first, then warnings, then the passed ones (the tab's order).
	rank := map[string]int{CheckBlock: 0, CheckWarn: 1, CheckPass: 2}
	slices.SortStableFunc(out, func(a, b PageCheck) int { return rank[a.Status] - rank[b.Status] })
	return out, counts, nil
}

func timePtr(t time.Time) *time.Time { return &t }

// fileName is an image's file name ("beta-dashboard.png").
func fileName(src string) string {
	if u, err := url.Parse(src); err == nil {
		src = u.Path
	}
	if i := strings.LastIndexByte(src, '/'); i >= 0 {
		src = src[i+1:]
	}
	return src
}

type badLink struct {
	link   Link
	status int
}

// linkCache keeps link answers for ten minutes, so checks on every change
// don't fetch every link every time.
var linkCache = struct {
	sync.Mutex
	m map[string]linkAnswer
}{m: map[string]linkAnswer{}}

type linkAnswer struct {
	status int
	at     time.Time
}

// brokenLinks are the outside links that answer 4xx or 5xx (or not at all).
func (s *Service) brokenLinks(ctx context.Context, links []Link) []badLink {
	var out []badLink
	if s.Probe == nil {
		return out
	}
	seen := map[string]bool{}
	for _, l := range links {
		href := strings.TrimSpace(l.Href)
		if !strings.HasPrefix(href, "https://") && !strings.HasPrefix(href, "http://") {
			continue
		}
		if seen[href] || len(seen) >= 30 {
			continue
		}
		seen[href] = true
		linkCache.Lock()
		a, ok := linkCache.m[href]
		linkCache.Unlock()
		if !ok || time.Since(a.at) > 10*time.Minute {
			status, err := s.Probe(ctx, href)
			if err != nil {
				status = 0
			}
			a = linkAnswer{status: status, at: time.Now()}
			linkCache.Lock()
			if len(linkCache.m) > 5000 {
				linkCache.m = map[string]linkAnswer{}
			}
			linkCache.m[href] = a
			linkCache.Unlock()
		}
		if a.status == 0 || a.status >= 400 {
			out = append(out, badLink{link: l, status: a.status})
		}
	}
	return out
}

// liveName is a product's published name.
func liveName(p store.Product) string {
	var l struct {
		Name string `json:"name"`
	}
	if len(p.Live) > 0 && jsonInto(p.Live, &l) == nil && l.Name != "" {
		return l.Name
	}
	return p.Name
}

// livePrice is a published product's price ("$29"); pwyw: it's the
// pay-what-you-want minimum.
func livePrice(p store.Product) (string, bool) {
	var l struct {
		PriceKind  string `json:"priceKind"`
		PriceCents int64  `json:"priceCents"`
		MinCents   int64  `json:"minCents"`
	}
	if len(p.Live) == 0 || jsonInto(p.Live, &l) != nil {
		return "", false
	}
	if l.PriceKind == "pwyw" {
		return Money(l.MinCents, p.Currency), true
	}
	return Money(l.PriceCents, p.Currency), false
}
