import { test as setup } from '@playwright/test'
import { signIn } from '../scripts/lib/demo-auth.ts'
import { demoOwnerState } from './demo-owner.ts'

// The demo's owner signs in once for the whole run, before the desktop and
// phone projects start, and the specs that need a member reuse the session.
// The server accepts each authenticator code once: when every test signed
// in, parallel workers raced for the same code, the losers waited for the
// same next one, and each lost race counted toward the sign-in lockout.
setup('the demo owner signs in', async ({ page }) => {
  setup.setTimeout(120_000) // a run started within 30 s of another waits for the next code
  await signIn(page.context(), '', 'linh@acme.io')
  await page.context().storageState({ path: demoOwnerState })
})
