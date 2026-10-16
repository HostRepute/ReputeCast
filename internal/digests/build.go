package digests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// built is an issue's content, from the template and its posts.
type built struct {
	Name, Subject, Preheader string
	MJML, HTML, Plain        string
	// Items are the posts the issue shows, in its order; Rest the new
	// posts left out (or kept for the next issue).
	Items []BlockItem
	Rest  []BlockItem
	Block DigestFeedBlock
}

// issueName is "Engineering blog roundup #3".
func issueName(digest string, number int) string { return fmt.Sprintf("%s #%d", digest, number) }

// build makes an issue's content: the Feed items block expanded with the
// posts, feed tokens filled in, MJML compiled. Personalization stays for
// the send.
func build(d store.Digest, tpl store.DigestCampaignTemplateRow, posts []BlockItem, number int, sendAt *time.Time, loc *time.Location) (built, error) {
	block, _ := BlockOf(tpl.Mjml)
	shown, rest := block.Select(posts)
	for i := range shown {
		shown[i] = block.withImage(shown[i])
	}
	data := IssueData{FeedTitle: d.FeedTitle, FeedURL: d.FeedUrl, Number: number, Items: shown, Loc: loc}
	if sendAt != nil {
		data.Date = *sendAt
	}
	out := built{Name: issueName(d.Name, number), Items: shown, Rest: rest, Block: block}
	var err error
	if out.Subject, err = RenderIssue(tpl.Subject, data, false); err != nil {
		return out, fmt.Errorf("the subject: %w", err)
	}
	if out.Preheader, err = RenderIssue(tpl.Preheader, data, false); err != nil {
		return out, fmt.Errorf("the preheader: %w", err)
	}
	out.Subject, out.Preheader = strings.TrimSpace(out.Subject), strings.TrimSpace(out.Preheader)
	src := ExpandBlock(tpl.Mjml, shown, loc)
	if src, err = RenderIssue(src, data, true); err != nil {
		return out, fmt.Errorf("the email: %w", err)
	}
	out.MJML = mjml.SetHeadMeta(src, out.Subject, out.Preheader)
	res := mjml.Compile(out.MJML)
	if len(res.Errors) > 0 {
		return out, fmt.Errorf("the email doesn't compile: %s", res.Errors[0].Message)
	}
	out.HTML = res.HTML
	return out, nil
}

