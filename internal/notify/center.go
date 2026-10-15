package notify

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// RetentionDays is how long notifications are kept.
const RetentionDays = 90

// Service serves the notification center and the delivery preferences.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	// Mail reports whether email can go out (System stream connected).
	Mail func() mailer.Sender
}

// NotificationItem is one entry of the notification center.
type NotificationItem struct {
	ID            string    `json:"id"`
	Kind          string    `json:"kind"`
	Category      string    `json:"category" enum:"approvals,sending,providers,team,hostrepute,other,transactional"`
	CategoryLabel string    `json:"categoryLabel" example:"Approvals"`
	Severity      string    `json:"severity" enum:"info,ok,warn,bad"`
	Title         string    `json:"title" example:"Approval requested"`
	Summary       string    `json:"summary" doc:"Short line for the bell popover"`
	Body          string    `json:"body" doc:"Full sentence for the notification center"`
	Link          string    `json:"link" doc:"App path of the screen where you can act"`
	CreatedAt     time.Time `json:"createdAt"`
	Read          bool      `json:"read"`
}

// NotificationList is a page of the notification center.
type NotificationList struct {
	Items []NotificationItem `json:"items"`
	// Unread counts every unread notification in the workspace (all tabs).
	Unread int64 `json:"unread"`
	// Total counts the notifications matching the filter.
	Total         int64  `json:"total"`
	NextCursor    string `json:"nextCursor,omitempty"`
	RetentionDays int    `json:"retentionDays"`
}

// Filters of the notification center tabs.
const (
	FilterAll    = "all"
	FilterUnread = "unread"
)

func userAndWorkspace(ctx context.Context) (uuid.UUID, *uuid.UUID, error) {
	u := api.UserFrom(ctx)
	ws := api.WorkspaceFrom(ctx)
	if u == nil || ws == nil {
		return uuid.Nil, nil, api.Err(401, api.CodeUnauthenticated, "Sign in to continue.")
	}
	id := ws.ID
	return u.ID, &id, nil
}

// List returns the caller's notifications in the current workspace, unread
// first, then newest first.
func (s *Service) List(ctx context.Context, filter, cursor string, limit int) (NotificationList, error) {
	userID, wsID, err := userAndWorkspace(ctx)
	if err != nil {
		return NotificationList{}, err
	}
	var kinds []string
	unreadOnly := false
	switch filter {
	case "", FilterAll:
	case FilterUnread:
		unreadOnly = true
	default:
		kinds = KindsOf(filter)
		if kinds == nil {
			return NotificationList{}, api.Invalid("query.filter", "Unknown filter.")
		}
	}
	off := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return NotificationList{}, api.Invalid("query.cursor", "Invalid cursor.")
		}
		off = n
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.Q.ListNotifications(ctx, store.ListNotificationsParams{
		UserID: userID, WorkspaceID: wsID, Kinds: kinds, UnreadOnly: unreadOnly, Off: int32(off), Lim: int32(limit),
	})
	if err != nil {
		return NotificationList{}, err
	}
	counts, err := s.Q.CountNotifications(ctx, store.CountNotificationsParams{UserID: userID, WorkspaceID: wsID, Kinds: kinds, UnreadOnly: unreadOnly})
	if err != nil {
		return NotificationList{}, err
	}
	unread, err := s.Q.CountUnreadNotifications(ctx, store.CountUnreadNotificationsParams{UserID: userID, WorkspaceID: wsID})
	if err != nil {
		return NotificationList{}, err
	}
	out := NotificationList{Items: make([]NotificationItem, 0, len(rows)), Unread: unread, Total: counts.Total, RetentionDays: RetentionDays}
	for _, n := range rows {
		out.Items = append(out.Items, item(n))
	}
	if int64(off+len(rows)) < counts.Total {
		out.NextCursor = strconv.Itoa(off + len(rows))
	}
	return out, nil
}

func item(n store.Notification) NotificationItem {
	c := CategoryOf(n.Kind)
	summary := n.Summary
	if summary == "" {
		summary = n.Body
	}
	return NotificationItem{
		ID: n.ID.String(), Kind: n.Kind, Category: c, CategoryLabel: CategoryLabel(c), Severity: n.Severity,
		Title: n.Title, Summary: summary, Body: n.Body, Link: n.Link, CreatedAt: n.CreatedAt, Read: n.ReadAt != nil,
	}
}

