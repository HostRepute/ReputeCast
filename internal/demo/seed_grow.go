package demo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedGrow = seedGrowImpl }

// Grow (design expansion phase 5; brief reputecast-design-addon-grow.md §4):
// the Weekly Digest referral program, on since 23 Sep and set up by Sofia
// Rossi, its three milestones, a link for every Weekly Digest subscriber,
// 214 referrals from the top referrers and 54 more readers, the rewards
// they earned, link visits by channel, the leaderboard, the review queue
// (Alex Moreau's burst, Mateo Silva's shared address), and the
// recommendations: Ops Weekly and The Deliverability Desk as partners on
// other instances, Fabrikam Engineering Letter as a link, and Ops Weekly
// recommending Acme in return.
//
// The demo's audience numbers are pinned by other screens (31,988 Weekly
// Digest subscribers, 48,213 contacts), so referred readers and partner
// signups are existing recent Weekly Digest subscribers: a few of them are
// renamed to the board's addresses, and their growth sources stay as the
// audience sample set them (DESIGN_INVENTORY). Everything is relative to
// now, like the rest of the demo: "15 Oct" is today.

// growReferral is one referral of the sample.
type growReferral struct {
	id, contact               uuid.UUID
	seq                       int
	status, reason, via       string
	signals                   []string
	ip                        string
	network, asn, kind        string
	signed                    time.Time
	confirmed, holdUntil      *time.Time
	counted, notCounted, held *time.Time
	reminded                  *time.Time
}

// growReferrer is one referrer of the sample with what they did.
type growReferrer struct {
	key                 string
	contact, link       uuid.UUID
	email, first, last  string
	code                string
	counted, hold       int
	review              int
	awaiting, never     int
	notCounted          []string
	showName, showBoard bool
	visits              int
	refs                []*growReferral
}

type growContact struct {
	id                 uuid.UUID
	email, first, last string
	created            time.Time
}

type growWorld struct {
	ctx                  context.Context
	s                    *Seeder
	ws                   store.Workspace
	digest               uuid.UUID
	program              uuid.UUID
	sofia, marcus, linh  uuid.UUID
	sender               *uuid.UUID
	rng                  *rand.Rand
	start, leaderboardAt time.Time
	now                  time.Time
	subs                 []growContact
	byEmail              map[string]*growContact
	links                map[uuid.UUID]uuid.UUID // contact → link
	codes                map[uuid.UUID]string
	referrers            []*growReferrer
	milestones           []growMilestone
	used                 map[uuid.UUID]bool // referred and partner contacts
	partnerPool          []growContact
	recAudit             []growAudit
}

// growAudit is a change in the sample's history (Marcus's partner links).
type growAudit struct {
	at                        time.Time
	action, ttype, tid, tname string
	after                     map[string]any
}

type growMilestone struct {
	id      uuid.UUID
	count   int
	name    string
	typ     string
	file    *uuid.UUID
	fileLen int64
}

func seedGrowImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	w := &growWorld{ctx: ctx, s: s, ws: ws, digest: s.Lists["Weekly Digest"], program: ID(), sofia: s.Users["sofia@acme.io"].ID,
		marcus: s.Users["marcus@acme.io"].ID, linh: s.Users["linh@acme.io"].ID, rng: audRand(1525, 2026), now: s.Now,
		byEmail: map[string]*growContact{}, links: map[uuid.UUID]uuid.UUID{}, codes: map[uuid.UUID]string{}, used: map[uuid.UUID]bool{}}
	if id, ok := s.Senders["digest@news.acme.io"]; ok {
		w.sender = &id
	}
	w.start = s.designDay(22, 9, 0)
	w.leaderboardAt = s.designDay(14, 10, 30)
	steps := []struct {
		name string
		fn   func() error
	}{
		{"program", w.seedProgram},
		{"milestones", w.seedMilestones},
		{"subscribers", w.loadSubscribers},
		{"links", w.seedLinks},
		{"referrals", w.seedReferrals},
		{"rewards", w.seedRewards},
		{"visits", w.seedVisits},
		{"leaderboard", w.seedLeaderboard},
		{"recommendations", w.seedRecommendations},
		{"audit", w.seedGrowAudit},
		{"blocks", w.seedBlocks},
	}
	for _, st := range steps {
		if err := st.fn(); err != nil {
			return fmt.Errorf("grow %s: %w", st.name, err)
		}
	}
	return nil
}

func (w *growWorld) exec(sql string, args ...any) error {
	_, err := w.s.Tx.Exec(w.ctx, sql, args...)
	return err
}

func (w *growWorld) day(daysAgo, hour, minute int) time.Time {
	return w.s.designDay(daysAgo, hour, minute)
}

func gtp(t time.Time) *time.Time { return &t }

// --- The program, its files and milestones -----------------------------------------------

