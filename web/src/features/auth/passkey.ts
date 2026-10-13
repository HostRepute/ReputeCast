import {
  browserSupportsWebAuthn,
  startAuthentication,
  startRegistration,
  WebAuthnError,
  type PublicKeyCredentialCreationOptionsJSON,
  type PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'

/**
 * Passkeys (WebAuthn) through @simplewebauthn/browser. The server sends the
 * standard options JSON and verifies the browser's answer; nothing secret
 * ever reaches the page.
 */

export const passkeysSupported = () => typeof window !== 'undefined' && browserSupportsWebAuthn()

/** "Sign in with a passkey": no email needed, the passkey names the account. */
export async function passkeySignIn(remember = true): Promise<Schemas['SignInResult']> {
  const { data: opts } = await http.POST('/api/v1/auth/passkey/options')
  const credential = await startAuthentication({ optionsJSON: opts!.options as PublicKeyCredentialRequestOptionsJSON })
  const { data } = await http.POST('/api/v1/auth/passkey', { body: { challenge: opts!.challenge, credential, remember } })
  return data!
}

/** Confirms a password sign-in with a passkey (second factor). */
export async function passkeySecondFactor(challenge: string): Promise<Schemas['SignInResult']> {
  const { data: opts } = await http.POST('/api/v1/auth/two-factor/passkey/options', { body: { challenge } })
  const credential = await startAuthentication({ optionsJSON: opts!.options as PublicKeyCredentialRequestOptionsJSON })
  const { data } = await http.POST('/api/v1/auth/two-factor/passkey', { body: { challenge, passkeyChallenge: opts!.challenge, credential } })
  return data!
}

/** Adds a passkey to the signed-in account. */
export async function registerPasskey(name?: string): Promise<Schemas['AccountPasskey']> {
  const { data: opts } = await http.POST('/api/v1/me/passkeys/options')
  const credential = await startRegistration({ optionsJSON: opts!.options as PublicKeyCredentialCreationOptionsJSON })
  const { data } = await http.POST('/api/v1/me/passkeys', { body: { challenge: opts!.challenge, credential, name: name || undefined } })
  return data!
}

/** True when the person closed the browser's passkey prompt. */
export function passkeyCancelled(e: unknown): boolean {
  if (e instanceof WebAuthnError) return e.code === 'ERROR_CEREMONY_ABORTED' || (e.cause as Error | undefined)?.name === 'NotAllowedError'
  return e instanceof Error && (e.name === 'NotAllowedError' || e.name === 'AbortError')
}

/** A sentence for a failed passkey ceremony. */
export function passkeyErrorMessage(e: unknown, action: 'sign-in' | 'add'): string {
  if (e instanceof ApiError) return e.message
  if (!passkeysSupported()) return "This browser doesn't support passkeys. Use another browser, or sign in with your password."
  if (passkeyCancelled(e)) {
    return action === 'add'
      ? 'Adding the passkey was cancelled or timed out. Try again when your device is ready.'
      : 'The passkey prompt was closed or timed out. Try again, or use your password.'
  }
  if (e instanceof WebAuthnError) {
    switch (e.code) {
      case 'ERROR_AUTHENTICATOR_PREVIOUSLY_REGISTERED':
        return 'This device already has a passkey for your account.'
      case 'ERROR_INVALID_DOMAIN':
      case 'ERROR_INVALID_RP_ID':
        return "Passkeys only work on the instance's public URL. Open ReputeCast at that address and try again."
    }
  }
  return errorMessage(e)
}
