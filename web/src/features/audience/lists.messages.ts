import { defineMessages } from '@/i18n'

/** Deleting and restoring a list (Dialogs board, "Delete list"; listDelete.ts). */
export const m = defineMessages('audience.lists', {
  // ── Delete list dialog ───────────────────────────────────────────────
  badge: 'Delete list',
  title: 'Delete "{name}"?',
  stillCounting: "Still counting who's on {name}. Try again in a moment.",
  counting: "Counting who's on {name}…",
  confirm: 'Delete list',
  deleting: 'Deleting…',
  typeName: 'Type the list name to confirm',
  typeHint: 'Deleted lists can be restored from the audit log for 30 days; sends in the meantime skip it.',
  affects: 'Affects',
  deleted: 'Deleted "{name}" · contacts kept',
  restored: 'Restored "{name}"',
  restoreFailed: 'Couldn\'t restore "{name}"',

  // ── What deleting it does (membersText) ──────────────────────────────
  kept: 'Suppressions are never removed.',
  nobody: 'Nobody is subscribed to this list. Suppressions are never removed.',
  onList: '{n, plural, one {{count} contact is on this list.} other {{count} contacts are on this list.}}',
  allElsewhere:
    '{n, plural, one {It is also on {lists} and stays subscribed there.} other {All of them are also on {lists} and stay subscribed there.}}',
  allAlone: '{n, plural, one {It is on no other list and will become a contact without a subscription.} other {All of them are on no other list and will become contacts without a subscription.}}',
  split:
    '{elsewhere, plural, one {{elsewhereCount} is also on {lists} and stays subscribed there} other {{elsewhereCount} are also on {lists} and stay subscribed there}}; {alone, plural, one {{aloneCount} is on no other list and will become a contact without a subscription.} other {{aloneCount} are on no other list and will become contacts without a subscription.}}',
  otherLists: '{n, plural, one {{count} other list} other {{count} other lists}}',

  // ── What uses it (affectsText) ───────────────────────────────────────
  forms: '{n, plural, one {{count} signup form} other {{count} signup forms}}',
  automations: '{n, plural, one {{count} automation trigger} other {{count} automation triggers}}',
  automationsNamed: '{n, plural, one {{count} automation trigger ({names})} other {{count} automation triggers ({names})}}',
  campaigns: '{n, plural, one {{count} scheduled campaign} other {{count} scheduled campaigns}}',
  usesIt: '{one, select, yes {{things} uses this list. It will pause.} other {{things} use this list. They will pause.}}',

  // ── What a restore turned back on (restoredText) ─────────────────────
  restoredForms: '{n, plural, one {{count} signup form} other {{count} signup forms}}',
  restoredAutomations: '{n, plural, one {{count} automation} other {{count} automations}}',
  liveAgain: '{one, select, yes {{things} is live again} other {{things} are live again}}',
  backOnSchedule: '{n, plural, one {{campaigns} is back on schedule} other {{campaigns} are back on schedule}}',
  membersBack: 'Its members are back.',
  stayAsIs: '{n, plural, one {{names} stays as it is.} other {{names} stay as they are.}}',
})
