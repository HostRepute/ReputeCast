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
  { board: 'FormBuilder', path: `${ws}/forms?form={form:Newsletter signup}` },
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
  // X06 · accessibility. Settings › General shows the Accessibility card (and X05's Languages card); the board is
  // the page scrolled to them. Review & send of Weekly Digest #143 has the Accessibility group (X05 adds Languages).
  { board: 'SettingsLanguages', path: `${ws}/settings` },
  { board: 'SettingsLanguagesTablet', path: `${ws}/settings`, widths: [834] },
  { board: 'SettingsLanguagesPhone', path: `${ws}/settings`, widths: [390] },
  { board: 'SendReadinessAccessibility', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/review`, widths: [1440, 834] },
  { board: 'SendReadinessAccessibilityPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/review`, widths: [390] },
  // The policy set to Block, the one-click preview, the checks running or failing: switch the policy on
  // Settings › General or through PUT …/accessibility/policy, then reproduce by hand.
  { board: 'AccessibilityStates', manual: true },
  // X06 · the composer's Accessibility tab on Weekly Digest #143; the color picker opens from a field (manual).
  { board: 'ComposerAccessibility', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?select=1-0-0&panel=accessibility`, widths: [1440] },
  { board: 'ComposerAccessibilityTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?panel=accessibility`, widths: [834] },
  { board: 'ComposerAccessibilityPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?panel=accessibility`, widths: [390] },
  { board: 'ComposerContrast', manual: true },
  // X05 · content languages (the Settings › General entries above cover its Languages card).
  { board: 'LanguageStates', manual: true },
  { board: 'ComposerLanguages', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [1440] },
  { board: 'ComposerLanguagesTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [834] },
  { board: 'ComposerLanguagesPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de`, widths: [390] },
  { board: 'ComposerCompare', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?language=de&view=compare`, widths: [1440] },
  { board: 'ReportLanguages', manual: true },
  { board: 'AutomationLanguages', manual: true },
  // X07 · app languages. Language & region sits under Profile; its states (picked and not saved,
  // saved in Tiếng Việt with Undo, the browser's suggestion, the pseudo-locale) come from picking.
  { board: 'LanguageRegion', path: `${ws}/account`, viewportOnly: true, height: 1000 },
  { board: 'LanguageRegionPhone', path: `${ws}/account`, widths: [390], viewportOnly: true, height: 844 },
  { board: 'ShellUserMenu', manual: true },
  { board: 'TeamInvite', manual: true },
  // The stress tests: pick Tiếng Việt or Pseudo (+40%) in Account & security › Language & region,
  // then compare Campaigns, Settings › Team and Weekly Digest #142's Review & send.
  { board: 'StressCampaignsVi', manual: true },
  { board: 'StressCampaignsPseudo', manual: true },
  { board: 'StressCampaignsPseudoPhone', manual: true },
  { board: 'StressTeamVi', manual: true },
  { board: 'StressTeamPseudo', manual: true },
  { board: 'StressTeamPseudoPhone', manual: true },
  { board: 'StressReviewVi', manual: true },
  { board: 'StressReviewPseudo', manual: true },
  { board: 'StressReviewPseudoPhone', manual: true },
  // X08 · content languages of forms, public pages and contacts. The builder in Spanish; its
  // FormsStates frames (the consent text after English changes, Vietnamese not set up) come from
  // changing the English consent text and picking Tiếng Việt.
  { board: 'FormBuilderLanguages', path: `${ws}/forms?form={form:Newsletter signup}&language=es`, widths: [1440] },
  // X02 · transactional email. Sam Rivera's password reset has the board's message ID; the log's
  // second row (Jordan, rejected) opens with a click. The guide's step 5 is delivered after "Send it
  // from here instead"; its other states (waiting, rejected, done) and the boards' states (first run,
  // alert, no route, outage, read-only, retention, relay off or unreachable) need data changes.
  { board: 'Transactional', path: `${ws}/transactional`, widths: [1440] },
  { board: 'TransactionalTablet', path: `${ws}/transactional`, widths: [834] },
  { board: 'TransactionalPhone', path: `${ws}/transactional`, widths: [390] },
  { board: 'TransactionalStates', manual: true },
  { board: 'TransactionalGuideSteps', path: `${ws}/transactional/guide?step=1`, widths: [1440], viewportOnly: true, height: 900 },
  { board: 'TransactionalGuide', manual: true },
  { board: 'TransactionalGuideStates', manual: true },
  { board: 'TransactionalMessages', path: `${ws}/transactional/messages`, widths: [1440] },
  { board: 'TransactionalMessagesTablet', path: `${ws}/transactional/messages`, widths: [834] },
  { board: 'TransactionalMessagesPhone', path: `${ws}/transactional/messages`, widths: [390] },
  { board: 'TransactionalMessage', path: `${ws}/transactional/messages/0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44`, widths: [1440] },
  { board: 'TransactionalMessagePhone', path: `${ws}/transactional/messages/0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44`, widths: [390] },
  { board: 'TransactionalMessageStates', manual: true },
  { board: 'TransactionalSetup', path: `${ws}/transactional/setup`, widths: [1440] },
  { board: 'TransactionalSetupStates', manual: true },
  { board: 'TransactionalSettings', path: `${ws}/transactional/settings`, widths: [1440] },
  { board: 'PublicUnsubscribeCategory', manual: true },
  // X13 · contact scoring and growth sources. The states (an edited rule, the save dialog,
  // recalculating, stopped, the rule panel, read-only as aisha@acme.io, no rules on Acme
  // Engineering Blog; imports included, one list, a flagged API key, the referring pages, only
  // imports, loading and error) come from interacting with the page or another workspace;
  // ScoringUses gathers changes to Contacts, the segment builder, the flow editor and X11's
  // Visibility.
  { board: 'ScoringRules', path: `${ws}/audience/scoring`, widths: [1440, 390] },
  { board: 'ScoringRulesTablet', path: `${ws}/audience/scoring`, widths: [834] },
  { board: 'ScoringStates', manual: true },
  { board: 'ScoringProfile', path: `${ws}/audience/contacts/{contact:sam.rivera@outlook.com}`, widths: [1440, 834] },
  { board: 'ScoringProfilePhone', path: `${ws}/audience/contacts/{contact:sam.rivera@outlook.com}`, widths: [390] },
  { board: 'ScoringUses', manual: true },
  { board: 'GrowthSources', path: `${ws}/audience/growth`, widths: [1440] },
  { board: 'GrowthSourcesTablet', path: `${ws}/audience/growth`, widths: [834] },
  { board: 'GrowthSourcesPhone', path: `${ws}/audience/growth`, widths: [390] },
  { board: 'GrowthStates', manual: true },
  // X10 · recurring digests. The Engineering blog roundup (#3 collecting two posts, #2 and #1 sent),
  // its settings and its template in the composer's digest mode. The overview's states (waiting,
  // held, feed failing, sending, paused, skipped, draft mode, first run), the setup's states (feed
  // errors, a site's feed, a schedule by post count) and the composer's states (6 items, "and 0
  // more", the token menu, a post without an image, a prepared issue, read-only) need data changes
  // or clicks: "6 items" and the token menu are a pick in the composer.
  { board: 'DigestOverview', path: `${ws}/campaigns/digests/{digest:Engineering blog roundup}`, widths: [1440] },
  { board: 'DigestOverviewTablet', path: `${ws}/campaigns/digests/{digest:Engineering blog roundup}`, widths: [834] },
  { board: 'DigestOverviewPhone', path: `${ws}/campaigns/digests/{digest:Engineering blog roundup}`, widths: [390] },
  { board: 'DigestStates', manual: true },
  { board: 'DigestSetup', path: `${ws}/campaigns/digests/{digest:Engineering blog roundup}/settings`, widths: [1440] },
  { board: 'DigestSetupTablet', path: `${ws}/campaigns/digests/{digest:Engineering blog roundup}/settings`, widths: [834] },
  { board: 'DigestSetupStates', manual: true },
  { board: 'DigestComposer', path: `${ws}/campaigns/{digestTemplate:Engineering blog roundup}/edit?select=2&panel=settings`, widths: [1440], viewportOnly: true, height: 1000 },
  // X11 · conditional content, polls and surveys. Weekly Digest #143's Beta corner is section 7 and
  // its topics poll section 9; the composer's other frames (the picker open, a deleted segment, an
  // empty version, more than 50 versions, Gmail's clipping size) and the answer pages (signed links,
  // see internal/publicpages/polls_test.go) need data changes or a signed token.
  { board: 'ComposerVisibility', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?select=7&panel=settings&left=layers`, widths: [1440] },
  { board: 'VisibilityStates', manual: true },
  { board: 'ComposerPoll', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit?select=9&panel=settings`, widths: [1440] },
  { board: 'PublicPoll', manual: true },
  { board: 'PublicPollPhone', manual: true, widths: [390] },
  { board: 'PollResults', path: `${ws}/campaigns/{campaign:Weekly Digest #141}/report`, widths: [1440] },
  { board: 'SurveyBuilder', path: `${ws}/forms/surveys/{survey:Beta feedback · October 2026}?q=q3`, widths: [1440] },
  // X14 · address verification and placement tests. VerificationRun's job page is the Sunset tier's
  // check; its dialog opens from List health's tier cards. PlacementRun's first frame opens from
  // Review & send through the URL an entry point builds; its other frames (composer, warm-up planner,
  // template editor, a provider that is down, no verified sender) and the …States boards need clicks
  // or data changes.
  { board: 'VerificationServices', path: `${ws}/settings/integrations/verification`, widths: [1440] },
  { board: 'VerificationRun', path: `${ws}/audience/verification/{verification:Sunset}`, widths: [1440] },
  { board: 'VerificationStates', manual: true },
  { board: 'PlacementTests', path: `${ws}/deliverability/placement`, widths: [1440] },
  {
    board: 'PlacementRun',
    path: `${ws}/deliverability/placement?run=review&kind=campaign&content={campaign:Weekly Digest #142}&through=compare&from=Weekly%20Digest%20%23142`,
    widths: [1440],
    viewportOnly: true,
    height: 1000,
  },
  { board: 'PlacementResult', path: `${ws}/deliverability/placement/{placement:Weekly Digest #142 · SES vs Postal}`, widths: [1440] },
  { board: 'PlacementResultTablet', path: `${ws}/deliverability/placement/{placement:Weekly Digest #142 · SES vs Postal}`, widths: [834] },
  { board: 'PlacementResultPhone', path: `${ws}/deliverability/placement/{placement:Weekly Digest #142 · SES vs Postal}`, widths: [390] },
  { board: 'PlacementSeeds', path: `${ws}/deliverability/placement/seeds`, widths: [1440] },
  { board: 'PlacementStates', manual: true },
  { board: 'PlacementSeedStates', manual: true },
  // X12 · the AI assistant. Acme Newsletter has the demo's sample Anthropic connection (answers from
  // recordings in development); Acme Engineering Blog has none, for the invitation and the connect
  // dialog (?connect=anthropic). The board's other frames (test results, a private server allowed or
  // refused, read-only, disconnect) and AIStates' cap and key states need clicks or data changes.
  { board: 'AISettings', path: `${ws}/settings/integrations/ai`, widths: [1440] },
  { board: 'AIStates', manual: true },
  // The composer with the assistant: Weekly Digest #143, the draft. Its frames (Ideas, a rewrite
  // shown inline, alt text, the accessibility fixes, a translation's blocks) open from the Assist
  // menu and the selection's toolbar, so the sweep compares the composer around them.
  { board: 'AIComposer', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit`, widths: [1440], viewportOnly: true, height: 1000 },
  { board: 'AIComposerTablet', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit`, widths: [834], viewportOnly: true, height: 1000 },
  { board: 'AIComposerPhone', path: `${ws}/campaigns/{campaign:Weekly Digest #143}/edit`, widths: [390], viewportOnly: true, height: 1000 },
  // Describe the segment (a new segment) and Summarize results (the October Product Update's report,
  // with its kept summary): each starts from a click.
  { board: 'AISegmentReport', manual: true },
  // X16 · payments and paid subscriptions. Acme Newsletter has Stripe connected (live and test keys) and
  // 1,252 paid members; Acme Engineering Blog has no payment provider (the wizard's start, the first-run
  // checklist, no paid members). The editors, the member panel and the tabs of Settings open from the
  // pages; the public pages carry personal links (preferences) or need a checkout session.
  { board: 'EarnConnect', path: '/w/acme-engineering-blog/earn/connect', widths: [1440] },
  { board: 'EarnConnectStates', manual: true },
  { board: 'EarnSettings', path: `${ws}/earn/settings`, widths: [1440] },
  { board: 'EarnSettingsTabs', path: `${ws}/earn/settings?tab=emails`, widths: [1440] },
  { board: 'EarnSettingsStates', manual: true },
  { board: 'EarnPlans', path: `${ws}/earn/plans`, widths: [1440] },
  { board: 'EarnPlansTablet', path: `${ws}/earn/plans`, widths: [834] },
  { board: 'EarnPlansPhone', path: `${ws}/earn/plans`, widths: [390] },
  { board: 'EarnPlansStates', manual: true },
  { board: 'EarnPlanEditors', manual: true },
  { board: 'EarnUpgradeEditor', path: `${ws}/earn/plans/upgrade-page`, widths: [1440] },
  { board: 'EarnMembers', path: `${ws}/earn/members`, widths: [1440] },
  { board: 'EarnMembersTablet', path: `${ws}/earn/members`, widths: [834] },
  { board: 'EarnMembersPhone', path: `${ws}/earn/members`, widths: [390] },
  { board: 'EarnMembersStates', manual: true },
  { board: 'EarnMemberPanel', manual: true },
  { board: 'EarnMemberPanelStates', manual: true },
  { board: 'EarnImport', path: `${ws}/earn/members?import=new`, widths: [1440] },
  { board: 'EarnImportStates', manual: true },
  { board: 'EarnInAudience', path: `${ws}/audience?paid=with_access`, widths: [1440] },
  { board: 'ComposerEarnBlocks', manual: true },
  { board: 'PublicUpgrade', path: '/p/acme-newsletter/upgrade', user: 'anon', widths: [1440, 720] },
  { board: 'PublicUpgradePhone', path: '/p/acme-newsletter/upgrade', user: 'anon', widths: [390] },
  { board: 'PublicPreferencesPaid', manual: true },
  { board: 'PublicEarnStates', manual: true },
  // X18 · the publication website. The demo's website is on at news.acme.io. WebsiteStates holds
  // the off and turning-on pages, address problems, read-only views and the elsewhere-in-the-app
  // frames: reproduce them by hand (turn the website off in a dev database, sign in as aisha@acme.io).
  { board: 'WebsitePosts', path: `${ws}/forms/website`, widths: [1440] },
  { board: 'WebsitePostsTablet', path: `${ws}/forms/website`, widths: [834] },
  { board: 'WebsitePostsPhone', path: `${ws}/forms/website`, widths: [390] },
  { board: 'WebsiteHomeNav', path: `${ws}/forms/website/home`, widths: [1440] },
  { board: 'WebsiteAccess', path: `${ws}/forms/website/access`, widths: [1440] },
  { board: 'WebsiteStates', manual: true },
  // Post mode (the runbook, with its free preview marker) and campaign mode (#142's Web tab).
  { board: 'ComposerPublishToWeb', path: `${ws}/forms/website/posts/{post:Inside our Postgres migration: the runbook}`, widths: [1440], viewportOnly: true, height: 1000 },
  { board: 'ComposerPublishToWebPhone', path: `${ws}/forms/website/posts/{post:Inside our Postgres migration: the runbook}`, widths: [390], viewportOnly: true, height: 1000 },
  // The public site at /p/acme-newsletter, signed out (the runbook stops at the paywall). The
  // account needs a member's session (sign in on the site with a link from Mailpit) and the site
  // states need data changes: both by hand.
  { board: 'PublicHome', path: '/p/acme-newsletter/', user: 'anon', widths: [720] },
  { board: 'PublicHomePhone', path: '/p/acme-newsletter/', user: 'anon', widths: [390] },
  { board: 'PublicPost', path: '/p/acme-newsletter/posts/postgres-migration-runbook', user: 'anon', widths: [720] },
  { board: 'PublicPostPhone', path: '/p/acme-newsletter/posts/postgres-migration-runbook', user: 'anon', widths: [390] },
  { board: 'PublicAccount', manual: true },
  { board: 'PublicAccountPhone', manual: true },
  { board: 'PublicSiteStates', manual: true },
]
