package contentcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SR-7: the heuristics' known false positives stay quiet: "free" inside a
// compound, "only" without a deadline, a number of hours that isn't one.
func TestHeuristicsFalsePositives(t *testing.T) {
	for _, subject := range []string{"Our new gluten-free menu", "Members only: the Q4 roadmap", "2 hours of Q&A with the founders", "Toll-free support is back",
		"Free-range eggs, now in stock"} {
		if got := SubjectHeuristics(subject); len(got) != 0 {
			t.Errorf("%q trips %v", subject, got)
		}
	}
	for subject, want := range map[string]string{"48 hours only: 30% off": "urgency", "Only 3 days left": "urgency", "Get it free": "money words", "Risk-free trial": "money words"} {
		if got := strings.Join(SubjectHeuristics(subject), ", "); got != want {
			t.Errorf("%q: got %q want %q", subject, got, want)
		}
	}
}

// SR-7: the preheader and the body are checked too; a phrase or two in the
// body is normal, the pile-up is what filters weigh.
func TestPreheaderAndBodyHeuristics(t *testing.T) {
	c := &Checker{}
	body := `<p>Act now: this offer is free, guaranteed, and ends soon. Hurry, it's a last chance!!! Really!!! Now!!!</p><p>Acme Inc · 500 Market St · <a href="{{ unsubscribe_url }}">Unsubscribe</a></p>`
	r, err := c.Check(context.Background(), Input{Subject: "Our November update", Preheader: "Last chance to save", HTML: body})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.PreheaderHeuristics, ",") != "urgency" {
		t.Errorf("preheader: %v", r.PreheaderHeuristics)
	}
	if strings.Join(r.BodyHeuristics, ",") != "spam phrases,repeated punctuation" {
		t.Errorf("body: %v", r.BodyHeuristics)
	}
	calm := `<p>The gluten-free menu is free for members only. Two hours of Q&A follow.</p>`
	if got := BodyHeuristics(PlainText(calm), "en"); len(got) != 0 {
		t.Errorf("calm body: %v", got)
	}
	// Other languages are checked for punctuation only.
	if got := BodyHeuristics("act now, free, hurry, last chance", "de"); len(got) != 0 {
		t.Errorf("German body: %v", got)
	}
}

