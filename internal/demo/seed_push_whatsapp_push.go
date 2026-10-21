package demo

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	mrand "math/rand/v2"
	"net/netip"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/push/webpush"
)

// The push part of the X22 seed: keys, settings, sites, prompts, the
// subscribers and the campaign "New post: Feature flags at scale".

// pushBrowsers are the browsers in the order the app lists them.
var pushBrowsers = []string{"chrome", "chrome_android", "edge", "firefox", "safari", "safari_ios"}

// pushUA is a browser's User-Agent, for the consent records.
var pushUA = map[string]string{
	"chrome":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36",
	"chrome_android": "Mozilla/5.0 (Linux; Android 15; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36",
	"edge":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0",
	"firefox":        "Mozilla/5.0 (Macintosh; Intel Mac OS X 15.6; rv:143.0) Gecko/20100101 Firefox/143.0",
	"safari":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
	"safari_ios":     "Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1",
}

var pushPlatform = map[string]string{"chrome": "Windows", "chrome_android": "Android", "edge": "Windows", "firefox": "macOS", "safari": "macOS", "safari_ios": "iOS"}

// pushEndpoint is a demo subscription's push resource on its browser's
// push service (the in-process one answers for all of them).
func pushEndpoint(browser string, rng *mrand.Rand) string {
	b := make([]byte, 20)
	for i := range b {
		b[i] = byte(rng.IntN(256))
	}
	id := "demo-" + hex.EncodeToString(b)
	switch browser {
	case "chrome", "chrome_android":
		return "https://fcm.googleapis.com/fcm/send/" + id
	case "edge":
		return "https://wns2-par02p.notify.windows.com/w/?token=" + id
	case "firefox":
		return "https://updates.push.services.mozilla.com/wpush/v2/" + id
	}
	return "https://web.push.apple.com/" + id
}

// pushKeys are a browser's subscription keys: a P-256 point and 16 bytes.
func pushKeys() ([]byte, []byte) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	return k.PublicKey().Bytes(), auth
}

// pwSpread splits total over weights, exactly (largest remainder).
func pwSpread(total int, weights []float64) []int {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	out := make([]int, len(weights))
	type rem struct {
		i int
		r float64
	}
	rs := make([]rem, 0, len(weights))
	given := 0
	for i, w := range weights {
		v := float64(total) * w / sum
		out[i] = int(v)
		given += out[i]
		rs = append(rs, rem{i, v - float64(out[i])})
	}
	sort.SliceStable(rs, func(a, b int) bool { return rs[a].r > rs[b].r })
	for k := 0; given < total; k++ {
		out[rs[k%len(rs)].i]++
		given++
	}
	return out
}

// pushPromptConfig is a prompt as the designer stores it.
func pushPromptConfig(style, title, message string, seconds, scroll int, pages string, paths []string, ios string) json.RawMessage {
	when := map[string]any{"seconds": map[string]any{"on": seconds > 0, "n": seconds}, "scroll": map[string]any{"on": scroll > 0, "n": scroll},
		"posts": map[string]any{"on": false, "n": 3}, "button": false}
	c := map[string]any{"style": style, "title": title, "message": message, "allow": "Allow", "notNow": "Not now", "when": when, "pages": pages,
		"paths": paths, "askAgainDays": 14, "ios": ios}
	if style == "bell" {
		c["corner"] = "bottom_right"
	}
	b, _ := json.Marshal(c)
	return b
}

const (
	newsPromptTitle   = "Get new posts as notifications"
	newsPromptMessage = "One notification when a new post is out. Never more than one a day."
	bellPromptTitle   = "Get release notes as notifications"
	bellPromptMessage = "Only when we ship something worth knowing. Turn them off any time."
)

