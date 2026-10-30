// Package mailbox decides which mailbox provider filters an address's mail.
//
// Consumer domains (gmail.com, outlook.com, yahoo.co.jp…) are known by name.
// A company's own domain is classified from its MX records: Google Workspace
// mail is filtered by Gmail, Microsoft 365 by Outlook (Exchange Online
// Protection), Yahoo-hosted domains by Yahoo and iCloud+ custom domains by
// iCloud. The answer is kept on the contact (mailbox_provider,
// mailbox_detail, mailbox_checked_at), so routing conditions, per-mailbox
// throttles, the auto-pause guard, warm-up and reports all agree on it.
package mailbox

import (
	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// Mailbox provider keys, as stored on contacts and messages.
const (
	Gmail   = emailaddr.Gmail
	Outlook = emailaddr.Outlook
	Yahoo   = emailaddr.Yahoo
	ICloud  = emailaddr.ICloud
	Other   = emailaddr.Other
)

// Providers lists the keys in chart order.
var Providers = emailaddr.MailboxProviders

// Class is the mailbox provider that filters a domain's mail and, for a
// company domain, the hosting behind it ("Google Workspace").
type Class struct {
	Provider string `json:"provider"`
	// Detail names the hosting of a company domain; empty for consumer
	// domains and for domains whose hosting isn't recognised.
	Detail string `json:"detail,omitempty"`
}

// Hosted reports whether the domain is a company's own, hosted by a
// mailbox provider.
func (c Class) Hosted() bool { return c.Detail != "" && c.Provider != Other }

// Static classifies a domain by name only: consumer domains get their
// provider, every other domain is Other until its MX records are known.
func Static(domain string) Class {
	return Class{Provider: emailaddr.MailboxProvider(domain)}
}

// FromHosting classifies a domain whose MX hosting is known (see
// emailaddr.HostingFromMX). Consumer domains keep their provider.
func FromHosting(domain, hosting string) Class {
	if p := emailaddr.MailboxProvider(domain); p != Other {
		return Class{Provider: p}
	}
	switch hosting {
	case emailaddr.HostGoogleWorkspace:
		return Class{Provider: Gmail, Detail: hosting}
	case emailaddr.HostMicrosoft365, emailaddr.HostOutlookCom:
		return Class{Provider: Outlook, Detail: hosting}
	case emailaddr.HostYahoo:
		return Class{Provider: Yahoo, Detail: hosting}
	case emailaddr.HostICloud:
		return Class{Provider: ICloud, Detail: hosting}
	}
	return Class{Provider: Other, Detail: hosting}
}

// ReportsComplaints reports whether a mailbox provider tells senders about
// each complaint (a feedback loop: Outlook.com's JMRP, Yahoo's CFL and the
// ISPs behind "other"). Gmail and Apple never do: Gmail's spam rate comes
// only as a daily aggregate in Postmaster Tools. A complaint rate measured
// over their delivered mail would read 0% whatever readers did, so rates
// leave their mail out and their own rate is "not reported".
func ReportsComplaints(provider string) bool {
	switch provider {
	case Gmail, ICloud:
		return false
	}
	return true
}

// NotReported is the reason an API gives for a complaint rate it leaves
// empty because the mailbox provider doesn't report complaints.
const NotReported = "not_reported"

// Name is a mailbox provider's display name.
func Name(provider string) string { return emailaddr.MailboxName(provider) }
