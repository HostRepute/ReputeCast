package transactional

import (
	"strings"

	"github.com/google/uuid"
)

// Helpers for the demo seeder (internal/demo), which writes Acme Cloud's
// templates, versions and messages straight into its transaction.

// DemoTemplate is a template version's content.
type DemoTemplate struct {
	Key, Name, Description, Kind string
	Subject, Preheader           string
	MJML, HTML                   string
	Variables                    []TransactionalVariable
	TestData                     []TransactionalTestData
}

// usageSummary is the demo's weekly usage summary, an Optional template
// that isn't offered as a starter.
var usageSummary = TransactionalStarter{
	Key: "usage-summary", Name: "Weekly usage summary", Description: "Your week in numbers, every Monday", Kind: KindOptional,
	Subject: "Your week in {{product}}", Preheader: "{{ api_requests }} API requests across {{ projects }} projects.",
	Variables: []TransactionalVariable{
		{Name: "first_name", Type: TypeText, Sample: "Alex", Default: "there"},
		{Name: "week_label", Type: TypeText, Required: true, Sample: "5 to 11 Oct"},
		{Name: "projects", Type: TypeNumber, Required: true, Sample: "4"},
		{Name: "api_requests", Type: TypeNumber, Required: true, Sample: "182400"},
		{Name: "dashboard_url", Type: TypeURL, Required: true, Sample: "https://app.example.com/usage"},
	},
	TestData: []TransactionalTestData{
		{Name: "Busy week", Data: map[string]any{"first_name": "Alex", "week_label": "5 to 11 Oct", "projects": 4, "api_requests": 182400, "dashboard_url": "https://app.example.com/usage"}},
	},
	heading: "Your week in {{product}}",
	body:    []string{"Hi {{ first_name }},", "In the week of {{ week_label }}, your {{ projects }} projects made {{ api_requests }} API requests."},
	button:  "Open your dashboard", href: "{{ dashboard_url }}",
	after: "You get this summary every Monday. Turn it off below if you'd rather not.",
}

// DemoTemplateContent builds a template's content from its starter, with
// the product name and the footer's address filled in.
func DemoTemplateContent(key, product, address string) (DemoTemplate, error) {
	sd, ok := starterByKey(key)
	if key == usageSummary.Key {
		sd, ok = usageSummary, true
	}
	if !ok {
		return DemoTemplate{}, errNoStarter(key)
	}
	t := DemoTemplate{Key: sd.Key, Name: sd.Name, Description: sd.Description, Kind: sd.Kind, Subject: fillProduct(sd.Subject, product),
		Preheader: fillProduct(sd.Preheader, product), MJML: sd.mjmlFor(product, address), Variables: sd.Variables, TestData: sd.TestData}
	return t, t.Compile()
}

// Compile fills HTML from the MJML.
func (t *DemoTemplate) Compile() error {
	html, err := content{Subject: t.Subject, Preheader: t.Preheader, MJML: t.MJML}.compiledHTML()
	t.HTML = html
	return err
}

// DemoHash is the content hash a published version keeps, equal to its
// draft's when nothing changed since.
func DemoHash(t DemoTemplate, sender *uuid.UUID, replyTo string, opens, clicks bool) string {
	return draftHash(t.Subject, t.Preheader, "visual", t.MJML, t.HTML, "", t.Variables, sender, replyTo, opens, clicks)
}

// DemoRender renders what a message's log keeps: sensitive values masked.
func DemoRender(t DemoTemplate, data map[string]any, toEmail, toName string, links map[string]string) (subject, html, text string, err error) {
	c := content{Subject: t.Subject, Preheader: t.Preheader, MJML: t.MJML, HTML: t.HTML, Variables: t.Variables}
	out, err := render(c, maskData(t.Variables, data), recipient{Email: toEmail, Name: toName}, links)
	if err != nil {
		return "", "", "", err
	}
	return out.Subject, out.HTML, out.Text, nil
}

type errNoStarter string

func (e errNoStarter) Error() string { return "no starter called " + strings.TrimSpace(string(e)) }
