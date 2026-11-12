package tracker

import (
	"context"
	"encoding/json"
	"html"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The snippets a site adds. The script tag carries the site's consent
// mode: the script waits for reputecast('consent', true) unless the tag
// says data-wait-consent="false", and the collect endpoint drops the
// events of a waiting site that don't say consent was given, so nothing is
// stored before consent even when a tag is copied without the attribute
// or edited.

// queueStub lets the page call reputecast() before the script has loaded:
// the calls wait in reputecast.q.
const queueStub = `window.reputecast=window.reputecast||function(){(reputecast.q=reputecast.q||[]).push(arguments)};`

func (s *Service) snippet(ctx context.Context, ws *store.Workspace, id uuid.UUID, name string, wait bool) TrackerSnippet {
	l := i18n.LangFrom(ctx)
	attr := html.EscapeString
	var b strings.Builder
	b.WriteString("<script>" + queueStub + "</script>\n")
	b.WriteString(`<script async src="` + attr(s.URLs.TrackerScript()) + `" data-workspace="` + attr(ws.Slug) + `" data-site="` + id.String() + `"`)
	if !wait {
		b.WriteString(` data-wait-consent="false"`)
	}
	b.WriteString("></script>")
	return TrackerSnippet{
		Script:       b.String(),
		ShopifyPixel: s.shopifyPixel(l, ws, id, name, wait),
		WordPress:    mSnippetWordPress.Text(l, i18n.Args{"site": id.String()}),
	}
}

// jsString is a JavaScript string literal (JSON is one).
func jsString(v string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// shopifyPixel is the code of a Shopify custom pixel (Settings › Customer
// events). Custom pixels run in a sandboxed frame without the page's
// cookies or storage, so it keeps the visitor id through Shopify's
// browser.cookie, sends Shopify's standard events page_viewed,
// product_viewed and product_added_to_cart, and, for a site that waits,
// sends only once the visitor allows analytics (init.customerPrivacy,
// visitorConsentCollected).
func (s *Service) shopifyPixel(l i18n.Lang, ws *store.Workspace, id uuid.UUID, name string, wait bool) string {
	var b strings.Builder
	line := func(v string) { b.WriteString(v + "\n") }
	line("// ReputeCast · " + strings.ReplaceAll(name, "\n", " "))
	line("// " + mPixelOpen.Text(l, nil))
	if wait {
		line("// " + mPixelConsent.Text(l, nil))
	}
	line("const RC_URL = " + jsString(s.URLs.TrackerCollect(ws.Slug, id.String())) + ";")
	if wait {
		line("let rcAllowed = !!(init.customerPrivacy && init.customerPrivacy.analyticsProcessingAllowed);")
		line("api.customerPrivacy.subscribe(\"visitorConsentCollected\", (e) => { rcAllowed = !!e.customerPrivacy.analyticsProcessingAllowed; if (rcAllowed) rcFlush(); else rcQueue.length = 0; });")
	} else {
		line("let rcAllowed = true;")
	}
	b.WriteString(`let rcVid = null, rcTimer = 0, rcClick = null;
const rcQueue = [];
async function rcId() {
  if (rcVid) return rcVid;
  let v = await browser.cookie.get("rc_vid");
  if (!/^[A-Za-z0-9_-]{22}$/.test(v || "")) {
    const r = new Uint8Array(16);
    crypto.getRandomValues(r);
    v = btoa(String.fromCharCode(...r)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  }
  await browser.cookie.set("rc_vid=" + v + "; path=/; max-age=34560000; SameSite=Lax; Secure");
  return (rcVid = v);
}
async function rcFlush() {
  clearTimeout(rcTimer);
  if (!rcAllowed || navigator.globalPrivacyControl === true) return;
  while (rcQueue.length) {
    const body = { vid: await rcId(), c: 1, events: rcQueue.splice(0, 20) };
    if (rcClick) { body.rc = rcClick; rcClick = null; }
    fetch(RC_URL, { method: "POST", body: JSON.stringify(body), keepalive: true, credentials: "omit", headers: { "Content-Type": "text/plain" } });
  }
}
function rcSend(ev) {
  if (navigator.globalPrivacyControl === true) return;
  ev.at = Date.now();
  if (rcQueue.length < 100) rcQueue.push(ev);
  clearTimeout(rcTimer);
  rcTimer = setTimeout(rcFlush, 1000);
}
analytics.subscribe("page_viewed", (e) => {
  const loc = e.context.document.location;
  const rc = new URLSearchParams(loc.search).get("rc");
  if (rc) rcClick = rc;
  rcSend({ t: "page", url: loc.href, ref: e.context.document.referrer });
});
analytics.subscribe("product_viewed", (e) => {
  const p = e.data.productVariant && e.data.productVariant.product;
  if (p && p.id) rcSend({ t: "product", url: e.context.document.location.href, id: String(p.id) });
});
analytics.subscribe("product_added_to_cart", (e) => {
  const c = e.data.cartLine;
  const p = c && c.merchandise && c.merchandise.product;
  if (p && p.id) rcSend({ t: "cart", url: e.context.document.location.href, id: String(p.id), qty: c.quantity || 1 });
});
`)
	return b.String()
}
