package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/segments"
)

// demoModel is the demo's sample connection (Anthropic, Claude Sonnet 5.5,
// "connected by Marcus Chen"): in development it answers from recordings of
// the brief's samples and of the demo's emails, and with plain rewrites of
// anything else. It never reaches the network: the demo's key isn't real.
type demoModel struct{}

var _ llm.Provider = demoModel{}

func (demoModel) Models(context.Context) ([]llm.Model, error) {
	return []llm.Model{{ID: "claude-sonnet-5-5", Name: "Claude Sonnet 5.5"}, {ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5"}}, nil
}

// estimate counts tokens the way the providers roughly do: four
// characters a token.
func estimate(s string) int { return max(1, (len(s)+3)/4) }

func (demoModel) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	if err := ctx.Err(); err != nil {
		return llm.Response{}, &llm.Error{Kind: llm.ErrNetwork, Err: err, Message: "cancelled"}
	}
	in := len(req.System)
	for _, m := range req.Messages {
		for _, p := range m.Parts {
			in += len(p.Text)
			if p.Image != nil {
				in += 4 * 1100
			}
		}
	}
	var answer any
	switch t := req.Task.(type) {
	case testTask:
		return llm.Response{Text: "Ready.", Model: "claude-sonnet-5-5", InputTokens: 14, OutputTokens: 3, StopReason: llm.StopEnd}, nil
	case subjectTask:
		answer = map[string]any{"subjects": demoSubjects(t)}
	case rewriteTask:
		if t.Action == "instruction" && demoRefuses(t.Instruction) {
			answer = map[string]any{"refusal": "I can't write a subject that pretends to be a reply to an email the reader never sent."}
		} else {
			answer = map[string]any{"text": demoRewrite(t)}
		}
	case translateTask:
		segs := []translateSegment{}
		for _, s := range t.Segments {
			segs = append(segs, translateSegment{ID: s.ID, Text: demoTranslate(t.Target, s.Text)})
		}
		answer = map[string]any{"subject": demoTranslate(t.Target, t.Subject), "preheader": demoTranslate(t.Target, t.Preheader), "segments": segs}
	case altTask:
		answer = map[string]any{"alt": demoAlt(t)}
	case linkTask:
		answer = map[string]any{"text": demoLinkText(t)}
	case segmentTask:
		answer = demoSegment(t)
	case summaryTask:
		answer = map[string]any{"summary": demoSummary(t.Numbers)}
	default:
		return llm.Response{}, &llm.Error{Kind: llm.ErrBadRequest, Status: 400, Code: "invalid_request_error", Message: "the demo model answers the assistant's features only"}
	}
	b, _ := json.Marshal(answer)
	return llm.Response{Text: string(b), Model: "claude-sonnet-5-5", InputTokens: estimate(strings.Repeat("x", in)), OutputTokens: estimate(string(b)), StopReason: llm.StopEnd}, nil
}

// demoRefuses: an instruction to fake a reply.
func demoRefuses(instruction string) bool {
	low := strings.ToLower(instruction)
	return strings.Contains(low, "re:") || strings.Contains(low, "reply to them") || strings.Contains(low, "look like a reply") || strings.Contains(low, "fwd:")
}

func demoSubjects(t subjectTask) []string {
	text := strings.ToLower(t.Subject + " " + t.Text)
	switch {
	case strings.Contains(text, "on-call") && strings.Contains(text, "migration"):
		// The brief's sample (Weekly Digest #143 as phase 3 draws it).
		return []string{"A year of on-call, in numbers", "How we test every database migration", "On-call data, safer migrations and what to read next",
			"You won't believe our on-call numbers", "Weekly Digest #143: on-call, migrations, your vote"}
	case strings.Contains(text, "scheduled rollouts") || strings.Contains(text, "schedule flag changes"):
		// Weekly Digest #143 as the demo has it.
		return []string{"Scheduled rollouts are here", "Plan a flag change for 2 a.m., then sleep through it", "Scheduled rollouts, faster CSV exports and the v1 API",
			"You won't believe how fast exports got", "Weekly Digest #143: rollouts, exports, v1 API"}
	}
	heads := demoHeadlines(t.Text)
	base := strings.TrimSpace(t.Subject)
	if base == "" && len(heads) > 0 {
		base = heads[0]
	}
	out := []string{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && len([]rune(s)) <= 70 {
			out = append(out, s)
		}
	}
	for _, h := range heads {
		add(h)
	}
	add("This week: " + strings.ToLower(firstWords(base, 6)))
	add(firstWords(base, 8))
	add("What's new: " + strings.ToLower(firstWords(base, 6)))
	add("In short: " + strings.ToLower(firstWords(base, 6)))
	for len(out) < 5 {
		out = append(out, fmt.Sprintf("%s (%d)", firstWords(base, 6), len(out)+1))
	}
	return out[:5]
}

// demoHeadlines are the short lines of an email's text.
func demoHeadlines(text string) []string {
	out := []string{}
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if n := len(strings.Fields(l)); n >= 3 && n <= 9 && !strings.ContainsAny(l, "{}") && len(out) < 2 {
			out = append(out, l)
		}
	}
	return out
}

