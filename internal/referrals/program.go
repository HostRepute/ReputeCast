package referrals

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReferralListOption is a list a program can use.
type ReferralListOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Subscribers int64  `json:"subscribers"`
	// Taken: the list already has a program.
	Taken bool `json:"taken"`
}

// ReferralProgramSummary is one program of the workspace.
type ReferralProgramSummary struct {
	ID          string     `json:"id"`
	ListID      string     `json:"listId"`
	ListName    string     `json:"listName"`
	Status      string     `json:"status" enum:"off,on"`
	SetupStep   int        `json:"setupStep" doc:"The set-up guide's step (1 to 5) until the program is first turned on; 0 afterwards"`
	Subscribers int64      `json:"subscribers"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
}

// ReferralProgramList is the workspace's programs and the lists a new one
// can use.
type ReferralProgramList struct {
	Items   []ReferralProgramSummary `json:"items"`
	Lists   []ReferralListOption     `json:"lists"`
	Helpers GrowHelperSets           `json:"helpers"`
}

// ReferralRule is a suspicion rule with its threshold.
type ReferralRule struct {
	On        bool `json:"on"`
	Threshold int  `json:"threshold" minimum:"1" maximum:"1000"`
}

// ReferralCounting is the counting rules (brief §3.2). Self-referral,
// "already on the list" and "never confirmed" always apply.
type ReferralCounting struct {
	HoldDays   int          `json:"holdDays" enum:"0,3,7,14,30"`
	SameIP     bool         `json:"sameIp"`
	Disposable bool         `json:"disposable"`
	Left       bool         `json:"left"`
	Burst      ReferralRule `json:"burst"`
	SharedIP   ReferralRule `json:"sharedIp"`
	Numbered   bool         `json:"numbered"`
}

// ReferralPageSettings is what readers see when they share and when friends
// arrive.
type ReferralPageSettings struct {
	// InviteHeadline holds {first_name}: the referrer's name when they allow
	// it, otherwise "A reader".
	InviteHeadline string `json:"inviteHeadline" maxLength:"200"`
	InviteText     string `json:"inviteText" maxLength:"600"`
	HubIntro       string `json:"hubIntro" maxLength:"600"`
	// ShareMessage holds {link}, the reader's own link.
	ShareMessage string   `json:"shareMessage" maxLength:"500"`
	ShareButtons []string `json:"shareButtons" maxItems:"5" enum:"copy,email,x,linkedin,whatsapp"`
}

// ReferralLeaderboardSettings is the public leaderboard (off by default).
type ReferralLeaderboardSettings struct {
	On      bool       `json:"on"`
	At      *time.Time `json:"at,omitempty"`
	By      string     `json:"by"`
	Periods []string   `json:"periods" maxItems:"2" enum:"month,all"`
	Rows    int        `json:"rows" enum:"5,10,25"`
	URL     string     `json:"url"`
}

// ReferralSenderOption is a sender milestone emails can come from.
type ReferralSenderOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	// Default: the list's default sender (the Automations stream's).
	Default bool `json:"default"`
}

// ReferralEmailSettings is where milestone emails come from. The stream is
// always Automations.
type ReferralEmailSettings struct {
	SenderID string `json:"senderId"`
	ReplyTo  string `json:"replyTo"`
	// On is how many milestones send an email, of Milestones.
	On         int `json:"on"`
	Milestones int `json:"milestones"`
}

// ReferralProgramDetail is a program with its settings and the facts the
// pages' headers show.
type ReferralProgramDetail struct {
	ID          string `json:"id"`
	ListID      string `json:"listId"`
	ListName    string `json:"listName"`
	Status      string `json:"status" enum:"off,on"`
	Subscribers int64  `json:"subscribers"`
	Links       int64  `json:"links"`
	SetupStep   int    `json:"setupStep"`
	// SetupSavedAt and SetupSavedBy say when the guide was left ("Welcome
	// back, Sofia").
	SetupSavedAt *time.Time `json:"setupSavedAt,omitempty"`
	SetupSavedBy string     `json:"setupSavedBy"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	StartedBy    string     `json:"startedBy"`
	// StatusAt and StatusBy are the last change ("off since 15 Oct, 09:12").
	StatusAt *time.Time `json:"statusAt,omitempty"`
	StatusBy string     `json:"statusBy"`
	// AnnouncedIn is the first campaign sent with the Referral block.
	AnnouncedIn string                      `json:"announcedIn"`
	Counting    ReferralCounting            `json:"counting"`
	Pages       ReferralPageSettings        `json:"pages"`
	Leaderboard ReferralLeaderboardSettings `json:"leaderboard"`
	Emails      ReferralEmailSettings       `json:"emails"`
	Senders     []ReferralSenderOption      `json:"senders"`
	// LinkHost and LinkExample show the link format ("news.acme.io/join/K7Q2MX").
	LinkHost    string `json:"linkHost"`
	LinkExample string `json:"linkExample"`
	// Held is the review queue (the Review tab's badge) and its oldest.
	Held       int64          `json:"held"`
	HeldOldest *time.Time     `json:"heldOldest,omitempty"`
	InHold     int64          `json:"inHold"`
	Awaiting   int64          `json:"awaiting"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	Helpers    GrowHelperSets `json:"helpers"`
}

// ReferralCreateInput starts a program on a list (the guide's first step).
type ReferralCreateInput struct {
	ListID string `json:"listId" format:"uuid"`
}

// ReferralSettingsInput saves Referrals › Settings.
type ReferralSettingsInput struct {
	Counting ReferralCounting     `json:"counting"`
	Pages    ReferralPageSettings `json:"pages"`
	// Leaderboard's periods and rows (its switch is its own action).
	LeaderboardPeriods []string `json:"leaderboardPeriods" maxItems:"2" enum:"month,all"`
	LeaderboardRows    int      `json:"leaderboardRows" enum:"5,10,25"`
	SenderID           string   `json:"senderId,omitempty" format:"uuid"`
	ReplyTo            string   `json:"replyTo" maxLength:"254"`
	// BaseUpdatedAt refuses the save when someone saved since.
	BaseUpdatedAt *time.Time `json:"baseUpdatedAt,omitempty"`
}

// ReferralSettingsImpact is what a change of the counting rules does to
// referrals not decided yet ("18 referrals in hold would count up to 7 days
// later").
type ReferralSettingsImpact struct {
	HoldFrom int   `json:"holdFrom"`
	HoldTo   int   `json:"holdTo"`
	InHold   int64 `json:"inHold"`
	// Awaiting referrals are checked against the exclusion and suspicion
	// rules when they confirm.
	Awaiting     int64    `json:"awaiting"`
	RulesChanged []string `json:"rulesChanged"`
}

// ReferralSetupInput saves the guide's step ("Save and exit", Continue).
type ReferralSetupInput struct {
	Step   int    `json:"step" minimum:"1" maximum:"5"`
	ListID string `json:"listId,omitempty" format:"uuid"`
}

// ReferralCheck is one check before a program turns on.
type ReferralCheck struct {
	Key    string `json:"key" enum:"milestones,emails,files,stripe,sender"`
	Status string `json:"status" enum:"pass,fail"`
	// Facts the sentence is written from (the app writes it).
	Count   int                 `json:"count"`
	Counts  []int               `json:"counts"`
	Files   []ReferralCheckFile `json:"files"`
	Rewards []string            `json:"rewards"`
	Account string              `json:"account"`
	Offer   string              `json:"offer"`
	Sender  string              `json:"sender"`
	Emails  int                 `json:"emails"`
	Missing []string            `json:"missing"`
}

// ReferralCheckFile is a download reward's file.
type ReferralCheckFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// ReferralChecks are the turn-on checks.
type ReferralChecks struct {
	Checks []ReferralCheck `json:"checks"`
	Pass   bool            `json:"pass"`
	// Links is how many links turning on creates (confirmed subscribers).
	Links int64 `json:"links"`
	// NextIssue is the list's next draft, to add the Referral block to
	// (the guide's step 4).
	NextIssue *ReferralCampaignRef `json:"nextIssue,omitempty"`
}

// ReferralTurnedOn is what turning a program on did.
type ReferralTurnedOn struct {
	Program      ReferralProgramDetail `json:"program"`
	LinksCreated int64                 `json:"linksCreated"`
	// NextIssue is the list's next draft, to add the Referral block to.
	NextIssue *ReferralCampaignRef `json:"nextIssue,omitempty"`
}

// ReferralCampaignRef names a campaign.
type ReferralCampaignRef struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
}

var (
	mListTaken     = msgs.M("program.listTaken")
	mListUnknown   = msgs.M("program.listUnknown")
	mListFixed     = msgs.M("program.listFixed")
	mChecksFail    = msgs.M("program.checksFail")
	mAlreadyOn     = msgs.M("program.alreadyOn")
	mAlreadyOff    = msgs.M("program.alreadyOff")
	mDeleteOn      = msgs.M("program.deleteOn")
	mSenderUnknown = msgs.M("program.senderUnknown")
	mReplyTo       = msgs.M("program.replyTo")
	mStale         = msgs.M("program.stale")
	mHeadlineName  = msgs.M("program.headlineName")
	mShareLink     = msgs.M("program.shareLink")
	mPeriods       = msgs.M("program.periods")
	mShareButtons  = msgs.M("program.shareButtons")
)

func (s *Service) text(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// Programs lists the workspace's programs and the lists they can use.
func (s *Service) Programs(ctx context.Context, ws *store.Workspace) (ReferralProgramList, error) {
	out := ReferralProgramList{Items: []ReferralProgramSummary{}, Lists: []ReferralListOption{}, Helpers: s.helperSets(ctx, ws.ID)}
	lists, err := s.Q.ReferralListsOffered(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	subscribers := make(map[uuid.UUID]int64, len(lists))
	for _, l := range lists {
		subscribers[l.ID] = l.Subscribers
		out.Lists = append(out.Lists, ReferralListOption{ID: l.ID.String(), Name: l.Name, Description: l.Description, Subscribers: l.Subscribers, Taken: l.Taken})
	}
	rows, err := s.Q.ReferralProgramsOf(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		n, ok := subscribers[r.ListID]
		if !ok {
			// A deleted list isn't offered: it is counted on its own.
			if n, err = s.Q.ReferralListSubscribers(ctx, r.ListID); err != nil {
				return out, err
			}
		}
		out.Items = append(out.Items, ReferralProgramSummary{ID: r.ID.String(), ListID: r.ListID.String(), ListName: r.ListName,
			Status: r.Status, SetupStep: int(r.SetupStep), Subscribers: n, StartedAt: r.FirstOnAt})
	}
	return out, nil
}

// defaultCopy is the copy a new program starts with, written for its list.
func defaultCopy(publisher, list, description string) (headline, text, intro, share string) {
	name := list
	if publisher != "" && !strings.HasPrefix(strings.ToLower(list), strings.ToLower(publisher)) {
		name = publisher + " " + list
	}
	headline = "{first_name} invited you to the " + name
	text = description
	intro = "Thanks for sharing the " + list + ". Every friend who confirms brings you closer to the next reward."
	share = "I read the " + name + ". Here's my link if you'd like it too: {link}"
	return headline, text, intro, share
}

// Create starts a program on a list: off, at the guide's second step, with
// the default rules and copy.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, in ReferralCreateInput) (ReferralProgramDetail, error) {
	listID, err := uuid.Parse(in.ListID)
	if err != nil {
		return ReferralProgramDetail{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	list, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: listID})
	if err != nil {
		return ReferralProgramDetail{}, api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
	}
	if _, err := s.Q.ReferralProgramForList(ctx, listID); err == nil {
		return ReferralProgramDetail{}, api.Err(http.StatusConflict, "program_exists", s.text(ctx, mListTaken, i18n.Args{"list": list.Name}))
	}
	p := api.PrincipalFrom(ctx)
	var by *uuid.UUID
	if p != nil && p.Type == authz.ActorUser {
		id := p.ID
		by = &id
	}
	publisher := publisherName(ws)
	headline, text, intro, share := defaultCopy(publisher, list.Name, list.Description)
	sender := ws.DefaultAutomationSenderID
	if sender == nil {
		sender = ws.DefaultSenderID
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramInsert(ctx, store.ReferralProgramInsertParams{ID: id, WorkspaceID: ws.ID, ListID: listID, Now: now, CreatedBy: by,
			InviteHeadline: headline, InviteText: text, HubIntro: intro, ShareMessage: share, SenderID: sender}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.program_created", TargetType: "referral_program",
			TargetID: id.String(), TargetName: list.Name, After: map[string]any{"list": list.Name, "status": "off"}})
	})
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// publisherName is the workspace's publisher name for default copy: its
// branding's, else the workspace's name without "Newsletter".
func publisherName(ws *store.Workspace) string {
	name := strings.TrimSpace(strings.TrimSuffix(ws.Name, " Newsletter"))
	return name
}

func (s *Service) program(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.ReferralProgramGetRow, error) {
	r, err := s.Q.ReferralProgramGet(ctx, store.ReferralProgramGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return r, api.MapError(err, "Referral program")
	}
	return r, nil
}

// Detail is a program with its settings.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralProgramDetail, error) {
	r, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	p := r.ReferralProgram
	out := ReferralProgramDetail{
		ID: p.ID.String(), ListID: p.ListID.String(), ListName: r.ListName, Status: p.Status, SetupStep: int(p.SetupStep),
		SetupSavedAt: p.SetupSavedAt, SetupSavedBy: r.SetupSavedName, StartedAt: p.FirstOnAt, StartedBy: r.FirstOnName,
		StatusAt: p.StatusAt, StatusBy: r.StatusByName, UpdatedAt: p.UpdatedAt,
		Counting: countingOf(p),
		Pages: ReferralPageSettings{InviteHeadline: p.InviteHeadline, InviteText: p.InviteText, HubIntro: p.HubIntro, ShareMessage: p.ShareMessage,
			ShareButtons: orderedButtons(p.ShareButtons)},
		Leaderboard: ReferralLeaderboardSettings{On: p.LeaderboardOn, At: p.LeaderboardAt, By: r.LeaderboardByName, Periods: nonNil(p.LeaderboardPeriods),
			Rows: int(p.LeaderboardRows)},
		Senders: []ReferralSenderOption{},
		Helpers: s.helperSets(ctx, ws.ID),
	}
	if out.Subscribers, err = s.Q.ReferralListSubscribers(ctx, p.ListID); err != nil {
		return out, err
	}
	if out.Links, err = s.Q.ReferralLinkCount(ctx, p.ID); err != nil {
		return out, err
	}
	if links, err := s.pages(ctx, ws); err == nil {
		out.LinkHost = hostOf(links.Origin())
		out.LinkExample = strings.TrimPrefix(strings.TrimPrefix(links.Join("K7Q2MX"), "https://"), "http://")
		out.Leaderboard.URL = links.Leaderboard()
	}
	counts, err := s.queueCounts(ctx, p.ID)
	if err != nil {
		return out, err
	}
	out.Held, out.HeldOldest, out.InHold, out.Awaiting = counts.held, counts.oldest, counts.hold, counts.awaiting
	out.AnnouncedIn = s.announcedIn(ctx, p)
	ms, err := s.Q.ReferralMilestones(ctx, p.ID)
	if err != nil {
		return out, err
	}
	out.Emails = ReferralEmailSettings{ReplyTo: p.ReplyTo, Milestones: len(ms)}
	for _, m := range ms {
		if m.EmailOn {
			out.Emails.On++
		}
	}
	if p.SenderID != nil {
		out.Emails.SenderID = p.SenderID.String()
	}
	senders, err := s.Q.SenderList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	def := ws.DefaultAutomationSenderID
	if def == nil {
		def = ws.DefaultSenderID
	}
	for _, sd := range senders {
		out.Senders = append(out.Senders, ReferralSenderOption{ID: sd.ID.String(), Name: sd.Name, Email: sd.Email, Verified: senderVerified(sd.Verification),
			Default: def != nil && *def == sd.ID})
	}
	return out, nil
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func hostOf(origin string) string {
	o := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
	if i := strings.IndexByte(o, '/'); i >= 0 {
		o = o[:i]
	}
	return o
}

// orderedButtons keeps the share buttons in the pages' order.
func orderedButtons(in []string) []string {
	out := []string{}
	for _, b := range ShareButtons {
		if slices.Contains(in, b) {
			out = append(out, b)
		}
	}
	return out
}

func countingOf(p store.ReferralProgram) ReferralCounting {
	return ReferralCounting{HoldDays: int(p.HoldDays), SameIP: p.RuleSameIp, Disposable: p.RuleDisposable, Left: p.RuleLeft,
		Burst: ReferralRule{On: p.BurstOn, Threshold: int(p.BurstThreshold)}, SharedIP: ReferralRule{On: p.SharedIpOn, Threshold: int(p.SharedIpThreshold)},
		Numbered: p.NumberedOn}
}

type queueCounts struct {
	held, hold, awaiting int64
	oldest               *time.Time
}

func (s *Service) queueCounts(ctx context.Context, programID uuid.UUID) (queueCounts, error) {
	var c queueCounts
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'review'), count(*) FILTER (WHERE status = 'hold'),
		count(*) FILTER (WHERE status = 'awaiting'), min(held_at) FILTER (WHERE status = 'review')
		FROM referrals WHERE program_id = $1`, programID).Scan(&c.held, &c.hold, &c.awaiting, &c.oldest)
	return c, err
}

