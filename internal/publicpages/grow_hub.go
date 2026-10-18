package publicpages

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

// --- The referrer's hub (/referrals/{token}) ----------------------------------------------

type shareView struct {
	Channel, Label, URL string
}

type cellView struct {
	// State: done (counted), soon (in its hold), todo.
	State string
}

type milestoneView struct {
	Count   int
	Name    string
	Label   string
	Reached bool
	Next    bool
}

type legendView struct {
	Kind, Text string
}

type rewardView struct {
	ID     string
	Name   string
	Line   string
	Badge  string
	BadgeK string // ok, info, warn, mut
	// Action is a link ("Download", "Claim", "Open").
	ActionLabel, ActionHref string
	Code                    string
	Text                    string
	Note                    string
	// Saved is the shipping address the reader gave.
	Saved string
	// AddressAction asks for a shipping address (no address yet).
	AddressAction string
	Done          bool
}

type referralView struct {
	Number int
	Title  string
	Badge  string
	BadgeK string
	Line   string
}

type hubBody struct {
	Email, NotYouHref              string
	ListName, LinkURL, LinkDisplay string
	Share                          []shareView
	VisitsLine                     string
	ProgramOn, LinkActive          bool
	PausedRecord                   string
	Rank                           string
	NextLine, BringsIt             string
	AllEarned                      bool
	Cells                          []cellView
	CellsLabel                     string
	You                            string
	Milestones                     []milestoneView
	Legend                         []legendView
	TrackLabel                     string
	Rewards                        []rewardView
	Referrals                      []referralView
	ReferralsTitle                 string
	HoldDays                       int
	Step3, Rules                   string
	ShowName, ShowBoard, BoardOn   bool
	NameText, BoardLabel           string
	PrivacyAction                  string
	Saved                          bool
	BoardURL, PrefsURL             string
	Publisher                      string
	Proof                          string
}

func (h *Handler) hubClaims(w http.ResponseWriter, r *http.Request) (*site, linktoken.Claims, *referrals.Service, bool) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return nil, linktoken.Claims{}, nil, false
	}
	rs := h.referrals()
	c, ok := h.claims(s, chi.URLParam(r, "token"), linktoken.ReferralHub)
	if rs == nil || !ok {
		h.notFound(w, r, s)
		return nil, linktoken.Claims{}, nil, false
	}
	return s, c, rs, true
}

// hubLink is a hub's own address, from its verified claims: the page a
// form on it goes back to.
func (h *Handler) hubLink(s *site, c linktoken.Claims) string {
	return s.links.Referrals(h.Tokens.Sign(c))
}

