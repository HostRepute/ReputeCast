import { fileURLToPath } from 'node:url'

// The demo owner's session for the run, saved by sign-in.setup.ts and
// reused by the specs that need a member (test.use({ storageState })). It
// lives in the output directory, which Playwright empties at the start of
// each run and which is neither committed nor uploaded by CI.
export const demoOwnerState = fileURLToPath(new URL('../test-results/.auth/demo-owner.json', import.meta.url))