func (w *growWorld) seedProgram() error {
	created := w.day(23, 15, 10)
	return w.exec(`INSERT INTO referral_programs (id, workspace_id, list_id, status, setup_step, setup_saved_at, setup_saved_by, first_on_at, first_on_by,
			status_at, status_by, hold_days, invite_headline, invite_text, hub_intro, share_message, leaderboard_on, leaderboard_at, leaderboard_by,
			leaderboard_periods, leaderboard_rows, sender_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, 'on', 0, $4, $5, $6, $5, $6, $5, 7, $7, $8, $9, $10, true, $11, $5, '{month,all}', 10, $12, $5, $13, $11)`,
		w.program, w.ws.ID, w.digest, w.start.Add(-10*time.Minute), w.sofia, w.start,
		"{first_name} invited you to the Acme Weekly Digest",
		"What shipped at Acme, what's next, and a deep dive every Tuesday. Written by the Acme team for the people who build on it.",
		"Thanks for sharing the Weekly Digest. Every friend who confirms brings you closer to the next reward.",
		"I read the Acme Weekly Digest every Tuesday. Here's my link if you'd like it too: {link}",
		w.leaderboardAt, w.sender, created)
}

// growPDF is a valid PDF of exactly size bytes (a one-page document padded
// with a comment), so the hub and the editor show the board's file sizes.
func growPDF(title string, size int) []byte {
	head := "%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n" +
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >> endobj\n"
	stream := "BT /F1 24 Tf 72 700 Td (" + title + ") Tj ET"
	head += fmt.Sprintf("4 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", len(stream), stream)
	tail := "trailer << /Root 1 0 R >>\n%%EOF\n"
	pad := size - len(head) - len(tail) - 2
	var b bytes.Buffer
	b.WriteString(head)
	b.WriteString("%")
	b.Write(bytes.Repeat([]byte(" "), max(pad, 0)))
	b.WriteString("\n")
	b.WriteString(tail)
	return b.Bytes()
}

func (w *growWorld) seedFile(name string, data []byte, at time.Time) (uuid.UUID, error) {
	id := ID()
	sum := sha256.Sum256(data)
	if err := w.exec(`INSERT INTO referral_files (id, workspace_id, program_id, file_name, content_type, size_bytes, sha256, downloads, uploaded_by, uploaded_at)
		VALUES ($1, $2, $3, $4, 'application/pdf', $5, $6, 0, $7, $8)`, id, w.ws.ID, w.program, name, len(data), sum[:], w.sofia, at); err != nil {
		return id, err
	}
	for seq, off := 0, 0; off < len(data); seq, off = seq+1, off+1<<20 {
		end := min(off+1<<20, len(data))
		if err := w.exec(`INSERT INTO referral_file_chunks (file_id, seq, data) VALUES ($1, $2, $3)`, id, seq, data[off:end]); err != nil {
			return id, err
		}
	}
	return id, nil
}

func (w *growWorld) seedMilestones() error {
	at := w.day(23, 15, 30)
	sticker := growPDF("Acme sticker pack", 2516582)
	guide := growPDF("The Acme Field Guide", 18874368)
	stickerID, err := w.seedFile("acme-sticker-pack.pdf", sticker, at)
	if err != nil {
		return err
	}
	guideID, err := w.seedFile("acme-field-guide.pdf", guide, at)
	if err != nil {
		return err
	}
	type spec struct {
		count            int
		name, desc, typ  string
		file             *uuid.UUID
		size             int
		offerID, offerNm string
	}
	specs := []spec{
		{1, "Sticker pack", "A sheet of Acme stickers to print at home.", referrals.RewardDownload, &stickerID, len(sticker), "", ""},
		{3, "The Acme Field Guide", "Two years of the Digest's best deep dives, edited into one PDF.", referrals.RewardDownload, &guideID, len(guide), "", ""},
		{10, "Insider for 3 months", "Three months of Insider, free: every deep dive and the archive.", referrals.RewardCoupon, nil, 0,
			"offer_referral_insider_3m", "Referral reward · 3 months of Insider free"},
	}
	for i, sp := range specs {
		id, email := ID(), ID()
		subject, preheader, src := referrals.DefaultMilestoneEmail(w.program, "Weekly Digest", sp.name, sp.typ)
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return fmt.Errorf("milestone email: %s", res.Errors[0].Message)
		}
		created := at.Add(time.Duration(i) * 4 * time.Minute)
		if err := w.exec(`INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by, created_at, updated_at, sender_id)
			VALUES ($1, $2, NULL, $3, 1, $4, $5, $5, $6)`, email, w.ws.ID, "Milestone email · "+fmt.Sprint(sp.count)+" · "+sp.name, w.sofia, created, w.sender); err != nil {
			return err
		}
		if err := w.exec(`INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at)
			VALUES ($1, 1, $2, $3, $4, $5, $6, $7, $8)`, email, subject, preheader, src, res.HTML, contentcheck.PlainText(res.HTML), w.sofia, created); err != nil {
			return err
		}
		if err := w.exec(`INSERT INTO referral_milestones (id, workspace_id, program_id, count, reward_name, reward_description, reward_type, file_id,
				coupon_offer_id, coupon_offer_name, coupon_valid_days, email_on, email_id, email_subject, email_preheader, email_mjml, email_html, email_revision,
				email_saved_at, email_saved_by, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 60, true, $11, $12, $13, $14, $15, 1, $16, $17, $17, $16, $16)`,
			id, w.ws.ID, w.program, sp.count, sp.name, sp.desc, sp.typ, sp.file, sp.offerID, sp.offerNm, email, subject, preheader, src, res.HTML, created, w.sofia); err != nil {
			return err
		}
		w.milestones = append(w.milestones, growMilestone{id: id, count: sp.count, name: sp.name, typ: sp.typ, file: sp.file, fileLen: int64(sp.size)})
	}
	return nil
}