func firstWords(s string, n int) string {
	w := strings.Fields(s)
	if len(w) > n {
		w = w[:n]
	}
	return strings.TrimRight(strings.Join(w, " "), ",.;:")
}

var (
	demoIntroOnCall = regexp.MustCompile(`this week we're sharing a lot of numbers from our first full year of on-call`)
	demoIntro143    = regexp.MustCompile(`you can now schedule flag changes ahead of time`)
	fillerRe        = regexp.MustCompile(`(?i)\b(a lot of|also|every single|really|very|just|quite|actually|basically|in order to|so you can|at the end)\b ?`)
	contractions    = strings.NewReplacer("we're", "we are", "We're", "We are", "you're", "you are", "it's", "it is", "It's", "It is", "don't", "do not", "can't", "cannot", "won't", "will not", "you'll", "you will", "we'll", "we will", "here's", "here is", "Here's", "Here is", "what's", "what is", "isn't", "is not")
)

func demoRewrite(t rewriteTask) string {
	src := t.HTML
	switch t.Action {
	case "shorten":
		switch {
		case demoIntroOnCall.MatchString(src):
			return `Hi {{ first_name | default: "there" }}, this week: a year of on-call in numbers, how our database team tests every migration before production, and a one-click poll on what we should write next.`
		case demoIntro143.MatchString(src):
			return strings.Replace(strings.Replace(src, "you can now schedule flag changes ahead of time, in any timezone.", "flag changes can now go out on a schedule, in any timezone.", 1),
				">Click here</a> to see how it works.", ">See how it works</a>.", 1)
		}
		out := fillerRe.ReplaceAllString(src, "")
		return demoTrimWords(out, max(8, words(src)*6/10))
	case "friendlier":
		out := strings.Replace(src, "Hi ", "Hey ", 1)
		if !strings.Contains(out, "Good news") {
			out = strings.Replace(out, ", ", "! Good news: ", 1)
		}
		return out
	case "formal":
		out := contractions.Replace(src)
		out = strings.Replace(out, "Hi ", "Hello ", 1)
		out = strings.Replace(out, "Hey ", "Hello ", 1)
		return out
	case "direct":
		out := fillerRe.ReplaceAllString(src, "")
		out = strings.NewReplacer("you can now ", "", "We think ", "", "we think ", "", "perhaps ", "", "might ", "will ").Replace(out)
		return out
	}
	// With an instruction: the demo keeps the text and tightens it.
	return fillerRe.ReplaceAllString(src, "")
}

// demoTrimWords keeps the first n words of an HTML text, its tokens whole
// and its tags closed.
func demoTrimWords(src string, n int) string {
	text := plain(src)
	if words(text) <= n {
		return src
	}
	// Cut the plain text, then keep the source's tokens.
	parts := wordRe.FindAllString(text, -1)
	out := strings.Join(parts[:n], " ")
	out = strings.TrimRight(out, ",;:")
	if !strings.HasSuffix(out, ".") {
		out += "."
	}
	return out
}

