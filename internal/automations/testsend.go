package automations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// An automation email can be previewed as a real contact and sent as a
// test, as campaigns are (review 2026-11 AT-3).

// AutomationEmailPreviewInput picks the version and the contact.
type AutomationEmailPreviewInput struct {
	Version   int    `json:"version,omitempty" minimum:"0" doc:"Defaults to the latest version"`
	ContactID string `json:"contactId,omitempty" format:"uuid" doc:"Personalize as this contact; without one the fallbacks show"`
}

// AutomationEmailPreview is the email as the contact gets it.
type AutomationEmailPreview struct {
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	// Contact is who it was personalized as ("Maya Patel"), empty for none.
	Contact  string   `json:"contact"`
	Problems []string `json:"problems"`
}

// PreviewEmail renders an automation email as a contact gets it, its links
// pointing nowhere.
func (s *Service) PreviewEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID, in AutomationEmailPreviewInput) (AutomationEmailPreview, error) {
	links := map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"}
	return s.renderEmail(ctx, ws, id, in, links)
}

func (s *Service) renderEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID, in AutomationEmailPreviewInput, links map[string]string) (AutomationEmailPreview, error) {
	out := AutomationEmailPreview{Problems: []string{}}
	content, err := s.Email(ctx, ws, id, in.Version)
	if err != nil {
		return out, err
	}
	first, last, email := "", "", ""
	var attrs map[string]any
	if in.ContactID != "" {
		cid, err := uuid.Parse(in.ContactID)
		if err != nil {
			return out, api.NotFound("Contact")
		}
		if _, err := s.Q.AutomationContactInWorkspace(ctx, store.AutomationContactInWorkspaceParams{WorkspaceID: ws.ID, ID: cid}); err != nil {
			return out, api.MapError(err, "Contact")
		}
		c, err := s.Q.AutomationContact(ctx, cid)
		if err != nil {
			return out, api.MapError(err, "Contact")
		}
		first, last, email = c.FirstName, c.LastName, c.Email
		_ = json.Unmarshal(c.Attributes, &attrs)
		out.Contact = strings.TrimSpace(first + " " + last)
		if out.Contact == "" {
			out.Contact = email
		}
	}
	vars := liquid.Vars(email, first, last, attrs, links)
	render := func(src string) string {
		r, err := liquid.Render(src, vars)
		if err != nil {
			out.Problems = append(out.Problems, err.Error())
			return src
		}
		return r
	}
	out.Subject = render(content.Subject)
	out.Preheader = render(content.Preheader)
	out.HTML = render(content.HTML)
	return out, nil
}

// AutomationEmailTestInput is where a test goes and who it's personalized as.
type AutomationEmailTestInput struct {
	AutomationEmailPreviewInput
	To []string `json:"to" minItems:"1" maxItems:"10"`
	// AutomationID is the flow the test is for, for the audit log.
	AutomationID string `json:"automationId,omitempty" maxLength:"40"`
}

// AutomationEmailTestResult says where the test went.
type AutomationEmailTestResult struct {
	Sent     []string `json:"sent"`
	Provider string   `json:"provider"`
}

// TestEmail sends an automation email through the System route with
// "[Test]" before its subject; it counts in no stats. Without the
// campaigns:send permission a member can send tests only to their own
// address, as with campaigns.
func (s *Service) TestEmail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AutomationEmailTestInput) (AutomationEmailTestResult, error) {
	l := langOf(ctx)
	to := []string{}
	seen := map[string]bool{}
	for _, raw := range in.To {
		addr, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return AutomationEmailTestResult{}, api.Invalid("body.to", say(l, txt(mProblemTestTo)))
		}
		e := strings.ToLower(addr.Address)
		if seen[e] {
			continue
		}
		seen[e] = true
		if p == nil || (!strings.EqualFold(e, p.Email) && !p.Can(authz.CampaignsSend)) {
			return AutomationEmailTestResult{}, api.Forbidden(authz.CampaignsSend, nil).WithMeta("reason", say(l, txt(mProblemTestOwn)))
		}
		to = append(to, e)
	}
	links := map[string]string{
		"unsubscribe_url":     s.URLs.Unsubscribe(ws.Slug, "test"),
		"preferences_url":     s.URLs.Preferences(ws.Slug, "test"),
		"view_in_browser_url": s.URLs.ViewOnline(ws.Slug, "test"),
	}
	pv, err := s.renderEmail(ctx, ws, id, in.AutomationEmailPreviewInput, links)
	if err != nil {
		return AutomationEmailTestResult{}, err
	}
	sender := mailer.Sender(mailer.Unavailable{})
	if s.Mail != nil {
		if m := s.Mail(); m != nil {
			sender = m
		}
	}
	if !sender.SystemAvailable(ctx, &ws.ID) {
		return AutomationEmailTestResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable, say(l, txt(mProblemNoSystemRoute)))
	}
	provider := ""
	for _, addr := range to {
		name, err := sender.SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: addr, Subject: "[Test] " + pv.Subject, HTML: pv.HTML, Text: contentcheck.PlainText(pv.HTML), Kind: "automation_test"})
		if err != nil {
			if errors.Is(err, mailer.ErrNoSystemProvider) {
				return AutomationEmailTestResult{}, api.Err(http.StatusConflict, api.CodeProviderUnavailable, say(l, txt(mProblemNoSystemRoute)))
			}
			return AutomationEmailTestResult{}, api.Err(http.StatusBadGateway, api.CodeProviderUnavailable, say(l, txt(mProblemTestRefused, "error", err.Error())))
		}
		provider = name
	}
	after := map[string]any{"to": to, "provider": provider}
	if in.ContactID != "" {
		after["contactId"] = in.ContactID
	}
	if in.AutomationID != "" {
		after["automationId"] = in.AutomationID
	}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "automation.test_send", TargetType: "automation_email", TargetID: id.String(), After: after})
	return AutomationEmailTestResult{Sent: to, Provider: provider}, err
}