// announcedIn names the first campaign sent with the Referral block after
// the program first turned on ("Weekly Digest #139").
func (s *Service) announcedIn(ctx context.Context, p store.ReferralProgram) string {
	if p.FirstOnAt == nil {
		return ""
	}
	var name string
	err := s.Pool.QueryRow(ctx, `SELECT name FROM campaigns
		WHERE workspace_id = $1 AND deleted_at IS NULL AND started_at >= $2 AND status IN ('sending', 'sent')
		  AND html LIKE '%rc:referral%'
		ORDER BY started_at LIMIT 1`, p.WorkspaceID, p.FirstOnAt.Add(-24*time.Hour)).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

func senderVerified(raw []byte) bool {
	return strings.Contains(string(raw), `"verified"`)
}

// UpdateSettings saves Referrals › Settings. Changes apply going forward:
// counted referrals and sent rewards never change.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralSettingsInput) (ReferralProgramDetail, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	p := cur.ReferralProgram
	if in.BaseUpdatedAt != nil && !in.BaseUpdatedAt.Equal(p.UpdatedAt) && in.BaseUpdatedAt.Before(p.UpdatedAt.Add(-time.Millisecond)) {
		return ReferralProgramDetail{}, api.Err(http.StatusConflict, "stale_settings", s.text(ctx, mStale, nil))
	}
	if err := s.checkSettings(ctx, ws, in); err != nil {
		return ReferralProgramDetail{}, err
	}
	var sender *uuid.UUID
	if in.SenderID != "" {
		sid, _ := uuid.Parse(in.SenderID)
		sender = &sid
	}
	before := settingsSnapshot(p)
	periods := in.LeaderboardPeriods
	if len(periods) == 0 {
		periods = []string{"month", "all"}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramUpdate(ctx, store.ReferralProgramUpdateParams{ID: id, HoldDays: int32(in.Counting.HoldDays),
			RuleSameIp: in.Counting.SameIP, RuleDisposable: in.Counting.Disposable, RuleLeft: in.Counting.Left,
			BurstOn: in.Counting.Burst.On, BurstThreshold: int32(in.Counting.Burst.Threshold),
			SharedIpOn: in.Counting.SharedIP.On, SharedIpThreshold: int32(in.Counting.SharedIP.Threshold), NumberedOn: in.Counting.Numbered,
			InviteHeadline: strings.TrimSpace(in.Pages.InviteHeadline), InviteText: strings.TrimSpace(in.Pages.InviteText),
			HubIntro: strings.TrimSpace(in.Pages.HubIntro), ShareMessage: strings.TrimSpace(in.Pages.ShareMessage),
			ShareButtons: orderedButtons(in.Pages.ShareButtons), LeaderboardPeriods: periods, LeaderboardRows: int32(in.LeaderboardRows),
			SenderID: sender, ReplyTo: strings.TrimSpace(in.ReplyTo), Now: s.now()}); err != nil {
			return err
		}
		// A new hold period applies to referrals still in hold, counting
		// from their confirmation.
		if int(p.HoldDays) != in.Counting.HoldDays {
			if _, err := tx.Exec(ctx, `UPDATE referrals SET hold_until = confirmed_at + make_interval(days => $2), updated_at = now()
				WHERE program_id = $1 AND status = 'hold' AND confirmed_at IS NOT NULL`, id, in.Counting.HoldDays); err != nil {
				return err
			}
		}
		after, err := q.ReferralProgramByID(ctx, id)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.program_updated", TargetType: "referral_program",
			TargetID: id.String(), TargetName: cur.ListName, Before: before, After: settingsSnapshot(after)})
	})
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

