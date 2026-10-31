package imports

import "github.com/hostrepute/reputecast/internal/mailbox"

// The MX check lives in internal/mailbox, which also classifies company
// domains by their hosting; the importer uses it to spot MX-less domains.
type (
	// Resolver is the part of net.Resolver the MX check uses.
	Resolver = mailbox.Resolver
	// MXResult is what a domain's DNS says.
	MXResult = mailbox.MXResult
	// MXChecker tells whether a domain accepts mail, caching answers.
	MXChecker = mailbox.MXChecker
)
