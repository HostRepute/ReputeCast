package commerce

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// IssueProductRemoved is the content check of a store block that names a
// product the store removed: the composer flags it, Send Readiness warns,
// and sends leave the product out.
const IssueProductRemoved = "store.product_removed"

// RemovedProducts is a contentcheck.Extra: a store block's products the
// store removed since the block was made.
func (s *Service) RemovedProducts(ctx context.Context, in contentcheck.Input) []contentcheck.ContentCheckIssue {
	if in.WorkspaceID == uuid.Nil || !strings.Contains(in.MJML+in.HTML, itemsOpen) {
		return nil
	}
	type found struct {
		block   CommerceBlock
		section int
		path    []int
	}
	var blocks []found
	parse := func(text string, section int, path []int) {
		for _, m := range storeItemsRe.FindAllStringSubmatch(text, -1) {
			var b CommerceBlock
			if json.Unmarshal([]byte(m[1]), &b) == nil {
				blocks = append(blocks, found{block: b, section: section, path: path})
			}
		}
	}
	if root, err := mjml.Parse(in.MJML); err == nil && mjml.Body(root) != nil {
		for i, n := range mjml.Body(root).Children {
			if n.Start >= 0 && n.End <= len(in.MJML) && n.Start < n.End {
				parse(in.MJML[n.Start:n.End], i, []int{i})
			}
		}
	} else {
		parse(in.HTML, -1, nil)
	}
	if len(blocks) == 0 {
		return nil
	}
	loc := location("")
	if ws, err := s.Q.GetWorkspace(ctx, in.WorkspaceID); err == nil {
		loc = location(ws.Timezone)
	}
	var out []contentcheck.ContentCheckIssue
	seen := map[string]bool{}
	stores := map[string]*store.Store{}
	for _, f := range blocks {
		if f.block.Mode == "collection" || len(f.block.Products) == 0 {
			continue
		}
		st, ok := stores[f.block.Store]
		if !ok {
			stores[f.block.Store] = nil
			if id, err := uuid.Parse(f.block.Store); err == nil {
				if row, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: id, WorkspaceID: in.WorkspaceID}); err == nil {
					stores[f.block.Store] = &row
				}
			}
			st = stores[f.block.Store]
		}
		if st == nil {
			continue
		}
		ids := f.block.Products
		if f.block.Mode == "card" && len(ids) > 1 {
			ids = ids[:1]
		}
		rows, err := s.Q.CommerceProductsByExternal(ctx, store.CommerceProductsByExternalParams{StoreID: st.ID, Ids: ids})
		if err != nil {
			continue
		}
		for _, r := range rows {
			if r.RemovedAt == nil || !slices.Contains(ids, r.ExternalID) {
				continue
			}
			key := fmt.Sprintf("%d:%s", f.section, r.ExternalID)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, contentcheck.ContentCheckIssue{ID: IssueProductRemoved, Severity: contentcheck.Warn, Group: "Content",
				Title:  fmt.Sprintf("%s is no longer in %s", r.Title, st.Name),
				Tag:    "No longer in the store",
				Detail: fmt.Sprintf("Deleted in %s on %s. Sends leave it out. Remove it or pick another product.", platformName(st.Platform), r.RemovedAt.In(loc).Format("2 Jan, 15:04")),
				Path:   f.path, Section: f.section})
		}
	}
	return out
}
