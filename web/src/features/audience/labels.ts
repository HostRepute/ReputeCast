import type { Tone } from '@/components/ui/Badge'
import { t } from '@/i18n'
import { m } from './messages'

/** Engagement tiers, in the order the boards list them. */
export type Tier = 'active' | 'cooling' | 'at_risk' | 'sunset'
export const tiers: Tier[] = ['active', 'cooling', 'at_risk', 'sunset']

// Getters, so each read is in the app's current language.
export const tierLabel: Record<Tier, string> = {
  get active() {
    return t(m.tierActive)
  },
  get cooling() {
    return t(m.tierCooling)
  },
  get at_risk() {
    return t(m.tierAtRisk)
  },
  get sunset() {
    return t(m.tierSunset)
  },
}

/** Tier badge classes (.t1–.t4 in audience.css): darkest blue is Active. */
export const tierClass: Record<Tier, string> = { active: 't1', cooling: 't2', at_risk: 't3', sunset: 't4' }

export const tierColor: Record<Tier, string> = {
  active: 'var(--chart-tier-1)',
  cooling: 'var(--chart-tier-2)',
  at_risk: 'var(--chart-tier-3)',
  sunset: 'var(--chart-tier-4)',
}

/** The tier windows under the default sunset policy, as on the List health cards. */
export const tierWindow: Record<Tier, string> = {
  get active() {
    return t(m.windowActive)
  },
  get cooling() {
    return t(m.windowCooling)
  },
  get at_risk() {
    return t(m.windowAtRisk)
  },
  get sunset() {
    return t(m.windowSunset)
  },
}

export type Consent = 'double' | 'single' | 'pending'
export const consentLabel: Record<Consent, string> = {
  get double() {
    return t(m.consentDouble)
  },
  get single() {
    return t(m.consentSingle)
  },
  get pending() {
    return t(m.consentPending)
  },
}
export const consentTone: Record<Consent, Tone> = { double: 'ok', single: 'ok', pending: 'warn' }
export const consentFilterLabel: Record<Consent, string> = {
  get double() {
    return t(m.consentDouble)
  },
  get single() {
    return t(m.consentSingle)
  },
  get pending() {
    return t(m.consentPendingFilter)
  },
}

export type Mailbox = 'gmail' | 'outlook' | 'yahoo' | 'icloud' | 'other'
export const mailboxes: Mailbox[] = ['gmail', 'outlook', 'yahoo', 'icloud', 'other']
export const mailboxLabel: Record<Mailbox, string> = {
  gmail: 'Gmail',
  outlook: 'Outlook',
  yahoo: 'Yahoo',
  icloud: 'iCloud',
  get other() {
    return t(m.mailboxOther)
  },
}
export const mailboxColor: Record<string, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

/** Sending provider adapters and their series colours. */
export const providerColor: Record<string, string> = {
  ses: 'var(--chart-ses)',
  postal: 'var(--chart-postal)',
  postmark: 'var(--chart-postmark)',
}