// --- People ---------------------------------------------------------------------------------

func (w *growWorld) loadSubscribers() error {
	rows, err := w.s.Tx.Query(w.ctx, `SELECT c.id, c.email, c.first_name, c.last_name, c.created_at FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $1 AND m.status = 'subscribed'
		WHERE c.workspace_id = $2 AND c.erased_at IS NULL ORDER BY c.created_at, c.email`, w.digest, w.ws.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var c growContact
		if err := rows.Scan(&c.id, &c.email, &c.first, &c.last, &c.created); err != nil {
			return err
		}
		w.subs = append(w.subs, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range w.subs {
		w.byEmail[w.subs[i].email] = &w.subs[i]
	}
	return nil
}

// rename gives a sample contact the board's name and address.
func (w *growWorld) rename(c *growContact, email, first, last string) error {
	if err := w.exec(`UPDATE contacts SET email = $2, first_name = $3, last_name = $4 WHERE id = $1`, c.id, email, first, last); err != nil {
		return err
	}
	delete(w.byEmail, c.email)
	c.email, c.first, c.last = email, first, last
	w.byEmail[email] = c
	return nil
}

var growNamed = []struct {
	key, email                      string
	counted, hold, review, awaiting int
	showName, showBoard             bool
	visits                          int
}{
	{"omar", "omar.haddad@gmail.com", 23, 1, 0, 0, true, true, 412},
	{"kenji", "kenji.w@icloud.com", 17, 2, 0, 0, true, true, 318},
	{"priya", "priya.n@gmail.com", 9, 0, 0, 0, false, false, 160},
	{"diego", "d.alvarez@yahoo.com", 6, 1, 0, 0, true, true, 121},
	{"sam", "sam.rivera@outlook.com", 2, 1, 0, 1, false, false, 61},
	{"mateo", "mateo.silva@gmail.com", 2, 0, 3, 0, false, false, 54},
	{"alex", "alex.moreau@gmail.com", 1, 0, 6, 0, false, false, 38},
}

// --- Links -------------------------------------------------------------------------------------

// newCode is a link code (the program's alphabet: no 0, O, 1, I or L).
func (w *growWorld) newCode() string {
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[w.rng.IntN(len(alphabet))]
	}
	return string(b)
}

func (w *growWorld) seedLinks() error {
	seen := map[string]bool{"K7Q2MX": true}
	rows := make([][]any, 0, len(w.subs))
	for _, c := range w.subs {
		code := "K7Q2MX"
		if c.email != "sam.rivera@outlook.com" {
			for {
				code = w.newCode()
				if !seen[code] {
					break
				}
			}
		}
		seen[code] = true
		id := ID()
		w.links[c.id], w.codes[c.id] = id, code
		created := w.start
		if c.created.After(created) {
			created = c.created
		}
		rows = append(rows, []any{id, w.ws.ID, w.program, c.id, code, "active", created})
	}
	_, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_links"}, []string{"id", "workspace_id", "program_id", "contact_id", "code", "status", "created_at"},
		pgx.CopyFromRows(rows))
	return err
}

// --- Referrals --------------------------------------------------------------------------------

// growVia is a referral's channel, in the board's proportions.
func (w *growWorld) via() string {
	n := w.rng.IntN(1000)
	switch {
	case n < 480:
		return "copy"
	case n < 685:
		return "x"
	case n < 860:
		return "linkedin"
	case n < 940:
		return "whatsapp"
	}
	return "email"
}

var growNetworks = []struct{ prefix, name, asn, kind string }{
	{"198.51.100.", "Comcast Cable", "AS7922", "residential"},
	{"203.0.113.", "Deutsche Telekom", "AS3320", "residential"},
	{"192.0.2.", "T-Mobile USA", "AS21928", "mobile"},
	{"198.18.4.", "Verizon Fios", "AS701", "residential"},
	{"198.18.12.", "BT", "AS2856", "residential"},
}

func (w *growWorld) netFor(r *growReferral) {
	n := growNetworks[w.rng.IntN(len(growNetworks))]
	r.ip = fmt.Sprintf("%s%d", n.prefix, 2+w.rng.IntN(250))
	r.network, r.asn, r.kind = n.name, n.asn, n.kind
}

// at is a time on a design day, at a deterministic hour.
func (w *growWorld) at(daysAgo int) time.Time {
	t := w.day(daysAgo, 7+w.rng.IntN(14), w.rng.IntN(60))
	// Nobody signed up through a link before the program was on (no extra
	// random draw, so the rest of the sample stays as it was).
	if t.Before(w.start) {
		t = w.start.Add(time.Duration(15+t.Minute()) * time.Minute)
	}
	return t
}

