import {
  bgOf,
  buttonDisplay,
  buttonTemplate,
  conflictBlocks,
  datePlus,
  diffPages,
  formatUnits,
  freeId,
  isVideoUrl,
  joinAt,
  joinList,
  layerRows,
  liveAttrs,
  liveHTML,
  parseList,
  parseUnits,
  sectionKind,
  soleLive,
  spaceOf,
  splitAt,
  timeLeft,
  withBg,
  withSpace,
  zoneAbbr,
  zoneCity,
  zonedInstant,
  type PbNode,
} from './model'

/** A DOM element behind the model's face, as the editor's components are. */
function node(el: Element): PbNode {
  return {
    key: el.getAttribute('data-pb-id') || el.getAttribute('data-pb-name') || el.tagName.toLowerCase(),
    tag: el.tagName.toLowerCase(),
    attr: (n) => el.getAttribute(n) ?? '',
    classes: Array.from(el.classList),
    get children() {
      return Array.from(el.children).map(node)
    },
  }
}

function blocks(html: string): PbNode[] {
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  return Array.from(tpl.content.children).map(node)
}

function section(kind: string, id: string, name: string, cls: string, inner: string): string {
  return `<section data-pb="${kind}" data-pb-id="${id}" data-pb-name="${name}" data-pb-bg="white" data-pb-space="normal" class="${cls}">${inner}</section>`
}

const chapter = (n: number) => `<li><span class="n">0${n}</span><span class="t"><b>Chapter ${n}</b><span class="d">One line.</span></span></li>`
const quote = `<figure class="quote"><blockquote>“Good.”</blockquote><figcaption><span class="qa" aria-hidden="true">AB</span><span class="qn"><b>A reader</b><span class="qr">Who</span></span></figcaption></figure>`
const question = (q: string) => `<details><summary>${q}</summary><p>An answer.</p></details>`

/** The Sales page template (internal/pages/templates.go), as The Deliverability Field Guide uses it. */
const SALES = [
  section(
    'hero',
    'b-hero',
    'Hero',
    'sp-hero',
    `<div class="sp-in"><div class="hero-grid"><div class="hero-copy" data-pb-name="Heading, text and checklist"><span class="eyebrow">Guide · PDF</span><h1>The Deliverability Field Guide</h1><p class="sp-lead">Lead.</p>` +
      `<ul class="ticks"><li>One</li><li>Two</li></ul><rc-product data-pb-id="buy" data-pb-name="Product" id="buy" product="p1" button="Buy for {price}" price="1" offer="1" layout="box"></rc-product></div>` +
      `<div class="hero-art"><span class="cov xl blank" role="img" aria-label="Cover"></span></div></div></div>`,
  ),
  section('features', 'b-features', 'Features list · What’s inside', 'sp-sec w', `<div class="sp-in"><div class="sec-head"><span class="eyebrow">What’s inside</span><h2 class="sp-h2">Parts</h2></div><ol class="chapters">${[1, 2, 3, 4].map(chapter).join('')}</ol></div>`),
  section('bio', 'b-bio', 'Author bio', 'sp-sec lav', `<div class="sp-in nar"><div class="bio"><span class="av7" aria-hidden="true">LN</span><div class="tx"><span class="eyebrow">About</span><h2 class="sp-h2">Linh</h2><span class="role">Editor</span><p>Why.</p></div></div></div>`),
  section('testimonials', 'b-testimonials', 'Testimonials', 'sp-sec', `<div class="sp-in"><div class="sec-head"><span class="eyebrow">From readers</span><h2 class="sp-h2">What people say</h2></div><div class="quotes">${quote.repeat(3)}</div></div>`),
  section('faq', 'b-faq', 'FAQ · Before you buy', 'sp-sec lav', `<div class="sp-in nar"><div class="sec-head"><span class="eyebrow">Questions</span><h2 class="sp-h2">Before you buy</h2></div><div class="faq">${question('What?')}${question('How?')}${question('Who?')}</div></div>`),
  section('product', 'b-product', 'Product · second Buy button', 'sp-sec lav top0', `<div class="sp-in nar"><rc-product data-pb-id="buy-2" data-pb-name="Product · second Buy button" product="p1" button="Buy for {price}" price="1" offer="1" layout="final"></rc-product></div>`),
].join('\n')

/** The Webinar template's second block. */
const WEBINAR_DETAILS = section(
  'section',
  'b-section',
  'Details, agenda and registration',
  'sp-sec cmp',
  `<div class="sp-in nar"><div class="lp-body"><rc-event data-pb-id="event" data-pb-name="Event details" date="2026-11-12" start="17:00" end="17:30" tz="UTC" local="1" where="online"></rc-event>` +
    `<div class="host" data-pb-name="Speaker"><span class="hav" aria-hidden="true">LN</span><span class="hn"><b>Linh</b><span class="hr">Editor</span></span></div>` +
    `<div data-pb-name="Agenda"><h2 class="agenda-h">Agenda</h2><ol class="agenda"><li><span class="m">5 min</span>First</li></ol></div>` +
    `<rc-form data-pb-id="register" data-pb-name="Signup form · registration" form="f1" heading="Save your seat"></rc-form>` +
    `<img data-pb="image" data-pb-name="Slide" class="pg-img" src="/media/a.png" alt=""></div></div>`,
)