func (h *Handler) hubPage(w http.ResponseWriter, r *http.Request) {
	s, c, rs, ok := h.hubClaims(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	if rs.HubExpired(c) {
		h.hubExpired(w, r, s, rs, c, token)
		return
	}
	hv, err := rs.Hub(r.Context(), s.ws, c.ContactID, token, clientIP(r))
	if errors.Is(err, referrals.ErrNoHub) {
		h.notFound(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	b := h.hubBody(s, hv, token)
	b.Saved = r.URL.Query().Get("saved") == "1"
	h.growPage(w, r, s, "grow-hub", http.StatusOK, gt("hub.title", "publisher", s.set.Branding.PublisherName), b, true)
}

func (h *Handler) hubBody(s *site, hv *referrals.HubView, token string) hubBody {
	now := h.now()
	pub := s.set.Branding.PublisherName
	t := hv.Track
	b := hubBody{Email: hv.Email, NotYouHref: s.links.ReferralsRequest(), ListName: hv.ListName, LinkURL: hv.LinkURL, LinkDisplay: hv.Link,
		ProgramOn: hv.ProgramOn, LinkActive: hv.LinkActive, HoldDays: hv.HoldDays, ShowName: hv.ShowName, ShowBoard: hv.ShowBoard, BoardOn: hv.BoardOn,
		PrivacyAction: s.links.Referrals(token) + "/privacy", Publisher: pub,
		VisitsLine: gt("hub.visits", "visits", hv.Visits, "signups", hv.Signups),
		PrefsURL:   s.links.Preferences(h.sign(s, linktoken.Preferences, hv.ContactID, uuid.Nil)),
	}
	for _, sl := range hv.Share {
		b.Share = append(b.Share, shareView{Channel: sl.Channel, Label: gt("hub.channel." + sl.Channel), URL: sl.URL})
	}
	if hv.BoardOn {
		b.BoardURL = s.links.Referrals(token) + "/leaderboard"
		if hv.Rank > 0 {
			b.Rank = gt("hub.rank", "rank", ordinal(hv.Rank))
		}
	}
	if !hv.ProgramOn {
		b.PausedRecord = gt("hub.pausedRecord", "count", t.Counted, "publisher", pub)
	}
	// The next reward and when it comes.
	if t.NextCount > 0 {
		b.NextLine = gt("hub.toNext", "count", t.ToNext, "reward", articled(t.NextName))
		if t.NextBySeq > 0 {
			for _, ref := range hv.Referrals {
				if ref.Number == t.NextBySeq && ref.CountsOn != nil {
					b.BringsIt = gt("hub.bringsIt", "number", ref.Number, "date", shortDate(*ref.CountsOn, now))
				}
			}
		}
		total := t.NextCount
		if total <= 50 {
			for i := 1; i <= total; i++ {
				st := "todo"
				switch {
				case int64(i) <= t.Counted:
					st = "done"
				case int64(i) <= t.Counted+t.InHold:
					st = "soon"
				}
				b.Cells = append(b.Cells, cellView{State: st})
			}
		}
	} else if len(t.Points) > 0 {
		b.AllEarned = true
	}
	b.You = gt("hub.you", "count", t.Counted)
	var labels []string
	for _, p := range t.Points {
		mv := milestoneView{Count: p.Count, Name: p.Reward, Label: gt("hub.milestone", "count", p.Count), Reached: int64(p.Count) <= t.Counted,
			Next: p.Count == t.NextCount}
		b.Milestones = append(b.Milestones, mv)
		l := p.Reward
		if mv.Reached {
			l += " (reached)"
		}
		labels = append(labels, mv.Label+" "+l)
	}
	b.TrackLabel = gt("hub.trackLabel", "counted", t.Counted, "milestones", strings.Join(labels, ", "))
	b.Legend = append(b.Legend, legendView{Kind: "done", Text: gt("hub.legendCounted", "count", t.Counted)})
	if t.InHold > 0 {
		date := ""
		if t.NextHold != nil {
			date = shortDate(*t.NextHold, now)
		}
		b.Legend = append(b.Legend, legendView{Kind: "soon", Text: gt("hub.legendSoon", "count", t.InHold, "date", date)})
	}
	if t.Awaiting > 0 {
		b.Legend = append(b.Legend, legendView{Kind: "todo", Text: gt("hub.legendAwaiting", "count", t.Awaiting)})
	}
	if t.Held > 0 {
		b.Legend = append(b.Legend, legendView{Kind: "todo", Text: gt("hub.legendHeld", "count", t.Held)})
	}
	for _, rw := range hv.Rewards {
		b.Rewards = append(b.Rewards, h.rewardView(s, rw, hv, token, now))
	}
	// Referrals, newest first.
	b.ReferralsTitle = gt("hub.referrals", "count", len(hv.Referrals))
	for i := len(hv.Referrals) - 1; i >= 0; i-- {
		ref := hv.Referrals[i]
		v := referralView{Number: ref.Number, Title: gt("hub.referral", "number", ref.Number)}
		confirmed := ""
		if ref.ConfirmedAt != nil {
			confirmed = shortDate(*ref.ConfirmedAt, now)
		}
		switch ref.Status {
		case referrals.StatusCounted:
			v.Badge, v.BadgeK = gt("hub.rCounted"), "ok"
			counted := ""
			if ref.CountedAt != nil {
				counted = shortDate(*ref.CountedAt, now)
			}
			v.Line = gt("hub.rlCounted", "confirmed", confirmed, "counted", counted)
		case referrals.StatusHold:
			date := ""
			if ref.CountsOn != nil {
				date = shortDate(*ref.CountsOn, now)
			}
			v.Badge, v.BadgeK = gt("hub.rCountsOn", "date", date), "info"
			v.Line = gt("hub.rlHold", "confirmed", confirmed, "date", date)
		case referrals.StatusReview:
			v.Badge, v.BadgeK = gt("hub.rHeld"), "warn"
			v.Line = gt("hub.rlHeld", "confirmed", confirmed)
		case referrals.StatusAwaiting:
			v.Badge, v.BadgeK = gt("hub.rAwaiting"), "mut"
			v.Line = gt("hub.rlAwaiting", "date", shortDate(ref.SignedUpAt, now))
		default:
			v.Badge, v.BadgeK = gt("hub.rNotCounted"), "mut"
			v.Line = gt("hub.reason." + ref.Reason)
		}
		b.Referrals = append(b.Referrals, v)
	}
	// How counting works, from the program's rules.
	b.Step3 = gt("hub.step3", "days", hv.HoldDays)
	excluded := []string{gt("hub.excludedOwn")}
	if hv.SameIP {
		excluded = append(excluded, gt("hub.excludedNetwork"))
	}
	if hv.Disposable {
		excluded = append(excluded, gt("hub.excludedThrowaway"))
	}
	b.Rules = gt("hub.rules", "days", hv.HoldDays) + " " + gt("hub.rulesExcluded", "excluded", orList(excluded))
	if hv.FirstName != "" {
		b.NameText = gt("hub.showNameText", "name", hv.FirstName)
	} else {
		b.NameText = gt("hub.showNameNoName")
	}
	if hv.ShortName != "" {
		b.BoardLabel = gt("hub.showBoard", "name", hv.ShortName)
	} else {
		b.BoardLabel = gt("hub.showBoardAnon")
	}
	return b
}

// orList is "a, b, or c" (English).
func orList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
}

func (h *Handler) rewardView(s *site, rw referrals.HubReward, hv *referrals.HubView, token string, now time.Time) rewardView {
	pub := s.set.Branding.PublisherName
	v := rewardView{ID: rw.ID.String(), Name: rw.Name}
	file := ""
	if rw.FileName != "" {
		file = fileDesc(rw.FileType, rw.FileSize)
	}
	if !rw.Reached {
		if file != "" {
			v.Line = gt("hub.rewardAtFile", "count", rw.Count, "file", file)
		} else {
			v.Line = gt("hub.rewardAt", "count", rw.Count)
		}
		if !hv.ProgramOn {
			v.Line = gt("hub.rewardAt", "count", rw.Count) + " · " + gt("hub.pausedReward")
			v.Badge, v.BadgeK = gt("hub.statusPaused"), "mut"
			return v
		}
		v.Badge, v.BadgeK = gt("hub.rewardToGo", "count", max(rw.ToGo, 1)), "mut"
		return v
	}
	sent := ""
	if rw.SentAt != nil {
		sent = shortDate(*rw.SentAt, now)
	}
	switch rw.Status {
	case referrals.RewardSent, referrals.RewardFulfilled:
		v.Done = true
		switch rw.Type {
		case referrals.RewardDownload:
			if file != "" {
				v.Line = gt("hub.rewardSentFile", "date", sent, "file", file)
			} else {
				v.Line = gt("hub.rewardSent", "date", sent)
			}
			v.ActionLabel, v.ActionHref = gt("hub.download"), rw.DownloadURL
		case referrals.RewardCoupon:
			if rw.ExpiresAt != nil {
				v.Line = gt("hub.claimBy", "date", shortDate(*rw.ExpiresAt, now))
			} else {
				v.Line = gt("hub.claimNoDate")
			}
			v.ActionLabel, v.ActionHref, v.Code = gt("hub.claim"), rw.ClaimURL, rw.Code
		case referrals.RewardManual:
			v.Badge, v.BadgeK = gt("hub.statusSent"), "ok"
			if rw.Address != nil && rw.Address.Name != "" {
				v.Line = gt("hub.sentTo", "date", sent, "name", rw.Address.Name, "city", rw.Address.City, "country", rw.Address.Country)
			} else {
				v.Line = gt("hub.sentPlain", "date", sent)
			}
			v.Note = rw.Note
		default:
			v.Line = gt("hub.rewardSent", "date", sent)
			v.Code = rw.Code
			if rw.AnswerURL != "" {
				v.ActionLabel, v.ActionHref = gt("hub.open"), rw.AnswerURL
			}
		}
	case referrals.RewardWaitingCodes:
		v.Badge, v.BadgeK, v.Text = gt("hub.onItsWay"), "info", gt("hub.waitingCodes", "publisher", pub)
	case referrals.RewardToFulfill:
		v.Badge, v.BadgeK = gt("hub.onItsWay"), "info"
		if rw.AskAddress && rw.Address == nil {
			v.Text = gt("hub.addressAsk", "publisher", pub)
			v.AddressAction = s.links.Referrals(token) + "/address/" + rw.ID.String()
		} else {
			v.Text = gt("hub.toFulfill", "publisher", pub)
			if rw.Address != nil {
				v.Saved = gt("hub.addressSaved", "name", rw.Address.Name, "city", rw.Address.City, "country", rw.Address.Country)
			}
		}
	case referrals.RewardPending:
		v.Badge, v.BadgeK, v.Text = gt("hub.onItsWay"), "info", gt("hub.pending")
	case referrals.RewardFailed:
		v.Badge, v.BadgeK, v.Text = gt("hub.onItsWay"), "warn", gt("hub.failed", "publisher", pub)
	case referrals.RewardExpired:
		v.Text = gt("hub.expired")
	}
	return v
}

// hubRequestBody is the hub's expired link, the "Email me my link" page and
// its answer.
type hubRequestBody struct {
	Heading, Text, Masked, Tip string
	Action, Button             string
	// Ask shows an address field (no token); Sent the answer.
	Ask   bool
	Sent  bool
	Error string
	Email string
}

func (h *Handler) hubExpired(w http.ResponseWriter, r *http.Request, s *site, rs *referrals.Service, c linktoken.Claims, token string) {
	var list string
	if hv, err := rs.Hub(r.Context(), s.ws, c.ContactID, "", nil); err == nil {
		list = hv.ListName
	}
	b := hubRequestBody{Heading: gt("expired.title"), Text: gt("expired.text"), Masked: gt("expired.to", "email", rs.MaskedEmail(r.Context(), c.ContactID)),
		Action: s.links.Referrals(token) + "/send", Button: gt("expired.button")}
	if list != "" {
		b.Tip = gt("expired.tip", "list", list)
	}
	h.growPage(w, r, s, "grow-request", http.StatusOK, b.Heading+" · "+s.set.Branding.PublisherName, b, true)
}

// hubSend emails a fresh hub link to the address an expired link was for.
func (h *Handler) hubSend(w http.ResponseWriter, r *http.Request) {
	s, c, rs, ok := h.hubClaims(w, r)
	if !ok {
		return
	}
	masked := rs.MaskedEmail(r.Context(), c.ContactID)
	b := hubRequestBody{Heading: gt("sent.title"), Sent: true, Text: gt("sent.text", "email", masked)}
	if !h.Forms.Allow("hub-link:"+c.ContactID.String(), time.Minute) {
		h.growPage(w, r, s, "grow-request", http.StatusOK, b.Heading+" · "+s.set.Branding.PublisherName, b, true)
		return
	}
	if err := rs.SendHubLink(r.Context(), s.ws, s.set, c.ContactID); err != nil {
		h.log().WarnContext(r.Context(), "send a hub link", "err", err)
		b.Text = gt("sent.unavailable")
	}
	h.growPage(w, r, s, "grow-request", http.StatusOK, b.Heading+" · "+s.set.Branding.PublisherName, b, true)
}

func (h *Handler) hubRequestPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.referrals() == nil {
		h.notFound(w, r, s)
		return
	}
	b := hubRequestBody{Heading: gt("request.title"), Text: gt("request.text"), Ask: true, Action: s.links.ReferralsRequest(), Button: gt("request.button")}
	h.growPage(w, r, s, "grow-request", http.StatusOK, b.Heading+" · "+s.set.Branding.PublisherName, b, false)
}

// hubRequestPost emails a hub link to a reader by address. The answer is
// the same whether or not the address has a hub.
func (h *Handler) hubRequestPost(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.referrals()
	if rs == nil {
		h.notFound(w, r, s)
		return
	}
	if !h.parseForm(w, r, 8<<10) {
		http.Error(w, "The form couldn't be read.", http.StatusBadRequest)
		return
	}
	email, problem := emailaddr.Check(r.PostForm.Get("email"))
	if problem != "" {
		b := hubRequestBody{Heading: gt("request.title"), Text: gt("request.text"), Ask: true, Action: s.links.ReferralsRequest(), Button: gt("request.button"),
			Error: gt("request.invalid"), Email: r.PostForm.Get("email")}
		h.growPage(w, r, s, "grow-request", http.StatusUnprocessableEntity, b.Heading+" · "+s.set.Branding.PublisherName, b, true)
		return
	}
	b := hubRequestBody{Heading: gt("sent.title"), Sent: true, Text: gt("sent.text", "email", email)}
	ip := ""
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	if h.Forms.Allow("hub-request:"+email, time.Minute) && h.Forms.AllowPerMinute("hub-request-ip:"+ip, 5) {
		var contact uuid.UUID
		if err := h.Pool.QueryRow(r.Context(), `SELECT c.id FROM contacts c JOIN referral_links rl ON rl.contact_id = c.id
			WHERE c.workspace_id = $1 AND c.email = $2 AND c.erased_at IS NULL LIMIT 1`, s.ws.ID, email).Scan(&contact); err == nil {
			if err := rs.SendHubLink(r.Context(), s.ws, s.set, contact); err != nil {
				h.log().WarnContext(r.Context(), "send a hub link", "err", err)
			}
		}
	}
	h.growPage(w, r, s, "grow-request", http.StatusOK, b.Heading+" · "+s.set.Branding.PublisherName, b, true)
}

// hubPrivacy saves the reader's two switches.
func (h *Handler) hubPrivacy(w http.ResponseWriter, r *http.Request) {
	s, c, rs, ok := h.hubClaims(w, r)
	if !ok {
		return
	}
	token := chi.URLParam(r, "token")
	if rs.HubExpired(c) || !h.parseForm(w, r, 4<<10) {
		http.Redirect(w, r, h.hubLink(s, c), http.StatusSeeOther) //nolint:gosec // our own origin, signed from the verified token
		return
	}
	hv, err := rs.Hub(r.Context(), s.ws, c.ContactID, token, nil)
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	if err := rs.SetChoices(r.Context(), hv.LinkID, r.PostForm.Get("show_name") != "", r.PostForm.Get("show_board") != ""); err != nil {
		h.serverError(w, r, s, err)
		return
	}
	http.Redirect(w, r, h.hubLink(s, c)+"?saved=1#privacy", http.StatusSeeOther) //nolint:gosec // our own origin, signed from the verified token
}

// hubAddress saves where a manual reward goes.
func (h *Handler) hubAddress(w http.ResponseWriter, r *http.Request) {
	s, c, rs, ok := h.hubClaims(w, r)
	if !ok {
		return
	}
	reward, err := uuid.Parse(chi.URLParam(r, "reward"))
	if err != nil || rs.HubExpired(c) || !h.parseForm(w, r, 8<<10) {
		http.Redirect(w, r, h.hubLink(s, c), http.StatusSeeOther) //nolint:gosec // our own origin, signed from the verified token
		return
	}
	f := r.PostForm
	a := referrals.ReferralShippingAddress{Name: clip(f.Get("name"), 200), Street: clip(f.Get("street"), 500), City: clip(f.Get("city"), 200), Country: clip(f.Get("country"), 100)}
	if strings.TrimSpace(a.Name) != "" && strings.TrimSpace(a.Street) != "" {
		if err := rs.SaveAddress(r.Context(), c.ContactID, reward, a); err != nil {
			h.notFound(w, r, s)
			return
		}
	}
	http.Redirect(w, r, h.hubLink(s, c)+"#rewards", http.StatusSeeOther) //nolint:gosec // our own origin, signed from the verified token
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// --- Reward downloads (/rewards/{token}) ------------------------------------------------

func (h *Handler) rewardDownload(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.referrals()
	if rs == nil || h.Tokens == nil {
		h.notFound(w, r, s)
		return
	}
	c, err := h.Tokens.Verify(chi.URLParam(r, "token"), linktoken.RewardDownload)
	if err != nil || c.WorkspaceID != s.ws.ID {
		h.notFound(w, r, s)
		return
	}
	d, err := rs.Download(r.Context(), c)
	if errors.Is(err, referrals.ErrLinkExpired) {
		h.growMessage(w, r, s, http.StatusGone, gt("download.title"), gt("download.text"),
			[]action{{Label: gt("download.button"), Href: s.links.Referrals(rs.HubToken(s.ws.ID, c.ContactID, h.now()))}})
		return
	}
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	if d.FileID == uuid.Nil {
		if d.URL == "" {
			h.notFound(w, r, s)
			return
		}
		http.Redirect(w, r, d.URL, http.StatusFound) //nolint:gosec // the reward's download address, set by the team
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", d.ContentType)
	hd.Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(d.FileName, `"`, "")+`"`)
	hd.Set("Content-Length", formatInt(d.Size))
	hd.Set("Cache-Control", "no-store")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("X-Robots-Tag", "noindex, nofollow")
	hd.Set("Referrer-Policy", "no-referrer")
	w.WriteHeader(http.StatusOK)
	if err := rs.FileChunks(r.Context(), d.FileID, func(b []byte) error { _, err := w.Write(b); return err }); err != nil {
		h.log().WarnContext(r.Context(), "reward download", "err", err)
	}
}

func formatInt(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{digits[n%10]}, b...)
		n /= 10
	}
	return string(b)
}

