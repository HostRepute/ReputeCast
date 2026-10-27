package engine

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// The incident panel's sentences (causes, recommendations, options, notes)
// are kept as messages with their values beside the English the engine
// stores, so the live monitor writes them in each reader's language and
// region. Their values are formatted here rather than by the message: a
// value's name says how ("…Rate" a rate per second with at most one
// decimal, "…Pct" a percentage with two decimals below 1% and one above,
// "…Day" a day without its year, "mailbox" a mailbox provider's key (it
// reads as the provider's name: "Outlook", "Other"), any
// other number a count), which writes the English exactly as the engine
// always has.

// detailTexts are the messages of an incident detail's sentences, by the
// field whose English each one wrote ("i18n" in campaign_incidents.detail).
type detailTexts struct {
	Note            i18n.Text     `json:"note,omitzero"`
	Causes          []causeTexts  `json:"causes,omitempty"`
	Recommendations []i18n.Text   `json:"recommendations,omitempty"`
	Options         []optionTexts `json:"options,omitempty"`
	RiskText        i18n.Text     `json:"riskText,omitzero"`
	WarmupNote      i18n.Text     `json:"warmupNote,omitzero"`
	MoveChecks      []i18n.Text   `json:"moveChecks,omitempty"`
}

type causeTexts struct {
	Lead i18n.Text `json:"lead,omitzero"`
	Body i18n.Text `json:"body,omitzero"`
}

type optionTexts struct {
	Title  i18n.Text `json:"title,omitzero"`
	Body   i18n.Text `json:"body,omitzero"`
	Action i18n.Text `json:"action,omitzero"`
}

// storedDetail is campaign_incidents.detail: the detail in English and the
// messages its sentences come from.
type storedDetail struct {
	IncidentDetail
	I18n *detailTexts `json:"i18n,omitempty"`
}

// stored is the detail as campaign_incidents.detail keeps it.
func (d IncidentDetail) stored() []byte {
	b, _ := json.Marshal(storedDetail{IncidentDetail: d, I18n: d.texts})
	return b
}

// parseDetail reads campaign_incidents.detail with its kept messages.
func parseDetail(raw []byte) IncidentDetail {
	var s storedDetail
	_ = jsonUnmarshal(raw, &s)
	d := s.IncidentDetail
	d.texts = s.I18n
	return d
}

func (d *IncidentDetail) kept() *detailTexts {
	if d.texts == nil {
		d.texts = &detailTexts{}
	}
	return d.texts
}

// setNote sets the note under "What tripped".
func (d *IncidentDetail) setNote(t i18n.Text) {
	d.Note = sentence(i18n.Default, t)
	d.kept().Note = t
}

// addCause adds a likely cause: a bold lead (or none) and the evidence.
func (d *IncidentDetail) addCause(lead, body i18n.Text) {
	d.Causes = append(d.Causes, IncidentCause{Lead: sentence(i18n.Default, lead), Body: sentence(i18n.Default, body)})
	k := d.kept()
	k.Causes = append(k.Causes, causeTexts{Lead: lead, Body: body})
}

// addRecommendation adds a recommendation.
func (d *IncidentDetail) addRecommendation(t i18n.Text) {
	d.Recommendations = append(d.Recommendations, sentence(i18n.Default, t))
	k := d.kept()
	k.Recommendations = append(k.Recommendations, t)
}

// option is one way to continue, as messages.
type option struct {
	key                 string
	recommended         bool
	title, body, action i18n.Text
}

// setOptions sets the ways to continue.
func (d *IncidentDetail) setOptions(opts []option) {
	k := d.kept()
	d.Options, k.Options = nil, nil
	for _, o := range opts {
		d.Options = append(d.Options, IncidentOption{Key: o.key, Title: sentence(i18n.Default, o.title), Body: sentence(i18n.Default, o.body),
			Recommended: o.recommended, Action: sentence(i18n.Default, o.action)})
		k.Options = append(k.Options, optionTexts{Title: o.title, Body: o.body, Action: o.action})
	}
}

// setWarmupNote sets the note under "If you move the share".
func (d *IncidentDetail) setWarmupNote(t i18n.Text) {
	d.WarmupNote = sentence(i18n.Default, t)
	d.kept().WarmupNote = t
}

// setMoveChecks sets the checks a move's target passed.
func (d *IncidentDetail) setMoveChecks(checks []moveCheck) {
	if d.Move == nil {
		return
	}
	k := d.kept()
	d.Move.Checks, k.MoveChecks = nil, nil
	for _, c := range checks {
		d.Move.Checks = append(d.Move.Checks, MoveCheck{Key: c.key, Result: c.result, Text: sentence(i18n.Default, c.text)})
		k.MoveChecks = append(k.MoveChecks, c.text)
	}
}

