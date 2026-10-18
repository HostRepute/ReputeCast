package referrals

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// ShareLink is a plain share link of a reader's referral link: no
// third-party scripts, no trackers.
type ShareLink struct {
	Channel string
	URL     string
}

// ShareLinks are a reader's share links, each adding its ?via=.
func ShareLinks(link, message, subject string, buttons []string) []ShareLink {
	with := func(via string) string {
		if via == "" {
			return link
		}
		sep := "?"
		if strings.Contains(link, "?") {
			sep = "&"
		}
		return link + sep + "via=" + via
	}
	text := func(via string) string {
		m := strings.TrimSpace(message)
		if m == "" {
			return with(via)
		}
		if strings.Contains(m, "{link}") {
			return strings.ReplaceAll(m, "{link}", with(via))
		}
		return m + " " + with(via)
	}
	var out []ShareLink
	for _, b := range buttons {
		switch b {
		case ViaEmail:
			out = append(out, ShareLink{Channel: b, URL: "mailto:?subject=" + url.PathEscape(subject) + "&body=" + url.PathEscape(text(ViaEmail))})
		case ViaX:
			out = append(out, ShareLink{Channel: b, URL: "https://x.com/intent/post?text=" + url.QueryEscape(text(ViaX))})
		case ViaLinkedIn:
			out = append(out, ShareLink{Channel: b, URL: "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(with(ViaLinkedIn))})
		case ViaWhatsApp:
			out = append(out, ShareLink{Channel: b, URL: "https://wa.me/?text=" + url.QueryEscape(text(ViaWhatsApp))})
		}
	}
	return out
}

// HubToken is a reader's hub link token: it opens their hub for 60 days.
func (s *Service) HubToken(wsID, contactID uuid.UUID, at time.Time) string {
	if s.Links == nil {
		return ""
	}
	return s.Links.Sign(linktoken.Claims{Kind: linktoken.ReferralHub, WorkspaceID: wsID, ContactID: contactID, Link: linktoken.IssueDay(at)})
}

// HubReward is a reward on the hub.
type HubReward struct {
	ID        uuid.UUID
	Name      string
	Count     int
	Type      string
	Status    string
	Reached   bool
	SentAt    *time.Time
	UsedAt    *time.Time
	ExpiresAt *time.Time
	FileName  string
	FileType  string
	FileSize  int64
	// DownloadURL is a fresh expiring link (or the reward's own URL).
	DownloadURL string
	ClaimURL    string
	Code        string
	AnswerURL   string
	Note        string
	AskAddress  bool
	Address     *ReferralShippingAddress
	// ToGo is how many more counted referrals reach it (not reached yet).
	ToGo int64
}

// HubReferral is one referral on the hub: numbers and dates only.
type HubReferral struct {
	Number      int
	Status      string
	Reason      string
	SignedUpAt  time.Time
	ConfirmedAt *time.Time
	CountsOn    *time.Time
	CountedAt   *time.Time
}

// HubView is a reader's referral hub.
type HubView struct {
	Workspace    *store.Workspace
	ProgramID    uuid.UUID
	LinkID       uuid.UUID
	ContactID    uuid.UUID
	ProgramOn    bool
	LinkActive   bool
	Email        string
	FirstName    string
	ShortName    string
	ListName     string
	Link         string // "news.acme.io/join/K7Q2MX"
	LinkURL      string
	Code         string
	Share        []ShareLink
	ShareButtons []string
	Visits       int64
	Signups      int64
	Intro        string
	HoldDays     int
	SameIP       bool
	Disposable   bool
	Track        ReferralTrack
	Rewards      []HubReward
	Referrals    []HubReferral
	ShowName     bool
	ShowBoard    bool
	BoardOn      bool
	Rank         int
	BoardURL     string
	Token        string
}

// ErrNoHub: the contact has no link in a program (not a referrer).
var ErrNoHub = errors.New("referrals: no hub for this contact")

// Hub is a reader's hub, opened from their signed link; ip is where they
// opened it (the same-IP rule).
func (s *Service) Hub(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, token string, ip *netip.Addr) (*HubView, error) {
	links, err := s.Q.ReferralLinksOfContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, ErrNoHub
	}
	l := links[0]
	for _, x := range links {
		if x.ProgramStatus == "on" {
			l = x
			break
		}
	}
	p, err := s.Q.ReferralProgramByID(ctx, l.ProgramID)
	if err != nil {
		return nil, err
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return nil, err
	}
	var email, first, last, listName string
	if err := s.Pool.QueryRow(ctx, `SELECT c.email, c.first_name, c.last_name, li.name FROM contacts c, lists li WHERE c.id = $1 AND li.id = $2`,
		contactID, p.ListID).Scan(&email, &first, &last, &listName); err != nil {
		return nil, err
	}
	s.SeenReferrer(ctx, l.ID, ip)
	full := pages.Join(l.Code)
	v := &HubView{Workspace: ws, ProgramID: p.ID, LinkID: l.ID, ContactID: contactID, ProgramOn: p.Status == "on", LinkActive: l.Status == "active",
		Email: email, FirstName: first, ShortName: shortName(first, last), ListName: listName, LinkURL: full, Code: l.Code,
		Link: strings.TrimPrefix(strings.TrimPrefix(full, "https://"), "http://"), ShareButtons: orderedButtons(p.ShareButtons),
		Visits: l.Visits, Signups: l.Signups, Intro: p.HubIntro, HoldDays: int(p.HoldDays), SameIP: p.RuleSameIp, Disposable: p.RuleDisposable,
		ShowName: l.ShowName, ShowBoard: l.ShowLeaderboard, BoardOn: p.LeaderboardOn, Token: token}
	v.Share = ShareLinks(full, p.ShareMessage, listName, v.ShareButtons)
	items, err := s.referralItems(ctx, l.ID, false)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		v.Referrals = append(v.Referrals, HubReferral{Number: it.Number, Status: it.Status, Reason: it.Reason, SignedUpAt: it.SignedUpAt,
			ConfirmedAt: it.ConfirmedAt, CountsOn: it.CountsOn, CountedAt: it.CountedAt})
	}
	if v.Track, err = s.track(ctx, p, l.ID, items); err != nil {
		return nil, err
	}
	if v.Rewards, err = s.hubRewards(ctx, ws, p, contactID, v.Track.Counted); err != nil {
		return nil, err
	}
	if p.LeaderboardOn {
		v.BoardURL = pages.Referrals(token) + "/leaderboard"
		var rank int32
		if err := s.Pool.QueryRow(ctx, `SELECT rank FROM referral_leaderboard WHERE program_id = $1 AND period = 'all' AND link_id = $2`, p.ID, l.ID).Scan(&rank); err == nil {
			v.Rank = int(rank)
		}
	}
	return v, nil
}

