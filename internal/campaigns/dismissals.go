package campaigns

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Dismissing warnings in the composer (review 2026-10, CO-14,
// ComposerWarnings): a warning someone judged with a reason ("the image is
// decorative") stays listed with their name and reason instead of counting
// as a warning. Blocking issues can't be dismissed. A dismissal is for the
// warning as it was: when it changes (the email grew past what it said,
// the subject changed) it comes back.

// ComposerDismissal is a dismissed warning.
type ComposerDismissal struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	CheckID string `json:"checkId"`
	Reason  string `json:"reason" enum:"decorative,size,subject,wrong,other"`
	// ReasonText is the reason in the reader's language ("the image is decorative").
	ReasonText string    `json:"reasonText"`
	Note       string    `json:"note"`
	By         string    `json:"by"`
	At         time.Time `json:"at"`
}

// ComposerDismissedIssue marks one of the checks' issues as dismissed.
type ComposerDismissedIssue struct {
	// Index is the issue's position in issues.
	Index     int               `json:"index"`
	Dismissal ComposerDismissal `json:"dismissal"`
}

// ComposerDismissInput dismisses a warning of the last checks.
type ComposerDismissInput struct {
	// Key is the warning's key from the checks (issueKeys).
	Key     string `json:"key" minLength:"1" maxLength:"64" pattern:"^[a-f0-9]+$"`
	CheckID string `json:"checkId" minLength:"1" maxLength:"80"`
	Title   string `json:"title,omitempty" maxLength:"300"`
	Reason  string `json:"reason" enum:"decorative,size,subject,wrong,other"`
	Note    string `json:"note,omitempty" maxLength:"500"`
}

// IssueKey names a warning as it is: its check, element and words (and the
// subject, for subject checks). English words, whatever the reader's language.
func IssueKey(is contentcheck.ContentCheckIssue, subject string) string {
	path := make([]string, len(is.Path))
	for i, p := range is.Path {
		path[i] = strconv.Itoa(p)
	}
	parts := []string{is.ID, strings.Join(path, "."), is.Title, is.URL}
	if is.Group == "Subject" {
		parts = append(parts, subject)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:16])
}

var dismissReasons = map[string]i18n.Msg{
	"decorative": msgs.M("dismiss.decorative"),
	"size":       msgs.M("dismiss.size"),
	"subject":    msgs.M("dismiss.subject"),
	"wrong":      msgs.M("dismiss.wrong"),
	"other":      msgs.M("dismiss.other"),
}

func reasonText(l i18n.Lang, reason, note string) string {
	if reason == "other" && strings.TrimSpace(note) != "" {
		return strings.TrimSpace(note)
	}
	if m, ok := dismissReasons[reason]; ok {
		return m.Text(l, nil)
	}
	return reason
}

func dismissalOf(l i18n.Lang, r store.CheckDismissalsListRow) ComposerDismissal {
	return ComposerDismissal{ID: r.ID.String(), Key: r.IssueKey, CheckID: r.CheckID, Reason: r.Reason, ReasonText: reasonText(l, r.Reason, r.Note), Note: r.Note, By: r.DismissedByName, At: r.CreatedAt}
}

// markDismissed gives the checks their issue keys and marks the warnings
// someone dismissed. Blocking issues are never marked.
func (s *Service) markDismissed(ctx context.Context, ws *store.Workspace, id uuid.UUID, subject string, out *ComposerChecks) error {
	out.IssueKeys = make([]string, len(out.Issues))
	out.Dismissed = []ComposerDismissedIssue{}
	rows, err := s.Q.CheckDismissalsList(ctx, store.CheckDismissalsListParams{WorkspaceID: ws.ID, CampaignID: id})
	if err != nil {
		return err
	}
	byKey := make(map[string]store.CheckDismissalsListRow, len(rows))
	for _, r := range rows {
		byKey[r.IssueKey] = r
	}
	l := i18n.LangFrom(ctx)
	for i, is := range out.Issues {
		key := IssueKey(is, subject)
		out.IssueKeys[i] = key
		if r, ok := byKey[key]; ok && is.Severity == contentcheck.Warn {
			out.Dismissed = append(out.Dismissed, ComposerDismissedIssue{Index: i, Dismissal: dismissalOf(l, r)})
		}
	}
	return nil
}

// Dismiss records why someone dismissed a warning.
func (s *Service) Dismiss(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerDismissInput) (ComposerDismissal, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerDismissal{}, api.MapError(err, "Campaign")
	}
	l := i18n.LangFrom(ctx)
	if in.Reason == "other" && strings.TrimSpace(in.Note) == "" {
		return ComposerDismissal{}, api.Invalid("body.note", mDismissNote.Text(l, nil))
	}
	now := s.now()
	did := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		got, err := q.CheckDismissalInsert(ctx, store.CheckDismissalInsertParams{ID: did, WorkspaceID: ws.ID, CampaignID: id, IssueKey: in.Key, CheckID: in.CheckID,
			Title: in.Title, Reason: in.Reason, Note: strings.TrimSpace(in.Note), DismissedBy: &p.UserID, CreatedAt: now})
		if err != nil {
			return err
		}
		did = got
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.check.dismiss", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			After: map[string]any{"check": in.CheckID, "title": in.Title, "reason": in.Reason, "note": strings.TrimSpace(in.Note)}})
	})
	if err != nil {
		return ComposerDismissal{}, err
	}
	return ComposerDismissal{ID: did.String(), Key: in.Key, CheckID: in.CheckID, Reason: in.Reason, ReasonText: reasonText(l, in.Reason, in.Note), Note: strings.TrimSpace(in.Note), By: p.Name, At: now}, nil
}

// Undismiss counts a dismissed warning as a warning again (Undo).
func (s *Service) Undismiss(ctx context.Context, ws *store.Workspace, id, dismissal uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.CheckDismissalDelete(ctx, store.CheckDismissalDeleteParams{WorkspaceID: ws.ID, CampaignID: id, ID: dismissal})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Dismissal")
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.check.undismiss", TargetType: "campaign", TargetID: id.String(),
			Before: map[string]any{"check": row.CheckID, "title": row.Title, "reason": row.Reason}})
	})
}
