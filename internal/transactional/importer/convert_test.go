package importer

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/liquid"
)

func body(d Dialect, src string) Converted {
	return Convert(Source{Ref: "t", Name: "T", Body: src}, Options{Dialect: d, Transactional: true})
}

func hasNote(c Converted, kind NoteKind, sub string) *Note {
	for i, n := range c.Notes {
		if n.Kind == kind && strings.Contains(n.Message, sub) {
			return &c.Notes[i]
		}
	}
	return nil
}

func varByName(c Converted, name string) *Variable {
	for i, v := range c.Variables {
		if v.Name == name {
			return &c.Variables[i]
		}
	}
	return nil
}

func render(t *testing.T, src string, data map[string]any) string {
	t.Helper()
	out, err := liquid.Render(src, data)
	if err != nil {
		t.Fatalf("render %q: %v", src, err)
	}
	return out
}

func TestMustachioVariablesAndURLs(t *testing.T) {
	c := Convert(Source{Ref: "verify-email", Name: "Account verification", Subject: "Verify your email, {{ name }}",
		Body: `<p>Hi {{ name }},</p>` + "\n" + `<a href="{{ action_url }}">Verify</a>`}, Options{Dialect: DialectPostmark, Transactional: true})
	if c.Body != `<p>Hi {{ name }},</p>`+"\n"+`<a href="{{ action_url }}">Verify</a>` || c.Subject != "Verify your email, {{ name }}" {
		t.Fatalf("body %q subject %q", c.Body, c.Subject)
	}
	if c.Status != StatusConverted || c.Summary() != "2 variables declared: name, action_url" {
		t.Fatalf("status %s summary %q notes %+v", c.Status, c.Summary(), c.Notes)
	}
	n, u := varByName(c, "name"), varByName(c, "action_url")
	if n == nil || n.Type != TypeText || !n.Required || u == nil || u.Type != TypeURL || !u.Required {
		t.Fatalf("variables: %+v", c.Variables)
	}
	if got := render(t, c.Body, c.TestData); !strings.Contains(got, `href="https://example.com/action-url"`) {
		t.Fatalf("sample render: %s", got)
	}
}

func TestMustachioEachBecomesFor(t *testing.T) {
	src := "<table>\n{{#each items}}\n<tr><td>{{description}}</td><td>{{amount}}</td><td>{{../currency}}</td></tr>\n{{/each}}\n</table>"
	c := body(DialectPostmark, src)
	want := "<table>\n{% for item in items %}\n<tr><td>{{ item.description }}</td><td>{{ item.amount }}</td><td>{{ currency }}</td></tr>\n{% endfor %}\n</table>"
	if c.Body != want {
		t.Fatalf("got\n%s\nwant\n%s", c.Body, want)
	}
	n := hasNote(c, NoteChange, "`{{#each items}}` became `{% for item in items %}`")
	if c.Status != StatusNotes || n == nil || n.Line != 2 {
		t.Fatalf("status %s notes %+v", c.Status, c.Notes)
	}
	items := varByName(c, "items")
	if items == nil || items.Type != TypeList || !items.Required {
		t.Fatalf("items: %+v", c.Variables)
	}
	// currency is only printed inside the loop: optional.
	if cur := varByName(c, "currency"); cur == nil || cur.Required {
		t.Fatalf("currency: %+v", c.Variables)
	}
	got := render(t, c.Body, map[string]any{"currency": "USD", "items": []any{map[string]any{"description": "Pro plan", "amount": "$29"}}})
	if !strings.Contains(got, "<td>Pro plan</td><td>$29</td><td>USD</td>") {
		t.Fatalf("render: %s", got)
	}
	// The sample has one item with both fields.
	sample, _ := c.TestData["items"].([]any)
	if len(sample) != 1 {
		t.Fatalf("sample: %+v", c.TestData)
	}
	if obj, _ := sample[0].(map[string]any); obj["description"] == nil || obj["amount"] == nil {
		t.Fatalf("sample item: %+v", sample[0])
	}
}

func TestMustachioPartialNeedsManualFix(t *testing.T) {
	src := strings.Repeat("<p>line</p>\n", 30) + "{{> footer}}\n"
	c := body(DialectPostmark, src)
	n := hasNote(c, NoteManual, "the Mustachio partial `{{> footer}}` has no Liquid equivalent. Use a saved section.")
	if c.Status != StatusManual || n == nil || n.Line != 31 {
		t.Fatalf("status %s notes %+v", c.Status, c.Notes)
	}
	// Kept as written, so the draft shows a Liquid error on that line.
	if !strings.Contains(c.Body, "{{> footer}}") {
		t.Fatalf("body: %s", c.Body)
	}
	if p := liquid.Validate(c.Body); len(p) == 0 || p[0].Line != 31 {
		t.Fatalf("validate: %+v", p)
	}
	if !strings.HasPrefix(c.Summary(), "the Mustachio partial") {
		t.Fatalf("summary: %q", c.Summary())
	}
}