func (s *Service) hubRewards(ctx context.Context, ws *store.Workspace, p store.ReferralProgram, contactID uuid.UUID, counted int64) ([]HubReward, error) {
	rows, err := s.Q.ReferralRewardsOfContact(ctx, store.ReferralRewardsOfContactParams{ProgramID: p.ID, ContactID: contactID})
	if err != nil {
		return nil, err
	}
	pages, _ := s.pages(ctx, ws)
	now := s.now()
	have := map[int32]bool{}
	var out []HubReward
	for _, row := range rows {
		r := row.ReferralReward
		have[r.MilestoneCount] = true
		h := HubReward{ID: r.ID, Name: r.RewardName, Count: int(r.MilestoneCount), Type: r.RewardType, Status: r.Status, Reached: true, SentAt: r.SentAt,
			UsedAt: r.UsedAt, ExpiresAt: r.ExpiresAt, FileName: row.FileName, FileType: row.FileType, FileSize: row.FileSize, ClaimURL: r.ClaimUrl,
			Note: r.FulfillNote, AskAddress: row.AskAddress}
		if r.Status == RewardSent || r.Status == RewardFulfilled {
			switch r.RewardType {
			case RewardDownload:
				if row.FileID != nil {
					h.DownloadURL = pages.Download(s.DownloadToken(ws.ID, r.ID, contactID, now))
				} else {
					h.DownloadURL = row.DownloadUrl
				}
			case RewardCodes, RewardWebhook, RewardCoupon:
				h.Code, h.AnswerURL = s.rewardCode(ctx, r)
			}
		}
		if r.RewardType == RewardManual {
			h.Address = s.openAddress(r.ID, r.Address)
		}
		out = append(out, h)
	}
	ms, err := s.Q.ReferralMilestones(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		if have[m.Count] {
			continue
		}
		h := HubReward{Name: m.RewardName, Count: int(m.Count), Type: m.RewardType, ToGo: int64(m.Count) - counted}
		if m.FileID != nil {
			if f, err := s.Q.ReferralFileGet(ctx, *m.FileID); err == nil {
				h.FileName, h.FileType, h.FileSize = f.FileName, f.ContentType, f.SizeBytes
			}
		}
		out = append(out, h)
	}
	return out, nil
}

