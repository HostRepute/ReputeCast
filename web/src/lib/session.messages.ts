import { defineMessages } from '@/i18n'

/**
 * Why a control is off for want of a permission (principle 7): "Sending
 * needs a permission you don't have. Ask Marcus Chen or Linh Nguyen."
 * People read who can help, never the permission key (review X-1). The
 * words are the server's (internal/authz/messages), so a disabled button
 * and the problem behind it say the same thing.
 */
export const m = defineMessages('access.denied', {
  denied: "{action} needs a permission you don't have.",
  deniedAsk: "{action} needs a permission you don't have. Ask {helpers}.",
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
  'subscribers:create': 'Adding contacts',
  'subscribers:edit': 'Editing contacts',
  'lists:manage': 'Managing lists',
  'segments:edit': 'Editing segments',
  'automations:edit': 'Editing automations',
  'scoring:edit': 'Editing scoring',
  'transactional:send': 'Sending through the API or SMTP',
  'transactional:publish': 'Publishing transactional templates',
  'sms:send': 'Sending SMS campaigns',
  'push:send': 'Sending push campaigns',
  'whatsapp:send': 'Sending WhatsApp campaigns',
  'reports:view': 'Viewing reports',
  'products:publish': 'Publishing products',
  'orders:manage': 'Managing orders',
  'grow:edit': 'Editing the referral program',
  'grow:connect': 'Managing partner links',
})
