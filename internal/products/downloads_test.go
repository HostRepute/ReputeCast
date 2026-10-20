package products_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// open opens a link's file as the download route does: count is false for a
// download resumed from the middle (a Range request). served says the file
// came with it.
func (s *shop) open(token string, file uuid.UUID, count bool) (state string, served bool, err error) {
	s.t.Helper()
	_, f, state, err := s.svc.OpenFile(s.ctx, &s.ws, token, file, "198.51.100.7", count)
	if f != nil {
		b, _ := io.ReadAll(f)
		_ = f.Close()
		served = len(b) > 0
	}
	return state, served, err
}

// firstFile is a product's first file.
func (s *shop) firstFile(product uuid.UUID) uuid.UUID {
	s.t.Helper()
	var id uuid.UUID
	s.scan(`SELECT id FROM download_files WHERE owner_kind = 'product' AND owner_id = $1 AND removed_at IS NULL ORDER BY position, uploaded_at LIMIT 1`, []any{product}, &id)
	return id
}

// A link works for the product's days and downloads, counted on the server
// with the address kept only as a hash. A download resumed from the middle
// (a Range request) doesn't count again, and once the count is used up the
// rest of a download already counted still comes. Past its days the link is
// expired, resumed downloads too; "Send a new link" makes a new one from
// now, for the order's address, and the old link then says it was replaced.
func TestDownloadLimitAndExpiry(t *testing.T) {
	s := newShop(t)
	fg := s.liveOf(s.fieldGuide)
	allowed := int(fg.LinkDownloads)
	if allowed < 3 {
		t.Fatalf("the Field Guide allows %d downloads", allowed)
	}
	o, _ := s.buy(s.fieldGuide, buyer{Email: "casey@example.com"})
	link := s.link(o.ID)
	tok := s.token(o.ID, link.ID)
	v, err := s.svc.Download(s.ctx, &s.ws, tok)
	if err != nil || v.State != products.DownloadOK || v.Allowed != fg.LinkDownloads || v.Used != 0 || v.Left() != fg.LinkDownloads || v.LinkDays != fg.LinkDays ||
		v.EmailMasked != "c•••@example.com" || v.OrderNumber != o.Number || len(v.Files) == 0 || v.ProductName != "The Deliverability Field Guide" || v.Test {
		t.Fatalf("download page: %+v %v", v, err)
	}
	file := v.Files[0].ID
	if file != s.firstFile(s.fieldGuide) {
		t.Fatalf("the page's file %s isn't the product's", file)
	}

	// Each download from the start counts; a resumed one doesn't.
	for i := 1; i <= allowed; i++ {
		if st, served, err := s.open(tok, file, true); st != products.DownloadOK || !served || err != nil {
			t.Fatalf("download %d: %s %v %v", i, st, served, err)
		}
		if i == 2 {
			if st, served, _ := s.open(tok, file, false); st != products.DownloadOK || !served {
				t.Fatalf("resumed download: %s %v", st, served)
			}
		}
	}
	if l := s.link(o.ID); l.DownloadsUsed != int32(allowed) {
		t.Fatalf("%d downloads counted, want %d", l.DownloadsUsed, allowed)
	}
	// Used up: a new download is refused, the rest of a counted one comes.
	if st, served, _ := s.open(tok, file, true); st != products.DownloadUsedUp || served {
		t.Errorf("past the limit: %s %v", st, served)
	}
	if st, served, _ := s.open(tok, file, false); st != products.DownloadOK || !served {
		t.Errorf("resuming past the limit: %s %v", st, served)
	}
	if l := s.link(o.ID); l.DownloadsUsed != int32(allowed) {
		t.Errorf("%d downloads counted after the limit", l.DownloadsUsed)
	}
	if n := s.count(`SELECT count(*) FROM product_order_downloads WHERE order_id = $1`, o.ID); n != allowed {
		t.Errorf("%d downloads recorded, want %d", n, allowed)
	}
	if n := s.count(`SELECT count(*) FROM product_order_downloads t WHERE t.order_id = $1 AND (t.ip_hash IS NULL OR strpos(t::text, '198.51.100.7') > 0)`, o.ID); n != 0 {
		t.Errorf("%d downloads keep the address in the clear", n)
	}
	if v, _ := s.svc.Download(s.ctx, &s.ws, tok); v.State != products.DownloadUsedUp || v.Left() != 0 {
		t.Errorf("download page used up: %s %d", v.State, v.Left())
	}

	// Only the product's files, with a genuine token of the workspace.
	if _, _, err := s.open(tok, s.firstFile(s.launchKit), true); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("another product's file: %v", err)
	}
	if _, _, err := s.open(tok, uuid.New(), true); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("an unknown file: %v", err)
	}
	if _, err := s.svc.Download(s.ctx, &s.ws, tok[:len(tok)-4]+"AAAA"); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("a forged token: %v", err)
	}
	var other store.Workspace
	s.scan(`SELECT id, slug FROM workspaces WHERE slug <> $1 ORDER BY created_at LIMIT 1`, []any{s.ws.Slug}, &other.ID, &other.Slug)
	if _, err := s.svc.Download(s.ctx, &other, tok); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("another workspace's site: %v", err)
	}

	// Past its days.
	o2, _ := s.buy(s.fieldGuide, buyer{Email: "dana@example.com"})
	old := s.link(o2.ID)
	tok2 := s.token(o2.ID, old.ID)
	s.clock.add(time.Duration(fg.LinkDays)*24*time.Hour - time.Second)
	if st, served, _ := s.open(tok2, file, true); st != products.DownloadOK || !served {
		t.Fatalf("a second before expiry: %s %v", st, served)
	}
	s.clock.add(time.Second)
	if st, served, _ := s.open(tok2, file, true); st != products.DownloadExpired || served {
		t.Errorf("expired: %s %v", st, served)
	}
	if st, served, _ := s.open(tok2, file, false); st != products.DownloadExpired || served {
		t.Errorf("resuming an expired link: %s %v", st, served)
	}
	if v, _ := s.svc.Download(s.ctx, &s.ws, tok2); v.State != products.DownloadExpired || v.Used != 1 {
		t.Errorf("expired page: %s, %d used", v.State, v.Used)
	}

	// Send a new link.
	masked, err := s.svc.BuyerNewLink(s.ctx, &s.ws, tok2)
	if err != nil || masked != "d•••@example.com" {
		t.Fatalf("new link: %q %v", masked, err)
	}
	links := s.links(o2.ID)
	if len(links) != 2 || links[1].ID != old.ID || links[1].SupersededAt == nil {
		t.Fatalf("links: %+v", links)
	}
	fresh := links[0]
	if fresh.Reason != "buyer" || fresh.DownloadsAllowed != fg.LinkDownloads || fresh.DownloadsUsed != 0 || fresh.SupersededAt != nil ||
		!fresh.ExpiresAt.Equal(s.clock.now().Add(time.Duration(fg.LinkDays)*24*time.Hour)) || s.deliveryJobs(fresh.ID) != 1 {
		t.Errorf("new link: %+v", fresh)
	}
	if d := s.event(o2.ID, "buyer_asked"); d["email"] != "d•••@example.com" {
		t.Errorf("buyer_asked: %v", d)
	}
	if v, _ := s.svc.Download(s.ctx, &s.ws, tok2); v.State != products.DownloadReplaced {
		t.Errorf("the old link: %s", v.State)
	}
	if st, served, _ := s.open(tok2, file, true); st != products.DownloadReplaced || served {
		t.Errorf("downloading from the old link: %s %v", st, served)
	}
	if st, served, _ := s.open(s.token(o2.ID, fresh.ID), file, true); st != products.DownloadOK || !served {
		t.Errorf("downloading from the new link: %s %v", st, served)
	}
}

