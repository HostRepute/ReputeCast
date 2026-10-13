package app

// W15 · More sending adapters. Each adapter registers itself in the
// providers registry from its init function; the catalog, the connect
// wizard and the Adapters tab list them from there.
import (
	_ "github.com/hostrepute/reputecast/internal/providers/httpgeneric"
	_ "github.com/hostrepute/reputecast/internal/providers/resend"
)
