package publicpages_test

import (
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The shop (/products): the published products set to show in it, each
// card opening its sales page; the shop's switch takes it off the site.
func TestShopPage(t *testing.T) {
	e := spDemo(t)
	// A card's link (the header links the Field Guide's page too).
	card := func(path, name string) string {
		return `<h2><a href="` + e.Server.URL + spSite + path + `">` + name + `</a></h2>`
	}
	shop := get(t, e.Env, spSite+"products")
	must(t, shop, http.StatusOK, card("field-guide", "The Deliverability Field Guide"), "$29", "See the guide",
		card("launch-kit", "Newsletter Launch Kit"), "Pay what you want · from $5", "See the kit", `<link rel="canonical" href="http://news.acme.io/products">`)
	lacks(t, shop, "Warm-up planner")

	// Hidden from the shop.
	e.exec(`UPDATE products SET live = jsonb_set(live, '{showInShop}', 'false') WHERE workspace_id = $1 AND slug = 'newsletter-launch-kit'`, e.ws.ID)
	shop = get(t, e.Env, spSite+"products")
	must(t, shop, http.StatusOK, "The Deliverability Field Guide")
	lacks(t, shop, "Newsletter Launch Kit")

	// A product without a sales page opens its default page.
	e.exec(`UPDATE products SET live = live - 'salesPageId' WHERE workspace_id = $1 AND slug = 'deliverability-field-guide'`, e.ws.ID)
	must(t, get(t, e.Env, spSite+"products"), http.StatusOK, card("products/deliverability-field-guide", "The Deliverability Field Guide"))

	// Off, then on again.
	e.api.Do(http.MethodPut, wsAPI+"/pages/shop", map[string]any{"on": false}).MustStatus(t, http.StatusOK)
	must(t, get(t, e.Env, spSite+"products"), http.StatusNotFound)
	e.api.Do(http.MethodPut, wsAPI+"/pages/shop", map[string]any{"on": true}).MustStatus(t, http.StatusOK)
	must(t, get(t, e.Env, spSite+"products"), http.StatusOK, "The Deliverability Field Guide")

	// Its views count as the nil page's.
	before := e.views(uuid.Nil.String())
	do(t, e.Env, http.MethodGet, spSite+"products", nil, hdr{"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile/15E148 Safari/604.1"})
	if after := e.views(uuid.Nil.String()); after != before+1 {
		t.Errorf("shop views %d → %d", before, after)
	}
}

// A sales page whose changes are scheduled is still live: the shop keeps
// sending buyers to it.
func TestShopLinksASalesPageWithScheduledChanges(t *testing.T) {
	e := spDemo(t)
	at := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+e.pageID("field-guide")+"/publish", map[string]any{"when": "at", "at": at}).MustStatus(t, http.StatusOK)
	must(t, get(t, e.Env, spSite+"field-guide"), http.StatusOK, "<h1>The Deliverability Field Guide</h1>")
	// The card's link, not the header's.
	shop := get(t, e.Env, spSite+"products")
	m := regexp.MustCompile(`<h2><a href="([^"]*)">The Deliverability Field Guide</a></h2>`).FindStringSubmatch(shop.Body)
	if m == nil || m[1] != e.Server.URL+spSite+"field-guide" {
		t.Fatalf("the Field Guide's card links %v, want its sales page", m)
	}
}
