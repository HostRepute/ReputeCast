import { defineMessages } from '@/i18n'

/** The top bar's Assistant button and the panel's place in the shell (first load). */
export const ml = defineMessages('assistant.launcher', {
  launcher: 'Assistant',
  launcherWaiting: 'Assistant, a confirmation is waiting',
  // keys: the shortcut ("⌘J", "Ctrl J").
  launcherTitle: 'Assistant ({keys})',
  loading: 'Loading the assistant…',
  loadFailed: 'The assistant didn’t load',
})
