import { t } from '@/i18n'
import { blockDefs, type BlockDef } from '../editor/blocks'
import { encodeTables } from '../editor/components'
import { mb } from './messages'

const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/**
 * The transactional footer: company and address. Essential categories get
 * no unsubscribe link (recipients can't turn account email off); Optional
 * ones get the category's unsubscribe and preference links, as X02's
 * starters do.
 */
export function transactionalFooter(address: string, product: string, kind: 'essential' | 'optional'): string {
  const addr = esc(address.trim() || t(mb.footerAddress))
  // The product's name: Settings › Transactional, or the workspace's name.
  const name = product.trim()
  const line =
    kind === 'optional'
      ? `${esc(t(mb.footerOptional, { product: name }))} <a href="{{ unsubscribe_url }}" style="color:#5B6472">${esc(t(mb.footerTurnOff))}</a> · <a href="{{ preferences_url }}" style="color:#5B6472">${esc(t(mb.footerManage))}</a>`
      : esc(t(mb.footerEssential, { product: name }))
  return `<!-- rc:transactional-footer --><mj-section background-color="#F3F4F6" padding="16px 28px"><mj-column><mj-text font-family="${plex}" font-size="12px" line-height="1.6" color="#5B6472" padding="0px">${addr}<br>${line}</mj-text></mj-column></mj-section>`
}

/** Receipt lines: a row per item of `items`, then the total. */
function receiptLines(): string {
  const row = (a: string, b: string, bold = false) =>
    `<tr><td style="padding:6px 0;${bold ? 'font-weight:700;' : ''}">${a}</td><td style="padding:6px 0;text-align:right;${bold ? 'font-weight:700;' : ''}">${b}</td></tr>`
  const head = `<tr style="border-bottom:1px solid #E3E6EB;text-align:left"><th style="padding:6px 0">${esc(t(mb.receiptItem))}</th><th style="padding:6px 0;text-align:right">${esc(t(mb.receiptAmount))}</th></tr>`
  const rows = `{% for item in items %}${row('{{ item.name }}', '{{ item.amount }}')}{% endfor %}`
  const total = row(esc(t(mb.receiptTotal)), '{{ total }}', true)
  return encodeTables(
    `<!-- rc:receipt-lines --><mj-section background-color="#ffffff" padding="8px 28px"><mj-column><mj-table font-family="${plex}" font-size="14px" color="#1F2937" padding="0px">${head}${rows}${total}</mj-table></mj-column></mj-section>`,
  )
}

const LAYOUT = ['mj-1-column', 'mj-2-columns', 'mj-wrapper']
const CONTENT = ['mj-text', 'mj-image', 'mj-button', 'mj-divider', 'mj-spacer', 'mj-table']

/**
 * The Blocks panel of a transactional template (TransactionalTemplate board):
 * the composer's layout and content blocks, then Header + logo, Receipt
 * lines and the Transactional footer in place of the newsletter's cards.
 */
export function transactionalBlocks(address: string, product: string, kind: 'essential' | 'optional'): BlockDef[] {
  const all = blockDefs(address)
  const pick = (ids: string[]) => ids.map((id) => all.find((d) => d.id === id)).filter((d): d is BlockDef => !!d)
  const header = all.find((d) => d.id === 'rc-header')
  // The newsletter's sample text greets a contact; product mail starts plainer.
  const content = pick(CONTENT).map((d) => (d.id === 'mj-text' ? { ...d, content: `<mj-text font-size="15px" line-height="1.6" color="#1F2937">${esc(t(mb.textBlock))}</mj-text>` } : d))
  return [
    ...pick(LAYOUT),
    ...content,
    ...(header ? [header] : []),
    { id: 'tx-receipt', label: t(mb.receiptLines), group: 'reputecast', icon: 'rcReceipt', kind: 'section', content: receiptLines(), keywords: 'invoice order items total table' },
    { id: 'tx-footer', label: t(mb.transactionalFooter), group: 'reputecast', icon: 'rcFooter', kind: 'section', content: transactionalFooter(address, product, kind), keywords: 'address company footer legal' },
  ]
}
