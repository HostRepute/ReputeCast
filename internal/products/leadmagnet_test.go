package products_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// openLead opens a lead magnet's file as /l/{token} does; served says the
// file came.
func (s *shop) openLead(w *store.Workspace, token string, count bool) (bool, error) {
	s.t.Helper()
	_, f, err := s.svc.OpenLeadMagnet(s.ctx, w, token, count)
	if f == nil {
		return false, err
	}
	b, _ := io.ReadAll(f)
	_ = f.Close()
	return len(b) > 0, err
}

// The form builder's "After confirmation: offer a download", through the
// API: a form starts with none; a file goes up in chunks like a product's
// and becomes the form's download, with its days (1 to 30); a second file
// replaces it for the readers who come later, and removing it stops the
// links already sent. Another workspace's form isn't reachable.
func TestFormDownloadAPI(t *testing.T) {
	s := newShop(t)
	var form uuid.UUID
	s.scan(`SELECT id FROM forms WHERE workspace_id = $1 AND download_file_id IS NULL ORDER BY created_at LIMIT 1`, []any{s.ws.ID}, &form)
	base := ws + "/forms/" + form.String() + "/download"
	var fd products.FormDownload
	s.owner.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &fd)
	if fd.File != nil {
		t.Fatalf("a form without a download: %+v", fd)
	}
	upload := func(name string, data []byte) products.FormDownload {
		t.Helper()
		var st products.UploadState
		s.owner.Do("POST", base+"/uploads", map[string]any{"name": name, "size": len(data)}).MustStatus(t, 201).JSON(t, &st)
		s.owner.DoRaw("PUT", fmt.Sprintf("%s/uploads/%s?offset=0", base, st.ID), "application/octet-stream", bytes.NewReader(data)).MustStatus(t, 200)
		var out products.FormDownload
		s.owner.Do("POST", base+"/uploads/"+st.ID+"/complete", nil).MustStatus(t, 200).JSON(t, &out)
		return out
	}
	first := upload("checklist.pdf", pdf("the first checklist"))
	if first.File == nil || first.File.Kind != "pdf" || first.File.Name != "checklist.pdf" {
		t.Fatalf("the form's download: %+v", first)
	}
	s.owner.Do("PUT", base, map[string]any{"days": 0}).MustStatus(t, 422)
	s.owner.Do("PUT", base, map[string]any{"days": 31}).MustStatus(t, 422)
	s.owner.Do("PUT", base, map[string]any{"days": 14}).MustStatus(t, 200).JSON(t, &fd)
	if fd.Days != 14 || fd.File == nil || fd.File.ID != first.File.ID {
		t.Errorf("14 days: %+v", fd)
	}
	fern := s.addContact("form.reader@example.com", "Fern")
	offer, ok, err := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, fern)
	if err != nil || !ok || offer.Days != 14 || offer.Name != "checklist.pdf" {
		t.Fatalf("offer: %+v %v %v", offer, ok, err)
	}
	if served, err := s.openLead(&s.ws, strings.TrimPrefix(offer.URL, s.site()+"/l/"), true); err != nil || !served {
		t.Fatalf("the first file: %v %v", served, err)
	}

	// A second file replaces the first.
	second := upload("checklist-v2.pdf", pdf("the second checklist"))
	if second.File == nil || second.File.ID == first.File.ID || second.File.Name != "checklist-v2.pdf" || second.Days != 14 {
		t.Fatalf("the replacement: %+v", second)
	}
	if n := s.count(`SELECT count(*) FROM download_files WHERE id = $1 AND removed_at IS NOT NULL`, uuid.MustParse(first.File.ID)); n != 1 {
		t.Error("the replaced file is still the form's")
	}
	if _, ok := s.audited("form.download_replaced", form); !ok {
		t.Error("the replacement isn't in the audit log")
	}
	later := s.addContact("form.later@example.com", "Lou")
	again, ok, err := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, later)
	if err != nil || !ok || again.Name != "checklist-v2.pdf" {
		t.Fatalf("a later reader's offer: %+v %v %v", again, ok, err)
	}

	// Removed: nothing offered, and the links already sent stop working.
	if r := s.owner.Do("DELETE", base, nil); r.Status != 204 && r.Status != 200 {
		t.Fatalf("remove: %d %s", r.Status, r.Body)
	}
	var removed products.FormDownload
	s.owner.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &removed)
	if removed.File != nil {
		t.Errorf("after removing: %+v", removed)
	}
	if _, ok := s.audited("form.download_removed", form); !ok {
		t.Error("the removal isn't in the audit log")
	}
	if _, ok, _ := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, fern); ok {
		t.Error("a form without a download still offers one")
	}
	if served, err := s.openLead(&s.ws, strings.TrimPrefix(again.URL, s.site()+"/l/"), true); served || err == nil {
		t.Errorf("a link sent before the download was removed still works: %v %v", served, err)
	}

	// Another workspace's form (the demo's engineering blog has none, so
	// one is made there).
	other := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO forms (id, workspace_id, name, slug) SELECT $1, id, 'Engineering signup', 'engineering-signup' FROM workspaces WHERE slug = 'acme-engineering-blog'`, other)
	if n := s.count(`SELECT count(*) FROM forms WHERE id = $1 AND workspace_id <> $2`, other, s.ws.ID); n != 1 {
		t.Fatalf("the other workspace's form: %d", n)
	}
	s.owner.Do("GET", ws+"/forms/"+other.String()+"/download", nil).MustStatus(t, 404)
	s.owner.Do("PUT", ws+"/forms/"+other.String()+"/download", map[string]any{"days": 10}).MustStatus(t, 404)
	if d := s.count(`SELECT download_days FROM forms WHERE id = $1`, other); d != 7 {
		t.Errorf("another workspace's form changed: %d days", d)
	}
}

// A signup form's "offer a download": its file and how many days its links
// work (1 to 30, audited). After confirmation, LeadMagnetFor gives the
// contact a link of their own, the same one while it works and a new one
// after; OpenLeadMagnet sends the file, counting downloads from the start
// only, on the workspace's site only; past its days the link answers
// ErrLinkExpired.
func TestLeadMagnetLinks(t *testing.T) {
	s := newShop(t)
	var form, file uuid.UUID
	var days int32
	s.scan(`SELECT id, download_file_id, download_days FROM forms WHERE workspace_id = $1 AND download_file_id IS NOT NULL ORDER BY created_at LIMIT 1`,
		[]any{s.ws.ID}, &form, &file, &days)
	fd, err := s.svc.FormDownloadOf(s.ctx, &s.ws, form)
	if err != nil || fd.File == nil || fd.File.ID != file.String() || fd.Days != days {
		t.Fatalf("form download: %+v %v", fd, err)
	}
	links, downloaded := fd.Links, fd.Downloaded

	// Its days.
	for _, bad := range []int32{0, 31} {
		if _, err := s.svc.FormDownloadDays(s.ctx, &s.ws, nil, form, bad); err == nil {
			t.Errorf("%d days accepted", bad)
		}
	}
	if fd, err = s.svc.FormDownloadDays(s.ctx, &s.ws, nil, form, 3); err != nil || fd.Days != 3 {
		t.Fatalf("3 days: %+v %v", fd, err)
	}
	if a, ok := s.audited("form.download_updated", form); !ok || a.After["days"] != float64(3) || a.Before["days"] != float64(days) {
		t.Errorf("audit: %v %+v", ok, a)
	}

	// A confirmed contact's link, the same while it works.
	lena := s.addContact("lena.lead@example.com", "Lena")
	offer, ok, err := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, lena)
	if err != nil || !ok || !strings.HasPrefix(offer.URL, s.site()+"/l/") || offer.Days != 3 || offer.Kind != "PDF" || offer.Size <= 0 || offer.Name == "" ||
		!offer.ExpiresAt.Equal(s.clock.now().Add(3*24*time.Hour)) {
		t.Fatalf("offer: %+v %v %v", offer, ok, err)
	}
	if again, _, _ := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, lena); again.URL != offer.URL {
		t.Errorf("a second link while the first works")
	}
	if n := s.count(`SELECT count(*) FROM lead_magnet_links WHERE form_id = $1 AND contact_id = $2`, form, lena); n != 1 {
		t.Errorf("%d links for one contact", n)
	}
	token := strings.TrimPrefix(offer.URL, s.site()+"/l/")

	// Downloads count from the start only.
	if served, err := s.openLead(&s.ws, token, true); err != nil || !served {
		t.Fatalf("download: %v %v", served, err)
	}
	if served, err := s.openLead(&s.ws, token, false); err != nil || !served {
		t.Fatalf("resumed download: %v %v", served, err)
	}
	if n := s.count(`SELECT downloads FROM lead_magnet_links WHERE form_id = $1 AND contact_id = $2`, form, lena); n != 1 {
		t.Errorf("%d downloads counted, want 1", n)
	}
	if fd, _ = s.svc.FormDownloadOf(s.ctx, &s.ws, form); fd.Links != links+1 || fd.Downloaded != downloaded+1 {
		t.Errorf("form numbers: %d links, %d downloaded (were %d, %d)", fd.Links, fd.Downloaded, links, downloaded)
	}

	// Not on another workspace's site, not with another kind of token.
	var other store.Workspace
	s.scan(`SELECT id, slug FROM workspaces WHERE slug <> $1 ORDER BY created_at LIMIT 1`, []any{s.ws.Slug}, &other.ID, &other.Slug)
	if _, err := s.openLead(&other, token, true); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("another workspace's site: %v", err)
	}
	o, _ := s.buy(s.fieldGuide, buyer{Email: "lena.lead@example.com"})
	if _, err := s.openLead(&s.ws, s.token(o.ID, s.link(o.ID).ID), true); !errors.Is(err, products.ErrNoDownload) {
		t.Errorf("a product's download token: %v", err)
	}

	// Past its days.
	s.clock.add(3*24*time.Hour - time.Second)
	if served, err := s.openLead(&s.ws, token, true); err != nil || !served {
		t.Fatalf("a second before expiry: %v %v", served, err)
	}
	s.clock.add(time.Second)
	if served, err := s.openLead(&s.ws, token, true); !errors.Is(err, products.ErrLinkExpired) || served {
		t.Errorf("expired: %v %v", served, err)
	}
	fresh, ok, err := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), form, lena)
	if err != nil || !ok || fresh.URL == offer.URL || !fresh.ExpiresAt.Equal(s.clock.now().Add(3*24*time.Hour)) {
		t.Fatalf("a new link after expiry: %+v %v %v", fresh, ok, err)
	}
	if served, err := s.openLead(&s.ws, strings.TrimPrefix(fresh.URL, s.site()+"/l/"), true); err != nil || !served {
		t.Errorf("the new link: %v %v", served, err)
	}

	// A form without a download offers none.
	var plain uuid.UUID
	s.scan(`SELECT id FROM forms WHERE workspace_id = $1 AND download_file_id IS NULL ORDER BY created_at LIMIT 1`, []any{s.ws.ID}, &plain)
	if _, ok, err := s.svc.LeadMagnetFor(s.ctx, &s.ws, s.site(), plain, lena); ok || err != nil {
		t.Errorf("a form without a download: %v %v", ok, err)
	}
}
