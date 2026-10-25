package polls

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service runs polls and surveys.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Links *linktoken.Signer
	URLs  publicurl.Builder
	// Forms gives the workspace's public page links (its verified custom
	// domain, else the instance's address).
	Forms *forms.Service
	Log   *slog.Logger
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Settings stored with a poll (polls.settings): everything of its config
// but the question and answers.
type pollSettings struct {
	Layout      string    `json:"layout"`
	Save        string    `json:"save"`
	Field       string    `json:"field,omitempty"`
	Close       PollClose `json:"close"`
	After       string    `json:"after"`
	Thanks      string    `json:"thanks,omitempty"`
	Survey      string    `json:"survey,omitempty"`
	NonContacts string    `json:"nonContacts"`
	Results     bool      `json:"results"`
	NPS         bool      `json:"nps,omitempty"`
	Low         string    `json:"low,omitempty"`
	High        string    `json:"high,omitempty"`
	Faces       int       `json:"faces,omitempty"`
}

func settingsOf(c PollConfig) pollSettings {
	return pollSettings{Layout: c.Layout, Save: c.Save, Field: c.Field, Close: c.Close, After: c.After, Thanks: c.Thanks, Survey: c.Survey,
		NonContacts: c.NonContacts, Results: c.Results, NPS: c.NPS, Low: c.Low, High: c.High, Faces: c.Faces}
}

// configOf rebuilds a poll's config from its row.
func configOf(blockID, kind, question string, answersRaw, settingsRaw json.RawMessage) (PollConfig, []Answer) {
	var st pollSettings
	_ = json.Unmarshal(settingsRaw, &st)
	var answers []Answer
	_ = json.Unmarshal(answersRaw, &answers)
	c := PollConfig{ID: blockID, Kind: kind, Question: question, Layout: st.Layout, Save: st.Save, Field: st.Field, Close: st.Close, After: st.After,
		Thanks: st.Thanks, Survey: st.Survey, NonContacts: st.NonContacts, Results: st.Results, NPS: st.NPS, Low: st.Low, High: st.High, Faces: st.Faces}
	for _, a := range answers {
		if kind == KindChoice {
			c.Answers = append(c.Answers, PollAnswerConfig{Label: a.Label, Emoji: a.Emoji, Tag: a.Tag})
		}
	}
	c.normalize()
	if answers == nil {
		answers = c.AnswerList()
	}
	return c, answers
}

// closesAt is when a poll stops recording: N days after the send started
// (unknown until it starts), the end of a date in the workspace's time
// zone, or never (closed by hand).
func closesAt(c PollClose, started *time.Time, tz string) *time.Time {
	switch c.Mode {
	case CloseDays:
		if started == nil || c.Days <= 0 {
			return nil
		}
		t := started.Add(time.Duration(c.Days) * 24 * time.Hour)
		return &t
	case CloseDate:
		d, err := time.ParseInLocation("2006-01-02", c.Date, state.Location(tz))
		if err != nil {
			return nil
		}
		t := d.Add(24*time.Hour - time.Second)
		return &t
	}
	return nil
}

// Synced is a poll of an email after Sync: its id, number and settings.
type Synced struct {
	ID      uuid.UUID
	Number  int64
	Config  PollConfig
	Answers []Answer
}

// Sync copies the polls of a campaign's content into polls, so their links
// can be signed (a test send, a send). started is when the send started,
// nil before.
func (s *Service) Sync(ctx context.Context, q *store.Queries, wsID, campaignID uuid.UUID, src string, started *time.Time, tz string) ([]Synced, error) {
	blocks := Read(src)
	out := make([]Synced, 0, len(blocks))
	cid := campaignID
	for _, b := range blocks {
		c := b.Config
		answers := c.AnswerList()
		ab, _ := json.Marshal(answers)
		sb, _ := json.Marshal(settingsOf(c))
		row, err := q.PollsUpsert(ctx, store.PollsUpsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, CampaignID: &cid, BlockID: c.ID, Kind: c.Kind,
			Question: strings.TrimSpace(c.Question), Answers: ab, Settings: sb, ClosesAt: closesAt(c.Close, started, tz), Now: s.now()})
		if err != nil {
			return nil, err
		}
		out = append(out, Synced{ID: row.ID, Number: row.Number, Config: c, Answers: answers})
	}
	return out, nil
}

// syncCampaign syncs a campaign's polls from its saved content.
func (s *Service) syncCampaign(ctx context.Context, wsID, campaignID uuid.UUID) ([]Synced, *store.Workspace, error) {
	ws, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.Q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return nil, nil, err
	}
	synced, err := s.Sync(ctx, s.Q, wsID, campaignID, c.Mjml, c.StartedAt, ws.Timezone)
	return synced, &ws, err
}

// pageLinks are the workspace's public page links: on its verified custom
// domain, else on the instance's address.
func (s *Service) pageLinks(ctx context.Context, ws *store.Workspace) forms.PageLinks {
	if s.Forms != nil {
		if set, err := s.Forms.LoadSettings(ctx, ws); err == nil {
			return s.Forms.Links(ws, set)
		}
	}
	return forms.LinksOn(s.URLs.Base(), ws.Slug, false)
}

// sign signs a poll or survey token.
func (s *Service) sign(kind linktoken.Kind, wsID uuid.UUID, number int64, message, contact uuid.UUID) string {
	return s.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: wsID, MessageID: message, ContactID: contact, Link: uint32(number)})
}

