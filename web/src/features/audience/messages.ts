import { defineMessages } from '@/i18n'

/**
 * Words the audience screens share: the section tabs, engagement tiers,
 * consent and mailbox providers (labels.ts). Each screen's own text is in
 * its `<screen>.messages.ts`.
 */
export const m = defineMessages('audience', {
  // ── Section tabs ─────────────────────────────────────────────────────
  title: 'Audience',
  sections: 'Audience sections',
  tabContacts: 'Contacts',
  tabContactsCount: 'Contacts · {count}',
  tabSegments: 'Segments',
  tabSegmentsCount: 'Segments · {count}',
  tabHealth: 'List health',
  tabSuppressions: 'Suppression list',
  tabSuppressionsCount: 'Suppression list · {count}',

  // ── Engagement tiers ─────────────────────────────────────────────────
  tierActive: 'Active',
  tierCooling: 'Cooling',
  tierAtRisk: 'At risk',
  tierSunset: 'Sunset',
  windowActive: 'opened in 30 days',
  windowCooling: '31 to 90 days',
  windowAtRisk: '91 to 180 days',
  windowSunset: 'no open in 180 days',

  // ── Consent ──────────────────────────────────────────────────────────
  consentDouble: 'Double opt-in',
  consentSingle: 'Single opt-in',
  consentPending: 'Imported · pending',
  consentPendingFilter: 'Imported, pending',

  // ── Mailbox providers ────────────────────────────────────────────────
  mailboxOther: 'Other',
})
