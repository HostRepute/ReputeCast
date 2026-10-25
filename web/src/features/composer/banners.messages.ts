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
  /** A section the server names, in the conflict text. */
  partSubject: 'subject',
  partBody: 'email body',
})
