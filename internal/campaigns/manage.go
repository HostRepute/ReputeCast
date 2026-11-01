package campaigns

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Deleting, restoring and archiving campaigns, tags and bulk actions
// (review 2026-11 CA-1, CA-2, D4). Sent campaigns are archived, never
// deleted: reports, web posts, retries and approvals keep pointing at them.
// Drafts and campaigns that sent nothing are deleted, restorable for 30 days.

// RestoreWindow is how long a deleted campaign can be restored.
const RestoreWindow = 30 * 24 * time.Hour

var (
	mProblemDeleteSent     = msgs.M("problem.deleteSent")
	mProblemDeleteRunning  = msgs.M("problem.deleteRunning")
	mProblemNotDeleted     = msgs.M("problem.notDeleted")
	mProblemRestoreExpired = msgs.M("problem.restoreExpired")
	mProblemArchiveUnsent  = msgs.M("problem.archiveUnsent")
	mProblemNotArchived    = msgs.M("problem.notArchived")
	mProblemTag            = msgs.M("problem.tag")
	mApprovalClosed        = msgs.M("approval.closedByDelete")
)

// manageRow is what decides what a campaign allows.
type manageRow = store.LockCampaignForManageRow

func deletable(c manageRow) bool {
	switch c.Status {
	case "draft", "approval", "approved":
		return !c.SentAny
	case "cancelled", "failed":
		return !c.SentAny
	}
	return false
}

func archivable(c manageRow) bool {
	return c.Status == "sent" || ((c.Status == "cancelled" || c.Status == "failed") && c.SentAny)
}

func (s *Service) lockManage(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID) (manageRow, error) {
	c, err := q.LockCampaignForManage(ctx, store.LockCampaignForManageParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return c, api.MapError(err, "Campaign")
	}
	return c, nil
}

// Delete moves a draft, or a campaign that sent nothing, to Deleted; its
// open approval request closes. Sent campaigns are archived instead.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error { return s.deleteTx(ctx, store.New(tx), ws, by, id) })
}

func (s *Service) deleteTx(ctx context.Context, q *store.Queries, ws *store.Workspace, by *uuid.UUID, id uuid.UUID) error {
	l := i18n.LangFrom(ctx)
	c, err := s.lockManage(ctx, q, ws, id)
	if err != nil {
		return err
	}
	if c.DeletedAt != nil {
		return api.NotFound("Campaign")
	}
	if !deletable(c) {
		if c.Status == "scheduled" || c.Status == "sending" || c.Status == "paused" {
			return api.Conflict(mProblemDeleteRunning.Text(l, i18n.Args{"status": c.Status}))
		}
		return api.Err(409, "archive_instead", mProblemDeleteSent.Text(l, nil))
	}
	now := s.now()
	if err := q.SetCampaignDeleted(ctx, store.SetCampaignDeletedParams{ID: id, DeletedAt: &now, DeletedBy: by, Now: now}); err != nil {
		return err
	}
	closed, err := q.CloseCampaignApprovals(ctx, store.CloseCampaignApprovalsParams{CampaignID: &id, Reason: mApprovalClosed.Text(i18n.Default, nil)})
	if err != nil {
		return err
	}
	// A draft whose approval just closed goes back to being a plain draft.
	if c.Status == "approval" || c.Status == "approved" {
		if err := q.SetCampaignDraftStatus(ctx, id); err != nil {
			return err
		}
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.delete", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
		Before: map[string]any{"status": c.Status}, Metadata: map[string]any{"approvalsClosed": closed, "restorableUntil": now.Add(RestoreWindow)}})
}

// RestoreDeleted brings a deleted campaign back within 30 days.
func (s *Service) RestoreDeleted(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	l := i18n.LangFrom(ctx)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := s.lockManage(ctx, q, ws, id)
		if err != nil {
			return err
		}
		if c.DeletedAt == nil {
			return api.Conflict(mProblemNotDeleted.Text(l, nil))
		}
		now := s.now()
		if now.Sub(*c.DeletedAt) > RestoreWindow {
			return api.Err(410, "restore_expired", mProblemRestoreExpired.Text(l, nil))
		}
		if err := q.SetCampaignDeleted(ctx, store.SetCampaignDeletedParams{ID: id, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.restore", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"deleted_at": c.DeletedAt}})
	})
}

// Archive takes a sent campaign off the list; Unarchive puts it back.
func (s *Service) Archive(ctx context.Context, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, archive bool) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error { return s.archiveTx(ctx, store.New(tx), ws, by, id, archive) })
}

