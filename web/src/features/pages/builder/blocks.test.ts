import { t } from '@/i18n'
import { contentHtml, PALETTE, paletteItem, sectionHtml, type BlockContext, type PaletteId } from './blocks'
import { cleanHtml, prepareHtml } from './editor/prepare'
import { BGS, bgOf, COMMON_LIVE_ATTRS, isLiveTag, LIVE_ATTRS, LIVE_TAGS, SPACES, spaceOf } from './model'
import { blockCopy } from './wording'

/**
 * The palette's blocks against the server: internal/pages/html.go's
 * allow-list (what a saved page may contain), blocks.go's live blocks and
 * their attributes, the public pages' classes (static/pages.css), and the
 * markup internal/pages/templates.go writes for the same blocks.
 */

/**
 * A file of the repository, read with Node (outside web/, where Vite's
 * imports don't reach; Node's types stay out of the app's program).
 */
async function repoFile(path: string): Promise<string> {
  const fs = (await import(/* @vite-ignore */ `node:${'fs'}`)) as { readFileSync(p: URL, encoding: 'utf8'): string }
  // The test's own address in a variable: Vite would take new URL('…', import.meta.url) for an asset.
  const here = import.meta.url
  return fs.readFileSync(new URL('../../../../../' + path, here), 'utf8')
}

const htmlGo = await repoFile('internal/pages/html.go')
const blocksGo = await repoFile('internal/pages/blocks.go')
const pagesCss = await repoFile('internal/publicpages/static/pages.css')

/** html.go's `elements`: the allowed elements and their own attributes. */
function goElements(src: string): Record<string, string[]> {
  const start = src.indexOf('var elements = map[string][]string{')
  const body = src.slice(start, src.indexOf('\n}\n', start))
  const out: Record<string, string[]> = {}
  for (const m of body.matchAll(/"([a-z0-9]+)":\s*(nil|\{([^}]*)\})/g)) out[m[1]!] = m[3] ? [...m[3].matchAll(/"([a-z-]+)"/g)].map((x) => x[1]!) : []
  return out
}

/** html.go's `common`: the attributes every element may carry. */
function goCommon(src: string): string[] {
  const m = /var common = \[\]string\{([^}]*)\}/.exec(src)
  return m ? [...m[1]!.matchAll(/"([a-z-]+)"/g)].map((x) => x[1]!) : []
}

/** blocks.go's `blockSpecs`: each live block's attributes. */
function goLiveAttrs(src: string): Record<string, string[]> {
  const tags = Object.fromEntries([...src.matchAll(/^\s*(El\w+)\s*=\s*"(rc-[a-z-]+)"/gm)].map((m) => [m[1]!, m[2]!]))
  const start = src.indexOf('var blockSpecs = map[string]blockSpec{')
  const parts = src.slice(start, src.indexOf('\n}\n', start)).split(/\n\t(El\w+):/)
  const out: Record<string, string[]> = {}
  for (let i = 1; i < parts.length; i += 2) out[tags[parts[i]!]!] = [...parts[i + 1]!.matchAll(/"([a-z]+)":/g)].map((m) => m[1]!).sort()
  return out
}

const ELEMENTS = goElements(htmlGo)
const COMMON = goCommon(htmlGo)
const CLASSES = new Set([...pagesCss.matchAll(/\.([a-z][a-z0-9-]*)/g)].map((m) => m[1]!))
const classRe = /^[a-z][a-z0-9-]{0,40}$/

const ctx: BlockContext = {
  copy: blockCopy(t),
  tz: 'America/New_York',
  date: '2026-11-02',
  formId: '0199f3a2-7c41-7a10-9a51-3c2f4d5e6f70',
  productId: '0199f3a2-7c41-7a10-9a51-3c2f4d5e6f71',
  pageName: 'DMARC, live: from p=none to p=reject',
  initials: 'AC',
}

function fragment(html: string): Element[] {
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  return Array.from(tpl.content.children)
}

/** What's wrong with a block's markup for the server and the theme. */
function problems(root: Element): string[] {
  const out: string[] = []
  for (const el of [root, ...Array.from(root.querySelectorAll('*'))]) {
    const tag = el.tagName.toLowerCase()
    if (isLiveTag(tag)) {
      const allowed = [...COMMON_LIVE_ATTRS, ...LIVE_ATTRS[tag]]
      for (const a of Array.from(el.attributes)) if (!allowed.includes(a.name)) out.push(`<${tag} ${a.name}>`)
      continue
    }
    const own = ELEMENTS[tag]
    if (!own) {
      out.push(`<${tag}>`)
      continue
    }
    for (const a of Array.from(el.attributes)) if (!COMMON.includes(a.name) && !own.includes(a.name)) out.push(`<${tag} ${a.name}>`)
    for (const c of Array.from(el.classList)) if (!classRe.test(c) || !CLASSES.has(c)) out.push(`.${c}`)
  }
  return out
}

