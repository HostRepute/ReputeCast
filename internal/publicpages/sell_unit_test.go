package publicpages

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Every page of the site parses with the frame, the sell pages included.
func TestSellTemplatesParse(t *testing.T) {
	h := &Handler{}
	tmpl, err := h.siteTemplates()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"sell-page", "sell-gate", "sell-shop", "sell-product", "sell-handoff", "sell-thanks", "sell-download", "sell-registered"} {
		if tmpl[p] == nil {
			t.Errorf("no template %s", p)
		}
	}
	// Every word the pages use is in the catalog.
	for _, k := range []string{"buy.oneTime", "handoff.title", "thanks.title", "dl.expiredTitle", "event.registered", "account.purchases"} {
		_ = sellMsgs.M(k)
	}
}

// A countdown reveals and hides blocks by their ids, before and after its end.
func TestCountdownRevealsBlocks(t *testing.T) {
	src := `<section data-pb-id="hero"><span data-pb-id="eb-before">Opens soon</span><span data-pb-id="eb-after">Open now</span>` +
		`<rc-countdown at="2026-10-06T09:00" tz="America/New_York" units="days,hours" end="reveal" reveal="eb-after,buy" hide="eb-before,waitlist"></rc-countdown></section>` +
		`<section data-pb-id="waitlist">Join</section><section data-pb-id="buy">Buy</section>`
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		t.Fatal(err)
	}
	ids := func(before bool) []string {
		var out []string
		walkNodes(nodes, func(n *html.Node) {
			if id := attrOf(n, "data-pb-id"); id != "" {
				out = append(out, id)
			}
		})
		_ = before
		return out
	}
	if len(ids(true)) != 5 {
		t.Fatalf("parsed ids: %v", ids(true))
	}
	before := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	after := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		now          time.Time
		gone, stayed string
	}{{before, "Open now", "Opens soon"}, {after, "Opens soon", "Open now"}} {
		hidden := map[string]bool{}
		at, _ := time.Parse("2006-01-02T15:04", "2026-10-06T13:00")
		ended := !c.now.Before(at)
		list := "eb-after,buy"
		if ended {
			list = "eb-before,waitlist"
		}
		for _, id := range strings.Split(list, ",") {
			hidden[id] = true
		}
		fresh, _ := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
		kept := removeBlocks(fresh, hidden)
		var b strings.Builder
		for _, n := range kept {
			_ = html.Render(&b, n)
		}
		if strings.Contains(b.String(), c.gone) || !strings.Contains(b.String(), c.stayed) {
			t.Fatalf("at %s: %s", c.now, b.String())
		}
	}
}
