import { expect, test } from '@playwright/test'
import { openFeature, shownColumns } from './helpers'

// The pickers are radio buttons and CSS shows what they pick.
test.use({ javaScriptEnabled: false })

test('the comparison switches rival and view without JavaScript', async ({ page }) => {
  await page.goto('/')
  expect(await shownColumns(page)).toEqual(['reputecast', 'mailchimp'])
  await page.locator('label[for="cmp-kit"]').click()
  expect(await shownColumns(page)).toEqual(['reputecast', 'kit'])
  await page.locator('label[for="cmp-all"]').click()
  expect(await shownColumns(page)).toHaveLength(6)
})

test('the receipt opens the picked line without JavaScript', async ({ page }) => {
  await page.goto('/')
  expect(await openFeature(page)).toEqual(['Drag-and-drop composer'])
  await page.locator('label[for="fx-segments"]').click()
  expect(await openFeature(page)).toEqual(['Segments without SQL'])
})
