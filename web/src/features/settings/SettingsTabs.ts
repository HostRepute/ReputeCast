import type { NavTab } from '@/components/ui/Tabs'

/**
 * Workspace settings sections, in the order of the design (A5 §6: Settings
 * splits into Workspace settings and the Instance admin area). Every
 * settings screen passes these to <Page tabs={settingsTabs(slug)}>.
 */
export function settingsTabs(workspace: string): NavTab[] {
  const params = { workspace }
  return [
    { label: 'General', to: '/w/$workspace/settings', params, exact: true },
    { label: 'Senders', to: '/w/$workspace/settings/senders', params },
    { label: 'Providers', to: '/w/$workspace/settings/providers', params },
    { label: 'Team', to: '/w/$workspace/settings/team', params },
    { label: 'Roles', to: '/w/$workspace/settings/roles', params },
    { label: 'Approvals', to: '/w/$workspace/settings/approvals', params },
    { label: 'Keys & MCP', to: '/w/$workspace/settings/keys', params },
    { label: 'Webhooks', to: '/w/$workspace/settings/webhooks', params },
    { label: 'Integrations', to: '/w/$workspace/settings/integrations', params },
    { label: 'Audit log', to: '/w/$workspace/settings/audit', params },
  ]
}

/** Instance admin sections (only visible to instance admins). */
export function adminTabs(workspace: string): NavTab[] {
  const params = { workspace }
  return [
    { label: 'Users', to: '/w/$workspace/admin/users', params },
    { label: 'Workspaces', to: '/w/$workspace/admin/workspaces', params },
    { label: 'Authentication', to: '/w/$workspace/admin/authentication', params },
    { label: 'SSO providers', to: '/w/$workspace/admin/sso', params },
    { label: 'Instance & updates', to: '/w/$workspace/admin/instance', params },
    { label: 'Audit log', to: '/w/$workspace/admin/audit', params },
  ]
}
