import { defineMessages } from '@/i18n'

/**
 * The channel model's shared words (design expansion phase 8): the channel
 * menu of Sending providers, and the channels' names wherever a screen
 * names one (campaigns, consent per channel, approvals).
 */
export const m = defineMessages('channels', {
  switchLabel: 'Channel',
  /** The channel menu's button; {channel} is the section's channel ("Email"). */
  menuButton: 'Channel: {channel}. Change channel',
  email: 'Email',
  sms: 'SMS',
  push: 'Push',
  whatsapp: 'WhatsApp',
  smsTermsTab: 'Text message terms',
  all: 'All',
  sendsOn: 'Sends {channels}',
  noChannelCampaigns: '{channel, select, sms {No text campaigns yet} push {No push campaigns yet} whatsapp {No WhatsApp campaigns yet} other {No email campaigns yet}}',
})