func (s *Service) archiveTx(ctx context.Context, q *store.Queries, ws *store.Workspace, by *uuid.UUID, id uuid.UUID, archive bool) error {
	l := i18n.LangFrom(ctx)
	c, err := s.lockManage(ctx, q, ws, id)
	if err != nil {
		return err
	}
	if c.DeletedAt != nil {
		return api.NotFound("Campaign")
	}
	now := s.now()
	action := "campaign.archive"
	if archive {
		if c.ArchivedAt != nil {
			return nil
		}
		if !archivable(c) {
			return api.Err(409, "delete_instead", mProblemArchiveUnsent.Text(l, nil))
		}
		if err := q.SetCampaignArchived(ctx, store.SetCampaignArchivedParams{ID: id, ArchivedAt: &now, ArchivedBy: by, Now: now}); err != nil {
			return err
		}
	} else {
		if c.ArchivedAt == nil {
			return api.Conflict(mProblemNotArchived.Text(l, nil))
		}
		action = "campaign.unarchive"
		if err := q.SetCampaignArchived(ctx, store.SetCampaignArchivedParams{ID: id, Now: now}); err != nil {
			return err
		}
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
		Before: map[string]any{"archived": c.ArchivedAt != nil}, After: map[string]any{"archived": archive}})
}

// reTag is a campaign tag: lower case letters, digits and - _ . / and spaces.
var reTag = regexp.MustCompile(`^[\p{Ll}\p{N}][\p{Ll}\p{N} _./-]{0,39}$`)

// NormalizeTag lower-cases and trims a tag, or reports it invalid.
func NormalizeTag(raw string) (string, bool) {
	t := strings.ToLower(strings.Join(strings.Fields(raw), " "))
	return t, reTag.MatchString(t)
}

// CampaignBulkInput is a bulk action on campaigns (CA-2).
type CampaignBulkInput struct {
	IDs    []string `json:"ids" minItems:"1" maxItems:"200"`
	Action string   `json:"action" enum:"archive,unarchive,delete,tag,untag"`
	// Tag is the tag for tag and untag.
	Tag string `json:"tag,omitempty" maxLength:"40"`
}

// CampaignBulkSkip is a campaign a bulk action left alone, and why.
type CampaignBulkSkip struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// CampaignBulkResult says what a bulk action did.
type CampaignBulkResult struct {
	Done    int                `json:"done"`
	Skipped []CampaignBulkSkip `json:"skipped"`
}

// Bulk archives, unarchives, deletes or tags campaigns, each on its own:
// one that can't (a draft can't be archived, a sent one deleted) is
// skipped with the reason and the others go on. Each change is audited.
func (s *Service) Bulk(ctx context.Context, ws *store.Workspace, p *authz.Principal, in CampaignBulkInput) (CampaignBulkResult, error) {
	l := i18n.LangFrom(ctx)
	out := CampaignBulkResult{Skipped: []CampaignBulkSkip{}}
	tag := ""
	if in.Action == "tag" || in.Action == "untag" {
		t, ok := NormalizeTag(in.Tag)
		if !ok {
			return out, api.Invalid("body.tag", mProblemTag.Text(l, nil))
		}
		tag = t
	}
	var by *uuid.UUID
	if p != nil && p.UserID != uuid.Nil {
		by = &p.UserID
	}
	seen := map[uuid.UUID]bool{}
	for _, raw := range in.IDs {
		id, err := uuid.Parse(raw)
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		name := ""
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			switch in.Action {
			case "archive", "unarchive":
				return s.archiveTx(ctx, q, ws, by, id, in.Action == "archive")
			case "delete":
				return s.deleteTx(ctx, q, ws, by, id)
			default:
				c, err := s.lockManage(ctx, q, ws, id)
				if err != nil {
					return err
				}
				name = c.Name
				tags := slices.DeleteFunc(slices.Clone(c.Tags), func(t string) bool { return t == tag })
				if in.Action == "tag" {
					tags = append(tags, tag)
					slices.Sort(tags)
				}
				if slices.Equal(tags, c.Tags) {
					return nil
				}
				if err := q.SetCampaignTags(ctx, store.SetCampaignTagsParams{ID: id, Tags: tags, Now: s.now()}); err != nil {
					return err
				}
				return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.tags", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
					Before: map[string]any{"tags": c.Tags}, After: map[string]any{"tags": tags}})
			}
		})
		if err != nil {
			if name == "" {
				if c, lerr := s.Q.LockCampaignForManage(ctx, store.LockCampaignForManageParams{WorkspaceID: ws.ID, ID: id}); lerr == nil {
					name = c.Name
				}
			}
			out.Skipped = append(out.Skipped, CampaignBulkSkip{ID: id.String(), Name: name, Reason: problemText(err)})
			continue
		}
		out.Done++
	}
	return out, nil
}

// CampaignTags lists the workspace's campaign tags.
type CampaignTags struct {
	Tags []string `json:"tags"`
}

// Tags returns the tags in use.
func (s *Service) Tags(ctx context.Context, ws *store.Workspace) (CampaignTags, error) {
	tags, err := s.Q.CampaignTagsInUse(ctx, ws.ID)
	if tags == nil {
		tags = []string{}
	}
	return CampaignTags{Tags: tags}, err
}

