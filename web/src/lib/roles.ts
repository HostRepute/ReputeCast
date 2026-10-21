import { t } from '@/i18n'
import { m } from './roles.messages'

const builtin = { owner: m.owner, admin: m.admin, editor: m.editor, viewer: m.viewer, client_approver: m.clientApprover, client_viewer: m.clientViewer } as const

/** The English names the server stores for the built-in roles (internal/authz). */
const english: Record<string, string> = {
  owner: 'Owner',
  admin: 'Admin',
  editor: 'Editor',
  viewer: 'Viewer',
  client_approver: 'Client approver',
  client_viewer: 'Client viewer',
}

/**
 * A role's name in the app language: the built-in roles (Owner, Admin,
 * Editor, Viewer) are translated; a custom role, or a built-in one stored
 * under another name, keeps its name ("Content Writer"). Pass the role's
 * key when the data has it, else only its name.
 */
export function roleName(role: { key?: string; name: string; builtin?: boolean }): string {
  const key = role.key ?? Object.keys(english).find((k) => english[k] === role.name)
  const d = key ? builtin[key as keyof typeof builtin] : undefined
  if (!d || role.builtin === false || (role.name && role.name !== english[key!])) return role.name
  return t(d)
}
