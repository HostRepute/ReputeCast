/**
 * The app's first questions, asked while its scripts load: who is signed in
 * (/me) and, on a workspace's address, their place in it. Its own tiny entry
 * in index.html with no imports, so it runs as soon as the page is parsed;
 * the app asked only once its scripts had loaded and run, a second or more
 * later on a slow phone, and the screen's data waited behind both answers.
 *
 * Only a 200 is kept (lib/early.ts hands it to the app once); for anything
 * else the app asks again through its API client, which handles a signed-out
 * session, maintenance and errors.
 */
import type { EarlyAnswers } from './lib/early'

// The theme first, before anything is drawn: the member's choice kept in
// this browser (lib/theme, which this mirrors without importing it: an
// import would pull the app's shared chunk into this entry), else the
// device's setting. styles/themes.css reads html[data-theme].
try {
  const kept = localStorage.getItem('rc.theme')
  const dark = kept === 'dark' || (kept !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.dataset.theme = dark ? 'dark' : 'light'
} catch {
  // Blocked storage: themes.css follows the device until lib/theme runs.
}

function ask(path: string): Promise<unknown> {
  return fetch(path, { headers: { Accept: 'application/json' }, credentials: 'same-origin' })
    .then((r) => (r.ok ? r.json() : undefined))
    .catch(() => undefined)
}

const path = location.pathname
// The sign-in, invitation and password pages have no session to ask about.
if (!/^\/(sign-in|invite\/|reset-password)/.test(path)) {
  const early: EarlyAnswers = { me: ask('/api/v1/me') }
  const slug = /^\/w\/([^/]+)/.exec(path)?.[1]
  try {
    if (slug) early.workspace = { slug: decodeURIComponent(slug), me: ask(`/api/v1/workspaces/${slug}/me`) }
  } catch {
    // A malformed address: the app reports it.
  }
  ;(window as { __rcEarly?: EarlyAnswers }).__rcEarly = early
}
