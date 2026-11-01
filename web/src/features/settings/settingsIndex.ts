import { m as cm } from '@/features/automations/complete.messages'
import { mp as ap } from '@/features/approvals/policy.messages'
import { mk } from '@/features/brandkit/messages'
import { m as km } from '@/features/keys/messages'
import { m as sm } from '@/features/senders/messages'
import { m as rm } from '@/features/sending-rules/messages'
import { m as tm } from '@/features/team/messages'
import { m as hm } from '@/features/webhooks/messages'
import { m as wm } from '@/features/workspace/messages'
import type { MessageDescriptor } from '@/i18n'
import { m } from './messages'

/** A settings page, as settingsTabs names it. */
export type SettingsPath =
  | '/w/$workspace/settings'
  | '/w/$workspace/settings/brand-kit'
  | '/w/$workspace/settings/senders'
  | '/w/$workspace/settings/sending-rules'
  | '/w/$workspace/settings/team'
  | '/w/$workspace/settings/approvals'
  | '/w/$workspace/settings/keys'
  | '/w/$workspace/settings/webhooks'

/** One setting the command palette finds by its own words, and its page. */
export interface SettingRow {
  label: MessageDescriptor
  to: SettingsPath
  /** The page's tab name, for "Settings › Sending rules". */
  tab: MessageDescriptor
}

/**
 * The single settings the command palette finds (review 2026-11 FE-10):
 * each row's label is the words its page shows, so it reads the same in
 * every language, and it opens the page that holds it.
 */
export const settingRows: SettingRow[] = [
  { label: wm.nameLabel, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.timezoneLabel, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.senderLabel, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.retentionTitle, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.unconfirmedLabel, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.detailLabel, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.transferTitle, to: '/w/$workspace/settings', tab: m.general },
  { label: wm.deleteTitle, to: '/w/$workspace/settings', tab: m.general },
  { label: mk.headingFont, to: '/w/$workspace/settings/brand-kit', tab: mk.tab },
  { label: mk.importTitle, to: '/w/$workspace/settings/brand-kit', tab: mk.tab },
  { label: sm.addSender, to: '/w/$workspace/settings/senders', tab: m.senders },
  { label: rm.rulesTitle, to: '/w/$workspace/settings/sending-rules', tab: m.sendingRules },
  { label: rm.addRule, to: '/w/$workspace/settings/sending-rules', tab: m.sendingRules },
  { label: rm.defaultsTitle, to: '/w/$workspace/settings/sending-rules', tab: m.sendingRules },
  { label: rm.typedLabel, to: '/w/$workspace/settings/sending-rules', tab: m.sendingRules },
  { label: cm.capTitle, to: '/w/$workspace/settings/sending-rules', tab: m.sendingRules },
  { label: tm.inviteMember, to: '/w/$workspace/settings/team', tab: m.team },
  { label: ap.whenTitle, to: '/w/$workspace/settings/approvals', tab: m.approvals },
  { label: ap.approversTitle, to: '/w/$workspace/settings/approvals', tab: m.approvals },
  { label: ap.automationTitle, to: '/w/$workspace/settings/approvals', tab: m.approvals },
  { label: ap.bypassTitle, to: '/w/$workspace/settings/approvals', tab: m.approvals },
  { label: km.createTitleKey, to: '/w/$workspace/settings/keys', tab: m.keys },
  { label: km.createTitleAgent, to: '/w/$workspace/settings/keys', tab: m.keys },
  { label: hm.addEndpoint, to: '/w/$workspace/settings/webhooks', tab: m.webhooks },
  { label: hm.verifyTitle, to: '/w/$workspace/settings/webhooks', tab: m.webhooks },
  { label: hm.logTitle, to: '/w/$workspace/settings/webhooks', tab: m.webhooks },
]
