package readiness

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// A run keeps its texts' messages in "i18n" members next to the English it
// stores: in readiness_runs (checks, groups, the problem and the
// estimates), readiness_progress, and wherever a check is copied (an
// approval's snapshot, a held send). They aren't part of the API: In writes
// the texts in a reader's language and drops them, and WithoutTexts drops
// them from stored JSON handed out as it is (the MCP server's).

// checkTexts are the messages of a check's texts.
type checkTexts struct {
	Title     Text        `json:"title,omitempty"`
	FlatTitle Text        `json:"flatTitle,omitempty"`
	Detail    Text        `json:"detail,omitempty"`
	Label     Text        `json:"label,omitempty"`
	Evidence  Text        `json:"evidence,omitempty"`
	Short     Text        `json:"short,omitempty"`
	Ack       Text        `json:"ack,omitempty"`
	Note      Text        `json:"note,omitempty"`
	Fixes     []Text      `json:"fixes,omitempty"`
	Rows      []rowTexts  `json:"rows,omitempty"`
	Technical []techTexts `json:"technical,omitempty"`
}

// techTexts are the messages of a line of a check's technical details.
type techTexts struct {
	Label Text `json:"label,omitempty"`
	Value Text `json:"value,omitempty"`
}

// rowTexts are the messages of a row of a check's table.
type rowTexts struct {
	Label    Text `json:"label,omitempty"`
	Detail   Text `json:"detail,omitempty"`
	GetsName Text `json:"getsName,omitempty"`
}

// groupTexts are the messages of a group's texts.
type groupTexts struct {
	Title   Text `json:"title,omitempty"`
	Caption Text `json:"caption,omitempty"`
	Summary Text `json:"summary,omitempty"`
	Running Text `json:"running,omitempty"`
	Note    Text `json:"note,omitempty"`
}

// progressTexts are the messages of a group of a run in progress.
type progressTexts struct {
	Title Text `json:"title,omitempty"`
	Text  Text `json:"text,omitempty"`
}

// estimateTexts are the messages of the estimates' labels, by position.
type estimateTexts struct {
	Rows      []Text       `json:"rows,omitempty"`
	Stream    Text         `json:"stream,omitempty"`
	Split     Text         `json:"split,omitempty"`
	Shares    []shareTexts `json:"shares,omitempty"`
	Throttles []Text       `json:"throttles,omitempty"`
	// Basis says what the estimate includes (Route.Timeline.BasisLabel).
	Basis Text `json:"basis,omitempty"`
}

type shareTexts struct {
	Status Text `json:"status,omitempty"`
	Price  Text `json:"price,omitempty"`
	Rate   Text `json:"rate,omitempty"`
}

// MarshalJSON adds the kept messages ("i18n") to the check.
func (c ReadinessCheck) MarshalJSON() ([]byte, error) {
	type plain ReadinessCheck
	return json.Marshal(struct {
		plain
		I18n *checkTexts `json:"i18n,omitempty"`
	}{plain(c), c.texts})
}

