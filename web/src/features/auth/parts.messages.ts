import { defineMessages } from '@/i18n'

/** Pieces the sign-in, invitation, setup and account screens share. */
export const m = defineMessages('auth.parts', {
  notYou: 'not you?',
  digitLabel: 'Digit {n}',

  // Passkeys.
  passkeyUnsupported: "This browser doesn't support passkeys. Use another browser, or sign in with your password.",
  passkeyAddCancelled: 'Adding the passkey was cancelled or timed out. Try again when your device is ready.',
  passkeySignInCancelled: 'The passkey prompt was closed or timed out. Try again, or use your password.',
  passkeyAlreadyRegistered: 'This device already has a passkey for your account.',
  passkeyWrongDomain: "Passkeys only work on the instance's public URL. Open ReputeCast at that address and try again.",

  // Set up an authenticator app.
  totpEnterDigits: 'Enter the 6 digits the app shows.',
  tryAgain: 'Try again',
  qrLabel: 'QR code for the authenticator app',
  scanHelp: "Scan with 1Password, Google Authenticator, Authy or any TOTP app. Can't scan? Enter the key by hand:",
  copy: 'Copy',
  keyCopied: 'Key copied',
  copyFailed: "Couldn't copy",
  copyKeyByHand: 'Select the key and copy it by hand.',
  keyFacts: 'Account {account} · issuer {issuer} · 30-second codes',
  preparingKey: 'Preparing a key…',
  codeLabel: 'Enter the 6-digit code the app shows',
  confirmContinue: 'Confirm and continue',
  cancel: 'Cancel',

  // Recovery codes.
  codesFileTitle: '{instance} recovery codes for {email}',
  codesFileGenerated: 'Generated {date}. Each code signs you in once if you lose your authenticator or passkey.',
  codesFileKeep: 'Keep them somewhere safe, like a password manager. Generating new codes replaces these.',
  download: 'Download .txt',
  print: 'Print',
  copyAll: 'Copy all',
  codesCopied: 'Recovery codes copied',
  copyCodesByHand: 'Select the codes and copy them by hand.',
  codesList: 'Recovery codes',
  shownOnce: 'Shown once',
  shownOnceBody: "Each code signs you in once if you lose your authenticator. Store them somewhere safe; you won't see them again.",
  savedCodes: 'I have saved these codes',
  finish: 'Finish',
  confirmSavedFirst: 'Confirm that you saved the codes first',
})
