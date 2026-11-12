/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { act, StrictMode } from 'react'
import { hydrateRoot } from 'react-dom/client'
import { render as renderToString } from './entry-server'
import { App } from './App'
import { links } from './content/links'
import { APP_LANGUAGE_NAMES, groups } from './sections/AllFeatures'
import { PLATFORMS, RIVALS, rows, slug } from './sections/Compare'
import { HERO_COMMAND } from './sections/Hero'
import { DEPLOY_PLATFORMS, HELM_COMMAND, INSTALLED_LINE } from './sections/Install'
import { formatStars } from './content/stars'

const repoFile = (path: string) => readFileSync(join(import.meta.dirname, '../..', path), 'utf8')

function spoken(el: Element): string {
  const copy = el.cloneNode(true) as Element
  for (const hidden of copy.querySelectorAll('[aria-hidden="true"]')) hidden.remove()
  return copy.textContent?.trim() ?? ''
}

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

  it('offers the Helm chart and the platforms next to the Compose command, each from a file in the repository', () => {
    const install = prerendered().querySelector('#install')!
    const pres = [...install.querySelectorAll('pre')]
    expect(pres).toHaveLength(2)
    expect(pres[1]?.textContent).toBe(HELM_COMMAND)
    expect(HELM_COMMAND).toContain(`helm install reputecast \\\n  ${links.chartUrl} \\`)
    expect(links.chartUrl).toBe('oci://ghcr.io/hostrepute/charts/reputecast')
    expect(repoFile('.github/workflows/release.yml')).toContain('helm push dist/reputecast-*.tgz oci://ghcr.io/hostrepute/charts')
    // Every value the command sets is one the chart's schema knows.
    const schema = JSON.parse(repoFile('deploy/helm/reputecast/values.schema.json')) as { properties: Record<string, { properties?: object }> }
    const sets = [...HELM_COMMAND.matchAll(/--set ([\w.]+)=/g)].map((m) => m[1]!)
    expect(sets).toEqual(['publicURL', 'database.url', 'admin.email'])
    for (const path of sets) {
      const [top, sub] = path.split('.')
      expect(schema.properties[top!]).toBeDefined()
      if (sub) expect(schema.properties[top!]?.properties).toHaveProperty(sub)
    }

    const group = install.querySelector('[role="group"]')!
    expect(install.querySelector(`#${group.getAttribute('aria-labelledby')}`)?.textContent).toBe('On a platform, with Postgres beside it:')
    const buttons = [...group.querySelectorAll('a')].map((a) => [a.textContent, a.getAttribute('href')])
    expect(buttons).toEqual(DEPLOY_PLATFORMS.map((p) => [p.name, p.href]))
    expect(links.render).toBe('https://render.com/deploy?repo=https://github.com/hostrepute/reputecast')
    expect(links.digitalOcean).toBe('https://cloud.digitalocean.com/apps/new?repo=https://github.com/hostrepute/reputecast/tree/main')
    // The buttons read these files from the repository; the other two open them.
    expect(repoFile('render.yaml')).toContain('url: ghcr.io/hostrepute/reputecast:latest')
    expect(repoFile('.do/deploy.template.yaml')).toMatch(/^spec:$/m)
    expect(links.railway).toBe(`${links.repo}/tree/main/deploy/railway`)
    expect(repoFile('deploy/railway/README.md')).toContain('${{Postgres.DATABASE_URL}}')
    expect(links.coolify).toBe(`${links.repo}/blob/main/deploy/coolify/reputecast.yaml`)
    expect(repoFile('deploy/coolify/reputecast.yaml')).toMatch(/^# port: 8080$/m)
  })

  it('names every language the app ships in (web/src/i18n/languages.ts)', () => {
    const source = repoFile('web/src/i18n/languages.ts')
    const list = /export const APP_LANGUAGES[^=]*=\s*\[([\s\S]*?)\n\]/.exec(source)?.[1] ?? ''
    const codes = [...list.matchAll(/code: '([a-z]+)'/g)].map((m) => m[1]!)
    expect(codes.length).toBeGreaterThan(1)
    const english = new Intl.DisplayNames('en', { type: 'language' })
    expect([...APP_LANGUAGE_NAMES].sort()).toEqual(codes.map((c) => english.of(c)).sort())
    expect(prerendered().textContent).toContain(`The app itself in ${APP_LANGUAGE_NAMES.slice(0, -1).join(', ')} and ${APP_LANGUAGE_NAMES.at(-1)}`)
  })

  it('links only to the repository, the docs, HostRepute and the platforms that deploy it (there is no hosted demo or Discord server)', () => {
    const page = prerendered()
    const hosts = [...page.querySelectorAll('a[href^="http"]')].map((a) => new URL(a.getAttribute('href')!).host)
    expect([...new Set(hosts)].sort()).toEqual(['cloud.digitalocean.com', 'docs.reputecast.com', 'github.com', 'hostrepute.com', 'render.com'])
    for (const a of page.querySelectorAll('a[href^="#"]')) expect(page.querySelector(a.getAttribute('href')!)).not.toBeNull()
  })

  it('compares with paid platforms only, with the list-price caveat', () => {
    const page = prerendered()
    const heads = [...page.querySelectorAll('#compare [role="columnheader"]')].map(spoken)
    expect(heads).toEqual(['Feature', ...PLATFORMS])
    const table = page.querySelector('#compare [role="table"]')?.textContent ?? ''
    for (const oss of ['listmonk', 'Keila', 'Notifuse', 'Mautic', 'Plunk']) expect(table).not.toContain(oss)
    expect(page.querySelector('#compare')?.textContent).toContain('List prices change often, so check each provider')
    expect(page.querySelector('#cost')?.textContent).toContain('so check each provider')
  })

  it('picks the rival and the view with radio buttons, so the comparison works without JavaScript', () => {
    const page = prerendered()
    const radios = (name: string) => [...page.querySelectorAll<HTMLInputElement>(`#compare input[type="radio"][name="${name}"]`)]
    expect(radios('cmp-rival').map((r) => r.value)).toEqual(RIVALS.map(slug))
    expect(radios('cmp-view').map((r) => r.value)).toEqual(['vs', 'all'])
    for (const name of ['cmp-rival', 'cmp-view']) {
      expect(radios(name).filter((r) => r.defaultChecked).map((r) => r.value)).toEqual([name === 'cmp-rival' ? 'mailchimp' : 'vs'])
      for (const r of radios(name)) expect(page.querySelector(`label[for="${r.id}"]`)?.textContent).toBeTruthy()
    }
    const rowsShown = page.querySelectorAll('#compare [role="rowgroup"]:last-child [role="row"]')
    expect(rowsShown).toHaveLength(rows.length)
    for (const row of rowsShown) {
      expect(row.querySelectorAll('[role="rowheader"]')).toHaveLength(1)
      expect([...row.querySelectorAll('[role="cell"]')].map((c) => c.getAttribute('data-p'))).toEqual(PLATFORMS.map(slug))
    }
  })

  it('lists every feature on a $0 receipt whose lines are radio buttons, each followed by its card', () => {
    const page = prerendered()
    const features = groups.flatMap((g) => g.features)
    const radios = [...page.querySelectorAll<HTMLInputElement>('.fx input[type="radio"][name="feature"]')]
    expect(radios.map((r) => r.value)).toEqual(features.map((f) => f.id))
    expect(radios.filter((r) => r.defaultChecked).map((r) => r.value)).toEqual([features[0]!.id])
    for (const [i, r] of radios.entries()) {
      // CSS shows a card with .fx-row:has(input:checked) + .fx-detail.
      const label = r.parentElement
      const card = label?.nextElementSibling
      expect(label?.matches(`label.fx-row[for="${r.id}"]`)).toBe(true)
      expect(label?.querySelector('.fx-name')?.textContent).toBe(features[i]!.line)
      expect(card?.matches('.fx-detail')).toBe(true)
      expect(card?.querySelector('h3')?.textContent).toBe(features[i]!.title)
      expect(card?.querySelectorAll('li')).toHaveLength(features[i]!.items.length)
      expect(card?.querySelector('a')?.getAttribute('href')).toMatch(/^https:\/\/docs\.reputecast\.com\/.+\/$/)
    }
    expect(page.querySelector('.fx-total')?.textContent).toContain(`$0.00A month, for all ${features.length}.`)
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
