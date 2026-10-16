import { defineMessages } from '@/i18n'

/**
 * Why a control is off for want of a permission (principle 7): "Sending
 * needs the campaigns:send permission. Ask Marcus Chen or Linh Nguyen."
 * The actions read as the server's (internal/api/messages), so a disabled
 * button and the problem behind it say the same thing.
 */
export const m = defineMessages('access.denied', {
  denied: '{action} needs the {permission} permission.',
  deniedAsk: '{action} needs the {permission} permission. Ask {helpers}.',
  this: 'This',
  'campaigns:send': 'Sending',
  'campaigns:pause': 'Pausing a send',
  'campaigns:approve': 'Approving',
  'campaigns:edit': 'Editing campaigns',
  'subscribers:export': 'Exporting contacts',
  'subscribers:import': 'Importing contacts',
  'subscribers:delete': 'Erasing personal data',
  'providers:manage': 'Managing providers',
  'domains:manage': 'Managing domains',
  'team:invite': 'Inviting members',
  'team:manage': 'Managing the team',
  'segments:sql': 'Writing SQL segments',
  'apikeys:manage': 'Managing API keys',
  'mcp:manage': 'Managing MCP agents',
  'automations:publish': 'Publishing automations',
  'workspace:manage': 'Changing workspace settings',
  'webhooks:manage': 'Managing webhooks',
  'integrations:hostrepute': 'Connecting HostRepute',
  'forms:edit': 'Editing forms',
  'reports:share': 'Sharing reports',
})
