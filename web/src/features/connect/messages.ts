import { defineMessages } from '@/i18n'

/**
 * The connect wizard's frame (Wizard.tsx: top bar, stepper) and the parts
 * every step shares (parts.tsx). Each step's own text is in its
 * `<screen>.messages.ts`.
 */
export const m = defineMessages('connect', {
  // ── Top bar ──────────────────────────────────────────────────────────
  exitSetup: 'Exit setup',
  stepBadge: 'Step {n} of 6',
  title: 'Connect a sending provider',

  // ── Stepper ──────────────────────────────────────────────────────────
  stepsLabel: 'Setup steps',
  /** A step behind the current one: "Done:" is read by screen readers only. */
  stepDone: '<vh>Done:</vh> {step}',
  stepProvider: 'Provider',
  stepProviderLine: 'Pick from the catalog',
  stepConnection: 'Connection',
  stepConnectionLine: 'Keys, region, webhooks',
  stepDomain: 'Domain verification',
  stepDomainLine: 'DKIM, return-path, SPF, DMARC',
  stepTest: 'Test send',
  stepTestLine: 'Watch the event arrive',
  stepStreams: 'Streams',
  stepStreamsLine: 'Broadcasts, Automations, System',
  stepContacts: 'Contacts',
  stepContactsLine: 'Import or start empty',
  /** Under "Streams" while it's the current step; the provider's short name ("SES"). */
  stepStreamsCurrent: 'What {provider} will carry',
  stepStreamsCurrentNoName: 'What it will carry',
  /** Under the stepper: how long setup takes, honestly (PR-15). */
  stepsFoot:
    "Most setups take 10–15 minutes; Amazon SES takes longer when ReputeCast doesn't run on AWS, and a new AWS account can wait up to 24 hours for production access. You can add more providers later and send through several at once.",

  // ── Status badges of a row ───────────────────────────────────────────
  stateDone: 'Done',
  /** Events arrived (SetupDone, PR-3). */
  stateVerified: 'Verified',
  stateFailed: 'Failed',
  stateSkipped: 'Skipped',
  statePending: 'Pending',
  stateYes: 'Yes',
  statePartial: 'Partial',
  stateNo: 'No',

  // ── Copying ──────────────────────────────────────────────────────────
  copied: '{what, select, zone {Copied the zone file} secret {Copied the webhook secret} url {Copied the webhook URL} other {Copied the {label} value}}',
  copyFailed:
    "{what, select, zone {Couldn't copy the zone file} secret {Couldn't copy the webhook secret} url {Couldn't copy the webhook URL} other {Couldn't copy the {label} value}}",
  copyBlocked: 'Your browser blocked the clipboard. Select the text and copy it instead.',

  // ── Read-only notes ──────────────────────────────────────────────────
  readOnlyCatalog: "You can look at the catalog and at setups others started. Your role can't connect providers.",
  readOnlySetup: "You can follow this setup. Your role can't connect or change providers.",
})