// SR-7: every image is listed with where it loads from; plain-HTTP images
// are counted; the media library's are relative until the send.
func TestImageList(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column>
<mj-image src="https://cdn.acme.io/hero.png" alt="Hero"></mj-image>
<mj-image src="http://old.acme.io/logo.gif" alt="Logo"></mj-image>
<mj-image src="/media/1/chart.png" alt="Chart"></mj-image>
<mj-image src="{{ avatar_url }}" alt=""></mj-image>
<mj-text>Acme Inc · 500 Market St <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text>
</mj-column></mj-section></mj-body></mjml>`
	r, err := (&Checker{}).Check(context.Background(), Input{Subject: "Charts", MJML: src})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ImageList) != 4 || r.ImagesInsecure != 1 {
		t.Fatalf("images: %+v (insecure %d)", r.ImageList, r.ImagesInsecure)
	}
	if r.ImageList[0].Host != "cdn.acme.io" || r.ImageList[1].Host != "old.acme.io" || !r.ImageList[1].Insecure || r.ImageList[2].Host != "" || r.ImageList[3].Host != "" {
		t.Fatalf("hosts: %+v", r.ImageList)
	}
}

type fakeProber map[string]ProbeResult

func (f fakeProber) Probe(_ context.Context, urls []string) map[string]ProbeResult {
	out := map[string]ProbeResult{}
	for _, u := range urls {
		if r, ok := f[u]; ok {
			out[u] = r
		}
	}
	return out
}

// SR-7: when links are requested, the images are too: whether they load
// and how heavy they are.
func TestImagesRequestedWithLinks(t *testing.T) {
	p := fakeProber{
		"https://cdn.acme.io/hero.png":          {Status: 200, Bytes: 186368},
		"https://cdn.acme.io/missing.png":       {Status: 404, Broken: true},
		"https://www.linkedin.com/company/acme": {Status: 999, BotBlocked: true},
	}
	html := `<img src="https://cdn.acme.io/hero.png" alt="Hero"><img src="https://cdn.acme.io/missing.png" alt="Gone">
<a href="https://www.linkedin.com/company/acme">LinkedIn</a> <p>Acme Inc · 500 Market St <a href="{{ unsubscribe_url }}">Unsubscribe</a></p>`
	r, err := (&Checker{Prober: p}).Check(context.Background(), Input{Subject: "Hello", HTML: html, ProbeLinks: true})
	if err != nil {
		t.Fatal(err)
	}
	if im := r.ImageList[0]; !im.Probed || im.Bytes != 186368 || im.Broken {
		t.Errorf("hero: %+v", im)
	}
	if im := r.ImageList[1]; !im.Probed || !im.Broken || im.Status != 404 {
		t.Errorf("missing: %+v", im)
	}
	var li ContentCheckLink
	for _, l := range r.LinkList {
		if strings.Contains(l.URL, "linkedin") {
			li = l
		}
	}
	if !li.Probed || !li.BotBlocked || li.Broken {
		t.Errorf("LinkedIn's 999 is unknown, not broken: %+v", li)
	}
}

// SR-7: the prober tells a site refusing automated requests (999, 429)
// from a broken link, and reads the size a server reports.
func TestProberBotBlockingAndSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/linkedin":
			w.WriteHeader(999)
		case "/busy":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/image":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Range", "bytes 0-0/186368")
			w.WriteHeader(http.StatusPartialContent)
		}
	}))
	defer srv.Close()
	p := &HTTPProber{Client: srv.Client()}
	res := p.Probe(context.Background(), []string{srv.URL + "/linkedin", srv.URL + "/busy", srv.URL + "/image"})
	for _, path := range []string{"/linkedin", "/busy"} {
		if r := res[srv.URL+path]; !r.BotBlocked || r.Broken {
			t.Errorf("%s: %+v", path, r)
		}
	}
	if r := res[srv.URL+"/image"]; r.Broken || r.Bytes != 186368 {
		t.Errorf("ranged GET: %+v", r)
	}
	if NewHTTPProber().MaxLinks != MaxProbedLinks || MaxProbedLinks < 60 {
		t.Errorf("a long digest's links are all requested: %d", NewHTTPProber().MaxLinks)
	}
}

// SR-7: one image-ratio threshold, the composer's and Send Readiness's.
func TestImageRatioThreshold(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column>
<mj-image src="/media/1/a.png" alt="A" width="552px" height="420px"></mj-image>
<mj-text>Short text.</mj-text>
<mj-text>Acme Inc · 500 Market St <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text>
</mj-column></mj-section></mj-body></mjml>`
	r, err := (&Checker{}).Check(context.Background(), Input{Subject: "Picture", MJML: src})
	if err != nil {
		t.Fatal(err)
	}
	flagged := false
	for _, is := range r.Issues {
		if is.ID == "content.image_ratio" {
			flagged = true
		}
	}
	if (r.ImageShare >= ImageShareLimit) != flagged {
		t.Fatalf("share %.2f, flagged %v", r.ImageShare, flagged)
	}
	if r.ImageShare < ImageShareLimit {
		t.Fatalf("the sample should be mostly image: %.2f", r.ImageShare)
	}
}