/** Tags, classes and attribute names, without text or ids; repeated items once. */
function skeleton(el: Element): string {
  const attrs = Array.from(el.attributes)
    .map((a) => a.name)
    .filter((n) => n !== 'class' && n !== 'data-pb-id')
    .sort()
  const head = [el.tagName.toLowerCase(), ...Array.from(el.classList).map((c) => `.${c}`)].join('') + (attrs.length ? `[${attrs.join(',')}]` : '')
  const kids: string[] = []
  for (const c of Array.from(el.children)) {
    const s = skeleton(c)
    if (kids.at(-1) !== s) kids.push(s)
  }
  return kids.length ? `${head}(${kids.join(' ')})` : head
}

describe('the server’s rules, read from its source', () => {
  it('finds html.go’s allow-list and the theme’s classes', () => {
    expect(Object.keys(ELEMENTS)).toEqual(expect.arrayContaining(['section', 'div', 'img', 'a', 'details']))
    expect(ELEMENTS.img).toContain('alt')
    expect(COMMON).toEqual(expect.arrayContaining(['class', 'data-pb', 'data-pb-name', 'data-pb-bg', 'data-pb-space']))
    expect(CLASSES.has('sp-sec')).toBe(true)
  })

  it('keeps the attributes blocks.go keeps for each live block', () => {
    expect(goLiveAttrs(blocksGo)).toEqual(Object.fromEntries(LIVE_TAGS.map((tag) => [tag, [...LIVE_ATTRS[tag]].sort()])))
  })
})

describe('the palette', () => {
  it.each(PALETTE.map((p) => p.id))('adds %s as a section the server keeps as it is', (id) => {
    const [top, ...rest] = fragment(sectionHtml(id, ctx))
    expect(rest).toEqual([])
    expect(top?.tagName).toBe('SECTION')
    expect(top?.getAttribute('data-pb')).toBe(paletteItem(id).kind)
    expect(top?.getAttribute('data-pb-name')).toBeTruthy()
    // The background and spacing the Style panel shows are the ones the classes draw.
    const bg = top?.getAttribute('data-pb-bg') ?? ''
    const space = top?.getAttribute('data-pb-space') ?? ''
    expect(BGS).toContain(bg)
    expect(SPACES).toContain(space)
    const classes = Array.from(top!.classList)
    expect(bgOf({ attr: () => '', classes })).toBe(bg)
    expect(spaceOf({ attr: () => '', classes })).toBe(space)
    expect(problems(top!)).toEqual([])
  })

  it.each(PALETTE.filter((p) => p.level === 'content').map((p) => p.id))('adds %s into a column as one element', (id) => {
    const els = fragment(contentHtml(id, ctx))
    expect(els).toHaveLength(1)
    expect(problems(els[0]!)).toEqual([])
    expect(els[0]!.getAttribute('data-pb-name')).toBeTruthy()
  })

  it('starts a product’s button with the price placeholder the site fills in', () => {
    const [el] = fragment(contentHtml('product', ctx))
    expect(el?.getAttribute('button')).toBe('Buy for {price}')
    expect(el?.getAttribute('product')).toBe(ctx.productId)
  })

  it('starts countdowns and events in the workspace’s time zone, two weeks ahead', () => {
    const [cd] = fragment(contentHtml('countdown', ctx))
    expect(cd?.getAttribute('at')).toBe('2026-11-02T09:00')
    expect(cd?.getAttribute('tz')).toBe('America/New_York')
    const [ev] = fragment(contentHtml('event', ctx))
    expect([ev?.getAttribute('date'), ev?.getAttribute('start'), ev?.getAttribute('tz'), ev?.getAttribute('title')]).toEqual(['2026-11-02', '17:00', 'America/New_York', ctx.pageName])
  })

  // internal/pages/templates.go's sections for the same blocks.
  const templates: Record<string, string> = {
    features:
      `<section data-pb="features" data-pb-id="b-features" data-pb-name="Features list · What's inside" data-pb-bg="white" data-pb-space="normal" class="sp-sec w"><div class="sp-in"><div class="sec-head"><span class="eyebrow">What's inside</span><h2 class="sp-h2">Say what each part covers</h2></div>` +
      `<ol class="chapters"><li><span class="n">01</span><span class="t"><b>The first chapter</b><span class="d">One line on what it covers.</span></span></li><li><span class="n">02</span><span class="t"><b>The second chapter</b><span class="d">One line on what it covers.</span></span></li></ol></div></section>`,
    testimonials:
      `<section data-pb="testimonials" data-pb-id="b-testimonials" data-pb-name="Testimonials" data-pb-bg="white" data-pb-space="normal" class="sp-sec"><div class="sp-in"><div class="sec-head"><span class="eyebrow">From readers</span><h2 class="sp-h2">What people say</h2></div><div class="quotes">` +
      `<figure class="quote"><blockquote>“What a reader said about it, in their words.”</blockquote><figcaption><span class="qa" aria-hidden="true">AB</span><span class="qn"><b>A reader</b><span class="qr">Who they are</span></span></figcaption></figure></div></div></section>`,
    faq:
      `<section data-pb="faq" data-pb-id="b-faq" data-pb-name="FAQ" data-pb-bg="white" data-pb-space="normal" class="sp-sec"><div class="sp-in nar"><div class="sec-head"><span class="eyebrow">Questions</span><h2 class="sp-h2">Questions</h2></div><div class="faq">` +
      `<details open><summary>Is it really free?</summary><p>Yes.</p></details><details><summary>What's in it?</summary><p>Say what the file covers.</p></details></div></div></section>`,
    bio:
      `<section data-pb="bio" data-pb-id="b-bio" data-pb-name="Author bio" data-pb-bg="lavender" data-pb-space="normal" class="sp-sec lav"><div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">AC</span><div class="tx"><span class="eyebrow">Who writes it</span>` +
      `<h2 class="sp-h2">Your name</h2><span class="role">Editor of Acme</span><p>A few lines about the person behind the newsletter.</p></div></div></div></section>`,
    signup:
      `<section data-pb="signup" data-pb-id="b-signup" data-pb-name="Signup form · subscribe band" data-pb-bg="lavender" data-pb-space="compact" class="sp-sec cmp lav"><div class="sp-in nar">` +
      `<rc-form data-pb-id="subscribe" data-pb-name="Signup form" form="f1" heading="Get Acme by email" hint="One email a week. Unsubscribe any time."></rc-form></div></section>`,
  }

  it.each(Object.keys(templates))('writes %s with the same markup as the page templates', (id) => {
    const [mine] = fragment(sectionHtml(id as PaletteId, ctx))
    const [theirs] = fragment(templates[id]!)
    expect(skeleton(mine!)).toBe(skeleton(theirs!))
  })
})

