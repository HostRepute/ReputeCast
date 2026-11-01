import { defineMessages } from '@/i18n'

/**
 * Instance admin › System (InstanceSystem; review 2026-11 OP-10): failed
 * background jobs with retry, cancel and delete, the queues, the largest
 * tables and the files folder. Job kinds, queue and table names, paths and
 * error text are arguments, never translated.
 */
export const m = defineMessages('admin.system', {
  documentTitle: 'System',
  loading: 'Loading the system',
  loadFailed: "Couldn't read the system",

  // ── Tiles ─────────────────────────────────────────────────────────────
  tileWaiting: 'Jobs waiting',
  // seconds: how long the oldest job that is due has waited.
  tileWaitingLag: 'oldest due job waited {seconds}',
  tileWaitingNone: 'none waits',
  tileRunning: 'Running',
  tileRunningSub: 'jobs being worked now',
  tileFailed: 'Failed · last 7 days',
  tileFailedSub: '{retrying, number} retrying, {givenUp, number} given up',
  tileDatabase: 'Database',
  // version: "16.4"; connections and max: numbers.
  tileDatabaseSub: 'PostgreSQL {version} · {connections, number} of {max, number} connections',
  tileFiles: 'Files folder',
  tileFilesSub: '{count, plural, one {# file} other {# files}}',
  tileFilesNone: 'nothing uploaded yet',

  // ── Failed jobs ───────────────────────────────────────────────────────
  failedTitle: 'Failed jobs',
  failedTitleCount: 'Failed jobs · {count}',
  failedCaption: 'Jobs that failed and wait for a retry, and jobs that ran out of attempts. Given-up jobs are kept 7 days.',
  failedShow: 'Show',
  showAll: 'Failed and retrying',
  showRetrying: 'Retrying',
  showGivenUp: 'Given up',
  failedTable: 'Failed jobs',
  colJob: 'Job',
  colWorkspace: 'Workspace',
  colState: 'State',
  colAttempts: 'Attempts',
  colError: 'Last error',
  colFailed: 'Failed',
  colActions: 'Actions',
  // queue: "events"; id: the job's number.
  jobWhere: 'queue {queue} · #{id}',
  noWorkspace: 'Instance',
  stateRetrying: 'Retrying',
  stateGivenUp: 'Given up',
  attempts: '{attempt, number} of {max, number}',
  nextTry: 'Next try at {time}',
  noError: 'No error recorded',
  retryNow: 'Retry now',
  retry: 'Retry',
  cancel: 'Cancel',
  delete: 'Delete',
  // job: "webhooks.deliver #48213"
  retryLabel: 'Retry {job}',
  cancelLabel: 'Cancel {job}',
  deleteLabel: 'Delete {job}',
  retried: '{job} runs again',
  noFailed: 'No failed jobs. Jobs that fail show here with their error.',
  noFailedState: 'None in this state.',
  loadMore: 'Load 50 more',
  failedFootnote: 'Retrying runs the job again now, with its attempts so far. Every retry, cancel and delete is in the instance audit log.',
  cancelBadge: 'Cancel job',
  cancelTitle: 'Cancel {job}?',
  cancelBody: 'It won’t be tried again. What it was doing stays undone; it stays in this list as given up for 7 days.',
  cancelling: 'Cancelling…',
  cancelled: '{job} cancelled',
  keepIt: 'Keep it',
  deleteBadge: 'Delete job',
  deleteTitle: 'Delete {job}?',
  deleteBody: 'It leaves this list. What it was doing stays undone.',
  deleting: 'Deleting…',
  deleted: '{job} deleted',

  // ── Queues ────────────────────────────────────────────────────────────
  queuesTitle: 'Queues',
  queuesCaption: 'Read from the database just now · alert on lag in Prometheus with reputecast_job_queue_lag_seconds',
  queuesTable: 'Queues',
  colQueue: 'Queue',
  colWhat: 'What runs there',
  colWaiting: 'Waiting',
  colScheduled: 'Scheduled',
  colRunning: 'Running',
  colLag: 'Oldest due job waited',
  queueSending: 'Campaign and automation sends',
  queueEvents: 'Provider webhooks, outgoing webhooks',
  queueDefault: 'Imports, exports, digests, reports',
  queueSystem: 'System mail, notifications',
  queueMaintenance: 'Retention, cleanup, health checks',
  // seconds: a whole number.
  seconds: '{n, number} s',
  // minutes and seconds: "2 min 05 s".
  minutes: '{m, number} min {s} s',
  hours: '{h, number} h {m, number} min',

  // ── Database ──────────────────────────────────────────────────────────
  tablesTitle: 'Largest tables',
  // size: "4.2 GB"; version: the schema's migration number.
  tablesCaption: '{size} in all · schema {version}, {current, select, yes {current} other {migrations pending}}',
  tablesTable: 'Largest tables',
  colTable: 'Table',
  colRows: 'Rows (about)',
  colSize: 'Size',
  colIndexes: 'Of which indexes',

  // ── Files ─────────────────────────────────────────────────────────────
  filesTitle: 'Files folder',
  filesCaption: 'Uploads live here, not in the database: back the folder up with it',
  filesFolder: 'Folder',
  filesHolds: 'Holds',
  // count: files; size: "1.3 GB".
  filesHoldsValue: '{count, plural, one {# file} other {# files}} · {size}',
  filesHoldsPartial: 'at least {count, plural, one {# file} other {# files}} · {size} (the count stopped early)',
  filesMissing: 'The folder doesn’t exist yet: it is made with the first upload.',
  filesError: 'Couldn’t read it: {error}',
  filesCounted: 'Counted',
  filesCountedAt: 'at {time} · counted again every 5 minutes',
  filesShared: 'Several processes',
  filesSharedValue: 'Every web and worker process needs the same folder.',

  cliTitle: 'From the command line',
  cliBody: '{doctor} checks the database, the secret key and this folder after a restore; {reseal} moves stored secrets to a new key.',
})
