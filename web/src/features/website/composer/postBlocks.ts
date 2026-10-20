import { t } from '@/i18n'
import { blockDefs, type BlockDef } from '@/features/composer/editor/blocks'
import { mc } from './messages'

/** The "Free preview ends here" block: an empty raw block the site cuts the preview at. */
export const PREVIEW_MARKER = '<mj-raw rc-preview="end"><!-- rc:preview-end --></mj-raw>'

const LAYOUT = ['mj-1-column', 'mj-2-columns', 'mj-3-columns']
const CONTENT = ['mj-text', 'mj-image', 'mj-button', 'mj-divider', 'mj-spacer', 'mj-raw', 'mj-table']

/**
 * The Blocks panel in post mode (ComposerPublishToWeb): layout and content
 * blocks, then Article card and Free preview ends here (the panel adds
 * Saved sections).
 * Email-only blocks (the footer and unsubscribe link, the view-online line,
 * referral) aren't offered.
 */
export function postBlocks(address: string): BlockDef[] {
  const all = blockDefs(address)
  const pick = (ids: string[]) => ids.map((id) => all.find((d) => d.id === id)).filter((d): d is BlockDef => !!d)
  const article = all.find((d) => d.id === 'rc-article')
  return [
    ...pick(LAYOUT),
    ...pick(CONTENT).map((d) => (d.id === 'mj-raw' ? { ...d, label: 'Code' } : d)),
    ...(article ? [article] : []),
    { id: 'rc-preview-end', label: t(mc.blockFreePreview), group: 'reputecast', icon: 'divider', kind: 'content', content: PREVIEW_MARKER, keywords: 'paywall marker preview members paid' },
  ]
}
