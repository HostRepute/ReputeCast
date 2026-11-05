/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { act, StrictMode } from 'react'
import { hydrateRoot } from 'react-dom/client'
import { render as renderToString } from './entry-server'
import { App } from './App'
import { links } from './content/links'
import { APP_LANGUAGE_NAMES } from './sections/AllFeatures'
import { PLATFORMS } from './sections/Compare'
import { HERO_COMMAND } from './sections/Hero'
import { INSTALLED_LINE } from './sections/Install'
import { formatStars } from './content/stars'

const repoFile = (path: string) => readFileSync(join(import.meta.dirname, '../..', path), 'utf8')

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
    expect(text).toContain('31,192 recipients')
    expect(text).toContain('19,594 of 31,192 sent')
    expect(text).toContain('1,284 entered')
    expect(text).toContain('Monthly sending cost at 200,000 emails')
  })

  it('says the install is Postgres and the app, with no Redis', () => {
    const text = prerendered().textContent ?? ''
    expect(text).toContain('One command. Postgres and the app.')
    expect(text).not.toMatch(/redis/i)
  })

  it('gives install commands that run: Compose reads no compose file over HTTPS, so they download it first', () => {
    const page = prerendered()
    expect(page.textContent).not.toMatch(/docker compose -f https?:/)
    expect(page.querySelector('pre.term')?.textContent).toBe(HERO_COMMAND.join('\n'))
    expect(HERO_COMMAND[0]).toBe(`curl -fsSLO ${links.composeUrl} &&`)
    expect(HERO_COMMAND[1]?.trim()).toBe('docker compose up -d')
    const lines = [...page.querySelectorAll<HTMLElement>('pre.term .ln')]
    expect(lines).toHaveLength(HERO_COMMAND.length)
    for (const line of lines) {
      expect(Number(line.style.getPropertyValue('--chars'))).toBe(line.querySelector('.typed')?.textContent?.length)
    }
    expect(page.querySelector('#install pre')?.textContent?.split('\n')[0]).toBe(`curl -fsSLO ${links.composeUrl}`)
  })

  it('finds the temporary password with the line the server prints after installing', () => {
    expect(repoFile('internal/cli/install.go')).toContain(`"  ${INSTALLED_LINE}."`)
    expect(prerendered().querySelector('#install pre')?.textContent).toContain(`docker compose logs app | grep -A4 "${INSTALLED_LINE}"`)
  })

  it('names every language the app ships in (web/src/i18n/languages.ts)', () => {
    const source = repoFile('web/src/i18n/languages.ts')
    const list = /export const APP_LANGUAGES[^=]*=\s*\[([\s\S]*?)\n\]/.exec(source)?.[1] ?? ''
    const codes = [...list.matchAll(/code: '([a-z]+)'/g)].map((m) => m[1]!)
    expect(codes.length).toBeGreaterThan(1)
    const english = new Intl.DisplayNames('en', { type: 'language' })
    expect([...APP_LANGUAGE_NAMES].sort()).toEqual(codes.map((c) => english.of(c)).sort())
    expect(prerendered().textContent).toContain(`the app itself in ${APP_LANGUAGE_NAMES.slice(0, -1).join(', ')} and ${APP_LANGUAGE_NAMES.at(-1)}.`)
  })

  it('links only to the repository, the docs and HostRepute (there is no hosted demo or Discord server)', () => {
    const page = prerendered()
    const hosts = [...page.querySelectorAll('a[href^="http"]')].map((a) => new URL(a.getAttribute('href')!).host)
    expect([...new Set(hosts)].sort()).toEqual(['docs.reputecast.com', 'github.com', 'hostrepute.com'])
    for (const a of page.querySelectorAll('a[href^="#"]')) expect(page.querySelector(a.getAttribute('href')!)).not.toBeNull()
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