// MarkRead marks one notification as read (opening it from the list).
// Read state is personal UI state, like a session's last-seen time, so it
// is not written to the workspace audit log.
func (s *Service) MarkRead(ctx context.Context, id uuid.UUID) error {
	userID, wsID, err := userAndWorkspace(ctx)
	if err != nil {
		return err
	}
	_, err = s.Q.MarkNotificationRead(ctx, store.MarkNotificationReadParams{ID: id, UserID: userID, WorkspaceID: wsID})
	return err
}

// MarkAllRead marks every notification of the workspace as read.
func (s *Service) MarkAllRead(ctx context.Context) (int64, error) {
	userID, wsID, err := userAndWorkspace(ctx)
	if err != nil {
		return 0, err
	}
	return s.Q.MarkAllNotificationsRead(ctx, store.MarkAllNotificationsReadParams{UserID: userID, WorkspaceID: wsID})
}

// NotificationEventPreference is one row of the preferences table.
type NotificationEventPreference struct {
	Type     string `json:"type"`
	Label    string `json:"label"`
	Audience string `json:"audience,omitempty"`
	InApp    bool   `json:"inApp"`
	Email    bool   `json:"email"`
}

// NotificationQuietHours hold back non-urgent emails, in the person's timezone.
type NotificationQuietHours struct {
	Start string `json:"start" pattern:"^([01][0-9]|2[0-3]):[0-5][0-9]$" example:"22:00"`
	End   string `json:"end" pattern:"^([01][0-9]|2[0-3]):[0-5][0-9]$" example:"07:00"`
}

// NotificationEmailDelivery says whether notification emails can go out right now.
type NotificationEmailDelivery struct {
	Available bool `json:"available"`
	// Provider is the sending provider of the System stream ("Postmark").
	Provider string `json:"provider,omitempty"`
}

// NotificationPreferences are the person's notification settings.
type NotificationPreferences struct {
	Events     []NotificationEventPreference `json:"events"`
	QuietHours *NotificationQuietHours       `json:"quietHours,omitempty"`
	Digest     string                        `json:"digest" enum:"immediate,hourly,daily"`
	Timezone   string                        `json:"timezone"`
	Email      NotificationEmailDelivery     `json:"email"`
}

// NotificationEventChoice turns one event type's channels on or off.
type NotificationEventChoice struct {
	Type  string `json:"type" maxLength:"64"`
	InApp bool   `json:"inApp"`
	Email bool   `json:"email"`
}

// NotificationPreferencesInput is PUT /me/notification-preferences.
type NotificationPreferencesInput struct {
	Events     []NotificationEventChoice `json:"events" maxItems:"20"`
	QuietHours *NotificationQuietHours   `json:"quietHours,omitempty" doc:"Omit or null to turn quiet hours off"`
	Digest     string                    `json:"digest" enum:"immediate,hourly,daily"`
}

