package visibility

import (
	"context"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// VisibilityUseEmail is an email whose conditional blocks use a segment,
// list, field or tag.
type VisibilityUseEmail struct {
	CampaignID  string     `json:"campaignId"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	// Blocks are the blocks' names ("Annual billing offer"), and Paths
	// where they are (to open the block in the composer).
	Blocks []string `json:"blocks"`
	Paths  [][]int  `json:"paths"`
}

// VisibilityUsage is where something is used by Visibility rules, in emails
// that haven't gone out yet.
type VisibilityUsage struct {
	Blocks int                  `json:"blocks"`
	Emails []VisibilityUseEmail `json:"emails"`
}

// Usage finds the unsent emails whose rules refer to a segment, a list
// (by id), a field (by key) or a tag.
func (s *Service) Usage(ctx context.Context, ws *store.Workspace, kind, ref string) (VisibilityUsage, error) {
	out := VisibilityUsage{Emails: []VisibilityUseEmail{}}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return out, nil
	}
	rows, err := s.Q.VisibilityUnsent(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if !strings.Contains(r.Mjml, ref) {
			continue
		}
		e := Read(r.Mjml)
		use := VisibilityUseEmail{CampaignID: r.ID.String(), Name: r.Name, Status: r.Status, ScheduledAt: r.ScheduledAt, Blocks: []string{}, Paths: [][]int{}}
		for _, b := range e.Blocks {
			for _, c := range b.Rule.Conditions {
				k, ok := Lookup(c.Kind)
				if !ok || k.Ref == nil {
					continue
				}
				if rf := k.Ref(c); rf.Kind == kind && rf.ID == ref {
					use.Blocks = append(use.Blocks, b.Name)
					use.Paths = append(use.Paths, b.Path)
					break
				}
			}
		}
		if len(use.Blocks) > 0 {
			out.Blocks += len(use.Blocks)
			out.Emails = append(out.Emails, use)
		}
	}
	return out, nil
}
