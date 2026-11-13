import { expect, test } from '@playwright/test'
import { demoOwnerState } from './demo-owner.ts'

// The smoke test (review 2026-11 FE-8): the demo's owner, signed in once for
// the run (sign-in.setup.ts), loads the main screens without an error, and
// ⌘K finds a campaign. Needs a server seeded with `reputecast seed`.
test.use({ storageState: demoOwnerState })

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
  for (const [path, title] of screens) {
    await page.goto(path)
    await expect(page.getByRole('heading', { level: 1 }).first()).toHaveText(title)
    await expect(page.getByRole('alert').filter({ hasText: /couldn't|went wrong/i })).toHaveCount(0)
  }
  expect(errors).toEqual([])
})

test('⌘K finds a campaign and opens it', async ({ page, isMobile }) => {
  test.skip(isMobile, 'a keyboard shortcut')
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

// Dark mode (M54): the user menu's theme row switches at once, the choice is
// the account's (it survives a reload and /me says so), and Same as device
// follows the device again. Desktop only: both projects use the same
// member, and the choice is theirs.
test('the theme switches from the user menu and is kept on the account', async ({ page, isMobile }) => {
  test.skip(isMobile, 'one member, one theme: the desktop run checks it')
  await page.emulateMedia({ colorScheme: 'light' })
  await page.goto('/w/acme-newsletter')
  await expect(page.getByRole('heading', { level: 1 }).first()).toHaveText(/Overview/)
  const html = page.locator('html')
  await expect(html).toHaveAttribute('data-theme', 'light')

  const menu = page.getByRole('button', { name: /account menu/ })
  await menu.click()
  await page.getByRole('menuitemradio', { name: 'Dark' }).click()
  await expect(html).toHaveAttribute('data-theme', 'dark')
  // The menu stays open, showing the choice.
  await expect(page.getByRole('menuitemradio', { name: 'Dark' })).toHaveAttribute('aria-checked', 'true')
  await expect.poll(async () => ((await (await page.request.get('/api/v1/me')).json()) as { user: { theme: string } }).user.theme).toBe('dark')
  await page.keyboard.press('Escape')

  await page.reload()
  await expect(html).toHaveAttribute('data-theme', 'dark')

  await menu.click()
  await page.getByRole('menuitemradio', { name: 'Same as device' }).click()
  await expect(html).toHaveAttribute('data-theme', 'light')
  await expect.poll(async () => ((await (await page.request.get('/api/v1/me')).json()) as { user: { theme: string } }).user.theme).toBe('system')
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(html).toHaveAttribute('data-theme', 'dark')
})