// demoPhrases are the demo emails' texts in Spanish and German.
//
//nolint:misspell // Spanish and German words
var demoPhrases = map[string]map[string]string{
	"es": {
		"Scheduled rollouts, faster exports, and 6 weeks left on the v1 API":             "Despliegues programados, exportaciones más rápidas y 6 semanas para la API v1",
		"Plan a flag change for 2 a.m. and sleep through it.":                            "Programa un cambio de flag para las 2 de la madrugada y sigue durmiendo.",
		"Scheduled rollouts are here":                                                    "Llegan los despliegues programados",
		"Scheduled rollouts":                                                             "Despliegues programados",
		"Faster CSV exports":                                                             "Exportaciones CSV más rápidas",
		"Pick the minute a flag changes. Every scheduled change shows in the audit log.": "Elige el minuto en que cambia un flag. Cada cambio programado aparece en el registro de auditoría.",
		"Exports of up to a million rows now finish in under a minute.":                  "Las exportaciones de hasta un millón de filas terminan ahora en menos de un minuto.",
		"Read the docs":              "Leer la documentación",
		"See the changelog":          "Ver las novedades",
		"6 weeks left on the v1 API": "Quedan 6 semanas para la API v1",
		"Plan your migration":        "Planifica tu migración",
		"Blog":                       "Blog", "Docs": "Documentación", "Changelog": "Novedades",
		"Unsubscribe": "Darse de baja", "Email preferences": "Preferencias de correo", "View in browser": "Ver en el navegador",
		"The schedule picker in the flag editor": "El selector de horario en el editor de flags",
		"The CSV export dialog":                  "El diálogo de exportación CSV",
		"Acme":                                   "Acme",
		"A year of on-call, in numbers":          "Un año de guardias, en cifras",
		"A year of on-call data, how we test migrations, and a one-click poll.": "Un año de datos de guardias, cómo probamos las migraciones y una encuesta de un clic.",
		"What should we write about next?":                                      "¿Sobre qué deberíamos escribir ahora?",
		"Postgres internals":                                                    "Detalles internos de Postgres",
		"Feature flag patterns":                                                 "Patrones de feature flags",
		"On-call stories":                                                       "Historias de guardias",
		"Rust in production":                                                    "Rust en producción",
	},
	"de": {
		"Scheduled rollouts, faster exports, and 6 weeks left on the v1 API":             "Geplante Rollouts, schnellere Exporte und noch 6 Wochen für die v1-API",
		"Plan a flag change for 2 a.m. and sleep through it.":                            "Planen Sie eine Flag-Änderung für 2 Uhr nachts und schlafen Sie durch.",
		"Scheduled rollouts are here":                                                    "Geplante Rollouts sind da",
		"Scheduled rollouts":                                                             "Geplante Rollouts",
		"Faster CSV exports":                                                             "Schnellere CSV-Exporte",
		"Pick the minute a flag changes. Every scheduled change shows in the audit log.": "Wählen Sie die Minute, in der sich ein Flag ändert. Jede geplante Änderung erscheint im Audit-Log.",
		"Exports of up to a million rows now finish in under a minute.":                  "Exporte mit bis zu einer Million Zeilen sind jetzt in unter einer Minute fertig.",
		"Read the docs":              "Zur Dokumentation",
		"See the changelog":          "Zum Changelog",
		"6 weeks left on the v1 API": "Noch 6 Wochen für die v1-API",
		"Plan your migration":        "Migration planen",
		"Blog":                       "Blog", "Docs": "Doku", "Changelog": "Changelog",
		"Unsubscribe": "Abmelden", "Email preferences": "E-Mail-Einstellungen", "View in browser": "Im Browser ansehen",
		"The schedule picker in the flag editor": "Die Zeitauswahl im Flag-Editor",
		"The CSV export dialog":                  "Der Dialog für den CSV-Export",
		"Acme":                                   "Acme",
	},
}

// demoSentences translate the demo's longer texts with links and tokens.
//
//nolint:misspell // Spanish and German words
var demoSentences = map[string][][2]string{
	"es": {
		{"you can now schedule flag changes ahead of time, in any timezone.", "ya puedes programar cambios de flags con antelación, en cualquier zona horaria."},
		{"you can now schedule flag changes ahead of time.", "ya puedes programar cambios de flags con antelación."},
		{">Click here</a> to see how it works.", ">Haz clic aquí</a> para ver cómo funciona."},
		{"Hi ", "Hola "},
		{"The v1 API stops answering on 30 November. Migration guide: ", "La API v1 deja de responder el 30 de noviembre. Guía de migración: "},
		{", this week: a year of on-call in numbers, how our database team tests every migration before production, and a one-click poll on what we should write next.",
			", esta semana: un año de guardias en cifras, cómo nuestro equipo de bases de datos prueba cada migración antes de producción y una encuesta de un clic sobre qué deberíamos escribir."},
	},
	"de": {
		{"you can now schedule flag changes ahead of time, in any timezone.", "Sie können Flag-Änderungen jetzt im Voraus planen, in jeder Zeitzone."},
		{"you can now schedule flag changes ahead of time.", "Sie können Flag-Änderungen jetzt im Voraus planen."},
		{">Click here</a> to see how it works.", ">Hier klicken</a>, um zu sehen, wie es funktioniert."},
		{"Hi ", "Hallo "},
		{"The v1 API stops answering on 30 November. Migration guide: ", "Die v1-API antwortet ab dem 30. November nicht mehr. Migrationsleitfaden: "},
	},
}

