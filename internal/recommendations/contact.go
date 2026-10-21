package recommendations

import (
	"context"
	"html"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactEvents implements referrals.PartnerEvents: on a contact's
// timeline, the partner requests that brought them ("Asked to subscribe on
// the Acme Weekly Digest's page"). The recommending side keeps no contact
// for its requests, so only the partner side has these events.
func (s *Service) ContactEvents(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) ([]referrals.ReferralTimelineEvent, error) {
	rows, err := s.Pool.Query(ctx, `SELECT ri.received_at, pl.remote_name FROM partner_requests_in ri JOIN partner_links pl ON pl.id = ri.link_id
		WHERE ri.contact_id = $1 AND ri.workspace_id = $2 AND ri.status = 'accepted' ORDER BY ri.received_at DESC LIMIT 20`, contactID, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []referrals.ReferralTimelineEvent{}
	for rows.Next() {
		var at time.Time
		var name string
		if err := rows.Scan(&at, &name); err != nil {
			return nil, err
		}
		out = append(out, referrals.ReferralTimelineEvent{Kind: "partner_request", At: at, Partner: name})
	}
	return out, rows.Err()
}

// ServeCodePage answers a partner code opened in a browser: it's meant to
// be pasted in another ReputeCast, not visited.
func (s *Service) ServeCodePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	host := html.EscapeString(hostOf(s.instance(), false))
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<meta name="robots" content="noindex, nofollow"><title>Partner code</title></head>` +
		`<body style="margin:0;padding:48px 16px;background:#F3F4F6;font-family:'IBM Plex Sans',Arial,sans-serif;color:#0B1220">` +
		`<main style="max-width:520px;margin:0 auto;background:#fff;border:1px solid #E3E6EB;border-radius:14px;padding:28px">` +
		`<h1 style="margin:0 0 12px;font-size:20px">This is a partner code</h1>` +
		`<p style="margin:0 0 12px;font-size:15px;line-height:1.55">It connects a newsletter on ` + host + ` with another ReputeCast instance. ` +
		`Paste the whole link in your ReputeCast under Grow › Recommendations › Add recommendation › “Yes, I have a partner code”.</p>` +
		`<p style="margin:0;font-size:13px;line-height:1.5;color:#5B6472">A code works once and expires after 7 days. Opening it here doesn't use it.</p>` +
		`</main></body></html>`))
}
