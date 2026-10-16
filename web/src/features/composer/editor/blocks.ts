/**
 * The Blocks panel: MJML building blocks (GrapesJS Block Manager entries)
 * and ReputeCast's own blocks. Content is MJML; sections drop into the body,
 * content blocks into columns. ReputeCast blocks start with an
 * <!-- rc:… --> marker so they stay recognisable (layer names, checks).
 */
export type BlockGroup = 'layout' | 'content' | 'reputecast'

export interface BlockDef {
  id: string
  label: string
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

const textBlock = `<mj-text font-size="15px" line-height="1.55" color="#1F2937">Write something people will want to read. Personalize it with {{ first_name | default: "there" }}.</mj-text>`

function column(inner: string, width?: string) {
  return `<mj-column${width ? ` width="${width}"` : ''}>${inner}</mj-column>`
}

function section(inner: string, padding = '16px 24px') {
  return `<mj-section background-color="#ffffff" padding="${padding}">${inner}</mj-section>`
}

function button(label: string, href = 'https://example.com', align = 'left') {
  return `<mj-button href="${href}" align="${align}" background-color="${teal}" color="#ffffff" font-family="${plex}" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px">${label}</mj-button>`
}

/** The footer block: postal address and the unsubscribe and preferences links. */
export function footerBlock(address: string): string {
  const addr = (address || 'Your company · Street and number · City, postcode').replace(/&/g, '&amp;').replace(/</g, '&lt;')
  return `<!-- rc:footer --><mj-section background-color="#F3F4F6" padding="18px 24px"><mj-column><mj-text align="center" font-family="${plex}" font-size="12px" line-height="1.6" color="#5B6472" padding="0px">${addr}<br><a href="{{ unsubscribe_url }}" style="color:${teal};text-decoration:none">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:${teal};text-decoration:none">Email preferences</a> · <a href="{{ view_in_browser_url }}" style="color:${teal};text-decoration:none">View in browser</a></mj-text></mj-column></mj-section>`
}

export function blockDefs(address: string): BlockDef[] {
  return [
    { id: 'mj-1-column', label: '1 column', group: 'layout', icon: 'col1', kind: 'section', content: section(column(textBlock)), keywords: 'section one' },
    {
      id: 'mj-2-columns',
      label: '2 columns',
      group: 'layout',
      icon: 'col2',
      kind: 'section',
      content: section(column(textBlock, '50%') + column(textBlock, '50%')),
      keywords: 'section two',
    },
    {
      id: 'mj-3-columns',
      label: '3 columns',
      group: 'layout',
      icon: 'col3',
      kind: 'section',
      content: section(column(textBlock, '33.33%') + column(textBlock, '33.33%') + column(textBlock, '33.33%')),
      keywords: 'section three',
    },
    {
      id: 'mj-wrapper',
      label: 'Wrapper',
      group: 'layout',
      icon: 'wrapper',
      kind: 'section',
      content: `<mj-wrapper background-color="#F3F4F6" padding="16px 16px">${section(column(textBlock))}</mj-wrapper>`,
      keywords: 'background group sections',
    },
    {
      id: 'mj-hero',
      label: 'Hero',
      group: 'layout',
      icon: 'hero',
      kind: 'section',
      content: `<mj-hero mode="fixed-height" height="280px" background-width="600px" background-height="280px" background-url="/media/starters/banner.png" background-color="#0B1220" padding="60px 24px"><mj-text align="center" font-size="28px" font-weight="700" color="#ffffff" padding="0px 0px 12px"><h1 style="margin:0;font-size:28px;font-weight:700">A headline over an image</h1></mj-text>${button('Find out more', 'https://example.com', 'center')}</mj-hero>`,
      keywords: 'banner background image',
    },
    {
      id: 'mj-group',
      label: 'Group',
      group: 'layout',
      icon: 'group',
      kind: 'section',
      content: section(`<mj-group>${column(textBlock, '50%')}${column(textBlock, '50%')}</mj-group>`),
      keywords: 'columns side by side mobile',
    },
    { id: 'mj-text', label: 'Text', group: 'content', icon: 'text', kind: 'content', content: textBlock, keywords: 'paragraph copy' },
    {
      id: 'mj-image',
      label: 'Image',
      group: 'content',
      icon: 'image',
      kind: 'content',
      content: `<mj-image src="/media/starters/card.png" alt="" width="268px" padding="0px 0px 10px"></mj-image>`,
      keywords: 'picture photo',
    },
    { id: 'mj-button', label: 'Button', group: 'content', icon: 'button', kind: 'content', content: button('Read more'), keywords: 'cta link' },
    {
      id: 'mj-divider',
      label: 'Divider',
      group: 'content',
      icon: 'divider',
      kind: 'content',
      content: `<mj-divider border-width="1px" border-color="#E3E6EB" padding="8px 0px"></mj-divider>`,
      keywords: 'line separator hr',
    },
    { id: 'mj-spacer', label: 'Spacer', group: 'content', icon: 'spacer', kind: 'content', content: `<mj-spacer height="24px"></mj-spacer>`, keywords: 'gap space' },
    {
      id: 'mj-social',
      label: 'Social',
      group: 'content',
      icon: 'social',
      kind: 'content',
      content: `<mj-social font-size="12px" icon-size="20px" mode="horizontal" align="left" padding="0px"><mj-social-element name="x" href="https://x.com">X</mj-social-element><mj-social-element name="linkedin" href="https://linkedin.com">LinkedIn</mj-social-element><mj-social-element name="github" href="https://github.com">GitHub</mj-social-element></mj-social>`,
      keywords: 'twitter linkedin github icons',
    },
    {
      id: 'mj-navbar',
      label: 'Navbar',
      group: 'content',
      icon: 'navbar',
      kind: 'content',
      content: `<mj-navbar align="left"><mj-navbar-link href="https://example.com/blog" color="#5B6472" font-family="${plex}" font-size="12px" padding="0px 12px 0px 0px">Blog</mj-navbar-link><mj-navbar-link href="https://example.com/docs" color="#5B6472" font-family="${plex}" font-size="12px" padding="0px 12px 0px 0px">Docs</mj-navbar-link></mj-navbar>`,
      keywords: 'menu links navigation',
    },
    {
      id: 'mj-raw',
      label: 'Raw HTML',
      group: 'content',
      icon: 'raw',
      kind: 'content',
      content: `<mj-raw><p style="margin:0;font-family:Georgia,serif;font-size:14px;color:#374151">Your HTML here.</p></mj-raw>`,
      keywords: 'code html custom',
    },
    {
      id: 'mj-table',
      label: 'Table',
      group: 'content',
      icon: 'table',
      kind: 'content',
      content: `<mj-table font-size="13px" color="#374151" padding="0px"><tr style="border-bottom:1px solid #E3E6EB;text-align:left"><th style="padding:6px 0">Plan</th><th style="padding:6px 0">Price</th></tr><tr><td style="padding:6px 0">Starter</td><td style="padding:6px 0">$9/mo</td></tr><tr><td style="padding:6px 0">Team</td><td style="padding:6px 0">$29/mo</td></tr></mj-table>`,
      keywords: 'rows columns pricing',
    },
    {
      id: 'rc-header',
      label: 'Header + logo',
      group: 'reputecast',
      icon: 'rcHeader',
      kind: 'section',
      content: `<!-- rc:header-logo --><mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3"><mj-column width="40%" vertical-align="middle"><mj-image src="/media/starters/logo.png" alt="Your logo" width="120px" align="left" padding="0px"></mj-image></mj-column><mj-column width="60%" vertical-align="middle"><mj-text align="right" font-family="${plex}" font-size="12px" color="#5B6472" padding="0px"><a href="https://example.com/blog" style="color:#5B6472;text-decoration:none">Blog</a> · <a href="https://example.com/docs" style="color:#5B6472;text-decoration:none">Docs</a></mj-text></mj-column></mj-section>`,
      keywords: 'logo navigation top',
    },
    {
      id: 'rc-article',
      label: 'Article card',
      group: 'reputecast',
      icon: 'rcArticle',
      kind: 'section',
      content: `<!-- rc:article-card --><mj-section background-color="#ffffff" padding="16px 24px"><mj-column><mj-image src="/media/starters/hero.png" alt="Describe the article's image" width="552px" padding="0px 0px 12px"></mj-image><mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 6px"><h2 style="margin:0;font-size:17px;font-weight:700">Article title</h2></mj-text><mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">One or two sentences that make people want to read the rest.</mj-text>${button('Read more')}</mj-column></mj-section>`,
      keywords: 'post story blog',
    },
    {
      id: 'rc-product',
      label: 'Product card',
      group: 'reputecast',
      icon: 'rcProduct',
      kind: 'section',
      content: `<!-- rc:product-card --><mj-section background-color="#ffffff" padding="16px 24px"><mj-column width="40%" padding="0px 12px 0px 0px"><mj-image src="/media/starters/product.png" alt="Describe the product" width="168px" padding="0px"></mj-image></mj-column><mj-column width="60%" vertical-align="middle"><mj-text font-size="17px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 4px">Product name</mj-text><mj-text font-size="14px" line-height="1.5" color="#374151" padding="0px 0px 10px">$29 · one line about what it does</mj-text>${button('View product')}</mj-column></mj-section>`,
      keywords: 'shop price ecommerce',
    },
    { id: 'rc-footer', label: 'Footer + unsubscribe', group: 'reputecast', icon: 'rcFooter', kind: 'section', content: footerBlock(address), keywords: 'address unsubscribe compliance' },
  ]
}

export function isSectionTag(tag: string) {
  return tag === 'mj-section' || tag === 'mj-wrapper' || tag === 'mj-hero' || tag === 'mj-raw'
}
