package forms

// Grow's public pages (internal/referrals, internal/recommendations): the
// invite page of a referral link, the referrer's hub, its downloads, the
// leaderboard and the recommendations page. Like the other pages they live
// on the workspace's custom domain when it has one.

// Join is a referral link's invite page: news.acme.io/join/K7Q2MX.
func (l PageLinks) Join(code string) string { return l.page("join", code) }

// Referrals is a referrer's hub, opened from a signed link.
func (l PageLinks) Referrals(token string) string { return l.page("referrals", token) }

// ReferralsRequest asks for a fresh hub link by email.
func (l PageLinks) ReferralsRequest() string { return l.page("referrals") }

// Leaderboard is the program's public leaderboard.
func (l PageLinks) Leaderboard() string { return l.page("leaderboard") }

// Download is a reward file, behind an expiring signed link.
func (l PageLinks) Download(token string) string { return l.page("rewards", token) }

// Recommends is the recommendations page; with a token it knows the reader.
func (l PageLinks) Recommends(token string) string {
	if token == "" {
		return l.page("recommends")
	}
	return l.page("recommends", token)
}

// RecommendsVisit counts a click on a recommended newsletter's link and
// sends the reader there.
func (l PageLinks) RecommendsVisit(id string) string { return l.page("recommends", "visit", id) }