// --- The leaderboard (/leaderboard, /referrals/{token}/leaderboard) --------------------

type boardRow struct {
	Rank    int
	Name    string
	Counted int
	Me      bool
}

type boardBody struct {
	ListName  string
	Month     bool
	MonthHref string
	AllHref   string
	Since     string
	Rows      []boardRow
	ListLabel string
	More      string
	Me        *boardRow
	MeText    string
	HubURL    string
	Empty     bool
}

func (h *Handler) boardPage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	rs := h.referrals()
	if rs == nil {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	var me *uuid.UUID
	var meName string
	token := chi.URLParam(r, "token")
	if token != "" {
		c, ok := h.claims(s, token, linktoken.ReferralHub)
		if !ok || rs.HubExpired(c) {
			h.notFound(w, r, s)
			return
		}
		hv, err := rs.Hub(ctx, s.ws, c.ContactID, token, nil)
		if err != nil {
			h.notFound(w, r, s)
			return
		}
		id := hv.LinkID
		me = &id
		if hv.ShowBoard {
			meName = hv.ShortName
		}
	}
	var programID uuid.UUID
	var list string
	var started *time.Time
	err := h.Pool.QueryRow(ctx, `SELECT p.id, l.name, p.first_on_at FROM referral_programs p JOIN lists l ON l.id = p.list_id
		WHERE p.workspace_id = $1 AND p.deleted_at IS NULL AND p.leaderboard_on AND p.status = 'on' ORDER BY p.created_at LIMIT 1`, s.ws.ID).
		Scan(&programID, &list, &started)
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	p, err := h.Q.ReferralProgramByID(ctx, programID)
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	b, err := h.boardBody(ctx, s, rs, p, list, started, r.URL.Query().Get("period") == "month", me, meName, token)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.growPage(w, r, s, "grow-board", http.StatusOK, gt("board.title")+" · "+s.set.Branding.PublisherName, b, token != "")
}