// SetChoices saves a reader's two switches on their hub: their first name
// on their invite page, their name on the leaderboard.
func (s *Service) SetChoices(ctx context.Context, linkID uuid.UUID, showName, showBoard bool) error {
	return s.Q.ReferralLinkSetChoices(ctx, store.ReferralLinkSetChoicesParams{ID: linkID, ShowName: showName, ShowLeaderboard: showBoard, Now: s.now()})
}

// SaveAddress keeps where a manual reward goes (sealed; deleted 30 days
// after the reward is marked fulfilled).
func (s *Service) SaveAddress(ctx context.Context, contactID, rewardID uuid.UUID, a ReferralShippingAddress) error {
	rw, err := s.Q.ReferralRewardByID(ctx, rewardID)
	if err != nil || rw.ContactID != contactID || rw.RewardType != RewardManual {
		return pgx.ErrNoRows
	}
	a.Name, a.Street, a.City, a.Country = strings.TrimSpace(a.Name), strings.TrimSpace(a.Street), strings.TrimSpace(a.City), strings.TrimSpace(a.Country)
	raw, _ := json.Marshal(a)
	sealed, err := s.Box.Seal(raw, addressAD(rewardID))
	if err != nil {
		return err
	}
	return s.Q.ReferralRewardAddress(ctx, store.ReferralRewardAddressParams{ID: rewardID, Address: sealed, Now: s.now()})
}

// HubDownload is a reward file to stream, or an address to send the reader
// to.
type HubDownload struct {
	FileID      uuid.UUID
	FileName    string
	ContentType string
	Size        int64
	URL         string
}

// Download resolves a reward's download link and counts it.
func (s *Service) Download(ctx context.Context, c linktoken.Claims) (*HubDownload, error) {
	if ExpiredLink(c, DownloadValid, s.now()) {
		return nil, ErrLinkExpired
	}
	rw, err := s.Q.ReferralRewardByID(ctx, c.MessageID)
	if err != nil || rw.ContactID != c.ContactID || rw.WorkspaceID != c.WorkspaceID || rw.MilestoneID == nil {
		return nil, pgx.ErrNoRows
	}
	m, err := s.Q.ReferralMilestoneByID(ctx, *rw.MilestoneID)
	if err != nil {
		return nil, err
	}
	out := &HubDownload{URL: m.DownloadUrl}
	if m.FileID != nil {
		f, err := s.Q.ReferralFileGet(ctx, *m.FileID)
		if err != nil {
			return nil, err
		}
		out = &HubDownload{FileID: f.ID, FileName: f.FileName, ContentType: f.ContentType, Size: f.SizeBytes}
		_ = s.Q.ReferralFileCounted(ctx, f.ID)
	}
	if err := s.Q.ReferralRewardUsed(ctx, store.ReferralRewardUsedParams{ID: rw.ID, Downloads: 1, Now: s.now()}); err != nil {
		return nil, err
	}
	return out, nil
}

// FileChunks streams a reward file's bytes in order.
func (s *Service) FileChunks(ctx context.Context, fileID uuid.UUID, write func([]byte) error) error {
	seq := int32(0)
	for {
		rows, err := s.Q.ReferralFileChunks(ctx, store.ReferralFileChunksParams{FileID: fileID, FromSeq: seq, Lim: 8})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, r := range rows {
			if err := write(r.Data); err != nil {
				return err
			}
			seq = r.Seq + 1
		}
	}
}

// ErrLinkExpired: a personal link is past its date.
var ErrLinkExpired = errors.New("referrals: the link has expired")

// HubExpired reports whether a hub token is past its 60 days.
func (s *Service) HubExpired(c linktoken.Claims) bool { return ExpiredLink(c, HubValid, s.now()) }

