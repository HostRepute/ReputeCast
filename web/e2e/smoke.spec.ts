import { expect, test, type Page } from '@playwright/test'
import { signIn as demoSignIn } from '../scripts/lib/demo-auth.ts'

// The smoke test (review 2026-11 FE-8): the demo's owner signs in and the
// main screens load without an error, and ⌘K finds a campaign. Needs a
// server seeded with `reputecast seed`.

// The server accepts each authenticator code once, so the shared sign-in
// waits for the next code when a test before already used this one.
async function signIn(page: Page) {
  test.setTimeout(120_000)
  await demoSignIn(page.context(), '', 'linh@acme.io')
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
