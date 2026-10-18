package website

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// WebsitePostPath is a post's path (exported: Huma reads embedded
// path structs only when their type is exported).
type WebsitePostPath struct {
	api.WorkspaceInput
	Post string `path:"post" format:"uuid" doc:"The post's id"`
}

func (p WebsitePostPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.Post)
	if err != nil {
		return uuid.Nil, api.NotFound("Post")
	}
	return id, nil
}

type postBody[T any] struct {
	WebsitePostPath
	Body T
}

// WebsiteTagPath is a tag's path.
type WebsiteTagPath struct {
	api.WorkspaceInput
	Tag string `path:"tag" format:"uuid" doc:"The tag's id"`
}

type tagBody struct {
	WebsiteTagPath
	Body WebsiteTagInput
}

// WebsiteCampaignPath is a campaign's path.
type WebsiteCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"The campaign's id"`
}

func (p WebsiteCampaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type campaignBody struct {
	WebsiteCampaignPath
	Body WebsitePostInput
}

// WebsiteContactPath is a contact's path.
type WebsiteContactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"The contact's id"`
}

type body[T any] struct {
	api.WorkspaceInput
	Body T
}

type postsInput struct {
	api.WorkspaceInput
	WebsitePostFilter
}

type campaignStatesInput struct {
	api.WorkspaceInput
	IDs string `query:"ids" maxLength:"4000" doc:"Campaign ids, comma-separated (up to 100)"`
}

// WebsiteCampaignStates are the Web column of the Campaigns list.
type WebsiteCampaignStates struct {
	SiteOn bool                      `json:"siteOn"`
	States []WebsiteCampaignWebState `json:"states"`
}

// WebsiteCampaignReportView is the Web section of a campaign's report.
type WebsiteCampaignReportView struct {
	// Report is nil when the campaign has no post on the site.
	Report *WebsiteCampaignReport `json:"report,omitempty"`
}

