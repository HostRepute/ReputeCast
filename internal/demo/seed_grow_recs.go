package demo

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Recommendations and partner links of the Grow sample (brief §4): Acme
// recommends Ops Weekly and The Deliverability Desk (partners on other
// instances; the Desk's link invalid since 13 Oct 08:12) and Fabrikam
// Engineering Letter (a link); Ops Weekly recommends Acme in return.
// Partner instances are fictional: nothing is ever sent to them.

// growSecret is a partner link's sealed signing secret.
func (w *growWorld) growSecret(link uuid.UUID) ([]byte, error) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return w.s.Box.SealString("rcps_"+base64.RawURLEncoding.EncodeToString(b), "partner_link:"+link.String()+":secret")
}

// spread splits total over n days, deterministically.
func (w *growWorld) spread(total, n int) []int {
	out := make([]int, n)
	if n <= 0 {
		return out
	}
	for i := 0; i < total; i++ {
		out[w.rng.IntN(n)]++
	}
	return out
}

func (w *growWorld) seedRecommendations() error {
	marcus, sofia := w.marcus, w.sofia
	if err := w.exec(`INSERT INTO recommendation_settings (workspace_id, confirm_page, profile_name, profile_description, profile_list_id, updated_by, updated_at)
		VALUES ($1, true, 'Acme Weekly Digest', 'What shipped at Acme, what''s next, and a deep dive every Tuesday.', $2, $3, $4)`,
		w.ws.ID, w.digest, marcus, w.day(23, 11, 0)); err != nil {
		return err
	}
	opsAt, deskAt := w.day(23, 11, 20), w.day(20, 14, 5)
	// Acme's code for Ops Weekly (used when they connected), and the two
	// links Acme pasted codes for.
	opsIn, opsOut, deskOut, codeID := ID(), ID(), ID(), ID()
	codeHash := sha256.Sum256([]byte("reputecast-partner-code:demo-acme-for-ops"))
	if err := w.exec(`INSERT INTO partner_codes (id, workspace_id, code_hash, label, list_id, daily_limit, created_by, created_at, expires_at)
		VALUES ($1, $2, $3, 'Ops Weekly', $4, 50, $5, $6, $7)`, codeID, w.ws.ID, codeHash[:], w.digest, marcus, opsAt.Add(-10*time.Minute), opsAt.Add(7*24*time.Hour)); err != nil {
		return err
	}
	type link struct {
		id               uuid.UUID
		dir, url, remote string
		name, desc, list string
		limit            int
		label            string
		code             *uuid.UUID
		listID           *uuid.UUID
		at               time.Time
		status, reason   string
		invalid          *time.Time
	}
	desk := w.day(2, 8, 12)
	links := []link{
		{opsOut, "out", "https://reputecast.opsweekly.example", ID().String(), "Ops Weekly", "Incidents, on-call and the tools behind them. Every Thursday.",
			"Ops Weekly", 50, "", nil, nil, opsAt.Add(3 * time.Minute), "connected", "", nil},
		{opsIn, "in", "https://reputecast.opsweekly.example", ID().String(), "Ops Weekly", "", "Ops Weekly", 50, "Ops Weekly", &codeID, &w.digest, opsAt,
			"connected", "", nil},
		{deskOut, "out", "https://mail.deliverabilitydesk.example", ID().String(), "The Deliverability Desk",
			"One deliverability question, answered properly, every other Monday.", "The Deliverability Desk", 50, "", nil, nil, deskAt, "invalid",
			"signature_rejected", &desk},
	}
	for _, l := range links {
		secret, err := w.growSecret(l.id)
		if err != nil {
			return err
		}
		var codeHashArg []byte
		if l.dir == "out" {
			h := sha256.Sum256([]byte("reputecast-partner-code:demo-" + l.id.String()))
			codeHashArg = h[:]
		}
		if err := w.exec(`INSERT INTO partner_links (id, workspace_id, direction, remote_url, remote_link_id, protocol_version, remote_name, remote_description,
				remote_list_name, daily_limit, list_id, label, code_id, code_hash, secret, status, invalid_since, invalid_reason, connected_at, connected_by,
				notified_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 1, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $16, $18, $18)`,
			l.id, w.ws.ID, l.dir, l.url, l.remote, l.name, l.desc, l.list, l.limit, l.listID, l.label, l.code, codeHashArg, secret, l.status, l.invalid,
			l.reason, l.at, marcus); err != nil {
			return err
		}
	}
	if err := w.exec(`UPDATE partner_codes SET used_at = $2, used_link_id = $3 WHERE id = $1`, codeID, opsAt, opsIn); err != nil {
		return err
	}
	// What Acme recommends, in order.
	type rec struct {
		id                    uuid.UUID
		kind, name, desc, url string
		link                  *uuid.UUID
		confirm, block        bool
		by                    uuid.UUID
		at                    time.Time
	}
	opsRec, deskRec, fabRec := ID(), ID(), ID()
	recs := []rec{
		{opsRec, "partner", "Ops Weekly", "Incidents, on-call and the tools behind them. Every Thursday.", "", &opsOut, true, true, marcus, opsAt.Add(3 * time.Minute)},
		{deskRec, "partner", "The Deliverability Desk", "One deliverability question, answered properly, every other Monday.", "", &deskOut, true, false, marcus, deskAt},
		{fabRec, "link", "Fabrikam Engineering Letter", "How Fabrikam's platform team ships. Monthly.", "https://fabrikam.example/newsletter", nil, true, true, sofia,
			w.day(21, 16, 40)},
	}
	for i, r := range recs {
		if err := w.exec(`INSERT INTO recommendations (id, workspace_id, kind, name, description, url, partner_link_id, show_confirm, show_block, position,
			created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)`,
			r.id, w.ws.ID, r.kind, r.name, r.desc, r.url, r.link, r.confirm, r.block, i, r.by, r.at); err != nil {
			return err
		}
	}
	// 30 days of numbers: 1,046 confirmation pages and 1,276 delivered
	// welcome emails showed Ops Weekly and Fabrikam; the Desk showed on
	// 958 confirmation pages until 13 Oct.
	var daily [][]any
	add := func(id uuid.UUID, from int, cols map[string]int) {
		n := from + 1
		series := map[string][]int{}
		for k, total := range cols {
			series[k] = w.spread(total, n)
		}
		for i := 0; i < n; i++ {
			day := dayOf(w.day(from-i, 12, 0))
			row := []any{id, day, series["shown_confirm"][i], series["shown_email"][i], series["clicks"][i], series["requests"][i], series["confirmed"][i]}
			if row[2].(int)+row[3].(int)+row[4].(int)+row[5].(int)+row[6].(int) > 0 {
				daily = append(daily, row)
			}
		}
	}
	add(opsRec, 22, map[string]int{"shown_confirm": 1046, "shown_email": 1276, "requests": 168, "confirmed": 119, "clicks": 0})
	add(fabRec, 21, map[string]int{"shown_confirm": 1046, "shown_email": 1276, "clicks": 96, "requests": 0, "confirmed": 0})
	add(deskRec, 19, map[string]int{"shown_confirm": 958, "requests": 97, "confirmed": 64, "shown_email": 0, "clicks": 0})
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"recommendation_daily"}, []string{"recommendation_id", "day", "shown_confirm", "shown_email", "clicks",
		"requests", "confirmed"}, pgx.CopyFromRows(daily)); err != nil {
		return err
	}
	// The request logs: Acme's requests to its partners (addresses cleared
	// once accepted) and Ops Weekly's requests to Acme.
	var out [][]any
	request := func(link, rec uuid.UUID, n, confirmed, fromDay, toDay int) {
		for i := 0; i < n; i++ {
			at := w.at(toDay + w.rng.IntN(fromDay-toDay+1))
			var conf *time.Time
			if i < confirmed {
				conf = tp(at.Add(time.Duration(5+w.rng.IntN(240)) * time.Minute))
				if conf.After(w.now) {
					conf = tp(w.now.Add(-time.Minute))
				}
			}
			origin := "confirm"
			if i%4 == 3 {
				origin = "page"
			}
			out = append(out, []any{ID(), w.ws.ID, link, rec, origin, "accepted", 1, at, at.Add(2 * time.Second), conf})
		}
	}
	request(opsOut, opsRec, 168, 119, 22, 0)
	request(deskOut, deskRec, 97, 64, 19, 3)
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"partner_requests_out"}, []string{"id", "workspace_id", "link_id", "recommendation_id", "origin",
		"status", "attempts", "created_at", "sent_at", "confirmed_at"}, pgx.CopyFromRows(out)); err != nil {
		return err
	}
	var in [][]any
	for i := 0; i < 143; i++ {
		at := w.at(w.rng.IntN(23))
		var contact *uuid.UUID
		var conf *time.Time
		optin := "sent"
		if i < 102 {
			if i < len(w.partnerPool) {
				c := w.partnerPool[i].id
				contact = &c
			}
			conf = tp(at.Add(time.Duration(10+w.rng.IntN(600)) * time.Minute))
			if conf.After(w.now) {
				conf = tp(w.now.Add(-2 * time.Minute))
			}
		}
		notice := ""
		var noticeAt *time.Time
		if conf != nil {
			notice, noticeAt = "sent", tp(conf.Add(time.Minute))
		}
		in = append(in, []any{ID(), w.ws.ID, opsIn, uuid.NewString(), "accepted", contact, true, optin, at.Add(time.Minute), "Ops Weekly", at, conf, notice, noticeAt})
	}
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"partner_requests_in"}, []string{"id", "workspace_id", "link_id", "request_id", "status", "contact_id",
		"asked", "optin_status", "optin_sent_at", "source_name", "received_at", "confirmed_at", "notice_status", "notice_sent_at"}, pgx.CopyFromRows(in)); err != nil {
		return err
	}
	w.recAudit = append(w.recAudit,
		growAudit{opsAt.Add(-10 * time.Minute), "partner.code_created", "partner_code", codeID.String(), "Ops Weekly", map[string]any{"label": "Ops Weekly", "list": "Weekly Digest", "dailyLimit": 50}},
		growAudit{opsAt.Add(3 * time.Minute), "partner.link_connected", "partner_link", opsOut.String(), "Ops Weekly", map[string]any{"direction": "you recommend", "instance": "reputecast.opsweekly.example"}},
		growAudit{deskAt, "partner.link_connected", "partner_link", deskOut.String(), "The Deliverability Desk", map[string]any{"direction": "you recommend", "instance": "mail.deliverabilitydesk.example"}},
	)
	return nil
}
