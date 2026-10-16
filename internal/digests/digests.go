package digests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// DigestCreateInput starts a digest from the type picker.
type DigestCreateInput struct {
	Name string `json:"name,omitempty" maxLength:"200"`
	// Source is a connected WordPress site (SiteID) or a feed or page address (FeedURL).
	Source  string `json:"source" enum:"url,wordpress"`
	SiteID  string `json:"siteId,omitempty" format:"uuid"`
	FeedURL string `json:"feedUrl,omitempty" maxLength:"500"`
}

// DigestSettingsInput is the digest settings page.
type DigestSettingsInput struct {
	Name    string        `json:"name" minLength:"1" maxLength:"200"`
	Archive bool          `json:"archive"`
	Mode    string        `json:"mode" enum:"auto,draft"`
	Source  string        `json:"source" enum:"url,wordpress"`
	SiteID  string        `json:"siteId,omitempty" format:"uuid"`
	FeedURL string        `json:"feedUrl,omitempty" maxLength:"500"`
	Filters DigestFilters `json:"filters"`
	// CategoryFilter: when false, every category goes in (Categories is ignored).
	CategoryFilter bool           `json:"categoryFilter"`
	Schedule       DigestSchedule `json:"schedule"`
	PrepareHours   int            `json:"prepareHours" minimum:"1" maximum:"24"`
	LateHours      int            `json:"lateHours" enum:"6,24,48"`
	// Audience: lists and segments of the issues.
	ListIDs    []string `json:"listIds" maxItems:"20"`
	SegmentIDs []string `json:"segmentIds" maxItems:"20"`
	SenderID   string   `json:"senderId,omitempty" maxLength:"64"`
	// Thresholds are the auto-pause limits (nil: the workspace's).
	Thresholds *DigestThresholdsInput `json:"thresholds,omitempty"`
}

// DigestThresholdsInput sets a digest's own auto-pause limits.
type DigestThresholdsInput struct {
	Bounce    float64 `json:"bounce" minimum:"0.1" maximum:"20"`
	Complaint float64 `json:"complaint" minimum:"0.01" maximum:"5"`
	Scope     string  `json:"scope" enum:"route,campaign"`
}

// DigestTurnOnInput turns a digest on (or back on).
type DigestTurnOnInput struct {
	Mode string `json:"mode,omitempty" enum:"auto,draft,"`
}

// DigestPauseInput pauses a digest, with an optional note.
type DigestPauseInput struct {
	Note string `json:"note,omitempty" maxLength:"500"`
}

var (
	mNameRequired   = msgs.M("name.required")
	mNameTaken      = msgs.M("name.taken")
	mFeedRequired   = msgs.M("feed.required")
	mFeedSite       = msgs.M("feed.site")
	mFeedAddress    = msgs.M("feed.address")
	mCategoriesNone = msgs.M("categories.none")
	mSchedule       = msgs.M("schedule.invalid")
	mPrepare        = msgs.M("prepare.range")
	mLate           = msgs.M("late.values")
	mAudience       = msgs.M("audience.required")
	mAudienceGone   = msgs.M("audience.unknown")
	mSenderGone     = msgs.M("sender.unknown")
	mStartLatest    = msgs.M("start.latest")
	mOnNeedsSend    = msgs.M("on.needsSend")
	mOnAlready      = msgs.M("on.already")
	mOnNeedsFeed    = msgs.M("on.needsFeed")
	mOnNeedsAud     = msgs.M("on.needsAudience")
	mOnNeedsTpl     = msgs.M("on.needsTemplate")
	mPauseNotOn     = msgs.M("pause.notOn")
	mDeleteOn       = msgs.M("delete.on")
)

