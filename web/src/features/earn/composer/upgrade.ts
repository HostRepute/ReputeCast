/**
 * The Upgrade block (ComposerEarnBlocks, brief §3.7): the paywall card. A
 * headline, one line, the tier's price and trial, and a button to
 * {{ upgrade_url }}, the reader's own link to the upgrade page, which
 * credits the email. It shows to free subscribers only unless its
 * Visibility changes. Its settings live in the section's rc-upgrade
 * attribute; the compile step drops every rc-* attribute.
 */

export const ATTR_UPGRADE = 'rc-upgrade'
export const UPGRADE_MARKER = 'upgrade'

export interface UpgradeConfig {
  /** The tier whose price and trial the card reads. */
  tier: string
  headline: string
  line: string
  button: string
  /** The price line as last written ("$8/month or $80/year · 14 days free"). */
  price: string
}

const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
const teal = '#0F766E'

/** html.EscapeString, as the server writes it. */
function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&#34;')
}

/** Reads an rc-upgrade attribute; null when it isn't an Upgrade block. */
export function parseUpgrade(raw: unknown): UpgradeConfig | null {
  if (typeof raw !== 'string' || !raw.trim()) return null
  try {
    const c = JSON.parse(raw) as Partial<UpgradeConfig>
    return { tier: c.tier ?? '', headline: c.headline ?? '', line: c.line ?? '', button: c.button ?? '', price: c.price ?? '' }
  } catch {
    return null
  }
}

export function encodeUpgrade(c: UpgradeConfig): string {
  return JSON.stringify({ tier: c.tier, headline: c.headline, line: c.line, button: c.button, price: c.price })
}

/** The rule a new Upgrade block starts with: free subscribers only. */
export function upgradeRule(id = 'upgrade'): string {
  return JSON.stringify({ id, show: 'only', match: 'all', conditions: [{ kind: 'paid', op: 'free' }] })
}

/**
 * The card's column: what the settings draw. The card is one text block
 * with inline styles (like the demo's trial card), so it looks the same in
 * the canvas and in every mail app.
 */
export function upgradeColumn(c: UpgradeConfig): string {
  let card = `<div style="border:1px solid #E3E6EB;border-radius:8px;padding:22px 24px;background-color:#F9FAFB;text-align:center;">`
  card += `<div style="font-size:19px;font-weight:700;line-height:1.3;color:#0B1220;padding-bottom:8px;">${esc(c.headline)}</div>`
  if (c.line.trim()) card += `<div style="font-size:14px;line-height:1.55;color:#374151;padding-bottom:8px;">${esc(c.line)}</div>`
  if (c.price.trim()) card += `<div style="font-size:13px;font-weight:600;line-height:1.4;color:#0B1220;padding-bottom:14px;">${esc(c.price)}</div>`
  card +=
    `<a href="{{ upgrade_url }}" style="display:inline-block;background-color:${teal};color:#ffffff;font-family:${esc(plex)};font-size:14px;font-weight:600;` +
    `line-height:1;padding:11px 18px;border-radius:6px;text-decoration:none;">${esc(c.button)}</a></div>`
  return `<mj-column><mj-text align="center" font-family="${esc(plex)}" padding="0px">${card}</mj-text></mj-column>`
}

/** The whole block, as the Blocks panel inserts it. */
export function upgradeSection(c: UpgradeConfig): string {
  return (
    `<!-- rc:${UPGRADE_MARKER} -->\n` +
    `<mj-section ${ATTR_UPGRADE}="${esc(encodeUpgrade(c))}" rc-name="Upgrade" rc-visible="${esc(upgradeRule())}" background-color="#ffffff" padding="16px 24px">\n` +
    upgradeColumn(c) +
    '\n</mj-section>'
  )
}
