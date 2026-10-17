package publicpages

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// A placement test copy (internal/placement) went to a seed mailbox the
// workspace owns, with the same links as a real send but no contact behind
// them: its unsubscribe, preference and view-online links open a page that
// says so, and nobody is unsubscribed.

const placementCopyText = "This is a placement test copy. Nobody was unsubscribed."

// placementCopy reports whether token is a link of one of this workspace's
// placement test copies.
func (h *Handler) placementCopy(r *http.Request, s *site, token string, kind linktoken.Kind) bool {
	if h.Tokens == nil || s == nil || token == "" || len(token) > 200 {
		return false
	}
	c, err := h.Tokens.Verify(token, kind)
	if err != nil || c.WorkspaceID != s.ws.ID || c.ContactID != uuid.Nil || c.MessageID == uuid.Nil {
		return false
	}
	k, err := h.Q.PlacementMessageKind(r.Context(), c.MessageID)
	return err == nil && k == "placement"
}

// placementNotice is the page a placement test copy's links open. Only the
// workspace's own seed mailboxes ever get these links, so its words aren't
// among the labels a workspace translates.
func (h *Handler) placementNotice(w http.ResponseWriter, r *http.Request, s *site) {
	pl := h.visitorLang(w, r, s, nil)
	h.messageIn(w, r, s, pl, http.StatusOK, "Placement test copy · "+s.set.Branding.PublisherName, messageBody{Icon: "mail",
		Heading: "Placement test copy", Text: placementCopyText}, true)
}

// placementOneClick answers a mail app's one-click unsubscribe of a copy.
func placementOneClick(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(placementCopyText + "\n"))
}
