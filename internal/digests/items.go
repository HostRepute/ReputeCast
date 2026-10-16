package digests

import (
	"context"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// item is a feed item with what the digest does with it.
type item struct {
	store.DigestItemsRow
	Status    string // new, sent, left_out, before_start
	LeftOutBy string // category, tag, age
}

// when is the item's date: published, else first seen.
func (it item) when() time.Time {
	if it.PublishedAt != nil {
		return *it.PublishedAt
	}
	return it.FirstSeenAt
}

func hasAny(have, want []string) bool {
	for _, w := range want {
		for _, h := range have {
			if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(w)) {
				return true
			}
		}
	}
	return false
}

// classify decides each item's place: sent in an issue, left out by a
// filter, published before the digest was turned on, or new. firstIssue is
// true while no issue was sent yet (the "latest N" starting point counts
// then). Items are newest first.
func classify(d store.Digest, rows []store.DigestItemsRow, now time.Time, firstIssue bool) []item {
	start := now
	if d.FirstOnAt != nil {
		start = *d.FirstOnAt
	}
	out := make([]item, 0, len(rows))
	latest := 0
	for _, r := range rows {
		it := item{DigestItemsRow: r, Status: "new"}
		switch {
		case r.SentIn > 0:
			it.Status = "sent"
		case len(d.Categories) > 0 && !hasAny(r.Categories, d.Categories):
			it.Status, it.LeftOutBy = "left_out", "category"
		case len(d.Tags) > 0 && !hasAny(r.Tags, d.Tags):
			it.Status, it.LeftOutBy = "left_out", "tag"
		case it.when().Before(start):
			// The back catalogue: only the latest N go in a first issue that
			// asks for them (and are young enough).
			if d.StartMode == "latest" && firstIssue && latest < int(d.StartLatest) && (d.MaxAgeDays == nil || !it.when().Before(now.AddDate(0, 0, -int(*d.MaxAgeDays)))) {
				latest++
			} else {
				it.Status = "before_start"
			}
		case d.MaxAgeDays != nil && it.when().Before(now.AddDate(0, 0, -int(*d.MaxAgeDays))):
			it.Status, it.LeftOutBy = "left_out", "age"
		}
		out = append(out, it)
	}
	return out
}

// items loads and classifies a digest's items, newest first.
func (s *Service) items(ctx context.Context, d store.Digest, limit int, withGone bool) ([]item, error) {
	rows, err := s.Q.DigestItems(ctx, store.DigestItemsParams{DigestID: d.ID, WithGone: withGone, Lim: int32(limit)})
	if err != nil {
		return nil, err
	}
	first := true
	if _, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil {
		first = false
	}
	return classify(d, rows, s.now(), first), nil
}

// newItems are the items for the next issue, newest first.
func newItems(all []item) []item {
	out := []item{}
	for _, it := range all {
		if it.Status == "new" {
			out = append(out, it)
		}
	}
	return out
}

// blockItem is an item as the Feed items block shows it.
func blockItem(it store.DigestItemsRow) BlockItem {
	return BlockItem{ID: it.ID.String(), Title: it.Title, URL: it.Url, Author: it.Author, Published: it.PublishedAt, Categories: it.Categories,
		Excerpt: it.Excerpt, Image: it.MediaUrl, ImageAlt: it.ImageAlt}
}

func blockItemOf(it store.DigestItem) BlockItem {
	return BlockItem{ID: it.ID.String(), Title: it.Title, URL: it.Url, Author: it.Author, Published: it.PublishedAt, Categories: it.Categories,
		Excerpt: it.Excerpt, Image: it.MediaUrl, ImageAlt: it.ImageAlt}
}

// feedItem is an item for the screens.
func feedItem(it item, b DigestFeedBlock) DigestFeedItem {
	bi := b.withImage(blockItem(it.DigestItemsRow))
	out := DigestFeedItem{ID: it.ID.String(), Title: it.Title, URL: it.Url, Author: it.Author, Published: it.PublishedAt, Categories: nonNil(it.Categories),
		Tags: nonNil(it.Tags), Excerpt: it.Excerpt, Image: bi.Image, ImageAlt: bi.ImageAlt, AltFromTitle: bi.AltFromTitle, Status: it.Status,
		SentIn: int(it.SentIn), LeftOutBy: it.LeftOutBy, FirstSeenAt: it.FirstSeenAt, FirstSeenVia: it.FirstSeenVia}
	return out
}
