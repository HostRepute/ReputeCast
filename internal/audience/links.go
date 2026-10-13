package audience

import (
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

const (
	linkConfirm     = linktoken.Confirm
	linkPreferences = linktoken.Preferences
)

// linkClaims identifies a contact outside a message (confirmation and
// preference center links).
func linkClaims(ws, contact uuid.UUID, kind linktoken.Kind) linktoken.Claims {
	return linktoken.Claims{Kind: kind, WorkspaceID: ws, ContactID: contact}
}

func mailMessage(ws *store.Workspace, c store.Contact, subject, text string) mailer.Message {
	id := ws.ID
	return mailer.Message{
		WorkspaceID: &id, To: c.Email, ToName: strings.TrimSpace(c.FirstName + " " + c.LastName),
		Subject: subject, Text: text, Kind: "optin_confirmation",
	}
}