describe('live blocks', () => {
  it('keeps the common attributes first, then the block’s own, and drops the rest', () => {
    expect(liveHTML('rc-product', { product: 'p1', button: 'Buy for {price}', 'data-pb-id': 'buy', 'data-pb-name': 'Product', id: ' ', price: '1', onclick: 'x()', style: 'color:red' })).toBe(
      '<rc-product data-pb-id="buy" data-pb-name="Product" product="p1" button="Buy for {price}" price="1"></rc-product>',
    )
  })

  it('keeps a block’s own empty attributes: a signup block without a form yet', () => {
    expect(liveAttrs('rc-form', { form: '', heading: 'Join', hint: undefined, bg: false })).toEqual([
      ['form', ''],
      ['heading', 'Join'],
    ])
  })

  it('escapes what goes into attributes', () => {
    expect(liveHTML('rc-form', { form: 'f1', heading: 'Say "hi" & <go>' })).toBe('<rc-form form="f1" heading="Say &quot;hi&quot; &amp; &lt;go&gt;"></rc-form>')
  })

  it('takes the event’s contact date field (the DMARC page’s dmarc_live_at)', () => {
    expect(liveHTML('rc-event', { field: 'dmarc_live_at', tz: 'America/New_York' })).toBe('<rc-event tz="America/New_York" field="dmarc_live_at"></rc-event>')
  })
})

describe('layers', () => {
  it('lists the Field Guide’s blocks as the board does: columns, named parts and counts', () => {
    const rows = layerRows(blocks(SALES)).map((r) => [r.kind, r.name, r.sub, r.tag])
    expect(rows).toEqual([
      ['hero', 'Hero', false, { kind: 'columns', n: 2 }],
      ['text', 'Heading, text and checklist', true, null],
      ['product', 'Product', true, null],
      ['features', 'Features list · What’s inside', false, { kind: 'count', n: 4 }],
      ['bio', 'Author bio', false, null],
      ['testimonials', 'Testimonials', false, { kind: 'count', n: 3 }],
      ['faq', 'FAQ · Before you buy', false, { kind: 'count', n: 3 }],
      // The second Buy button's band stands for its product: one row.
      ['product', 'Product · second Buy button', false, null],
    ])
  })

  it('names the parts of a webinar’s details block and says when an image has no alt text', () => {
    const rows = layerRows(blocks(WEBINAR_DETAILS))
    expect(rows.map((r) => [r.kind, r.name, r.sub])).toEqual([
      ['section', 'Details, agenda and registration', false],
      ['event', 'Event details', true],
      ['bio', 'Speaker', true],
      ['features', 'Agenda', true],
      ['signup', 'Signup form · registration', true],
      ['image', 'Slide', true],
    ])
    expect(rows.at(-1)?.alt).toBe(false)
  })

  it('finds the live block a band stands for, and none when the band has more', () => {
    const [band] = blocks(section('signup', 'b-signup', 'Signup form', 'sp-sec cmp lav', '<div class="sp-in nar"><rc-form form="f1"></rc-form></div>'))
    expect(soleLive(band!)?.tag).toBe('rc-form')
    const [hero] = blocks(SALES)
    expect(soleLive(hero!)).toBeUndefined()
  })

  it('reads a section’s kind from data-pb, or from a hero’s classes', () => {
    const [a, b, c] = blocks('<section data-pb="faq"></section><section data-pb="weird" class="lp-band"></section><section class="sp-sec"></section>')
    expect([sectionKind(a!), sectionKind(b!), sectionKind(c!)]).toEqual(['faq', 'hero', 'section'])
  })
})

