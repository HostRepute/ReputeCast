import { expect, test } from '@playwright/test'
import { axe, openFeature, shownColumns, sideways, textSpacing } from './helpers'

// Final states (no entrance animations), so axe measures what readers see.
test.use({ reducedMotion: 'reduce' })

test('the page has no accessibility violations', async ({ page }) => {
  await page.goto('/')
  expect(await axe(page)).toEqual([])
})

test('nor once the pickers have changed: all six platforms, another feature', async ({ page }) => {
  await page.goto('/')
  await page.locator('label[for="cmp-all"]').click()
  await page.locator('label[for="fx-privacy"]').click()
  expect(await shownColumns(page)).toHaveLength(6)
  expect(await openFeature(page)).toEqual(['Privacy built in'])
  expect(await axe(page)).toEqual([])
})

for (const width of [320, 834]) {
  test(`at ${width} px wide`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/')
    expect(await axe(page)).toEqual([])
    expect(await sideways(page)).toBe(0)
  })
}

test('reflows at 320 px with wider text spacing: nothing scrolls sideways, no picker label overflows', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 900 })
  await page.goto('/')
  await page.addStyleTag({ content: textSpacing })
  expect(await sideways(page)).toBe(0)
  const squeezed = await page.evaluate(() =>
    [...document.querySelectorAll<HTMLElement>('.cmp-opts label, .fx-row, .btn, .badge')].filter((el) => el.offsetWidth > 0 && el.scrollWidth > el.clientWidth + 1).map((el) => el.textContent?.trim()),
  )
  expect(squeezed).toEqual([])
})

test.describe('with motion', () => {
  test.use({ reducedMotion: 'no-preference' })

  test('paused, the page has no accessibility violations either', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'Pause motion' }).click()
    await expect(page.getByRole('button', { name: 'Play motion' })).toBeVisible()
    expect(await axe(page)).toEqual([])
  })

  test('an option picked while motion is paused shows at once', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'Pause motion' }).click()
    await page.locator('label[for="cmp-brevo"]').click()
    await page.locator('label[for="fx-earn"]').click()
    const opacity = await page.evaluate(() =>
      [document.querySelector('.cmp-table > [role="rowgroup"]:last-child .cmp-cell[data-p="brevo"]'), document.querySelector('#fx-earn')?.closest('label')?.nextElementSibling].map(
        (el) => getComputedStyle(el!).opacity,
      ),
    )
    expect(opacity).toEqual(['1', '1'])
  })
})

test.describe('forced colours', () => {
  test.use({ forcedColors: 'active' })

  test('mark the picked options with the system highlight', async ({ page }) => {
    await page.goto('/')
    // Picked, not picked: the comparison's rivals, then the receipt's lines.
    const filled = await page.evaluate(() =>
      ['#cmp-mailchimp', '#cmp-brevo', '#fx-composer', '#fx-automations'].map((id) => !/, 0\)$/.test(getComputedStyle(document.querySelector(id)!.closest('label')!).backgroundColor)),
    )
    expect(filled).toEqual([true, false, true, false])
  })
})
