package agency_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// The Agency area's bell and its incidents read in the reader's language:
// what a notification and an incident kept for later readers, written
// while their workspace read English. Text that wasn't kept stays English.
func TestBellAndIncidentsInVietnamese(t *testing.T) {
	env, p := northwind(t)
	ctx := context.Background()
	ws := workspaceID(t, getOverview(t, p["tomas"]), "litware-books")
	var tomas uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'tomas@northwind.studio'`).Scan(&tomas); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, body, link, data)
		VALUES ($1, $2, $3, 'webhooks_silent', 'warn', 'Mailgun webhooks have been silent for 6 hours', 'Bounce protection for this route is blind.', '/w/litware-books/providers',
		'{"i18n": {"title": {"id": "providers.manage.notify.silentTitle", "args": {"name": "Mailgun"}}}}')`, uuid.Must(uuid.NewV7()), tomas, ws); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO incidents (id, workspace_id, kind, severity, title, texts)
		VALUES ($1, $2, 'webhooks_silent', 'warn', 'Mailgun: no webhook events since 09:30 while mail went out.',
		'{"title": {"id": "providers.manage.incident.silent", "args": {"name": "Mailgun", "since": "2026-10-20T09:30:00Z"}}}')`, uuid.Must(uuid.NewV7()), ws); err != nil {
		t.Fatal(err)
	}
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("tomas@northwind.studio")

	type bell struct {
		Items []struct{ Kind, Title, Summary string }
	}
	var b bell
	vi.Do("GET", "/agency/notifications?limit=50", nil).MustStatus(t, 200).JSON(t, &b)
	found := false
	for _, n := range b.Items {
		if n.Kind == "webhooks_silent" && n.Title == "Webhook của Mailgun đã im lặng 6 giờ" && n.Summary == "Bounce protection for this route is blind." {
			found = true
		}
	}
	if !found {
		t.Errorf("bell: %+v", b.Items)
	}
	var en bell
	p["tomas"].Do("GET", "/agency/notifications?limit=50", nil).MustStatus(t, 200).JSON(t, &en)
	found = false
	for _, n := range en.Items {
		if n.Kind == "webhooks_silent" && n.Title == "Mailgun webhooks have been silent for 6 hours" {
			found = true
		}
	}
	if !found {
		t.Errorf("bell in English: %+v", en.Items)
	}

	type incidents struct {
		Incidents []struct{ Kind, Title string }
	}
	var o incidents
	vi.Do("GET", "/agency/overview", nil).MustStatus(t, 200).JSON(t, &o)
	found = false
	for _, i := range o.Incidents {
		if i.Kind == "webhooks_silent" && i.Title == "Mailgun: không có sự kiện webhook nào từ 09:30 trong khi thư vẫn được gửi đi." {
			found = true
		}
	}
	if !found {
		t.Errorf("incidents: %+v", o.Incidents)
	}
}