func (s *Service) checkSettings(ctx context.Context, ws *store.Workspace, in ReferralSettingsInput) error {
	if h := in.Pages.InviteHeadline; strings.TrimSpace(h) != "" && !strings.Contains(h, "{first_name}") {
		return api.Invalid("body.pages.inviteHeadline", s.text(ctx, mHeadlineName, nil))
	}
	if m := in.Pages.ShareMessage; strings.TrimSpace(m) != "" && !strings.Contains(m, "{link}") {
		return api.Invalid("body.pages.shareMessage", s.text(ctx, mShareLink, nil))
	}
	if len(in.Pages.ShareButtons) == 0 {
		return api.Invalid("body.pages.shareButtons", s.text(ctx, mShareButtons, nil))
	}
	if len(in.LeaderboardPeriods) == 0 {
		return api.Invalid("body.leaderboardPeriods", s.text(ctx, mPeriods, nil))
	}
	if in.SenderID != "" {
		sid, err := uuid.Parse(in.SenderID)
		if err != nil {
			return api.Invalid("body.senderId", s.text(ctx, mSenderUnknown, nil))
		}
		if _, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: sid}); err != nil {
			return api.Invalid("body.senderId", s.text(ctx, mSenderUnknown, nil))
		}
	}
	if r := strings.TrimSpace(in.ReplyTo); r != "" && (!strings.Contains(r, "@") || strings.ContainsAny(r, " <>")) {
		return api.Invalid("body.replyTo", s.text(ctx, mReplyTo, nil))
	}
	return nil
}

