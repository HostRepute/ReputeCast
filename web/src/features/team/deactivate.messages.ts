import { defineMessages } from '@/i18n'

/**
 * Deactivate one member or several (TeamDialogs): what happens, the API keys
 * they created and a reason for the audit log. A member's name is their
 * first name in the description and the button, as on the board.
 */
export const m = defineMessages('team.deactivate', {
  eyebrow: 'Dialog · deactivate member',
  titleOne: 'Deactivate {name}?',
  titleMany: 'Deactivate {count, plural, one {# member} other {# members}}?',
  descriptionOne: '{name} keeps their history and can be reactivated later. Removing instead deletes the membership; the audit log still names them.',
  descriptionMany: 'They keep their history and can be reactivated later. The audit log still names them.',
  loadingChanges: 'Loading what changes…',
  confirmOne: 'Deactivate {name}',
  confirmMany: 'Deactivate {count, plural, one {# member} other {# members}}',
  doneOne: '{name} deactivated',
  doneMany: '{count, plural, one {# member deactivated} other {# members deactivated}}',
  doneDetail: '{count, plural, one {signed out of # session} other {signed out of # sessions}}',
  failedOne: "Couldn't deactivate {name}",
  failedMany: "Couldn't deactivate {count, plural, one {# member} other {# members}}",

  // What changes.
  signedOutEverywhere: '{count, plural, one {Signed out everywhere, now (# session)} other {Signed out everywhere, now (# sessions)}}',
  cantSignIn: "Can't sign in to {workspace}; other workspaces are unaffected",
  scheduled:
    '{count, plural, one {Their # scheduled campaign keeps its schedule and run under the workspace} other {Their # scheduled campaigns keep their schedule and run under the workspace}}',
  approvals: '{count, plural, one {# pending approval request stays open for approvers} other {# pending approval requests stay open for approvers}}',

  // The API keys they created. keys: the keys' names in quotes, joined as "A, B and C".
  keysLegend: 'API keys they created · {count}',
  keepKey: 'Keep {keys} working <k>({permissions})</k>',
  keepKeys: 'Keep {keys} working',
  revoke: '{count, plural, one {Revoke it now} other {Revoke them now}}',

  reason: 'Reason <k>· recorded in the audit log</k>',
  reasonPlaceholder: 'Contract ended',
})
