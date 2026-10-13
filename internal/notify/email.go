package notify

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// EmailArgs is the periodic job that emails notifications to the people who
// asked for email.
type EmailArgs struct{}

// Kind implements river.JobArgs.
func (EmailArgs) Kind() string { return "notify.email" }

// InsertOpts keeps one delivery run at a time on the System queue.
func (EmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Second}}
}

// maxEmailAttempts before a notification email is given up.
const maxEmailAttempts = 5

// EmailWorker delivers pending notification emails through the System
// stream. Urgent events (auto-pauses, provider outages) go out at once;
// others respect the person's quiet hours and digest setting. Without a
// System-stream provider the emails are skipped silently: the in-app
// notification is already there.
type EmailWorker struct {
	river.WorkerDefaults[EmailArgs]
	Pool *pgxpool.Pool
	// Mail is read at run time: the system mailer is wired after the workers.
	Mail func() mailer.Sender
	// PublicURL returns the instance's public URL for links in emails.
	PublicURL func(context.Context) string
	Log       *slog.Logger
	Now       func() time.Time
}

func (w *EmailWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// Work implements river.Worker.
func (w *EmailWorker) Work(ctx context.Context, _ *river.Job[EmailArgs]) error {
	q := store.New(w.Pool)
	users, err := q.ListUsersWithPendingEmail(ctx)
	if err != nil {
		return err
	}
	for _, id := range users {
		if err := w.deliver(ctx, q, id); err != nil {
			w.Log.WarnContext(ctx, "notification email failed", "user", id, "err", err)
		}
	}
	return nil
}

func (w *EmailWorker) deliver(ctx context.Context, q *store.Queries, userID uuid.UUID) error {
	u, err := q.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	pending, err := q.ListPendingEmailNotifications(ctx, userID)
	if err != nil || len(pending) == 0 {
		return err
	}
	if u.Status != "active" {
		return q.SetNotificationsEmailStatus(ctx, store.SetNotificationsEmailStatusParams{EmailStatus: "skipped", Ids: ids(pending)})
	}
	st, err := q.GetNotificationSettings(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := w.now()
	local := now.In(loc)
	quiet := st.QuietStart != nil && st.QuietEnd != nil && inQuietHours(local, int(*st.QuietStart), int(*st.QuietEnd))
	digest := st.Digest
	if digest == "" {
		digest = "immediate"
	}

	var urgent, normal []store.ListPendingEmailNotificationsRow
	for _, n := range pending {
		if EventFor(n.Kind).Urgent {
			urgent = append(urgent, n)
		} else {
			normal = append(normal, n)
		}
	}
	for _, n := range urgent {
		w.sendOne(ctx, q, u, n)
	}
	if len(normal) == 0 || quiet {
		return nil
	}
	switch digest {
	case "hourly":
		if now.Sub(normal[0].CreatedAt) < time.Hour {
			return nil
		}
	case "daily":
		if local.Hour() < 9 || (st.LastDigestAt != nil && sameDay(st.LastDigestAt.In(loc), local)) {
			return nil
		}
	default:
		for _, n := range normal {
			w.sendOne(ctx, q, u, n)
		}
		return nil
	}
	// One digest per workspace (each goes out on that workspace's System stream).
	byWS := map[string][]store.ListPendingEmailNotificationsRow{}
	var order []string
	for _, n := range normal {
		key := n.WorkspaceSlug
		if _, ok := byWS[key]; !ok {
			order = append(order, key)
		}
		byWS[key] = append(byWS[key], n)
	}
	sort.Strings(order)
	for _, key := range order {
		w.sendDigest(ctx, q, u, byWS[key])
	}
	return q.SetLastDigestAt(ctx, userID)
}

func inQuietHours(t time.Time, start, end int) bool {
	m := t.Hour()*60 + t.Minute()
	if start < end {
		return m >= start && m < end
	}
	return m >= start || m < end // over midnight (22:00–07:00)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func ids(rows []store.ListPendingEmailNotificationsRow) []uuid.UUID {
	out := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func (w *EmailWorker) base(ctx context.Context) string {
	if w.PublicURL == nil {
		return ""
	}
	return strings.TrimRight(w.PublicURL(ctx), "/")
}

func (w *EmailWorker) sendOne(ctx context.Context, q *store.Queries, u store.User, n store.ListPendingEmailNotificationsRow) {
	base := w.base(ctx)
	subject := n.Title
	if n.WorkspaceName != "" {
		subject = "[" + n.WorkspaceName + "] " + n.Title
	}
	var text, body strings.Builder
	fmt.Fprintf(&text, "%s\n\n%s\n", n.Title, n.Body)
	fmt.Fprintf(&body, "<p style=\"font:600 16px sans-serif;margin:0 0 8px\">%s</p><p style=\"font:14px/1.5 sans-serif;margin:0 0 16px\">%s</p>", html.EscapeString(n.Title), html.EscapeString(n.Body))
	if n.Link != "" {
		fmt.Fprintf(&text, "\nOpen: %s%s\n", base, n.Link)
		fmt.Fprintf(&body, "<p style=\"font:14px sans-serif\"><a href=\"%s%s\">Open in ReputeCast</a></p>", html.EscapeString(base), html.EscapeString(n.Link))
	}
	w.footer(&text, &body, base, n.WorkspaceSlug)
	w.send(ctx, q, u, n.WorkspaceID, subject, text.String(), body.String(), []uuid.UUID{n.ID})
}

func (w *EmailWorker) sendDigest(ctx context.Context, q *store.Queries, u store.User, rows []store.ListPendingEmailNotificationsRow) {
	base := w.base(ctx)
	ws := rows[0].WorkspaceName
	subject := fmt.Sprintf("%d notifications", len(rows))
	if len(rows) == 1 {
		subject = rows[0].Title
	}
	if ws != "" {
		subject = "[" + ws + "] " + subject
	}
	var text, body strings.Builder
	for _, n := range rows {
		fmt.Fprintf(&text, "• %s: %s\n", n.Title, n.Body)
		fmt.Fprintf(&body, "<p style=\"font:14px/1.5 sans-serif;margin:0 0 12px\"><b>%s</b><br>%s", html.EscapeString(n.Title), html.EscapeString(n.Body))
		if n.Link != "" {
			fmt.Fprintf(&text, "  %s%s\n", base, n.Link)
			fmt.Fprintf(&body, "<br><a href=\"%s%s\">Open</a>", html.EscapeString(base), html.EscapeString(n.Link))
		}
		body.WriteString("</p>")
	}
	w.footer(&text, &body, base, rows[0].WorkspaceSlug)
	w.send(ctx, q, u, rows[0].WorkspaceID, subject, text.String(), body.String(), ids(rows))
}

func (w *EmailWorker) footer(text, body *strings.Builder, base, slug string) {
	prefs := base + "/"
	if slug != "" {
		prefs = base + "/w/" + slug + "/notifications#preferences"
	}
	fmt.Fprintf(text, "\n—\nYou get this email because of your notification preferences: %s\n", prefs)
	fmt.Fprintf(body, "<p style=\"font:12px sans-serif;color:#5B6472;margin-top:24px\">You get this email because of your <a href=\"%s\">notification preferences</a>.</p>", html.EscapeString(prefs))
}

func (w *EmailWorker) send(ctx context.Context, q *store.Queries, u store.User, wsID *uuid.UUID, subject, text, htmlBody string, ids []uuid.UUID) {
	var m mailer.Sender = mailer.Unavailable{}
	if w.Mail != nil {
		m = w.Mail()
	}
	_, err := m.SendSystem(ctx, mailer.Message{WorkspaceID: wsID, To: u.Email, ToName: u.Name, Subject: subject, Text: text, HTML: htmlBody, Kind: "notification"})
	status := "sent"
	switch {
	case errors.Is(err, mailer.ErrNoSystemProvider):
		status = "skipped"
	case err != nil:
		w.Log.WarnContext(ctx, "notification email not sent", "user", u.ID, "err", err)
		_ = q.BumpNotificationEmailAttempts(ctx, store.BumpNotificationEmailAttemptsParams{MaxAttempts: maxEmailAttempts, Ids: ids})
		return
	}
	if err := q.SetNotificationsEmailStatus(ctx, store.SetNotificationsEmailStatusParams{EmailStatus: status, Ids: ids}); err != nil {
		w.Log.WarnContext(ctx, "notification email status", "err", err)
	}
}

// CleanupArgs deletes notifications past retention.
type CleanupArgs struct{}

// Kind implements river.JobArgs.
func (CleanupArgs) Kind() string { return "notify.cleanup" }

// InsertOpts runs cleanup on the maintenance queue.
func (CleanupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// CleanupWorker keeps notifications for RetentionDays.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupArgs]
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// Work implements river.Worker.
func (w *CleanupWorker) Work(ctx context.Context, _ *river.Job[CleanupArgs]) error {
	n, err := store.New(w.Pool).DeleteOldNotifications(ctx, RetentionDays)
	if err == nil && n > 0 {
		w.Log.InfoContext(ctx, "notifications past retention deleted", "count", n)
	}
	return err
}
