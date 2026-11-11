import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'

// The site tracker's consent and Global Privacy Control (M62 G-28), on a
// fake site: the page, rc.js (read from internal/tracker/static) and the
// collect endpoint are all answered by page.route, so this needs no server.

const rcJS = readFileSync(fileURLToPath(new URL('../../internal/tracker/static/rc.js', import.meta.url)), 'utf8')
const SHOP = 'https://shop.tracker.test'
const RC = 'https://rc.tracker.test'
const SITE = '0192f0a0-0000-7000-8000-00000000c0de'

function html(waitConsent: boolean): string {
  const attr = waitConsent ? '' : ' data-wait-consent="false"'
  return `<!doctype html><html><head><title>Shop</title>
<script>window.reputecast=window.reputecast||function(){(reputecast.q=reputecast.q||[]).push(arguments)};</script>
<script async src="${RC}/track/v1/rc.js" data-workspace="acme-newsletter" data-site="${SITE}"${attr}></script>
</head><body><h1>Hoodie</h1>
<button id="allow" onclick="reputecast('consent', true)">Allow</button>
<button id="product" onclick="reputecast('product', {id: '8841203'})">View</button>
<button id="nav" onclick="history.pushState({}, '', '/p/next?utm_source=news')">Next</button>
</body></html>`
}

type Sent = { url: string; body: { vid: string; c?: number; events: Array<{ t: string; url: string; id?: string }> } }

async function shop(page: Page, waitConsent: boolean): Promise<Sent[]> {
  const sent: Sent[] = []
  await page.route(`${SHOP}/**`, (route) => route.fulfill({ status: 200, contentType: 'text/html', body: html(waitConsent) }))
  await page.route(`${RC}/track/v1/rc.js`, (route) => route.fulfill({ status: 200, contentType: 'text/javascript', body: rcJS }))
  await page.route(`${RC}/p/*/track/*/e`, (route) => {
    const req = route.request()
    if (req.method() === 'POST') sent.push({ url: req.url(), body: JSON.parse(req.postData() ?? '{}') })
    return route.fulfill({ status: 204, headers: { 'Access-Control-Allow-Origin': SHOP } })
  })
  return sent
}

const cookie = async (page: Page) => (await page.context().cookies(SHOP)).find((c) => c.name === 'rc_vid')

test('with wait-consent nothing is sent or stored before consent, then events arrive', async ({ page }) => {
  const sent = await shop(page, true)
  await page.goto(`${SHOP}/p/hoodie?utm_source=news&rc=tok`)
  await page.click('#product')
  await page.waitForTimeout(2_000)
  expect(sent).toHaveLength(0)
  expect(await cookie(page)).toBeUndefined()
  expect(await page.evaluate(() => localStorage.getItem('rc_vid'))).toBeNull()

  await page.click('#allow')
  await expect.poll(() => sent.length).toBeGreaterThan(0)
  const first = sent[0].body
  expect(sent[0].url).toBe(`${RC}/p/acme-newsletter/track/${SITE}/e`)
  expect(first.vid).toMatch(/^[A-Za-z0-9_-]{22}$/)
  expect(first.c).toBe(1)
  expect((first as { rc?: string }).rc).toBe('tok')
  expect(first.events.map((e) => e.t)).toEqual(['page', 'product'])
  expect((await cookie(page))?.value).toBe(first.vid)

  // A history navigation is a page view; the click token goes once.
  await page.click('#nav')
  await expect.poll(() => sent.length).toBeGreaterThan(1)
  expect(sent[1].body.events[0].url).toBe(`${SHOP}/p/next?utm_source=news`)
  expect((sent[1].body as { rc?: string }).rc).toBeUndefined()

  // consent(false) deletes the visitor id and stops sending.
  await page.evaluate(() => (window as unknown as { reputecast: (...a: unknown[]) => void }).reputecast('consent', false))
  expect(await cookie(page)).toBeUndefined()
  expect(await page.evaluate(() => localStorage.getItem('rc_vid'))).toBeNull()
  const before = sent.length
  await page.click('#product')
  await page.waitForTimeout(1_500)
  expect(sent).toHaveLength(before)
})

test('a site that does not wait sends at once', async ({ page }) => {
  const sent = await shop(page, false)
  await page.goto(`${SHOP}/`)
  await expect.poll(() => sent.length).toBeGreaterThan(0)
  expect(sent[0].body.c).toBeUndefined()
  expect(sent[0].body.events[0].t).toBe('page')
})

test('Global Privacy Control: nothing is ever sent or stored', async ({ page }) => {
  await page.addInitScript(() => Object.defineProperty(Navigator.prototype, 'globalPrivacyControl', { get: () => true }))
  const sent = await shop(page, false)
  await page.goto(`${SHOP}/p/hoodie`)
  await page.click('#allow')
  await page.click('#product')
  await page.click('#nav')
  await page.waitForTimeout(2_000)
  expect(sent).toHaveLength(0)
  expect(await cookie(page)).toBeUndefined()
  expect(await page.evaluate(() => [localStorage.getItem('rc_vid'), localStorage.getItem('rc_consent')])).toEqual([null, null])
})