// Create makes a digest from the type picker: off, in draft mode for a
// member without campaigns:send, with the Blog digest template, the
// workspace's default sender and its time zone.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in DigestCreateInput) (DigestDetail, error) {
	l := i18n.LangFrom(ctx)
	src, siteID, feedURL, err := s.resolveSource(ctx, ws, l, in.Source, in.SiteID, in.FeedURL)
	if err != nil {
		return DigestDetail{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Blog digest"
		if siteID != nil {
			if site, err := s.site(ctx, ws, *siteID); err == nil && site != nil {
				name = site.Host + " digest"
			}
		} else if h := hostOf(feedURL); h != "" {
			name = h + " digest"
		}
	}
	name = s.freeName(ctx, ws.ID, name)
	mode := "auto"
	if !p.Can(authz.CampaignsSend) {
		mode = "draft"
	}
	id := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())
	now := s.now()
	by := actorID(p)
	sched, _ := json.Marshal(DefaultSchedule(ws.Timezone))
	mjmlSrc, _ := templates.StarterMJML(templates.BlogDigestStarter, templates.WorkspaceAddress(ws))
	subject := `{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from {{ feed.title }}`
	preheader := "New on " + firstNonEmpty(hostOf(feedURL), "the blog") + " · issue {{ issue.number }}"
	html := mjml.Compile(mjmlSrc).HTML
	html = accessibility.FinalizeStored(ctx, s.Q, ws.ID, accessibility.OwnerCampaign, uuid.Nil, 0, html)
	var sender *uuid.UUID
	if def, err := s.Q.ComposerDefaultSender(ctx, ws.ID); err == nil && def.SenderID != "" {
		if u, err := uuid.Parse(def.SenderID); err == nil {
			sender = &u
		}
	}
	if sender == nil {
		if senders, err := s.Q.ListComposerSenders(ctx, ws.ID); err == nil && len(senders) > 0 {
			sender = &senders[0].ID
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DigestInsert(ctx, store.DigestInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Mode: mode, Source: src, FeedUrl: feedURL, SiteID: siteID,
			Categories: []string{}, Tags: []string{}, MaxAgeDays: ptr32(14), StartMode: "after_on", StartLatest: 3, Schedule: sched, PrepareHours: 3,
			LateHours: 24, Archive: true, CreatedBy: by, Now: now}); err != nil {
			return err
		}
		if err := q.DigestCampaignInsert(ctx, store.DigestCampaignInsertParams{ID: templateID, WorkspaceID: ws.ID, Name: name, Subject: subject, Preheader: preheader,
			SenderID: sender, ReplyTo: "", Audience: []byte(`{}`), Mjml: mjmlSrc, Html: html, PlainText: contentcheck.PlainText(html),
			Schedule: []byte(`{"mode":"now"}`), Thresholds: []byte(`{}`), Tracking: []byte(`{"opens":true,"clicks":true}`),
			ContentHash: campaigns.ContentHash(subject, preheader, mjmlSrc, html, nil), ContentSettings: []byte(`{}`), DigestID: &id, DigestRole: "template",
			CreatedBy: by, Now: now}); err != nil {
			return err
		}
		if err := q.DigestCampaignVersionInsert(ctx, store.DigestCampaignVersionInsertParams{CampaignID: templateID, Version: 1, Subject: subject, Preheader: preheader,
			Mjml: mjmlSrc, Html: html, PlainText: contentcheck.PlainText(html), SavedBy: by, SavedAt: now, Label: "created from the Blog digest starter",
			Source: "template", SizeBytes: int32(len(html))}); err != nil {
			return err
		}
		if err := q.DigestSetTemplate(ctx, store.DigestSetTemplateParams{ID: id, TemplateID: &templateID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.created", TargetType: "digest", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "source": src, "feed": feedURL, "mode": mode}})
	})
	if err != nil {
		return DigestDetail{}, err
	}
	// The first read lists the feed's items in the preview at once.
	if _, err := s.Read(ctx, ws, id, "setup"); err != nil {
		s.log().InfoContext(ctx, "first read of a new digest's feed", "digest", id, "err", err)
	}
	return s.Detail(ctx, ws, p, id)
}

