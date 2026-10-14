package app

// Each sending adapter registers itself in the providers registry from its
// init function; the catalog, the connect wizard and the Adapters tab list
// them from there.
import (
	_ "github.com/hostrepute/reputecast/internal/providers/azureacs"
	_ "github.com/hostrepute/reputecast/internal/providers/brevo"
	_ "github.com/hostrepute/reputecast/internal/providers/httpgeneric"
	_ "github.com/hostrepute/reputecast/internal/providers/mailgun"
	_ "github.com/hostrepute/reputecast/internal/providers/mailjet"
	_ "github.com/hostrepute/reputecast/internal/providers/resend"
	_ "github.com/hostrepute/reputecast/internal/providers/sendgrid"
	_ "github.com/hostrepute/reputecast/internal/providers/sparkpost"
)
