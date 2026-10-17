package digests

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// History is the digest's recent history, newest first: posts that went
// in or were left out, issues sent or skipped, and the digest turned on or
// paused (the audit log keeps every settings change).
func (s *Service) History(ctx context.Context, ws *store.Workspace, id uuid.UUID, limit int) (DigestHistory, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestHistory{}, api.MapError(err, "Digest")
	}
	d := r.Digest
	if limit <= 0 {
		limit = 5
	}
	out := DigestHistory{Items: []DigestHistoryItem{}}
	// Posts first seen since the last sent issue (or since it was turned on).
	all, err := s.items(ctx, d, 50, false)
	if err != nil {
		return out, err
	}
	since := time.Time{}
	if d.FirstOnAt != nil {
		since = *d.FirstOnAt
	}
	if last, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil && last.StartedAt != nil {
		since = *last.StartedAt
	}
	draftCats, draftList := []string{}, ""
	if d.SiteID != nil {
		if site, err := s.site(ctx, ws, *d.SiteID); err == nil && site != nil {
			draftCats, draftList = site.DraftCategories, site.DraftList
		}
	}
	for _, it := range all {
		if !it.FirstSeenAt.After(since) {
			continue
		}
		h := DigestHistoryItem{At: it.FirstSeenAt, Title: it.Title, Via: it.FirstSeenVia, Published: it.PublishedAt}
		switch it.Status {
		case "new":
			h.Kind = "goes_in"
		case "left_out":
			h.Kind, h.LeftOutBy = "left_out", it.LeftOutBy
			if len(it.Categories) > 0 {
				h.Category = it.Categories[0]
			}
			for _, c := range it.Categories {
				if hasAny(draftCats, []string{c}) {
					h.Category, h.DraftList = c, draftList
				}
			}
		default:
			continue
		}
		out.Items = append(out.Items, h)
	}
	// Issues sent or skipped.
	issues, err := s.Issues(ctx, ws, id)
	if err != nil {
		return out, err
	}
	for _, is := range issues.Items {
		switch is.Status {
		case "sent", "sending":
			at := is.SendAt
			if is.SentAt != nil {
				at = is.SentAt
			}
			if at == nil {
				continue
			}
			h := DigestHistoryItem{Kind: "sent", At: *at, SendAt: is.SendAt, Owner: r.OwnerName}
			if is.Number != nil {
				h.Number = *is.Number
			}
			if is.Recipients != nil {
				h.Recipients = *is.Recipients
			}
			if is.Approval != nil && is.Approval.Status != "pending" && is.Approval.DecidedBy != nil {
				h.ApprovedBy, h.ApprovedAt = is.Approval.DecidedBy.Name, is.Approval.DecidedAt
			}
			out.Items = append(out.Items, h)
		case "skipped", "not_sent":
			at := is.SkippedAt
			if at == nil {
				at = is.SendAt
			}
			if at == nil {
				continue
			}
			out.Items = append(out.Items, DigestHistoryItem{Kind: is.Status, At: *at, SendAt: is.SendAt, SkipReason: is.SkipReason, Actor: is.SkippedBy, Note: is.SkipNote})
		}
	}
	// Turned on, paused, created: from the audit log.
	entries, err := s.Q.DigestAuditFor(ctx, store.DigestAuditForParams{WorkspaceID: &ws.ID, TargetID: id.String(), Lim: 30})
	if err != nil {
		return out, err
	}
	var created *DigestHistoryItem
	for _, e := range entries {
		var after, before map[string]any
		_ = json.Unmarshal(e.After, &after)
		_ = json.Unmarshal(e.Before, &before)
		switch e.Action {
		case "digest.turned_on", "digest.resumed":
			mode, _ := after["mode"].(string)
			out.Items = append(out.Items, DigestHistoryItem{Kind: "turned_on", At: e.OccurredAt, Actor: e.ActorName, Mode: mode})
		case "digest.paused":
			var meta map[string]any
			_ = json.Unmarshal(e.Metadata, &meta)
			reason, _ := meta["reason"].(string)
			out.Items = append(out.Items, DigestHistoryItem{Kind: "paused", At: e.OccurredAt, Actor: e.ActorName, Note: e.Reason, PauseReason: reason})
		case "digest.created":
			mode, _ := after["mode"].(string)
			created = &DigestHistoryItem{Kind: "created", At: e.OccurredAt, Actor: e.ActorName, Mode: mode}
		}
	}
	// "Daniel Park built it in draft mode that morning": who built it, on the
	// line of the first turn-on by someone else.
	if created != nil {
		attached := false
		for i := len(out.Items) - 1; i >= 0; i-- {
			if out.Items[i].Kind == "turned_on" {
				if out.Items[i].Actor != created.Actor {
					out.Items[i].BuiltBy, out.Items[i].BuiltMode = created.Actor, created.Mode
					at := created.At
					out.Items[i].BuiltAt = &at
					attached = true
				}
				break
			}
		}
		if !attached {
			out.Items = append(out.Items, *created)
		}
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].At.After(out.Items[j].At) })
	// What happened since the last sent issue, that issue, and the last time
	// the digest was turned on or paused: how it got to where it is.
	kept := []DigestHistoryItem{}
	sentSeen, stateSeen := false, false
	for _, h := range out.Items {
		switch h.Kind {
		case "sent":
			if sentSeen {
				continue
			}
			sentSeen = true
		case "skipped", "not_sent":
			if sentSeen {
				continue
			}
		case "turned_on", "paused", "created":
			if stateSeen {
				continue
			}
			stateSeen = true
		}
		kept = append(kept, h)
	}
	if len(kept) > limit {
		kept = kept[:limit]
	}
	out.Items = kept
	return out, nil
}
