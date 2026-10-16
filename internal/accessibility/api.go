package accessibility

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/templates"
)

type policyInput struct {
	api.WorkspaceInput
	Body AccessibilityPolicyInput
}

type CampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

func (in CampaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type checkInput struct {
	CampaignPath
	Body AccessibilityCheckRequest
}

// AccessibilityFixInput applies one fix to the composer's MJML.
type AccessibilityFixInput struct {
	MJML string `json:"mjml" maxLength:"2000000"`
	Kind string `json:"kind" enum:"alt_text,decorative,link_text,color,heading,text_size,text_role"`
	// Path is the element (mj-body element-index path); Index the nth link
	// or image inside a text or Raw HTML block.
	Path  []int `json:"path" maxItems:"12"`
	Index int   `json:"index,omitempty" minimum:"0"`
	// Value is the alt text, the link text or the color.
	Value string `json:"value,omitempty" maxLength:"500"`
	// Target and From are a color fix's attribute ("background" or "text")
	// and the color it replaces inside the element.
	Target   string                       `json:"target,omitempty" enum:"background,text"`
	From     string                       `json:"from,omitempty" maxLength:"60"`
	Headings []AccessibilityHeadingChange `json:"headings,omitempty" maxItems:"50"`
	ToPx     int                          `json:"toPx,omitempty"`
	// Role is a Text block's new role (Settings › Text role).
	Role string `json:"role,omitempty" enum:"p,h1,h2,h3,h4,h5,h6"`
}

// AccessibilityFixed is the MJML after a fix.
type AccessibilityFixed struct {
	MJML string `json:"mjml"`
}

type fixInput struct {
	api.WorkspaceInput
	Body AccessibilityFixInput
}

// FixContent applies one fix to an email's MJML: the composer's buttons
// change the email the way Review & send's fixes do. Nothing is saved.
func FixContent(in AccessibilityFixInput) (AccessibilityFixed, error) {
	out, err := ApplyFix(in.MJML, FixRequest{Kind: in.Kind, Path: in.Path, Index: in.Index, Value: in.Value, Target: in.Target, From: in.From,
		Headings: in.Headings, ToPx: in.ToPx, Role: in.Role})
	if err != nil {
		return AccessibilityFixed{}, api.Invalid("body", sentenceOf(fixError(err).Error()))
	}
	return AccessibilityFixed{MJML: out}, nil
}

// sentenceOf is an error as a sentence: capital first letter, a full stop.
func sentenceOf(s string) string {
	s = capitalize(strings.TrimSpace(s))
	if s != "" && !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

// Register adds the accessibility operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Accessibility"}
	api.Get(h, api.Op{ID: "get-accessibility-policy", Path: "/workspaces/{workspace}/accessibility/policy", Summary: "The workspace's accessibility policy",
		Description: "Whether serious issues only warn or block sends and publishing, and the smallest text size.", Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[AccessibilityPolicy], error) {
			out, err := s.Policy(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-accessibility-policy", Path: "/workspaces/{workspace}/accessibility/policy", Summary: "Change the accessibility policy",
		Description: "Block makes Send Readiness turn Blocked on serious issues and stops publishing automations and transactional templates that have them. Live automations and published templates keep sending.",
		Tags:        tags, Access: api.Workspace(authz.WorkspaceManage)},
		func(ctx context.Context, in *policyInput) (*api.Body[AccessibilityPolicy], error) {
			out, err := s.UpdatePolicy(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-accessibility-policy", Path: "/workspaces/{workspace}/accessibility/policy/impact", Summary: "What a policy would affect now",
		Description: "Scheduled campaigns that would be held at their send-time recheck, and drafts and automation drafts with serious issues. Nothing is saved.",
		Tags:        tags, Access: api.Workspace(authz.WorkspaceManage)},
		func(ctx context.Context, in *policyInput) (*api.Body[AccessibilityImpact], error) {
			out, err := s.Impact(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-accessibility", Path: "/workspaces/{workspace}/campaigns/{campaignId}/accessibility", Summary: "Accessibility issues of a campaign as saved",
		Description: "Every language version's compiled HTML is checked; an issue in several versions is listed once with its languages.", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *CampaignPath) (*api.Body[AccessibilityReport], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignReport(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-campaign-accessibility", Path: "/workspaces/{workspace}/campaigns/{campaignId}/accessibility", Summary: "Check the composer's current content for accessibility issues",
		Description: "The composer calls it as you edit. The version being edited is checked with your unsaved changes, the others as saved. Nothing is saved.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView), MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *checkInput) (*api.Body[AccessibilityReport], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CheckCampaign(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "apply-accessibility-fix", Path: "/workspaces/{workspace}/accessibility/fix", Summary: "Apply an accessibility fix to MJML",
		Description: "The composer's fixes: alt text, decorative images, link text, a color, heading levels, a text size, and a Text block's role. Returns the changed MJML; nothing is saved, the composer's autosave stores it.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit), MaxBodyBytes: 3 << 20},
		func(_ context.Context, in *fixInput) (*api.Body[AccessibilityFixed], error) {
			out, err := FixContent(in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "template-gallery-accessibility", Path: "/workspaces/{workspace}/accessibility/templates", Summary: "Accessibility results of the template gallery",
		Description: "Each starter, saved template and the last sent campaign, checked in the workspace's default language.", Tags: tags, Access: api.Workspace(authz.TemplatesView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[AccessibilityGallery], error) {
			ws := api.WorkspaceFrom(ctx)
			g, err := (&templates.Service{Pool: s.Pool, Q: s.Q, Audit: s.Audit}).Gallery(ctx, ws)
			if err != nil {
				return nil, err
			}
			ids := make([]string, 0, len(g.Starters))
			for _, st := range g.Starters {
				ids = append(ids, st.ID)
			}
			out, err := s.Gallery(ctx, ws, ids)
			return api.OK(out), err
		})
}
