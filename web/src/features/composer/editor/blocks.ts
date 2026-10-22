/**
 * The Blocks panel: MJML building blocks (GrapesJS Block Manager entries)
 * and ReputeCast's own blocks. Content is MJML; sections drop into the body,
 * content blocks into columns. ReputeCast blocks start with an
 * <!-- rc:… --> marker so they stay recognisable (layer names, checks).
 */
import { newPoll, pollSection } from '@/features/polls/config'
import { growBlockDefs } from '@/features/grow/composer/blocks'
import { upgradeSection } from '@/features/earn/composer/upgrade'
import { sponsorSection } from '@/features/earn/sponsors/composer/sponsor'
import { t, type MessageDescriptor } from '@/i18n'
import { mb } from '../blocks.messages'

export type BlockGroup = 'layout' | 'content' | 'reputecast' | 'grow' | 'earn'

export interface BlockDef {
  id: string
  label: string
  /** A badge on the tile ("New"). */
  badge?: string
  group: BlockGroup
  /** Icon from icons.tsx (24-unit grid). */
  icon: string
  /** Where it can be dropped: sections go in the body, content in a column. */
  kind: 'section' | 'content'
  content: string
  keywords?: string
}

const plex = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
const teal = '#0F766E'

/** Sample text in the app language, escaped for the MJML around it. */
function text(d: MessageDescriptor, values?: Record<string, string>): string {
  return t(d, values).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/** The same, for an attribute value. */
function attr(d: MessageDescriptor): string {
  return text(d).replace(/"/g, '&quot;')
}

/** The sample text block, with a first-name token that falls back to "there" in the app language. */
function textBlock(): string {
  const token = `{{ first_name | default: "${t(mb.there).replace(/"/g, '')}" }}`
  return `<mj-text font-size="15px" line-height="1.55" color="#1F2937">${text(mb.sampleText, { token })}</mj-text>`
}

function column(inner: string, width?: string) {
  return `<mj-column${width ? ` width="${width}"` : ''}>${inner}</mj-column>`
}

function section(inner: string, padding = '16px 24px') {
  return `<mj-section background-color="#ffffff" padding="${padding}">${inner}</mj-section>`
}

function button(label: string, href = 'https://example.com', align = 'left') {
  // `label` comes escaped (text()).
  return `<mj-button href="${href}" align="${align}" background-color="${teal}" color="#ffffff" font-family="${plex}" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">${label}</mj-button>`
}

/** The footer block: postal address and the unsubscribe and preferences links. */
export function footerBlock(address: string): string {
  const addr = (address || t(mb.address)).replace(/&/g, '&amp;').replace(/</g, '&lt;')
  return `<!-- rc:footer --><mj-section background-color="#F3F4F6" padding="18px 24px"><mj-column><mj-text align="center" font-family="${plex}" font-size="12px" line-height="1.6" color="#5B6472" padding="0px">${addr}<br><a href="{{ unsubscribe_url }}" style="color:${teal};text-decoration:none">${text(mb.unsubscribe)}</a> · <a href="{{ preferences_url }}" style="color:${teal};text-decoration:none">${text(mb.preferences)}</a> · <a href="{{ view_in_browser_url }}" style="color:${teal};text-decoration:none">${text(mb.viewInBrowser)}</a></mj-text></mj-column></mj-section>`
}

export function blockDefs(address: string): BlockDef[] {
  const sample = textBlock()
  return [
    { id: 'mj-1-column', label: t(mb.oneColumn), group: 'layout', icon: 'col1', kind: 'section', content: section(column(sample)), keywords: 'section one' },
    {
      id: 'mj-2-columns',
      label: t(mb.twoColumns),
      group: 'layout',
      icon: 'col2',
      kind: 'section',
      content: section(column(sample, '50%') + column(sample, '50%')),
      keywords: 'section two',
    },
    {
      id: 'mj-3-columns',
      label: t(mb.threeColumns),
      group: 'layout',
      icon: 'col3',
      kind: 'section',
      content: section(column(sample, '33.33%') + column(sample, '33.33%') + column(sample, '33.33%')),
      keywords: 'section three',
    },
    {
      id: 'mj-wrapper',
      label: t(mb.wrapper),
      group: 'layout',
      icon: 'wrapper',
      kind: 'section',
      content: `<mj-wrapper background-color="#F3F4F6" padding="16px 16px">${section(column(sample))}</mj-wrapper>`,
      keywords: 'background group sections',
    },
    {
      id: 'mj-hero',
      label: t(mb.hero),
      group: 'layout',
      icon: 'hero',
      kind: 'section',
      content: `<mj-hero mode="fixed-height" height="280px" background-width="600px" background-height="280px" background-url="/media/starters/banner.png" background-color="#0B1220" padding="60px 24px"><mj-text align="center" font-size="28px" font-weight="700" color="#ffffff" padding="0px 0px 12px"><h1 style="margin:0;font-size:28px;font-weight:700">${text(mb.heroHeadline)}</h1></mj-text>${button(text(mb.findOutMore), 'https://example.com', 'center')}</mj-hero>`,
      keywords: 'banner background image',
    },
    {
      id: 'mj-group',
      label: t(mb.group),
      group: 'layout',
      icon: 'group',
      kind: 'section',
      content: section(`<mj-group>${column(sample, '50%')}${column(sample, '50%')}</mj-group>`),
      keywords: 'columns side by side mobile',
    },
    { id: 'mj-text', label: t(mb.text), group: 'content', icon: 'text', kind: 'content', content: sample, keywords: 'paragraph copy' },
    {
      id: 'mj-image',
      label: t(mb.image),
      group: 'content',
      icon: 'image',
      kind: 'content',
      content: `<mj-image src="/media/starters/card.png" alt="" width="268px" padding="0px 0px 10px"></mj-image>`,
      keywords: 'picture photo',
    },
    { id: 'mj-button', label: t(mb.button), group: 'content', icon: 'button', kind: 'content', content: button(text(mb.readMore)), keywords: 'cta link' },
    {
      id: 'mj-divider',
      label: t(mb.divider),
      group: 'content',
      icon: 'divider',
      kind: 'content',
      content: `<mj-divider border-width="1px" border-color="#E3E6EB" padding="8px 0px"></mj-divider>`,
      keywords: 'line separator hr',
    },
    { id: 'mj-spacer', label: t(mb.spacer), group: 'content', icon: 'spacer', kind: 'content', content: `<mj-spacer height="24px"></mj-spacer>`, keywords: 'gap space' },
    {
      id: 'mj-social',
      label: t(mb.social),
      group: 'content',
      icon: 'social',
      kind: 'content',
      content: `<mj-social font-size="12px" icon-size="20px" mode="horizontal" align="left" padding="0px"><mj-social-element name="x" href="https://x.com">X</mj-social-element><mj-social-element name="linkedin" href="https://linkedin.com">LinkedIn</mj-social-element><mj-social-element name="github" href="https://github.com">GitHub</mj-social-element></mj-social>`,
      keywords: 'twitter linkedin github icons',
    },
    {
      id: 'mj-navbar',
      label: t(mb.navbar),
      group: 'content',
      icon: 'navbar',
      kind: 'content',
      content: `<mj-navbar align="left"><mj-navbar-link href="https://example.com/blog" color="#5B6472" font-family="${plex}" font-size="12px" padding="0px 12px 0px 0px">${text(mb.blog)}</mj-navbar-link><mj-navbar-link href="https://example.com/docs" color="#5B6472" font-family="${plex}" font-size="12px" padding="0px 12px 0px 0px">${text(mb.docs)}</mj-navbar-link></mj-navbar>`,
      keywords: 'menu links navigation',
    },
    {
      id: 'mj-raw',
      label: t(mb.rawHtml),
      group: 'content',
      icon: 'raw',
      kind: 'content',
      content: `<mj-raw><p style="margin:0;font-family:Georgia,serif;font-size:14px;color:#374151">${text(mb.rawSample)}</p></mj-raw>`,
      keywords: 'code html custom',
    },
    {
      id: 'mj-table',
      label: t(mb.table),
      group: 'content',
      icon: 'table',
      kind: 'content',
      content: `<mj-table font-size="13px" color="#374151" padding="0px"><tr style="border-bottom:1px solid #E3E6EB;text-align:left"><th style="padding:6px 0">${text(mb.tablePlan)}</th><th style="padding:6px 0">${text(mb.tablePrice)}</th></tr><tr><td style="padding:6px 0">${text(mb.tableStarter)}</td><td style="padding:6px 0">${text(mb.tableStarterPrice)}</td></tr><tr><td style="padding:6px 0">${text(mb.tableTeam)}</td><td style="padding:6px 0">${text(mb.tableTeamPrice)}</td></tr></mj-table>`,
      keywords: 'rows columns pricing',
    },
    {
      id: 'rc-header',
      label: t(mb.headerLogo),
      group: 'reputecast',
      icon: 'rcHeader',
      kind: 'section',
      content: `<!-- rc:header-logo --><mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3"><mj-column width="40%" vertical-align="middle"><mj-image src="/media/starters/logo.png" alt="${attr(mb.logoAlt)}" width="120px" align="left" padding="0px"></mj-image></mj-column><mj-column width="60%" vertical-align="middle"><mj-text align="right" font-family="${plex}" font-size="12px" color="#5B6472" padding="0px"><a href="https://example.com/blog" style="color:#5B6472;text-decoration:none">${text(mb.blog)}</a> · <a href="https://example.com/docs" style="color:#5B6472;text-decoration:none">${text(mb.docs)}</a></mj-text></mj-column></mj-section>`,
      keywords: 'logo navigation top',
    },
    {
      id: 'rc-article',
      label: t(mb.articleCard),
      group: 'reputecast',
      icon: 'rcArticle',
      kind: 'section',
      content: `<!-- rc:article-card --><mj-section background-color="#ffffff" padding="16px 24px"><mj-column><mj-image src="/media/starters/hero.png" alt="${attr(mb.articleAlt)}" width="552px" padding="0px 0px 12px"></mj-image><mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px"><h2 style="margin:0;font-size:17px;font-weight:700">${text(mb.articleTitle)}</h2></mj-text><mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">${text(mb.articleLine)}</mj-text>${button(text(mb.readMore))}</mj-column></mj-section>`,
      keywords: 'post story blog',
    },
    {
      id: 'rc-product',
      label: t(mb.productCard),
      group: 'reputecast',
      icon: 'rcProduct',
      kind: 'section',
      content: `<!-- rc:product-card --><mj-section background-color="#ffffff" padding="16px 24px"><mj-column width="40%" padding="0px 12px 0px 0px"><mj-image src="/media/starters/product.png" alt="${attr(mb.productAlt)}" width="168px" padding="0px"></mj-image></mj-column><mj-column width="60%" vertical-align="middle"><mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 4px">${text(mb.productName)}</mj-text><mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">${text(mb.productLine)}</mj-text>${button(text(mb.viewProduct))}</mj-column></mj-section>`,
      keywords: 'shop price ecommerce',
    },
    { id: 'rc-footer', label: t(mb.footer), group: 'reputecast', icon: 'rcFooter', kind: 'section', content: footerBlock(address), keywords: 'address unsubscribe compliance' },
    // The Poll block (design expansion, X11): each answer a plain link.
    { id: 'rc-poll', label: t(mb.poll), badge: t(mb.newBadge), group: 'reputecast', icon: 'rcPoll', kind: 'section', content: pollSection(newPoll()), keywords: 'poll question vote nps rating survey feedback' },
    // Grow's blocks (design expansion, X15): Referral and Recommendations.
    ...growBlockDefs(),
    // Earn (design expansion, X17): the Sponsor block fills itself from the
    // issue's booking (its settings fetch the creative).
    {
      id: 'rc-sponsor',
      label: t(mb.sponsor),
      group: 'earn',
      icon: 'rcSponsor',
      kind: 'section',
      content: sponsorSection(t(mb.sponsorFill)),
      keywords: 'sponsor sponsored ad advertising booking creative earn',
    },
    // Earn (design expansion, X16): the Upgrade block, the paywall card; its
    // settings fill in the tier's price and trial.
    {
      id: 'rc-upgrade',
      label: t(mb.upgrade),
      group: 'earn',
      icon: 'rcUpgrade',
      kind: 'section',
      content: upgradeSection({ tier: '', headline: t(mb.upgradeHeadline), line: t(mb.upgradeLine), button: t(mb.upgradeButton), price: '', offer: '' }),
      keywords: 'paywall paid membership subscribe price trial earn',
    },
  ]
}

export function isSectionTag(tag: string) {
  return tag === 'mj-section' || tag === 'mj-wrapper' || tag === 'mj-hero' || tag === 'mj-raw'
}
