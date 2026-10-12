/**
 * Signs a browser context in as a demo user, for the QA scripts. Uses the
 * demo password and the demo TOTP secret from internal/demo.
 */
import { createHmac } from 'node:crypto'
import type { BrowserContext } from '@playwright/test'

export const DEMO_PASSWORD = 'reputecast-demo'
export const DEMO_TOTP = 'JBSWY3DPEHPK3PXP'

function base32(s: string) {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const c of s.replace(/=+$/, '').toUpperCase()) bits += alphabet.indexOf(c).toString(2).padStart(5, '0')
  const bytes: number[] = []
  for (let i = 0; i + 8 <= bits.length; i += 8) bytes.push(parseInt(bits.slice(i, i + 8), 2))
  return Buffer.from(bytes)
}

/** totp returns the current 6-digit code for a base32 secret (RFC 6238, SHA-1, 30 s). */
export function totp(secret: string, now = Date.now()) {
  const counter = Buffer.alloc(8)
  counter.writeBigUInt64BE(BigInt(Math.floor(now / 1000 / 30)))
  const h = createHmac('sha1', base32(secret)).update(counter).digest()
  const o = h[h.length - 1]! & 0xf
  const n = ((h[o]! & 0x7f) << 24) | (h[o + 1]! << 16) | (h[o + 2]! << 8) | h[o + 3]!
  return String(n % 1_000_000).padStart(6, '0')
}

/** signIn signs ctx in as email through the API, answering the TOTP challenge. */
export async function signIn(ctx: BrowserContext, app: string, email: string) {
  const r = await ctx.request.post(`${app}/api/v1/auth/password`, { data: { email, password: DEMO_PASSWORD, remember: true } })
  if (!r.ok()) throw new Error(`sign-in as ${email} failed: ${r.status()} ${await r.text()}`)
  const body = (await r.json()) as { status: string; challenge?: string }
  if (body.status === 'two_factor') {
    const r2 = await ctx.request.post(`${app}/api/v1/auth/two-factor`, { data: { challenge: body.challenge, method: 'totp', code: totp(DEMO_TOTP) } })
    if (!r2.ok()) throw new Error(`two-factor for ${email} failed: ${r2.status()} ${await r2.text()}`)
  }
}