func TestMustachioSectionsAndInverted(t *testing.T) {
	c := body(DialectPostmark, `{{#person}}Hello {{first_name}}{{/person}}{{^ years_employed}}since who knows when?{{/years_employed}}{{#position}} as a {{.}}{{/position}}`)
	want := `{% if person %}Hello {{ person.first_name }}{% endif %}{% unless years_employed %}since who knows when?{% endunless %}{% if position and position != "" %} as a {{ position }}{% endif %}`
	if c.Body != want {
		t.Fatalf("got  %s\nwant %s", c.Body, want)
	}
	if hasNote(c, NoteChange, "the section `{{#person}}` became `{% if person %}`") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	got := render(t, c.Body, map[string]any{"person": map[string]any{"first_name": "Sam"}, "position": "lead"})
	if got != "Hello Samsince who knows when? as a lead" {
		t.Fatalf("render: %q", got)
	}
	if got := render(t, c.Body, map[string]any{"years_employed": 4, "position": ""}); got != "" {
		t.Fatalf("empty render: %q", got)
	}
}

func TestMustachioPlaceholdersCommentsAndRaw(t *testing.T) {
	c := body(DialectPostmark, `{{! internal note }}{{{ body_html }}} {{& signature }} <a href="{{{ pm:unsubscribe }}}">Unsubscribe</a>`)
	want := `{% comment %}internal note{% endcomment %}{{ body_html }} {{ signature }} <a href="{{ unsubscribe_url }}">Unsubscribe</a>`
	if c.Body != want {
		t.Fatalf("got  %s\nwant %s", c.Body, want)
	}
	if hasNote(c, NoteChange, "Postmark's `{{{ pm:unsubscribe }}}` became `{{ unsubscribe_url }}`.") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if varByName(c, "unsubscribe_url") != nil {
		t.Fatal("system links aren't declared")
	}
	if p := liquid.Validate(c.Body); len(p) != 0 {
		t.Fatalf("validate: %+v", p)
	}
}

func TestHandlebarsConditionsKeepTruthiness(t *testing.T) {
	c := body(DialectSendGrid, `{{#if first_name}}Hi {{first_name}}{{else}}Hi there{{/if}},{{#if user.profile.male}} Mr{{else if user.profile.female}} Ms{{else}} Mx{{/if}}{{#unless paid}} Pay now{{/unless}}`)
	want := `{% if first_name and first_name != "" %}Hi {{ first_name }}{% else %}Hi there{% endif %},{% if user.profile.male %} Mr{% elsif user.profile.female %} Ms{% else %} Mx{% endif %}{% unless paid %} Pay now{% endunless %}`
	if c.Body != want {
		t.Fatalf("got  %s\nwant %s", c.Body, want)
	}
	if c.Status != StatusConverted {
		t.Fatalf("plain conditions convert without notes: %+v", c.Notes)
	}
	if f := varByName(c, "first_name"); f == nil || f.Required || f.Type != TypeText {
		t.Fatalf("first_name: %+v", c.Variables)
	}
	if p := varByName(c, "paid"); p == nil || p.Type != TypeBool {
		t.Fatalf("paid: %+v", c.Variables)
	}
	for _, tc := range []struct {
		data map[string]any
		want string
	}{
		{map[string]any{"first_name": "", "user": map[string]any{"profile": map[string]any{"female": true}}}, "Hi there, Ms Pay now"},
		{map[string]any{"first_name": "Sam", "paid": true}, "Hi Sam, Mx"},
	} {
		if got := render(t, c.Body, tc.data); got != tc.want {
			t.Fatalf("render %v: %q, want %q", tc.data, got, tc.want)
		}
	}
}