func ptr32(v int32) *int32 { return &v }

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// freeName adds " 2", " 3"… to a name another digest has.
func (s *Service) freeName(ctx context.Context, wsID uuid.UUID, name string) string {
	try := name
	for i := 2; i < 50; i++ {
		taken, err := s.Q.DigestNameTaken(ctx, store.DigestNameTakenParams{WorkspaceID: wsID, Name: try, ID: uuid.Nil})
		if err != nil || !taken {
			return try
		}
		try = fmt.Sprintf("%s %d", name, i)
	}
	return try
}

// resolveSource checks a feed source: a connected WordPress site of the
// workspace, or an address the guard accepts.
func (s *Service) resolveSource(ctx context.Context, ws *store.Workspace, l i18n.Lang, source, siteRaw, feedRaw string) (string, *uuid.UUID, string, error) {
	switch source {
	case "wordpress":
		id, err := uuid.Parse(siteRaw)
		if err != nil {
			return "", nil, "", api.Invalid("body.siteId", mFeedSite.Text(l, nil))
		}
		site, err := s.site(ctx, ws, id)
		if err != nil {
			return "", nil, "", err
		}
		if site == nil {
			return "", nil, "", api.Invalid("body.siteId", mFeedSite.Text(l, nil))
		}
		return "wordpress", &id, site.FeedURL, nil
	default:
		if strings.TrimSpace(feedRaw) == "" {
			return "", nil, "", api.Invalid("body.feedUrl", mFeedRequired.Text(l, nil))
		}
		u, err := feed.NormalizeURL(feedRaw)
		if err != nil {
			return "", nil, "", api.Invalid("body.feedUrl", mFeedAddress.Text(l, i18n.Args{"detail": strings.TrimSuffix(sentence(err.Error()), ".")}))
		}
		if _, err := s.fetcher().Policy.ValidateURL(u); err != nil {
			return "", nil, "", api.Invalid("body.feedUrl", mFeedAddress.Text(l, i18n.Args{"detail": strings.TrimSuffix(sentence(err.Error()), ".")}))
		}
		return "url", nil, u, nil
	}
}