func (w *growWorld) seedReferrals() error {
	// Recent subscribers are the referred readers; older ones the referrers.
	cut, old := w.now.AddDate(0, 0, -30), w.now.AddDate(0, 0, -60)
	named := map[string]bool{}
	for _, n := range growNamed {
		named[n.email] = true
	}
	var recent, older []*growContact
	for i := range w.subs {
		c := &w.subs[i]
		switch {
		case named[c.email]:
		case c.created.After(cut):
			recent = append(recent, c)
		case c.created.Before(old) && c.first != "":
			older = append(older, c)
		}
	}
	if len(recent) < 230 || len(older) < 500 {
		return fmt.Errorf("not enough Weekly Digest subscribers for the sample (%d recent, %d older)", len(recent), len(older))
	}
	// Mateo Silva and Alex Moreau are two long-time readers, renamed.
	step := len(older) / 64
	pick := func(i int) *growContact { return older[i*step] }
	if err := w.rename(pick(0), "mateo.silva@gmail.com", "Mateo", "Silva"); err != nil {
		return err
	}
	if err := w.rename(pick(1), "alex.moreau@gmail.com", "Alex", "Moreau"); err != nil {
		return err
	}
	for _, n := range growNamed {
		c := w.byEmail[n.email]
		if c == nil {
			return fmt.Errorf("no %s in the demo", n.email)
		}
		w.referrers = append(w.referrers, &growReferrer{key: n.key, contact: c.id, link: w.links[c.id], code: w.codes[c.id], email: c.email, first: c.first,
			last: c.last, counted: n.counted, hold: n.hold, review: n.review, awaiting: n.awaiting, showName: n.showName, showBoard: n.showBoard, visits: n.visits})
	}
	// 54 more referrers: 2 with 4, 8 with 3, 3 with 2, 41 with 1 counted;
	// 13 with a referral in hold, 5 with one not counted, 42 waiting or
	// never confirmed.
	counts := []int{4, 4, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 2}
	for len(counts) < 54 {
		counts = append(counts, 1)
	}
	reasons := []string{referrals.ReasonSameIP, referrals.ReasonSameIP, referrals.ReasonSameIP, referrals.ReasonDisposable, referrals.ReasonLeft}
	for i := 0; i < 54; i++ {
		c := pick(i + 2)
		r := &growReferrer{key: fmt.Sprintf("r%02d", i), contact: c.id, link: w.links[c.id], code: w.codes[c.id], email: c.email, first: c.first, last: c.last,
			counted: counts[i], showBoard: i%3 == 0, showName: i%4 == 0}
		if i < 13 {
			r.hold = 1
		}
		if j := i - 13; j >= 0 && j < len(reasons) {
			r.notCounted = []string{reasons[j]}
		}
		if i >= 18 {
			// 42 not confirmed: 11 still waiting, 31 never confirmed.
			if i < 29 {
				r.awaiting = 1
			} else {
				r.never = 1
			}
			if i >= 54-6 {
				r.never++ // a few links with two
			}
		}
		w.referrers = append(w.referrers, r)
	}
	// Trim to the board's 42: the extra "never" above add 6; take them
	// back from the end.
	extra := 0
	for _, r := range w.referrers {
		extra += r.awaiting + r.never
	}
	for i := len(w.referrers) - 1; extra > 43 && i >= 0; i-- {
		if w.referrers[i].never > 1 {
			w.referrers[i].never--
			extra--
		}
	}
	// Who was referred: the recent subscribers, the board's addresses
	// first.
	next := 0
	take := func() *growContact {
		c := recent[next]
		next++
		w.used[c.id] = true
		return c
	}
	renames := map[string][]struct{ email, first, last string }{
		"sam": {{"nina.patel@gmail.com", "Nina", "Patel"}, {"leo.martin@yahoo.com", "Leo", "Martin"}, {"ivy.walker@icloud.com", "Ivy", "Walker"},
			{"owen.price@outlook.com", "Owen", "Price"}},
		"alex":  {{"ben.t01@outlook.com", "Ben", "T"}, {"ben.t02@outlook.com", "Ben", "T"}, {"ben.t03@outlook.com", "Ben", "T"}, {"ben.t04@outlook.com", "Ben", "T"}, {"ben.t05@outlook.com", "Ben", "T"}, {"ben.t06@outlook.com", "Ben", "T"}},
		"mateo": {{"a.adeyemi@kestrel-logistics.example", "Ade", "Adeyemi"}, {"t.okoro@kestrel-logistics.example", "Tobi", "Okoro"}, {"f.nwosu@kestrel-logistics.example", "Femi", "Nwosu"}},
	}
	for _, r := range w.referrers {
		if err := w.planReferrals(r, take, renames[r.key]); err != nil {
			return err
		}
	}
	// The disposable signup reads like one.
	for _, r := range w.referrers {
		for _, ref := range r.refs {
			if ref.reason == referrals.ReasonDisposable {
				if c := w.contactByID(ref.contact); c != nil {
					if err := w.rename(c, "zx81.trial@mailinator.com", "", ""); err != nil {
						return err
					}
				}
			}
		}
	}
	w.partnerPool = nil
	for _, c := range recent[next:] {
		w.partnerPool = append(w.partnerPool, *c)
	}
	return w.insertReferrals()
}

