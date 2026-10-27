package approvals

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// Snapshot is what an approval covers (approval_requests.snapshot): the
// content and subject, the audience definition and count, the route and
// the schedule, with the readiness result and estimates it was given on.
type Snapshot struct {
	Version     int    `json:"version"`
	ContentHash string `json:"content_hash"`
	Subject     string `json:"subject"`
	Preheader   string `json:"preheader"`
	SenderID    string `json:"sender_id,omitempty"`
	From        string `json:"from"`
	// Links lists the email's links in order; Sections its section outline.
	Links    []string `json:"links"`
	Sections []string `json:"sections"`

	Audience       json.RawMessage `json:"audience"`
	AudienceLabels []string        `json:"audience_labels"`
	AudienceCount  int64           `json:"audience_count"`
	// AudienceBaseline is the audience counted the way revalidation counts
	// it, at request time: the audience "moves" against this number.
	AudienceBaseline int64 `json:"audience_baseline,omitempty"`
	Suppressed       int64 `json:"suppressed"`
	Role             int64 `json:"role"`
	Bounced          int64 `json:"bounced"`

	Route      []SnapshotShare `json:"route"`
	RouteKey   string          `json:"route_key"`
	SplitLabel string          `json:"split_label"`
	Minutes    int             `json:"minutes"`
	Cost       float64         `json:"cost"`

	Schedule   json.RawMessage               `json:"schedule"`
	Thresholds readiness.ReadinessThresholds `json:"thresholds"`

	Readiness SnapshotReadiness   `json:"readiness"`
	Rules     []ApprovalRuleMatch `json:"rules"`
	// RuleTexts keeps the matched rules' messages for later readers, by
	// rule key (filled from Rules when the snapshot is stored).
	RuleTexts map[string]ruleTexts `json:"rule_texts,omitempty"`
	// AuthorRole is the requester's role ("Content Writer").
	AuthorRole string `json:"author_role"`

	Automation *ApprovalAutomationSnapshot `json:"automation,omitempty"`
	// AutomationItems keep Automation.Changes' kinds and titles as messages
	// for later readers (requests made before have none).
	AutomationItems []automationItem `json:"automation_items,omitempty"`
	// Languages are the language versions the approval covers.
	Languages []languages.VersionSnapshot `json:"languages,omitempty"`
	// DraftHash and Transactional: what a transactional template publish
	// request covers (one draft of the template).
	DraftHash     string                         `json:"draft_hash,omitempty"`
	Transactional *ApprovalTransactionalSnapshot `json:"transactional,omitempty"`
	// Website: what a post publish request covers (DraftHash is the post's hash).
	Website *ApprovalWebsiteSnapshot `json:"website,omitempty"`

	// Channel campaigns (SMS; push and WhatsApp later) cover their text and
	// a schedule written for people, compared as they are.
	Channel       string `json:"channel,omitempty"`
	Text          string `json:"text,omitempty"`
	ScheduleLabel string `json:"schedule_label,omitempty"`
	// ScheduleText is the label's message, read in each reader's language;
	// requests compare the English label.
	ScheduleText i18n.Text `json:"schedule_text,omitzero"`
	// SMS is what an SMS campaign's approval shows: the text as sent, its
	// segments, the senders and the cost per country.
	SMS *ApprovalSMSSnapshot `json:"sms,omitempty"`
	// Page: what a page publish request covers (DraftHash is the page's hash).
	Page *ApprovalPageSnapshot `json:"page,omitempty"`
	// Push and WhatsApp are what push and WhatsApp campaigns' approvals
	// show: the notification, or the template with its variables (X22).
	Push     *ApprovalPushSnapshot     `json:"push,omitempty"`
	WhatsApp *ApprovalWhatsAppSnapshot `json:"whatsapp,omitempty"`
}