func parseID(s, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

// Register adds the website's operations: the Website tab (its states,
// posts, home and navigation, access), the composer's Web tab, the
// Campaigns list's Web column, the report's Web section and the contact
// profile's Website card. Publishing and unpublishing need website:publish
// (or an approval); turning the site on or off and its access rules need
// website:manage.
func Register(h huma.API, s *Service) {
	tags := []string{"Website"}
	base := "/workspaces/{workspace}/website"
	view := api.Workspace(authz.WebsiteView)
	edit := api.Workspace(authz.WebsiteEdit)
	publish := api.Workspace(authz.WebsitePublish)
	manage := api.WorkspaceHuman(authz.WebsiteManage)

	api.Get(h, api.Op{ID: "get-website", Path: base, Summary: "The website: its address, state, numbers and what turning it on or off does", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[WebsiteOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "turn-website-on", Path: base + "/turn-on", Summary: "Turn the website on",
		Description: "Archived campaigns become posts (their archive addresses redirect), archived lists become tags and the domain's root becomes the home page. It runs in the background; the website's state says how far it got.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *body[WebsiteTurnOnInput]) (*api.Body[WebsiteOverview], error) {
			out, err := s.TurnOn(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "turn-website-off", Path: base + "/turn-off", Summary: "Turn the website off",
		Description: "The archive comes back; web-only posts, sign-in, the account page and the paywall go offline. Every post and setting is kept. Type the site's address to confirm.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *body[WebsiteTurnOffInput]) (*api.Body[WebsiteOverview], error) {
			out, err := s.TurnOff(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Posts.
	api.Get(h, api.Op{ID: "list-website-posts", Path: base + "/posts", Summary: "Website › Posts, with their 30-day numbers", Tags: tags, Access: view},
		func(ctx context.Context, in *postsInput) (*api.Body[WebsitePostList], error) {
			out, err := s.Posts(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.WebsitePostFilter)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-website-post", Path: base + "/posts", Summary: "Start a web-only post", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *body[WebsitePostCreate]) (*api.Body[WebsitePost], error) {
			out, err := s.CreatePost(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-website-post", Path: base + "/posts/{post}", Summary: "A post: its settings, content, web checks and whether you can publish it", Tags: tags, Access: view},
		func(ctx context.Context, in *WebsitePostPath) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Post(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-website-post", Path: base + "/posts/{post}", Summary: "Save a post's settings: title, address, excerpt, cover, byline, tags, visibility and SEO",
		Description: "Changing a published post's address keeps the old one as a redirect.", Tags: tags, Access: edit},
		func(ctx context.Context, in *postBody[WebsitePostInput]) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.UpdatePost(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-website-post-content", Path: base + "/posts/{post}/content", Summary: "Autosave or save a web-only post's content",
		Description: "Another save since baseVersion is a conflict (409) unless force is set.", Tags: tags, Access: edit, MaxBodyBytes: 4 << 20},
		func(ctx context.Context, in *postBody[WebsitePostContent]) (*api.Body[WebsitePostSaved], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SavePostContent(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-website-post", Path: base + "/posts/{post}", Summary: "Delete a post",
		Description: "A published post leaves the site too (its address answers 410); that needs website:publish.", Tags: tags, Access: edit},
		func(ctx context.Context, in *WebsitePostPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeletePost(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "publish-website-post", Path: base + "/posts/{post}/publish", Summary: "Publish a post now, or schedule it",
		Description: "When publishing needs approval the answer is 409 approval_required with the rule and the approvers: ask for approval instead.",
		Tags:        tags, Access: publish},
		func(ctx context.Context, in *postBody[WebsitePublishInput]) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "undo-website-post-publish", Path: base + "/posts/{post}/undo-publish", Summary: "Take back a publish within seconds (the toast's Undo)", Tags: tags, Access: publish},
		func(ctx context.Context, in *WebsitePostPath) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.UndoPublish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unpublish-website-post", Path: base + "/posts/{post}/unpublish", Summary: "Take a post off the site, or unschedule it",
		Description: "Its address answers 410, or redirects to redirectTo.", Tags: tags, Access: publish},
		func(ctx context.Context, in *postBody[WebsiteUnpublishInput]) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Unpublish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-website-post-redirect", Path: base + "/posts/{post}/redirect", Summary: "Change where an unpublished post's address leads", Tags: tags, Access: publish},
		func(ctx context.Context, in *postBody[WebsiteUnpublishInput]) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetRedirect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-website-post-approval", Path: base + "/posts/{post}/approval", Summary: "Ask for approval to publish a post as it is",
		Description: "The request covers the content, the address, the visibility and the publish time; a change needs a new one. Once approved, the post goes live by itself.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.WebsiteEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *postBody[WebsiteApprovalInput]) (*api.Body[WebsitePost], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RequestApproval(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-website-post-as-campaign", Path: base + "/posts/{post}/send-as-campaign", Summary: "Make a campaign draft of a web-only post",
		Description: "Its title becomes the subject, its excerpt the preheader and its blocks the email. Pick the audience in the composer.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *WebsitePostPath) (*api.Body[WebsiteSentAsCampaign], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SendAsCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-website-past-campaigns", Path: base + "/past-campaigns", Summary: "Sent campaigns that aren't posts yet", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[[]WebsitePastCampaign], error) {
			out, err := s.PastCampaigns(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-website-past-campaigns", Path: base + "/past-campaigns/publish", Summary: "Publish sent campaigns as posts, at their send dates", Tags: tags, Access: publish},
		func(ctx context.Context, in *body[WebsitePublishPastInput]) (*api.Body[WebsitePublishPastResult], error) {
			out, err := s.PublishPast(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Tags.
	api.Get(h, api.Op{ID: "list-website-tags", Path: base + "/tags", Summary: "The site's tags with their posts", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[[]WebsiteTag], error) {
			out, err := s.Tags(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-website-tag", Path: base + "/tags", Summary: "Add a tag", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *body[WebsiteTagInput]) (*api.Body[WebsiteTag], error) {
			out, err := s.CreateTag(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-website-tag", Path: base + "/tags/{tag}", Summary: "Rename a tag, change its address or description", Tags: tags, Access: edit},
		func(ctx context.Context, in *tagBody) (*api.Body[WebsiteTag], error) {
			id, err := parseID(in.Tag, "Tag")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateTag(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-website-tag", Path: base + "/tags/{tag}", Summary: "Delete a tag (its posts stay)", Tags: tags, Access: edit},
		func(ctx context.Context, in *WebsiteTagPath) (*api.Empty, error) {
			id, err := parseID(in.Tag, "Tag")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeleteTag(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})

	// Home & navigation, Access.
	api.Get(h, api.Op{ID: "get-website-home-nav", Path: base + "/home-nav", Summary: "Website › Home & navigation", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[WebsiteHomeNav], error) {
			out, err := s.HomeNav(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-website-home-nav", Path: base + "/home-nav", Summary: "Save the home page, the navigation and the site's identity", Tags: tags, Access: edit},
		func(ctx context.Context, in *body[WebsiteHomeNavInput]) (*api.Body[WebsiteHomeNav], error) {
			out, err := s.SaveHomeNav(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-website-access", Path: base + "/access", Summary: "Website › Access: members, sign-up, visibility defaults, the meter and the paywall", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[WebsiteAccess], error) {
			out, err := s.Access(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-website-access", Path: base + "/access", Summary: "Save who can read what", Tags: tags, Access: manage},
		func(ctx context.Context, in *body[WebsiteAccessInput]) (*api.Body[WebsiteAccess], error) {
			out, err := s.SaveAccess(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// The composer's Web tab, the Campaigns list and the report.
	ctags := []string{"Campaigns"}
	api.Get(h, api.Op{ID: "get-campaign-web", Path: "/workspaces/{workspace}/campaigns/{campaignId}/web", Summary: "A campaign's Web tab: whether and how it goes on the website",
		Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *WebsiteCampaignPath) (*api.Body[WebsiteCampaignWeb], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignWeb(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-campaign-web", Path: "/workspaces/{workspace}/campaigns/{campaignId}/web", Summary: "Save a campaign's Web tab",
		Description: "A live post's settings need website:edit too.", Tags: ctags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *campaignBody) (*api.Body[WebsiteCampaignWeb], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SaveCampaignWeb(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-web-states", Path: "/workspaces/{workspace}/campaigns-web", Summary: "The Campaigns list's Web column: the posts of campaigns",
		Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *campaignStatesInput) (*api.Body[WebsiteCampaignStates], error) {
			ws := api.WorkspaceFrom(ctx)
			ids := []uuid.UUID{}
			for _, part := range strings.Split(in.IDs, ",") {
				if id, err := uuid.Parse(strings.TrimSpace(part)); err == nil && len(ids) < 100 {
					ids = append(ids, id)
				}
			}
			on, err := s.IsOn(ctx, ws)
			if err != nil {
				return nil, err
			}
			states, err := s.CampaignWebStates(ctx, ws, ids)
			return api.OK(WebsiteCampaignStates{SiteOn: on, States: states}), err
		})
	api.Get(h, api.Op{ID: "get-campaign-web-report", Path: "/workspaces/{workspace}/campaigns/{campaignId}/web-report", Summary: "The Web section of a campaign's report",
		Tags: []string{"Reports"}, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *WebsiteCampaignPath) (*api.Body[WebsiteCampaignReportView], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignReport(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(WebsiteCampaignReportView{Report: out}), err
		})

	// The contact profile's Website card.
	api.Get(h, api.Op{ID: "get-contact-website", Path: "/workspaces/{workspace}/contacts/{contactId}/website", Summary: "A contact's Website card: membership, sign-ins, sessions and the meter",
		Tags: []string{"Contacts"}, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *WebsiteContactPath) (*api.Body[WebsiteContactCard], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactCard(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "sign-contact-out-everywhere", Path: "/workspaces/{workspace}/contacts/{contactId}/website/sign-out", Summary: "Sign a member out of the website on every device",
		Description: "Audited, and on the contact's timeline.", Tags: []string{"Contacts"}, Access: api.WorkspaceHuman(authz.SubscribersEdit)},
		func(ctx context.Context, in *WebsiteContactPath) (*api.Body[WebsiteSignedOut], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.SignOutEverywhere(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
}