func (w *growWorld) contactByID(id uuid.UUID) *growContact {
	for i := range w.subs {
		if w.subs[i].id == id {
			return &w.subs[i]
		}
	}
	return nil
}

// planReferrals dates a referrer's referrals: counted ones confirmed 8 to
// 22 days ago, holds in the last 6 days, the board's named ones on their
// days.
func (w *growWorld) planReferrals(r *growReferrer, take func() *growContact, names []struct{ email, first, last string }) error {
	var refs []*growReferral
	add := func(status, reason string, signed time.Time, confirmed *time.Time) *growReferral {
		c := take()
		ref := &growReferral{id: ID(), contact: c.id, status: status, reason: reason, via: w.via(), signed: signed, confirmed: confirmed}
		w.netFor(ref)
		refs = append(refs, ref)
		return ref
	}
	hold := 7 * 24 * time.Hour
	switch r.key {
	case "sam":
		// Nina, Leo, Ivy and Owen (brief §4).
		a := add(referrals.StatusCounted, "", w.day(19, 18, 2), gtp(w.day(19, 18, 40)))
		a.counted = gtp(a.confirmed.Add(hold))
		b := add(referrals.StatusCounted, "", w.day(14, 20, 15), gtp(w.day(13, 8, 5)))
		b.counted = gtp(b.confirmed.Add(hold))
		c := add(referrals.StatusHold, "", w.day(3, 12, 30), gtp(w.day(3, 12, 48)))
		c.holdUntil = gtp(c.confirmed.Add(hold))
		d := add(referrals.StatusAwaiting, "", w.day(1, 19, 6), nil)
		d.reminded = gtp(w.day(0, 9, 0))
		a.via, b.via, c.via, d.via = "copy", "whatsapp", "copy", "email"
	case "alex":
		a := add(referrals.StatusCounted, "", w.day(17, 10, 20), gtp(w.day(17, 10, 31)))
		a.counted = gtp(a.confirmed.Add(hold))
		// Six on 14 Oct between 21:10 and 21:24: a burst, four from one
		// address, numbered addresses.
		for i := 0; i < 6; i++ {
			at := w.day(1, 21, 10+i*2+i/3)
			ref := add(referrals.StatusReview, "", at, gtp(at.Add(90*time.Second)))
			ref.signals = []string{referrals.SignalBurst, referrals.SignalNumbered}
			ref.held, ref.via = ref.confirmed, "copy"
			if i < 4 {
				ref.ip, ref.network, ref.asn, ref.kind = "203.0.113.77", "DigitalOcean", "AS14061", "hosting"
				ref.signals = []string{referrals.SignalBurst, referrals.SignalSharedIP, referrals.SignalNumbered}
			}
		}
	case "mateo":
		a := add(referrals.StatusCounted, "", w.day(20, 9, 12), gtp(w.day(20, 9, 40)))
		a.counted = gtp(a.confirmed.Add(hold))
		b := add(referrals.StatusCounted, "", w.day(16, 14, 2), gtp(w.day(16, 14, 30)))
		b.counted = gtp(b.confirmed.Add(hold))
		for i, d := range []struct{ day, h, m int }{{2, 10, 4}, {2, 15, 41}, {1, 9, 18}} {
			ref := add(referrals.StatusReview, "", w.day(d.day, d.h, d.m), gtp(w.day(d.day, d.h, d.m+20)))
			ref.signals = []string{referrals.SignalSharedIP}
			ref.held, ref.via = ref.confirmed, "email"
			ref.ip, ref.network, ref.asn, ref.kind = "198.51.100.12", "Kestrel Logistics", "AS64511", "office"
			_ = i
		}
	default:
		// Counted: confirmed 8 to 22 days ago, spread evenly.
		for i := 0; i < r.counted; i++ {
			daysAgo := 22 - (i*14)/max(r.counted, 1)
			if r.counted > 15 {
				daysAgo = 22 - (i*15)/r.counted
			}
			signed := w.at(daysAgo)
			ref := add(referrals.StatusCounted, "", signed, gtp(signed.Add(time.Duration(5+w.rng.IntN(300))*time.Minute)))
			if ref.confirmed.Add(hold).After(w.now) {
				ref.confirmed = gtp(w.now.Add(-hold - 3*time.Hour))
			}
			ref.counted = gtp(ref.confirmed.Add(hold))
		}
		for i := 0; i < r.hold; i++ {
			signed := w.at(1 + w.rng.IntN(5))
			ref := add(referrals.StatusHold, "", signed, gtp(signed.Add(time.Duration(10+w.rng.IntN(120))*time.Minute)))
			ref.holdUntil = gtp(ref.confirmed.Add(hold))
		}
		for _, reason := range r.notCounted {
			signed := w.at(10 + w.rng.IntN(10))
			ref := add(referrals.StatusNotCounted, reason, signed, gtp(signed.Add(25*time.Minute)))
			switch reason {
			case referrals.ReasonLeft:
				ref.notCounted = gtp(ref.confirmed.Add(3 * 24 * time.Hour))
			default:
				ref.notCounted = ref.confirmed
			}
		}
		for i := 0; i < r.awaiting; i++ {
			add(referrals.StatusAwaiting, "", w.at(w.rng.IntN(6)), nil)
		}
		for i := 0; i < r.never; i++ {
			signed := w.at(9 + w.rng.IntN(13))
			ref := add(referrals.StatusNotCounted, referrals.ReasonNeverConfirmed, signed, nil)
			ref.notCounted = gtp(signed.Add(hold))
		}
	}
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].signed.Before(refs[j].signed) })
	for i, ref := range refs {
		ref.seq = i + 1
	}
	// The board's addresses, in order.
	if len(names) > 0 {
		k := 0
		for _, ref := range refs {
			if r.key != "sam" && ref.status != referrals.StatusReview {
				continue
			}
			if k >= len(names) {
				break
			}
			if c := w.contactByID(ref.contact); c != nil {
				if err := w.rename(c, names[k].email, names[k].first, names[k].last); err != nil {
					return err
				}
			}
			k++
		}
	}
	r.refs = refs
	return nil
}