// PurgeDeleted removes campaigns deleted more than 30 days ago.
func (s *Service) PurgeDeleted(ctx context.Context) (int64, error) {
	before := s.now().Add(-RestoreWindow)
	return s.Q.PurgeDeletedCampaigns(ctx, &before)
}

// CampaignCalendarItem is one thing on the campaigns calendar (CA-3).
type CampaignCalendarItem struct {
	// Kind: scheduled (can be moved), sent (or sending) and digest_issue
	// (an issue still to be prepared).
	Kind       string    `json:"kind" enum:"scheduled,sent,digest_issue"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Channel    string    `json:"channel"`
	Status     string    `json:"status"`
	At         time.Time `json:"at"`
	Recipients *int64    `json:"recipients,omitempty"`
	Sent       int64     `json:"sent"`
	// Local is true for a campaign that goes at a time in each recipient's
	// time zone; LocalTime is that time ("09:00").
	Local     bool   `json:"local,omitempty"`
	LocalTime string `json:"localTime,omitempty"`
	DigestID  string `json:"digestId,omitempty"`
	// CampaignID is the campaign of a digest issue, once there is one.
	CampaignID string `json:"campaignId,omitempty"`
	// Approved: an approval covers a scheduled campaign's time, so moving
	// it means unscheduling and asking again (review 2026-11 CA-10).
	Approved bool `json:"approved,omitempty"`
}

// CampaignCalendar is the calendar between two times.
type CampaignCalendar struct {
	From  time.Time              `json:"from"`
	To    time.Time              `json:"to"`
	Items []CampaignCalendarItem `json:"items"`
}

// Calendar lists scheduled and sent campaigns and digest issues to come.
func (s *Service) Calendar(ctx context.Context, ws *store.Workspace, from, to time.Time) (CampaignCalendar, error) {
	out := CampaignCalendar{From: from, To: to, Items: []CampaignCalendarItem{}}
	if !to.After(from) || to.Sub(from) > 62*24*time.Hour {
		return out, api.Invalid("to", mProblemCalendarRange.Text(i18n.LangFrom(ctx), nil))
	}
	rows, err := s.Q.CampaignCalendar(ctx, store.CampaignCalendarParams{WorkspaceID: ws.ID, FromAt: &from, ToAt: &to})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		it := CampaignCalendarItem{Kind: "sent", ID: r.ID.String(), Name: r.Name, Channel: r.Channel, Status: r.Status, At: r.At, Recipients: r.AudienceCount, Sent: r.Sent}
		if r.Status == "scheduled" {
			it.Kind = "scheduled"
		}
		var sc struct {
			Mode  string `json:"mode"`
			Local string `json:"local"`
		}
		if r.ChannelMode != "" {
			// Texts and WhatsApp keep their schedule with the channel (CA-9).
			sc.Mode, sc.Local = r.ChannelMode, r.ChannelLocal
		} else {
			_ = json.Unmarshal(r.Schedule, &sc)
		}
		if sc.Mode == "local" && len(sc.Local) >= 16 {
			// Shown on its wall time's day, in the workspace's zone (the
			// send itself starts in the earliest zone).
			it.Local, it.LocalTime = true, sc.Local[11:16]
			if at, err := time.ParseInLocation("2006-01-02 15:04", sc.Local, wsLocation(ws)); err == nil && r.Status == "scheduled" {
				it.At = at
			}
		}
		if r.DigestID != nil {
			it.DigestID = r.DigestID.String()
		}
		it.Approved = r.Approved && r.Status == "scheduled"
		out.Items = append(out.Items, it)
	}
	issues, err := s.Q.DigestIssuesDue(ctx, store.DigestIssuesDueParams{WorkspaceID: ws.ID, FromAt: &from, ToAt: &to})
	if err != nil {
		return out, err
	}
	for _, d := range issues {
		if d.SendAt == nil {
			continue
		}
		it := CampaignCalendarItem{Kind: "digest_issue", ID: d.ID.String(), Name: d.Name, Channel: "email", Status: d.Status, At: *d.SendAt, DigestID: d.DigestID.String()}
		if d.CampaignID != nil {
			it.CampaignID = d.CampaignID.String()
		}
		out.Items = append(out.Items, it)
	}
	slices.SortStableFunc(out.Items, func(a, b CampaignCalendarItem) int { return a.At.Compare(b.At) })
	return out, nil
}

var mProblemCalendarRange = msgs.M("problem.calendarRange")

// problemText is an error as the person reads it.
func problemText(err error) string {
	var p *api.Problem
	if errors.As(err, &p) {
		return p.Detail
	}
	var se huma.StatusError
	if errors.As(err, &se) {
		return se.Error()
	}
	return err.Error()
}
