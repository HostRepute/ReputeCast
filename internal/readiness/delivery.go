package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

// What the email weighs as sent, where its images and tracked links load
// from (SR-7): Gmail clips at 102 KB of the HTML it receives, which is the
// HTML after the send rewrote the links for tracking and added the open
// pixel; images over plain HTTP are blocked or flagged by some clients; a
// tracking domain that doesn't resolve breaks every link of the email.

// campaignTracking reads campaigns.tracking as the send does (both on
// when unset).
func campaignTracking(raw json.RawMessage) engine.Tracking {
	t := engine.Tracking{Opens: true, Clicks: true}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &t)
	}
	return t
}

// sampleSigner signs the sample tracking tokens: they have the length of
// real ones and are never sent.
var sampleSigner, _ = linktoken.NewSigner(make([]byte, 32))

// trackedSize is the size of the HTML the send renders for one recipient:
// the media library's images made absolute, the links rewritten to click
// redirects, the open pixel added and the system links filled in, as the
// sending engine does it (engine.Prepare and Render). 0 when nothing is
// tracked or the email doesn't render.
func trackedSize(html string, t engine.Tracking, urls publicurl.Builder, workspace string) int {
	if (!t.Opens && !t.Clicks) || strings.TrimSpace(html) == "" {
		return 0
	}
	tpl, err := engine.Prepare(engine.Content{HTML: media.Absolutize(html, urls.Base())}, t)
	if err != nil {
		return 0
	}
	id := uuid.Max
	sign := func(k linktoken.Kind, link uint32) string {
		return sampleSigner.Sign(linktoken.Claims{Kind: k, WorkspaceID: id, MessageID: id, ContactID: id, Link: link})
	}
	links := map[string]string{
		"unsubscribe_url":     urls.Unsubscribe(workspace, sign(linktoken.Unsubscribe, 0)),
		"preferences_url":     urls.Preferences(workspace, sign(linktoken.Preferences, 0)),
		"view_in_browser_url": urls.ViewOnline(workspace, sign(linktoken.ViewOnline, 0)),
	}
	if t.Opens {
		links[engine.OpenVar] = urls.Open(sign(linktoken.Open, 0))
	}
	for _, l := range tpl.Links {
		links[engine.LinkVar(l.Position)] = urls.Click(sign(linktoken.Click, uint32(l.Position)))
	}
	out, err := tpl.Render(liquid.Vars("reader@example.com", "Alex", "Reader", nil, links), true)
	if err != nil {
		return 0
	}
	return len(out.HTML)
}

// hostOfURL is a URL's host and whether it uses HTTPS.
func hostOfURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	return strings.ToLower(u.Hostname()), u.Scheme == "https"
}

// trackingFacts fills in where the tracked links and the library's images
// load from, and whether the tracking domain resolves.
func (s *Service) trackingFacts(ctx context.Context, ct *ContentFacts) {
	base := s.URLs.Base()
	ct.LibraryHost, ct.LibraryHTTPS = hostOfURL(base)
	if !ct.TrackOpens && !ct.TrackClicks {
		return
	}
	ct.TrackingHost = trackingHost(base)
	_, ct.TrackingHTTPS = hostOfURL(base)
	if ct.TrackingHost == "" || s.DNS == nil {
		// A development instance (localhost): nobody outside reaches it.
		return
	}
	ct.TrackingDNS = resolves(ctx, s.DNS, ct.TrackingHost)
}

// librarySizes fills in the size of the media library's images (relative
// /media/{id}/{name} addresses) from the library; one that isn't there
// any more doesn't load.
func (s *Service) librarySizes(ctx context.Context, wsID uuid.UUID, images []contentcheck.ContentCheckImage) {
	ids := []uuid.UUID{}
	for _, im := range images {
		if id, ok := libraryID(im.URL); ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.Q.ReadinessMediaSizes(ctx, store.ReadinessMediaSizesParams{WorkspaceID: wsID, Ids: ids})
	if err != nil {
		return
	}
	sizes := map[uuid.UUID]int64{}
	for _, r := range rows {
		sizes[r.ID] = int64(r.SizeBytes)
	}
	for i, im := range images {
		id, ok := libraryID(im.URL)
		if !ok {
			continue
		}
		images[i].Probed = true
		if n, found := sizes[id]; found {
			images[i].Status, images[i].Bytes = 200, n
		} else {
			images[i].Status, images[i].Broken = 404, true
		}
	}
}

// libraryID is the media asset of a library image's relative address.
func libraryID(src string) (uuid.UUID, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(src), media.PathPrefix)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(strings.SplitN(rest, "/", 2)[0])
	return id, err == nil
}

// resolves looks a host up (A, then AAAA): ok, missing (no address), or
// the kind of DNS trouble when DNS didn't answer.
func resolves(ctx context.Context, r dnscheck.Resolver, host string) string {
	a, errA := r.A(ctx, host)
	if errA == nil && len(a) > 0 {
		return "ok"
	}
	aaaa, errAAAA := r.AAAA(ctx, host)
	if errAAAA == nil && len(aaaa) > 0 {
		return "ok"
	}
	for _, err := range []error{errA, errAAAA} {
		if err != nil && !dnscheck.IsNotFound(err) {
			return dnsTrouble(err.Error())
		}
	}
	return "missing"
}

