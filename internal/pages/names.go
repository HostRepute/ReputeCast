package pages

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The website's header links name custom pages by their address; a link
// to a page that isn't published is left out of the header.
func init() {
	website.RegisterPageNames(pageNames)
}

func pageNames(ctx context.Context, q *store.Queries, wsID uuid.UUID, slugs []string) (map[string]string, error) {
	rows, err := q.PageTitlesBySlug(ctx, store.PageTitlesBySlugParams{WorkspaceID: wsID, Slugs: slugs})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Slug] = r.Title
	}
	return out, nil
}