// R1.U3: a listed or shortened link is offered "Replace with destination
// URL" only when the address it redirects to is known and clean on the URI
// blocklists; a listed destination, a destination whose lookup failed and a
// link nobody requested are edited by hand. No address is ever guessed.
func TestReplaceOnlyWithKnownCleanDestination(t *testing.T) {
	html := `<a href="https://go.acme-promo.link/sale">Claim your discount</a> <a href="https://bit.ly/3xK9tQ">Read the post</a>
<a href="https://go.acme-promo.link/bad">Other offer</a> <a href="https://bit.ly/unknown">Webinar</a>
<p>Acme Inc · 500 Market St <a href="{{ unsubscribe_url }}">Unsubscribe</a></p>`
	p := fakeProber{
		"https://go.acme-promo.link/sale": {Status: 200, Final: "https://acme.io/pricing/annual"},
		"https://bit.ly/3xK9tQ":           {Status: 200, Final: "https://blog.acme.io/annual-plans"},
		"https://go.acme-promo.link/bad":  {Status: 200, Final: "https://bad.example/offer"},
		"https://bit.ly/unknown":          {Status: 200, Final: "https://slow.example/webinar"},
	}
	lists := StaticBlocklist{"go.acme-promo.link": {"SURBL", "Spamhaus DBL"}, "bad.example": {"SURBL"}}
	checker := &Checker{Blocklist: unknownFor{lists, "slow.example"}, Prober: p}
	r, err := checker.Check(context.Background(), Input{Subject: "Annual plans", HTML: html, ProbeLinks: true})
	if err != nil {
		t.Fatal(err)
	}
	fixes := map[string][]ContentCheckFix{}
	for _, is := range r.Issues {
		if is.ID == "links.blocklist" || is.ID == "links.shortener" {
			fixes[is.URL] = is.Fixes
		}
	}
	replace := func(url string) string {
		f := fixes[url]
		if len(f) == 0 || f[len(f)-1].Kind != "edit_link" {
			t.Fatalf("%s: every listed or shortened link can be edited: %+v", url, f)
		}
		if len(f) == 2 && f[0].Kind == "replace_link" {
			if f[0].Label != "Replace with destination URL" {
				t.Errorf("%s: label %q", url, f[0].Label)
			}
			return f[0].Value
		}
		return ""
	}
	if got := replace("https://go.acme-promo.link/sale"); got != "https://acme.io/pricing/annual" {
		t.Errorf("listed link with a clean destination: %q", got)
	}
	if got := replace("https://bit.ly/3xK9tQ"); got != "https://blog.acme.io/annual-plans" {
		t.Errorf("shortener with a clean destination: %q", got)
	}
	if got := replace("https://go.acme-promo.link/bad"); got != "" {
		t.Errorf("a listed destination is never offered: %q", got)
	}
	if got := replace("https://bit.ly/unknown"); got != "" {
		t.Errorf("a destination that couldn't be looked up is never offered: %q", got)
	}
	for _, l := range r.LinkList {
		want := l.URL == "https://go.acme-promo.link/sale" || l.URL == "https://bit.ly/3xK9tQ"
		if l.FinalClean != want {
			t.Errorf("%s: finalClean %v", l.URL, l.FinalClean)
		}
	}

	// The composer doesn't request links: nothing is known about where they
	// lead, so they are only edited.
	r, err = checker.Check(context.Background(), Input{Subject: "Annual plans", HTML: html})
	if err != nil {
		t.Fatal(err)
	}
	for _, is := range r.Issues {
		if (is.ID == "links.blocklist" || is.ID == "links.shortener") && (len(is.Fixes) != 1 || is.Fixes[0].Kind != "edit_link") {
			t.Errorf("%s without a probe: %+v", is.URL, is.Fixes)
		}
	}
}

// unknownFor is a live blocklist whose lookups of one domain fail.
type unknownFor struct {
	StaticBlocklist
	failing string
}

func (u unknownFor) Check(ctx context.Context, domains []string) (BlocklistReport, error) {
	listed, _ := u.Listings(ctx, domains)
	rep := BlocklistReport{Listed: listed, Unknown: map[string]string{}, Lists: URIListNames, Domains: len(domains)}
	for _, d := range domains {
		if d == u.failing {
			rep.Unknown[d] = TroubleTimeout
		}
	}
	return rep, nil
}
