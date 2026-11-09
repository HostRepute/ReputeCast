import { defineMessages } from '@/i18n'

/**
 * Appearance (M54): Account & security › Appearance, the user menu's theme
 * choice and the command palette's theme commands.
 */
export const m = defineMessages('appearance', {
  title: 'Appearance',
  caption: 'Light or dark, on every screen you sign in on. Emails, public pages and their previews keep their own colours.',
  legend: 'Theme',
  system: 'Same as device',
  light: 'Light',
  dark: 'Dark',
  systemDetail: 'Switches with your device’s light or dark setting. It’s {theme} now.',
  lightDetail: 'Dark text on light surfaces.',
  darkDetail: 'Light text on dark surfaces. Easier on the eyes at night.',
  nowLight: 'light',
  nowDark: 'dark',
  savedNote: 'Saved to your account, so it applies in every browser you sign in on.',
  // Toasts.
  switchedLight: 'Light theme on.',
  switchedDark: 'Dark theme on.',
  switchedSystem: 'The theme follows your device.',
  notSaved: 'Your theme wasn’t saved, so it’s back to how it was.',
  // The user menu.
  menuLabel: 'Theme',
  // The command palette (typed, like single settings).
  useLight: 'Use the light theme',
  useDark: 'Use the dark theme',
  useSystem: 'Use the device’s theme',
  paletteDetail: 'Appearance',
})
