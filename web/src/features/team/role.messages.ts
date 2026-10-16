import { defineMessages } from '@/i18n'

/** Change role or scope, and the blocked change for the last Owner (TeamDialogs). */
export const m = defineMessages('team.role', {
  eyebrow: 'Dialog · change role or scope',
  title: 'Change role or scope for {name}',
  description: "Changes apply at once, on {name}'s next request. Both the old and the new value go to the audit log.",
  nothingChanged: 'Nothing changed yet.',
  current: '{role} (current)',
  updated: '{name} updated',
  scopeChanged: 'scope changed',
  nowRole: 'now {role}',
  ssoGroup: "{name}'s role comes from an SSO group mapping. A change here holds until the next sign-in applies the mapping again; change the mapping to make it last.",
  allScope: 'All lists and sender domains',

  lastOwnerEyebrow: 'Dialog · change role · blocked',
  lastOwnerTitle: '{name} is the last Owner',
  onlyOwnerTransfers: 'Only the Owner can transfer ownership.',
  transfer: 'Transfer ownership',
  cantDemote: "Can't demote",
  keepsOwner: "A workspace always keeps at least one Owner, so this role can't be changed to Admin and the member can't be removed or deactivated.",
  transferFirst: 'To change it, first transfer ownership to another member with two-factor enabled. {name} then becomes an Admin automatically.',
  transferFirstSuggest:
    'To change it, first transfer ownership to another member with two-factor enabled, for example {suggest}. {name} then becomes an Admin automatically.',
})