func settingsSnapshot(p store.ReferralProgram) map[string]any {
	return map[string]any{
		"hold_days": p.HoldDays, "same_ip": p.RuleSameIp, "disposable": p.RuleDisposable, "left_during_hold": p.RuleLeft,
		"burst":     map[string]any{"on": p.BurstOn, "threshold": p.BurstThreshold},
		"shared_ip": map[string]any{"on": p.SharedIpOn, "threshold": p.SharedIpThreshold}, "numbered": p.NumberedOn,
		"invite_headline": p.InviteHeadline, "invite_text": p.InviteText, "hub_intro": p.HubIntro, "share_message": p.ShareMessage,
		"share_buttons": p.ShareButtons, "leaderboard_periods": p.LeaderboardPeriods, "leaderboard_rows": p.LeaderboardRows,
		"sender": p.SenderID, "reply_to": p.ReplyTo,
	}
}

// Impact says what a change of the counting rules does to referrals not
// decided yet.
func (s *Service) Impact(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralCounting) (ReferralSettingsImpact, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralSettingsImpact{}, err
	}
	p := cur.ReferralProgram
	counts, err := s.queueCounts(ctx, id)
	if err != nil {
		return ReferralSettingsImpact{}, err
	}
	out := ReferralSettingsImpact{HoldFrom: int(p.HoldDays), HoldTo: in.HoldDays, RulesChanged: []string{}}
	if out.HoldFrom != out.HoldTo {
		out.InHold = counts.hold
	}
	old := countingOf(p)
	changed := func(name string, a, b bool) {
		if a != b {
			out.RulesChanged = append(out.RulesChanged, name)
		}
	}
	changed("sameIp", old.SameIP, in.SameIP)
	changed("disposable", old.Disposable, in.Disposable)
	changed("left", old.Left, in.Left)
	changed("burst", old.Burst != in.Burst, false)
	changed("sharedIp", old.SharedIP != in.SharedIP, false)
	changed("numbered", old.Numbered, in.Numbered)
	if len(out.RulesChanged) > 0 {
		out.Awaiting = counts.awaiting
	}
	return out, nil
}