func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Update saves the settings page. Template, feed and schedule changes apply
// from the next preparation; a prepared issue keeps its content.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in DigestSettingsInput) (DigestDetail, error) {
	l := i18n.LangFrom(ctx)
	row, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestDetail{}, api.MapError(err, "Digest")
	}
	d := row.Digest
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return DigestDetail{}, api.Invalid("body.name", mNameRequired.Text(l, nil))
	}
	if taken, err := s.Q.DigestNameTaken(ctx, store.DigestNameTakenParams{WorkspaceID: ws.ID, Name: name, ID: id}); err != nil {
		return DigestDetail{}, err
	} else if taken {
		return DigestDetail{}, api.Invalid("body.name", mNameTaken.Text(l, i18n.Args{"name": name}))
	}
	src, siteID, feedURL, err := s.resolveSource(ctx, ws, l, in.Source, in.SiteID, in.FeedURL)
	if err != nil {
		return DigestDetail{}, err
	}
	if in.Mode == "auto" && d.Mode != "auto" && d.State == "on" && !p.Can(authz.CampaignsSend) {
		return DigestDetail{}, api.Forbidden(authz.CampaignsSend, nil).WithMeta("reason", mOnNeedsSend.Text(l, nil))
	}
	cats := cleanTerms(in.Filters.Categories)
	if !in.CategoryFilter {
		cats = []string{}
	} else if len(cats) == 0 {
		return DigestDetail{}, api.Invalid("body.filters.categories", mCategoriesNone.Text(l, nil))
	}
	tags := cleanTerms(in.Filters.Tags)
	sched := in.Schedule
	if sched.Timezone == "" {
		sched.Timezone = ws.Timezone
	}
	if field, msg := sched.Validate(); field != "" {
		return DigestDetail{}, api.Invalid("body.schedule."+field, mSchedule.Text(l, i18n.Args{"detail": msg}))
	}
	if in.PrepareHours < 1 || in.PrepareHours > 24 {
		return DigestDetail{}, api.Invalid("body.prepareHours", mPrepare.Text(l, nil))
	}
	if in.LateHours != 6 && in.LateHours != 24 && in.LateHours != 48 {
		return DigestDetail{}, api.Invalid("body.lateHours", mLate.Text(l, nil))
	}
	startLatest := in.Filters.StartLatest
	if startLatest == 0 {
		startLatest = 3
	}
	if startLatest < 1 || startLatest > 20 {
		return DigestDetail{}, api.Invalid("body.filters.startLatest", mStartLatest.Text(l, nil))
	}
	startMode := in.Filters.StartMode
	if startMode != "latest" {
		startMode = "after_on"
	}
	var maxAge *int32
	if in.Filters.MaxAgeDays != nil {
		maxAge = ptr32(int32(*in.Filters.MaxAgeDays))
	}
	// The template's audience, sender and limits.
	tpl, err := s.template(ctx, d)
	if err != nil {
		return DigestDetail{}, err
	}
	audience, err := s.audienceJSON(ctx, ws, l, in.ListIDs, in.SegmentIDs)
	if err != nil {
		return DigestDetail{}, err
	}
	if d.State == "on" && string(audience) == "{}" {
		return DigestDetail{}, api.Invalid("body.listIds", mAudience.Text(l, nil))
	}
	// No sender sent keeps the template's.
	var sender *uuid.UUID
	if tpl != nil {
		sender = tpl.SenderID
	}
	if in.SenderID != "" {
		sid, err := uuid.Parse(in.SenderID)
		if err != nil {
			return DigestDetail{}, api.Invalid("body.senderId", mSenderGone.Text(l, nil))
		}
		ok, err := s.senderExists(ctx, ws.ID, sid)
		if err != nil {
			return DigestDetail{}, err
		}
		if !ok {
			return DigestDetail{}, api.Invalid("body.senderId", mSenderGone.Text(l, nil))
		}
		sender = &sid
	}
	thresholds := []byte(`{}`)
	if in.Thresholds != nil {
		thresholds, _ = json.Marshal(map[string]any{"bounce": in.Thresholds.Bounce, "complaint": in.Thresholds.Complaint, "scope": firstNonEmpty(in.Thresholds.Scope, "route")})
	}
	schedJSON, _ := json.Marshal(sched)
	before := settingsAudit(d, tpl)
	now := s.now()
	feedChanged := feedURL != d.FeedUrl
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DigestUpdateSettings(ctx, store.DigestUpdateSettingsParams{ID: id, Name: name, Mode: in.Mode, Source: src, FeedUrl: feedURL, SiteID: siteID,
			Categories: cats, Tags: tags, MaxAgeDays: maxAge, StartMode: startMode, StartLatest: int32(startLatest), Schedule: schedJSON,
			PrepareHours: int32(in.PrepareHours), LateHours: int32(in.LateHours), Archive: in.Archive, UpdatedBy: actorID(p), Now: now}); err != nil {
			return err
		}
		if feedChanged {
			if err := q.DigestResetFeed(ctx, id); err != nil {
				return err
			}
		}
		if tpl != nil {
			if err := q.DigestTemplateSettings(ctx, store.DigestTemplateSettingsParams{ID: tpl.ID, Name: name, Audience: audience, SenderID: sender,
				Thresholds: thresholds, Now: now}); err != nil {
				return err
			}
		}
		after := settingsAudit(store.Digest{Name: name, Mode: in.Mode, Source: src, FeedUrl: feedURL, SiteID: siteID, Categories: cats, Tags: tags, MaxAgeDays: maxAge,
			StartMode: startMode, StartLatest: int32(startLatest), Schedule: schedJSON, PrepareHours: int32(in.PrepareHours), LateHours: int32(in.LateHours),
			Archive: in.Archive}, &store.DigestCampaignTemplateRow{Audience: audience, SenderID: sender, Thresholds: thresholds})
		changed := changedKeys(before, after)
		if len(changed) == 0 {
			return nil
		}
		b, a := map[string]any{}, map[string]any{}
		for _, k := range changed {
			b[k], a[k] = before[k], after[k]
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.updated", TargetType: "digest", TargetID: id.String(), TargetName: name,
			Before: b, After: a, Metadata: map[string]any{"changed": changed}})
	})
	if err != nil {
		return DigestDetail{}, err
	}
	// A schedule change moves the issue being collected.
	if err := s.reschedule(ctx, ws, id); err != nil {
		return DigestDetail{}, err
	}
	if feedChanged {
		if _, err := s.Read(ctx, ws, id, "setup"); err != nil {
			s.log().InfoContext(ctx, "read a digest's new feed", "digest", id, "err", err)
		}
	}
	return s.Detail(ctx, ws, p, id)
}

