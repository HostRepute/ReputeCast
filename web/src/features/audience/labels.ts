import type { Tone } from '@/components/ui/Badge'

/** Engagement tiers, in the order the boards list them. */
export type Tier = 'active' | 'cooling' | 'at_risk' | 'sunset'
export const tiers: Tier[] = ['active', 'cooling', 'at_risk', 'sunset']

export const tierLabel: Record<Tier, string> = { active: 'Active', cooling: 'Cooling', at_risk: 'At risk', sunset: 'Sunset' }

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
  active: 'opened in 30 days',
  cooling: '31 to 90 days',
  at_risk: '91 to 180 days',
  sunset: 'no open in 180 days',
}

export type Consent = 'double' | 'single' | 'pending'
export const consentLabel: Record<Consent, string> = { double: 'Double opt-in', single: 'Single opt-in', pending: 'Imported · pending' }
export const consentTone: Record<Consent, Tone> = { double: 'ok', single: 'ok', pending: 'warn' }
/** Labels in the consent filter. */
export const consentFilterLabel: Record<Consent, string> = { double: 'Double opt-in', single: 'Single opt-in', pending: 'Imported, pending' }

export type Mailbox = 'gmail' | 'outlook' | 'yahoo' | 'icloud' | 'other'
export const mailboxes: Mailbox[] = ['gmail', 'outlook', 'yahoo', 'icloud', 'other']
export const mailboxLabel: Record<Mailbox, string> = { gmail: 'Gmail', outlook: 'Outlook', yahoo: 'Yahoo', icloud: 'iCloud', other: 'Other' }
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
