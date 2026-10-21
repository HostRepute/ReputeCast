import { defineMessages } from '@/i18n'

/**
 * Text the route files show themselves, where no feature screen does: the
 * import wizard's entry routes while they find or open an import, and
 * Instance admin › Workspaces' document title. (The leading "-" keeps this
 * file out of the route tree.)
 */
export const mr = defineMessages('routes', {
  importTitle: 'Import contacts',
  importLooking: 'Looking for an import in progress',
  importOpenFailed: "Couldn't open this import",
  importOpening: 'Opening the import',
  adminWorkspaces: 'Workspaces',
})