func (w *growWorld) insertReferrals() error {
	var rows, events [][]any
	for _, r := range w.referrers {
		counted := 0
		for _, ref := range r.refs {
			var ip any
			network := ""
			if ref.ip != "" {
				a := netip.MustParseAddr(ref.ip)
				ip, network = a, referrals.NetworkOf(a)
			}
			signals := ref.signals
			if signals == nil {
				signals = []string{}
			}
			rows = append(rows, []any{ref.id, w.ws.ID, w.program, r.link, r.contact, ref.contact, ref.seq, ref.status, ref.reason, signals, ref.via,
				ip, network, ref.network, ref.asn, ref.kind, ref.signed, ref.reminded, ref.confirmed, ref.holdUntil, ref.counted, ref.notCounted, ref.held,
				ref.signed, w.now})
			ev := func(kind string, contact uuid.UUID, at time.Time, detail map[string]any) {
				raw, _ := json.Marshal(detail)
				events = append(events, []any{ID(), w.ws.ID, w.program, ref.id, r.link, contact, kind, raw, at})
			}
			ev(referrals.EventSignedUp, ref.contact, ref.signed, map[string]any{"referrer": r.contact.String(), "via": ref.via, "status": referrals.StatusAwaiting})
			switch ref.status {
			case referrals.StatusCounted:
				counted++
				ev(referrals.EventConfirmed, ref.contact, *ref.confirmed, map[string]any{"countsOn": ref.confirmed.Add(7 * 24 * time.Hour)})
				ev(referrals.EventCounted, r.contact, *ref.counted, map[string]any{"referred": ref.contact.String(), "counted": counted})
			case referrals.StatusHold:
				ev(referrals.EventConfirmed, ref.contact, *ref.confirmed, map[string]any{"countsOn": ref.holdUntil})
			case referrals.StatusReview:
				ev(referrals.EventHeld, ref.contact, *ref.confirmed, map[string]any{"signals": ref.signals})
			case referrals.StatusNotCounted:
				ev(referrals.EventNotCounted, ref.contact, *ref.notCounted, map[string]any{"reason": ref.reason})
			}
		}
	}
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referrals"}, []string{"id", "workspace_id", "program_id", "link_id", "referrer_id", "contact_id", "seq",
		"status", "reason", "signals", "via", "ip", "ip_network", "network_name", "network_asn", "network_kind", "signed_up_at", "reminded_at", "confirmed_at",
		"hold_until", "counted_at", "not_counted_at", "held_at", "created_at", "updated_at"}, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	_, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_events"}, []string{"id", "workspace_id", "program_id", "referral_id", "link_id", "contact_id",
		"kind", "detail", "occurred_at"}, pgx.CopyFromRows(events))
	return err
}

// --- Rewards ------------------------------------------------------------------------------------

