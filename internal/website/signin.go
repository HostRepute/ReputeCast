package website

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// Members sign in without passwords (brief §3.2): a link and a 6-digit
// code by email, on the System stream, in the contact's language.
//
//   - A link works for the minutes set in Access, once; a newer link
//     replaces older ones. Opening it shows a page with Continue: link
//     scanners that fetch the address don't use it up.
//   - The code is typed in the browser that asked for it (its sign-in
//     cookie) and allows 3 tries.
//   - 5 links per address an hour, and limits per IP address; past the
//     soft limit Turnstile asks whether it's a person, when it is set up.
//   - Every answer to "send me a link" is the same, whether or not the
//     address belongs to a member; suppressed addresses get no link.

// Link purposes.
const (
	PurposeSignin = "signin"
	PurposeSignup = "signup"
	PurposeEmail  = "email"
)

// Limits of sign-in links.
const (
	MaxLinksPerHour = 5
	CodeTries       = 3
	// SignupLinkDays is how long a sign-up's confirmation link works.
	SignupLinkDays = 7
	// ResendAfter is how long "Send again" waits.
	ResendAfter = 60 * time.Second
	// The per-IP limits: past the soft one Turnstile asks (or, without it,
	// the request waits); past the hard one nothing is sent.
	ipSoftLimit  = 10
	ipSoftWindow = 10 * time.Minute
	ipHardLimit  = 60
	ipHardWindow = time.Hour
)

// Sign-in outcomes.
const (
	SigninSent      = "sent"
	SigninInvalid   = "invalid"
	SigninLimited   = "limited"
	SigninChallenge = "challenge"
	SigninOff       = "off"
)

// ipLimiter counts requests per IP address in this process.
type ipLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

// note records a request from ip and reports whether it passes the soft
// and the hard limits.
func (l *ipLimiter) note(ip string, now time.Time) (soft, hard bool) {
	if ip == "" {
		return true, true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hits == nil {
		l.hits = map[string][]time.Time{}
	}
	kept := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if now.Sub(t) < ipHardWindow {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	l.hits[ip] = kept
	recent := 0
	for _, t := range kept {
		if now.Sub(t) < ipSoftWindow {
			recent++
		}
	}
	if len(l.hits) > 50000 {
		// A flood of addresses: forget the oldest window.
		l.hits = map[string][]time.Time{ip: kept}
	}
	return recent <= ipSoftLimit, len(kept) <= ipHardLimit
}

// browserHash is the stored form of a browser's sign-in cookie.
func browserHash(cookie string) []byte {
	sum := sha256.Sum256([]byte("signin:" + cookie))
	return sum[:]
}

// sixDigits is a random 6-digit code.
func sixDigits() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// CodeText is a code as the email shows it ("482 913").
func CodeText(code string) string {
	if len(code) == 6 {
		return code[:3] + " " + code[3:]
	}
	return code
}

// SafeReturn keeps a return path on the site: a root-relative path, not
// another host.
func SafeReturn(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.Contains(p, "\\") || len(p) > 500 {
		return ""
	}
	// Browsers drop tabs and line breaks from addresses ("/\t/x" opens
	// "//x"), so no control characters at all.
	if strings.IndexFunc(p, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return ""
	}
	return p
}

// SigninInput asks for a sign-in link.
type SigninInput struct {
	Email    string
	ReturnTo string
	// Browser is the browser's sign-in cookie (the code is typed there).
	Browser   string
	IP        *netip.Addr
	Turnstile string
	// Verify checks a Turnstile answer (nil: Turnstile isn't set up).
	Verify func(answer string) bool
}

// SigninOutcome is the answer to "send me a link".
type SigninOutcome struct {
	Status  string
	Email   string
	RetryIn time.Duration
	// Minutes is how long links work.
	Minutes int
	Message string
}

func ipKey(ip *netip.Addr) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

// RequestSignin sends a sign-in link and code when the address belongs
// to a member who can get one. The answer is the same either way.
func (s *Service) RequestSignin(ctx context.Context, ws *store.Workspace, in SigninInput) (SigninOutcome, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return SigninOutcome{}, err
	}
	out := SigninOutcome{Minutes: int(c.row.LinkMinutes)}
	if !c.On() || !c.row.SigninOn {
		out.Status = SigninOff
		return out, nil
	}
	email, problem := emailaddr.Check(in.Email)
	if problem != "" {
		out.Status, out.Message = SigninInvalid, tr(ctx, "signin.emailInvalid", nil)
		return out, nil
	}
	out.Email = email
	if st := s.ipCheck(in.IP, in.Turnstile, in.Verify); st != "" {
		out.Status = st
		if st == SigninLimited {
			out.RetryIn = time.Hour
		}
		return out, nil
	}
	out.Status = SigninSent
	contact, ok, err := s.Audience.FindPublicContact(ctx, s.Q, ws, email)
	if err != nil || !ok {
		return out, err
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, contact)
	if errors.Is(err, audience.ErrContactGone) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if blocked, err := s.blocked(ctx, ws, pc); err != nil || blocked {
		if blocked {
			_ = s.Q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: contact, Kind: "link_refused",
				Detail: memberEventDetail("memberEvent.suppressed"), At: s.now()})
		}
		return out, err
	}
	if wait, err := s.addressLimit(ctx, contact); err != nil || wait > 0 {
		if wait > 0 {
			out.Status, out.RetryIn = SigninLimited, wait
		}
		return out, err
	}
	err = s.sendLink(ctx, ws, c, pc, linkRequest{purpose: PurposeSignin, browser: in.Browser, returnTo: SafeReturn(in.ReturnTo),
		expires: time.Duration(c.row.LinkMinutes) * time.Minute})
	return out, err
}