// pushSetup: the keys (3 Oct, Marcus), the cap, the sites and the prompts.
func pushSetup(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	marcus, sofia := s.Users["marcus@acme.io"], s.Users["sofia@acme.io"]
	keysAt := s.pushDay(3, 10, 14)
	k, err := webpush.GenerateVAPIDKey()
	if err != nil {
		return err
	}
	st.key = ID()
	sealed, err := s.Box.Seal(k.PrivateBytes(), "push_key:"+st.key.String()+":private")
	if err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_keys (id, workspace_id, public_key, private_enc, contact, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		st.key, ws.ID, k.PublicKeyString(), sealed, "mailto:support@acme.io", marcus.ID, keysAt); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_settings (workspace_id, frequency_cap, quiet_hours, updated_by, updated_at) VALUES ($1, 2, true, $2, $3)`,
		ws.ID, marcus.ID, keysAt); err != nil {
		return err
	}
	// news.acme.io: the publication website, its worker built in.
	st.news, st.acmeSite = ID(), ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_sites (id, workspace_id, origin, host, kind, label, check_status, icon_media_id, demo, added_by, added_at)
		VALUES ($1,$2,'https://news.acme.io','news.acme.io','publication','Publication site','unchecked',$3,true,$4,$5)`,
		st.news, ws.ID, st.icon, marcus.ID, keysAt); err != nil {
		return err
	}
	// acme.io: the snippet since 5 Oct, rc-push-sw.js v1 found at 08:00.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_sites (id, workspace_id, origin, host, kind, label, worker_path, worker_mode, check_status, check_version, check_scope,
			check_http_status, checked_at, icon_media_id, snippet_seen_at, demo, added_by, added_at)
		VALUES ($1,$2,'https://acme.io','acme.io','snippet','Company site','/rc-push-sw.js','file','found',1,'/',200,$3,$4,$5,true,$6,$7)`,
		st.acmeSite, ws.ID, s.pushDay(15, 8, 0), st.icon, s.pushDay(5, 11, 26), marcus.ID, s.pushDay(5, 10, 52)); err != nil {
		return err
	}
	// The prompts: forms of the "push" type.
	st.newsPrompt, st.bellPrompt = ID(), ID()
	type version struct {
		n          int
		at         time.Time
		message    string
		seconds    int
		scroll     int
		paths      []string
		ios, style string
		title      string
	}
	prompts := []struct {
		id      uuid.UUID
		name    string
		slug    string
		site    uuid.UUID
		created time.Time
		vs      []version
	}{
		{st.newsPrompt, "Get new posts as notifications", "push-new-posts", st.news, s.pushDay(3, 10, 40), []version{
			{1, s.pushDay(3, 11, 2), "One notification when a new post is out.", 30, 50, []string{"/account"}, "steps", "bar", newsPromptTitle},
			{2, s.pushDay(6, 14, 18), "One notification when a new post is out. No more than one a day.", 30, 50, []string{"/account", "/subscribe"}, "steps", "bar",
				newsPromptTitle},
			{3, s.pushDay(8, 10, 31), newsPromptMessage, 30, 50, []string{"/account", "/subscribe"}, "steps", "bar", newsPromptTitle},
		}},
		{st.bellPrompt, "Release notes bell", "push-release-notes", st.acmeSite, s.pushDay(5, 11, 10), []version{
			{1, s.pushDay(5, 11, 34), bellPromptMessage, 0, 0, []string{}, "hidden", "bell", bellPromptTitle},
		}},
	}
	for _, p := range prompts {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, kind, slug, status, content, created_by, created_at, updated_at, published_at)
			VALUES ($1,$2,$3,'push',$4,'live','{}',$5,$6,$7,$7)`, p.id, ws.ID, p.name, p.slug, sofia.ID, p.created, p.vs[len(p.vs)-1].at); err != nil {
			return err
		}
		last := p.vs[len(p.vs)-1]
		pages := "all_except"
		if len(last.paths) == 0 && last.style == "bell" {
			pages = "all_except"
		}
		draft := pushPromptConfig(last.style, last.title, last.message, last.seconds, last.scroll, pages, last.paths, last.ios)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO push_prompts (form_id, workspace_id, site_id, draft, live_version, live_since, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, p.id, ws.ID, p.site, draft, last.n, p.vs[0].at, sofia.ID, last.at); err != nil {
			return err
		}
		for _, v := range p.vs {
			cfg := pushPromptConfig(v.style, v.title, v.message, v.seconds, v.scroll, pages, v.paths, v.ios)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO push_prompt_versions (form_id, version, config, published_by, published_at) VALUES ($1,$2,$3,$4,$5)`,
				p.id, v.n, cfg, sofia.ID, v.at); err != nil {
				return err
			}
		}
	}
	return pushPromptDays(ctx, s, st)
}

// pushPromptDays writes the prompts' numbers per day: the slide-in bar
// since 3 Oct (18,240 shown in the last 30 days), the bell since 5 Oct.
func pushPromptDays(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	rows := [][]any{}
	add := func(form uuid.UUID, first int, totals map[string]int) {
		days := 15 - first + 1
		w := make([]float64, days)
		for i := range w {
			// A launch bump, a weekly rhythm and a partial today.
			w[i] = 1 + 0.6*float64(max(0, 3-i))/3
			d := s.pushDay(first+i, 12, 0).Weekday()
			if d == time.Saturday || d == time.Sunday {
				w[i] *= 0.7
			}
			if i == days-1 {
				w[i] *= 0.45
			}
		}
		cols := []string{"shown", "allowed", "granted", "not_now", "closed", "bell_clicked", "blocked", "ios_steps", "ios_added"}
		split := map[string][]int{}
		for _, c := range cols {
			split[c] = pwSpread(totals[c], w)
		}
		for i := 0; i < days; i++ {
			t := s.pushDay(first+i, 12, 0)
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			rows = append(rows, []any{form, ws.ID, day, split["shown"][i], split["allowed"][i], split["granted"][i], split["not_now"][i], split["closed"][i],
				split["bell_clicked"][i], split["blocked"][i], split["ios_steps"][i], split["ios_added"][i]})
		}
	}
	add(st.newsPrompt, 3, map[string]int{"shown": 18240, "allowed": 4980, "granted": 4412, "not_now": 9104, "closed": 4156, "blocked": 186, "ios_steps": 1312,
		"ios_added": 141})
	add(st.bellPrompt, 5, map[string]int{"shown": 1980, "allowed": 1540, "granted": 1352, "not_now": 312, "closed": 128, "bell_clicked": 1980, "blocked": 41})
	return smsCopy(ctx, s, "push_prompt_days", []string{"form_id", "workspace_id", "day", "shown", "allowed", "granted", "not_now", "closed", "bell_clicked",
		"blocked", "ios_steps", "ios_added"}, rows)
}

// pushSub is one seeded subscription.
type pushSub struct {
	id        uuid.UUID
	site      uuid.UUID
	browser   string
	created   time.Time
	contact   *uuid.UUID
	zone      string
	linkedBy  string
	linkedAt  *time.Time
	version   int
	page      string
	ip        netip.Addr
	removed   string // gone, unsubscribed
	removedAt *time.Time
	// The campaign: whether it went to this subscription and what happened.
	recipient bool
	shown     bool
	clicked   bool
}

var pushZones = []string{"America/New_York", "America/New_York", "America/Chicago", "America/Los_Angeles", "America/Denver", "Europe/London", "Europe/Berlin",
	"Europe/Amsterdam", "America/Toronto", "Asia/Singapore", "Australia/Sydney", "Asia/Ho_Chi_Minh"}

var newsPages = []string{"https://news.acme.io/", "https://news.acme.io/posts/feature-flags-at-scale", "https://news.acme.io/posts/postgres-migration-runbook",
	"https://news.acme.io/posts/rate-limits-best-customers", "https://news.acme.io/posts/q4-roadmap-deep-dive", "https://news.acme.io/summit",
	"https://news.acme.io/posts/ask-acme-deliverability-questions"}

var acmePages = []string{"https://acme.io/blog/release-3-4", "https://acme.io/changelog", "https://acme.io/docs/flags", "https://acme.io/blog/release-3-3",
	"https://acme.io/pricing"}

// pushPeople writes the 4,264 subscribers, the 26 that ended after the
// campaign, and the campaign itself with its notifications and clicks.
func pushPeople(ctx context.Context, s *Seeder, st *pwState) error {
	rng := mrand.New(mrand.NewPCG(22, 2026)) //nolint:gosec // G404: a seeded generator, so the demo is the same every time
	chicago := smsLoc("America/Chicago")
	st.sent = s.pushDay(14, 9, 5)
	now := s.Now
	// Live subscribers per site and browser (brief §4), and what the
	// campaign did on news.acme.io: sent before the 41 that came after it,
	// 19 expired and 7 unsubscribed within 24 hours.
	newsLive := map[string]int{"chrome": 1275, "chrome_android": 373, "edge": 351, "firefox": 274, "safari": 577, "safari_ios": 96}
	acmeLive := map[string]int{"chrome": 591, "chrome_android": 171, "edge": 161, "firefox": 127, "safari": 268, "safari_ios": 0}
	newAfter := map[string]int{"safari_ios": 33, "safari": 2, "chrome": 2, "chrome_android": 2, "edge": 1, "firefox": 1}
	unsub := map[string]int{"chrome": 3, "safari": 2, "chrome_android": 1, "firefox": 1}
	gone := map[string]int{"chrome": 9, "firefox": 5, "edge": 3, "chrome_android": 2}
	shown := map[string]int{"chrome": 1248, "safari": 569, "chrome_android": 352, "edge": 341, "firefox": 232, "safari_ios": 62}
	clicks := map[string]int{"chrome": 168, "safari": 92, "chrome_android": 71, "edge": 44, "firefox": 31, "safari_ios": 15}

	subs := []*pushSub{}
	// Sam Rivera: Chrome on Android, a signed-in member, 9 Oct 08:14:52.
	samAt := time.Date(s.pushDay(9, 12, 0).Year(), s.pushDay(9, 12, 0).Month(), s.pushDay(9, 12, 0).Day(), 8, 14, 52, 0, chicago)
	sam, phone, err := pushContactByEmail(ctx, s, "sam.rivera@outlook.com")
	if err != nil {
		return err
	}
	st.sam, st.samPhone = sam, phone
	samSub := &pushSub{id: ID(), site: st.news, browser: "chrome_android", created: samAt, contact: &sam, zone: "America/Chicago", linkedBy: "member",
		linkedAt: &samAt, version: 3, page: "https://news.acme.io/posts/acme-summit-2026-agenda", ip: netip.MustParseAddr("198.51.100.77"), recipient: true, shown: true}
	subs = append(subs, samSub)
	newsStart, acmeStart := s.pushDay(3, 11, 5), s.pushDay(5, 11, 30)
	mkTime := func(from, to time.Time) time.Time {
		// More subscriptions in the first days of a prompt.
		f := rng.Float64()
		f = f * f * 0.55
		if rng.IntN(100) < 55 {
			f = rng.Float64()
		}
		return from.Add(time.Duration(f * float64(to.Sub(from))))
	}
	for _, b := range pushBrowsers {
		// news.acme.io: live before the send, the 41 after it, and the 26 that
		// ended after receiving it.
		before := newsLive[b] - newAfter[b]
		if b == "chrome_android" {
			before-- // Sam is one of them
		}
		for i := 0; i < before; i++ {
			subs = append(subs, &pushSub{id: ID(), site: st.news, browser: b, created: mkTime(newsStart, st.sent.Add(-10*time.Minute)), recipient: true})
		}
		for i := 0; i < newAfter[b]; i++ {
			subs = append(subs, &pushSub{id: ID(), site: st.news, browser: b, created: mkTime(st.sent.Add(10*time.Minute), now.Add(-20*time.Minute))})
		}
		for i := 0; i < unsub[b]; i++ {
			at := st.sent.Add(time.Duration(30+rng.IntN(20*60)) * time.Minute)
			subs = append(subs, &pushSub{id: ID(), site: st.news, browser: b, created: mkTime(newsStart, st.sent.Add(-time.Hour)), recipient: true,
				removed: "unsubscribed", removedAt: &at})
		}
		for i := 0; i < gone[b]; i++ {
			at := st.sent.Add(time.Duration(rng.IntN(120)) * time.Second)
			subs = append(subs, &pushSub{id: ID(), site: st.news, browser: b, created: mkTime(newsStart, st.sent.Add(-time.Hour)), recipient: true,
				removed: "gone", removedAt: &at})
		}
		for i := 0; i < acmeLive[b]; i++ {
			subs = append(subs, &pushSub{id: ID(), site: st.acmeSite, browser: b, created: mkTime(acmeStart, now.Add(-15*time.Minute))})
		}
	}
	// Shown and clicked, per browser, among those the push service accepted.
	for _, b := range pushBrowsers {
		acc := []*pushSub{}
		for _, p := range subs {
			if p.recipient && p.browser == b && p.removed != "gone" {
				acc = append(acc, p)
			}
		}
		rng.Shuffle(len(acc), func(i, j int) { acc[i], acc[j] = acc[j], acc[i] })
		need := shown[b]
		if samSub.browser == b {
			need-- // Sam saw it
		}
		n := 0
		for _, p := range acc {
			if p == samSub {
				continue
			}
			if n < need {
				p.shown = true
				n++
			}
		}
		c := 0
		for _, p := range acc {
			if p.shown && c < clicks[b] && p != samSub {
				p.clicked = true
				c++
			}
		}
	}
	// Linked: 612 Summit registrants with SMS consent (Sam among them) on
	// news.acme.io, and 1,506 other contacts.
	summit, err := pushSummitContacts(ctx, s, st, 611, sam)
	if err != nil {
		return err
	}
	others, err := pushOtherContacts(ctx, s, st, 948+558)
	if err != nil {
		return err
	}
	link := func(site uuid.UUID, ids []pushContact) {
		cands := []*pushSub{}
		for _, p := range subs {
			if p.site == site && p.removed == "" && p.contact == nil {
				cands = append(cands, p)
			}
		}
		rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
		for i, c := range ids {
			if i >= len(cands) {
				break
			}
			p := cands[i]
			id := c.id
			p.contact, p.zone = &id, c.zone
			p.linkedBy = []string{"member", "link", "link", "account"}[rng.IntN(4)]
			at := p.created.Add(time.Duration(rng.IntN(3600)) * time.Second)
			if p.linkedBy == "member" || p.linkedBy == "link" {
				at = p.created
			}
			if at.After(now) {
				at = p.created
			}
			p.linkedAt = &at
		}
	}
	link(st.news, append(summit, others[:948]...))
	link(st.acmeSite, others[948:])
	// The rest of the consent record.
	for _, p := range subs {
		if p.zone == "" {
			p.zone = pushZones[rng.IntN(len(pushZones))]
		}
		if p.version == 0 {
			switch {
			case p.site == st.acmeSite:
				p.version = 1
			case p.created.Before(s.pushDay(6, 14, 18)):
				p.version = 1
			case p.created.Before(s.pushDay(8, 10, 31)):
				p.version = 2
			default:
				p.version = 3
			}
		}
		if p.page == "" {
			if p.site == st.news {
				p.page = newsPages[rng.IntN(len(newsPages))]
			} else {
				p.page = acmePages[rng.IntN(len(acmePages))]
			}
		}
		if !p.ip.IsValid() {
			p.ip = netip.AddrFrom4([4]byte{byte([]int{192, 198, 203}[rng.IntN(3)]), byte([]int{0, 51, 0}[rng.IntN(3)]), byte([]int{2, 100, 113}[rng.IntN(3)]),
				byte(1 + rng.IntN(250))})
		}
	}
	return pushWrite(ctx, s, st, subs, rng)
}

// pushContact is a contact a subscription is linked to.
type pushContact struct {
	id   uuid.UUID
	zone string
}

func pushContactByEmail(ctx context.Context, s *Seeder, email string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var phone string
	err := s.Tx.QueryRow(ctx, `SELECT id, phone FROM contacts WHERE workspace_id = $1 AND email = $2`, s.WS().ID, email).Scan(&id, &phone)
	return id, phone, err
}

// pushSummitContacts are Summit registrants who can get texts, Sam first.
func pushSummitContacts(ctx context.Context, s *Seeder, st *pwState, n int, sam uuid.UUID) ([]pushContact, error) {
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.timezone FROM channel_consents cc JOIN contacts c ON c.id = cc.contact_id
		WHERE cc.list_id = $1 AND cc.channel = 'sms' AND cc.status = 'subscribed' AND c.id <> $2
		  AND NOT EXISTS (SELECT 1 FROM channel_suppressions x WHERE x.workspace_id = c.workspace_id AND x.channel = 'sms' AND x.phone = c.phone AND x.removed_at IS NULL)
		ORDER BY c.email LIMIT $3`, st.summit, sam, n)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (pushContact, error) {
		var c pushContact
		return c, r.Scan(&c.id, &c.zone)
	})
	if err != nil {
		return nil, err
	}
	if len(out) < n {
		return nil, fmt.Errorf("only %d Summit registrants can get texts", len(out))
	}
	return out, nil
}

