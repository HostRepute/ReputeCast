package campaigns

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Collaboration operations (review 2026-10, CO-10): comments, review links
// and presence, plus the public review page behind a link.

type addCommentInput struct {
	CampaignPath
	Body ComposerCommentInput
}

// ThreadPath adds a comment thread's {threadId}.
type ThreadPath struct {
	CampaignPath
	ThreadID string `path:"threadId" format:"uuid"`
}

type resolveInput struct {
	ThreadPath
	Body ComposerResolveInput
}

type reviewLinkPath struct {
	CampaignPath
	LinkID string `path:"linkId" format:"uuid"`
}

type dismissInput struct {
	CampaignPath
	Body ComposerDismissInput
}

type dismissalPath struct {
	CampaignPath
	DismissalID string `path:"dismissalId" format:"uuid"`
}

type presenceInput struct {
	CampaignPath
	Body ComposerPresenceInput
}

// ReviewToken is a review link's {token}.
type ReviewToken struct {
	Token string `path:"token" maxLength:"64" doc:"The token from the review link"`
}

type reviewCommentInput struct {
	ReviewToken
	Body ReviewCommentInput
}

// reviewOutput keeps review pages out of caches and search engines.
type reviewOutput[T any] struct {
	CacheControl string `header:"Cache-Control"`
	RobotsTag    string `header:"X-Robots-Tag"`
	Body         T
}

func reviewClient(ctx context.Context) string {
	if ip := audit.RequestFrom(ctx).IP; ip != nil {
		return ip.String()
	}
	return "unknown"
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

func registerCollaboration(h huma.API, s *Service, tags []string) {
	view, edit := api.Workspace(authz.CampaignsView), api.Workspace(authz.CampaignsEdit)

	api.Get(h, api.Op{ID: "list-campaign-comments", Path: "/workspaces/{workspace}/campaigns/{campaignId}/comments", Summary: "Comment threads on the campaign, open ones first",
		Tags: tags, Access: view},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerThreadList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Comments(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-campaign-comment", Path: "/workspaces/{workspace}/campaigns/{campaignId}/comments", Summary: "Comment on a block or the whole email, or reply to a thread",
		Description: "Anyone who can view the campaign can comment. The members in mentions get a notification (and an email unless they turned it off) linking to the thread.",
		Tags:        tags, Access: view, Status: http.StatusCreated},
		func(ctx context.Context, in *addCommentInput) (*api.Body[ComposerThread], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.AddComment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "resolve-campaign-comment", Path: "/workspaces/{workspace}/campaigns/{campaignId}/comments/{threadId}/resolved", Summary: "Resolve a comment thread, or open it again",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *resolveInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			tid, err := parseID(in.ThreadID, "Comment")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.ResolveThread(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, tid, in.Body)
		})
	api.Get(h, api.Op{ID: "list-mentionable-members", Path: "/workspaces/{workspace}/campaigns/{campaignId}/mentionable", Summary: "Members who can be @mentioned in the campaign's comments",
		Tags: tags, Access: view},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerMemberList], error) {
			if _, err := in.id(); err != nil {
				return nil, err
			}
			out, err := s.Mentionable(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-campaign-review-link", Path: "/workspaces/{workspace}/campaigns/{campaignId}/review-links", Summary: "Share the saved version for review",
		Description: "Anyone with the link sees that version of the email (personalized for a sample reader) and can comment without signing in, for 7 days or until the link is revoked. Nothing about the audience, settings or other campaigns.",
		Tags:        tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerReviewLink], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CreateReviewLink(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-review-links", Path: "/workspaces/{workspace}/campaigns/{campaignId}/review-links", Summary: "The campaign's working review links, newest first",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerReviewLinkList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.ReviewLinks(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "revoke-campaign-review-link", Path: "/workspaces/{workspace}/campaigns/{campaignId}/review-links/{linkId}", Summary: "Revoke a review link",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *reviewLinkPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			lid, err := parseID(in.LinkID, "Link")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.RevokeReviewLink(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, lid)
		})

	api.Put(h, api.Op{ID: "campaign-presence", Path: "/workspaces/{workspace}/campaigns/{campaignId}/presence", Summary: "Say you have the campaign open; returns who else does",
		Description: "The composer calls it every 20 seconds while open. Someone not heard from for a minute is gone.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *presenceInput) (*api.Body[ComposerPresence], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Present(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "leave-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/presence", Summary: "Say you closed the campaign",
		Tags: tags, Access: view},
		func(ctx context.Context, in *CampaignPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Leave(ctx, api.PrincipalFrom(ctx), id)
		})

	api.Post(h, api.Op{ID: "dismiss-campaign-warning", Path: "/workspaces/{workspace}/campaigns/{campaignId}/dismissals", Summary: "Dismiss a warning of the composer's checks, with a reason",
		Description: "key is the warning's issueKeys entry from the checks. The warning stays listed with who dismissed it and why, and comes back when it changes. Blocking issues are never dismissed.",
		Tags:        tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *dismissInput) (*api.Body[ComposerDismissal], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Dismiss(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "undismiss-campaign-warning", Path: "/workspaces/{workspace}/campaigns/{campaignId}/dismissals/{dismissalId}", Summary: "Count a dismissed warning again (Undo)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *dismissalPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			did, err := parseID(in.DismissalID, "Dismissal")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Undismiss(ctx, api.WorkspaceFrom(ctx), id, did)
		})

	api.Get(h, api.Op{ID: "get-review", Path: "/review/{token}", Summary: "The email behind a review link (public, rate-limited)",
		Description: "One version of one campaign, personalized for a sample reader, and the comments written through the link. Limited per client address.",
		Tags:        tags, Access: api.Public},
		func(ctx context.Context, in *ReviewToken) (*reviewOutput[ReviewPage], error) {
			out, err := s.Review(ctx, in.Token, reviewClient(ctx))
			if err != nil {
				return nil, err
			}
			return &reviewOutput[ReviewPage]{CacheControl: "no-store", RobotsTag: "noindex, nofollow", Body: out}, nil
		})
	api.Post(h, api.Op{ID: "add-review-comment", Path: "/review/{token}/comments", Summary: "Comment through a review link (public, rate-limited)",
		Description: "Signed with the name the reviewer gives. At most 30 comments an hour through one link.",
		Tags:        tags, Access: api.Public, Status: http.StatusCreated, MaxBodyBytes: 64 << 10},
		func(ctx context.Context, in *reviewCommentInput) (*reviewOutput[ComposerThread], error) {
			out, err := s.ReviewComment(ctx, in.Token, reviewClient(ctx), in.Body)
			if err != nil {
				return nil, err
			}
			return &reviewOutput[ComposerThread]{CacheControl: "no-store", RobotsTag: "noindex, nofollow", Body: out}, nil
		})
}
