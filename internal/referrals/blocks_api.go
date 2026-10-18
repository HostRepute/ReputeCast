package referrals

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/store"
)

// GrowBlockRenderInput renders a Grow block for the composer's canvas.
type GrowBlockRenderInput struct {
	Kind     string          `json:"kind" enum:"referral,reward"`
	Settings json.RawMessage `json:"settings,omitempty"`
	// ContactID previews the block as a reader; without it, a sample.
	ContactID string `json:"contactId,omitempty" format:"uuid"`
	// MilestoneID is the milestone email a Reward block is in.
	MilestoneID string `json:"milestoneId,omitempty" format:"uuid"`
}

// GrowBlockRender is a block's HTML for the canvas, and why it is hidden.
type GrowBlockRender struct {
	HTML string `json:"html"`
	// Hidden: none (it shows); not_on_list (the reader isn't on the
	// program's list, or their link is paused); program_off; no_program.
	Hidden    string     `json:"hidden" enum:"none,not_on_list,program_off,no_program"`
	ProgramID string     `json:"programId"`
	ListName  string     `json:"listName"`
	OffAt     *time.Time `json:"offAt,omitempty"`
	OffBy     string     `json:"offBy"`
	// For is who the block was rendered for ("" for the sample).
	For string `json:"for"`
}

// GrowPreviewContact is a reader the composer previews Grow blocks as.
type GrowPreviewContact struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// OnList: on the program's list with an active link (the block shows).
	OnList  bool  `json:"onList"`
	Counted int64 `json:"counted"`
	// Lists names the reader's lists ("Product Updates").
	Lists []string `json:"lists"`
}

// GrowPreviewContacts are the readers to preview Grow blocks as: the
// program's top referrers and a few readers outside it.
type GrowPreviewContacts struct {
	Items     []GrowPreviewContact `json:"items"`
	ProgramID string               `json:"programId"`
	ListName  string               `json:"listName"`
	Status    string               `json:"status" enum:"on,off,"`
}