// ipCheck applies the per-IP limits: "" passes.
func (s *Service) ipCheck(ip *netip.Addr, answer string, verify func(string) bool) string {
	soft, hard := s.ips.note(ipKey(ip), s.now())
	switch {
	case !hard:
		return SigninLimited
	case soft:
		return ""
	case verify == nil:
		return SigninLimited
	case answer == "":
		return SigninChallenge
	case !verify(answer):
		return SigninChallenge
	}
	return ""
}

// blocked reports a contact who must get no email: bounced, complained,
// or suppressed for a reason other than an unsubscribe.
func (s *Service) blocked(ctx context.Context, ws *store.Workspace, pc audience.PublicContact) (bool, error) {
	switch pc.Status {
	case "bounced", "complained", "suppressed":
		return true, nil
	}
	sup, err := s.Q.SignupSuppression(ctx, store.SignupSuppressionParams{WorkspaceID: ws.ID, Email: pc.Email})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sup.Reason != "unsubscribe", nil
}

// addressLimit is how long an address waits before its next link (0: now).
func (s *Service) addressLimit(ctx context.Context, contact uuid.UUID) (time.Duration, error) {
	since := s.now().Add(-time.Hour)
	n, err := s.Q.WebsiteLinksLastHour(ctx, store.WebsiteLinksLastHourParams{ContactID: contact, Since: since})
	if err != nil || n < MaxLinksPerHour {
		return 0, err
	}
	oldest, err := s.Q.WebsiteLinkOldestInHour(ctx, store.WebsiteLinkOldestInHourParams{ContactID: contact, Since: since})
	if err != nil {
		return 0, err
	}
	wait := oldest.Add(time.Hour).Sub(s.now())
	if wait < time.Minute {
		wait = time.Minute
	}
	return wait, nil
}

// linkRequest is one link to send.
type linkRequest struct {
	purpose  string
	browser  string
	returnTo string
	newEmail string
	lists    []uuid.UUID
	expires  time.Duration
	// postTitle names the post a sign-up came from.
	postTitle string
	listNames []string
}

