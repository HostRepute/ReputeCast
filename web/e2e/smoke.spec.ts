import { createHmac } from 'node:crypto'
import { expect, test, type Page } from '@playwright/test'

// The smoke test (review 2026-11 FE-8): the demo's owner signs in and the
// main screens load without an error, and ⌘K finds a campaign. Needs a
// server seeded with `reputecast seed`.

const TOTP_SECRET = 'JBSWY3DPEHPK3PXP'

function base32(s: string): Buffer {
  const a = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const c of s) bits += a.indexOf(c).toString(2).padStart(5, '0')
  const out: number[] = []
  for (let i = 0; i + 8 <= bits.length; i += 8) out.push(parseInt(bits.slice(i, i + 8), 2))
  return Buffer.from(out)
}

function totp(secret: string): string {
  const counter = Buffer.alloc(8)
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000)))
  const h = createHmac('sha1', base32(secret)).update(counter).digest()
  const o = h[h.length - 1]! & 0xf
  const n = ((h[o]! & 0x7f) << 24) | (h[o + 1]! << 16) | (h[o + 2]! << 8) | h[o + 3]!
  return String(n % 1e6).padStart(6, '0')
}

async function signIn(page: Page) {
  const r = await page.request.post('/api/v1/auth/password', { data: { email: 'linh@acme.io', password: 'reputecast-demo', remember: true } })
  const body = await r.json()
  if (body.status === 'two_factor') {
    const r2 = await page.request.post('/api/v1/auth/two-factor', { data: { challenge: body.challenge, method: 'totp', code: totp(TOTP_SECRET) } })
    expect(r2.ok()).toBeTruthy()
  }
}

const screens: [string, RegExp][] = [
  ['/w/acme-newsletter', /Overview/],
  ['/w/acme-newsletter/campaigns', /Campaigns/],
  ['/w/acme-newsletter/audience', /Audience/],
  ['/w/acme-newsletter/automations', /Automations/],
  ['/w/acme-newsletter/deliverability', /Deliverability/],
  ['/w/acme-newsletter/settings', /Settings/],
  ['/w/acme-newsletter/admin/system', /Instance admin/],
]

test('the main screens load without errors', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  await signIn(page)
  for (const [path, title] of screens) {
    await page.goto(path)
    await expect(page.getByRole('heading', { level: 1 }).first()).toHaveText(title)
    await expect(page.getByRole('alert').filter({ hasText: /couldn't|went wrong/i })).toHaveCount(0)
  }
  expect(errors).toEqual([])
})

test('⌘K finds a campaign and opens it', async ({ page, isMobile }) => {
  test.skip(isMobile, 'a keyboard shortcut')
  await signIn(page)
  await page.goto('/w/acme-newsletter')
  await expect(page.getByRole('heading', { level: 1 }).first()).toHaveText(/Overview/)
  await page.keyboard.press('ControlOrMeta+k')
  const box = page.getByRole('combobox', { name: /Search campaigns, contacts/ })
  await expect(box).toBeFocused()
  await box.fill('digest')
  await expect(page.getByRole('group', { name: /^Campaigns · / })).toBeVisible()
  await box.press('Enter')
  await expect(page).toHaveURL(/\/campaigns\//)
})