describe('background and spacing as the theme’s classes', () => {
  const look = (classes: string[], attrs: Record<string, string> = {}) => ({ classes, attr: (n: string) => attrs[n] ?? '' })

  it('reads the background from data-pb-bg, else from the classes', () => {
    expect(bgOf(look(['sp-sec'], { 'data-pb-bg': 'indigo' }))).toBe('indigo')
    expect(bgOf(look(['sp-sec', 'lav']))).toBe('lavender')
    expect(bgOf(look(['sp-hero']))).toBe('lavender')
    expect(bgOf(look(['sp-sec', 'ink']))).toBe('ink')
    expect(bgOf(look(['sp-sec']))).toBe('white')
  })

  it('swaps one background class for another', () => {
    expect(withBg(['sp-sec', 'lav'], 'indigo')).toEqual(['sp-sec', 'ind'])
    expect(withBg(['sp-sec', 'lav', 'top0'], 'white')).toEqual(['sp-sec', 'top0'])
    expect(withBg(['lp-band'], 'ink')).toEqual(['lp-band', 'ink'])
  })

  it('adds no lavender class to a hero, whose own class draws it, and a white class instead', () => {
    expect(withBg(['sp-hero'], 'lavender')).toEqual(['sp-hero'])
    expect(withBg(['sp-sec'], 'lavender')).toEqual(['sp-sec', 'lav'])
    expect(withBg(['sp-hero'], 'white')).toEqual(['sp-hero', 'wht'])
    expect(withBg(['lp-band', 'wht'], 'ink')).toEqual(['lp-band', 'ink'])
    expect(withBg(['sp-sec'], 'white')).toEqual(['sp-sec'])
    expect(bgOf(look(['sp-hero', 'wht']))).toBe('white')
  })

  it('swaps the spacing class, keeping top0 last', () => {
    expect(spaceOf(look(['sp-sec', 'cmp']))).toBe('compact')
    expect(spaceOf(look(['sp-sec'], { 'data-pb-space': 'roomy' }))).toBe('roomy')
    expect(withSpace(['sp-sec', 'lav', 'top0'], 'compact')).toEqual(['sp-sec', 'lav', 'cmp', 'top0'])
    expect(withSpace(['sp-sec', 'cmp'], 'normal')).toEqual(['sp-sec'])
    expect(withSpace(['sp-sec', 'cmp'], 'roomy')).toEqual(['sp-sec', 'roomy'])
  })
})

describe('ids', () => {
  it('numbers ids the page already uses, as internal/pages does', () => {
    expect(freeId('b-faq', new Set())).toBe('b-faq')
    expect(freeId('b-faq', new Set(['b-faq']))).toBe('b-faq-2')
    expect(freeId('b-faq', new Set(['b-faq', 'b-faq-2']))).toBe('b-faq-3')
  })

  it('makes any name a valid id', () => {
    expect(freeId('Buy Now!', new Set())).toBe('buy-now')
    expect(freeId('2nd', new Set())).toBe('b-2nd')
  })
})

describe('attribute values and the fields behind them', () => {
  it('reads and writes lists of block ids (a countdown’s reveal and hide)', () => {
    expect(parseList(' eb-after, lead-after,,buy ')).toEqual(['eb-after', 'lead-after', 'buy'])
    expect(parseList(undefined)).toEqual([])
    expect(joinList(['buy', 'waitlist', 'buy', ''])).toBe('buy,waitlist')
  })

  it('keeps a countdown’s units in their order, at least one', () => {
    expect([...parseUnits('minutes,days,weeks')]).toEqual(['minutes', 'days'])
    expect(formatUnits(new Set(['seconds', 'days']))).toBe('days,seconds')
    expect(formatUnits(new Set())).toBe('minutes')
  })

  it('splits and joins a countdown’s moment', () => {
    expect(splitAt('2026-10-06T09:00')).toEqual({ date: '2026-10-06', time: '09:00' })
    expect(splitAt('6 Oct')).toEqual({ date: '', time: '' })
    expect(joinAt('2026-10-06', '')).toBe('2026-10-06T09:00')
  })

  it('shows the Buy button with the price filled in and stores what’s typed with the placeholders', () => {
    expect(buttonDisplay('Buy for {price}', '$29', 'The Field Guide')).toBe('Buy for $29')
    expect(buttonDisplay('Buy {product}', '$29', 'The Field Guide')).toBe('Buy The Field Guide')
    expect(buttonTemplate('Buy for $29', '$29', 'The Field Guide')).toBe('Buy for {price}')
    expect(buttonTemplate('Get The Field Guide for $29', '$29', 'The Field Guide')).toBe('Get {product} for {price}')
    // A different amount is the person's own text.
    expect(buttonTemplate('Buy for $290', '$29', 'The Field Guide')).toBe('Buy for $290')
    expect(buttonTemplate('Buy for $29.50', '$29', 'The Field Guide')).toBe('Buy for $29.50')
    // Round trip.
    expect(buttonTemplate(buttonDisplay('Buy for {price}', '$5+', 'Tips'), '$5+', 'Tips')).toBe('Buy for {price}')
  })

  it('accepts the videos the site plays', () => {
    for (const ok of ['', 'https://www.youtube.com/watch?v=dQw4w9WgXcQ', 'https://youtu.be/dQw4w9WgXcQ', 'https://www.youtube.com/shorts/dQw4w9WgXcQ', 'https://vimeo.com/123456789', 'https://player.vimeo.com/video/123456789']) {
      expect(isVideoUrl(ok), ok).toBe(true)
    }
    for (const bad of ['https://example.com/video.mp4', 'javascript:alert(1)', 'youtube.com/watch?v=dQw4w9WgXcQ', 'https://www.youtube.com/watch?v=', 'https://vimeo.com/channels']) {
      expect(isVideoUrl(bad), bad).toBe(false)
    }
  })
})