// sendLink stores a link and its code (their hashes) and emails them; a
// newer link replaces the contact's older ones of the same purpose.
func (s *Service) sendLink(ctx context.Context, ws *store.Workspace, c *config, pc audience.PublicContact, r linkRequest) error {
	token, code := Token(), sixDigits()
	now := s.now()
	id := newID()
	lists := r.lists
	if lists == nil {
		lists = []uuid.UUID{}
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsiteLinkInsert(ctx, store.WebsiteLinkInsertParams{ID: id, WorkspaceID: ws.ID, ContactID: pc.ID, Purpose: r.purpose,
			TokenHash: hash(token), CodeHash: hash(code), BrowserHash: browserHash(r.browser), ReturnTo: r.returnTo, NewEmail: r.newEmail, Lists: lists,
			At: now, ExpiresAt: now.Add(r.expires)}); err != nil {
			return err
		}
		return q.WebsiteLinkReplaceOlder(ctx, store.WebsiteLinkReplaceOlderParams{At: &now, ContactID: pc.ID, Purpose: r.purpose, ID: id})
	})
	if err != nil {
		return err
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return err
	}
	to := pc.Email
	if r.purpose == PurposeEmail {
		to = r.newEmail
	}
	loc, _ := i18n.ParseLocale(pc.Language)
	msg := s.linkEmail(ctx, ws, c, set, i18n.For(loc), linkEmail{purpose: r.purpose, to: to, first: pc.FirstName, link: s.sitePath(ctx, ws, "signin", "link", token),
		code: code, minutes: int(r.expires / time.Minute), postTitle: r.postTitle, lists: r.listNames})
	msg.WorkspaceID, msg.To, msg.ToName = &ws.ID, to, pc.FirstName
	if _, err := s.mail().SendSystem(ctx, msg); err != nil {
		// The answer stays the same: the reader can ask again.
		s.log().WarnContext(ctx, "website sign-in email", "purpose", r.purpose, "err", err)
	}
	return nil
}

func (s *Service) mail() mailer.Sender {
	if s.Mail != nil {
		if m := s.Mail(); m != nil {
			return m
		}
	}
	return mailer.Unavailable{}
}

// SigninDone is a reader just signed in: the session's cookie and where
// they go next.
type SigninDone struct {
	Token     string
	ExpiresAt time.Time
	Email     string
	ReturnTo  string
	Purpose   string
	// Lists were confirmed by a sign-up.
	Lists []uuid.UUID
	// SameBrowser: the link was opened where it was asked for.
	SameBrowser bool
	// Taken: an email change whose new address another contact holds.
	Taken bool
}

// Code outcomes.
const (
	CodeOK      = "ok"
	CodeWrong   = "wrong"
	CodeExpired = "expired"
)

// CodeOutcome is the answer to a typed code.
type CodeOutcome struct {
	Status    string
	TriesLeft int
	Email     string
	Done      *SigninDone
}

// SigninWithCode checks a code typed in the browser that asked for it.
func (s *Service) SigninWithCode(ctx context.Context, ws *store.Workspace, browser, code string, ev audience.PublicEvidence) (CodeOutcome, error) {
	code = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, code)
	link, err := s.Q.WebsiteLinkLatestForBrowser(ctx, store.WebsiteLinkLatestForBrowserParams{WorkspaceID: ws.ID, BrowserHash: browserHash(browser)})
	if errors.Is(err, pgx.ErrNoRows) {
		return CodeOutcome{Status: CodeExpired}, nil
	}
	if err != nil {
		return CodeOutcome{}, err
	}
	out := CodeOutcome{Email: s.contactEmail(ctx, link.ContactID)}
	now := s.now()
	if link.UsedAt != nil || link.ReplacedAt != nil || !link.ExpiresAt.After(now) || link.CodeTries >= CodeTries {
		out.Status = CodeExpired
		return out, nil
	}
	if len(code) != 6 || subtle.ConstantTimeCompare(hash(code), link.CodeHash) != 1 {
		tries, err := s.Q.WebsiteLinkTry(ctx, link.ID)
		if err != nil {
			return out, err
		}
		out.Status, out.TriesLeft = CodeWrong, max(CodeTries-int(tries), 0)
		if out.TriesLeft == 0 {
			out.Status = CodeExpired
		}
		return out, nil
	}
	done, err := s.finish(ctx, ws, link, true, "code", ev)
	if err != nil {
		return out, err
	}
	if done == nil {
		out.Status = CodeExpired
		return out, nil
	}
	out.Status, out.Done = CodeOK, done
	return out, nil
}