func (w *growWorld) seedRewards() error {
	var rows, events [][]any
	coupon := 0
	for _, r := range w.referrers {
		var countedAt []time.Time
		for _, ref := range r.refs {
			if ref.status == referrals.StatusCounted {
				countedAt = append(countedAt, *ref.counted)
			}
		}
		sort.Slice(countedAt, func(i, j int) bool { return countedAt[i].Before(countedAt[j]) })
		for _, m := range w.milestones {
			if len(countedAt) < m.count {
				continue
			}
			reached := countedAt[m.count-1]
			id := ID()
			sent := reached.Add(time.Minute)
			var used, expires *time.Time
			claim := ""
			downloads := 0
			switch m.typ {
			case referrals.RewardDownload:
				// 54 of 61 sticker packs and 13 of 14 guides downloaded.
				if (m.count == 1 && w.rng.IntN(61) < 54) || (m.count == 3 && r.key != "r00") || r.key == "sam" {
					used, downloads = gtp(sent.Add(time.Duration(10+w.rng.IntN(600))*time.Minute)), 1+w.rng.IntN(2)
					if used.After(w.now) {
						used = gtp(w.now.Add(-time.Hour))
					}
				}
			case referrals.RewardCoupon:
				coupon++
				issued := w.day(6, 11, 2)
				if r.key == "kenji" {
					issued = w.day(2, 9, 47)
				}
				reached, sent = issued, issued
				expires = gtp(issued.AddDate(0, 0, 60))
				claim = fmt.Sprintf("https://acme.io/insider/claim?code=INSIDER-%s", strings.ToUpper(r.key))
			}
			rows = append(rows, []any{id, w.ws.ID, w.program, m.id, r.contact, r.link, m.count, m.name, m.typ, referrals.RewardSent, reached, sent, used,
				downloads, claim, expires, sent, reached, w.now})
			raw, _ := json.Marshal(map[string]any{"count": m.count, "reward": m.name})
			events = append(events, []any{ID(), w.ws.ID, w.program, nil, r.link, r.contact, referrals.EventMilestoneReached, raw, reached},
				[]any{ID(), w.ws.ID, w.program, nil, r.link, r.contact, referrals.EventRewardSent, raw, sent})
		}
	}
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_rewards"}, []string{"id", "workspace_id", "program_id", "milestone_id", "contact_id", "link_id",
		"milestone_count", "reward_name", "reward_type", "status", "reached_at", "sent_at", "used_at", "downloads", "claim_url", "expires_at", "email_sent_at",
		"created_at", "updated_at"}, pgx.CopyFromRows(rows)); err != nil {
		return err
	}
	if _, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_events"}, []string{"id", "workspace_id", "program_id", "referral_id", "link_id", "contact_id",
		"kind", "detail", "occurred_at"}, pgx.CopyFromRows(events)); err != nil {
		return err
	}
	// The files' download counts follow the rewards'.
	return w.exec(`UPDATE referral_files f SET downloads = coalesce((SELECT sum(rw.downloads) FROM referral_rewards rw
		JOIN referral_milestones m ON m.id = rw.milestone_id WHERE m.file_id = f.id), 0) WHERE f.program_id = $1`, w.program)
}

// --- Visits, choices and the leaderboard ------------------------------------------------------

func (w *growWorld) seedVisits() error {
	// 2,940 visits: copy 1,412, X 604, LinkedIn 512, WhatsApp 233, email 179.
	channels := []string{"copy", "x", "linkedin", "whatsapp", "email"}
	totals := []int{1412, 604, 512, 233, 179}
	var pool []string
	for i, ch := range channels {
		for j := 0; j < totals[i]; j++ {
			pool = append(pool, ch)
		}
	}
	w.rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	// Each referrer's visits: the named ones' from the board, the others
	// share the rest by their signups.
	fixed, signups := 0, 0
	for _, r := range w.referrers {
		fixed += r.visits
		if r.visits == 0 {
			signups += len(r.refs)
		}
	}
	rest := len(pool) - fixed
	given := 0
	others := 0
	for _, r := range w.referrers {
		if r.visits == 0 {
			others++
		}
	}
	k := 0
	for _, r := range w.referrers {
		if r.visits != 0 {
			continue
		}
		k++
		share := rest * len(r.refs) / max(signups, 1)
		if k == others {
			share = rest - given
		}
		r.visits = max(share, len(r.refs))
		given += r.visits
	}
	type key struct {
		link uuid.UUID
		day  time.Time
		via  string
	}
	agg := map[key][2]int{}
	pos := 0
	days := int(w.now.Sub(w.start).Hours()/24) + 1
	for _, r := range w.referrers {
		for i := 0; i < r.visits && pos < len(pool); i++ {
			at := w.start.Add(time.Duration(w.rng.IntN(days*24)) * time.Hour)
			if at.After(w.now) {
				at = w.now.Add(-time.Hour)
			}
			d := growDayOf(at)
			v := agg[key{r.link, d, pool[pos]}]
			v[0]++
			agg[key{r.link, d, pool[pos]}] = v
			pos++
		}
		for _, ref := range r.refs {
			d := growDayOf(ref.signed)
			v := agg[key{r.link, d, ref.via}]
			v[1]++
			agg[key{r.link, d, ref.via}] = v
		}
		if err := w.exec(`UPDATE referral_links SET visits = $2, signups = $3, show_name = $4, show_leaderboard = $5,
			choices_at = CASE WHEN $4 OR $5 THEN $6::timestamptz END WHERE id = $1`, r.link, r.visits, len(r.refs), r.showName, r.showBoard, w.leaderboardAt); err != nil {
			return err
		}
	}
	rows := make([][]any, 0, len(agg))
	for k, v := range agg {
		rows = append(rows, []any{k.link, w.program, k.day, k.via, v[0], v[1]})
	}
	_, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_visits_daily"}, []string{"link_id", "program_id", "day", "via", "visits", "signups"}, pgx.CopyFromRows(rows))
	return err
}

func growDayOf(t time.Time) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

