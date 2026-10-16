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
  { board: 'Composer', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?select=2` },
  { board: 'ComposerEmpty', path: `${ws}/campaigns/{campaign:Untitled campaign}/edit` },
  { board: 'ComposerStates', manual: true },
  { board: 'ComposerPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?select=2`, widths: [390] },
  { board: 'ComposerTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?select=2`, widths: [834] },
  { board: 'ComposerMobile', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?device=mobile&select=2-0&panel=settings&left=layers` },
  { board: 'ComposerWarnings', path: `${ws}/campaigns/{campaign:Flash sale: 48 hours only}/edit?select=1&panel=deliverability` },
  { board: 'ComposerCode', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?view=mjml` },
  { board: 'ComposerPreview', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/edit?view=preview` },
  // Daniel Park signs in with SSO only: reproduce by hand (a dev session for daniel@acme.io).
  { board: 'ComposerRequestApproval', manual: true },
  // Weekly Digest #142 is scheduled in the demo (SendReadinessStates · scheduled); SendReadiness is
  // the same campaign before scheduling, put back to draft in a dev database.
  { board: 'SendReadiness', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/review` },
  { board: 'SendReadinessBlocked', path: `${ws}/campaigns/{campaign:Flash sale: 48 hours only}/review` },
  // The confirmation opens from "Schedule send"; the approval states need Daniel (SSO only) or data changes.
  { board: 'SendReadinessConfirm', manual: true },
  { board: 'SendReadinessApproval', manual: true },
  { board: 'SendReadinessStates', manual: true },
  // The live boards need a send in progress: start one first with
  // `reputecast demo send "Weekly Digest #142" --elapsed 20m --reset` (normal
  // scenario for LiveMonitor*, --scenario outlook-complaints for
  // LiveMonitorPaused*, --scenario postal-outage --elapsed 17m for
  // LiveMonitorOutage), then sweep those boards with --only.
  { board: 'LiveMonitor', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live`, widths: [1440] },
  { board: 'LiveMonitorTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live`, widths: [834] },
  { board: 'LiveMonitorMobile', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live`, widths: [390] },
  { board: 'LiveMonitorPaused', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live`, widths: [1440, 834] },
  { board: 'LiveMonitorPausedPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live`, widths: [390] },
  { board: 'LiveMonitorOutage', path: `${ws}/campaigns/{campaign:Weekly Digest #142}/live` },
  // Connecting, stream lost and complete: reproduce by hand (see the README).
  { board: 'LiveMonitorStates', manual: true },
  // Marcus Chen signs in with SSO only; Linh is an approver too and sees the same list.
  { board: 'ApprovalInbox', path: `${ws}/approvals` },
  // The approval request's id is generated at seed time.
  { board: 'ApprovalReview', manual: true },
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
  // The demo's finished SES setup at each step, and the unfinished Mailgun one (SetupError).
  { board: 'SetupSES', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=connection` },
  { board: 'SetupError', path: `${ws}/providers/connect/mailgun?setup={setup:mailgun}&step=connection` },
  { board: 'SetupDomain', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=domain` },
  { board: 'SetupTestSend', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=test` },
  { board: 'SetupStreams', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=streams` },
  { board: 'SetupContacts', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=contacts` },
  { board: 'SetupDone', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=done` },
  // Needs a connected SMTP server and bounce mailbox in a dev database.
  { board: 'SetupSMTP', manual: true },
  { board: 'SetupCatalogPhone', path: `${ws}/providers/connect`, widths: [390] },
  { board: 'SetupCatalogTablet', path: `${ws}/providers/connect`, widths: [834] },
  { board: 'SetupSESPhone', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=connection`, widths: [390] },
  { board: 'SetupSESTablet', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=connection`, widths: [834] },
  { board: 'SetupDomainPhone', path: `${ws}/providers/connect/ses?setup={setup:ses}&step=domain`, widths: [390] },
  { board: 'Deliverability', path: `${ws}/deliverability` },
  { board: 'DeliverabilityNoHostRepute', manual: true },
  { board: 'DeliverabilityStates', manual: true },
  { board: 'ConnectHostRepute', manual: true },
  { board: 'HostReputeStates', manual: true },
  { board: 'Contacts', path: `${ws}/audience` },
  { board: 'ContactProfile', path: `${ws}/audience/contacts/{contact:sam.rivera@outlook.com}` },
  { board: 'SegmentBuilder', path: `${ws}/audience/segments` },
  { board: 'ListHealth', path: `${ws}/audience/health` },
  { board: 'Suppressions', path: `${ws}/audience/suppressions` },
  { board: 'AudienceStates', manual: true },
  { board: 'MigrationSource', path: `${ws}/import` },
  { board: 'MigrationFields', path: `${ws}/import/{import:draft}/fields` },
  { board: 'MigrationConsent', path: `${ws}/import/{import:draft}/consent` },
  { board: 'MigrationPreview', path: `${ws}/import/{import:draft}/preview` },
  { board: 'MigrationProgress', manual: true },
  { board: 'MigrationDone', path: `${ws}/import/{import:brevo}/report` },
  { board: 'MigrationStates', manual: true },
  { board: 'AutomationsList', path: `${ws}/automations` },
  // ?step= selects the email step as on the board; below 1100 px the deep link also scrolls
  // the step into view, which a click on it (as the board shows) doesn't.
  { board: 'AutomationFlow', path: `${ws}/automations/{automation:Welcome series}?step=n3` },
  { board: 'AutomationsStates', manual: true },
  { board: 'FormBuilder', path: `${ws}/forms` },
  { board: 'FormsStates', manual: true },
  { board: 'Notifications', path: `${ws}/notifications` },
  { board: 'AccountSecurity', path: `${ws}/account` },
  { board: 'AccountStates', manual: true },
  { board: 'WorkspaceSettings', path: `${ws}/settings` },
  { board: 'SettingsTeam', path: `${ws}/settings/team` },
  { board: 'TeamViewer', path: `${ws}/settings/team`, user: 'aisha@acme.io' },
  { board: 'TeamMember', path: `${ws}/settings/team/{member:daniel@acme.io}` },
  { board: 'Roles', path: `${ws}/settings/roles` },
  { board: 'RoleEditor', path: `${ws}/settings/roles/new?from={role:Content Writer}` },
  { board: 'SettingsKeys', path: `${ws}/settings/keys` },
  { board: 'SettingsMCP', path: `${ws}/settings/mcp` },
  { board: 'SettingsAPI', path: `${ws}/settings/webhooks` },
  { board: 'SettingsAudit', path: `${ws}/settings/audit` },
  { board: 'SettingsIntegrations', path: `${ws}/settings/integrations/hostrepute` },
  { board: 'SettingsSenders', path: `${ws}/settings/senders` },
  { board: 'SettingsSendersPhone', path: `${ws}/settings/senders`, widths: [390] },
  { board: 'SettingsSendersTablet', path: `${ws}/settings/senders`, widths: [834] },
  { board: 'SettingsProviders', path: `${ws}/settings/providers` },
  { board: 'SettingsStates', manual: true },
  { board: 'InstanceUsers', path: `${ws}/admin/users` },
  { board: 'InstanceUsersTablet', path: `${ws}/admin/users`, widths: [834] },
  { board: 'InstanceUsersPhone', path: `${ws}/admin/users`, widths: [390] },
  { board: 'InstanceUserDialogs', manual: true },
  { board: 'InstanceAuth', path: `${ws}/admin/authentication` },
  { board: 'InstanceSettings', path: `${ws}/admin/instance` },
  { board: 'SSOSetup', path: `${ws}/admin/sso/{sso:Microsoft Entra ID}` },
  { board: 'SSOSetupTablet', path: `${ws}/admin/sso/{sso:Microsoft Entra ID}`, widths: [834] },
  { board: 'SSOSetupPhone', path: `${ws}/admin/sso/{sso:Microsoft Entra ID}`, widths: [390] },
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
  // X01 · the phone tab bar's More sheet: open it with More at 390 × 844 (any screen; the board draws it over Overview).
  { board: 'ShellMore', manual: true },
  // X04 · the integrations hub. The connect dialog, the states and the WordPress board's dialogs need clicks or data changes.
  { board: 'IntegrationsCatalog', path: `${ws}/settings/integrations`, widths: [1440] },
  { board: 'IntegrationsCatalogTablet', path: `${ws}/settings/integrations`, widths: [834] },
  { board: 'IntegrationsCatalogPhone', path: `${ws}/settings/integrations`, widths: [390] },
  { board: 'IntegrationsZapier', path: `${ws}/settings/integrations/zapier`, widths: [1440] },
  { board: 'IntegrationsZapierTablet', path: `${ws}/settings/integrations/zapier`, widths: [834] },
  { board: 'IntegrationsZapierPhone', path: `${ws}/settings/integrations/zapier`, widths: [390] },
  { board: 'IntegrationsWordPress', path: `${ws}/settings/integrations/wordpress` },
  { board: 'IntegrationsConnect', manual: true },
  { board: 'IntegrationsStates', manual: true },
  // X03 · transactional templates. The editor board has reset_url's row open (click it in Variables).
  // The editor's states need edits, Daniel (SSO only) or Aisha; the publish states need the approval
  // rule on or a conflict; the import needs its steps (in development the Postmark token
  // pm-0f3c-recorded-8a21 answers from the recorded sample account).
  { board: 'TransactionalTemplate', path: `${ws}/transactional/templates/password-reset`, widths: [1440] },
  { board: 'TransactionalTemplateStates', manual: true },
  { board: 'TransactionalPublish', path: `${ws}/transactional/templates/trial-ending?publish=1`, widths: [1440] },
  { board: 'TransactionalPublishStates', manual: true },
  { board: 'TransactionalImport', manual: true },
  { board: 'TransactionalImportStates', manual: true },
  // X05 · content languages. The Languages card is on Settings › General; the boards draw it alone.
  { board: 'SettingsLanguages', path: `${ws}/settings`, widths: [1440] },
  { board: 'SettingsLanguagesTablet', path: `${ws}/settings`, widths: [834] },
  { board: 'SettingsLanguagesPhone', path: `${ws}/settings`, widths: [390] },
  { board: 'LanguageStates', manual: true },
  { board: 'ComposerLanguages', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [1440] },
  { board: 'ComposerLanguagesTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [834] },
  { board: 'ComposerLanguagesPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [390] },
  { board: 'ComposerCompare', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de&view=compare`, widths: [1440] },
  { board: 'ReportLanguages', manual: true },
  { board: 'AutomationLanguages', manual: true },
]