// setRisk sets the move's reputation risk.
func (d *IncidentDetail) setRisk(risk string, t i18n.Text) {
	if d.Move == nil {
		return
	}
	d.Move.Risk, d.Move.RiskText = risk, sentence(i18n.Default, t)
	d.kept().RiskText = t
}

// sentence writes a kept sentence in a language and region ("" for none,
// or for a message the catalog no longer has).
func sentence(l i18n.Lang, t i18n.Text) string {
	if t.IsZero() {
		return ""
	}
	m, err := catalog.Lookup(strings.TrimPrefix(t.ID, catalog.Namespace+"."))
	if err != nil {
		return ""
	}
	args := make(i18n.Args, len(t.Args)+1)
	for k, v := range t.Args {
		args[k] = value(l, k, v)
	}
	if k, ok := t.Args["mailbox"].(string); ok {
		args["who"] = mailboxIn(l, k)
	}
	return m.Text(l, args)
}

// value formats one of a sentence's values in a reader's region (see above).
func value(l i18n.Lang, name string, v any) any {
	r := l.Region
	switch x := v.(type) {
	case string:
		if name == "mailbox" && slices.Contains(routing.MailboxProviders, x) {
			// A mailbox provider's key reads as its name ("Outlook", "Other").
			return mailboxIn(l, x)
		}
		if strings.HasSuffix(name, "Day") {
			if t, err := time.Parse(time.RFC3339, x); err == nil {
				return r.DayMonth(t)
			}
		}
		return x
	case float64, int, int64, int32:
		f, _ := v.(float64)
		switch n := v.(type) {
		case int:
			f = float64(n)
		case int64:
			f = float64(n)
		case int32:
			f = float64(n)
		}
		switch {
		case strings.HasSuffix(name, "Rate"):
			// rateText: whole rates without decimals, others with one.
			if f == math.Trunc(f) {
				return r.Decimal(f, 0)
			}
			return r.Decimal(f, 1)
		case strings.HasSuffix(name, "Pct"):
			// fmtRate: two decimals below 1%, one above.
			if f < 1 {
				return r.Percent(f, 2)
			}
			return r.Percent(f, 1)
		}
		return r.Int(int64(math.Round(f)))
	}
	return v
}

