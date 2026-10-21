import { defineMessages } from '@/i18n'

/**
 * The channel model's shared words (design expansion phase 8): the channel
 * switch of Sending providers, and the channels' names wherever a screen
 * names one (campaigns, consent per channel, approvals).
 */
export const m = defineMessages('channels', {
  switchLabel: 'Channel',
  email: 'Email',
  sms: 'SMS',
  push: 'Push',
  whatsapp: 'WhatsApp',
  addChannel: 'Add a channel',
  addChannelHint: 'Texts, push notifications or WhatsApp, next to email. The switch appears once a second channel is set up.',
  smsTermsTab: 'Text message terms',
  all: 'All',
  sendsOn: 'Sends {channels}',
  noChannelCampaigns: '{channel, select, sms {No text campaigns yet} push {No push campaigns yet} whatsapp {No WhatsApp campaigns yet} other {No email campaigns yet}}',
})
