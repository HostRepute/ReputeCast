package workspaces_test

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

type settings struct {
	Name, Timezone, DefaultSenderID string
	Senders                         []struct {
		ID, Email  string
		VerifiedOn []string
	}
	Approvals struct {
		Rules     []string
		Approvers []string
	}
	Members struct {
		Active, Invited, Deactivated int64
		SignIn                       string
		Custom                       []string
	}
	Danger struct {
		Lists, Campaigns, Providers, APIKeys, Agents int64
	}
	Candidates []struct {
		MemberID, Name string
		Eligible       bool
		Note           string
	}
	CanManage, CanTransfer, CanDelete bool
	Reason                            string
}

func senderID(t *testing.T, s settings, email string) string {
	t.Helper()
	for _, sn := range s.Senders {
		if sn.Email == email {
			return sn.ID
		}
	}
	t.Fatalf("no sender %s in %+v", email, s.Senders)
	return ""
}

func TestGeneralSettings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var s settings
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	// Other seeders add senders too, so they are found by address, not counted.
	digest, product := senderID(t, s, "digest@news.acme.io"), senderID(t, s, "product@news.acme.io")
	if s.Name != "Acme Newsletter" || s.DefaultSenderID != digest {
		t.Fatalf("general: %+v", s)
	}
	for _, sn := range s.Senders {
		if sn.ID == digest && strings.Join(sn.VerifiedOn, ",") != "SES,Postal" {
			t.Fatalf("digest@ verified on %v", sn.VerifiedOn)
		}
	}
	if strings.Join(s.Approvals.Rules, " · ") != "audience ≥ 10,000 · route includes a Warming-up provider · author lacks campaigns:send" ||
		strings.Join(s.Approvals.Approvers, ", ") != "Linh Nguyen, Marcus Chen" {
		t.Fatalf("approvals: %+v", s.Approvals)
	}
	if s.Members.Active != 5 || s.Members.Invited != 1 || s.Members.Deactivated != 1 ||
		s.Members.SignIn != "Google Workspace SSO enforced for acme.io · password kept for the Owner" || len(s.Members.Custom) != 1 {
		t.Fatalf("members: %+v", s.Members)
	}
	if !s.CanManage || !s.CanTransfer || !s.CanDelete {
		t.Fatal("owner can do everything")
	}
	c.Do("PUT", ws+"/settings", map[string]any{"name": "Acme Weekly", "timezone": "Europe/Berlin", "defaultSenderId": product}).MustStatus(t, 204)
	c.Do("PUT", ws+"/settings", map[string]any{"name": "Acme Weekly", "timezone": "Mars/Olympus"}).MustStatus(t, 422)
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	if s.Name != "Acme Weekly" || s.Timezone != "Europe/Berlin" || s.DefaultSenderID != product {
		t.Fatalf("saved: %+v", s)
	}
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	if s.CanManage || !strings.Contains(s.Reason, "workspace:manage") {
		t.Fatalf("viewer: %+v", s)
	}
	v.Do("PUT", ws+"/settings", map[string]any{"name": "Nope", "timezone": "UTC"}).MustStatus(t, 403)
}

func TestTransferDeleteExport(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	// Linh signs in with her real password so re-authentication can be checked.
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var s settings
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	var marcus, aisha string
	for _, cand := range s.Candidates {
		switch cand.Name {
		case "Marcus Chen":
			marcus = cand.MemberID
			if !cand.Eligible || !strings.Contains(cand.Note, "Google Workspace") {
				t.Fatalf("marcus: %+v", cand)
			}
		case "Aisha Bello":
			aisha = cand.MemberID
			if cand.Eligible {
				t.Fatal("aisha has no two-factor")
			}
		}
	}

	// Export everything first: a ZIP with contacts, consent, campaigns and the audit log.
	r := c.Do("GET", ws+"/export", nil).MustStatus(t, 200)
	zr, err := zip.NewReader(bytes.NewReader(r.Body), int64(len(r.Body)))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"README.txt", "contacts.csv", "consent.csv", "lists.csv", "campaigns.csv", "reports.csv", "audit-log.json"} {
		if !names[want] {
			t.Fatalf("export lacks %s: %v", want, names)
		}
	}

	c.Do("POST", ws+"/transfer", map[string]any{"memberId": marcus, "password": "wrong"}).MustStatus(t, 422)
	c.Do("POST", ws+"/transfer", map[string]any{"memberId": aisha, "password": "reputecast-demo"}).MustStatus(t, 409)
	c.Do("POST", ws+"/transfer", map[string]any{"memberId": marcus, "password": "reputecast-demo"}).MustStatus(t, 204)
	var role string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT r.key FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id JOIN workspaces w ON w.id = m.workspace_id
		WHERE u.email = 'linh@acme.io' AND w.slug = 'acme-newsletter'`).Scan(&role)
	if role != "admin" {
		t.Fatalf("linh is now %s", role)
	}
	// Only the Owner deletes; Linh is an Admin now.
	c.Do("POST", ws+"/delete", map[string]any{"confirm": "Acme Newsletter"}).MustStatus(t, 403)
	m := env.Client()
	m.SignInDemo("marcus@acme.io")
	m.Do("POST", ws+"/delete", map[string]any{"confirm": "Acme News"}).MustStatus(t, 422)
	var del struct{ Cancelled int64 }
	m.Do("POST", ws+"/delete", map[string]any{"confirm": "Acme Newsletter"}).MustStatus(t, 200).JSON(t, &del)
	if del.Cancelled != 1 {
		t.Fatalf("scheduled sends stop: %+v", del)
	}
	m.Do("GET", ws+"/settings", nil).MustStatus(t, 404)
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('workspace.transferred','workspace.deleted','workspace.exported')`).Scan(&n)
	if n != 3 {
		t.Fatalf("audit: %d", n)
	}
}
