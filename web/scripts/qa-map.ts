/**
 * Which app screen reproduces which design board, for the QA sweep
 * (scripts/qa.ts). Paths may contain placeholders resolved through the API
 * of the demo workspace, e.g. {campaign:October Product Update}.
 *
 * Boards whose states need data changes (…States, outage, empty) are listed
 * with `manual: true`: the sweep captures only the design, and the state is
 * reproduced by hand in a dev database (see docs/dev/CONVENTIONS.md §6).
 */
export type Entry = {
  /** Board file name in design/boards, without .dc.html. */
  board: string
  /** App path; omit for design-only entries. */
  path?: string
  /** Demo user to sign in as (default linh@acme.io); 'anon' to stay signed out. */
  user?: string
  /** Widths to compare (default 1440, 834, 390). */
  widths?: number[]
  /** The state needs data changes: capture the design only. */
  manual?: boolean
  /** Capture only the viewport (for boards that show one screen of a long page). */
  viewportOnly?: boolean
  /** Viewport height when viewportOnly is set. */
  height?: number
}

const ws = '/w/acme-newsletter'

export const entries: Entry[] = [
  { board: 'Main', path: `${ws}` },
  { board: 'OverviewEmpty', manual: true },
  { board: 'OverviewStates', manual: true },
  { board: 'Campaigns', path: `${ws}/campaigns` },
  { board: 'Report', path: `${ws}/campaigns/{campaign:October Product Update}/report` },
  { board: 'ReportStates', manual: true },
  { board: 'Composer', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit` },
  { board: 'ComposerEmpty', path: `${ws}/campaigns/{campaign:Untitled campaign}/edit` },
  { board: 'ComposerStates', manual: true },
  { board: 'SendReadiness', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/review` },
  { board: 'SendReadinessBlocked', path: `${ws}/campaigns/{campaign:Flash sale: 48 hours only}/review` },
  { board: 'SendReadinessStates', manual: true },
  { board: 'LiveMonitor', manual: true },
  { board: 'LiveMonitorPaused', manual: true },
  { board: 'LiveMonitorOutage', manual: true },
  { board: 'LiveMonitorStates', manual: true },
  { board: 'ApprovalInbox', path: `${ws}/approvals`, user: 'marcus@acme.io' },
  { board: 'ApprovalStates', manual: true },
  { board: 'ApprovalPolicy', path: `${ws}/settings/approvals` },
  { board: 'Providers', path: `${ws}/providers` },
  { board: 'ProvidersRouting', path: `${ws}/providers/routing` },
  { board: 'ProvidersWarmup', path: `${ws}/providers/warmup` },
  { board: 'ProvidersEvents', path: `${ws}/providers/events` },
  { board: 'ProvidersAdapters', path: `${ws}/providers/adapters` },
  { board: 'ProvidersOutage', manual: true },
  { board: 'ProvidersWebhooksSilent', manual: true },
  { board: 'ProvidersBulkWarning', manual: true },
  { board: 'ProvidersStates', manual: true },
  { board: 'SetupCatalog', path: `${ws}/providers/connect` },
  { board: 'Deliverability', path: `${ws}/deliverability` },
  { board: 'DeliverabilityNoHostRepute', manual: true },
  { board: 'DeliverabilityStates', manual: true },
  { board: 'Contacts', path: `${ws}/audience` },
  { board: 'SegmentBuilder', manual: true },
  { board: 'ListHealth', path: `${ws}/audience/health` },
  { board: 'Suppressions', path: `${ws}/audience/suppressions` },
  { board: 'AudienceStates', manual: true },
  { board: 'MigrationSource', path: `${ws}/import` },
  { board: 'MigrationStates', manual: true },
  { board: 'AutomationsList', path: `${ws}/automations` },
  { board: 'AutomationsStates', manual: true },
  { board: 'FormBuilder', path: `${ws}/forms` },
  { board: 'FormsStates', manual: true },
  { board: 'Notifications', path: `${ws}/notifications` },
  { board: 'AccountSecurity', path: `${ws}/account` },
  { board: 'AccountStates', manual: true },
  { board: 'WorkspaceSettings', path: `${ws}/settings` },
  { board: 'SettingsTeam', path: `${ws}/settings/team` },
  { board: 'Roles', path: `${ws}/settings/roles` },
  { board: 'SettingsKeys', path: `${ws}/settings/keys` },
  { board: 'SettingsMCP', path: `${ws}/settings/mcp` },
  { board: 'SettingsAPI', path: `${ws}/settings/webhooks` },
  { board: 'SettingsAudit', path: `${ws}/settings/audit` },
  { board: 'SettingsIntegrations', path: `${ws}/settings/integrations` },
  { board: 'SettingsSenders', path: `${ws}/settings/senders` },
  { board: 'SettingsProviders', path: `${ws}/settings/providers` },
  { board: 'SettingsStates', manual: true },
  { board: 'InstanceUsers', path: `${ws}/admin/users` },
  { board: 'InstanceAuth', path: `${ws}/admin/authentication` },
  { board: 'InstanceSettings', path: `${ws}/admin/instance` },
  { board: 'SSOSetup', path: `${ws}/admin/sso` },
  { board: 'SSOStates', manual: true },
  { board: 'AuditInstance', path: `${ws}/admin/audit` },
  { board: 'AuthScreens', path: '/sign-in', user: 'anon' },
  { board: 'SignInStates', manual: true },
  { board: 'TwoFactorStates', manual: true },
  { board: 'InviteStates', manual: true },
  { board: 'InstanceSetup', manual: true },
  { board: 'SystemStates', manual: true },
  { board: 'Dialogs', manual: true },
  { board: 'Feedback', manual: true },
  { board: 'PublicSubscribe', path: '/p/acme-newsletter/subscribe', user: 'anon' },
  { board: 'PublicArchive', path: '/p/acme-newsletter/archive', user: 'anon' },
  { board: 'PublicConfirm', manual: true },
  { board: 'PublicPreferences', manual: true },
  { board: 'PublicUnsubscribe', manual: true },
]