func TestHandlebarsListsAndLoopData(t *testing.T) {
	c := body(DialectSendGrid, `{{#if people}}<ul>{{#each people}}<li>{{@index}}. {{this.name}} ({{@root.company}}){{#if @last}}.{{/if}}</li>{{/each}}</ul>{{/if}}`)
	want := `{% if people.size > 0 %}<ul>{% for person in people %}<li>{{ forloop.index0 }}. {{ person.name }} ({{ company }}){% if forloop.last %}.{% endif %}</li>{% endfor %}</ul>{% endif %}`
	if c.Body != want {
		t.Fatalf("got  %s\nwant %s", c.Body, want)
	}
	got := render(t, c.Body, map[string]any{"company": "Acme", "people": []any{map[string]any{"name": "Sam"}, map[string]any{"name": "Lee"}}})
	if got != "<ul><li>0. Sam (Acme)</li><li>1. Lee (Acme).</li></ul>" {
		t.Fatalf("render: %q", got)
	}
	if render(t, c.Body, map[string]any{"people": []any{}}) != "" {
		t.Fatal("an empty list is false, as in Handlebars")
	}
}

func TestSendGridHelpers(t *testing.T) {
	src := "{{formatDate timeStamp \"MMMM DD, YYYY h:mm A\"}}\n{{insert name \"default=Customer\"}}\n" +
		"{{#equals customerCode winningCode}}win{{else}}lose{{/equals}}\n" +
		"{{#greaterThan (length cartItems) 0}}{{length cartItems}} items{{/greaterThan}}\n" +
		"{{#and favoriteFood favoriteDrink}}both{{/and}}{{#or isRunner isCyclist}}active{{/or}}\n" +
		"<a href=\"{{{unsubscribe}}}\">Unsubscribe</a> <a href=\"<%asm_preferences_raw_url%>\">Preferences</a>"
	c := body(DialectSendGrid, src)
	want := "{{ timeStamp | date: \"%B %d, %Y %-I:%M %p\" }}\n{{ name | default: \"Customer\" }}\n" +
		"{% if customerCode == winningCode %}win{% else %}lose{% endif %}\n" +
		"{% if cartItems.size > 0 %}{{ cartItems | size }} items{% endif %}\n" +
		"{% if favoriteFood and favoriteDrink %}both{% endif %}{% if isRunner or isCyclist %}active{% endif %}\n" +
		"<a href=\"{{ unsubscribe_url }}\">Unsubscribe</a> <a href=\"{{ preferences_url }}\">Preferences</a>"
	if c.Body != want {
		t.Fatalf("got\n%s\nwant\n%s", c.Body, want)
	}
	if ts := varByName(c, "timeStamp"); ts == nil || ts.Type != TypeDate {
		t.Fatalf("timeStamp: %+v", c.Variables)
	}
	if n := varByName(c, "name"); n == nil || n.Required || n.Default != "Customer" {
		t.Fatalf("name: %+v", c.Variables)
	}
	if ci := varByName(c, "cartItems"); ci == nil || ci.Type != TypeList {
		t.Fatalf("cartItems: %+v", c.Variables)
	}
	for _, sub := range []string{"`{{formatDate timeStamp \"MMMM DD, YYYY h:mm A\"}}` became", "`{{insert name \"default=Customer\"}}` became `{{ name | default: \"Customer\" }}`.", "SendGrid's `{{{unsubscribe}}}` became", "SendGrid's `<%asm_preferences_raw_url%>` became `{{ preferences_url }}`."} {
		if hasNote(c, NoteChange, sub) == nil {
			t.Fatalf("missing note %q in %+v", sub, c.Notes)
		}
	}
	got := render(t, c.Body, map[string]any{"timeStamp": "2026-10-15T14:05:00Z", "customerCode": "A", "winningCode": "A", "cartItems": []any{1, 2},
		"favoriteFood": true, "favoriteDrink": true, "isCyclist": true, "unsubscribe_url": "u", "preferences_url": "p"})
	if !strings.HasPrefix(got, "October 15, 2026 2:05 PM\nCustomer\nwin\n2 items\nbothactive") {
		t.Fatalf("render: %q", got)
	}
}