// demoTranslate translates the demo's texts; anything else stays as it is.
func demoTranslate(lang, s string) string {
	t := strings.TrimSpace(s)
	if v, ok := demoPhrases[lang][t]; ok {
		return v
	}
	// A text whose tags hold a known phrase: "<h1 …>Scheduled rollouts are here</h1>".
	out := s
	for en, tr := range demoPhrases[lang] {
		out = strings.ReplaceAll(out, ">"+en+"<", ">"+tr+"<")
	}
	for _, p := range demoSentences[lang] {
		out = strings.ReplaceAll(out, p[0], p[1])
	}
	return out
}

func demoAlt(t altTask) string {
	name := strings.ToLower(t.FileName)
	switch {
	case strings.Contains(name, "oncall-heatmap"):
		return "Heat map of on-call pages by hour and weekday in 2025. Most pages came between 09:00 and 11:00 on weekdays."
	case strings.Contains(name, "scheduled-rollouts"):
		return "The flag editor's schedule picker, with a rollout set for 02:00 in New York."
	case strings.Contains(name, "article-card"):
		return "Screenshot of the feature the card describes."
	}
	base := strings.TrimSuffix(name, ".png")
	base = strings.TrimSuffix(strings.TrimSuffix(base, ".jpg"), ".gif")
	words := strings.FieldsFunc(base, func(r rune) bool { return r == '-' || r == '_' || unicode.IsDigit(r) })
	if len(words) == 0 {
		return "Illustration for this part of the email."
	}
	w := strings.Join(words, " ")
	return strings.ToUpper(w[:1]) + w[1:] + ", as the email describes it."
}

func demoLinkText(t linkTask) string {
	href := strings.ToLower(t.Href)
	switch {
	case strings.Contains(href, "on-call-2025-charts") || strings.Contains(href, "on-call"):
		return "See all the 2025 on-call charts"
	case strings.Contains(href, "scheduled-rollouts"):
		return "See how scheduled rollouts work"
	case strings.Contains(href, "migrate-v2"):
		return "Read the v2 migration guide"
	}
	u := strings.Trim(href, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		u = u[i+1:]
	}
	w := strings.Join(strings.FieldsFunc(u, func(r rune) bool { return r == '-' || r == '_' || r == '.' }), " ")
	if w == "" {
		return "Read more about this"
	}
	return "Read about " + w
}

var (
	demoDaysRe = regexp.MustCompile(`(?i)(\d+)\s*(day|days|week|weeks)`)
	twoWeeksRe = regexp.MustCompile(`(?i)two weeks|2 weeks`)
)

func cond(field, op, value string, amount int, unit string) segments.SegmentCondition {
	return segments.SegmentCondition{Field: field, Op: op, Value: value, Amount: amount, Unit: unit}
}

// demoSegment drafts conditions for the brief's samples, and for plain
// descriptions with mailbox providers, tiers, opens and clicks.
func demoSegment(t segmentTask) map[string]any {
	d := strings.ToLower(t.Description)
	hasField := func(key string) bool {
		for _, f := range t.Schema.Fields {
			if f.Key == key {
				return true
			}
		}
		return false
	}
	switch {
	case strings.Contains(d, "phone") || strings.Contains(d, "mobile") || strings.Contains(d, "device"):
		return map[string]any{"nothing": "There's no condition for “opened on a phone”."}
	case strings.Contains(d, "team-plan") || strings.Contains(d, "team plan"):
		conds := []segments.SegmentCondition{}
		if hasField("custom:plan") {
			conds = append(conds, cond("custom:plan", "equals", "team", 0, ""))
		}
		conds = append(conds, cond("clicked", "in_last", "", 30, "days"), cond("last_open", "not_in_last", "", 14, "days"))
		return map[string]any{"groups": []segments.SegmentGroup{{Match: "all", Conditions: conds}},
			"note": "There's no condition for a specific link, so I used “Clicked any email”."}
	case strings.Contains(d, "instead of") && len(t.Current.Groups) > 0:
		groups := []segments.SegmentGroup{}
		for i, g := range t.Current.Groups {
			ng := segments.SegmentGroup{Match: g.Match, Conditions: []segments.SegmentCondition{}}
			for _, c := range g.Conditions {
				if c.Field == "mailbox_provider" && strings.Contains(d, "gmail") {
					c.Value = "gmail"
				}
				ng.Conditions = append(ng.Conditions, c)
			}
			if i == 0 && strings.Contains(d, "active") {
				ng.Conditions = append(ng.Conditions, cond("tier", "is", "active", 0, ""))
			}
			groups = append(groups, ng)
		}
		note := ""
		if len(groups) > 1 {
			note = "Group 2 stays as it is."
		}
		return map[string]any{"groups": groups, "note": note}
	}
	conds := []segments.SegmentCondition{}
	for _, mp := range []string{"gmail", "outlook", "yahoo", "icloud"} {
		if strings.Contains(d, mp) {
			conds = append(conds, cond("mailbox_provider", "is", mp, 0, ""))
		}
	}
	for _, tier := range []string{"active", "cooling", "at risk", "sunset"} {
		if strings.Contains(d, tier) {
			conds = append(conds, cond("tier", "is", strings.ReplaceAll(tier, " ", "_"), 0, ""))
		}
	}
	days := 30
	if m := demoDaysRe.FindStringSubmatch(d); m != nil {
		n := atoi(m[1])
		if strings.HasPrefix(m[2], "week") {
			n *= 7
		}
		days = n
	}
	if twoWeeksRe.MatchString(d) {
		days = 14
	}
	switch {
	case strings.Contains(d, "haven't opened") || strings.Contains(d, "not opened") || strings.Contains(d, "didn't open"):
		conds = append(conds, cond("last_open", "not_in_last", "", days, "days"))
	case strings.Contains(d, "opened"):
		conds = append(conds, cond("opened", "in_last", "", days, "days"))
	}
	if strings.Contains(d, "clicked") {
		conds = append(conds, cond("clicked", "in_last", "", days, "days"))
	}
	if len(conds) == 0 {
		return map[string]any{"nothing": "There's no condition for that description."}
	}
	return map[string]any{"groups": []segments.SegmentGroup{{Match: "all", Conditions: conds}}}
}