// UnmarshalJSON reads a check and its kept messages.
func (c *ReadinessCheck) UnmarshalJSON(b []byte) error {
	type plain ReadinessCheck
	var v struct {
		plain
		I18n *checkTexts `json:"i18n"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*c = ReadinessCheck(v.plain)
	c.texts = v.I18n
	return nil
}

// MarshalJSON adds the kept messages ("i18n") to the group.
func (g ReadinessGroup) MarshalJSON() ([]byte, error) {
	type plain ReadinessGroup
	return json.Marshal(struct {
		plain
		I18n *groupTexts `json:"i18n,omitempty"`
	}{plain(g), g.texts})
}

// UnmarshalJSON reads a group and its kept messages.
func (g *ReadinessGroup) UnmarshalJSON(b []byte) error {
	type plain ReadinessGroup
	var v struct {
		plain
		I18n *groupTexts `json:"i18n"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*g = ReadinessGroup(v.plain)
	g.texts = v.I18n
	return nil
}

// MarshalJSON adds the kept messages ("i18n") to the group in progress.
func (g ReadinessProgressGroup) MarshalJSON() ([]byte, error) {
	type plain ReadinessProgressGroup
	return json.Marshal(struct {
		plain
		I18n *progressTexts `json:"i18n,omitempty"`
	}{plain(g), g.texts})
}

// UnmarshalJSON reads a group in progress and its kept messages.
func (g *ReadinessProgressGroup) UnmarshalJSON(b []byte) error {
	type plain ReadinessProgressGroup
	var v struct {
		plain
		I18n *progressTexts `json:"i18n"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*g = ReadinessProgressGroup(v.plain)
	g.texts = v.I18n
	return nil
}

// MarshalJSON adds the labels' kept messages ("i18n") to the estimates.
func (e ReadinessEstimates) MarshalJSON() ([]byte, error) {
	type plain ReadinessEstimates
	return json.Marshal(struct {
		plain
		I18n *estimateTexts `json:"i18n,omitempty"`
	}{plain(e), e.texts})
}

// UnmarshalJSON reads the estimates and their labels' kept messages.
func (e *ReadinessEstimates) UnmarshalJSON(b []byte) error {
	type plain ReadinessEstimates
	var v struct {
		plain
		I18n *estimateTexts `json:"i18n"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*e = ReadinessEstimates(v.plain)
	e.texts = v.I18n
	return nil
}

// asStored reports whether a reader reads the English a run stored: English
// with the standard formats, as API clients and agents do.
func asStored(l i18n.Lang) bool {
	return (l.Locale == i18n.English || l.Locale == "") && (l.Region == i18n.Standard || l.Region == "")
}

// pick is a text in a reader's language, or the English the run stored when
// it kept no message for it (or the message is gone).
func pick(stored string, t Text, l i18n.Lang) string {
	if t.Empty() || asStored(l) {
		return stored
	}
	if s := t.In(l); s != "" {
		return s
	}
	return stored
}

// In is the check as a reader reads it, in their language and region,
// without its kept messages.
func (c ReadinessCheck) In(l i18n.Lang) ReadinessCheck {
	out := c
	out.texts = nil
	out.Fixes = append(make([]ReadinessFix, 0, len(c.Fixes)), c.Fixes...)
	if c.Rows != nil {
		out.Rows = append(make([]ReadinessCheckRow, 0, len(c.Rows)), c.Rows...)
	}
	if c.Technical != nil {
		out.Technical = append(make([]ReadinessTechRow, 0, len(c.Technical)), c.Technical...)
	}
	if c.Status == Warn {
		out.AckKey = ackKey(c)
	}
	t := c.texts
	if t == nil || asStored(l) {
		return out
	}
	out.Title = pick(c.Title, t.Title, l)
	out.FlatTitle = pick(c.FlatTitle, t.FlatTitle, l)
	out.Detail = pick(c.Detail, t.Detail, l)
	out.Label = pick(c.Label, t.Label, l)
	out.Short = pick(c.Short, t.Short, l)
	out.Ack = pick(c.Ack, t.Ack, l)
	out.Note = pick(c.Note, t.Note, l)
	if !t.Evidence.Empty() {
		if s := t.Evidence.sentencesIn(l); s != "" {
			out.Evidence = s
		}
	}
	for i := range out.Fixes {
		if i < len(t.Fixes) {
			out.Fixes[i].Label = pick(c.Fixes[i].Label, t.Fixes[i], l)
		}
	}
	for i := range out.Technical {
		if i < len(t.Technical) {
			out.Technical[i].Label = pick(c.Technical[i].Label, t.Technical[i].Label, l)
			out.Technical[i].Value = pick(c.Technical[i].Value, t.Technical[i].Value, l)
		}
	}
	for i := range out.Rows {
		if i < len(t.Rows) {
			r := t.Rows[i]
			out.Rows[i].Label = pick(c.Rows[i].Label, r.Label, l)
			out.Rows[i].Detail = pick(c.Rows[i].Detail, r.Detail, l)
			out.Rows[i].GetsName = pick(c.Rows[i].GetsName, r.GetsName, l)
		}
	}
	return out
}

// ackKey is the same for warnings a person confirms with the same
// sentence, in whatever language they read it: the English the run stored
// is the sentence's identity.
func ackKey(c ReadinessCheck) string {
	s := "ack:" + c.Ack
	if c.Ack == "" {
		s = "title:" + c.Title
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// In is the group as a reader reads it.
func (g ReadinessGroup) In(l i18n.Lang) ReadinessGroup {
	out := g
	out.texts = nil
	t := g.texts
	if t == nil || asStored(l) {
		return out
	}
	out.Title = pick(g.Title, t.Title, l)
	out.Caption = pick(g.Caption, t.Caption, l)
	out.Summary = pick(g.Summary, t.Summary, l)
	out.Running = pick(g.Running, t.Running, l)
	out.Note = pick(g.Note, t.Note, l)
	return out
}

// In is the group in progress as a reader reads it.
func (g ReadinessProgressGroup) In(l i18n.Lang) ReadinessProgressGroup {
	out := g
	out.texts = nil
	if g.texts == nil || asStored(l) {
		return out
	}
	out.Title = pick(g.Title, g.texts.Title, l)
	out.Text = pick(g.Text, g.texts.Text, l)
	return out
}

// In is the estimates as a reader reads them.
func (e ReadinessEstimates) In(l i18n.Lang) ReadinessEstimates {
	out := e
	out.texts = nil
	out.Audience.Rows = append(make([]ReadinessAudienceRow, 0, len(e.Audience.Rows)), e.Audience.Rows...)
	out.Route.Shares = append(make([]ReadinessRouteShare, 0, len(e.Route.Shares)), e.Route.Shares...)
	out.Throttling = append(make([]ReadinessThrottle, 0, len(e.Throttling)), e.Throttling...)
	t := e.texts
	if t == nil || asStored(l) {
		return out
	}
	for i := range out.Audience.Rows {
		if i < len(t.Rows) {
			out.Audience.Rows[i].Label = pick(e.Audience.Rows[i].Label, t.Rows[i], l)
		}
	}
	out.Route.StreamLabel = pick(e.Route.StreamLabel, t.Stream, l)
	out.Route.SplitLabel = pick(e.Route.SplitLabel, t.Split, l)
	for i := range out.Route.Shares {
		if i < len(t.Shares) {
			s := t.Shares[i]
			out.Route.Shares[i].StatusLabel = pick(e.Route.Shares[i].StatusLabel, s.Status, l)
			out.Route.Shares[i].PriceLabel = pick(e.Route.Shares[i].PriceLabel, s.Price, l)
			out.Route.Shares[i].RateLabel = pick(e.Route.Shares[i].RateLabel, s.Rate, l)
		}
	}
	for i := range out.Throttling {
		if i < len(t.Throttles) {
			out.Throttling[i].Name = pick(e.Throttling[i].Name, t.Throttles[i], l)
		}
	}
	if e.Route.Timeline != nil {
		tl := *e.Route.Timeline
		tl.BasisLabel = pick(tl.BasisLabel, t.Basis, l)
		out.Route.Timeline = &tl
	}
	return out
}

// In is the result as a reader reads it.
func (r ReadinessResult) In(l i18n.Lang) ReadinessResult {
	out := r
	out.Checks = InChecks(r.Checks, l)
	if r.Groups != nil {
		out.Groups = make([]ReadinessGroup, len(r.Groups))
		for i, g := range r.Groups {
			out.Groups[i] = g.In(l)
		}
	}
	out.Problem = pick(r.Problem, r.problem, l)
	out.problem = nil
	out.Estimates = r.Estimates.In(l)
	return out
}

// InChecks is checks as a reader reads them.
func InChecks(checks []ReadinessCheck, l i18n.Lang) []ReadinessCheck {
	if checks == nil {
		return nil
	}
	out := make([]ReadinessCheck, len(checks))
	for i, c := range checks {
		out[i] = c.In(l)
	}
	return out
}

// In is the run as a reader reads it: its texts in their language and
// region, or as stored for English readers with the standard formats (API
// clients, agents) and for runs stored before their texts were kept.
func (r ReadinessRun) In(l i18n.Lang) ReadinessRun {
	r.ReadinessResult = r.ReadinessResult.In(l)
	return r
}

// In is the run in progress as a reader reads it.
func (p ReadinessProgress) In(l i18n.Lang) ReadinessProgress {
	out := p
	if p.Groups != nil {
		out.Groups = make([]ReadinessProgressGroup, len(p.Groups))
		for i, g := range p.Groups {
			out.Groups[i] = g.In(l)
		}
	}
	return out
}

// ShareStatus is the kept message of the status label of the route's i-th
// provider, for a reader that copies the label (an approval's snapshot);
// nothing when the run kept none.
func (e ReadinessEstimates) ShareStatus(i int) Text {
	if e.texts == nil || i < 0 || i >= len(e.texts.Shares) {
		return nil
	}
	return e.texts.Shares[i].Status
}

// ShareStatusIn is a copied status label of a route's provider in a
// reader's language: from its kept message, or from its status and warm-up
// day when they say what the copied English says. Any other label stays as
// it was copied.
func ShareStatusIn(english string, kept Text, status string, day, days int, l i18n.Lang) string {
	if asStored(l) {
		return english
	}
	if !kept.Empty() {
		return pick(english, kept, l)
	}
	said := []Text{statusText(status)}
	if status == state.WarmingUp && days > 0 {
		said = append(said, tx("share.warmingUp", A{"day": day, "days": days}))
	}
	for _, t := range said {
		if t.en() == english {
			return t.In(l)
		}
	}
	return english
}

// SplitLabelIn is a copied split label of the Broadcasts route ("Broadcasts
// 90 / 10", as Route writes it for an approval's snapshot) in a reader's
// language. Any other label stays as it was copied.
func SplitLabelIn(english string, l i18n.Lang) string {
	if asStored(l) {
		return english
	}
	if t := broadcastSplit(strings.TrimSpace(strings.TrimPrefix(english, "Broadcasts"))); t.en() == english {
		return t.In(l)
	}
	return english
}

// ForReader is a run in the request's language (nil stays nil).
func ForReader(ctx context.Context, r *ReadinessRun) *ReadinessRun {
	if r == nil {
		return nil
	}
	out := r.In(i18n.LangFrom(ctx))
	return &out
}

// ProgressForReader is a run in progress in the request's language.
func ProgressForReader(ctx context.Context, p *ReadinessProgress) *ReadinessProgress {
	if p == nil {
		return nil
	}
	out := p.In(i18n.LangFrom(ctx))
	return &out
}

// WithoutTexts drops the kept messages ("i18n" members) from stored JSON,
// for readers that get it as it is stored (MCP agents read English).
// Everything else keeps its order and values.
func WithoutTexts(raw json.RawMessage) json.RawMessage {
	if !bytes.Contains(raw, []byte(`"i18n"`)) {
		return raw
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var b bytes.Buffer
	if err := copyWithout(dec, &b); err != nil {
		return raw
	}
	return b.Bytes()
}

func copyWithout(dec *json.Decoder, b *bytes.Buffer) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		v, err := json.Marshal(tok)
		if err != nil {
			return err
		}
		b.Write(v)
		return nil
	}
	switch d {
	case '{':
		b.WriteByte('{')
		first := true
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return err
			}
			key, _ := kt.(string)
			if key == "i18n" {
				var skip json.RawMessage
				if err := dec.Decode(&skip); err != nil {
					return err
				}
				continue
			}
			if !first {
				b.WriteByte(',')
			}
			first = false
			kb, err := json.Marshal(key)
			if err != nil {
				return err
			}
			b.Write(kb)
			b.WriteByte(':')
			if err := copyWithout(dec, b); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return err
		}
		b.WriteByte('}')
	case '[':
		b.WriteByte('[')
		first := true
		for dec.More() {
			if !first {
				b.WriteByte(',')
			}
			first = false
			if err := copyWithout(dec, b); err != nil {
				return err
			}
		}
		if _, err := dec.Token(); err != nil {
			return err
		}
		b.WriteByte(']')
	}
	return nil
}
