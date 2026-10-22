import { defineMessages } from '@/i18n'

/**
 * Instance admin › Audit log (AuditInstance): instance-level entries, the
 * export, retention and the instance event stream. The words it shares with
 * Settings › Audit log (actor types, results, date ranges, an entry's details,
 * the stream's status) come from features/audit/messages.
 */
export const m = defineMessages('admin.audit', {
  documentTitle: 'Audit log',
  eventsCaption: 'Denied events are highlighted. Sign-in rejections carry the reference ID users see on the error page.',
  tableCaption: 'Instance audit entries',
  noMatch: 'No events match these filters. Widen the date range or clear the actor filter.',

  // The bar above the events: scope, retention and export.
  instanceLevel: 'Instance level',
  instanceLevelCaption: "Sign-ins, SSO, policies, users and every workspace's events",
  retained: 'Retained {days, plural, one {# day} other {# days}} · immutable',
  retention: 'Retention',
  exportDownloaded: '{format} export downloaded',
  exportFailed: 'Export failed',
  exportTimedOut: 'The {format} export timed out after {seconds} s. Narrow the range or use the streaming endpoint for bulk access.',
  exportError: 'The {format} export failed: {error} Narrow the range or try again.',

  // Filters.
  searchPlaceholder: 'Search account, target or reference',
  filterWorkspace: 'Filter by workspace',
  allWorkspaces: 'All workspaces',
  instanceOnly: 'Instance only',

  // The Retention and streaming card.
  cardTitle: 'Retention and streaming',
  kvRetention: 'Retention',
  retentionValue:
    '{days, plural, one {# day} other {# days}}{isDefault, select, yes { (default)} other {}} · older instance-level entries are removed by the hourly cleanup; each workspace keeps its own',
  kvStreaming: 'Streaming',
  streamingValue:
    'Workspace events → {workspace, select, yes {{hosts} (HMAC-signed)} other {not streamed}} · instance events → {instance, select, on {{host} (HMAC-signed)} failing {{host} (HMAC-signed), failing} other {not streamed}}',
  kvExport: 'Export',
  exportValue: 'CSV or JSON, filtered or full',
  changeRetention: 'Change retention',
  streamButtonOn: 'Instance event stream',
  streamButtonOff: 'Stream instance events',

  // Change retention.
  retentionTitle: 'Instance audit retention',
  retentionBody: 'How long instance-level entries (sign-ins, SSO, policies and users) are kept. Each workspace keeps its own retention in its audit settings.',
  pickAnother: 'Pick another retention to save',
  retentionSaved: 'Instance entries are kept {days, plural, one {# day} other {# days}}',
  saveRetention: 'Save retention',
  keepFor: 'Keep instance-level entries for',
  retentionOption: '{days, plural, one {# day} other {# days}}{isDefault, select, yes { (default)} other {}}',
  retentionKept: '{count, plural, one {{total} instance-level entry is} other {{total} instance-level entries are}} kept now. Entries are immutable until they age out.',
  shorter: 'Shorter',
  shorterNote: 'Entries older than {days, plural, one {# day} other {# days}} are removed at the next hourly cleanup. Export them first if you need them.',

  // Stream instance events.
  streamTitle: 'Stream instance events',
  streamBody:
    'Post sign-ins, SSO, policy and user changes to your SIEM or warehouse as they happen, in signed batches. Workspaces stream their own events from their audit settings.',
  streamOn: 'Instance events are streamed',
  streamOff: 'Instance events are no longer streamed',
  secretInput: 'Instance audit stream signing secret',
  streamToggle: 'Stream instance-level entries to a webhook',
  workspaceStreams: 'Streaming their own events: {list}.',
  workspaceStream: '{name} → {host}{failing, select, yes { (failing)} other {}}',
})
