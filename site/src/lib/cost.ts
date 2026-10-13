/**
 * The cost calculator from the Landing board: monthly sending cost at the
 * visitor's volume, with every price editable. Defaults are example list
 * prices for pay-as-you-go sending; the page says to check each provider.
 */
export interface CostInputs {
  vol: string
  ses: string
  resend: string
  mailgun: string
  postmark: string
  postal: string
  brevo: string
  mailchimp: string
  kit: string
}

export type CostField = keyof CostInputs

export const DEFAULT_INPUTS: CostInputs = {
  vol: '200000',
  ses: '0.10',
  resend: '0.40',
  mailgun: '0.80',
  postmark: '1.25',
  postal: '24',
  brevo: '120',
  mailchimp: '350',
  kit: '300',
}

/** The price fields, in the board's order (two columns). */
export const PRICE_FIELDS: { key: Exclude<CostField, 'vol'>; label: string }[] = [
  { key: 'ses', label: 'Amazon SES · $ per 1,000' },
  { key: 'resend', label: 'Resend · $ per 1,000' },
  { key: 'mailgun', label: 'Mailgun · $ per 1,000' },
  { key: 'postmark', label: 'Postmark · $ per 1,000' },
  { key: 'postal', label: 'Postal server · $ per month' },
  { key: 'brevo', label: 'Brevo plan · $ per month' },
  { key: 'mailchimp', label: 'Mailchimp plan · $ per month' },
  { key: 'kit', label: 'Kit plan · $ per month' },
]

export interface CostRow {
  name: string
  color: string
  note: string
  value: number
  cost: string
  /** Bar width relative to the most expensive row, at least 2%. */
  pct: string
}

/** Reads a number the way people type it: "200,000", "$0.10", " 24 ". */
export function num(v: string): number {
  const n = parseFloat(String(v).replace(/[^0-9.]/g, ''))
  return Number.isNaN(n) ? 0 : n
}

export function money(n: number): string {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function costRows(s: CostInputs): CostRow[] {
  const per = num(s.vol) / 1000
  const rows = [
    { name: 'Amazon SES', color: '#4a3aa7', value: per * num(s.ses), note: 'Pay as you go. Free tier may apply in your first year.' },
    { name: 'Resend', color: '#1baf7a', value: per * num(s.resend), note: 'Plans with included volume are usually cheaper than per-1,000 at this size.' },
    { name: 'Mailgun', color: '#eb6834', value: per * num(s.mailgun), note: 'Plans include volume; the per-1,000 rate is for overage.' },
    { name: 'Postmark', color: '#008300', value: per * num(s.postmark), note: 'Plan-based; transactional streams only unless you use a Broadcast stream.' },
    { name: 'Postal, self-hosted', color: '#e87ba4', value: num(s.postal), note: 'Flat server cost; a dedicated IP needs a 21-day warm-up, which ReputeCast runs for you.' },
    { name: 'Brevo', color: '#8b94a3', value: num(s.brevo), note: 'A SaaS plan for comparison; priced by email volume, hosting included.' },
    { name: 'Mailchimp', color: '#6b7280', value: num(s.mailchimp), note: 'A SaaS plan for comparison; priced by contacts, hosting included.' },
    { name: 'Kit', color: '#4b5563', value: num(s.kit), note: 'A SaaS plan for comparison; priced by subscribers, hosting included.' },
  ]
  const max = rows.reduce((m, r) => Math.max(m, r.value), 0)
  return rows.map((r) => ({
    ...r,
    cost: money(r.value),
    pct: `${max > 0 ? Math.max(2, Math.round((r.value / max) * 100)) : 0}%`,
  }))
}

export function volumeLabel(s: CostInputs): string {
  return num(s.vol).toLocaleString('en-US')
}