// Preferences returns a person's notification settings. workspaceID, when
// set, names the workspace whose System-stream provider is shown.
func (s *Service) Preferences(ctx context.Context, user store.User, workspaceID *uuid.UUID) (NotificationPreferences, error) {
	rows, err := s.Q.ListNotificationPreferences(ctx, user.ID)
	if err != nil {
		return NotificationPreferences{}, err
	}
	saved := map[string]store.NotificationPreference{}
	for _, r := range rows {
		saved[r.EventType] = r
	}
	out := NotificationPreferences{Events: make([]NotificationEventPreference, 0, len(Events)), Digest: "immediate", Timezone: user.Timezone}
	for _, e := range Events {
		p := NotificationEventPreference{Type: e.Key, Label: e.Label, Audience: e.Audience, InApp: true, Email: e.DefaultEmail}
		if r, ok := saved[e.Key]; ok {
			p.InApp, p.Email = r.InApp, r.Email
		}
		out.Events = append(out.Events, p)
	}
	st, err := s.Q.GetNotificationSettings(ctx, user.ID)
	switch {
	case err == nil:
		out.Digest = st.Digest
		if st.QuietStart != nil && st.QuietEnd != nil {
			out.QuietHours = &NotificationQuietHours{Start: hhmm(int(*st.QuietStart)), End: hhmm(int(*st.QuietEnd))}
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return NotificationPreferences{}, err
	}
	if s.Mail != nil {
		out.Email.Available = s.Mail().SystemAvailable(ctx, workspaceID)
	}
	if workspaceID != nil {
		if name, err := s.Q.SystemStreamProviderName(ctx, *workspaceID); err == nil {
			out.Email.Provider = name
			if s.Mail == nil {
				out.Email.Available = true
			}
		}
	}
	return out, nil
}

// SavePreferences stores the person's notification settings.
func (s *Service) SavePreferences(ctx context.Context, user store.User, in NotificationPreferencesInput) error {
	known := map[string]bool{}
	for _, e := range Events {
		known[e.Key] = true
	}
	for i, e := range in.Events {
		if !known[e.Type] {
			return api.Invalid(fmt.Sprintf("body.events[%d].type", i), "Unknown event type "+e.Type+".")
		}
	}
	var qs, qe *int16
	if in.QuietHours != nil {
		a, err1 := minutes(in.QuietHours.Start)
		b, err2 := minutes(in.QuietHours.End)
		if err1 != nil || err2 != nil {
			return api.Invalid("body.quietHours", "Use times like 22:00 and 07:00.")
		}
		if a == b {
			return api.Invalid("body.quietHours.end", "Quiet hours need a different start and end time.")
		}
		qs, qe = &a, &b
	}
	before, _ := s.Preferences(ctx, user, nil)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, e := range in.Events {
			if err := q.UpsertNotificationPreference(ctx, store.UpsertNotificationPreferenceParams{UserID: user.ID, EventType: e.Type, InApp: e.InApp, Email: e.Email}); err != nil {
				return err
			}
		}
		if err := q.UpsertNotificationSettings(ctx, store.UpsertNotificationSettingsParams{UserID: user.ID, QuietStart: qs, QuietEnd: qe, Digest: in.Digest}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "account.notification_preferences_update", TargetType: "user", TargetID: user.ID.String(), TargetName: user.Email,
			Before: map[string]any{"events": before.Events, "quiet_hours": before.QuietHours, "digest": before.Digest},
			After:  map[string]any{"events": in.Events, "quiet_hours": in.QuietHours, "digest": in.Digest}})
	})
}

func hhmm(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

func minutes(s string) (int16, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, err
	}
	return int16(t.Hour()*60 + t.Minute()), nil
}

// NotificationSummary is the Notification preferences card in Account & security.
type NotificationSummary struct {
	InApp      string `json:"inApp" example:"All events"`
	Email      string `json:"email" example:"Approvals, auto-pauses, provider down"`
	QuietHours string `json:"quietHours" example:"22:00–07:00 America/New_York, email only"`
}

// Summarize describes preferences in one line per channel.
func Summarize(p NotificationPreferences) NotificationSummary {
	pick := func(on func(NotificationEventPreference) bool) string {
		var all, approvals = true, 0
		var parts []string
		for _, e := range p.Events {
			if !on(e) {
				all = false
				continue
			}
			switch e.Type {
			case KindApprovalRequested, KindApprovalDecided, KindApprovalInvalidated:
				approvals++
			}
		}
		if all {
			return "All events"
		}
		if approvals == 3 {
			parts = append(parts, "approvals")
		}
		for _, e := range p.Events {
			if !on(e) {
				continue
			}
			isApproval := e.Type == KindApprovalRequested || e.Type == KindApprovalDecided || e.Type == KindApprovalInvalidated
			if isApproval && approvals == 3 {
				continue
			}
			parts = append(parts, EventFor(e.Type).Short)
		}
		if len(parts) == 0 {
			return "Off"
		}
		s := strings.Join(parts, ", ")
		return strings.ToUpper(s[:1]) + s[1:]
	}
	out := NotificationSummary{
		InApp: pick(func(e NotificationEventPreference) bool { return e.InApp }),
		Email: pick(func(e NotificationEventPreference) bool { return e.Email }),
	}
	if p.QuietHours != nil {
		out.QuietHours = fmt.Sprintf("%s–%s %s, email only", p.QuietHours.Start, p.QuietHours.End, p.Timezone)
	} else {
		out.QuietHours = "Off"
	}
	return out
}