func (w *growWorld) seedLeaderboard() error {
	monthStart := time.Date(w.now.Year(), w.now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var rows [][]any
	for _, period := range []string{referrals.PeriodAll, referrals.PeriodMonth} {
		type entry struct {
			link    uuid.UUID
			counted int
		}
		var es []entry
		for _, r := range w.referrers {
			n := 0
			for _, ref := range r.refs {
				if ref.status == referrals.StatusCounted && (period == referrals.PeriodAll || !ref.counted.Before(monthStart)) {
					n++
				}
			}
			if n > 0 {
				es = append(es, entry{r.link, n})
			}
		}
		sort.SliceStable(es, func(i, j int) bool { return es[i].counted > es[j].counted })
		rank := 0
		for i, e := range es {
			if i == 0 || e.counted != es[i-1].counted {
				rank = i + 1
			}
			rows = append(rows, []any{w.program, period, e.link, rank, e.counted, w.now.Add(-37 * time.Minute)})
		}
	}
	_, err := w.s.Tx.CopyFrom(w.ctx, pgx.Identifier{"referral_leaderboard"}, []string{"program_id", "period", "link_id", "rank", "counted", "computed_at"},
		pgx.CopyFromRows(rows))
	return err
}

// --- History and blocks --------------------------------------------------------------------------

func (w *growWorld) seedGrowAudit() error {
	type ev struct {
		at                              time.Time
		actor                           uuid.UUID
		name, action, ttype, tid, tname string
		after                           map[string]any
	}
	sofia, marcus := w.s.Users["sofia@acme.io"], w.s.Users["marcus@acme.io"]
	evs := []ev{
		{w.day(23, 15, 10), sofia.ID, sofia.Name, "referral.program_created", "referral_program", w.program.String(), "Weekly Digest", map[string]any{"list": "Weekly Digest"}},
		{w.day(23, 15, 30), sofia.ID, sofia.Name, "referral.milestone_created", "referral_milestone", w.milestones[0].id.String(), "Sticker pack", map[string]any{"count": 1, "type": "download"}},
		{w.day(23, 15, 34), sofia.ID, sofia.Name, "referral.milestone_created", "referral_milestone", w.milestones[1].id.String(), "The Acme Field Guide", map[string]any{"count": 3, "type": "download"}},
		{w.day(23, 15, 38), sofia.ID, sofia.Name, "referral.milestone_created", "referral_milestone", w.milestones[2].id.String(), "Insider for 3 months", map[string]any{"count": 10, "type": "coupon"}},
		{w.start, sofia.ID, sofia.Name, "referral.program_turned_on", "referral_program", w.program.String(), "Weekly Digest", map[string]any{"status": "on", "links": len(w.subs)}},
		{w.leaderboardAt, sofia.ID, sofia.Name, "referral.leaderboard_turned_on", "referral_program", w.program.String(), "Weekly Digest", map[string]any{"leaderboard": true}},
	}
	for _, e := range w.recAudit {
		evs = append(evs, ev{e.at, marcus.ID, marcus.Name, e.action, e.ttype, e.tid, e.tname, e.after})
	}
	for _, e := range evs {
		after, _ := json.Marshal(e.after)
		if err := w.exec(`INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
			result, after, metadata) VALUES ($1, $2, $3, 'user', $4, $5, $6, $7, $8, $9, 'allowed', $10, '{}')`,
			ID(), w.ws.ID, e.at, e.actor, e.name, e.action, e.ttype, e.tid, e.tname, after); err != nil {
			return err
		}
	}
	return nil
}

// seedBlocks puts the Referral block in Weekly Digest #139 (which announced
// the program) and the Recommendations block in the welcome email.
func (w *growWorld) seedBlocks() error {
	referral := `<!-- rc:referral {"program":"` + w.program.String() + `","layout":"progress","share":["email","x","linkedin","whatsapp"],"button":"Share your link","progress":true} -->`
	if err := w.exec(`UPDATE campaigns SET html = replace(html, '</body>', $2 || '</body>') WHERE workspace_id = $1 AND name = 'Weekly Digest #139'`,
		w.ws.ID, referral); err != nil {
		return err
	}
	var id uuid.UUID
	var version int32
	var src string
	err := w.s.Tx.QueryRow(w.ctx, `SELECT e.id, e.latest_version, v.mjml FROM automation_emails e
		JOIN automation_email_versions v ON v.email_id = e.id AND v.version = e.latest_version
		WHERE e.workspace_id = $1 AND e.name LIKE 'Welcome · step 1%'`, w.ws.ID).Scan(&id, &version, &src)
	if err != nil {
		return nil //nolint:nilerr // no welcome email in this demo
	}
	block := `<mj-raw><!-- rc:recommendations {"which":"all","layout":"stacked","heading":"Newsletters we recommend","intro":"Picked by the Acme team."} --></mj-raw>`
	i := strings.LastIndex(src, "</mj-body>")
	if i < 0 {
		return nil
	}
	// Before the footer section when there is one.
	if j := strings.LastIndex(src[:i], "<mj-section"); j > 0 && strings.Contains(src[j:i], "nsubscribe") {
		i = j
	}
	src = src[:i] + block + "\n    " + src[i:]
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		return fmt.Errorf("welcome email: %s", res.Errors[0].Message)
	}
	return w.exec(`UPDATE automation_email_versions SET mjml = $3, html = $4, plain_text = $5 WHERE email_id = $1 AND version = $2`,
		id, version, src, res.HTML, contentcheck.PlainText(res.HTML))
}