// demoSummary describes a report's numbers in the brief's words.
func demoSummary(n summaryNumbers) string {
	pct := func(p *float64) string {
		if p == nil {
			return "0.0%"
		}
		return fmt.Sprintf("%.1f%%", *p)
	}
	num := func(v int64) string { return groupThousands(v) }
	var b strings.Builder
	fmt.Fprintf(&b, "%s reached %s inboxes, %s of the %s sent. ", n.Campaign, num(n.Delivered), pct(n.DeliveredRate), num(n.Sent))
	fmt.Fprintf(&b, "%s opened it as people; another %s were machine opens, which aren't counted as people. ", pct(n.HumanOpenRate), pct(n.MachineOpenRate))
	if len(n.TopLinks) > 0 {
		fmt.Fprintf(&b, "%s clicked, and the “%s” link drew %.0f%% of the clicks. ", pct(n.ClickRate), n.TopLinks[0].Label, n.TopLinks[0].Share)
	} else {
		fmt.Fprintf(&b, "%s clicked. ", pct(n.ClickRate))
	}
	if ab := n.ABTest; ab != nil && ab.Winner != "" && len(ab.Variants) >= 2 {
		var win, other summaryVariant
		for _, v := range ab.Variants {
			if v.Key == ab.Winner {
				win = v
			} else if other.Key == "" {
				other = v
			}
		}
		if ab.Kind == "subject" {
			fmt.Fprintf(&b, "Subject %s, “%s”, won the test with %s human opens against %s. ", win.Key, win.Label, pct(win.HumanOpenRate), pct(other.HumanOpenRate))
		} else {
			fmt.Fprintf(&b, "Variant %s won the test with %s human opens against %s. ", win.Key, pct(win.HumanOpenRate), pct(other.HumanOpenRate))
		}
	}
	var worst *summaryMailbox
	for i := range n.Mailboxes {
		m := &n.Mailboxes[i]
		if m.BounceRate != nil && (worst == nil || *m.BounceRate > *worst.BounceRate) {
			worst = m
		}
	}
	if worst != nil {
		fmt.Fprintf(&b, "%s had the highest bounce rate (%s). ", worst.Name, pct(worst.BounceRate))
	}
	if n.Unsubscribes > 0 {
		if len(n.Unsubscribe) > 0 {
			top := n.Unsubscribe[0]
			for _, u := range n.Unsubscribe {
				if u.Count > top.Count {
					top = u
				}
			}
			fmt.Fprintf(&b, "%s people unsubscribed, most often for “%s”.", num(n.Unsubscribes), strings.ToLower(top.Reason))
		} else {
			fmt.Fprintf(&b, "%s people unsubscribed.", num(n.Unsubscribes))
		}
	}
	return strings.TrimSpace(b.String())
}

// groupThousands writes 47548 as 47,548.
func groupThousands(v int64) string {
	s := fmt.Sprintf("%d", v)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