// ApprovalPushSnapshot is a push campaign at request time.
type ApprovalPushSnapshot struct {
	Title      string               `json:"title"`
	Message    string               `json:"message"`
	Link       string               `json:"link"`
	Image      string               `json:"image,omitempty"`
	Icon       string               `json:"icon,omitempty"`
	Host       string               `json:"host,omitempty"`
	Buttons    []ApprovalPushButton `json:"buttons"`
	TTLMinutes int                  `json:"ttlMinutes"`
	Tag        string               `json:"tag,omitempty"`
	Sites      []string             `json:"sites"`
	Recipients int64                `json:"recipients"`
	Schedule   string               `json:"schedule"`
}

// ApprovalPushButton is a notification's button.
type ApprovalPushButton struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// ApprovalWhatsAppSnapshot is a WhatsApp campaign at request time.
type ApprovalWhatsAppSnapshot struct {
	Template   string                  `json:"template"`
	Language   string                  `json:"language"`
	Category   string                  `json:"category"`
	Body       string                  `json:"body"`
	Preview    string                  `json:"preview"`
	Buttons    []string                `json:"buttons"`
	Variables  []ApprovalWhatsAppValue `json:"variables"`
	Business   string                  `json:"business"`
	Number     string                  `json:"number"`
	Recipients int64                   `json:"recipients"`
	Cost       float64                 `json:"cost"`
	Schedule   string                  `json:"schedule"`
}

// ApprovalWhatsAppValue is a template variable and what fills it.
type ApprovalWhatsAppValue struct {
	Key      string `json:"key"`
	Source   string `json:"source"`
	Fallback string `json:"fallback,omitempty"`
}

// ApprovalSMSSnapshot is an SMS campaign at request time.
type ApprovalSMSSnapshot struct {
	Text       string               `json:"text"`
	Prefix     string               `json:"prefix"`
	OptOut     string               `json:"optOut"`
	Body       string               `json:"body"`
	Chars      int                  `json:"chars"`
	Segments   int                  `json:"segments"`
	Encoding   string               `json:"encoding"`
	Recipients int64                `json:"recipients"`
	Cost       float64              `json:"cost"`
	Schedule   string               `json:"schedule"`
	Countries  []ApprovalSMSCountry `json:"countries"`
}

// ApprovalSMSCountry is one country of an SMS campaign's route.
type ApprovalSMSCountry struct {
	Country    string  `json:"country"`
	Sender     string  `json:"sender"`
	Provider   string  `json:"provider"`
	Recipients int64   `json:"recipients"`
	Price      float64 `json:"price"`
	Cost       float64 `json:"cost"`
	Skipped    bool    `json:"skipped"`
}

// SnapshotShare is one provider of the route at request time.
type SnapshotShare struct {
	ProviderID  string  `json:"provider_id"`
	Name        string  `json:"name"`
	Short       string  `json:"short"`
	Label       string  `json:"label,omitempty"`
	Adapter     string  `json:"adapter"`
	Percent     float64 `json:"percent"`
	Recipients  int64   `json:"recipients"`
	Cost        float64 `json:"cost"`
	PriceLabel  string  `json:"price_label"`
	Status      string  `json:"status"`
	StatusLabel string  `json:"status_label"`
	WarmupDay   int     `json:"warmup_day,omitempty"`
	WarmupDays  int     `json:"warmup_days,omitempty"`
	// StatusText is the status label's message, kept so the request reads
	// it in each reader's language.
	StatusText readiness.Text `json:"status_text,omitempty"`
}

// SnapshotReadiness is the Send Readiness result an approval was given on.
type SnapshotReadiness struct {
	RunID    string                     `json:"run_id"`
	Verdict  string                     `json:"verdict"`
	Passed   int                        `json:"passed"`
	Warnings int                        `json:"warnings"`
	Blocks   int                        `json:"blocks"`
	Total    int                        `json:"total"`
	RanAt    time.Time                  `json:"ran_at"`
	Issues   []readiness.ReadinessCheck `json:"issues"`
}

// ApprovalAutomationSnapshot is what an automation publish request covers.
type ApprovalAutomationSnapshot struct {
	Version   int      `json:"version"`
	Migration string   `json:"migration"`
	Changes   []string `json:"changes"`
}

