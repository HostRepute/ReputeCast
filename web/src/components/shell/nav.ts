import type { MessageDescriptor } from '@/i18n'
import type { IconName } from '../icons/Icon'
import { m } from './messages'

/** The workspace's top-level sections, in sidebar order. */
export type SectionKey =
  | 'overview'
  | 'campaigns'
  | 'automations'
  | 'transactional'
  | 'audience'
  | 'grow'
  | 'earn'
  | 'forms'
  | 'deliverability'
  | 'providers'
  | 'settings'

/** Sections a workspace can hide (Settings › General › Sidebar sections). */
export type HideableSection = 'transactional' | 'grow' | 'earn'

export const hideableSections: readonly HideableSection[] = ['transactional', 'grow', 'earn']

export interface NavItem {
  key: SectionKey
  to: string
  label: MessageDescriptor
  icon: IconName
  /** The overview is the workspace root, so it only matches exactly. */
  exact?: boolean
  /** Other screens the section stands for: approvals live under Campaigns, Instance admin under Settings. */
  also?: string[]
  /** One of the phone tab bar's four sections; the others are in the More sheet. */
  tab?: boolean
  hideable?: boolean
}

export const NAV: readonly NavItem[] = [
  { key: 'overview', to: '/w/$workspace', label: m.navOverview, icon: 'overview', exact: true, tab: true },
  { key: 'campaigns', to: '/w/$workspace/campaigns', label: m.navCampaigns, icon: 'campaigns', also: ['/w/$workspace/approvals'], tab: true },
  { key: 'automations', to: '/w/$workspace/automations', label: m.navAutomations, icon: 'automations' },
  { key: 'transactional', to: '/w/$workspace/transactional', label: m.navTransactional, icon: 'transactional', hideable: true },
  { key: 'audience', to: '/w/$workspace/audience', label: m.navAudience, icon: 'audience', tab: true },
  { key: 'grow', to: '/w/$workspace/grow', label: m.navGrow, icon: 'grow', hideable: true },
  { key: 'earn', to: '/w/$workspace/earn', label: m.navEarn, icon: 'earn', hideable: true },
  { key: 'forms', to: '/w/$workspace/forms', label: m.navForms, icon: 'forms' },
  { key: 'deliverability', to: '/w/$workspace/deliverability', label: m.navDeliverability, icon: 'deliverability', tab: true },
  { key: 'providers', to: '/w/$workspace/providers', label: m.navProviders, icon: 'providers' },
  { key: 'settings', to: '/w/$workspace/settings', label: m.navSettings, icon: 'settings', also: ['/w/$workspace/admin'] },
]

/**
 * The sections the sidebar, the rail, the tab bar and the More sheet show:
 * every section except the ones the workspace hides. A hidden section's
 * screens still open by link.
 */
export function visibleNav(hidden: readonly string[] = []): NavItem[] {
  return NAV.filter((item) => !(item.hideable && hidden.includes(item.key)))
}

/** Splits the visible sections between the phone tab bar and the More sheet. */
export function phoneNav(hidden: readonly string[] = []): { tabs: NavItem[]; more: NavItem[] } {
  const items = visibleNav(hidden)
  return { tabs: items.filter((i) => i.tab), more: items.filter((i) => !i.tab) }
}

/**
 * The section a path belongs to ("/w/acme/settings/team" → settings), or
 * undefined for screens outside the sections (account, notifications).
 */
export function sectionOf(pathname: string): SectionKey | undefined {
  const parts = pathname.split('/').filter(Boolean)
  if (parts[0] !== 'w' || !parts[1]) return undefined
  const first = parts[2]
  if (first === undefined) return 'overview'
  for (const item of NAV) {
    if (item.key === 'overview') continue
    const segments = [item.to, ...(item.also ?? [])].map((to) => to.split('/').pop())
    if (segments.includes(first)) return item.key
  }
  return undefined
}

/**
 * Whether the phone's More tab shows as current: while its sheet is open, or
 * when the current screen lives in the sheet (and its section is shown).
 */
export function moreIsCurrent(current: SectionKey | undefined, hidden: readonly string[] = [], sheetOpen = false): boolean {
  if (sheetOpen) return true
  return current !== undefined && phoneNav(hidden).more.some((i) => i.key === current)
}
