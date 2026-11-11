package deliverability_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
)

// M58 G-19: reports that arrive by email are stored "inbound" (only for the
// workspace's own domains) and the DMARC card says where they arrive, with
// the external destination record each sending domain under another
// organizational domain needs, as the checks found it.
func TestReportsCollectedByEmail(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	// Without HostRepute the summary reads the workspace's own reports.
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}

	route := "inbound.acmemail.net"
	zone := dnscheck.NewZone()
	zone.Add("news.acme.io._report._dmarc."+route, "TXT", "v=DMARC1")
	s := &deliverability.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DNS: zone,
		HostRepute: &hostrepute.Service{},
		Inbound: func(context.Context, uuid.UUID) (bool, string, *time.Time, error) {
			return true, route, nil, nil
		}}

	// Before the first check the records' state is unknown.
	o, err := s.Overview(ctx, &ws)
	if err != nil {
		t.Fatal(err)
	}
	col := o.DMARC.Collect
	if col == nil || col.Address != "dmarc@"+route || len(col.ExternalRecords) == 0 || col.ExternalRecords[0].Status != "unknown" {
		t.Fatalf("collect before the checks: %+v", col)
	}

	if err := s.RunChecks(ctx, deliverability.CheckArgs{WorkspaceID: ws.ID}); err != nil {
		t.Fatal(err)
	}
	end := time.Now().Unix()
	rep, err := deliverability.ParseReport([]byte(fmt.Sprintf(sampleReport, end-86400, end)))
	if err != nil {
		t.Fatal(err)
	}
	store := func(rep hostrepute.DMARCReport) error {
		return db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error { return s.StoreReport(ctx, tx, &ws, rep, "inbound") })
	}
	inbound := countRows(t, env, `SELECT count(*) FROM dmarc_reports WHERE via = 'inbound'`)
	if err := store(rep); err != nil {
		t.Fatal(err)
	}
	foreign := rep
	foreign.Domain, foreign.ReportID = "example.org", "fm-43"
	if err := store(foreign); !errors.Is(err, deliverability.ErrNotOwnedDomain) {
		t.Fatalf("a report about another domain: %v", err)
	}
	if n := countRows(t, env, `SELECT count(*) FROM dmarc_reports WHERE via = 'inbound'`); n != inbound+1 {
		t.Fatalf("%d inbound reports, %d before", n, inbound)
	}
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'dmarc_report.upload'`); n != 0 {
		t.Fatalf("a received report is audited as an upload: %d", n)
	}

	if o, err = s.Overview(ctx, &ws); err != nil {
		t.Fatal(err)
	}
	if o.DMARC.Via != "inbound" || o.DMARC.Reports == 0 {
		t.Fatalf("summary: %+v", o.DMARC)
	}
	col = o.DMARC.Collect
	domains := countRows(t, env, `SELECT count(*) FROM sending_domains WHERE workspace_id = $1`, ws.ID)
	if col == nil || col.LastReceivedAt == nil || int64(len(col.ExternalRecords)) != domains {
		t.Fatalf("collect: %+v (%d sending domains)", col, domains)
	}
	for _, r := range col.ExternalRecords {
		want := "missing"
		if r.Domain == "news.acme.io" {
			want = "found"
		}
		if r.Name != r.Domain+"._report._dmarc."+route || r.Type != "TXT" || r.Value != "v=DMARC1" || r.Status != want {
			t.Errorf("record: %+v", r)
		}
	}

	// A receiving domain under the sending domains' own organizational
	// domain needs no record; not receiving, no collect at all.
	route = "reply.news.acme.io"
	if o, err = s.Overview(ctx, &ws); err != nil {
		t.Fatal(err)
	}
	if col = o.DMARC.Collect; col == nil || col.Address != "dmarc@reply.news.acme.io" || int64(len(col.ExternalRecords)) != countRows(t, env,
		`SELECT count(*) FROM sending_domains WHERE workspace_id = $1 AND domain NOT LIKE '%acme.io'`, ws.ID) {
		t.Fatalf("same organizational domain: %+v", col)
	}
	s.Inbound = nil
	if o, err = s.Overview(ctx, &ws); err != nil || o.DMARC.Collect != nil {
		t.Fatalf("not receiving: %+v %v", o.DMARC.Collect, err)
	}
	if o.DMARC.Via != "inbound" {
		t.Fatalf("via without receiving: %s", o.DMARC.Via)
	}
}

func countRows(t *testing.T, env *apptest.Env, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