// SetLeaderboard turns the public leaderboard on or off.
func (s *Service) SetLeaderboard(ctx context.Context, ws *store.Workspace, id uuid.UUID, on bool) (ReferralProgramDetail, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	if cur.ReferralProgram.LeaderboardOn == on {
		return s.Detail(ctx, ws, id)
	}
	by := actorUser(ctx)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramSetLeaderboard(ctx, store.ReferralProgramSetLeaderboardParams{ID: id, LeaderboardOn: on, Now: s.now(), By: by}); err != nil {
			return err
		}
		action := "referral.leaderboard_turned_off"
		if on {
			action = "referral.leaderboard_turned_on"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "referral_program", TargetID: id.String(),
			TargetName: cur.ListName, Before: map[string]any{"leaderboard": cur.ReferralProgram.LeaderboardOn}, After: map[string]any{"leaderboard": on}})
	})
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	if on {
		_ = s.RefreshLeaderboard(ctx, id)
	}
	return s.Detail(ctx, ws, id)
}

// actorUser is the signed-in person acting, when it is a person.
func actorUser(ctx context.Context) *uuid.UUID {
	p := api.PrincipalFrom(ctx)
	if p == nil || p.Type != authz.ActorUser {
		return nil
	}
	id := p.ID
	return &id
}

