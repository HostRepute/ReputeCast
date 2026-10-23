package publicpages

// Previews of Grow's public pages for the app (Referrals › Settings, the
// set-up guide): the invite page, a referrer's hub, the leaderboard and the
// recommendations page, rendered for the referrer with the most signups (or
// a reader picked in the app). Like the other previews they are inert and
// never write anything: no visit is recorded, no link works.

import (
	"context"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

type growPreviewInput struct {
	api.WorkspaceInput
	Page    string `path:"page" enum:"invite,hub,leaderboard,recommends"`
	Program string `query:"program" doc:"The program (the first one by default)"`
}

// registerGrowPreviews adds the Grow previews to the public page previews.
func registerGrowPreviews(h huma.API, p *Handler) {
	api.Get(h, api.Op{ID: "preview-grow-page", Path: "/workspaces/{workspace}/grow/previews/{page}",
		Summary:     "Preview a Grow public page: the invite page, a referrer's hub, the leaderboard or the recommendations page",
		Description: "Rendered for the referrer with the most signups. Inert: links and buttons don't work and nothing is recorded.",
		Tags:        []string{"Grow"}, Access: api.Workspace(authz.GrowView)},
		func(ctx context.Context, in *growPreviewInput) (*api.Body[PublicPagePreview], error) {
			out, err := p.GrowPreview(ctx, api.WorkspaceFrom(ctx), in.Page, in.Program)
			return api.OK(out), err
		})
}

// GrowPreview renders one of Grow's public pages for the app.
func (h *Handler) GrowPreview(ctx context.Context, ws *store.Workspace, page, program string) (PublicPagePreview, error) {
	if err := h.Init(); err != nil {
		return PublicPagePreview{}, err
	}
	rs := h.referrals()
	if rs == nil {
		return PublicPagePreview{}, appNotFound(ctx, "growPage")
	}
	set, err := h.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return PublicPagePreview{}, err
	}
	s := &site{ws: ws, set: set, links: forms.LinksOn(h.URLs.Base(), ws.Slug, false), canon: h.Forms.Links(ws, set), preview: true}
	// The page as most readers see it: in the default language.
	h.fixedLang(ctx, s, "", nil)
	var programID *uuid.UUID
	if program != "" {
		id, err := uuid.Parse(program)
		if err != nil {
			return PublicPagePreview{}, appInvalid(ctx, "program", "pickProgram")
		}
		programID = &id
	}
	// The referrer the page is for: the one with the most signups.
	var contactID, linkID, progID uuid.UUID
	var code string
	err = h.Pool.QueryRow(ctx, `SELECT rl.contact_id, rl.id, rl.code, rl.program_id FROM referral_links rl
		JOIN referral_programs p ON p.id = rl.program_id AND p.deleted_at IS NULL
		WHERE rl.workspace_id = $1 AND ($2::uuid IS NULL OR rl.program_id = $2)
		ORDER BY rl.signups DESC, rl.visits DESC, rl.created_at LIMIT 1`, ws.ID, programID).Scan(&contactID, &linkID, &code, &progID)
	noLink := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !noLink {
		return PublicPagePreview{}, err
	}
	out := PublicPagePreview{}
	var v *view
	tmpl := "grow-" + page
	switch page {
	case "invite":
		if noLink {
			return out, appNotFound(ctx, "referralLink")
		}
		iv, err := rs.Invite(ctx, ws, code)
		if err != nil {
			return out, err
		}
		if iv == nil {
			return out, appNotFound(ctx, "referralLink")
		}
		c, hosted, err := h.joinForm(ctx, s, iv)
		if err != nil {
			return out, err
		}
		fl, err := h.formLangIn(ctx, s, c, "")
		if err != nil {
			return out, err
		}
		b, err := h.joinBodyWith(ctx, s, iv, code, c, hosted, fl, nil, forms.Outcome{})
		if err != nil {
			return out, err
		}
		v = h.view(s, b.Heading, b)
		v.Description = b.Description
		h.setLangCtx(v, ctx, s, fl.pageLang)
		tmpl = "grow-join"
		out.URL = s.canon.Join(code)
		out.Contact = code
	case "hub":
		if noLink {
			return out, appNotFound(ctx, "referralLink")
		}
		hv, err := rs.Hub(ctx, ws, contactID, previewToken, nil)
		if err != nil {
			return out, err
		}
		v = h.growView(nil, s, s.gt("hub.title", "publisher", s.set.Branding.PublisherName), h.hubBody(s, hv, previewToken))
		out.Contact = hv.Email
	case "leaderboard":
		var list string
		var started *time.Time
		var pid uuid.UUID
		if err := h.Pool.QueryRow(ctx, `SELECT p.id, l.name, p.first_on_at FROM referral_programs p JOIN lists l ON l.id = p.list_id
			WHERE p.workspace_id = $1 AND p.deleted_at IS NULL AND ($2::uuid IS NULL OR p.id = $2) ORDER BY p.created_at LIMIT 1`, ws.ID, programID).
			Scan(&pid, &list, &started); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return out, appNotFound(ctx, "referralProgram")
			}
			return out, err
		}
		p, err := h.Q.ReferralProgramByID(ctx, pid)
		if err != nil {
			return out, err
		}
		b, err := h.boardBody(ctx, s, rs, p, list, started, false, nil, "", "")
		if err != nil {
			return out, err
		}
		v = h.growView(nil, s, s.gt("board.title")+" · "+s.set.Branding.PublisherName, b)
		tmpl = "grow-board"
		out.URL = s.canon.Leaderboard()
	case "recommends":
		if h.recommendations() == nil {
			return out, appNotFound(ctx, "recommendsPage")
		}
		reader := contactID
		if noLink {
			reader = uuid.Nil
		}
		b, pg, err := h.recommendsBody(ctx, s, reader, previewToken)
		if err != nil {
			return out, err
		}
		b.Items = recViews(s, pg.Items, pg.Email, "", "")
		if !b.HasPartner {
			b.Ask = false
		}
		v = h.growView(nil, s, b.Heading, b)
		out.URL = s.canon.Recommends("")
		out.Contact = pg.Email
	default:
		return out, appNotFound(ctx, "growPage")
	}
	v.Preview = true
	html, err := h.html(tmpl, v)
	if err != nil {
		return out, err
	}
	out.HTML = string(html)
	return out, nil
}
