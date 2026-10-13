/**
 * Test sign-in in a popup. The popup goes to the IdP and comes back to the
 * provider's page with ?test=done; the app's Cross-Origin-Opener-Policy
 * cuts the popup off from this window once it visits the IdP, so it tells
 * us through a BroadcastChannel (same origin) and closes itself.
 */
const CHANNEL = 'reputecast-sso-test'

export type TestDone = { type: 'sso-test-done'; providerId: string }

/** Opens the IdP; falls back to this tab when popups are blocked. */
export function openTestPopup(url: string) {
  const w = window.open(url, 'reputecast-sso-test', 'popup,width=520,height=720')
  if (!w) window.location.assign(url)
}

/** Calls onDone when a test sign-in for providerId finishes in any tab. */
export function onTestDone(providerId: string | undefined, onDone: () => void): () => void {
  if (typeof BroadcastChannel === 'undefined') return () => {}
  const ch = new BroadcastChannel(CHANNEL)
  ch.onmessage = (e: MessageEvent<TestDone>) => {
    if (e.data?.type === 'sso-test-done' && (!providerId || e.data.providerId === providerId)) onDone()
  }
  return () => ch.close()
}

/**
 * Run on the page the test returns to: tells the window that started the
 * test, then tries to close this tab. Browsers only close tabs a script
 * opened, so when the test ran in this tab (popups blocked) nothing closes
 * and the page shows the result.
 */
export function finishTestPopup(providerId: string) {
  if (typeof BroadcastChannel !== 'undefined') {
    const ch = new BroadcastChannel(CHANNEL)
    ch.postMessage({ type: 'sso-test-done', providerId } satisfies TestDone)
    ch.close()
  }
  window.close()
}
