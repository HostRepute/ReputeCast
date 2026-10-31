package publicpages_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Grow's public pages over plain HTTP, as a browser without JavaScript
// sees them (brief §5.9–5.11, 5.14; PublicReferralHub, PublicReferralInvite,
// PublicRecommendations, PublicGrowStates): Sam Rivera's hub and its
// switches, the invite page and a signup through it, the leaderboard,
// reward downloads, the recommendations page, "While you're here" and
// "Invite friends".
func TestGrowPages(t *testing.T) {
	env, ids := demo(t)
	hubToken := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.ReferralHub, WorkspaceID: ids.ws, ContactID: ids.sam, Link: linktoken.IssueDay(time.Now())})
	hub := "/p/" + wsSlug + "/referrals/" + hubToken

	// Sam's hub: his link, progress, rewards and referrals as numbers and
	// dates only, how counting works, and his two switches.
	r := get(t, env, hub)
	must(t, r, 200, "Your referrals", "sam.rivera@outlook.com", "join/K7Q2MX", "Invite friends to the Weekly Digest", "61 visits to your link so far · 4 signups",
		"1 more referral for the Acme Field Guide", "Referral 3 brings it on", "Sticker pack", "Download", "Referral 4", "Waiting for confirmation",
		"Counts on", "How referrals count", "7 days later, it counts", "Show my first name on my invite page", "Show me on the leaderboard as Sam R.")
	lacks(t, r, "nina.patel", "ivy.walker", "owen.price")
	if r.Header.Get("Cache-Control") != "no-store" || !strings.Contains(r.Header.Get("X-Robots-Tag"), "noindex") {
		t.Errorf("a personal page: %v", r.Header)
	}
	// The Copy button needs the script; without it the field is selectable.
	if !strings.Contains(r.Body, `data-copy="ref-link" hidden`) || !strings.Contains(r.Body, `readonly`) {
		t.Error("the link isn't usable without JavaScript")
	}

	// The download link streams the file.
	dl := regexp.MustCompile(`href="([^"]*/rewards/[^"]+)"`).FindStringSubmatch(r.Body)
	if dl == nil {
		t.Fatal("no download link on the hub")
	}
	// Links in Grow's pages are on the custom domain (news.acme.io).
	u, _ := url.Parse(dl[1])
	d := do(t, env, http.MethodGet, u.Path, nil, hdr{"Host": u.Host})
	if d.Status != 200 || d.Header.Get("Content-Type") != "application/pdf" || d.Header.Get("Content-Length") != "2516582" || !strings.HasPrefix(d.Body, "%PDF-") {
		t.Fatalf("download: %d %v", d.Status, d.Header)
	}

	// The invite page: "A reader" until Sam turns his name on.
	must(t, get(t, env, "/p/"+wsSlug+"/join/K7Q2MX?via=whatsapp"), 200, "A reader invited you to the Acme Weekly Digest", "Weekly Digest",
		"The reader who invited you gets credit when you confirm, and never sees your address.")
	r = do(t, env, http.MethodPost, hub+"/privacy", url.Values{"show_name": {"1"}}, hdr{"Origin": env.Server.URL})
	if r.Status != http.StatusSeeOther || !strings.Contains(r.Header.Get("Location"), "saved=1") {
		t.Fatalf("privacy: %d %v", r.Status, r.Header)
	}
	must(t, get(t, env, hub+"?saved=1"), 200, "Saved.")
	must(t, get(t, env, "/p/"+wsSlug+"/join/K7Q2MX"), 200, "Sam invited you to the Acme Weekly Digest", "Sam gets credit when you confirm your email.")
	// A link nobody has is the normal subscribe page.
	r = get(t, env, "/p/"+wsSlug+"/join/ZZZZZZ")
	must(t, r, 200, "Subscribe")
	lacks(t, r, "invited you")

	// A friend signs up through the invite page: a referral waiting for
	// the confirmation, credited to Sam.
	r = do(t, env, http.MethodPost, "/p/"+wsSlug+"/join/K7Q2MX", url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {"jordan.lee@fastmail.com"},
		"first_name": {"Jordan"}, "consent": {"1"}, "_page": {env.Server.URL + "/p/" + wsSlug + "/join/K7Q2MX?via=whatsapp"}}, hdr{"Origin": env.Server.URL})
	must(t, r, 200, "Check your inbox")
	friend, ok := contactID(t, env, ids.ws, "jordan.lee@fastmail.com")
	if !ok {
		t.Fatal("no contact for the friend")
	}
	if n := count(t, env, `SELECT count(*) FROM referrals r JOIN contacts c ON c.id = r.referrer_id
		WHERE r.contact_id = $1 AND r.status = 'awaiting' AND r.via = 'whatsapp' AND c.email = 'sam.rivera@outlook.com'`, friend); n != 1 {
		t.Fatalf("Jordan's referral: %d", n)
	}
	if n := count(t, env, `SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND list_id = $2 AND status = 'pending'`, friend, ids.digest); n != 1 {
		t.Fatal("referrals always confirm by email")
	}

	// The leaderboard: names only for readers who chose to.
	must(t, get(t, env, "/p/"+wsSlug+"/leaderboard"), 200, "Top referrers", "Omar H.", "Kenji W.", "A reader", "Names show only for readers who chose to.")
	must(t, get(t, env, hub+"/leaderboard"), 200, "Your place", "Everyone else sees you as")

	// An expired hub link offers a fresh one, to the address it was for.
	old := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.ReferralHub, WorkspaceID: ids.ws, ContactID: ids.sam, Link: linktoken.IssueDay(time.Now().AddDate(0, 0, -90))})
	must(t, get(t, env, "/p/"+wsSlug+"/referrals/"+old), 200, "This link has expired", "Email me my link", "@outlook.com")
	must(t, get(t, env, "/p/"+wsSlug+"/referrals"), 200, "Email me my link")

	// The recommendations page: what shows in emails, Visit for links
	// (counted), and an expired link asking for an address.
	recToken := env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Recommendations, WorkspaceID: ids.ws, ContactID: ids.sam, Link: linktoken.IssueDay(time.Now())})
	r = get(t, env, "/p/"+wsSlug+"/recommends/"+recToken)
	must(t, r, 200, "Newsletters Acme recommends", "Ops Weekly", "Subscribing sends your email address to Ops Weekly. They'll email you to confirm.",
		"Fabrikam Engineering Letter", "Opens fabrikam.example. You subscribe on their site.")
	lacks(t, r, "The Deliverability Desk")
	visit := regexp.MustCompile(`href="([^"]*/recommends/visit/[^"]+)"`).FindStringSubmatch(r.Body)
	if visit == nil {
		t.Fatal("no Visit link")
	}
	vu, _ := url.Parse(visit[1])
	before := count(t, env, `SELECT coalesce(sum(clicks), 0)::int FROM recommendation_daily`)
	v := do(t, env, http.MethodGet, vu.Path, nil, hdr{"Host": vu.Host})
	if v.Status != http.StatusFound || v.Header.Get("Location") != "https://fabrikam.example/newsletter" {
		t.Fatalf("visit: %d %v", v.Status, v.Header)
	}
	if after := count(t, env, `SELECT coalesce(sum(clicks), 0)::int FROM recommendation_daily`); after != before+1 {
		t.Fatalf("clicks %d → %d", before, after)
	}
	old = env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Recommendations, WorkspaceID: ids.ws, ContactID: ids.sam, Link: linktoken.IssueDay(time.Now().AddDate(0, 0, -60))})
	must(t, get(t, env, "/p/"+wsSlug+"/recommends/"+old), 200, "Which email address should we use?")

	// "Invite friends" in Sam's preference center.
	prefs := sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil)
	must(t, get(t, env, "/p/"+wsSlug+"/preferences/"+prefs), 200, "Invite friends", "You've referred 2 friends. 1 more and you get the Acme Field Guide.",
		"Your link and rewards")

	// "While you're here" on the confirmation page.
	confirm := sign(env, linktoken.Confirm, ids.ws, friend, uuid.Nil)
	must(t, confirmClick(t, env, "/p/"+wsSlug+"/confirm/"+confirm, nil), 200, "While you're here: newsletters Acme recommends", "Ops Weekly", "Fabrikam Engineering Letter")

	// The app's previews (Referrals › Settings, the set-up guide): inert,
	// and nothing is recorded.
	c := env.Client()
	c.SignInDemo("aisha@acme.io")
	visits := count(t, env, `SELECT coalesce(sum(visits), 0)::int FROM referral_links`)
	for page, text := range map[string]string{"invite": "invited you to", "hub": "Your referrals", "leaderboard": "Top referrers", "recommends": "Newsletters Acme recommends"} {
		var pv struct{ HTML, URL string }
		c.Do("GET", "/api/v1/workspaces/"+wsSlug+"/grow/previews/"+page, nil).MustStatus(t, 200).JSON(t, &pv)
		if !strings.Contains(pv.HTML, text) {
			t.Errorf("%s preview lacks %q", page, text)
		}
		if strings.Contains(pv.HTML, "/referrals/"+hubToken) {
			t.Errorf("%s preview carries a working hub link", page)
		}
	}
	if after := count(t, env, `SELECT coalesce(sum(visits), 0)::int FROM referral_links`); after != visits {
		t.Fatalf("a preview recorded visits: %d → %d", visits, after)
	}
}
