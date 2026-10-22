import { defineMessages } from '@/i18n'

/**
 * 7.2: starting, loading and resuming a setup (SetupWizard) and the summary
 * once it's done (SetupDone). The summary's title and its rows come from the
 * server.
 */
export const ms = defineMessages('connect.setup', {
  // ── Starting and loading a setup ─────────────────────────────────────
  backToCatalog: 'Back to the catalog',
  unknownAdapter: "This provider isn't in this build",
  unknownAdapterBody: 'Pick another provider from the catalog, or ask an instance admin to install its adapter.',
  startFailed: "Couldn't start the setup",
  starting: 'Starting the setup…',
  notFound: "This setup doesn't exist",
  loadFailed: "Couldn't load the setup",
  loading: 'Loading the setup…',
  cancelledReason: 'This setup was cancelled. Start again from the catalog.',
  cancelledNote: 'This setup was cancelled. Start again from the catalog to connect the provider.',
  /** {provider} is the provider's name: "Amazon SES". */
  savedForLater: 'Saved. Pick {provider} in the catalog to carry on.',
  saveFailed: "Couldn't save the setup",

  // ── The summary ──────────────────────────────────────────────────────
  doneBadge: 'Done',
  took: '{minutes, plural, one {Setup took # minute. Everything below is live.} other {Setup took # minutes. Everything below is live.}}',
  tookDomain:
    '{minutes, plural, one {Setup took # minute. Everything below is live, and the Overview now shows sender health for {domain}.} other {Setup took # minutes. Everything below is live, and the Overview now shows sender health for {domain}.}}',
  configuredTitle: 'What was configured',
  configuredLine: 'all reversible from Sending providers',
  importNow: 'Import now',
  recommendedTitle: 'Recommended before your first campaign',
  addProviderTitle: 'Add another provider',
  addProviderLine: 'A second provider for System mail, or a Postal server for a weighted split. Several can run at once.',
  addProvider: 'Add another provider',
  hostReputeTitle: 'Connect HostRepute',
  hostReputeConnected: 'Connected: 84 blocklists, Google Postmaster and Microsoft SNDS show inside Send Readiness.',
  hostReputeOptional: 'Optional, free: 84 blocklists, Google Postmaster and Microsoft SNDS inside Send Readiness.',
  open: 'Open',
  connect: 'Connect',
  formTitle: 'Create a signup form',
  formLine: 'Double opt-in is on by default; confirmations go out on the System stream.',
  buildForm: 'Build a form',
  goToOverview: 'Go to Overview',
})
