package demo

import "context"

// Seeders owned by each workstream. Each file seed_<name>.go assigns its
// implementation in init(), so the order in Seed never changes and no two
// workstreams edit the same file.
var (
	seedTeam           = noop // team, roles, invitations (seed_team.go)
	seedAccount        = noop // passkeys, sessions, identities, preferences
	seedInstance       = noop // SSO providers, auth policy, social sign-in
	seedAudience       = noop // lists, contacts, fields, segments, suppressions, imports
	seedSenders        = noop // sending domains, senders, DNS status
	seedProviders      = noop // providers, routing, warm-up, events
	seedDeliverability = noop // blocklists, Postmaster, SNDS, DMARC, incidents, HostRepute
	seedCampaigns      = noop // campaign rows and their headline counters
	seedComposer       = noop // templates, saved sections, campaign content and versions
	seedReadiness      = noop // Send Readiness runs, approval policy and requests
	seedSending        = noop // a live send, incidents, held recipients
	seedReports        = noop // report details: hourly engagement, links, bounces, unsubscribes
	seedAutomations    = noop // flows, versions, runs and step stats
	seedForms          = noop // signup forms, public page branding, archive
	seedIntegrations   = noop // API keys, MCP agents, webhooks and deliveries
	seedNotifications  = noop // notification center items
	seedAudit          = noop // audit history
)

func noop(context.Context, *Seeder) error { return nil }
