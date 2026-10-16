import { defineMessages } from '@/i18n'

/** The built-in roles' names (lib/roles.ts). A custom role keeps the name the workspace gave it. */
export const m = defineMessages('roleNames', {
  owner: 'Owner',
  admin: 'Admin',
  editor: 'Editor',
  viewer: 'Viewer',
})
