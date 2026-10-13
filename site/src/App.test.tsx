import { act, StrictMode } from 'react'
import { hydrateRoot } from 'react-dom/client'
import { render as renderToString } from './entry-server'
import { App } from './App'
import { PLATFORMS } from './sections/Compare'
import { formatStars } from './content/stars'

function prerendered(): HTMLDivElement {
  const root = document.createElement('div')
  root.innerHTML = renderToString()
  return root
}

describe('the prerendered page', () => {
  it('is complete HTML with one h1 and the landmarks', () => {
    const page = prerendered()
    expect(page.querySelectorAll('h1')).toHaveLength(1)
    expect(page.querySelector('h1')?.textContent).toBe('The newsletter platform that protects your inbox placement.')
    expect(page.querySelector('header')).not.toBeNull()
    expect(page.querySelector('main#main')).not.toBeNull()
    expect(page.querySelector('footer')).not.toBeNull()
    expect(page.querySelector('nav[aria-label="Site"]')).not.toBeNull()
    for (const id of ['features', 'providers', 'automations', 'compare', 'cost']) expect(page.querySelector(`#${id}`)).not.toBeNull()
  })

  it('renders final values for everything that animates, so readers without JavaScript see the numbers', () => {
    const text = prerendered().textContent ?? ''
    expect(text).toContain('31,604 recipients')
    expect(text).toContain('19,594 of 31,604 sent')
    expect(text).toContain('1,284 entered')
    expect(text).toContain('Monthly sending cost at 200,000 emails')
  })

  it('says the install is Postgres and the app, with no Redis', () => {
    const text = prerendered().textContent ?? ''
    expect(text).toContain('One command. Postgres and the app.')
    expect(text).not.toMatch(/redis/i)
  })

  it('compares with paid platforms only, with the list-price caveat', () => {
    const page = prerendered()
    const heads = [...page.querySelectorAll('#compare thead th')].map((th) => th.textContent?.trim()).filter(Boolean)
    expect(heads).toEqual(['Feature', ...PLATFORMS])
    const table = page.querySelector('#compare table')?.textContent ?? ''
    for (const oss of ['listmonk', 'Keila', 'Notifuse', 'Mautic', 'Plunk']) expect(table).not.toContain(oss)
    expect(page.querySelector('#compare')?.textContent).toContain('List prices change often, so check each provider')
    expect(page.querySelector('#cost')?.textContent).toContain('so check each provider')
  })

  it('names every link and keeps decorative graphics out of the accessibility tree', () => {
    const page = prerendered()
    for (const a of page.querySelectorAll('a')) expect((a.getAttribute('aria-label') ?? a.textContent ?? '').trim()).not.toBe('')
    for (const svg of page.querySelectorAll('svg')) expect(svg.getAttribute('aria-hidden')).toBe('true')
    // The flow canvas is one described image.
    expect(page.querySelector('[role="img"]')?.getAttribute('aria-label')).toMatch(/automation flow/)
  })

  it('hydrates without a mismatch', async () => {
    ;(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true
    const container = prerendered()
    document.body.appendChild(container)
    const errors: unknown[] = []
    const spy = vi.spyOn(console, 'error').mockImplementation((...args) => errors.push(args))
    await act(async () => {
      hydrateRoot(
        container,
        <StrictMode>
          <App />
        </StrictMode>,
        { onRecoverableError: (e) => errors.push(e) },
      )
    })
    spy.mockRestore()
    expect(errors.map((e) => String(Array.isArray(e) ? e[0] : e).slice(0, 300))).toEqual([])
    container.remove()
  })
})

describe('formatStars', () => {
  it('shortens counts like GitHub does', () => {
    expect(formatStars(950)).toBe('950')
    expect(formatStars(1234)).toBe('1.2k')
    expect(formatStars(2000)).toBe('2k')
    expect(formatStars(12_345)).toBe('12k')
  })
})