// SaveSetup keeps the guide's step (and the list, which can change until
// the program has been on).
func (s *Service) SaveSetup(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralSetupInput) (ReferralProgramDetail, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	if cur.ReferralProgram.FirstOnAt != nil {
		if in.ListID != "" && in.ListID != cur.ReferralProgram.ListID.String() {
			return ReferralProgramDetail{}, api.Err(http.StatusConflict, "list_fixed", s.text(ctx, mListFixed, nil))
		}
		return s.Detail(ctx, ws, id)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if in.ListID != "" {
			lid, err := uuid.Parse(in.ListID)
			if err != nil {
				return api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
			}
			if lid != cur.ReferralProgram.ListID {
				l, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid})
				if err != nil {
					return api.Invalid("body.listId", s.text(ctx, mListUnknown, nil))
				}
				if other, err := q.ReferralProgramForList(ctx, lid); err == nil && other.ID != id {
					return api.Err(http.StatusConflict, "program_exists", s.text(ctx, mListTaken, i18n.Args{"list": l.Name}))
				}
				if err := q.ReferralProgramSetList(ctx, store.ReferralProgramSetListParams{ID: id, ListID: lid, Now: s.now()}); err != nil {
					return err
				}
				if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.program_updated", TargetType: "referral_program",
					TargetID: id.String(), TargetName: l.Name, Before: map[string]any{"list": cur.ListName}, After: map[string]any{"list": l.Name}}); err != nil {
					return err
				}
			}
		}
		return q.ReferralProgramSetStep(ctx, store.ReferralProgramSetStepParams{ID: id, SetupStep: int32(in.Step), Now: s.now(), By: actorUser(ctx)})
	})
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// Checks are what has to be true before the program turns on.
func (s *Service) Checks(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralChecks, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralChecks{}, err
	}
	p := cur.ReferralProgram
	ms, err := s.Q.ReferralMilestones(ctx, id)
	if err != nil {
		return ReferralChecks{}, err
	}
	out := ReferralChecks{Checks: []ReferralCheck{}, NextIssue: s.nextIssue(ctx, ws, p.ListID)}
	newCheck := func(key string) ReferralCheck {
		return ReferralCheck{Key: key, Status: "pass", Counts: []int{}, Files: []ReferralCheckFile{}, Rewards: []string{}, Missing: []string{}}
	}
	c := newCheck("milestones")
	c.Count = len(ms)
	for _, m := range ms {
		c.Counts = append(c.Counts, int(m.Count))
	}
	if len(ms) == 0 {
		c.Status = "fail"
	}
	out.Checks = append(out.Checks, c)

	c = newCheck("emails")
	for _, m := range ms {
		if m.EmailOn {
			if m.EmailID == nil || strings.TrimSpace(m.EmailSubject) == "" {
				c.Missing = append(c.Missing, m.RewardName)
				continue
			}
			c.Emails++
		}
	}
	c.Count = len(ms)
	if len(ms) == 0 || len(c.Missing) > 0 {
		c.Status = "fail"
	}
	out.Checks = append(out.Checks, c)

	files := newCheck("files")
	hasDownload := false
	for _, m := range ms {
		if m.RewardType != RewardDownload {
			continue
		}
		hasDownload = true
		switch {
		case m.FileID != nil:
			f, err := s.Q.ReferralFileGet(ctx, *m.FileID)
			if err != nil || f.RemovedAt != nil {
				files.Missing = append(files.Missing, m.RewardName)
				continue
			}
			files.Files = append(files.Files, ReferralCheckFile{Name: f.FileName, Size: f.SizeBytes})
		case strings.TrimSpace(m.DownloadUrl) != "":
			files.Files = append(files.Files, ReferralCheckFile{Name: m.DownloadUrl})
		default:
			files.Missing = append(files.Missing, m.RewardName)
		}
	}
	if len(files.Missing) > 0 {
		files.Status = "fail"
	}
	if hasDownload {
		out.Checks = append(out.Checks, files)
	}

	var coupons []store.ReferralMilestone
	for _, m := range ms {
		if m.RewardType == RewardCoupon {
			coupons = append(coupons, m)
		}
	}
	if len(coupons) > 0 {
		st := newCheck("stripe")
		status := s.couponStatus(ctx, ws)
		st.Account = status.Account
		for _, m := range coupons {
			st.Rewards = append(st.Rewards, m.RewardName)
			st.Offer = m.CouponOfferName
		}
		if !status.Connected {
			st.Status = "fail"
		}
		out.Checks = append(out.Checks, st)
	}

	sc := newCheck("sender")
	if p.SenderID == nil {
		sc.Status = "fail"
	} else if sd, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: *p.SenderID}); err != nil {
		sc.Status = "fail"
	} else {
		sc.Sender = sd.Name + " <" + sd.Email + ">"
		if !senderVerified(sd.Verification) {
			sc.Status = "fail"
		}
	}
	out.Checks = append(out.Checks, sc)

	out.Pass = true
	for _, c := range out.Checks {
		if c.Status != "pass" {
			out.Pass = false
		}
	}
	if out.Links, err = s.Q.ReferralListSubscribers(ctx, p.ListID); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) couponStatus(ctx context.Context, ws *store.Workspace) CouponStatus {
	if s.Coupons == nil {
		return CouponStatus{}
	}
	c := s.Coupons()
	if c == nil {
		return CouponStatus{}
	}
	st, err := c.Status(ctx, ws)
	if err != nil {
		return CouponStatus{}
	}
	return st
}

