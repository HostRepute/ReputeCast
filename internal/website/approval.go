package website

import (
	"time"

	"github.com/google/uuid"
)

// PostApprovalRequest is what an approval of a post covers: the content
// (its hash), the address, the visibility and when it goes live.
type PostApprovalRequest struct {
	PostID     uuid.UUID
	Title      string
	Kind       string // web or campaign
	Slug       string
	Path       string // "/posts/q4-roadmap-deep-dive"
	Visibility string
	// VisibilityLabel says the visibility in words ("Members", "Paid · all tiers").
	VisibilityLabel string
	// When: now, at (PublishAt), with_send.
	When      string
	PublishAt *time.Time
	// Hash covers the content and the settings above: a change after
	// approval needs a new one.
	Hash string
	// Excerpt and Preview are what the approval review shows (the web preview).
	Excerpt string
}

// PostApprovalDecision is whether a person may publish a post as it is.
type PostApprovalDecision struct {
	Allowed bool
	// NeedsApproval: Publish becomes Request approval, for Reason, with the
	// rule that matched ("author_lacks_send", "post_publish").
	NeedsApproval bool
	Rule          string
	Reason        string
	Approvers     []string
	// Pending is an open request for this post (its id), Approved one that
	// covers the post as it is.
	Pending  string
	Approved string
	// RequestedBy and RequestedAt describe the pending request.
	RequestedBy string
	RequestedAt *time.Time
}
