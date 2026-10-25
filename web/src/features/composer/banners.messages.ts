import { defineMessages } from '@/i18n'

/**
 * The banners under the composer's top bar (ComposerStates): a conflict with
 * someone else's save, and edits recovered from this browser's journal.
 */
export const mn = defineMessages('composer.banners', {
  // ── Conflict ─────────────────────────────────────────────────────────
  conflictTitle: '{who} saved v{version} while you were editing.',
  /** {parts} lists sections both people changed: "hero section and subject". */
  conflictBoth: 'You both changed the {parts}. Your edits stay in this browser until you choose; nothing is lost either way.',
  conflictTheirs: '{name} changed the {parts}. Your edits stay in this browser until you choose; nothing is lost either way.',
  conflictPlain: 'Your edits stay in this browser until you choose; nothing is lost either way.',
  resolveGroup: 'Resolve the conflict',
  keepMine: 'Keep mine',
  keepMineText: "Saves yours as v{next} on top of v{version}. {name}'s v{version} stays in version history.",
  compare: 'Compare',
  compareText: 'Shows both versions side by side, so you can take sections from each.',
  loadTheirs: 'Load theirs',
  loadTheirsText: "Keeps yours in version history as v{next}, then opens {name}'s v{version}.",
  someone: 'Someone',
  loadTheirsDone: "{name}'s v{version} is open",
  loadTheirsDoneDetail: 'Your edits are in version history as v{kept}.',
  loadTheirsFailed: "Couldn't load their version",
  // ── Recovered edits (the browser journal) ───────────────────────────
  /** {from} and {to} are times: "09:31–09:35". */
  recoveredTitle: "{minutes, plural, one {# minute} other {# minutes}} of edits from this browser weren't saved ({from}–{to}).",
  recoveredBody: "They're only on this device. Restoring saves them as v{next} on top of v{version}; discarding can't be undone.",
  /** The edits were made on an older version than the one open now. */
  recoveredBodyOlder:
    "They're only on this device, made on v{base}; {name} saved v{version} since. Restoring saves them as v{next} on top, and v{version} stays in version history. Discarding can't be undone.",
  /** On a language version, whose saves don't show version numbers. */
  recoveredBodyVersionless: "They're only on this device. Restoring saves them as a new version; discarding can't be undone.",
  restoreThem: 'Restore them',
  discard: 'Discard',
  /** The label of the version recovered edits are saved as. */
  recoveredLabel: 'recovered from this browser ({from}–{to})',
  recoveredDone: 'Your edits are back',
  recoveredDoneDetail: 'Saved as v{version}.',
  recoveredFailed: "The edits couldn't be saved yet",
  recoveredFailedDetail: 'They stay in this browser. Check the connection and try Restore again.',
  /** A section the server names, in the conflict text. */
  partSubject: 'subject',
  partBody: 'email body',
})