func cleanTerms(xs []string) []string {
	out := []string{}
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x != "" && !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, x) }) {
			out = append(out, x)
		}
	}
	return out
}

// settingsAudit is what the audit entry of a settings change compares.
func settingsAudit(d store.Digest, tpl *store.DigestCampaignTemplateRow) map[string]any {
	var sched DigestSchedule
	_ = json.Unmarshal(d.Schedule, &sched)
	out := map[string]any{
		"name": d.Name, "mode": d.Mode, "archive": d.Archive, "source": d.Source, "feed": d.FeedUrl,
		"categories": d.Categories, "tags": d.Tags, "start": d.StartMode, "schedule": sched.Describe(),
		"prepare_hours": d.PrepareHours, "late_hours": d.LateHours,
	}
	if d.SiteID != nil {
		out["site"] = d.SiteID.String()
	}
	if d.MaxAgeDays != nil {
		out["max_age_days"] = *d.MaxAgeDays
	} else {
		out["max_age_days"] = nil
	}
	if d.StartMode == "latest" {
		out["start"] = fmt.Sprintf("latest %d", d.StartLatest)
	}
	if tpl != nil {
		var aud any
		_ = json.Unmarshal(tpl.Audience, &aud)
		out["audience"] = aud
		if tpl.SenderID != nil {
			out["sender"] = tpl.SenderID.String()
		} else {
			out["sender"] = nil
		}
		var th any
		_ = json.Unmarshal(tpl.Thresholds, &th)
		out["thresholds"] = th
	}
	return out
}

func changedKeys(a, b map[string]any) []string {
	out := []string{}
	for _, k := range []string{"name", "mode", "archive", "source", "site", "feed", "categories", "tags", "max_age_days", "start", "schedule", "prepare_hours",
		"late_hours", "audience", "sender", "thresholds"} {
		x, _ := json.Marshal(a[k])
		y, _ := json.Marshal(b[k])
		if string(x) != string(y) {
			out = append(out, k)
		}
	}
	return out
}

// audienceJSON validates lists and segments into an audience definition.
func (s *Service) audienceJSON(ctx context.Context, ws *store.Workspace, l i18n.Lang, lists, segments []string) (json.RawMessage, error) {
	ls, ss := []uuid.UUID{}, []uuid.UUID{}
	known, err := s.Q.ListWorkspaceLists(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	have := map[uuid.UUID]bool{}
	for _, k := range known {
		have[k.ID] = true
	}
	for _, raw := range lists {
		id, err := uuid.Parse(raw)
		if err != nil || !have[id] {
			return nil, api.Invalid("body.listIds", mAudienceGone.Text(l, nil))
		}
		ls = append(ls, id)
	}
	for _, raw := range segments {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, api.Invalid("body.segmentIds", mAudienceGone.Text(l, nil))
		}
		exists, err := s.Q.DigestSegmentExists(ctx, store.DigestSegmentExistsParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, api.Invalid("body.segmentIds", mAudienceGone.Text(l, nil))
		}
		ss = append(ss, id)
	}
	if len(ls) == 0 && len(ss) == 0 {
		return []byte(`{}`), nil
	}
	out := map[string]any{}
	if len(ls) > 0 {
		out["lists"] = ls
	}
	if len(ss) > 0 {
		out["segments"] = ss
	}
	b, _ := json.Marshal(out)
	return b, nil
}