// trackingDomain adds a row when the tracked links' domain can't carry the
// clicks: it doesn't resolve (Block), DNS didn't answer (Unknown), or the
// links and the open pixel use plain HTTP (Warn). When it's fine, the link
// domains row names it.
func (b *builder) trackingDomain() {
	ct := b.f.Content
	host := ct.TrackingHost
	if host == "" || !ct.Checked {
		return
	}
	label := tx("label.trackingDomain", nil)
	settings := link(tx("fix.openSenders", nil), b.href("/settings/senders"))
	switch ct.TrackingDNS {
	case "missing":
		b.add(check{Key: "links.tracking_domain", Group: GroupLinks, Status: Block, Label: label,
			Title:  tx("tracking.domainMissing", A{"host": host}),
			Detail: tx("tracking.domainMissingDetail", A{"host": host}), Fixes: []fix{settings}})
		return
	case "", "ok":
	default:
		note := tx("note.notChecked", nil)
		if ct.TrackingDNS == TroubleTimeout {
			note = tx("note.timedOut", nil)
		}
		b.add(check{Key: "links.tracking_domain", Group: GroupLinks, Status: Unknown, Label: label, Note: note,
			Title:  tx("tracking.domainUnknown", A{"host": host}),
			Detail: tx("tracking.domainUnknownDetail", A{"trouble": ct.TrackingDNS})})
		return
	}
	if !ct.TrackingHTTPS {
		b.add(check{Key: "links.tracking_domain", Group: GroupLinks, Status: Warn, Label: label,
			Title:  tx("tracking.domainHTTP", A{"host": host}),
			Detail: tx("tracking.domainHTTPDetail", nil), Ack: tx("tracking.domainHTTPAck", nil)})
	}
}

// images is the "images load over HTTPS" row of the Content group: every
// image's address, and when the run requested them, whether they load and
// how heavy the largest is.
func (b *builder) images() {
	ct := b.f.Content
	if !ct.Checked || len(ct.Report.ImageList) == 0 {
		return
	}
	list := ct.Report.ImageList
	label := tx("label.images", nil)
	hosts := []string{}
	var insecure, broken []contentcheck.ContentCheckImage
	var largest int64
	local := 0
	for _, im := range list {
		host := im.Host
		switch {
		case host == "" && strings.Contains(im.URL, "{{"):
			// Personalized: known at send time only.
		case host == "":
			// The media library's: served by this instance.
			host = ct.LibraryHost
			switch {
			case localURL("http://" + host):
				local++
			case !ct.LibraryHTTPS:
				insecure = append(insecure, im)
			}
		case im.Insecure:
			insecure = append(insecure, im)
		}
		if host != "" && !slices.Contains(hosts, host) {
			hosts = append(hosts, host)
		}
		if im.Probed && im.Broken {
			broken = append(broken, im)
		}
		largest = max(largest, im.Bytes)
	}
	total := len(list)
	open := func(im contentcheck.ContentCheckImage) fix {
		return link(tx("fix.openInEditor", nil), b.editor(im.Path, "deliverability"))
	}
	switch {
	case len(broken) > 0:
		parts := []Text{}
		for i, im := range broken {
			if i == 2 {
				parts = append(parts, tx("links.brokenMore", A{"count": len(broken) - 2}))
				break
			}
			if im.Status == 0 {
				parts = append(parts, tx("links.noResponse", A{"url": displayURL(im.URL)}))
			} else {
				parts = append(parts, tx("links.returns", A{"url": displayURL(im.URL), "status": fmt.Sprint(im.Status)}))
			}
		}
		b.add(check{Key: "content.images", Group: GroupContent, Status: Warn, Label: label,
			Title:  tx("images.broken", A{"count": len(broken), "total": total}),
			Detail: sentences(joined(" · ", parts), tx("images.brokenDetail", nil)),
			Ack:    tx("images.brokenAck", A{"count": len(broken)}), Fixes: []fix{open(broken[0])}})
	case len(insecure) > 0:
		b.add(check{Key: "content.images", Group: GroupContent, Status: Warn, Label: label,
			Title:  tx("images.insecure", A{"count": len(insecure), "total": total}),
			Detail: sentences(RawText(displayURL(insecure[0].URL)), tx("images.insecureDetail", nil)),
			Ack:    tx("images.insecureAck", A{"count": len(insecure)}), Fixes: []fix{open(insecure[0])}})
	default:
		kb := int(math.Round(float64(largest) / 1024))
		c := check{Key: "content.images", Group: GroupContent, Status: Pass, Label: label,
			Title:  tx("images.https", A{"count": total}),
			Detail: tx("images.hosted", A{"hosts": ArgAnd(hosts), "hasHosts": yes(len(hosts) > 0), "kb": kb}),
			Short:  tx("images.httpsShort", A{"count": total})}
		if local > 0 {
			// A development instance serves them over plain HTTP, and
			// nobody outside reaches it: no claim about HTTPS.
			c.Title, c.Short = tx("images.local", A{"count": total}), tx("images.localShort", A{"count": total})
		}
		b.add(c)
	}
}
