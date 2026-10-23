package approvals

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

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
	// AuthorRole is the requester's role ("Content Writer").
	AuthorRole string `json:"author_role"`

	Automation *ApprovalAutomationSnapshot `json:"automation,omitempty"`
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

// ParseSnapshot reads a stored snapshot.
func ParseSnapshot(raw json.RawMessage) Snapshot {
	var s Snapshot
	_ = json.Unmarshal(raw, &s)
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
	// Text and ScheduleLabel are a channel campaign's (see Snapshot).
	Text          string
	ScheduleLabel string
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

// Diff compares a snapshot with the campaign as it stands. loc formats the
// schedule in the workspace zone.
func Diff(s Snapshot, c Current, loc *time.Location) []ApprovalChange {
	out := []ApprovalChange{}
	if s.Subject != c.Subject {
		out = append(out, ApprovalChange{Field: "subject", Label: "Subject", Before: s.Subject, After: c.Subject})
	}
	if s.Preheader != c.Preheader {
		out = append(out, ApprovalChange{Field: "preheader", Label: "Preheader", Before: s.Preheader, After: c.Preheader})
	}
	if s.Text != "" || c.Text != "" {
		if s.Text != c.Text {
			out = append(out, ApprovalChange{Field: "content", Label: "Text", Before: s.Text, After: c.Text})
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
				out = append(out, ApprovalChange{Field: "link", Label: fmt.Sprintf("Link %d", i+1), Before: before, After: after})
				linkChanges++
			}
		}
		secs := Sections(c.MJML)
		if !slices.Equal(secs, s.Sections) || (linkChanges == 0 && s.Subject == c.Subject && s.Preheader == c.Preheader) {
			after := fmt.Sprintf("v%d", c.Version)
			if changed := changedSections(s.Sections, secs); changed != "" {
				after += " · " + changed
			}
			out = append(out, ApprovalChange{Field: "content", Label: "Content", Before: fmt.Sprintf("v%d", s.Version), After: after})
		}
	}
	for _, vc := range languages.DiffVersions(s.Languages, c.Languages) {
		out = append(out, ApprovalChange{Field: "language", Label: vc.Name, Before: vc.Before, After: vc.After, What: vc.What})
	}
	if s.SenderID != c.SenderID {
		out = append(out, ApprovalChange{Field: "sender", Label: "From", Before: s.From, After: c.From})
	}
	if !sameAudience(s.Audience, c.Audience) {
		out = append(out, ApprovalChange{Field: "audience", Label: "Audience", Before: strings.Join(s.AudienceLabels, ", "), After: strings.Join(c.AudienceNames, ", ")})
	} else if base := s.Baseline(); base > 0 && c.AudienceCount > 0 {
		// (An empty audience can't be sent anyway: Send Readiness blocks it.)
		moved := float64(c.AudienceCount-base) / float64(base) * 100
		if math.Abs(moved) > InvalidationShare {
			out = append(out, ApprovalChange{Field: "audience_count", Label: "Audience",
				Before: readiness.Count(s.AudienceCount), After: fmt.Sprintf("%s (%s)", readiness.Count(s.AudienceCount+c.AudienceCount-base), signedPct(moved))})
		}
	}
	if s.RouteKey != "" && c.RouteKey != "" && s.RouteKey != c.RouteKey {
		out = append(out, ApprovalChange{Field: "route", Label: "Route", Before: s.SplitLabel, After: c.SplitLabel})
	}
	if s.ScheduleLabel != "" || c.ScheduleLabel != "" {
		if s.ScheduleLabel != c.ScheduleLabel {
			out = append(out, ApprovalChange{Field: "schedule", Label: "Schedule", Before: s.ScheduleLabel, After: c.ScheduleLabel})
		}
	} else if !sameSchedule(s.Schedule, c.Schedule) {
		out = append(out, ApprovalChange{Field: "schedule", Label: "Schedule", Before: ScheduleText(s.Schedule, loc), After: ScheduleText(c.Schedule, loc)})
	}
	return out
}

func changedSections(before, after []string) string {
	removed, added := []string{}, []string{}
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

func signedPct(v float64) string {
	s := fmt.Sprintf("%.0f%%", math.Abs(v))
	if math.Abs(v) < 10 {
		s = fmt.Sprintf("%.1f%%", math.Abs(v))
	}
	if v >= 0 {
		return "+" + s
	}
	return "−" + s
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
		eq(x.ExcludeTiers, y.ExcludeTiers) && x.IncludeSunset == y.IncludeSunset && x.Paid == y.Paid
}

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

// ScheduleText is "Nov 4, 09:00", "Send now" or "Best time within 24 h of Nov 4, 09:00".
func ScheduleText(raw json.RawMessage, loc *time.Location) string {
	s := readiness.ParseSchedule(raw)
	if s.Mode == "now" || s.At == "" {
		return "Send now"
	}
	t, err := time.Parse(time.RFC3339, s.At)
	if err != nil {
		return s.At
	}
	if s.Timezone != "" {
		loc = state.Location(s.Timezone)
	}
	text := t.In(loc).Format("Jan 2, 15:04")
	if s.Mode == "best_time" {
		return "Best time within 24 h of " + text
	}
	return text
}

// InvalidationReason sums up the changes: "subject changed", "audience
// moved +14% after approval".
func InvalidationReason(changes []ApprovalChange) string {
	fields := []string{}
	for _, c := range changes {
		switch c.Field {
		case "audience_count":
			after := c.After
			if i := strings.Index(after, "("); i >= 0 {
				after = strings.TrimSuffix(after[i+1:], ")")
			}
			return "audience moved " + after + " after approval"
		case "link":
			if !slices.Contains(fields, "a link") {
				fields = append(fields, "a link")
			}
		case "subject", "preheader", "content", "audience", "route", "schedule":
			if !slices.Contains(fields, c.Field) {
				fields = append(fields, c.Field)
			}
		case "sender":
			fields = append(fields, "sender")
		case "language":
			// "German: intro changed after approval".
			if len(changes) == 1 {
				return c.Label + ": " + c.What + " after approval"
			}
			fields = append(fields, c.Label+" version")
		}
	}
	if len(fields) == 0 {
		return "the campaign changed"
	}
	verb := "changed"
	return readiness.JoinAnd(fields) + " " + verb
}