// TurnOn turns a program on: its checks must pass. Every confirmed
// subscriber of the list gets a link.
func (s *Service) TurnOn(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralTurnedOn, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralTurnedOn{}, err
	}
	if cur.ReferralProgram.Status == "on" {
		return ReferralTurnedOn{}, api.Conflict(s.text(ctx, mAlreadyOn, nil))
	}
	checks, err := s.Checks(ctx, ws, id)
	if err != nil {
		return ReferralTurnedOn{}, err
	}
	if !checks.Pass {
		failing := []string{}
		for _, c := range checks.Checks {
			if c.Status != "pass" {
				failing = append(failing, c.Key)
			}
		}
		return ReferralTurnedOn{}, api.Err(http.StatusConflict, "checks_failing", s.text(ctx, mChecksFail, nil)).WithMeta("failing", failing)
	}
	by := actorUser(ctx)
	now := s.now()
	var created int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramSetStatus(ctx, store.ReferralProgramSetStatusParams{ID: id, Status: "on", Now: now, By: by}); err != nil {
			return err
		}
		n, err := s.ensureLinks(ctx, tx, cur.ReferralProgram, 0)
		if err != nil {
			return err
		}
		created = n
		action := "referral.program_turned_on"
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "referral_program", TargetID: id.String(),
			TargetName: cur.ListName, Before: map[string]any{"status": "off"}, After: map[string]any{"status": "on"},
			Metadata: map[string]any{"summary": "Links created: " + formatInt(n)}})
	})
	if err != nil {
		return ReferralTurnedOn{}, err
	}
	d, err := s.Detail(ctx, ws, id)
	if err != nil {
		return ReferralTurnedOn{}, err
	}
	out := ReferralTurnedOn{Program: d, LinksCreated: created}
	out.NextIssue = s.nextIssue(ctx, ws, cur.ReferralProgram.ListID)
	return out, nil
}