// blockProgram is the program a block reads: the one it's bound to, or
// the workspace's first.
func (s *Service) blockProgram(ctx context.Context, ws *store.Workspace, bound string) (*store.ReferralProgramGetRow, error) {
	if id, err := uuid.Parse(bound); err == nil {
		r, err := s.Q.ReferralProgramGet(ctx, store.ReferralProgramGetParams{WorkspaceID: ws.ID, ID: id})
		if err == nil && r.ReferralProgram.DeletedAt == nil {
			return &r, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	progs, err := s.Q.ReferralProgramsOf(ctx, ws.ID)
	if err != nil || len(progs) == 0 {
		return nil, err
	}
	r, err := s.Q.ReferralProgramGet(ctx, store.ReferralProgramGetParams{WorkspaceID: ws.ID, ID: progs[0].ID})
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// RenderBlock renders a Referral or Reward block for the canvas, as a
// reader ("Preview as") or as the sample: 2 counted, 1 counting soon.
func (s *Service) RenderBlock(ctx context.Context, ws *store.Workspace, in GrowBlockRenderInput) (GrowBlockRender, error) {
	out := GrowBlockRender{Hidden: "none"}
	block := parseReferralBlock(in.Settings)
	var milestone *store.ReferralMilestone
	if in.Kind == "reward" {
		mid, err := uuid.Parse(in.MilestoneID)
		if err != nil {
			return out, api.Invalid("body.milestoneId", "A Reward block belongs to a milestone email.")
		}
		m, err := s.Q.ReferralMilestoneByID(ctx, mid)
		if err != nil || m.WorkspaceID != ws.ID || m.DeletedAt != nil {
			return out, api.NotFound("Milestone")
		}
		milestone = &m
		block.Program = m.ProgramID.String()
	}
	pr, err := s.blockProgram(ctx, ws, block.Program)
	if err != nil {
		return out, err
	}
	if pr == nil {
		out.Hidden = "no_program"
		return out, nil
	}
	p := pr.ReferralProgram
	out.ProgramID, out.ListName = p.ID.String(), pr.ListName
	if p.Status != "on" {
		out.Hidden, out.OffAt, out.OffBy = "program_off", p.StatusAt, pr.StatusByName
	}
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return out, err
	}
	links := map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"}
	var vars map[string]any
	if cid, err := uuid.Parse(in.ContactID); err == nil {
		var email, first, last string
		if err := s.Pool.QueryRow(ctx, `SELECT email, first_name, last_name FROM contacts WHERE id = $1 AND workspace_id = $2 AND erased_at IS NULL`, cid, ws.ID).
			Scan(&email, &first, &last); err != nil {
			return out, api.NotFound("Contact")
		}
		out.For = displayName(first, last, email)
		vars = liquid.Vars(email, first, last, nil, links)
		refs, err := s.referralVars(ctx, ws, pages, []uuid.UUID{cid}, s.now())
		if err != nil {
			return out, err
		}
		vars["referral"] = map[string]any{"show": false}
		if rr, ok := refs[cid]; ok {
			if v, ok := rr.byProgram[p.ID.String()]; ok {
				vars["referral"] = v
			}
		}
		if v, _ := vars["referral"].(map[string]any); v["show"] != true && out.Hidden == "none" {
			out.Hidden = "not_on_list"
		}
		if milestone != nil {
			vars["reward"] = s.sampleReward(ctx, ws, *milestone)
			var rid uuid.UUID
			if err := s.Pool.QueryRow(ctx, `SELECT id FROM referral_rewards WHERE milestone_id = $1 AND contact_id = $2`, milestone.ID, cid).Scan(&rid); err == nil {
				if rw, err := s.Q.ReferralRewardByID(ctx, rid); err == nil {
					vars["reward"] = s.rewardMap(ctx, ws, pages, *milestone, rw, s.now())
				}
			}
		}
	} else {
		vars = liquid.Vars("reader@example.com", "Sam", "", nil, links)
		counted, soon := int64(2), int64(1)
		if milestone != nil {
			counted, soon = int64(milestone.Count), 0
		}
		ref, err := s.SampleReferral(ctx, ws, p.ID, counted, soon)
		if err != nil {
			return out, err
		}
		vars["referral"] = ref
		if milestone != nil {
			vars["reward"] = s.sampleReward(ctx, ws, *milestone)
		}
	}
	// A block the program hides still renders (the canvas shows it dimmed,
	// with the reason); a reader outside the program sees nothing.
	if out.Hidden == "program_off" {
		if v, ok := vars["referral"].(map[string]any); ok && v["show"] != true {
			ref, err := s.SampleReferral(ctx, ws, p.ID, 2, 1)
			if err != nil {
				return out, err
			}
			vars["referral"] = ref
		}
	}
	src := ExpandReferral(block)
	if in.Kind == "reward" {
		src = ExpandReward(parseRewardBlock(in.Settings))
	}
	if out.HTML, err = liquid.Render(src, vars); err != nil {
		return out, api.Invalid("body.settings", err.Error())
	}
	return out, nil
}

// PreviewContacts are the readers to preview Grow blocks as: the program's
// top referrers, then up to three readers the block hides for (on other
// lists).
func (s *Service) PreviewContacts(ctx context.Context, ws *store.Workspace, program *uuid.UUID) (GrowPreviewContacts, error) {
	out := GrowPreviewContacts{Items: []GrowPreviewContact{}}
	bound := ""
	if program != nil {
		bound = program.String()
	}
	pr, err := s.blockProgram(ctx, ws, bound)
	if err != nil || pr == nil {
		return out, err
	}
	p := pr.ReferralProgram
	out.ProgramID, out.ListName, out.Status = p.ID.String(), pr.ListName, p.Status
	rows, err := s.Pool.Query(ctx, `SELECT c.id, c.first_name, c.last_name, c.email, count(r.id) FILTER (WHERE r.status = 'counted') AS counted
		FROM referral_links rl
		JOIN contacts c ON c.id = rl.contact_id AND c.erased_at IS NULL
		JOIN list_memberships lm ON lm.list_id = $2 AND lm.contact_id = c.id AND lm.status = 'subscribed'
		LEFT JOIN referrals r ON r.link_id = rl.id
		WHERE rl.program_id = $1 AND rl.status = 'active'
		GROUP BY c.id ORDER BY counted DESC, c.email LIMIT 6`, p.ID, p.ListID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id uuid.UUID
		var first, last, email string
		var counted int64
		if err := rows.Scan(&id, &first, &last, &email, &counted); err != nil {
			rows.Close()
			return out, err
		}
		out.Items = append(out.Items, GrowPreviewContact{ID: id.String(), Name: displayName(first, last, email), Email: email, OnList: true, Counted: counted,
			Lists: []string{pr.ListName}})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	others, err := s.Pool.Query(ctx, `SELECT c.id, c.first_name, c.last_name, c.email, array_agg(l.name ORDER BY l.name)
		FROM contacts c
		JOIN list_memberships lm ON lm.contact_id = c.id AND lm.status = 'subscribed'
		JOIN lists l ON l.id = lm.list_id AND l.deleted_at IS NULL
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM list_memberships x WHERE x.contact_id = c.id AND x.list_id = $2 AND x.status = 'subscribed')
		GROUP BY c.id ORDER BY c.created_at LIMIT 3`, ws.ID, p.ListID)
	if err != nil {
		return out, err
	}
	defer others.Close()
	for others.Next() {
		var id uuid.UUID
		var first, last, email string
		var lists []string
		if err := others.Scan(&id, &first, &last, &email, &lists); err != nil {
			return out, err
		}
		out.Items = append(out.Items, GrowPreviewContact{ID: id.String(), Name: displayName(first, last, email), Email: email, Lists: lists})
	}
	return out, others.Err()
}
