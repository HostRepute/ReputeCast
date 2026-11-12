package forms

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Multi-step forms (M60 G-23): a form shows its fields one step at a time
// (at most MaxSteps), each step with its own heading, description and
// button; everything is posted once, from the last step, which carries the
// lists, the consent box and the submit button. Surfaces without the embed
// script (the hosted page, the plain HTML form, page blocks, Grow's join
// page) show every step at once, each as a fieldset.
//
// A choice question asks the visitor to pick one or several answers, each
// of which adds its tag to the contact (so tag triggers start automations).

// MaxSteps is the most steps a form can have.
const MaxSteps = 4

// FieldChoice is the type of a choice question (fieldDoc.Type).
const FieldChoice = "choice"

// Choice questions: at most this many answers each.
const maxChoiceOptions = 12

// stepDoc is one step of a multi-step form (contentDoc.Steps, variantDoc.Steps).
type stepDoc struct {
	ID          string   `json:"id"`
	Fields      []string `json:"fields"`
	Heading     string   `json:"heading,omitempty"`
	Description string   `json:"description,omitempty"`
	Button      string   `json:"button,omitempty"`
}

// optionDoc is one answer of a choice question.
type optionDoc struct {
	Label string `json:"label"`
	Tag   string `json:"tag"`
}

// SignupFormStep is one step of a multi-step form.
type SignupFormStep struct {
	ID string `json:"id" pattern:"^s[1-4]$" doc:"s1 to s4, kept when steps move so their translations stay"`
	// Fields are the keys of the step's fields, in order.
	Fields      []string `json:"fields" maxItems:"20"`
	Heading     string   `json:"heading,omitempty" maxLength:"120"`
	Description string   `json:"description,omitempty" maxLength:"500"`
	// Button is the step's button; empty: "Next" before the last step, the
	// form's button on the last.
	Button string `json:"button,omitempty" maxLength:"40"`
}

// SignupFormChoiceOption is one answer of a choice question.
type SignupFormChoiceOption struct {
	Label string `json:"label" minLength:"1" maxLength:"60"`
	// Tag is added to the contact who picks the answer.
	Tag string `json:"tag" minLength:"1" maxLength:"60"`
}

var stepIDPattern = regexp.MustCompile(`^s[1-4]$`)

// stepPartMax is the longest each part of a step's copy may be.
var stepPartMax = map[string]int{"heading": 120, "description": 500, "button": 40}

func stepsOut(steps []stepDoc) []SignupFormStep {
	out := make([]SignupFormStep, 0, len(steps))
	for _, s := range steps {
		out = append(out, SignupFormStep{ID: s.ID, Fields: slices.Clone(s.Fields), Heading: s.Heading, Description: s.Description, Button: s.Button})
	}
	return out
}

func optionsOut(opts []optionDoc) []SignupFormChoiceOption {
	out := make([]SignupFormChoiceOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, SignupFormChoiceOption(o))
	}
	return out
}

// parseSteps checks a form's (or a variant's) steps against its fields:
// at least two and at most MaxSteps, every field in exactly one step. A
// single step is no steps. loc is where errors point ("body.content.steps").
func parseSteps(ctx context.Context, in []SignupFormStep, fields []fieldDoc, loc string) ([]stepDoc, error) {
	if len(in) <= 1 {
		return nil, nil
	}
	if len(in) > MaxSteps {
		return nil, api.Invalid(loc, say(ctx, mStepsMax, i18n.Args{"max": MaxSteps}))
	}
	known := map[string]bool{}
	for _, f := range fields {
		known[f.Key] = true
	}
	placed := map[string]bool{}
	ids := map[string]bool{}
	out := make([]stepDoc, 0, len(in))
	for i, s := range in {
		at := fmt.Sprintf("%s[%d]", loc, i)
		id := s.ID
		if !stepIDPattern.MatchString(id) || ids[id] {
			return nil, api.Invalid(at+".id", say(ctx, mStepID, nil))
		}
		ids[id] = true
		d := stepDoc{ID: id, Fields: []string{}, Heading: strings.TrimSpace(s.Heading), Description: strings.TrimSpace(s.Description), Button: strings.TrimSpace(s.Button)}
		for _, k := range s.Fields {
			if !known[k] || placed[k] {
				return nil, api.Invalid(at+".fields", say(ctx, mStepsEveryField, nil))
			}
			placed[k] = true
			d.Fields = append(d.Fields, k)
		}
		if len(d.Fields) == 0 {
			return nil, api.Invalid(at+".fields", say(ctx, mStepEmpty, nil))
		}
		out = append(out, d)
	}
	if !placed[FieldEmail] {
		return nil, api.Invalid(loc, say(ctx, mStepsEmail, nil))
	}
	for k := range known {
		if !placed[k] {
			return nil, api.Invalid(loc, say(ctx, mStepsEveryField, nil))
		}
	}
	return out, nil
}