// with is args with more values.
func with(args i18n.Args, kv ...any) i18n.Args {
	out := make(i18n.Args, len(args)+len(kv)/2)
	for k, v := range args {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

// fixed are the sentences an incident stores whole, by their English: an
// incident analysed before its sentences were kept still reads its note,
// error kind and health check lines in the reader's language.
var fixed = map[string]string{}

func init() {
	for _, k := range []string{"note.complaintOutlook", "note.complaintGmail", "note.complaintYahoo", "note.complaint", "note.bounce", "note.deferral",
		"probe.failing", "probe.answering"} {
		fixed[english(k, nil)] = k
	}
}

// fixedIn is a whole stored sentence in a reader's language.
func fixedIn(l i18n.Lang, s string) string {
	if k, ok := fixed[s]; ok {
		return text(l, k, nil)
	}
	return s
}

// errorKindIn names the errors of an outage in a reader's language.
func errorKindIn(l i18n.Lang, s string) string {
	kind := map[string]string{"rejected credentials": "auth", "connection errors": "connection", "throttling errors": "throttling", "API errors": "api"}[s]
	if kind == "" {
		return s
	}
	return text(l, "errorKind", i18n.Args{"kind": kind})
}

// In is the detail in a reader's language: kept sentences in their
// messages, the rest as stored.
func (d IncidentDetail) In(l i18n.Lang) IncidentDetail {
	if isEnglish(l) {
		return d
	}
	t := d.texts
	if t == nil {
		t = &detailTexts{}
	}
	in := func(english string, kept i18n.Text) string {
		if s := sentence(l, kept); s != "" {
			return s
		}
		return english
	}
	out := d
	out.Note = fixedIn(l, d.Note)
	if !t.Note.IsZero() {
		out.Note = in(d.Note, t.Note)
	}
	out.ErrorKind = errorKindIn(l, d.ErrorKind)
	out.Continuing = mailboxNamesIn(l, d.Continuing)
	if d.Log != nil {
		out.Log = slices.Clone(d.Log)
		for i := range out.Log {
			out.Log[i].Text = fixedIn(l, out.Log[i].Text)
		}
	}
	if d.Causes != nil {
		out.Causes = slices.Clone(d.Causes)
		for i := range out.Causes {
			if i < len(t.Causes) {
				out.Causes[i].Lead = in(out.Causes[i].Lead, t.Causes[i].Lead)
				out.Causes[i].Body = in(out.Causes[i].Body, t.Causes[i].Body)
			}
		}
	}
	if d.Recommendations != nil {
		out.Recommendations = slices.Clone(d.Recommendations)
		for i := range out.Recommendations {
			if i < len(t.Recommendations) {
				out.Recommendations[i] = in(out.Recommendations[i], t.Recommendations[i])
			}
		}
	}
	if d.Options != nil {
		out.Options = slices.Clone(d.Options)
		for i := range out.Options {
			if i < len(t.Options) {
				o := &out.Options[i]
				o.Title, o.Body, o.Action = in(o.Title, t.Options[i].Title), in(o.Body, t.Options[i].Body), in(o.Action, t.Options[i].Action)
			}
		}
	}
	if d.Move != nil {
		m := *d.Move
		m.RiskText = in(m.RiskText, t.RiskText)
		if d.Move.Checks != nil {
			m.Checks = slices.Clone(d.Move.Checks)
			for i := range m.Checks {
				if i < len(t.MoveChecks) {
					m.Checks[i].Text = in(m.Checks[i].Text, t.MoveChecks[i])
				}
			}
		}
		out.Move = &m
	}
	out.WarmupNote = in(d.WarmupNote, t.WarmupNote)
	return out
}

// mailboxNamesIn writes mailbox provider names the engine stored in
// English ("Other" is a word) in a reader's language.
func mailboxNamesIn(l i18n.Lang, names []string) []string {
	if names == nil {
		return nil
	}
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n
		if n == mailboxName("other") {
			out[i] = mailboxIn(l, "other")
		}
	}
	return out
}

// liveText is what a snapshot needs to be written in another language.
type liveText struct {
	via         []viaPart
	audience    audienceParts
	shared      []string // SharedStreams' streams
	failovers   []string // Failovers' streams
	held        []string // MailboxesHeld's keys
	continuing  []string // Continuing's keys
	notReported []string // Totals.NotReported's keys
}

// viaPart is one way a send's events arrive: SNS, the bounce mailbox or a
// provider's webhooks.
type viaPart struct {
	kind, name string
}

func (v viaPart) in(l i18n.Lang) string {
	switch v.kind {
	case "sns":
		return "SNS"
	case "mailbox":
		return text(l, "via.mailbox", nil)
	}
	return text(l, "via.webhooks", i18n.Args{"provider": v.name})
}

// audienceParts are the Send details' audience: every public list ("All
// subscribers") or the lists' names, then the segments' names.
type audienceParts struct {
	all   bool
	names []string
}

func (a audienceParts) in(l i18n.Lang) string {
	parts := slices.Clone(a.names)
	if a.all {
		parts = append([]string{text(l, "audience.all", nil)}, parts...)
	}
	return strings.Join(parts, " + ")
}

// eventParts are what a feed line's words are made of.
type eventParts struct {
	kind                         string // the line's words: delivered, hardBounce, …
	short, code, diag, url, vary string
	suppresses                   bool
}

// detail is a feed line's detail in a language.
func (p eventParts) detail(l i18n.Lang) string {
	join := func(parts ...string) string {
		out := []string{}
		for _, s := range parts {
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return strings.Join(out, " · ")
	}
	codeDiag := strings.TrimSpace(p.code + " " + p.diag)
	switch p.kind {
	case "delivered":
		return join(p.short, p.code)
	case "hardBounce":
		d := join(p.short, codeDiag)
		if p.suppresses {
			d = join(d, text(l, "event.suppressed", nil))
		}
		return d
	case "softBounce", "rejected":
		return join(p.short, codeDiag)
	case "deferred":
		return join(p.short, p.code, p.diag)
	case "complaint":
		return join(p.short, text(l, "event.feedbackLoop", nil), text(l, "event.suppressed", nil))
	case "openedMachine":
		return tracking.DiagnosticIn(l, p.diag)
	case "openedHuman":
		d := tracking.DiagnosticIn(l, p.diag)
		if p.vary != "" {
			d = join(d, text(l, "event.variant", i18n.Args{"variant": p.vary}))
		}
		return d
	case "clickedScanner":
		return join(linkText(p.url), tracking.DiagnosticIn(l, p.diag))
	case "clicked":
		return linkText(p.url)
	}
	return p.short
}

// eventWords are a feed line's label and short label by its kind.
var eventWords = map[string][2]string{
	"delivered":      {"delivered", "delivered"},
	"hardBounce":     {"hard bounce", "bounce"},
	"softBounce":     {"soft bounce", "bounce"},
	"deferred":       {"deferred", "deferred"},
	"complaint":      {"complaint", "complaint"},
	"rejected":       {"rejected", "rejected"},
	"openedMachine":  {"opened · machine", "open"},
	"openedHuman":    {"opened · human", "open"},
	"clickedScanner": {"clicked · scanner", "scanner"},
	"clicked":        {"clicked", "click"},
	"unsubscribed":   {"unsubscribed", "unsubscribed"},
}

// In is the snapshot in a reader's language: what the engine writes in
// words, with names and the providers' own words as they are. English
// readers read it as built.
func (s LiveSnapshot) In(l i18n.Lang) LiveSnapshot {
	if isEnglish(l) {
		return s
	}
	out := s
	out.StartedBy, out.PausedBy, out.ScheduledBy = systemIn(l, s.StartedBy), systemIn(l, s.PausedBy), systemIn(l, s.ScheduledBy)
	out.Details.ScheduledBy = systemIn(l, s.Details.ScheduledBy)
	out.Error = sendErrorIn(l, s.Error)
	out.Mailboxes = slices.Clone(s.Mailboxes)
	for i := range out.Mailboxes {
		out.Mailboxes[i].Name = mailboxIn(l, out.Mailboxes[i].Key)
	}
	out.DeliveredMix = slices.Clone(s.DeliveredMix)
	for i := range out.DeliveredMix {
		out.DeliveredMix[i].Name = mailboxIn(l, out.DeliveredMix[i].Key)
	}
	routes := make([]string, 0, len(s.Routes))
	for _, r := range s.Routes {
		routes = append(routes, r.Short+" "+l.Region.Percent(r.Percent, 0))
	}
	if len(routes) > 0 {
		out.Details.Route = strings.Join(routes, " · ")
	}
	if t := s.text; t != nil {
		out.MailboxesHeld, out.Continuing = keysIn(l, t.held), keysIn(l, t.continuing)
		out.Totals.NotReported = keysIn(l, t.notReported)
		out.SharedStreams = make([]string, len(t.shared))
		for i, st := range t.shared {
			out.SharedStreams[i] = streamIn(l, st)
		}
		out.Failovers = slices.Clone(s.Failovers)
		for i := range out.Failovers {
			if i < len(t.failovers) {
				out.Failovers[i].Stream = streamIn(l, t.failovers[i])
			}
		}
		via := make([]string, len(t.via))
		for i, v := range t.via {
			via[i] = v.in(l)
		}
		out.EventsVia = i18n.AndList(l.Locale, via)
		out.Details.Audience = t.audience.in(l)
	}
	out.Events = slices.Clone(s.Events)
	for i := range out.Events {
		ev := &out.Events[i]
		if ev.parts == nil {
			continue
		}
		if w, ok := eventWords[ev.parts.kind]; ok {
			ev.Label = text(l, "event.label", i18n.Args{"kind": ev.parts.kind, "raw": w[0]})
			ev.Short = text(l, "event.short", i18n.Args{"kind": ev.parts.kind, "raw": w[1]})
		}
		ev.Detail = ev.parts.detail(l)
	}
	out.Incidents = slices.Clone(s.Incidents)
	for i := range out.Incidents {
		out.Incidents[i] = out.Incidents[i].In(l)
	}
	if s.Incident != nil {
		c := s.Incident.In(l)
		out.Incident = &c
	}
	return out
}

// In is an incident in a reader's language.
func (in LiveIncident) In(l i18n.Lang) LiveIncident {
	if in.Mailbox != "" {
		in.MailboxName = mailboxIn(l, in.Mailbox)
	}
	in.Detail = in.Detail.In(l)
	return in
}

func keysIn(l i18n.Lang, keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = mailboxIn(l, k)
	}
	return out
}

// sendErrorIn is why a send failed, stored in English, in a reader's
// language; anything else (a render error's own words) stays as stored.
func sendErrorIn(l i18n.Lang, s string) string {
	if s == "" || isEnglish(l) {
		return s
	}
	for _, k := range []string{"fail.noAudience", "fail.noSender", "fail.noRoute", "fail.routeGone"} {
		if s == english(k, nil) {
			return text(l, k, nil)
		}
	}
	for _, k := range []string{"fail.render", "fail.conditional"} {
		prefix, _, _ := strings.Cut(english(k, i18n.Args{"error": "\x00"}), "\x00")
		if rest, ok := strings.CutPrefix(s, prefix); ok {
			return text(l, k, i18n.Args{"error": rest})
		}
	}
	return s
}