func TestUnknownHelperNeedsManualFix(t *testing.T) {
	c := body(DialectSendGrid, "<p>Total</p>\n<p>{{formatCurrency amount \"USD\"}}</p>")
	n := hasNote(c, NoteManual, "Handlebars helper `formatCurrency` has no Liquid equivalent.")
	if c.Status != StatusManual || n == nil || n.Line != 2 || n.Construct != `{{formatCurrency amount "USD"}}` {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if !strings.Contains(c.Body, `{{formatCurrency amount "USD"}}`) {
		t.Fatalf("kept as written: %s", c.Body)
	}
	if p := liquid.Validate(c.Body); len(p) == 0 || p[0].Line != 2 {
		t.Fatalf("validate: %+v", p)
	}
}

func TestStructureProblems(t *testing.T) {
	c := body(DialectSendGrid, "{{#if a}}\n{{#each items}}x{{/if}}\n{{/each}}")
	if hasNote(c, NoteManual, "`{{#each items}}` is never closed.") == nil || hasNote(c, NoteManual, "`{{/each}}` closes a block that was never opened.") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	c = body(DialectPostmark, "{{#each items}}{{name}}")
	if n := hasNote(c, NoteManual, "`{{#each items}}` is never closed."); n == nil || n.Line != 1 {
		t.Fatalf("notes: %+v", c.Notes)
	}
	c = body(DialectSendGrid, "Hi {{name")
	if hasNote(c, NoteManual, "is never closed with `}}`") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
}

func TestMailgunHelpersAndPlaceholders(t *testing.T) {
	src := "Hi %recipient.first_name%,\n{{#equal subscription \"free\"}}Upgrade{{/equal}}\n{{#with author}}{{firstName}} {{lastName}}{{/with}}\n<a href=\"%unsubscribe_url%\">Unsubscribe</a> %recipient_email%"
	c := body(DialectMailgun, src)
	want := "Hi {{ first_name }},\n{% if subscription == \"free\" %}Upgrade{% endif %}\n{% if author %}{{ author.firstName }} {{ author.lastName }}{% endif %}\n<a href=\"{{ unsubscribe_url }}\">Unsubscribe</a> %recipient_email%"
	if c.Body != want {
		t.Fatalf("got\n%s\nwant\n%s", c.Body, want)
	}
	if hasNote(c, NoteChange, "Mailgun's `%recipient.first_name%` became `{{ first_name }}`.") == nil ||
		hasNote(c, NoteChange, "Mailgun compares the values as text") == nil ||
		hasNote(c, NoteChange, "`{{#with author}}` became `{% if author %}`; names inside it now start with `author.`.") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if n := hasNote(c, NoteManual, "Mailgun's `%recipient_email%` has no ReputeCast equivalent."); n == nil || n.Line != 4 {
		t.Fatalf("notes: %+v", c.Notes)
	}
}

func TestWhitespaceControlAndMultilineTags(t *testing.T) {
	c := body(DialectSendGrid, "{{~#if a~}}\n  x\n{{~/if~}}\n{{#each\n  items}}{{this}}{{/each}}\nend")
	want := "{%- if a -%}\n  x\n{%- endif -%}\n{% for item in items\n %}{{ item }}{% endfor %}\nend"
	if c.Body != want {
		t.Fatalf("got  %q\nwant %q", c.Body, want)
	}
	if strings.Count(c.Body, "\n") != strings.Count("{{~#if a~}}\n  x\n{{~/if~}}\n{{#each\n  items}}{{this}}{{/each}}\nend", "\n") {
		t.Fatal("lines must line up with the source")
	}
	// "~" strips the newlines around the if, as Handlebars does.
	if got := render(t, c.Body, map[string]any{"a": true, "items": []any{"p", "q"}}); got != "xpq\nend" {
		t.Fatalf("render: %q", got)
	}
}

func TestLoopNamesNeverShadowVariables(t *testing.T) {
	c := body(DialectSendGrid, `{{item}}{{#each items}}{{name}}{{/each}}{{#each items}}{{#each tags}}{{this}}{{/each}}{{/each}}`)
	want := `{{ item }}{% for item2 in items %}{{ item2.name }}{% endfor %}{% for item2 in items %}{% for tag in item2.tags %}{{ tag }}{% endfor %}{% endfor %}`
	if c.Body != want {
		t.Fatalf("got  %s\nwant %s", c.Body, want)
	}
	if c.Variables[0].Name != "item" || c.Variables[1].Name != "items" || len(c.Variables) != 2 {
		t.Fatalf("variables: %+v", c.Variables)
	}
}

func TestBlockParamsAndOuterIndex(t *testing.T) {
	c := body(DialectSendGrid, `{{#each rows as |row i|}}{{i}}:{{row.v}}{{#each row.cells}}{{@../index}}{{/each}}{{/each}}`)
	if !strings.HasPrefix(c.Body, `{% for row in rows %}{{ forloop.index0 }}:{{ row.v }}{% for cell in row.cells %}`) {
		t.Fatalf("body: %s", c.Body)
	}
	if hasNote(c, NoteManual, "refers to an outer loop") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
}

func TestNotifuseLiquidTransactional(t *testing.T) {
	src := "Hi {{ contact.first_name | default: \"there\" }},\n" +
		"{% for item in items %}{{ item.name }}: {{ item.price | money }}{% endfor %}\n" +
		"{% render \"footer\" %}\n" +
		"<a href=\"{{ notification_center_url }}\">Preferences</a> {% assign total = items | size %}{{ total }}"
	c := Convert(Source{Ref: "receipt", Name: "Receipt", Subject: "Your receipt, {{ contact.first_name }}", Body: src}, Options{Dialect: DialectNotifuse, Transactional: true})
	if !strings.HasPrefix(c.Body, "Hi {{ first_name | default: \"there\" }},\n") || c.Subject != "Your receipt, {{ first_name }}" {
		t.Fatalf("body %q subject %q", c.Body, c.Subject)
	}
	if hasNote(c, NoteChange, "`contact.first_name` became `first_name`") == nil ||
		hasNote(c, NoteChange, "Notifuse's `notification_center_url` became `preferences_url`.") == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if n := hasNote(c, NoteManual, "the Liquid filter `money` isn't available in ReputeCast."); n == nil || n.Line != 2 {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if n := hasNote(c, NoteManual, "`{% render %}` has no ReputeCast equivalent"); n == nil || n.Line != 3 {
		t.Fatalf("notes: %+v", c.Notes)
	}
	// first_name is required by the subject (printed bare there) — the body's
	// default doesn't make the subject safe.
	names := []string{}
	for _, v := range c.Variables {
		names = append(names, v.Name)
	}
	if strings.Join(names, ",") != "first_name,items" {
		t.Fatalf("variables: %v (total is a local)", names)
	}
	if it := varByName(c, "items"); it.Type != TypeList {
		t.Fatalf("items: %+v", it)
	}
}

func TestNotifuseLiquidForCampaignsKeepsContact(t *testing.T) {
	c := Convert(Source{Body: "Hi {{ contact.first_name }}", Subject: "News"}, Options{Dialect: DialectNotifuse})
	if c.Body != "Hi {{ contact.first_name }}" || c.Status != StatusConverted {
		t.Fatalf("campaign templates keep contact fields: %q %+v", c.Body, c.Notes)
	}
}

func TestLiquidParseErrorIsManual(t *testing.T) {
	c := Convert(Source{Body: "a\n{% if plan == \"pro\" %}\nPro"}, Options{Dialect: DialectFile})
	n := hasNote(c, NoteManual, "unterminated \"if\" block in `{% if plan == \"pro\" %}`.")
	if c.Status != StatusManual || n == nil || n.Line != 2 {
		t.Fatalf("notes: %+v", c.Notes)
	}
}

func TestMomentFormats(t *testing.T) {
	cases := map[string]string{
		"MMMM DD, YYYY h:mm:ss A": "%B %d, %Y %-I:%M:%S %p",
		"YYYY-MM-DD":              "%Y-%m-%d",
		"ddd D MMM [at] HH:mm ZZ": "%a %-d %b at %H:%M %z",
		"100%":                    "100%%",
	}
	for in, want := range cases {
		if got, unknown := momentToStrftime(in); got != want || len(unknown) != 0 {
			t.Fatalf("%q: %q %v, want %q", in, got, unknown, want)
		}
	}
	if got, unknown := momentToStrftime("Do MMMM"); got != "Do %B" || len(unknown) != 1 || unknown[0] != "`Do`" {
		t.Fatalf("ordinals: %q %v", got, unknown)
	}
}

func TestIDs(t *testing.T) {
	if SuggestID("Password_Reset", "") != "password-reset" || SuggestID("", "Weekly usage summary!") != "weekly-usage-summary" || SuggestID("", "") != "template" {
		t.Fatal("suggest")
	}
	taken := map[string]bool{"password-reset": true}
	if UniqueID("password-reset", taken) != "password-reset-2" || UniqueID("password-reset", taken) != "password-reset-3" || UniqueID("receipt", taken) != "receipt" {
		t.Fatal("unique")
	}
}

func TestSensitiveAndSamples(t *testing.T) {
	c := Convert(Source{Body: `<a href="{{reset_url}}">Reset</a> {{expires_in}}`, Sample: map[string]any{"expires_in": "30 minutes"}}, Options{Dialect: DialectSendGrid})
	r, e := varByName(c, "reset_url"), varByName(c, "expires_in")
	if r == nil || !r.Sensitive || r.Type != TypeURL || e == nil || e.Sensitive || e.Sample != "30 minutes" {
		t.Fatalf("variables: %+v", c.Variables)
	}
	if c.TestData["expires_in"] != "30 minutes" || c.TestData["reset_url"] == nil {
		t.Fatalf("test data: %+v", c.TestData)
	}
}
