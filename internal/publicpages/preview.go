package publicpages

// Previews of the public pages for the app's Forms & public pages tabs: the
// same templates, rendered for the subscriber who confirmed last (or an
// example), with unsaved options laid over the saved ones. Previews are
// inert (no links, no submits) and never write anything.

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// PublicPagePreview is a public page rendered for the app.
type PublicPagePreview struct {
	// HTML is the page as subscribers see it; links and buttons are inert.
	HTML string `json:"html"`
	// URL is where the page lives (the custom domain once verified); empty
	// for pages only signed links open (confirmation, unsubscribe).
	URL string `json:"url"`
	// Contact is whose page it is: the subscriber who confirmed last, or an example address.
	Contact string `json:"contact"`
}

type previewInput struct {
	api.WorkspaceInput
	Page    string `path:"page" enum:"subscribe,confirm,preferences,unsubscribe,archive"`
	Form    string `query:"form" doc:"Show a form's hosted page (drafts too) instead of the subscribe page"`
	Draft   string `query:"draft" maxLength:"20000" doc:"Unsaved page options to show: the body of save-public-pages, as JSON"`
	Contact string `query:"contact" maxLength:"320" doc:"Preview the pages of this subscriber"`
}

// RegisterAPI adds the preview operation.
func RegisterAPI(h huma.API, p *Handler) {
	api.Get(h, api.Op{ID: "preview-public-page", Path: "/workspaces/{workspace}/public-pages/preview/{page}",
		Summary: "Preview a public page as a subscriber sees it", Tags: []string{"Public pages"}, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *previewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.Preview(ctx, api.WorkspaceFrom(ctx), in.Page, in.Form, in.Draft, in.Contact)
			return api.OK(out), err
		})
	// The poll answer page and the survey page (polls_preview.go).
	registerPollPreviews(h, p)
}

// Preview renders one public page for the app.
func (h *Handler) Preview(ctx context.Context, ws *store.Workspace, page, form, draft, contact string) (PublicPagePreview, error) {
	if err := h.Init(); err != nil {
		return PublicPagePreview{}, err
	}
	set, err := h.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	if draft != "" {
		var d forms.PublicPageInput
		if err := json.Unmarshal([]byte(draft), &d); err != nil {
			return PublicPagePreview{}, api.Invalid("draft", "The unsaved options couldn't be read. Reload the page.")
		}
		if set, err = h.Forms.WithDraft(ctx, set, d); err != nil {
			return PublicPagePreview{}, err
		}
	}
	// Assets and links point at the instance, which serves them on every
	// host; URL names the page's canonical address.
	s := &site{ws: ws, set: set, links: forms.LinksOn(h.URLs.Base(), ws.Slug, false), canon: h.Forms.Links(ws, set), preview: true}
	out := PublicPagePreview{}
	var v *view
	tmpl := page
	switch page {
	case "subscribe":
		c, hosted, err := h.previewForm(ctx, s, form)
		if err != nil {
			return out, err
		}
		body, err := h.subscribeBody(ctx, s, c, hosted, nil, forms.Outcome{})
		if err != nil {
			return out, err
		}
		v = h.view(s, subscribeTitle(body.Heading), body)
		out.URL = s.canon.Subscribe(slugOf(c, hosted))
	case "archive":
		if v, _, err = h.archiveView(ctx, s, url.Values{}, true); err != nil {
			return out, err
		}
		out.URL = s.canon.Archive()
	default:
		pc, err := h.previewContact(ctx, s, contact)
		if err != nil {
			return out, err
		}
		out.Contact = pc.Email
		claims := linktoken.Claims{WorkspaceID: ws.ID, ContactID: pc.ID}
		publisher := s.set.Branding.PublisherName
		switch page {
		case "confirm":
			body, err := h.confirmBody(ctx, s, pc)
			if err != nil {
				return out, err
			}
			v = h.view(s, "You're in · "+publisher, body)
		case "preferences":
			token := h.sign(s, linktoken.Preferences, pc.ID, uuid.Nil)
			body, err := h.preferencesBody(ctx, s, pc, s.links.Preferences(token), claims)
			if err != nil {
				return out, err
			}
			v = h.view(s, "Email preferences · "+publisher, body)
			out.URL = s.canon.PreferencesRequest()
		case "unsubscribe":
			token := h.sign(s, linktoken.Unsubscribe, pc.ID, uuid.Nil)
			body, err := h.unsubscribeBody(ctx, s, pc, claims, token)
			if err != nil {
				return out, err
			}
			v = h.view(s, "Unsubscribe · "+publisher, body)
		default:
			return out, api.NotFound("Public page")
		}
	}
	v.Preview = true
	html, err := h.html(tmpl, v)
	if err != nil {
		return out, err
	}
	out.HTML = string(html)
	return out, nil
}

// previewForm is the form a subscribe preview shows: one picked in the app
// (drafts included), else the subscribe page's.
func (h *Handler) previewForm(ctx context.Context, s *site, form string) (forms.Config, bool, error) {
	if form == "" {
		return h.resolveForm(ctx, s, "", true)
	}
	id, err := uuid.Parse(form)
	if err != nil {
		return forms.Config{}, false, api.NotFound("Form")
	}
	f, err := h.Q.FormGet(ctx, store.FormGetParams{WorkspaceID: s.ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return forms.Config{}, false, api.NotFound("Form")
	}
	if err != nil {
		return forms.Config{}, false, err
	}
	hosted := s.set.HostedFormID() != nil && *s.set.HostedFormID() == f.ID
	return forms.ConfigOf(f), hosted, nil
}

// previewContact is the subscriber a preview is for: the address asked
// for, the workspace's "preview as" address, else the latest confirmation.
func (h *Handler) previewContact(ctx context.Context, s *site, email string) (audience.PublicContact, error) {
	id, ok, err := h.Audience.PreviewContact(ctx, h.Q, s.ws, email)
	if err != nil {
		return audience.PublicContact{}, err
	}
	if !ok && email != "" {
		return audience.PublicContact{}, api.Invalid("contact", "No contact has that address. Leave it empty to preview with the latest subscriber.")
	}
	if email == "" && s.set.Pages.PreviewEmail != "" {
		if pid, found, err := h.Audience.FindPublicContact(ctx, h.Q, s.ws, s.set.Pages.PreviewEmail); err == nil && found {
			id, ok = pid, true
		}
	}
	if !ok {
		lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
		if err != nil {
			return audience.PublicContact{}, err
		}
		return audience.SamplePublicContact(lists), nil
	}
	return h.Audience.LoadPublicContact(ctx, h.Q, s.ws, id)
}