// SendHubLink emails a reader a fresh link to their hub ("Email me my
// link"), from the System stream.
func (s *Service) SendHubLink(ctx context.Context, ws *store.Workspace, set *forms.Settings, contactID uuid.UUID) error {
	if s.Mail == nil || s.Mail() == nil {
		return errors.New("referrals: system mail isn't available")
	}
	m := s.Mail()
	if !m.SystemAvailable(ctx, &ws.ID) {
		return errors.New("referrals: system mail isn't available")
	}
	var email, first string
	if err := s.Pool.QueryRow(ctx, `SELECT email, first_name FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, contactID, ws.ID).Scan(&email, &first); err != nil {
		return err
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return err
	}
	link := pages.Referrals(s.HubToken(ws.ID, contactID, s.now()))
	b := set.Branding
	greeting := "Hi there,"
	if strings.TrimSpace(first) != "" {
		greeting = "Hi " + strings.TrimSpace(first) + ","
	}
	subject := "Your referrals at " + b.PublisherName
	text := greeting + "\n\nHere's a fresh link to your referrals page: your link, your progress and your rewards.\n\n" + link +
		"\n\nIf you didn't ask for it, you can ignore this email.\n\n" + b.PublisherName + "\n"
	esc := html.EscapeString
	body := `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>` + esc(subject) + `</title></head>` +
		`<body style="margin:0;padding:24px 16px;background:` + esc(b.Palette.Background) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">` +
		`<div style="max-width:520px;margin:0 auto;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px;padding:28px">` +
		`<p style="margin:0 0 16px;font-size:18px;font-weight:700;color:#0B1220">` + esc(b.PublisherName) + `</p>` +
		`<p style="margin:0 0 12px;font-size:15px;line-height:1.5">` + esc(greeting) + `</p>` +
		`<p style="margin:0 0 20px;font-size:15px;line-height:1.5">Here's a fresh link to your referrals page: your link, your progress and your rewards.</p>` +
		`<p style="margin:0 0 20px"><a href="` + esc(link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(b.Palette.Accent) +
		`;color:` + esc(b.Palette.OnAccent) + `;font-weight:700;font-size:15px;text-decoration:none">Open your referrals</a></p>` +
		`<p style="margin:0;font-size:13px;line-height:1.5;color:#5B6472">If you didn't ask for it, you can ignore this email.</p></div></body></html>`
	_, err = m.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: email, ToName: first, Subject: subject, Text: text, HTML: body, Kind: "referral_hub_link"})
	return err
}

// MaskedEmail is a contact's address as the expired hub shows it:
// "s•••@outlook.com".
func (s *Service) MaskedEmail(ctx context.Context, contactID uuid.UUID) string {
	var email string
	if err := s.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE id = $1`, contactID).Scan(&email); err != nil {
		return ""
	}
	return emailaddr.Mask(email)
}

// InviteView is the invite page of a referral link: the hosted subscribe
// page in invite mode, limited to the program's list.
type InviteView struct {
	ProgramID uuid.UUID
	LinkID    uuid.UUID
	ListID    uuid.UUID
	// Name is the referrer's first name when they allow it, else "".
	Name     string
	Headline string
	Text     string
}

// Invite is a working link's invite page, or nil: an unknown or paused
// code, or a program that's off, shows the normal subscribe page.
func (s *Service) Invite(ctx context.Context, ws *store.Workspace, code string) (*InviteView, error) {
	c, ok := NormalizeCode(code)
	if !ok {
		return nil, nil
	}
	al, err := s.activeLink(ctx, s.Q, ws, c)
	if err != nil || al == nil {
		return nil, err
	}
	v := &InviteView{ProgramID: al.program.ID, LinkID: al.link.ID, ListID: al.program.ListID, Text: al.program.InviteText}
	if al.link.ShowName {
		v.Name = strings.TrimSpace(al.link.ContactFirstName)
	}
	v.Headline = al.program.InviteHeadline
	return v, nil
}

// PreferenceCard is the preference center's "Invite friends" card: for a
// contact on a program's list while it's on.
type PreferenceCard struct {
	Counted int64
	HubURL  string
	// ListName is the program's list; NextCount and NextReward the next
	// milestone (0 when every reward is earned).
	ListName   string
	NextCount  int
	NextReward string
}

// PreferenceCardFor returns a contact's card, or nil.
func (s *Service) PreferenceCardFor(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) (*PreferenceCard, error) {
	links, err := s.Q.ReferralLinksOfContact(ctx, contactID)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.ProgramStatus != "on" {
			continue
		}
		on, err := s.Q.ReferralContactOnList(ctx, store.ReferralContactOnListParams{ListID: l.ListID, ContactID: contactID})
		if err != nil || !on {
			continue
		}
		pages, err := s.pages(ctx, ws)
		if err != nil {
			return nil, err
		}
		id := l.ID
		n, err := s.Q.ReferralCountedOf(ctx, store.ReferralCountedOfParams{ProgramID: l.ProgramID, LinkID: &id})
		if err != nil {
			return nil, err
		}
		card := &PreferenceCard{Counted: n, HubURL: pages.Referrals(s.HubToken(ws.ID, contactID, s.now()))}
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1`, l.ListID).Scan(&card.ListName)
		ms, err := s.Q.ReferralMilestones(ctx, l.ProgramID)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			if int64(m.Count) > n {
				card.NextCount, card.NextReward = int(m.Count), m.RewardName
				break
			}
		}
		return card, nil
	}
	return nil, nil
}