func (s *Service) senderExists(ctx context.Context, wsID, id uuid.UUID) (bool, error) {
	_, err := s.Q.GetComposerSender(ctx, store.GetComposerSenderParams{WorkspaceID: wsID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// template loads a digest's template campaign.
func (s *Service) template(ctx context.Context, d store.Digest) (*store.DigestCampaignTemplateRow, error) {
	if d.TemplateID == nil {
		return nil, nil
	}
	t, err := s.Q.DigestCampaignTemplate(ctx, *d.TemplateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TurnOn turns a digest on, or back on after a pause. Sending
// automatically needs campaigns:send; draft mode needs campaigns:edit.
// Whoever turns it on becomes its owner: issues send on their authority.
func (s *Service) TurnOn(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in DigestTurnOnInput) (DigestDetail, error) {
	l := i18n.LangFrom(ctx)
	row, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestDetail{}, api.MapError(err, "Digest")
	}
	d := row.Digest
	if d.State == "on" && (in.Mode == "" || in.Mode == d.Mode) {
		return DigestDetail{}, api.Conflict(mOnAlready.Text(l, nil))
	}
	mode := firstNonEmpty(in.Mode, d.Mode)
	ms, err := s.members(ctx, ws.ID)
	if err != nil {
		return DigestDetail{}, err
	}
	if mode == "auto" && !p.Can(authz.CampaignsSend) {
		return DigestDetail{}, api.Forbidden(authz.CampaignsSend, helpers(ms, authz.CampaignsSend, d.OwnerID)).WithMeta("reason", mOnNeedsSend.Text(l, nil))
	}
	tpl, err := s.template(ctx, d)
	if err != nil {
		return DigestDetail{}, err
	}
	if tpl == nil || !strings.Contains(tpl.Mjml, "<mj-body") {
		return DigestDetail{}, api.Conflict(mOnNeedsTpl.Text(l, nil))
	}
	if string(tpl.Audience) == "{}" || len(tpl.Audience) == 0 {
		return DigestDetail{}, api.Err(http.StatusConflict, "digest_needs_audience", mOnNeedsAud.Text(l, nil))
	}
	if d.FeedStatus == "unread" || d.LastOkAt == nil {
		return DigestDetail{}, api.Err(http.StatusConflict, "digest_feed_unread", mOnNeedsFeed.Text(l, nil))
	}
	now := s.now()
	prevOwner := ""
	if d.OwnerID != nil {
		prevOwner = row.OwnerName
	}
	action := "digest.turned_on"
	if d.State == "paused" {
		action = "digest.resumed"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.DigestTurnOn(ctx, store.DigestTurnOnParams{ID: id, Mode: mode, OwnerID: &p.UserID, Now: now}); err != nil {
			return err
		}
		before := map[string]any{"state": d.State, "mode": d.Mode}
		after := map[string]any{"state": "on", "mode": mode}
		if prevOwner != p.Name {
			before["owner"], after["owner"] = prevOwner, p.Name
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "digest", TargetID: id.String(), TargetName: d.Name,
			Before: before, After: after, Metadata: map[string]any{"permission": map[bool]string{true: "campaigns:send", false: "campaigns:edit"}[mode == "auto"]}})
	})
	if err != nil {
		return DigestDetail{}, err
	}
	if err := s.ensureCollecting(ctx, ws, id); err != nil {
		return DigestDetail{}, err
	}
	return s.Detail(ctx, ws, p, id)
}

// Pause stops new issues from being prepared. An issue waiting to go out
// is skipped (its posts stay new); one that is sending pauses like any send.
func (s *Service) Pause(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in DigestPauseInput) (DigestDetail, error) {
	l := i18n.LangFrom(ctx)
	row, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestDetail{}, api.MapError(err, "Digest")
	}
	d := row.Digest
	if d.State != "on" {
		return DigestDetail{}, api.Conflict(mPauseNotOn.Text(l, nil))
	}
	note := strings.TrimSpace(in.Note)
	if err := s.pause(ctx, ws, d, "person", note, actorID(p), nil); err != nil {
		return DigestDetail{}, err
	}
	// The issue in flight: a sending one pauses, a waiting one is skipped.
	if issue, err := s.Q.DigestIssueOpen(ctx, id); err == nil {
		if issue.Status != "collecting" {
			if err := s.skip(ctx, ws, d, issue, "person", note, actorID(p)); err != nil {
				return DigestDetail{}, err
			}
		} else if err := s.Q.DigestIssueDelete(ctx, issue.ID); err != nil {
			return DigestDetail{}, err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return DigestDetail{}, err
	}
	if sending, err := s.sendingIssue(ctx, id); err == nil && sending != nil && sending.CampaignID != nil && s.starter() != nil {
		if err := s.starter().Pause(ctx, *sending.CampaignID, p.ID, "The digest was paused"); err != nil {
			s.log().WarnContext(ctx, "pause a digest issue's send", "digest", id, "err", err)
		}
	}
	return s.Detail(ctx, ws, p, id)
}

// pause records a pause and tells who should hear about it.
func (s *Service) pause(ctx context.Context, ws *store.Workspace, d store.Digest, reason, note string, by *uuid.UUID, detail map[string]any) error {
	now := s.now()
	if detail == nil {
		detail = map[string]any{}
	}
	notified := []string{}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// Who hears about it: the owner (unless they paused it), and everyone
		// with campaigns:send when the owner can't send any more.
		targets := []uuid.UUID{}
		if d.OwnerID != nil && (by == nil || *by != *d.OwnerID) && reason != "owner" {
			targets = append(targets, *d.OwnerID)
		}
		if reason == "owner" {
			ids, err := notify.MembersWith(ctx, tx, ws.ID, authz.CampaignsSend)
			if err != nil {
				return err
			}
			if d.OwnerID != nil {
				targets = append(targets, *d.OwnerID)
			}
			for _, id := range ids {
				if d.OwnerID == nil || id != *d.OwnerID {
					targets = append(targets, id)
					if u, err := q.GetUser(ctx, id); err == nil {
						notified = append(notified, u.Name)
					}
				}
			}
		}
		if reason == "feed" && d.OwnerID != nil {
			targets = append(targets, *d.OwnerID)
		}
		detail["notified"] = notified
		db1, _ := json.Marshal(detail)
		if err := q.DigestPause(ctx, store.DigestPauseParams{ID: d.ID, Now: now, PausedBy: by, PauseReason: reason, PauseNote: note, PauseDetail: db1}); err != nil {
			return err
		}
		var actor *audit.Actor
		if by == nil {
			actor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}
		}
		meta := map[string]any{"reason": reason}
		for k, v := range detail {
			meta[k] = v
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.paused", TargetType: "digest", TargetID: d.ID.String(), TargetName: d.Name,
			Actor: actor, Reason: note, Before: map[string]any{"state": d.State}, After: map[string]any{"state": "paused"}, Metadata: meta}); err != nil {
			return err
		}
		n := pausedNotification(ws, d, reason, note, by, q, ctx, detail)
		told := map[uuid.UUID]bool{}
		for _, u := range targets {
			if told[u] {
				continue
			}
			told[u] = true
			if err := notify.ToUser(ctx, tx, u, n); err != nil {
				return err
			}
		}
		return nil
	})
}