// surveyVarRe finds survey link variables in content: {{ rc_survey_12 }}.
var surveyVarRe = regexp.MustCompile(`rc_survey_(\d{1,12})`)

// surveysIn lists the survey numbers an email links to.
func surveysIn(src string) []int64 {
	out := []int64{}
	seen := map[int64]bool{}
	for _, m := range surveyVarRe.FindAllStringSubmatch(src, -1) {
		n, err := strconv.ParseInt(m[1], 10, 32)
		if err == nil && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// RecipientLink is a per-recipient link variable of an email.
type RecipientLink struct {
	Var string
	URL func(messageID, contactID uuid.UUID) string
}

// SendLinks syncs a campaign's polls as its send starts and returns the
// link variables each recipient gets: every poll's answer link base and
// every survey's link.
func (s *Service) SendLinks(ctx context.Context, wsID, campaignID uuid.UUID) ([]RecipientLink, error) {
	if s.Links == nil {
		return nil, nil
	}
	synced, ws, err := s.syncCampaign(ctx, wsID, campaignID)
	if err != nil {
		return nil, err
	}
	links := s.pageLinks(ctx, ws)
	out := make([]RecipientLink, 0, len(synced))
	for _, p := range synced {
		number := p.Number
		out = append(out, RecipientLink{Var: VarName(p.Config.ID), URL: func(msg, contact uuid.UUID) string {
			return links.Poll(s.sign(linktoken.Poll, wsID, number, msg, contact))
		}})
	}
	c, err := s.Q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: wsID, ID: campaignID})
	if err != nil {
		return nil, err
	}
	for _, n := range surveysIn(c.Mjml) {
		sv, err := s.Q.SurveyByNumber(ctx, n)
		if err != nil || sv.WorkspaceID != wsID {
			continue
		}
		number, slug := n, sv.Slug
		out = append(out, RecipientLink{Var: SurveyVar(n), URL: func(msg, contact uuid.UUID) string {
			return links.Survey(slug) + "?t=" + s.sign(linktoken.Survey, wsID, number, msg, contact)
		}})
	}
	return out, nil
}

// PreviewLinks are a preview's poll and survey link variables: links that
// go nowhere, so nothing is recorded from the composer.
func PreviewLinks(src string) map[string]string {
	out := map[string]string{}
	for _, b := range Read(src) {
		out[VarName(b.Config.ID)] = "#poll-" + b.Config.ID
	}
	for _, n := range surveysIn(src) {
		out[SurveyVar(n)] = "#survey-" + strconv.FormatInt(n, 10)
	}
	return out
}

// TestLinks are the link variables of a test email to one address: its
// answers are recorded as that address's test answers, never counted.
func (s *Service) TestLinks(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, email string) (map[string]string, error) {
	out := map[string]string{}
	c, err := s.Q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return nil, err
	}
	if !strings.Contains(c.Mjml, "rc-poll") && !strings.Contains(c.Mjml, "rc_survey_") {
		return out, nil
	}
	synced, err := s.Sync(ctx, s.Q, ws.ID, campaignID, c.Mjml, c.StartedAt, ws.Timezone)
	if err != nil {
		return nil, err
	}
	links := s.pageLinks(ctx, ws)
	surveys := surveysIn(c.Mjml)
	if len(synced) == 0 && len(surveys) == 0 {
		return out, nil
	}
	// The test address's row: its answers are test answers, per address.
	rid := uuid.Must(uuid.NewV7())
	cid := campaignID
	if err := s.Q.PollTestRecipientInsert(ctx, store.PollTestRecipientInsertParams{ID: rid, WorkspaceID: ws.ID, CampaignID: &cid, Email: strings.ToLower(email), At: s.now()}); err != nil {
		return nil, err
	}
	for _, p := range synced {
		out[VarName(p.Config.ID)] = links.Poll(s.sign(linktoken.PollTest, ws.ID, p.Number, rid, uuid.Nil))
	}
	for _, n := range surveys {
		if sv, err := s.Q.SurveyByNumber(ctx, n); err == nil && sv.WorkspaceID == ws.ID {
			out[SurveyVar(n)] = links.Survey(sv.Slug) + "?t=" + s.sign(linktoken.SurveyTest, ws.ID, n, rid, uuid.Nil)
		}
	}
	return out, nil
}

// BrowserLinks are the link variables of an email shown on the web archive
// or "View in browser": its links carry no contact's token, so answers
// from there count as non-contacts' (or are refused, as each poll says).
// Polls are created when a test or the send needs them; until then the
// answer links go nowhere.
func (s *Service) BrowserLinks(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) map[string]string {
	out := map[string]string{}
	if s.Links == nil {
		return out
	}
	c, err := s.Q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return out
	}
	for k, v := range PreviewLinks(c.Mjml) {
		out[k] = v
	}
	links := s.pageLinks(ctx, ws)
	rows, err := s.Q.PollsOfCampaign(ctx, store.PollsOfCampaignParams{WorkspaceID: ws.ID, CampaignID: &campaignID})
	if err == nil {
		for _, p := range rows {
			out[VarName(p.BlockID)] = links.Poll(s.sign(linktoken.Poll, ws.ID, p.Number, uuid.Nil, uuid.Nil))
		}
	}
	for _, n := range surveysIn(c.Mjml) {
		if sv, err := s.Q.SurveyByNumber(ctx, n); err == nil && sv.WorkspaceID == ws.ID {
			out[SurveyVar(n)] = links.Survey(sv.Slug)
		}
	}
	return out
}