// fitSteps keeps steps in line with the form's fields after they change:
// fields gone are dropped, new fields join the last step, empty steps go,
// and fewer than two steps is one.
func fitSteps(steps []stepDoc, fields []fieldDoc) []stepDoc {
	if len(steps) == 0 {
		return nil
	}
	known := map[string]bool{}
	for _, f := range fields {
		known[f.Key] = true
	}
	placed := map[string]bool{}
	out := make([]stepDoc, 0, len(steps))
	for _, s := range steps {
		s.Fields = slices.DeleteFunc(slices.Clone(s.Fields), func(k string) bool {
			if !known[k] || placed[k] {
				return true
			}
			placed[k] = true
			return false
		})
		out = append(out, s)
	}
	for _, f := range fields {
		if !placed[f.Key] {
			out[len(out)-1].Fields = append(out[len(out)-1].Fields, f.Key)
		}
	}
	out = slices.DeleteFunc(out, func(s stepDoc) bool { return len(s.Fields) == 0 })
	if len(out) < 2 {
		return nil
	}
	return out
}

// parseChoice checks a choice question of the builder.
func parseChoice(ctx context.Context, f SignupFormField, loc string) (fieldDoc, error) {
	label := strings.TrimSpace(f.Label)
	if label == "" {
		return fieldDoc{}, api.Invalid(loc+".label", say(ctx, mChoiceLabel, nil))
	}
	if len(f.Options) < 2 || len(f.Options) > maxChoiceOptions {
		return fieldDoc{}, api.Invalid(loc+".options", say(ctx, mChoiceOptions, i18n.Args{"max": maxChoiceOptions}))
	}
	d := fieldDoc{Key: f.Key, Label: label, Required: f.Required, Type: FieldChoice, Multiple: f.Multiple}
	tags := map[string]bool{}
	for i, o := range f.Options {
		l, t := strings.TrimSpace(o.Label), cleanTag(o.Tag)
		if l == "" || t == "" {
			return fieldDoc{}, api.Invalid(fmt.Sprintf("%s.options[%d]", loc, i), say(ctx, mChoiceOption, nil))
		}
		if tags[t] {
			return fieldDoc{}, api.Invalid(fmt.Sprintf("%s.options[%d].tag", loc, i), say(ctx, mChoiceTagTwice, i18n.Args{"tag": t}))
		}
		tags[t] = true
		d.Options = append(d.Options, optionDoc{Label: l, Tag: t})
	}
	return d, nil
}

// cleanTag is a tag as contacts keep it: trimmed, lower case, at most 60.
func cleanTag(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if len(t) > 60 {
		return ""
	}
	return t
}

// ChoiceName is the name of a choice question in a form post ("c.<key>");
// its values are the indexes of the picked answers.
func ChoiceName(key string) string { return "c." + key }

// choiceTags turns the picked answers (indexes) into the tags they add.
func choiceTags(f fieldDoc, picked []string) []string {
	var out []string
	for _, v := range picked {
		i, err := strconv.Atoi(v)
		if err != nil || i < 0 || i >= len(f.Options) {
			continue
		}
		if t := f.Options[i].Tag; !slices.Contains(out, t) {
			out = append(out, t)
		}
		if !f.Multiple {
			break
		}
	}
	return out
}

// choiceLabels are the picked answers' labels, for the form.submitted event.
func choiceLabels(f fieldDoc, picked []string) []string {
	var out []string
	for _, v := range picked {
		i, err := strconv.Atoi(v)
		if err != nil || i < 0 || i >= len(f.Options) {
			continue
		}
		out = append(out, f.Options[i].Label)
		if !f.Multiple {
			break
		}
	}
	return out
}

// StepOf is a step as a page shows it: its copy and its fields, in order.
type StepOf struct {
	ID, Heading, Description, Button string
	Fields                           []fieldDoc
}

// FormSteps is the form's steps as pages show them (nil: one step).
func (c Config) FormSteps() []StepOf {
	if len(c.Content.Steps) < 2 {
		return nil
	}
	out := make([]StepOf, 0, len(c.Content.Steps))
	for _, s := range c.Content.Steps {
		st := StepOf{ID: s.ID, Heading: s.Heading, Description: s.Description, Button: s.Button}
		for _, k := range s.Fields {
			if f, ok := c.Field(k); ok {
				st.Fields = append(st.Fields, f)
			}
		}
		out = append(out, st)
	}
	return out
}