// snapshotVersion is the version of a language an approval covers (0 when
// it covers none: the current one).
func snapshotVersion(s Snapshot, lang string) int {
	for _, v := range s.Languages {
		if v.Language == lang {
			return v.Version
		}
	}
	return 0
}

// Baseline is the count audience drift is measured against.
func (s Snapshot) Baseline() int64 {
	if s.AudienceBaseline > 0 {
		return s.AudienceBaseline
	}
	return s.AudienceCount
}

// MarshalJSON stores the snapshot with its rules' messages (RuleTexts).
func (s Snapshot) MarshalJSON() ([]byte, error) {
	type plain Snapshot
	texts := map[string]ruleTexts{}
	for k, v := range s.RuleTexts {
		texts[k] = v
	}
	for _, r := range s.Rules {
		if r.texts != nil {
			texts[r.Key] = *r.texts
		}
	}
	s.RuleTexts = nil
	if len(texts) > 0 {
		s.RuleTexts = texts
	}
	return json.Marshal(plain(s))
}

// ParseSnapshot reads a stored snapshot. Its rules carry their kept
// messages, so they read in each reader's language (localRules).
func ParseSnapshot(raw json.RawMessage) Snapshot {
	var s Snapshot
	_ = json.Unmarshal(raw, &s)
	s.Rules = withRuleTexts(s.Rules, s.RuleTexts)
	if s.Links == nil {
		s.Links = []string{}
	}
	if s.Sections == nil {
		s.Sections = []string{}
	}
	if s.Route == nil {
		s.Route = []SnapshotShare{}
	}
	if s.Rules == nil {
		s.Rules = []ApprovalRuleMatch{}
	}
	if s.AudienceLabels == nil {
		s.AudienceLabels = []string{}
	}
	if s.Readiness.Issues == nil {
		s.Readiness.Issues = []readiness.ReadinessCheck{}
	}
	return s
}

// Current is the campaign as it stands, in snapshot terms.
type Current struct {
	Version       int
	ContentHash   string
	Subject       string
	Preheader     string
	SenderID      string
	From          string
	MJML          string
	HTML          string
	Audience      json.RawMessage
	AudienceNames []string
	AudienceCount int64
	RouteKey      string
	SplitLabel    string
	Schedule      json.RawMessage
	Timezone      string
	UpdatedBy     string
	Languages     []languages.VersionSnapshot
	// Text, ScheduleLabel and ScheduleText are a channel campaign's (see Snapshot).
	Text          string
	ScheduleLabel string
	ScheduleText  i18n.Text
}

var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)

// Links lists the links of an email in order (MJML when there is one).
func Links(mjmlSrc, html string) []string {
	src := mjmlSrc
	if strings.TrimSpace(src) == "" {
		src = html
	}
	out := []string{}
	for _, m := range hrefRe.FindAllStringSubmatch(src, -1) {
		h := strings.TrimSpace(m[1])
		if h == "" || strings.HasPrefix(h, "#") || strings.HasPrefix(h, "mailto:") || strings.Contains(h, "{{") {
			continue
		}
		out = append(out, h)
	}
	return out
}

// Sections is the section outline of the MJML ("header", "hero", "footer").
func Sections(mjmlSrc string) []string {
	out := []string{}
	secs, err := mjml.Outline(mjmlSrc)
	if err != nil {
		return out
	}
	for _, s := range secs {
		out = append(out, s.Label)
	}
	return out
}

