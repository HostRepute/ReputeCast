package reports

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linkrule"
	"github.com/hostrepute/reputecast/internal/store"
)

// Click maps are drawn on the email as the send captured it, with each
// tracked link marked with the position the sending engine gave it
// (internal/linkrule, review AN-5): the map uses the engine's rule for which
// links are tracked, so a bubble is on the link its clicks came from. An
// A/B test on content has a map per variant on the variant's own email; a
// test on the subject or the send time has a map per variant on the one
// email, with each variant's recipients' clicks.

// sentContent is campaign_sends.content (internal/sending/engine's
// sendContent): the email, each content variant's own HTML and the first
// link position of each ("" is the campaign's email, also variant A's).
type sentContent struct {
	HTML     string `json:"html"`
	Variants []struct {
		Key  string `json:"key"`
		HTML string `json:"html"`
	} `json:"variants"`
	Links map[string]int `json:"links"`
}

// own is the content key a variant renders: its own email, or "" (the
// campaign's).
func (c sentContent) own(variant string) (string, string) {
	if variant != "A" {
		for _, v := range c.Variants {
			if v.Key == variant && v.HTML != "" {
				return v.Key, v.HTML
			}
		}
	}
	return "", c.HTML
}

// span is the range of link positions of a content key: from its first to
// the next key's first (to = -1: to the end).
func (c sentContent) span(key string) (from, to int) {
	from, ok := c.Links[key]
	if !ok {
		return 0, -1
	}
	to = -1
	for _, off := range c.Links {
		if off > from && (to < 0 || off < to) {
			to = off
		}
	}
	return from, to
}

func within(pos, from, to int) bool { return pos >= from && (to < 0 || pos < to) }

// sent reads the captured content; a campaign without one (imported
// history, sample data) has the campaign's HTML.
func (s *Service) sent(ctx context.Context, c store.ReportCampaignRow) (sentContent, error) {
	out := sentContent{}
	raw, err := s.Q.ReportSendContent(ctx, c.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out.HTML == "" {
		out.HTML = c.Html
	}
	return out, nil
}

// clickMaps builds the campaign's click map (the campaign's email: variant
// A's for a test on content) and, with an A/B test, each variant's.
func (s *Service) clickMaps(ctx context.Context, l i18n.Lang, c store.ReportCampaignRow, ab *abConfig, links []store.ReportLinksRow) (ReportClickMap, error) {
	content, err := s.sent(ctx, c)
	if err != nil {
		return ReportClickMap{}, err
	}
	from, to := content.span("")
	lines := linkrule.Rows(content.HTML)
	out := clickMap(l, linkrule.Mark(content.HTML), local(links, from, to), lines)
	out.Variants = []ReportVariantClickMap{}
	if ab == nil || len(ab.Variants) < 2 {
		return out, nil
	}
	rows, err := s.Q.ReportLinkVariants(ctx, c.ID)
	if err != nil {
		return out, err
	}
	byVariant := map[string]map[int]store.ReportLinkVariantsRow{}
	for _, r := range rows {
		if byVariant[r.Variant] == nil {
			byVariant[r.Variant] = map[int]store.ReportLinkVariantsRow{}
		}
		byVariant[r.Variant][int(r.Position)] = r
	}
	maps := make([]ReportVariantClickMap, 0, len(ab.Variants))
	for _, v := range ab.Variants {
		key, html := content.own(v.Key)
		from, to := content.span(key)
		counts, kept := byVariant[v.Key]
		if !kept && key == "" {
			// A test on the subject or send time whose clicks weren't
			// kept per variant: no maps per variant.
			return out, nil
		}
		mine := make([]store.ReportLinksRow, 0, len(links))
		for _, r := range links {
			if !within(int(r.Position), from, to) {
				continue
			}
			if kept {
				// The variant's recipients' clicks.
				n := counts[int(r.Position)]
				r.Clicks, r.UniqueClicks, r.BotClicks = n.Clicks, n.UniqueClicks, n.BotClicks
			}
			// Else the variant's own email: its links' clicks are all its
			// recipients'.
			r.Position -= int32(from)
			mine = append(mine, r)
		}
		own := lines
		if key != "" {
			own = linkrule.Rows(html)
		}
		m := clickMap(l, "", mine, own)
		vm := ReportVariantClickMap{Key: v.Key, SameEmail: key == "", Available: out.Available, Total: m.Total, BotTotal: m.BotTotal, Links: m.Links, Rows: m.Rows,
			Other: m.Other}
		if key != "" {
			vm.HTML = linkrule.Mark(html)
			vm.Available = html != ""
		}
		maps = append(maps, vm)
	}
	sort.SliceStable(maps, func(i, j int) bool { return maps[i].Key < maps[j].Key })
	out.Variants = maps
	return out, nil
}

// local keeps the links of one content's span, numbered from its first.
func local(links []store.ReportLinksRow, from, to int) []store.ReportLinksRow {
	if from == 0 && to < 0 {
		return links
	}
	out := make([]store.ReportLinksRow, 0, len(links))
	for _, r := range links {
		if within(int(r.Position), from, to) {
			r.Position -= int32(from)
			out = append(out, r)
		}
	}
	return out
}