describe('dates in a time zone', () => {
  it('finds the moment a wall clock time stands for, summer and winter', () => {
    expect(zonedInstant('2026-10-06', '09:00', 'America/New_York')?.toISOString()).toBe('2026-10-06T13:00:00.000Z')
    expect(zonedInstant('2026-12-01', '09:00', 'America/New_York')?.toISOString()).toBe('2026-12-01T14:00:00.000Z')
    expect(zonedInstant('2026-10-06', '09:00', 'Asia/Ho_Chi_Minh')?.toISOString()).toBe('2026-10-06T02:00:00.000Z')
    expect(zonedInstant('2026-10-06', '09:00', 'UTC')?.toISOString()).toBe('2026-10-06T09:00:00.000Z')
  })

  it('refuses dates, times and zones it can’t read', () => {
    expect(zonedInstant('6 Oct', '09:00', 'UTC')).toBeNull()
    expect(zonedInstant('2026-10-06', '9am', 'UTC')).toBeNull()
    expect(zonedInstant('2026-10-06', '09:00', 'Mars/Olympus')).toBeNull()
  })

  it('names zones by their city and short name', () => {
    expect(zoneCity('America/New_York')).toBe('New York')
    expect(zoneCity('America/Argentina/Buenos_Aires')).toBe('Buenos Aires')
    expect(zoneCity('Etc/UTC')).toBe('UTC')
    expect(zoneAbbr('UTC', new Date())).toBe('UTC')
    expect(zoneAbbr('America/New_York', new Date('2026-10-06T13:00:00Z'))).toBe('EDT')
  })

  it('counts what’s left before a moment', () => {
    const now = new Date('2026-10-19T10:00:00Z')
    expect(timeLeft(new Date('2026-10-21T13:04:05Z'), now)).toEqual({ total: 2 * 86400 + 3 * 3600 + 4 * 60 + 5, days: 2, hours: 3, minutes: 4 })
    expect(timeLeft(new Date('2026-10-01T00:00:00Z'), now).total).toBe(0)
  })

  it('starts new countdowns and events on a date of the workspace’s calendar', () => {
    const now = new Date('2026-10-19T23:30:00Z')
    expect(datePlus(now, 14, 'UTC')).toBe('2026-11-02')
    expect(datePlus(now, 14, 'Asia/Ho_Chi_Minh')).toBe('2026-11-03')
  })
})

describe('comparing versions', () => {
  const v1 =
    '<section data-pb-id="b-hero" data-pb-name="Hero"><h1>Old title</h1><p>Lead</p><rc-product product="p1" button="Buy"></rc-product></section>' +
    '<section data-pb-id="b-faq" data-pb-name="FAQ"><details><summary>Q1</summary><p>A1</p></details></section>'
  const v2 =
    '<section data-pb-id="b-hero" data-pb-name="Hero"><h1>New title</h1><p>Lead</p><rc-product product="p1" button="Buy now"></rc-product></section>' +
    '<section data-pb-id="b-bio" data-pb-name="Author bio"><h2>Linh</h2></section>'

  it('says what changed block by block: text lines and live blocks’ settings', () => {
    expect(diffPages(v1, v2)).toEqual([
      {
        id: 'b-hero',
        name: 'Hero',
        kind: 'changed',
        lines: [
          { op: '-', text: 'Old title' },
          { op: '+', text: 'New title' },
          { op: '-', text: 'rc-product · product: p1 · button: Buy' },
          { op: '+', text: 'rc-product · product: p1 · button: Buy now' },
        ],
      },
      { id: 'b-bio', name: 'Author bio', kind: 'added', lines: [{ op: '+', text: 'Linh' }] },
      {
        id: 'b-faq',
        name: 'FAQ',
        kind: 'removed',
        lines: [
          { op: '-', text: 'Q1' },
          { op: '-', text: 'A1' },
        ],
      },
    ])
  })

  it('says nothing for the same page', () => {
    expect(diffPages(v1, v1)).toEqual([])
  })

  it('finds the blocks both people changed since the version they started from', () => {
    const mine = v1.replace('Old title', 'Mine').replace('A1', 'My answer')
    const theirs = v1.replace('A1', 'Their answer')
    expect(conflictBlocks(v1, mine, theirs)).toEqual({ both: ['FAQ'], mine: ['Hero', 'FAQ'], theirs: ['FAQ'] })
    expect(conflictBlocks(v1, mine, v1).both).toEqual([])
  })
})