// nextIssue is the list's next draft or scheduled campaign.
func (s *Service) nextIssue(ctx context.Context, ws *store.Workspace, listID uuid.UUID) *ReferralCampaignRef {
	var ref ReferralCampaignRef
	var cid uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id, name, scheduled_at FROM campaigns
		WHERE workspace_id = $1 AND deleted_at IS NULL AND status IN ('draft', 'approval', 'approved', 'scheduled') AND digest_role IN ('', 'issue')
		  AND audience->'lists' ? $2::text
		ORDER BY coalesce(scheduled_at, updated_at) LIMIT 1`, ws.ID, listID.String()).Scan(&cid, &ref.Name, &ref.ScheduledAt)
	if err != nil {
		return nil
	}
	ref.ID = cid.String()
	return &ref
}

// TurnOff turns a program off: links lead to the normal subscribe page,
// Referral blocks are hidden and nothing new is attributed. Earned rewards
// stay available, and counting and rewards for referrals already made go on.
func (s *Service) TurnOff(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralProgramDetail, error) {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	if cur.ReferralProgram.Status == "off" {
		return ReferralProgramDetail{}, api.Conflict(s.text(ctx, mAlreadyOff, nil))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramSetStatus(ctx, store.ReferralProgramSetStatusParams{ID: id, Status: "off", Now: s.now(), By: actorUser(ctx)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.program_turned_off", TargetType: "referral_program",
			TargetID: id.String(), TargetName: cur.ListName, Before: map[string]any{"status": "on"}, After: map[string]any{"status": "off"}})
	})
	if err != nil {
		return ReferralProgramDetail{}, err
	}
	return s.Detail(ctx, ws, id)
}

// Delete removes a program that is off. Its links stop for good; counted
// referrals and sent rewards stay in contacts' timelines.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	cur, err := s.program(ctx, ws, id)
	if err != nil {
		return err
	}
	if cur.ReferralProgram.Status != "off" {
		return api.Err(http.StatusConflict, "program_on", s.text(ctx, mDeleteOn, nil))
	}
	links, _ := s.Q.ReferralLinkCount(ctx, id)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralProgramDelete(ctx, store.ReferralProgramDeleteParams{ID: id, Now: s.now(), By: actorUser(ctx)}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE referral_links SET status = 'paused' WHERE program_id = $1`, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.program_deleted", TargetType: "referral_program",
			TargetID: id.String(), TargetName: cur.ListName, Before: map[string]any{"status": "off", "links": links}})
	})
}

// ensureLinks gives every confirmed subscriber of the program's list a link,
// in batches (limit 0: all of them). It returns how many it created.
func (s *Service) ensureLinks(ctx context.Context, tx pgx.Tx, p store.ReferralProgram, limit int) (int64, error) {
	q := store.New(tx)
	var created int64
	now := s.now()
	for {
		batch := int32(5000)
		if limit > 0 && int(batch) > limit-int(created) {
			batch = int32(limit - int(created))
		}
		if batch <= 0 {
			return created, nil
		}
		ids, err := q.ReferralLinksMissing(ctx, store.ReferralLinksMissingParams{ListID: p.ListID, ProgramID: p.ID, Lim: batch})
		if err != nil {
			return created, err
		}
		if len(ids) == 0 {
			return created, nil
		}
		n, err := insertLinks(ctx, tx, p, ids, now)
		if err != nil {
			return created, err
		}
		created += n
		if len(ids) < int(batch) {
			return created, nil
		}
	}
}

// insertLinks creates links for contacts with fresh codes; a code another
// link already has is drawn again.
func insertLinks(ctx context.Context, tx pgx.Tx, p store.ReferralProgram, contacts []uuid.UUID, now time.Time) (int64, error) {
	var created int64
	pending := contacts
	for attempt := 0; attempt < 5 && len(pending) > 0; attempt++ {
		ids := make([]uuid.UUID, len(pending))
		codes := make([]string, len(pending))
		for i := range pending {
			ids[i] = uuid.Must(uuid.NewV7())
			codes[i] = NewCode()
		}
		rows, err := tx.Query(ctx, `INSERT INTO referral_links (id, workspace_id, program_id, contact_id, code, created_at)
			SELECT u.id, $1, $2, u.contact, u.code, $6 FROM unnest($3::uuid[], $4::uuid[], $5::text[]) AS u(id, contact, code)
			ON CONFLICT DO NOTHING
			RETURNING contact_id`, p.WorkspaceID, p.ID, ids, pending, codes, now)
		if err != nil {
			return created, err
		}
		done := map[uuid.UUID]bool{}
		for rows.Next() {
			var c uuid.UUID
			if err := rows.Scan(&c); err != nil {
				rows.Close()
				return created, err
			}
			done[c] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return created, err
		}
		created += int64(len(done))
		var left []uuid.UUID
		for _, c := range pending {
			if done[c] {
				continue
			}
			// A contact that has a link now (another process) is done too.
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM referral_links WHERE program_id = $1 AND contact_id = $2)`, p.ID, c).Scan(&exists); err != nil {
				return created, err
			}
			if !exists {
				left = append(left, c)
			}
		}
		pending = left
	}
	if len(pending) > 0 {
		return created, errors.New("referrals: could not draw unique codes")
	}
	return created, nil
}

// formatInt writes 31988 as "31,988" (audit summaries are English).
func formatInt(n int64) string {
	raw := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(raw, "-")
	var b strings.Builder
	for i, c := range raw {
		if i > 0 && (len(raw)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
