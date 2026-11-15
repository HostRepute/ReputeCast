package ai

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// Names and Liquid expressions are human review data. The model receives
// only fixed inventory metadata and the source's existing opaque bindings.
type AIAgentPoolSnapshot struct {
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Source    string    `json:"source"`
	Free      int64     `json:"free"`
	At        time.Time `json:"at"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c agentContext) poolAccess(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, exact bool) error {
	if c.Brief.PoolID == nil {
		return nil
	}
	if !p.Can(authz.CampaignsView) {
		return api.Forbidden(authz.CampaignsView, nil)
	}
	row, err := q.AIAgentCouponPool(ctx, store.AIAgentCouponPoolParams{WorkspaceID: ws.ID, ID: *c.Brief.PoolID})
	if err != nil {
		return api.MapError(err, "Discount pool")
	}
	if exact && (row.Status != "active" || (c.Pool != nil && (row.Slug != c.Pool.Slug || !row.UpdatedAt.Equal(c.Pool.UpdatedAt)))) {
		return draftProblem(ctx, 409, "changed")
	}
	return nil
}

// Add only unexpanded code/URL tokens to the selected new-email source.
// Projection binds both locally; there is no rendering or allocation here.
func (s *Service) prepareAgentPool(ctx context.Context, ws *store.Workspace, p *authz.Principal, c *agentContext, source string) (string, error) {
	if c.Brief.PoolID == nil {
		return source, nil
	}
	if c.Brief.CampaignID != nil || c.Brief.Kind == "post" {
		return "", agentProblem(ctx, "input")
	}
	if err := c.poolAccess(ctx, s.Q, ws, p, true); err != nil {
		return "", err
	}
	row, err := s.Q.AIAgentCouponPool(ctx, store.AIAgentCouponPoolParams{WorkspaceID: ws.ID, ID: *c.Brief.PoolID})
	if err != nil {
		return "", err
	}
	free, err := s.Q.CouponPoolFree(ctx, row.ID)
	if err != nil {
		return "", err
	}
	c.Pool = &AIAgentPoolSnapshot{Name: row.Name, Slug: row.Slug, Source: row.Source, Free: free, At: s.now(), UpdatedAt: row.UpdatedAt}
	if c.Literals == nil {
		c.Literals = privateLiterals{}
	}
	for _, value := range []string{row.Name, row.Slug, row.ID.String()} {
		if value != "" && c.Literals[value] == "" {
			c.Literals[value] = "bound_" + strconv.Itoa(len(c.Literals)+1)
		}
	}
	code, link := "{{ coupons."+row.Slug+".code }}", "{{ coupons."+row.Slug+".url }}"
	// A template that already has both tokens is unchanged. A partial
	// selection needs local editing rather than duplicating a discount block.
	hasCode, hasLink := strings.Contains(source, code), strings.Contains(source, link)
	if hasCode && hasLink {
		return source, nil
	}
	if hasCode || hasLink {
		return "", draftProblem(ctx, 422, "source")
	}
	root, err := mjml.Parse(source)
	if err != nil {
		return "", draftProblem(ctx, 422, "source")
	}
	body := mjml.Body(root)
	if body == nil || len(body.Children) == 0 {
		return "", draftProblem(ctx, 422, "source")
	}
	// Insert after a complete section, never between its rc marker and
	// element. Projection moves the original footer to the end on restore.
	pos := body.Children[len(body.Children)-1].End
	markup := `<mj-section><mj-column><mj-text>` + code + `</mj-text><mj-button href="` + link + `">Use this offer</mj-button></mj-column></mj-section>`
	return source[:pos] + markup + source[pos:], nil
}

func (p AIAgentPoolSnapshot) modelJSON() []byte {
	data, _ := json.Marshal(struct {
		Source string    `json:"source"`
		Free   int64     `json:"available"`
		At     time.Time `json:"asOf"`
	}{p.Source, p.Free, p.At})
	return data
}