// sendingIssue is the digest's issue that is sending, if any.
func (s *Service) sendingIssue(ctx context.Context, digestID uuid.UUID) (*store.DigestIssue, error) {
	issues, err := s.Q.DigestIssuesInFlight(ctx, digestID)
	if err != nil {
		return nil, err
	}
	for i := range issues {
		if issues[i].Status == "sending" {
			return &issues[i], nil
		}
	}
	return nil, nil
}

// Delete removes a digest that isn't on. Its sent issues stay as campaigns.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	l := i18n.LangFrom(ctx)
	row, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Digest")
	}
	d := row.Digest
	if d.State == "on" {
		return api.Conflict(mDeleteOn.Text(l, nil))
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if issue, err := q.DigestIssueOpen(ctx, id); err == nil {
			if issue.CampaignID != nil {
				if err := q.DigestCampaignCancelApproval(ctx, store.DigestCampaignCancelApprovalParams{CampaignID: issue.CampaignID, Now: now, Reason: "digest deleted"}); err != nil {
					return err
				}
				if err := q.DigestCampaignRemove(ctx, store.DigestCampaignRemoveParams{ID: *issue.CampaignID, Now: now}); err != nil {
					return err
				}
			}
			if err := q.DigestIssueSkip(ctx, store.DigestIssueSkipParams{ID: issue.ID, Status: "skipped", Now: now, SkipReason: "person", SkipNote: "digest deleted",
				SkippedBy: actorID(p)}); err != nil {
				return err
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if d.TemplateID != nil {
			if err := q.DigestTemplateRemove(ctx, store.DigestTemplateRemoveParams{ID: *d.TemplateID, Now: now}); err != nil {
				return err
			}
		}
		if err := q.DigestSoftDelete(ctx, store.DigestSoftDeleteParams{ID: id, Now: now, UpdatedBy: actorID(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "digest.deleted", TargetType: "digest", TargetID: id.String(), TargetName: d.Name,
			Before: map[string]any{"state": d.State, "mode": d.Mode}})
	})
}

// pausedNotification is the "Digest paused" notification.
func pausedNotification(ws *store.Workspace, d store.Digest, reason, note string, by *uuid.UUID, q *store.Queries, ctx context.Context, detail map[string]any) notify.Notification {
	link := "/w/" + ws.Slug + "/campaigns/digests/" + d.ID.String()
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindDigestPaused, Severity: "warn", Title: "Digest paused", Link: link,
		Data: map[string]any{"digest_id": d.ID.String(), "reason": reason}}
	switch reason {
	case "owner":
		owner, _ := detail["owner"].(string)
		from, _ := detail["from"].(string)
		to, _ := detail["to"].(string)
		switch {
		case from != "" && to != "":
			n.Body = fmt.Sprintf("%s paused: %s's role changed from %s to %s, so issues can't send on their authority. Anyone with campaigns:send can turn it back on and becomes its owner.", d.Name, owner, from, to)
		default:
			n.Body = fmt.Sprintf("%s paused: %s no longer has campaigns:send in this workspace. Anyone with campaigns:send can turn it back on and becomes its owner.", d.Name, owner)
		}
		n.Summary = fmt.Sprintf("%s paused: its owner can't send any more", d.Name)
	case "feed":
		n.Severity = "bad"
		n.Body = fmt.Sprintf("%s paused after three issues in a row were skipped because the feed failed. Check the feed, then turn it back on.", d.Name)
		n.Summary = fmt.Sprintf("%s paused after three feed failures", d.Name)
	default:
		who := "Someone"
		if by != nil {
			if u, err := q.GetUser(ctx, *by); err == nil {
				who = u.Name
			}
		}
		n.Body = fmt.Sprintf("%s paused %s. No new issues are prepared until it is turned back on.", who, d.Name)
		if note != "" {
			n.Body += fmt.Sprintf(" “%s”", note)
		}
		n.Summary = fmt.Sprintf("%s paused %s", who, d.Name)
	}
	return n
}

// reschedule moves the issue being collected after a schedule change.
func (s *Service) reschedule(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	d, err := s.Q.DigestByID(ctx, id)
	if err != nil {
		return err
	}
	issue, err := s.Q.DigestIssueOpen(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.ensureCollecting(ctx, ws, id)
	}
	if err != nil {
		return err
	}
	if issue.Status != "collecting" {
		return nil
	}
	sendAt, prepareAt, deadline := s.nextTimes(ctx, d, s.now())
	return s.Q.DigestIssueSetTimes(ctx, store.DigestIssueSetTimesParams{ID: issue.ID, SendAt: sendAt, PrepareAt: prepareAt, Deadline: deadline, Mode: d.Mode, Now: s.now()})
}
