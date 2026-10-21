package transactional

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Periods of the numbers.
const (
	Period24h = "24h"
	Period7d  = "7d"
	Period30d = "30d"
)

// periodSpan is a period's length and its sparkline buckets.
func periodSpan(period string) (time.Duration, int) {
	switch period {
	case Period24h:
		return 24 * time.Hour, 24
	case Period30d:
		return 30 * 24 * time.Hour, 30
	}
	return 7 * 24 * time.Hour, 7
}

// TransactionalStats are a template's (or the workspace's) numbers over a
// period. Rates are percentages of sent; MedianMs is from the call to the
// delivery event (null when nothing was delivered).
type TransactionalStats struct {
	Sent           int64   `json:"sent"`
	Delivered      int64   `json:"delivered"`
	DeliveryRate   float64 `json:"deliveryRate"`
	Bounced        int64   `json:"bounced"`
	BounceRate     float64 `json:"bounceRate"`
	Complained     int64   `json:"complained"`
	ComplaintRate  float64 `json:"complaintRate"`
	Deferred       int64   `json:"deferred"`
	Rejected       int64   `json:"rejected"`
	Failed         int64   `json:"failed"`
	Opened         int64   `json:"opened"`
	Clicked        int64   `json:"clicked"`
	MedianMs       *int64  `json:"medianMs,omitempty"`
	RejectedReason string  `json:"rejectedReason,omitempty" doc:"10 suppressed · 2 opted out"`
}

// TransactionalTemplateListItem is one row of the Templates table.
type TransactionalTemplateListItem struct {
	ID          string                    `json:"id"`
	Key         string                    `json:"key"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Category    *TransactionalCategoryRef `json:"category,omitempty"`
	LiveVersion *int                      `json:"liveVersion,omitempty"`
	// DraftChanges is true when the draft differs from the live version;
	// DraftVersion is the number it would publish as.
	DraftChanges bool               `json:"draftChanges"`
	DraftVersion int                `json:"draftVersion"`
	Archived     bool               `json:"archived"`
	Attachments  bool               `json:"attachments" doc:"Recent messages carried attachments"`
	Stats        TransactionalStats `json:"stats"`
	// Spark is the sent count per bucket of the period (hours for 24 hours,
	// days otherwise), oldest first.
	Spark      []int64    `json:"spark"`
	LastSentAt *time.Time `json:"lastSentAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	// UsedByProducts is how many published digital products send the
	// template as their delivery email ("Used by 2 products").
	UsedByProducts int `json:"usedByProducts,omitempty"`
}

// TransactionalTemplateList is the Templates tab's table.
type TransactionalTemplateList struct {
	Items  []TransactionalTemplateListItem `json:"items"`
	Period string                          `json:"period" enum:"24h,7d,30d"`
	// CanEdit is transactional:edit (New template, Import templates).
	CanEdit        bool   `json:"canEdit"`
	CanPublish     bool   `json:"canPublish"`
	ReadOnlyReason string `json:"readOnlyReason,omitempty"`
}

func ratePct(n, d int64) float64 {
	if d <= 0 {
		return 0
	}
	return math.Round(float64(n)/float64(d)*100*1000) / 1000
}

// Templates lists the templates with their numbers over a period.
func (s *Service) Templates(ctx context.Context, ws *store.Workspace, p *authz.Principal, period string, includeTest bool) (TransactionalTemplateList, error) {
	if period != Period24h && period != Period30d {
		period = Period7d
	}
	span, buckets := periodSpan(period)
	since := s.now().Add(-span)
	rows, err := s.Q.TxnTemplates(ctx, ws.ID)
	if err != nil {
		return TransactionalTemplateList{}, err
	}
	stats, err := s.Q.TxnTemplateStats(ctx, store.TxnTemplateStatsParams{WorkspaceID: ws.ID, Since: since, IncludeTest: includeTest})
	if err != nil {
		return TransactionalTemplateList{}, err
	}
	byTemplate := map[uuid.UUID]store.TxnTemplateStatsRow{}
	for _, r := range stats {
		byTemplate[r.TemplateID] = r
	}
	spark, err := s.Q.TxnTemplateSpark(ctx, store.TxnTemplateSparkParams{WorkspaceID: ws.ID, Since: since, BucketSeconds: span.Seconds() / float64(buckets), IncludeTest: includeTest})
	if err != nil {
		return TransactionalTemplateList{}, err
	}
	sparks := map[uuid.UUID][]int64{}
	for _, r := range spark {
		sp := sparks[r.TemplateID]
		if sp == nil {
			sp = make([]int64, buckets)
			sparks[r.TemplateID] = sp
		}
		if b := int(r.Bucket); b >= 0 && b < buckets {
			sp[b] += r.N
		}
	}
	out := TransactionalTemplateList{Items: make([]TransactionalTemplateListItem, 0, len(rows)), Period: period, CanEdit: p.Can(authz.TransactionalEdit), CanPublish: p.Can(authz.TransactionalPublish)}
	if !out.CanEdit {
		out.ReadOnlyReason = "You can view templates, test renders, the log and reports. Editing needs transactional:edit and publishing needs transactional:publish. Ask " +
			joinNames(s.helpers(ctx, ws.ID, authz.TransactionalPublish, 3)) + "."
	}
	// Digital products' delivery templates (X19).
	usedBy := map[uuid.UUID]int{}
	if uses, err := s.Q.ProductTemplateUses(ctx, ws.ID); err == nil {
		for _, u := range uses {
			usedBy[u.TemplateID] = int(u.Products)
		}
	}
	for _, r := range rows {
		it := TransactionalTemplateListItem{ID: r.ID.String(), Key: r.Key, Name: r.Name, Description: r.Description, Category: categoryRef(r.CategoryID, r.CategoryName, r.CategoryKind),
			UsedByProducts: usedBy[r.ID],
			DraftChanges:   r.LiveHash == "" || r.LiveHash != r.DraftHash, DraftVersion: int(r.VersionsCount) + 1, Archived: r.ArchivedAt != nil, UpdatedAt: r.UpdatedAt}
		if r.LiveVersion != nil {
			v := int(*r.LiveVersion)
			it.LiveVersion = &v
		}
		if st, ok := byTemplate[r.ID]; ok {
			it.Stats = statsOf(st.Sent, st.Delivered, st.Bounced, st.Complained, st.Deferred, st.Rejected, st.Failed, st.Opened, st.Clicked, st.MedianMs)
			it.Attachments = st.Attachments
			if !st.LastSentAt.IsZero() && st.LastSentAt.Year() > 1970 {
				t := st.LastSentAt
				it.LastSentAt = &t
			}
		}
		it.Spark = sparks[r.ID]
		if it.Spark == nil {
			it.Spark = make([]int64, buckets)
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

func statsOf(sent, delivered, bounced, complained, deferred, rejected, failed, opened, clicked, median int64) TransactionalStats {
	st := TransactionalStats{Sent: sent, Delivered: delivered, Bounced: bounced, Complained: complained, Deferred: deferred, Rejected: rejected, Failed: failed,
		Opened: opened, Clicked: clicked, DeliveryRate: ratePct(delivered, sent), BounceRate: ratePct(bounced, sent), ComplaintRate: ratePct(complained, sent)}
	if median >= 0 && delivered > 0 {
		m := median
		st.MedianMs = &m
	}
	return st
}