describe('preparing a page for the editor', () => {
  const page = PALETTE.map((p) => sectionHtml(p.id, ctx)).join('\n')
  const canon = (html: string) => {
    const tpl = document.createElement('template')
    tpl.innerHTML = html
    return tpl.innerHTML.trim()
  }
  const flags = (el: Element | null | undefined) =>
    Object.fromEntries(
      Array.from(el?.attributes ?? [])
        .filter((a) => a.name.startsWith('data-gjs-'))
        .map((a) => [a.name.slice(9), a.value]),
    )
  const prepared = (html: string, sel: string) => fragment(prepareHtml(html))[0]?.parentNode?.querySelector(sel)

  it('gives the HTML back as it was once the editor’s marks are cleaned', () => {
    expect(cleanHtml(prepareHtml(page))).toBe(canon(page))
  })

  it('lets named parts be selected, removed and copied, and only containers take drops', () => {
    const hero = sectionHtml('hero', ctx)
    expect(flags(prepared(hero, 'section'))).toEqual({})
    expect(flags(prepared(hero, '.hero-copy'))).toEqual({ draggable: 'false', droppable: 'true', selectable: 'true', hoverable: 'true', removable: 'true', copyable: 'true' })
    expect(flags(prepared(hero, 'img'))).toMatchObject({ selectable: 'true', removable: 'true', droppable: 'false' })
    expect(flags(prepared(hero, 'h1'))).toEqual({ draggable: 'false', droppable: 'false' })
  })

  it('selects the block instead of its structure', () => {
    const hero = sectionHtml('hero', ctx)
    expect(flags(prepared(hero, '.hero-grid'))).toMatchObject({ selectable: 'false', removable: 'false', droppable: 'false' })
    expect(flags(prepared(hero, '.sp-in'))).toMatchObject({ selectable: 'false', droppable: 'true' })
    expect(flags(prepared(sectionHtml('spacer', ctx), '.pg-space'))).toMatchObject({ selectable: 'false', hoverable: 'false' })
  })

  it('selects a column to add blocks into it, but it goes with its block', () => {
    expect(flags(prepared(sectionHtml('columns2', ctx), '.pg-col'))).toMatchObject({ selectable: 'true', removable: 'false', copyable: 'false', droppable: 'true' })
  })

  it('leaves live blocks to their component type', () => {
    expect(flags(prepared(sectionHtml('signup', ctx), 'rc-form'))).toEqual({})
  })

  it('annotates a fragment for a column from its first element', () => {
    expect(flags(fragment(prepareHtml(contentHtml('text', ctx)))[0])).toMatchObject({ selectable: 'true', removable: 'true' })
  })

  it('drops what the editor adds from the saved HTML', () => {
    expect(cleanHtml('<div class="gjs-selected prose" data-gjs-type="text" contenteditable="true" draggable="true"><p class="gjs-hovered">Hi</p></div>')).toBe('<div class="prose"><p>Hi</p></div>')
  })
})