// boardBody is the leaderboard of a program; me is the referrer whose hub
// opened it (their row and rank), if any.
func (h *Handler) boardBody(ctx context.Context, s *site, rs *referrals.Service, p store.ReferralProgram, list string, started *time.Time, month bool,
	me *uuid.UUID, meName, token string) (boardBody, error) {
	period := referrals.PeriodAll
	if month {
		period = referrals.PeriodMonth
	}
	lb, err := rs.Leaderboard(ctx, p, period, me)
	if err != nil {
		return boardBody{}, err
	}
	base := s.links.Leaderboard()
	if token != "" {
		base = s.links.Referrals(token) + "/leaderboard"
	}
	b := boardBody{ListName: list, Month: month, MonthHref: base + "?period=month", AllHref: base, Empty: len(lb.Rows) == 0}
	now := h.now()
	if started != nil && lb.ComputedAt != nil {
		b.Since = gt("board.since", "date", shortDate(*started, now), "time", lb.ComputedAt.In(location(s.ws)).Format("15:04"))
	}
	for _, row := range lb.Rows {
		name := row.Name
		if name == "" {
			name = gt("board.anon")
		}
		b.Rows = append(b.Rows, boardRow{Rank: row.Rank, Name: name, Counted: row.Counted, Me: row.Me})
	}
	b.ListLabel = gt("board.listLabel", "count", len(b.Rows))
	if lb.More > 0 {
		b.More = gt("board.more", "count", lb.More, "referrals", lb.MoreCount)
	}
	if me != nil {
		b.HubURL = s.links.Referrals(token)
		if lb.Me != nil {
			b.Me = &boardRow{Rank: lb.Me.Rank, Name: gt("board.you"), Counted: lb.Me.Counted, Me: true}
			if meName != "" {
				b.MeText = gt("board.youNamed", "name", meName)
			} else {
				b.MeText = gt("board.youAnon")
			}
		} else {
			b.MeText = gt("board.youNone")
		}
	}
	return b, nil
}
