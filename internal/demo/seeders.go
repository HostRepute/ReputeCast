package demo

import "context"

// Each file seed_<name>.go assigns its seeder in init(), so the order in
// Seed never changes.
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
	seedTransactional  = noop // transactional email: Acme Cloud's templates, keys, messages, relay (seed_transactional.go)
	// Design expansion (append-only).
	seedIntegrationsHub = noop // integrations hub: Zapier, n8n, WordPress sites (seed_integrations_hub.go)
	// welcome prompts (seed_transactional_templates.go, which also holds the
	// transactional templates' content; seedTransactional writes them)
	seedTransactionalTemplates = noop
	seedAccessibility          = noop // accessibility policy, Weekly Digest #143, the gallery's and the publish dialog's issues (seed_accessibility.go)
	// content languages: workspace languages, contacts' languages, versions of #142 and #143 (seed_languages.go)
	seedLanguages    = noop
	seedAppLanguages = noop // app languages: instance defaults, Marcus's region format, the invitation's language (seed_app_languages.go)
	// content languages of forms, public pages and contacts: where contacts' languages came from, page and form copy (seed_page_languages.go)
	seedPageLanguages = noop
)

func noop(context.Context, *Seeder) error { return nil }