// pushOtherContacts are subscribed contacts who aren't Summit registrants.
func pushOtherContacts(ctx context.Context, s *Seeder, st *pwState, n int) ([]pushContact, error) {
	rows, err := s.Tx.Query(ctx, `SELECT c.id, c.timezone FROM contacts c
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.status = 'active'
		  AND NOT EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = $2)
		ORDER BY c.email LIMIT $3`, s.WS().ID, st.summit, n)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (pushContact, error) {
		var c pushContact
		return c, r.Scan(&c.id, &c.zone)
	})
	if err != nil {
		return nil, err
	}
	if len(out) < n {
		return nil, fmt.Errorf("only %d contacts to link", len(out))
	}
	return out, nil
}

// pushWrite stores the subscriptions and the campaign.
func pushWrite(ctx context.Context, s *Seeder, st *pwState, subs []*pushSub, rng *mrand.Rand) error {
	ws := s.WS()
	sofia := s.Users["sofia@acme.io"]
	texts := map[uuid.UUID]map[int]string{
		st.newsPrompt: {1: newsPromptTitle + " · One notification when a new post is out.",
			2: newsPromptTitle + " · One notification when a new post is out. No more than one a day.", 3: newsPromptTitle + " · " + newsPromptMessage},
		st.bellPrompt: {1: bellPromptTitle + " · " + bellPromptMessage},
	}
	rows := make([][]any, 0, len(subs))
	for _, p := range subs {
		form := st.newsPrompt
		if p.site == st.acmeSite {
			form = st.bellPrompt
		}
		endpoint := pushEndpoint(p.browser, rng)
		var key, auth []byte
		if p.removed == "" {
			key, auth = pushKeys()
		} else {
			endpoint = ""
		}
		var lastSeen *time.Time
		if p.shown {
			t := st.sent.Add(time.Duration(2+rng.IntN(40)) * time.Second)
			lastSeen = &t
		}
		ip := p.ip
		rows = append(rows, []any{p.id, ws.ID, p.site, st.key, endpoint, key, auth, p.browser, pushPlatform[p.browser], p.zone, p.contact, p.linkedAt, p.linkedBy,
			&form, int32(p.version), texts[form][p.version], p.page, &ip, pushUA[p.browser], p.created, lastSeen, int32(0), p.removedAt, p.removed, true})
	}
	if err := smsCopy(ctx, s, "push_subscriptions", []string{"id", "workspace_id", "site_id", "key_id", "endpoint", "p256dh", "auth", "browser", "platform",
		"time_zone", "contact_id", "linked_at", "linked_by", "prompt_form_id", "prompt_version", "prompt_text", "page", "ip", "user_agent", "created_at",
		"last_seen_at", "failures", "removed_at", "removed_reason", "demo"}, rows); err != nil {
		return err
	}
	// The campaign: sent by Sofia on 14 Oct at 09:05 to news.acme.io.
	st.campaign = ID()
	created := s.pushDay(14, 8, 36)
	link := "https://news.acme.io/posts/feature-flags-at-scale"
	buttons := []map[string]string{{"label": "Read the post", "url": link}, {"label": "Open the docs", "url": "https://acme.io/docs/flags"}}
	bjson, _ := json.Marshal(buttons)
	recipients := int64(0)
	for _, p := range subs {
		if p.recipient {
			recipients++
		}
	}
	done := st.sent.Add(2*time.Minute + 14*time.Second)
	audience, _ := json.Marshal(map[string]any{"push_sites": []string{st.news.String()}, "lists": []string{}, "segments": []string{}})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, audience, editor_mode, mjml, html, plain_text, version, channel,
			content_hash, created_by, updated_by, created_at, updated_at, scheduled_at, started_at, completed_at, started_by, audience_count)
		VALUES ($1,$2,$3,'sent','','',$4,'html','','','',4,'push',$5,$6,$6,$7,$8,$9,$9,$8,$6,$10)`,
		st.campaign, ws.ID, "New post: Feature flags at scale", audience, "demo-push-ff", sofia.ID, created, done, st.sent, recipients); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_campaigns (campaign_id, workspace_id, title, message, link, image_media_id, buttons, ttl_minutes, tag, replace_tag,
			site_ids, lists, segments, schedule_mode, outside, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,240,'new-post',true,$8,'{}','{}','now','hold',4)`,
		st.campaign, ws.ID, "Feature flags at scale", "How we ship to 2% of accounts first, watch the error budget, and switch a flag off in seconds.", link,
		st.image, bjson, []uuid.UUID{st.news}); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO push_sends (campaign_id, workspace_id, phase, recipients, left_out, started_by, scheduled_at, first_send_at, last_send_at,
			completed_at) VALUES ($1,$2,'completed',$3,'{}',$4,$5,$5,$6,$6)`, st.campaign, ws.ID, recipients, sofia.ID, st.sent, done); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"title": "Feature flags at scale", "message": "How we ship to 2% of accounts first, watch the error budget, and switch a flag off in seconds.",
		"link": link, "image": media.URL(st.image, "feature-flags.png"), "buttons": buttons, "tag": "new-post", "replace": true, "ttl": 240,
		"utm": "feature-flags-at-scale"})
	msgs, pms, clicks := [][]any{}, [][]any{}, [][]any{}
	i := 0
	for _, p := range subs {
		if !p.recipient {
			continue
		}
		id := ID()
		at := st.sent.Add(time.Duration(i) * 45 * time.Millisecond)
		i++
		tok := hex.EncodeToString(id[10:16]) + "demo"
		status, reason, answer := "sent", "", "201 Created"
		var sentAt, failedAt, delivered, shownAt, clickedAt *time.Time
		sentAt = &at
		switch {
		case p.removed == "gone":
			status, reason, answer = "failed", "gone", "410 Gone · Unregistered"
			if rng.IntN(4) == 0 {
				answer = "404 Not Found · NotFound"
			}
			sentAt, failedAt = nil, &at
		case p.shown:
			t := at.Add(time.Duration(2+rng.IntN(40)) * time.Second)
			if rng.IntN(10) == 0 {
				t = at.Add(time.Duration(20+rng.IntN(300)) * time.Minute) // an offline laptop
			}
			status, delivered, shownAt = "delivered", &t, &t
			if p.clicked {
				// Most clicks in the first hour, a tail over the day.
				d := time.Duration(rng.ExpFloat64()*35*60) * time.Second
				if d > 23*time.Hour {
					d = 23 * time.Hour
				}
				c := t.Add(10*time.Second + d)
				clickedAt = &c
				action := 0
				target := link + "?utm_source=push&utm_medium=push&utm_campaign=feature-flags-at-scale"
				switch r := rng.IntN(100); {
				case r < 11:
					action = 1
				case r < 15:
					action, target = 2, "https://acme.io/docs/flags?utm_source=push&utm_medium=push&utm_campaign=feature-flags-at-scale"
				}
				clicks = append(clicks, []any{ID(), ws.ID, id, int32(action), target, c})
			}
		}
		provider := ""
		if status != "failed" {
			provider = pushEndpointHost(p.browser) + "/m/" + hex.EncodeToString(id[8:16])
		}
		msgs = append(msgs, []any{id, ws.ID, "push", "campaign", &st.campaign, p.contact, p.id.String(), p.zone, string(body), int32(1), status, reason,
			provider, &at, sentAt, delivered, failedAt, at, at})
		pms = append(pms, []any{id, ws.ID, p.id, p.site, p.browser, tok, answer, shownAt, clickedAt})
	}
	if err := smsCopy(ctx, s, "channel_messages", []string{"id", "workspace_id", "channel", "kind", "campaign_id", "contact_id", "recipient", "time_zone", "body",
		"units", "status", "reason", "provider_message_id", "send_after", "sent_at", "delivered_at", "failed_at", "created_at", "updated_at"}, msgs); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "push_messages", []string{"message_id", "workspace_id", "subscription_id", "site_id", "browser", "token", "answer", "shown_at",
		"clicked_at"}, pms); err != nil {
		return err
	}
	if err := smsCopy(ctx, s, "push_clicks", []string{"id", "workspace_id", "message_id", "action", "target", "occurred_at"}, clicks); err != nil {
		return err
	}
	_, err := s.Tx.Exec(ctx, `UPDATE push_keys SET last_accepted_at = $2 WHERE id = $1`, st.key, done)
	return err
}

func pushEndpointHost(browser string) string {
	switch browser {
	case "chrome", "chrome_android":
		return "https://fcm.googleapis.com/fcm/send"
	case "edge":
		return "https://wns2-par02p.notify.windows.com/w"
	case "firefox":
		return "https://updates.push.services.mozilla.com/wpush/v2"
	}
	return "https://web.push.apple.com"
}
