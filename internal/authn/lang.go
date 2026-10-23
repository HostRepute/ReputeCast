package authn

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The password rules read in the person's language (PasswordProblemIn);
// the CLI keeps English (PasswordProblem).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "authn")
