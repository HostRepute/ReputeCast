import { expect, test, type Page } from '@playwright/test'
import { signIn as demoSignIn } from '../scripts/lib/demo-auth.ts'

// Signup forms in a visitor's browser (M60 G-23, G-24): a three-step popup
// on a page of the publisher's site moves with Next and Back, takes focus to
// each step's heading, says which step it shows, and sends nothing before
// the last step; a page rule keeps it off other pages without asking the
// server. The site's page is served by the test at the dev server's own
// address (an origin the form accepts). Needs a server seeded with
// `reputecast seed`.

const ws = '/api/v1/workspaces/acme-newsletter'

// The server takes each authenticator code once: a second sign-in may wait
// for the next one.
async function signIn(page: Page) {
  test.setTimeout(180_000)
  await demoSignIn(page.context(), '', 'linh@acme.io')
}

interface SignupForm {
  id: string
  name: string
  kind: string
  publicId: string
  fields: { key: string; label: string; required: boolean }[]
  listIds: string[]
  defaultListIds: string[]
  listChoice: boolean
  doubleOptIn: boolean
  consentText: string
  spam: Record<string, unknown>
  afterSubmit: string
  redirectUrl: string
  allowedOrigins: string[]
  content: Record<string, unknown>
  urls: { script: string }
}

/** A popup in three steps, shown only on /e2e-forms/landing pages. */
async function threeStepPopup(page: Page): Promise<SignupForm> {
  const created = (await (await page.request.post(`${ws}/forms`, { data: { name: `Three steps ${Date.now()}`, kind: 'popup' } })).json()) as SignupForm
  const body = {
    name: created.name,
    kind: 'popup',
    fields: [
      { key: 'email', label: '', required: true },
      { key: 'topics', label: 'Topics', required: true, type: 'choice', multiple: true, options: [{ label: 'Product news', tag: 'product' }, { label: 'Engineering deep dives', tag: 'engineering' }] },
      { key: 'first_name', label: '', required: false },
    ],
    listIds: created.listIds,
    defaultListIds: created.defaultListIds,
    listChoice: false,
    doubleOptIn: true,
    consentText: created.consentText,
    spam: { ...created.spam, minSeconds: 1, turnstile: false },
    afterSubmit: 'message',
    redirectUrl: '',
    allowedOrigins: [],
    content: {
      ...created.content,
      popupDelay: 1,
      popupScroll: 0,
      hideSubscribed: false,
      steps: [
        { id: 's1', fields: ['email'], heading: 'Your email' },
        { id: 's2', fields: ['topics'], heading: 'What are you into?' },
        { id: 's3', fields: ['first_name'], heading: 'Almost done' },
      ],
      targeting: {
        pages: { mode: 'only', rules: [{ op: 'starts_with', value: '/e2e-forms/landing' }] },
        sources: { utmSource: [], utmMedium: [], utmCampaign: [], referrer: 'any' },
        devices: [],
        countries: { mode: 'all', codes: [] },
        visitors: 'all',
        schedule: {},
      },
    },
    publish: true,
  }
  const saved = await page.request.put(`${ws}/forms/${created.id}`, { data: body })
  expect(saved.status(), await saved.text()).toBe(200)
  return (await saved.json()) as SignupForm
}

/**
 * Serves the publisher's page under /e2e-forms/ with the form's script. The
 * demo's verified custom domain (news.acme.io) is where the script and its
 * session, step and submit addresses point: the test sends those requests
 * to the server under test, with the page's Origin as a browser would.
 */
async function hostPage(page: Page, form: SignupForm) {
  const base = test.info().project.use.baseURL ?? 'http://localhost:5173'
  const site = new URL(form.urls.script).origin
  if (site !== new URL(base).origin) {
    await page.route(`${site}/**`, async (route) => {
      const u = new URL(route.request().url())
      await route.fulfill({ response: await route.fetch({ url: `${base}${u.pathname}${u.search}` }) })
    })
  }
  await page.route('**/e2e-forms/**', (route) =>
    route.fulfill({
      contentType: 'text/html; charset=utf-8',
      body: `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Acme</title></head><body><h1>Acme blog</h1><p>An article.</p><script src="${form.urls.script}" async></script></body></html>`,
    }),
  )
}

test('a three-step popup moves with Next and Back and sends only from the last step', async ({ page }) => {
  await signIn(page)
  const form = await threeStepPopup(page)
  await hostPage(page, form)
  const posts: string[] = []
  page.on('request', (r) => {
    if (r.method() === 'POST' && r.url().includes(`/forms/${form.publicId}/submit`)) posts.push(r.url())
  })
  try {
    await page.goto('/e2e-forms/landing/hello')
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 10_000 })
    await expect(dialog.getByText('Step 1 of 3')).toBeVisible()

    // Next checks the step first: the email is required.
    await dialog.getByRole('button', { name: 'Next' }).click()
    await expect(dialog.getByText('Enter your email address.', { exact: false })).toBeVisible()
    await dialog.getByRole('textbox', { name: 'Email' }).fill('three.steps@fastmail.com')
    await dialog.getByRole('button', { name: 'Next' }).click()
    await expect(dialog.getByText('What are you into?', { exact: true })).toBeFocused()
    await expect(dialog.locator('[aria-live="polite"]')).toHaveText('Step 2 of 3')
    await expect(dialog.getByRole('textbox', { name: 'Email' })).toBeHidden()

    // Back keeps the answers.
    await dialog.getByRole('button', { name: 'Back' }).click()
    await expect(dialog.getByText('Your email', { exact: true })).toBeFocused()
    await expect(dialog.getByRole('textbox', { name: 'Email' })).toHaveValue('three.steps@fastmail.com')
    await dialog.getByRole('button', { name: 'Next' }).click()

    // A required choice question needs an answer before the last step.
    await dialog.getByRole('button', { name: 'Next' }).click()
    await expect(dialog.getByText('Almost done', { exact: true })).toBeHidden()
    await dialog.getByLabel('Engineering deep dives').check()
    await dialog.getByRole('button', { name: 'Next' }).click()
    await expect(dialog.getByText('Almost done', { exact: true })).toBeFocused()
    await expect(dialog.locator('[aria-live="polite"]')).toHaveText('Step 3 of 3')
    expect(posts).toHaveLength(0)

    await dialog.getByRole('checkbox', { name: /I agree/ }).check()
    await page.waitForTimeout(1200) // the form's timing check: at least a second
    await dialog.getByRole('button', { name: 'Subscribe' }).click()
    await expect(dialog.getByRole('status')).toContainText('Check your inbox')
    expect(posts).toHaveLength(1)
  } finally {
    await page.request.delete(`${ws}/forms/${form.id}`)
  }
})

test('a page rule keeps the popup off other pages without asking the server', async ({ page }) => {
  await signIn(page)
  const form = await threeStepPopup(page)
  await hostPage(page, form)
  const sessions: string[] = []
  page.on('request', (r) => {
    if (r.url().includes(`/forms/${form.publicId}/session`)) sessions.push(r.url())
  })
  try {
    const script = page.waitForResponse((r) => r.url().endsWith(`/forms/${form.publicId}.js`))
    await page.goto('/e2e-forms/pricing')
    expect((await script).status()).toBe(200)
    await page.waitForTimeout(2500) // past the popup's one-second delay
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(sessions).toHaveLength(0)
  } finally {
    await page.request.delete(`${ws}/forms/${form.id}`)
  }
})