// RouteKey is "ses:90,postal:10" for the route of a run.
func RouteKey(shares []readiness.ReadinessRouteShare) string {
	parts := []string{}
	for _, s := range shares {
		parts = append(parts, fmt.Sprintf("%s:%s", s.ProviderID, trim(math.Round(s.Percent))))
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func trim(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	return strings.TrimSuffix(s, ".0")
}

// ApprovalChange is one difference between an approval's snapshot and the
// campaign.
type ApprovalChange struct {
	Field  string `json:"field" enum:"subject,preheader,link,content,sender,audience,audience_count,route,schedule,language"`
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	// What says how a language version changed ("intro changed").
	What string `json:"what,omitempty"`
}

// storedChange is a change as approval_requests.changes keeps it: its
// English, and what it keeps for readers in other languages. Changes
// stored before have no Kept and read as stored.
type storedChange struct {
	ApprovalChange
	Kept *keptChange `json:"kept,omitempty"`
}

// keptChange is a change's label and values as messages (a subject, a link
// or names stay as written: they have no message), and what it adds to the
// invalidation reason: a field key ("subject", "link", "language:German")
// and, for an audience that moved, how far (in percent).
type keptChange struct {
	// Language is a language version's change as data, written in each
	// reader's language (languages.VersionChangeDetail.Texts).
	Language *languages.VersionChangeDetail `json:"language,omitempty"`
	Label    i18n.Text                      `json:"label,omitzero"`
	Before   i18n.Text                      `json:"before,omitzero"`
	After    i18n.Text                      `json:"after,omitzero"`
	Field    string                         `json:"field,omitempty"`
	Moved    float64                        `json:"moved,omitempty"`
}

// Diff compares a snapshot with the campaign as it stands. loc formats the
// schedule in the workspace zone.
func Diff(s Snapshot, c Current, loc *time.Location) []ApprovalChange {
	return plainChanges(diff(s, c, loc))
}

func plainChanges(stored []storedChange) []ApprovalChange {
	out := make([]ApprovalChange, len(stored))
	for i, c := range stored {
		out[i] = c.ApprovalChange
	}
	return out
}

// localChanges writes changes for a reader: kept messages in the reader's
// language, everything else as stored.
func localChanges(l i18n.Lang, stored []storedChange) []ApprovalChange {
	out := make([]ApprovalChange, len(stored))
	for i, c := range stored {
		ch := c.ApprovalChange
		if k := c.Kept; k != nil {
			ch.Label = kept(l, ch.Label, k.Label)
			ch.Before = kept(l, ch.Before, k.Before)
			ch.After = kept(l, ch.After, k.After)
			if d := k.Language; d != nil && l.Locale != i18n.English {
				ch.Label, ch.Before, ch.After, ch.What = d.Texts(l)
			}
		}
		out[i] = ch
	}
	return out
}

// changesFor reads stored changes for a reader.
func changesFor(l i18n.Lang, raw json.RawMessage) []ApprovalChange {
	var stored []storedChange
	_ = json.Unmarshal(raw, &stored)
	return localChanges(l, stored)
}

func diff(s Snapshot, c Current, loc *time.Location) []storedChange {
	out := []storedChange{}
	add := func(ch ApprovalChange, k keptChange) {
		if k.Field == "" {
			k.Field = ch.Field
		}
		out = append(out, storedChange{ApprovalChange: ch, Kept: &k})
	}
	if s.Subject != c.Subject {
		add(ApprovalChange{Field: "subject", Label: "Subject", Before: s.Subject, After: c.Subject}, keptChange{Label: mChangeSubject.Keep(nil)})
	}
	if s.Preheader != c.Preheader {
		add(ApprovalChange{Field: "preheader", Label: "Preheader", Before: s.Preheader, After: c.Preheader}, keptChange{Label: mChangePreheader.Keep(nil)})
	}
	if s.Text != "" || c.Text != "" {
		if s.Text != c.Text {
			add(ApprovalChange{Field: "content", Label: "Text", Before: s.Text, After: c.Text}, keptChange{Label: mChangeText.Keep(nil)})
		}
	} else if s.ContentHash != "" && s.ContentHash != c.ContentHash {
		links := Links(c.MJML, c.HTML)
		linkChanges := 0
		for i := 0; i < max(len(s.Links), len(links)); i++ {
			var before, after string
			if i < len(s.Links) {
				before = s.Links[i]
			}
			if i < len(links) {
				after = links[i]
			}
			if before != after {
				add(ApprovalChange{Field: "link", Label: fmt.Sprintf("Link %d", i+1), Before: before, After: after}, keptChange{Label: mChangeLink.Keep(i18n.Args{"n": i + 1})})
				linkChanges++
			}
		}
		secs := Sections(c.MJML)
		if !slices.Equal(secs, s.Sections) || (linkChanges == 0 && s.Subject == c.Subject && s.Preheader == c.Preheader) {
			after := fmt.Sprintf("v%d", c.Version)
			if changed := changedSections(s.Sections, secs); changed != "" {
				after += " · " + changed
			}
			add(ApprovalChange{Field: "content", Label: "Content", Before: fmt.Sprintf("v%d", s.Version), After: after},
				keptChange{Label: mChangeContent.Keep(nil), Before: mChangeVersion.Keep(i18n.Args{"version": s.Version}), After: contentKept(c.Version, s.Sections, secs)})
		}
	}
	for _, vc := range languages.DiffVersions(s.Languages, c.Languages) {
		d := vc.Detail
		add(ApprovalChange{Field: "language", Label: vc.Name, Before: vc.Before, After: vc.After, What: vc.What}, keptChange{Field: "language:" + vc.Name, Language: &d})
	}
	if s.SenderID != c.SenderID {
		add(ApprovalChange{Field: "sender", Label: "From", Before: s.From, After: c.From}, keptChange{Label: mChangeFrom.Keep(nil)})
	}
	if !sameAudience(s.Audience, c.Audience) {
		add(ApprovalChange{Field: "audience", Label: "Audience", Before: strings.Join(s.AudienceLabels, ", "), After: strings.Join(c.AudienceNames, ", ")},
			keptChange{Label: mChangeAudience.Keep(nil)})
	} else if base := s.Baseline(); base > 0 && c.AudienceCount > 0 {
		// (An empty audience can't be sent anyway: Send Readiness blocks it.)
		moved := float64(c.AudienceCount-base) / float64(base) * 100
		if math.Abs(moved) > InvalidationShare {
			now := s.AudienceCount + c.AudienceCount - base
			add(ApprovalChange{Field: "audience_count", Label: "Audience",
				Before: readiness.Count(s.AudienceCount), After: fmt.Sprintf("%s (%s)", readiness.Count(now), signedPct(i18n.Default, moved))},
				keptChange{Label: mChangeAudience.Keep(nil), Before: mChangeCount.Keep(i18n.Args{"count": s.AudienceCount}),
					After: mChangeCountMoved.Keep(i18n.Args{"count": now, "moved": moved}), Moved: moved})
		}
	}
	if s.RouteKey != "" && c.RouteKey != "" && s.RouteKey != c.RouteKey {
		add(ApprovalChange{Field: "route", Label: "Route", Before: s.SplitLabel, After: c.SplitLabel}, keptChange{Label: mChangeRoute.Keep(nil)})
	}
	if s.ScheduleLabel != "" || c.ScheduleLabel != "" {
		if s.ScheduleLabel != c.ScheduleLabel {
			add(ApprovalChange{Field: "schedule", Label: "Schedule", Before: s.ScheduleLabel, After: c.ScheduleLabel},
				keptChange{Label: mChangeSchedule.Keep(nil), Before: s.ScheduleText, After: c.ScheduleText})
		}
	} else if !sameSchedule(s.Schedule, c.Schedule) {
		before, after := scheduleKept(s.Schedule, loc), scheduleKept(c.Schedule, loc)
		add(ApprovalChange{Field: "schedule", Label: "Schedule", Before: before.text(i18n.Default), After: after.text(i18n.Default)},
			keptChange{Label: mChangeSchedule.Keep(nil), Before: before.Text, After: after.Text})
	}
	return out
}

func changedSections(before, after []string) string {
	removed, added := sectionsAddedRemoved(before, after)
	parts := []string{}
	if len(added) > 0 {
		parts = append(parts, "added "+strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		parts = append(parts, "removed "+strings.Join(removed, ", "))
	}
	if len(parts) == 0 {
		return "edited"
	}
	return strings.Join(parts, " · ")
}

// sectionsAddedRemoved are the sections only before has and only after has.
func sectionsAddedRemoved(before, after []string) (removed, added []string) {
	removed, added = []string{}, []string{}
	for _, b := range before {
		if !slices.Contains(after, b) {
			removed = append(removed, b)
		}
	}
	for _, a := range after {
		if !slices.Contains(before, a) {
			added = append(added, a)
		}
	}
	return removed, added
}

// contentKept is a content change's new version and what changed in it, as a
// message: "v10 · added hero · removed footer".
func contentKept(version int, before, after []string) i18n.Text {
	removed, added := sectionsAddedRemoved(before, after)
	args := i18n.Args{"version": version, "added": added, "removed": removed}
	switch {
	case len(added) > 0 && len(removed) > 0:
		return mChangeAddedRemoved.Keep(args)
	case len(added) > 0:
		return mChangeAdded.Keep(args)
	case len(removed) > 0:
		return mChangeRemoved.Keep(args)
	}
	return mChangeEdited.Keep(args)
}

func sameAudience(a, b json.RawMessage) bool {
	x, y := readiness.ParseAudience(a), readiness.ParseAudience(b)
	eq := func(p, q []string) bool {
		p, q = slices.Clone(p), slices.Clone(q)
		slices.Sort(p)
		slices.Sort(q)
		return slices.Equal(p, q)
	}
	return eq(x.Lists, y.Lists) && eq(x.Segments, y.Segments) && eq(x.ExcludeLists, y.ExcludeLists) && eq(x.ExcludeSegments, y.ExcludeSegments) &&
		eq(x.ExcludeTiers, y.ExcludeTiers) && x.IncludeSunset == y.IncludeSunset && sameChoice(x.IncludeUnengaged, y.IncludeUnengaged) && x.Paid == y.Paid
}

// sameChoice compares an optional choice (unset, yes or no).
func sameChoice(a, b *bool) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func sameSchedule(a, b json.RawMessage) bool {
	x, y := readiness.ParseSchedule(a), readiness.ParseSchedule(b)
	if x.Mode != y.Mode {
		return false
	}
	if x.Mode == "now" {
		return true
	}
	tx, ex := time.Parse(time.RFC3339, x.At)
	ty, ey := time.Parse(time.RFC3339, y.At)
	if ex != nil || ey != nil {
		return x.At == y.At
	}
	return tx.Equal(ty)
}

// ScheduleText is "Nov 4, 09:00", "Send now" or "Best time within 24 h of
// Nov 4, 09:00", in English (as stored and as API clients read it).
func ScheduleText(raw json.RawMessage, loc *time.Location) string {
	return scheduleKept(raw, loc).text(i18n.Default)
}

// scheduleTextIn is ScheduleText in a reader's language and region.
func scheduleTextIn(l i18n.Lang, raw json.RawMessage, loc *time.Location) string {
	return scheduleKept(raw, loc).text(l)
}

// scheduleKept is a schedule as a kept message: send now, or the moment (in
// the schedule's zone), at best time or not.
func scheduleKept(raw json.RawMessage, loc *time.Location) keptValue {
	s := readiness.ParseSchedule(raw)
	if s.Mode == "now" || s.At == "" {
		return keptValue{Text: mScheduleNow.Keep(nil)}
	}
	t, err := time.Parse(time.RFC3339, s.At)
	if err != nil {
		return keptValue{Raw: s.At}
	}
	if s.Timezone != "" {
		loc = state.Location(s.Timezone)
	}
	at := i18n.Args{"at": t.In(loc).Format(time.RFC3339)}
	if s.Mode == "best_time" {
		return keptValue{Text: mScheduleBest.Keep(at)}
	}
	return keptValue{Text: mScheduleAt.Keep(at)}
}

// InvalidationReason sums up the changes: "subject changed", "audience
// moved +14%". The screens add "after approval" when an approval was given.
func InvalidationReason(changes []ApprovalChange) string {
	stored := make([]storedChange, len(changes))
	for i, c := range changes {
		stored[i] = storedChange{ApprovalChange: c}
	}
	return render(i18n.Default, reasonOf(stored))
}

// reasonOf is why changes invalidate an approval, as a message.
func reasonOf(changes []storedChange) i18n.Text {
	fields := []string{}
	for _, c := range changes {
		switch c.Field {
		case "audience_count":
			moved := 0.0
			if c.Kept != nil {
				moved = c.Kept.Moved
			} else if i := strings.Index(c.After, "("); i >= 0 {
				// "54,620 (+14%)"
				p := strings.TrimSuffix(strings.TrimSuffix(c.After[i+1:], ")"), "%")
				p = strings.Replace(strings.TrimPrefix(p, "+"), "−", "-", 1)
				moved, _ = strconv.ParseFloat(p, 64)
			}
			return mInvalidMoved.Keep(i18n.Args{"moved": moved})
		case "link":
			if !slices.Contains(fields, "link") {
				fields = append(fields, "link")
			}
		case "subject", "preheader", "content", "audience", "route", "schedule":
			if !slices.Contains(fields, c.Field) {
				fields = append(fields, c.Field)
			}
		case "sender":
			fields = append(fields, "sender")
		case "language":
			// "German: intro changed".
			if c.Kept != nil && c.Kept.Language != nil {
				if len(changes) == 1 {
					return mInvalidLanguage.Keep(i18n.Args{"language": c.Label, "what": c.What, "change": c.Kept.Language})
				}
				fields = append(fields, "lang:"+c.Kept.Language.Language)
				continue
			}
			if len(changes) == 1 {
				return mInvalidLanguage.Keep(i18n.Args{"language": c.Label, "what": c.What})
			}
			fields = append(fields, "language:"+c.Label)
		}
	}
	if len(fields) == 0 {
		return mInvalidCampaign.Keep(nil)
	}
	return mInvalidFields.Keep(i18n.Args{"fields": fields})
}

// reasonFor is a request's invalidation reason for a reader: its kept
// message, else the English stored with it. Requests invalidated before
// said "after approval" in their reason; the screens add it now.
func reasonFor(l i18n.Lang, english string, text json.RawMessage) string {
	english = strings.TrimSuffix(english, " after approval")
	var t i18n.Text
	if len(text) > 0 {
		_ = json.Unmarshal(text, &t)
	}
	return kept(l, english, t)
}

// fieldsFor is what an invalidation reason says changed, for a screen that
// writes its own sentence around it ("Daniel edited the subject and a
// link"): empty when the reason isn't about edited fields.
func fieldsFor(l i18n.Lang, english string, text json.RawMessage) string {
	var t i18n.Text
	if len(text) > 0 {
		_ = json.Unmarshal(text, &t)
	}
	if !t.IsZero() {
		if t.ID == mInvalidFields.ID() {
			return fieldList(l, strs(t.Args["fields"]))
		}
		return ""
	}
	english = strings.TrimSuffix(english, " after approval")
	if f, ok := strings.CutSuffix(english, " changed"); ok && english != "the campaign changed" && !strings.Contains(f, ": ") {
		return f
	}
	return ""
}

var (
	mChangeSubject      = msgs.M("change.subject")
	mChangePreheader    = msgs.M("change.preheader")
	mChangeText         = msgs.M("change.text")
	mChangeLink         = msgs.M("change.link")
	mChangeContent      = msgs.M("change.content")
	mChangeFrom         = msgs.M("change.from")
	mChangeAudience     = msgs.M("change.audience")
	mChangeRoute        = msgs.M("change.route")
	mChangeSchedule     = msgs.M("change.schedule")
	mChangeCount        = msgs.M("change.count")
	mChangeCountMoved   = msgs.M("change.countMoved")
	mChangeVersion      = msgs.M("change.version")
	mChangeEdited       = msgs.M("change.edited")
	mChangeAdded        = msgs.M("change.added")
	mChangeRemoved      = msgs.M("change.removed")
	mChangeAddedRemoved = msgs.M("change.addedRemoved")
	mInvalidFields      = msgs.M("invalid.fields")
	mInvalidMoved       = msgs.M("invalid.moved")
	mInvalidLanguage    = msgs.M("invalid.language")
	mInvalidCampaign    = msgs.M("invalid.campaign")
	mInvalidAfter       = msgs.M("invalid.afterApproval")
	mInvalidPage        = msgs.M("invalid.page")
	mInvalidPost        = msgs.M("invalid.post")
)
