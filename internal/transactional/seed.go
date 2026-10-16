package transactional

import (
	"github.com/google/uuid"
)

// Helpers for the demo seeder (internal/demo), which writes Acme Cloud's
// templates, versions and messages straight into its transaction: they keep
// its content, hashes and stored renders the same as the template API's.

// DemoTemplate is a template version's content.
type DemoTemplate struct {
	Subject, Preheader string
	MJML, HTML         string
	Variables          []TransactionalVariable
}

// Compile fills HTML from the MJML, as saving a draft does.
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