func (s *Service) contactEmail(ctx context.Context, id uuid.UUID) string {
	var e string
	_ = s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1`, id).Scan(&e)
	return e
}

// Link states.
const (
	LinkOK       = "ok"
	LinkExpired  = "expired"
	LinkUsed     = "used"
	LinkReplaced = "replaced"
	LinkUnknown  = "unknown"
)

// LinkState is what a sign-in link can do. Reading it uses nothing up.
type LinkState struct {
	Status   string
	Email    string
	Purpose  string
	ReturnTo string
	Minutes  int
	// SameBrowser: opened where it was asked for.
	SameBrowser bool
}

// Link describes a sign-in link (the page it opens).
func (s *Service) Link(ctx context.Context, ws *store.Workspace, token, browser string) (LinkState, error) {
	if !validToken(token) {
		return LinkState{Status: LinkUnknown}, nil
	}
	link, err := s.Q.WebsiteLinkByToken(ctx, hash(token))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && link.WorkspaceID != ws.ID) {
		return LinkState{Status: LinkUnknown}, nil
	}
	if err != nil {
		return LinkState{}, err
	}
	out := LinkState{Status: LinkOK, Purpose: link.Purpose, ReturnTo: link.ReturnTo, Minutes: int(link.ExpiresAt.Sub(link.CreatedAt) / time.Minute),
		Email: s.contactEmail(ctx, link.ContactID), SameBrowser: subtle.ConstantTimeCompare(browserHash(browser), link.BrowserHash) == 1}
	if link.Purpose == PurposeEmail {
		out.Email = link.NewEmail
	}
	switch {
	case link.UsedAt != nil:
		out.Status = LinkUsed
	case link.ReplacedAt != nil:
		out.Status = LinkReplaced
	case !link.ExpiresAt.After(s.now()):
		out.Status = LinkExpired
	}
	return out, nil
}

// SigninWithLink uses a link (the Continue of the page it opens).
func (s *Service) SigninWithLink(ctx context.Context, ws *store.Workspace, token, browser string, ev audience.PublicEvidence) (LinkState, *SigninDone, error) {
	st, err := s.Link(ctx, ws, token, browser)
	if err != nil || st.Status != LinkOK {
		return st, nil, err
	}
	link, err := s.Q.WebsiteLinkByToken(ctx, hash(token))
	if err != nil {
		return st, nil, err
	}
	method := "link"
	if link.Purpose == PurposeSignup {
		method = "signup"
	}
	done, err := s.finish(ctx, ws, link, st.SameBrowser, method, ev)
	if err != nil {
		return st, nil, err
	}
	if done == nil {
		// Used a moment ago (another tab, a double click).
		st.Status = LinkUsed
		return st, nil, nil
	}
	return st, done, nil
}

// finish uses a link once and signs its contact in: a sign-up confirms
// its lists, an email change moves the address. nil: the link was used,
// replaced or expired meanwhile.
func (s *Service) finish(ctx context.Context, ws *store.Workspace, link store.WebsiteSigninLink, same bool, method string, ev audience.PublicEvidence) (*SigninDone, error) {
	now := s.now()
	if _, err := s.Q.WebsiteLinkUse(ctx, store.WebsiteLinkUseParams{At: &now, SameBrowser: &same, ID: link.ID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	done := &SigninDone{ReturnTo: link.ReturnTo, Purpose: link.Purpose, SameBrowser: same}
	switch link.Purpose {
	case PurposeSignup:
		if len(link.Lists) > 0 {
			res, err := s.Forms.Confirm(ctx, ws, link.ContactID, ev)
			if err != nil {
				return nil, err
			}
			done.Lists = res.Confirmed
		}
		if err := s.Q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: link.ContactID,
			Kind: "joined", Method: "signup", SameBrowser: &same, Detail: link.ReturnTo, At: now}); err != nil {
			return nil, err
		}
		if post := s.postOfPath(ctx, ws, link.ReturnTo); post != uuid.Nil {
			cid := link.ContactID
			if err := s.Q.WebsiteConversionInsert(ctx, store.WebsiteConversionInsertParams{ID: newID(), WorkspaceID: ws.ID, PostID: post, Kind: "signup",
				ContactID: &cid, At: now}); err != nil {
				return nil, err
			}
		}
	case PurposeEmail:
		tag, err := s.Pool.Exec(ctx, `UPDATE contacts SET email = $3, updated_at = $4 WHERE id = $1 AND workspace_id = $2
			AND NOT EXISTS (SELECT 1 FROM contacts o WHERE o.workspace_id = $2 AND o.email = $3 AND o.id <> $1)`, link.ContactID, ws.ID, link.NewEmail, now)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			done.Taken = true
		} else if err := s.Q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: link.ContactID,
			Kind: "email_changed", Detail: link.NewEmail, At: now}); err != nil {
			return nil, err
		}
	}
	token, expires := Token(), now.AddDate(0, 0, int(c.row.SessionDays))
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsiteSessionInsert(ctx, store.WebsiteSessionInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: link.ContactID, TokenHash: hash(token),
			Method: method, At: now, ExpiresAt: expires}); err != nil {
			return err
		}
		if err := q.WebsiteMemberSignedIn(ctx, store.WebsiteMemberSignedInParams{ContactID: link.ContactID, WorkspaceID: ws.ID, At: now, Method: method}); err != nil {
			return err
		}
		return q.WebsiteMemberEventInsert(ctx, store.WebsiteMemberEventInsertParams{ID: newID(), WorkspaceID: ws.ID, ContactID: link.ContactID,
			Kind: "signed_in", Method: method, SameBrowser: &same, At: now})
	})
	if err != nil {
		return nil, err
	}
	done.Token, done.ExpiresAt, done.Email = token, expires, s.contactEmail(ctx, link.ContactID)
	return done, nil
}

// postOfPath is the published post at a path of the site ("/posts/x"), or nil.
func (s *Service) postOfPath(ctx context.Context, ws *store.Workspace, path string) uuid.UUID {
	i := strings.LastIndex(path, "/posts/")
	if i < 0 {
		return uuid.Nil
	}
	slug := strings.SplitN(path[i+len("/posts/"):], "?", 2)[0]
	slug = strings.SplitN(slug, "#", 2)[0]
	p, err := s.Q.WebsitePostBySlug(ctx, store.WebsitePostBySlugParams{WorkspaceID: ws.ID, Slug: slug})
	if err != nil {
		return uuid.Nil
	}
	return p.ID
}

// SignOut ends the session of a cookie.
func (s *Service) SignOut(ctx context.Context, ws *store.Workspace, sess *Session) error {
	if sess == nil {
		return nil
	}
	now := s.now()
	return s.Q.WebsiteSessionEnd(ctx, store.WebsiteSessionEndParams{At: &now, Reason: "signed_out", ID: sess.ID})
}

// JoinInput signs a reader up from the site: the paywall of a post, or
// the sign-in page.
type JoinInput struct {
	Email string
	// Lists are the newsletters ticked, among the offered ones.
	Lists     []uuid.UUID
	ReturnTo  string
	Browser   string
	IP        *netip.Addr
	UserAgent string
	Turnstile string
	Verify    func(answer string) bool
	// Page is the address the reader signed up on (consent evidence).
	Page string
	// Language is the page's language.
	Language string
}

// Join signs a reader up: one email, "Confirm your email and continue
// reading", confirms the newsletters they ticked and signs them in. The
// page, the time, the IP address and the exact consent text are kept.
func (s *Service) Join(ctx context.Context, ws *store.Workspace, in JoinInput) (SigninOutcome, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return SigninOutcome{}, err
	}
	out := SigninOutcome{Minutes: SignupLinkDays * 24 * 60}
	if !c.On() || !c.row.SigninOn || !c.row.SignupOn {
		out.Status = SigninOff
		return out, nil
	}
	email, problem := emailaddr.Check(in.Email)
	if problem != "" {
		out.Status, out.Message = SigninInvalid, tr(ctx, "signin.emailInvalid", nil)
		return out, nil
	}
	out.Email = email
	if st := s.ipCheck(in.IP, in.Turnstile, in.Verify); st != "" {
		out.Status = st
		return out, nil
	}
	out.Status = SigninSent
	offers, err := s.offers(ctx, ws, c, pagelabels.Lang(in.Language))
	if err != nil {
		return out, err
	}
	var lists []uuid.UUID
	var consent, names []string
	for _, o := range offers {
		if slices.Contains(in.Lists, o.ID) {
			lists = append(lists, o.ID)
			consent = append(consent, o.Consent)
			names = append(names, o.Name)
		}
	}
	ev := audience.PublicEvidence{Source: "website", IP: in.IP, UserAgent: in.UserAgent, ConsentText: strings.Join(consent, " "), Language: in.Language,
		Note: "signed up on " + in.Page, At: s.now()}
	var contact uuid.UUID
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if len(lists) > 0 {
			res, err := s.Audience.Signup(ctx, tx, ws, audience.SignupInput{Email: email, Lists: lists, DoubleOptIn: true, Evidence: ev, BrowserLanguage: in.Language,
				FormName: "Website"})
			if err != nil {
				return err
			}
			if res.Blocked {
				return nil
			}
			contact = res.ContactID
			return nil
		}
		// Membership alone: a contact on no list.
		q := store.New(tx)
		cur, err := q.SignupContactByEmail(ctx, store.SignupContactByEmailParams{WorkspaceID: ws.ID, Email: email})
		if errors.Is(err, pgx.ErrNoRows) {
			cur, err = q.SignupInsertContact(ctx, store.SignupInsertContactParams{ID: newID(), WorkspaceID: ws.ID, Email: email,
				MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), Consent: "pending", Source: "website", Attributes: []byte(`{}`),
				RoleAddress: emailaddr.IsRole(email), At: ev.At})
			if errors.Is(err, pgx.ErrNoRows) {
				cur, err = q.SignupContactByEmail(ctx, store.SignupContactByEmailParams{WorkspaceID: ws.ID, Email: email})
			}
		}
		if err != nil {
			return err
		}
		contact = cur.ID
		return nil
	})
	if err != nil {
		var fe *audience.FieldError
		if errors.As(err, &fe) {
			out.Status, out.Message = SigninInvalid, fe.Message
			return out, nil
		}
		return out, err
	}
	if contact == uuid.Nil {
		return out, nil
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, contact)
	if err != nil {
		if errors.Is(err, audience.ErrContactGone) {
			return out, nil
		}
		return out, err
	}
	if blocked, err := s.blocked(ctx, ws, pc); err != nil || blocked {
		return out, err
	}
	if wait, err := s.addressLimit(ctx, contact); err != nil || wait > 0 {
		if wait > 0 {
			out.Status, out.RetryIn = SigninLimited, wait
		}
		return out, err
	}
	title := ""
	if post := s.postOfPath(ctx, ws, in.ReturnTo); post != uuid.Nil {
		_ = s.Pool.QueryRow(ctx, `SELECT title FROM website_posts WHERE id = $1`, post).Scan(&title)
	}
	err = s.sendLink(ctx, ws, c, pc, linkRequest{purpose: PurposeSignup, browser: in.Browser, returnTo: SafeReturn(in.ReturnTo), lists: lists,
		expires: SignupLinkDays * 24 * time.Hour, postTitle: title, listNames: names})
	return out, err
}
