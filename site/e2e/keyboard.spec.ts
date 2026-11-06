import { expect, test, type Page } from '@playwright/test'
import { openFeature, shownColumns } from './helpers'

test.use({ reducedMotion: 'reduce' })

/** What shows that an element has keyboard focus: its ring, its label's, or the comparison card's. */
function ringOf(page: Page) {
  return page.evaluate(() => {
    const el = document.activeElement as HTMLElement | null
    if (!el || el === document.body) return null
    const holder = el.matches('input[type="radio"]') ? el.closest('label') : el.matches('.cmp-scroll') ? el.closest('.cmp') : el
    const s = getComputedStyle(holder!)
    const name = (el.getAttribute('aria-label') ?? holder?.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)
    return { name, ring: s.outlineStyle !== 'none' && parseFloat(s.outlineWidth) > 0 }
  })
}

test('the skip link comes first and moves focus to the content', async ({ page }) => {
  await page.goto('/')
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: 'Skip to content' })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.locator('main#main')).toBeFocused()
})

test('every stop on the way through the page shows a focus ring', async ({ page }) => {
  await page.goto('/')
  const stops: { name: string; ring: boolean }[] = []
  for (let i = 0; i < 80; i++) {
    await page.keyboard.press('Tab')
    const stop = await ringOf(page)
    if (!stop || (stops.length > 0 && stop.name === stops[0]!.name)) break
    stops.push(stop)
  }
  expect(stops.length).toBeGreaterThan(20)
  expect(stops.filter((s) => !s.ring).map((s) => s.name)).toEqual([])
})

test('arrow keys pick a rival and the view, and Tab moves between the two pickers', async ({ page }) => {
  await page.goto('/')
  await page.locator('#cmp-mailchimp').focus()
  await page.keyboard.press('ArrowRight')
  await expect(page.locator('#cmp-brevo')).toBeChecked()
  expect(await shownColumns(page)).toEqual(['reputecast', 'brevo'])
  await page.keyboard.press('Tab')
  await expect(page.locator('#cmp-vs')).toBeFocused()
  await page.keyboard.press('ArrowRight')
  await expect(page.locator('#cmp-all')).toBeChecked()
  expect(await shownColumns(page)).toHaveLength(6)
})

test("arrow keys move along the receipt's lines and open each one", async ({ page }) => {
  await page.goto('/')
  await page.locator('#fx-composer').focus()
  await page.keyboard.press('ArrowDown')
  await expect(page.locator('#fx-automations')).toBeChecked()
  expect(await openFeature(page)).toEqual(['Visual automations'])
  await page.keyboard.press('ArrowUp')
  await page.keyboard.press('ArrowUp')
  // From the first line the group wraps to the last.
  await expect(page.locator('#fx-migration')).toBeChecked()
  expect(await openFeature(page)).toEqual(['Migration that keeps consent'])
})

test('when "All six" scrolls sideways, Tab reaches the table and its card shows the ring', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 900 })
  await page.goto('/')
  await page.locator('#cmp-vs').focus()
  await page.keyboard.press('ArrowRight')
  await page.keyboard.press('Tab')
  await expect(page.getByRole('region', { name: 'Comparison table, scrolls sideways' })).toBeFocused()
  expect(await ringOf(page)).toMatchObject({ ring: true })
})