// New links go only to the address on the order. The buyer asks from a
// link's page (three a day), a member sends one from the order panel, and a
// signed-in member asks from the account page (three a day, the buyer's own
// requests counting); whatever address anyone types, every delivery email
// goes to the order's. A revoked order gets no new link, and a link made
// before the revocation isn't sent after it.
func TestNewLinks(t *testing.T) {
	s := newShop(t)
	s.deliverable()
	const email = "morgan.buyer@example.com"
	morgan := s.addContact(email, "Morgan")
	o, _ := s.buy(s.fieldGuide, buyer{Email: email, Name: "Morgan Buyer"})
	if o.ContactID == nil || *o.ContactID != morgan {
		t.Fatalf("buyer not linked to the contact: %+v", o.ContactID)
	}
	s.deliver(s.link(o.ID).ID)

	// The buyer, from the latest link's page each time: three today. (The
	// clock moves between links: links of the same instant have no order.)
	for i := 1; i <= 3; i++ {
		s.clock.add(time.Minute)
		cur := s.link(o.ID)
		if masked, err := s.svc.BuyerNewLink(s.ctx, &s.ws, s.token(o.ID, cur.ID)); err != nil || masked != "m•••@example.com" {
			t.Fatalf("buyer's new link %d: %q %v", i, masked, err)
		}
		next := s.link(o.ID)
		if next.ID == cur.ID || next.Reason != "buyer" {
			t.Fatalf("buyer's new link %d: %+v", i, next)
		}
		s.deliver(next.ID)
	}
	if _, err := s.svc.BuyerNewLink(s.ctx, &s.ws, s.token(o.ID, s.link(o.ID).ID)); !errors.Is(err, products.ErrTooManyLinks) {
		t.Errorf("a fourth new link today: %v", err)
	}
	// The account page counts them.
	if err := s.svc.AccountNewLink(s.ctx, &s.ws, morgan, "someone.else@example.com", o.ID); !errors.Is(err, products.ErrTooManyLinks) {
		t.Errorf("the account's new link after the buyer's three: %v", err)
	}

	// A member sends one from the order panel: not limited.
	var linh uuid.UUID
	s.scan(`SELECT id FROM users WHERE email = 'linh@acme.io'`, nil, &linh)
	var d products.ProductOrderDetail
	s.clock.add(time.Minute)
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/new-link", map[string]any{"requestedBy": "the buyer, by replying to the delivery email"}).
		MustStatus(t, 200).JSON(t, &d)
	person := s.link(o.ID)
	if person.Reason != "person" || person.CreatedBy == nil || *person.CreatedBy != linh || !d.Download.NewLink || d.Download.NewLinkBy == "" {
		t.Errorf("member's new link: %+v, panel %+v", person, d.Download)
	}
	s.deliver(person.ID)
	if ev := s.event(o.ID, "new_link"); ev["email"] != email || ev["note"] != "the buyer, by replying to the delivery email" {
		t.Errorf("new_link: %v", ev)
	}
	if a, ok := s.audited("order.link_sent", o.ID); !ok || a.ActorType != "user" || a.After["to"] != email {
		t.Errorf("audit: %v %+v", ok, a)
	}

	// A day later, from the account page: three, then no more.
	s.clock.add(25 * time.Hour)
	for i := 1; i <= 3; i++ {
		s.clock.add(time.Minute)
		if err := s.svc.AccountNewLink(s.ctx, &s.ws, morgan, "someone.else@example.com", o.ID); err != nil {
			t.Fatalf("account's new link %d: %v", i, err)
		}
		l := s.link(o.ID)
		if l.Reason != "account" {
			t.Fatalf("account's new link %d: %+v", i, l)
		}
		s.deliver(l.ID)
	}
	if err := s.svc.AccountNewLink(s.ctx, &s.ws, morgan, email, o.ID); !errors.Is(err, products.ErrTooManyLinks) {
		t.Errorf("a fourth from the account: %v", err)
	}
	// Nobody else's account can ask for it.
	other := s.addContact("not.the.buyer@example.com", "Nope")
	if err := s.svc.AccountNewLink(s.ctx, &s.ws, other, "not.the.buyer@example.com", o.ID); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("another contact's account: %v", err)
	}

	// Every delivery email went to the order's address, and only there.
	if n := s.sendMail(); n != 8 {
		t.Errorf("sent %d delivery emails, want 8", n)
	}
	mail := s.sink.Messages()
	if len(mail) != 8 {
		t.Errorf("the sink got %d messages, want 8", len(mail))
	}
	for _, m := range mail {
		if len(m.To) != 1 || m.To[0] != email {
			t.Errorf("a delivery email went to %v", m.To)
		}
	}

	// Revoked: no new link from anyone, and one made before isn't sent.
	s.clock.add(25 * time.Hour)
	if _, err := s.svc.BuyerNewLink(s.ctx, &s.ws, s.token(o.ID, s.link(o.ID).ID)); err != nil {
		t.Fatal(err)
	}
	waiting := s.link(o.ID)
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/revoke", nil).MustStatus(t, 200)
	s.deliver(waiting.ID)
	if l := s.link(o.ID); l.ID != waiting.ID || l.MessageID != nil || l.Delivery != "pending" {
		t.Errorf("a link made before the revocation was sent: %+v", l)
	}
	if _, err := s.svc.BuyerNewLink(s.ctx, &s.ws, s.token(o.ID, waiting.ID)); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("buyer's new link while revoked: %v", err)
	}
	if err := s.svc.AccountNewLink(s.ctx, &s.ws, morgan, email, o.ID); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("account's new link while revoked: %v", err)
	}
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/new-link", map[string]any{}).MustStatus(t, 409)
	if st := s.downloadState(o.ID, waiting.ID); st != products.DownloadRevoked {
		t.Errorf("download page while revoked: %s", st)
	}
	if a, ok := s.audited("order.access_revoked", o.ID); !ok || a.ActorType != "user" {
		t.Errorf("audit: %v %+v", ok, a)
	}
	if n := s.sendMail(); n != 0 {
		t.Errorf("sent %d emails while revoked", n)
	}
	// Restored, the buyer can ask again.
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/restore", nil).MustStatus(t, 200)
	if _, err := s.svc.BuyerNewLink(s.ctx, &s.ws, s.token(o.ID, waiting.ID)); err != nil {
		t.Errorf("buyer's new link once restored: %v", err)
	}
}

