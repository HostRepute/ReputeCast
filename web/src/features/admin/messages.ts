import { defineMessages } from '@/i18n'

/** The Instance admin frame (AdminPage): its title, tabs and the 403 state for everyone else. */
export const m = defineMessages('admin', {
  title: 'Instance admin',
  documentTitle: '{page} · Instance admin',
  adminsOnly: 'admins only',
  workspaceView: 'Workspace view',
  workspaceSettings: 'Workspace settings',
  tabsLabel: 'Instance admin sections',

  notAdminBadge: '403 · instance admins only',
  notAdminTitle: 'Only instance admins can open Instance admin',
  backToOverview: 'Back to overview',
  account: 'Account & security',
  notAdminFootnote: 'Ask an instance admin for changes to sign-in, SSO or workspaces.',
  notAdminBody:
    'Instance admin manages users, sign-in methods, SSO, security policies and workspaces for the whole instance. Your own sign-in methods are in Account & security.',
})
