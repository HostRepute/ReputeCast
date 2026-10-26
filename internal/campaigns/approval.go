package campaigns

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// CodeApprovalWouldCancel is the 409 a content save gets when the campaign
// has a pending or approved request: saving cancels it, so the composer
// asks first (CO-7) and saves again with cancelApproval.
const CodeApprovalWouldCancel = "approval_would_cancel"

// ComposerApproval is a campaign's pending or approved request, as the
// composer tells someone about to edit it (ComposerStates: "Edit an
// approved campaign?").
type ComposerApproval struct {
	ID     string `json:"id"`
	Status string `json:"status" enum:"pending,approved"`
	// RequestedBy is who asked; ApprovedBy who approved, in order.
	RequestedBy string     `json:"requestedBy"`
	RequestedAt time.Time  `json:"requestedAt"`
	ApprovedBy  []string   `json:"approvedBy"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
	// Version is the campaign version the request covers.
	Version int `json:"version"`
}

// openApproval is the campaign's pending or approved request, or nil.
func (s *Service) openApproval(ctx context.Context, q *store.Queries, id uuid.UUID) (*ComposerApproval, error) {
	r, err := q.ComposerOpenApproval(ctx, &id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &ComposerApproval{ID: r.ID.String(), Status: r.Status, RequestedBy: r.RequestedByName, RequestedAt: r.RequestedAt, DecidedAt: r.DecidedAt, Version: int(r.CampaignVersion), ApprovedBy: []string{}}
	if r.Status == "approved" {
		names, err := q.ComposerApprovalApprovers(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			if n != "" {
				out.ApprovedBy = append(out.ApprovedBy, n)
			}
		}
	}
	return out, nil
}

// approvalWouldCancel is the 409 for a save that would cancel a request.
func approvalWouldCancel(l i18n.Lang, a *ComposerApproval) error {
	name := strings.Join(a.ApprovedBy, ", ")
	if name == "" {
		name = a.RequestedBy
	}
	msg := mProblemApprovalCancel.Text(l, i18n.Args{"status": a.Status, "name": name, "version": strconv.Itoa(a.Version)})
	return api.Err(http.StatusConflict, CodeApprovalWouldCancel, msg).WithMeta("approvalId", a.ID, "status", a.Status, "approvedBy", a.ApprovedBy, "version", a.Version)
}