// A download whose file can't be opened (gone from the data volume, a disk
// error) isn't counted: the buyer keeps that download.
func TestDownloadThatFailsIsNotCounted(t *testing.T) {
	s := newShop(t)
	o, _ := s.buy(s.fieldGuide, buyer{Email: "erin@example.com"})
	link := s.link(o.ID)
	file := s.firstFile(s.fieldGuide)
	var key string
	s.scan(`SELECT storage_key FROM download_files WHERE id = $1`, []any{file}, &key)
	if err := os.Remove(filepath.Join(s.svc.Files.Dir, filepath.FromSlash(key))); err != nil {
		t.Fatal(err)
	}
	if _, served, err := s.open(s.token(o.ID, link.ID), file, true); err == nil || served {
		t.Fatalf("a missing file: served %v, %v", served, err)
	}
	if l := s.link(o.ID); l.DownloadsUsed != 0 {
		t.Errorf("a download that failed used %d of the link's downloads", l.DownloadsUsed)
	}
	if n := s.count(`SELECT count(*) FROM product_order_downloads WHERE order_id = $1`, o.ID); n != 0 {
		t.Errorf("%d downloads recorded for a file that never left", n)
	}
}

// "Send a new link" on the page of a link a newer one replaced (the buyer
// opened an older email) must get the buyer an email: the page says a link
// is on its way.
func TestNewLinkFromAReplacedLinkReachesTheBuyer(t *testing.T) {
	s := newShop(t)
	o, _ := s.buy(s.fieldGuide, buyer{Email: "riley@example.com"})
	purchase := s.link(o.ID)
	s.deliver(purchase.ID)
	// A member sent a newer link an hour later, delivered too.
	s.clock.add(time.Hour)
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/new-link", map[string]any{"requestedBy": "the buyer"}).MustStatus(t, 200)
	newer := s.link(o.ID)
	s.deliver(newer.ID)
	jobs := s.count(`SELECT count(*) FROM river_job WHERE kind = 'products.deliver'`)

	// The buyer opens the first email's link: replaced, with "Send a new link".
	tok := s.token(o.ID, purchase.ID)
	if st := s.downloadState(o.ID, purchase.ID); st != products.DownloadReplaced {
		t.Fatalf("the first link: %s", st)
	}
	if masked, err := s.svc.BuyerNewLink(s.ctx, &s.ws, tok); err != nil || masked != "r•••@example.com" {
		t.Fatalf("new link: %q %v", masked, err)
	}
	if n := s.count(`SELECT count(*) FROM river_job WHERE kind = 'products.deliver'`); n != jobs+1 {
		t.Errorf("pressing Send a new link queued %d delivery emails", n-jobs)
	}
	if s.event(o.ID, "buyer_asked") == nil {
		t.Error("the buyer's request isn't in the order's activity")
	}
}
