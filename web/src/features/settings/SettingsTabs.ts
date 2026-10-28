import type { NavTab } from '@/components/ui/Tabs'
import { t } from '@/i18n'
import { mk } from '@/features/brandkit/messages'
import { m } from './messages'

/**
 * Workspace settings sections, in the order of the design (A5 §6: Settings
 * splits into Workspace settings and the Instance admin area). Every
 * settings screen passes these to <Page tabs={settingsTabs(slug)}>.
 */
export function settingsTabs(workspace: string): NavTab[] {
  const params = { workspace }
  return [
    { label: t(m.general), to: '/w/$workspace/settings', params, exact: true },
    // The brand kit (review 2026-10, CO-11; contract C10).
    { label: t(mk.tab), to: '/w/$workspace/settings/brand-kit', params },
    { label: t(m.senders), to: '/w/$workspace/settings/senders', params },
    { label: t(m.providers), to: '/w/$workspace/settings/providers', params },
    { label: t(m.sendingRules), to: '/w/$workspace/settings/sending-rules', params },
    { label: t(m.team), to: '/w/$workspace/settings/team', params },
    { label: t(m.roles), to: '/w/$workspace/settings/roles', params },
    { label: t(m.approvals), to: '/w/$workspace/settings/approvals', params },
    { label: t(m.keys), to: '/w/$workspace/settings/keys', params },
    { label: t(m.webhooks), to: '/w/$workspace/settings/webhooks', params },
    { label: t(m.integrations), to: '/w/$workspace/settings/integrations', params },
    { label: t(m.audit), to: '/w/$workspace/settings/audit', params },
  ]
}

/** Instance admin sections (only visible to instance admins). */
export function adminTabs(workspace: string): NavTab[] {
  const params = { workspace }
  return [
    { label: t(m.users), to: '/w/$workspace/admin/users', params },
    { label: t(m.workspaces), to: '/w/$workspace/admin/workspaces', params },
    { label: t(m.authentication), to: '/w/$workspace/admin/authentication', params },
    { label: t(m.sso), to: '/w/$workspace/admin/sso', params },
    { label: t(m.instance), to: '/w/$workspace/admin/instance', params },
    // The bot filter's networks and user agents (review AN-2).
    { label: t(m.botFilter), to: '/w/$workspace/admin/bot-filter', params },
    { label: t(m.instanceAudit), to: '/w/$workspace/admin/audit', params },
  ]
}
