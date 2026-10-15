package transactional

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// content is what a template version renders from.
type content struct {
	Subject    string
	Preheader  string
	EditorMode string
	MJML       string
	HTML       string
	Text       string
	Variables  []TransactionalVariable
}

// rendered is one message's subject, preheader, HTML and text.
type rendered struct {
	Subject   string
	Preheader string
	HTML      string
	Text      string
}

// recipient is who a message goes to (the to.email and to.name variables).
type recipient struct {
	Email string
	Name  string
}

// compiledHTML returns the template's HTML: what the editor compiled, or
// the MJML compiled on the server (gomjml) with the subject and preheader
// in its head.
func (c content) compiledHTML() (string, error) {
	if strings.TrimSpace(c.HTML) != "" {
		return c.HTML, nil
	}
	if strings.TrimSpace(c.MJML) == "" {
		return "", nil
	}
	res := mjml.Compile(mjml.SetHeadMeta(c.MJML, c.Subject, c.Preheader))
	if len(res.Errors) > 0 {
		return "", fmt.Errorf("the MJML doesn't compile: %s", res.Errors[0].Message)
	}
	return res.HTML, nil
}

// bindings are the variables a render sees: what the call sent, the
// declared defaults of optional variables it didn't send, the recipient and
// the links ReputeCast fills in.
func bindings(vars []TransactionalVariable, data map[string]any, to recipient, links map[string]string) map[string]any {
	b := make(map[string]any, len(data)+len(vars)+3)
	for k, v := range data {
		b[k] = v
	}
	for _, v := range vars {
		if val, ok := b[v.Name]; ok && val != nil {
			continue
		}
		if v.Default == "" {
			b[v.Name] = nil
			continue
		}
		if val, err := typedValue(v.Type, v.Default); err == nil {
			b[v.Name] = val
		}
	}
	b["to"] = map[string]any{"email": to.Email, "name": to.Name}
	for k, v := range links {
		b[k] = v
	}
	return b
}

// renderError is a template that can't render: the part and the message.
type renderError struct {
	Part    string
	Message string
}

func (e *renderError) Error() string { return e.Part + ": " + e.Message }

// render renders a template for one message. Missing required variables
// are the caller's to check (missing_variables) before calling it.
func render(c content, data map[string]any, to recipient, links map[string]string) (rendered, error) {
	vars := bindings(c.Variables, data, to, links)
	var out rendered
	var err error
	if out.Subject, err = renderPart("Subject", c.Subject, vars); err != nil {
		return out, err
	}
	out.Subject = strings.Join(strings.Fields(out.Subject), " ")
	if out.Preheader, err = renderPart("Preheader", c.Preheader, vars); err != nil {
		return out, err
	}
	body, err := c.compiledHTML()
	if err != nil {
		return out, &renderError{Part: "Body", Message: err.Error()}
	}
	if out.HTML, err = renderPart("Body", body, vars); err != nil {
		return out, err
	}
	if c.EditorMode == "html" && out.Preheader != "" && !strings.Contains(out.HTML, out.Preheader) {
		out.HTML = withPreheader(out.HTML, out.Preheader)
	}
	if strings.TrimSpace(c.Text) != "" {
		if out.Text, err = renderPart("Text", c.Text, vars); err != nil {
			return out, err
		}
	} else {
		out.Text = contentcheck.PlainText(out.HTML)
	}
	return out, nil
}

func renderPart(part, src string, vars map[string]any) (string, error) {
	out, err := liquid.Render(src, vars)
	if err != nil {
		if errors.Is(err, liquid.ErrTooLarge) {
			return "", &renderError{Part: part, Message: "The rendered email is larger than 4 MB."}
		}
		return "", &renderError{Part: part, Message: err.Error()}
	}
	return out, nil
}

var bodyOpenRe = regexp.MustCompile(`(?i)<body[^>]*>`)

// withPreheader adds the hidden preview text inboxes show after the subject.
func withPreheader(doc, preheader string) string {
	div := `<div style="display:none;font-size:1px;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;mso-hide:all">` + html.EscapeString(preheader) + `</div>`
	if loc := bodyOpenRe.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + div + doc[loc[1]:]
	}
	return div + doc
}

// liquidMessage turns a parse error into the editor's sentence: "The
// {% if %} on line 14 has no {% endif %}."
func liquidMessage(p liquid.LiquidProblem) string {
	msg := p.Message
	if m := unterminatedRe.FindStringSubmatch(msg); m != nil {
		tag := m[1]
		if p.Line > 0 {
			return fmt.Sprintf("The {%% %s %%} on line %d has no {%% end%s %%}.", tag, p.Line, tag)
		}
		return fmt.Sprintf("A {%% %s %%} has no {%% end%s %%}.", tag, tag)
	}
	return msg
}

var unterminatedRe = regexp.MustCompile(`(?i)unterminated "(\w+)" block`)
