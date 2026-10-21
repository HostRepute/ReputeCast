import { defineMessages } from '@/i18n'

/** Push and WhatsApp campaigns in the campaigns list and New campaign (design expansion phase 8, X22). */
export const m = defineMessages('channelCampaigns', {
  newPush: 'Push notification',
  newPushBody: "A notification to readers' browsers on your push sites.",
  newPushHint: 'Title, message and link, with a preview per platform.',
  newWhatsApp: 'WhatsApp message',
  newWhatsAppBody: 'An approved template to people with WhatsApp consent.',
  newWhatsAppHint: 'Variables from contact fields; the cost shows before you send.',
  untitledPush: 'Untitled push campaign',
  untitledWhatsApp: 'Untitled WhatsApp campaign',
  noOpens: 'Notifications and WhatsApp messages have no opens',
  shownTitle: 'Shown, as the service worker reports it',
  local: "{date} · {time} in each recipient's time zone",
  notSite: 'No site',
  siteSubscribers: 'Site subscribers',
  duplicated: 'Duplicated as “{name}”',
  duplicateFailed: "Couldn't duplicate",
})