// materialize writes an issue's campaign from the template as it is now:
// a new campaign the first time (the next issue someone checks or tests is
// hidden from the campaigns list until it is prepared), a new version of it
// after. role is "next" or "issue".
func (s *Service) materialize(ctx context.Context, ws *store.Workspace, d store.Digest, issue store.DigestIssue, posts []BlockItem, number int, role string,
	by *uuid.UUID, label string) (uuid.UUID, built, error) {
	tpl, err := s.template(ctx, d)
	if err != nil {
		return uuid.Nil, built{}, err
	}
	if tpl == nil {
		return uuid.Nil, built{}, errNoTemplate
	}
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	b, err := build(d, *tpl, posts, number, issue.SendAt, loc)
	if err != nil {
		return uuid.Nil, b, err
	}
	sendAt := issue.SendAt
	schedule := []byte(`{"mode":"now"}`)
	if sendAt != nil {
		schedule, _ = json.Marshal(map[string]any{"mode": "at", "at": sendAt.UTC().Format(time.RFC3339), "timezone": loc.String()})
	}
	now := s.now()
	id := uuid.Nil
	if issue.CampaignID != nil {
		id = *issue.CampaignID
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		html := accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerCampaign, id, 0, b.HTML)
		plain := contentcheck.PlainText(html)
		hash := campaigns.ContentHash(b.Subject, b.Preheader, b.MJML, html, nil)
		version := int32(1)
		existing := false
		if id != uuid.Nil {
			st, err := q.DigestCampaignState(ctx, id)
			if err == nil && st.DeletedAt == nil {
				existing = true
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		if existing {
			v, err := q.DigestCampaignRebuild(ctx, store.DigestCampaignRebuildParams{ID: id, Name: b.Name, Subject: b.Subject, Preheader: b.Preheader, SenderID: tpl.SenderID,
				ReplyTo: tpl.ReplyTo, Audience: tpl.Audience, Mjml: b.MJML, Html: html, PlainText: plain, Thresholds: tpl.Thresholds, RouteOverride: tpl.RouteOverride,
				Tracking: tpl.Tracking, ContentHash: hash, ContentSettings: tpl.ContentSettings, Schedule: schedule, DigestRole: role, UpdatedBy: by, Now: now})
			if err != nil {
				return err
			}
			version = v
		} else {
			id = uuid.Must(uuid.NewV7())
			if err := q.DigestCampaignInsert(ctx, store.DigestCampaignInsertParams{ID: id, WorkspaceID: ws.ID, Name: b.Name, Subject: b.Subject, Preheader: b.Preheader,
				SenderID: tpl.SenderID, ReplyTo: tpl.ReplyTo, Audience: tpl.Audience, AudienceCount: tpl.AudienceCount, Mjml: b.MJML, Html: html, PlainText: plain,
				Schedule: schedule, Thresholds: tpl.Thresholds, RouteOverride: tpl.RouteOverride, Tracking: tpl.Tracking, ContentHash: hash,
				ContentSettings: tpl.ContentSettings, DigestID: &d.ID, DigestRole: role, CreatedBy: by, Now: now}); err != nil {
				return err
			}
			if err := q.DigestIssueSetCampaign(ctx, store.DigestIssueSetCampaignParams{ID: issue.ID, CampaignID: &id, Now: now}); err != nil {
				return err
			}
		}
		if err := q.DigestCampaignVersionInsert(ctx, store.DigestCampaignVersionInsertParams{CampaignID: id, Version: version, Subject: b.Subject, Preheader: b.Preheader,
			Mjml: b.MJML, Html: html, PlainText: plain, SavedBy: by, SavedAt: now, Label: label, Source: "template", SizeBytes: int32(len(html))}); err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(b.Items))
		for _, it := range b.Items {
			if u, err := uuid.Parse(it.ID); err == nil {
				ids = append(ids, u)
			}
		}
		if err := q.DigestIssueItemsClear(ctx, issue.ID); err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := q.DigestIssueItemsSet(ctx, store.DigestIssueItemsSetParams{IssueID: issue.ID, ItemIds: ids}); err != nil {
				return err
			}
		}
		if role != "issue" {
			return nil
		}
		basis, _ := json.Marshal(basisOf(d, *tpl, ws.Timezone))
		return q.DigestIssueSetBasis(ctx, store.DigestIssueSetBasisParams{ID: issue.ID, Basis: basis})
	})
	return id, b, err
}

// issueBasis is what an issue was built on: the template's version, its
// audience, route, sender and the schedule. An approver sees what changed
// since the last approved issue.
type issueBasis struct {
	TemplateVersion int32           `json:"template_version"`
	Audience        json.RawMessage `json:"audience"`
	Route           json.RawMessage `json:"route"`
	Sender          string          `json:"sender"`
	Schedule        string          `json:"schedule"`
}

func basisOf(d store.Digest, tpl store.DigestCampaignTemplateRow, tz string) issueBasis {
	b := issueBasis{TemplateVersion: tpl.Version, Audience: tpl.Audience, Route: tpl.RouteOverride, Schedule: ParseSchedule(d.Schedule, tz).Describe()}
	if tpl.SenderID != nil {
		b.Sender = tpl.SenderID.String()
	}
	if len(b.Route) == 0 {
		b.Route = json.RawMessage(`null`)
	}
	if len(b.Audience) == 0 {
		b.Audience = json.RawMessage(`{}`)
	}
	return b
}

// changedSince lists what differs between two bases: template, audience,
// route, schedule (the sender counts as the template's).
func changedSince(was, now issueBasis) []string {
	out := []string{}
	if was.TemplateVersion != now.TemplateVersion || was.Sender != now.Sender {
		out = append(out, "template")
	}
	if !jsonEqual(was.Audience, now.Audience) {
		out = append(out, "audience")
	}
	if !jsonEqual(was.Route, now.Route) {
		out = append(out, "route")
	}
	if was.Schedule != now.Schedule {
		out = append(out, "schedule")
	}
	return out
}

// jsonEqual compares two JSON documents by value ("{}" and "null" are both empty).
func jsonEqual(a, b json.RawMessage) bool {
	var x, y any
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal(b, &y)
	empty := func(v any) bool {
		if v == nil {
			return true
		}
		m, ok := v.(map[string]any)
		return ok && len(m) == 0
	}
	if empty(x) && empty(y) {
		return true
	}
	xa, _ := json.Marshal(x)
	ya, _ := json.Marshal(y)
	return string(xa) == string(ya)
}

var errNoTemplate = errors.New("the digest has no template")
