import type { Schemas } from '@/api/client'
import type { MessageDescriptor } from '@/i18n'
import { m } from './messages'

type Permission = Pick<Schemas['PermissionInfo'], 'key' | 'peopleOnly'>

/** Permissions with a line of their own on the boards (Roles, RoleEditor). */
const ownLines: Record<string, { editor: MessageDescriptor; matrix: MessageDescriptor }> = {
  'ai:use': { editor: m.aiUseEditor, matrix: m.aiUseMatrix },
}

/**
 * Notes under a group's rows in the role editor, by a permission of the
 * group: the server names groups in the reader's language.
 */
const groupNotes: Record<string, MessageDescriptor> = {
  'earn:view': m.earnGroupNote,
}

/**
 * What the role editor says about a permission besides its label: its own
 * line under the key (ai:use), or, for the other people-only permissions,
 * that keys and agents can't hold it.
 */
export function editorNote(p: Permission): { line?: MessageDescriptor; aside?: MessageDescriptor } {
  const own = ownLines[p.key]
  if (own) return { line: own.editor }
  return p.peopleOnly ? { aside: m.peopleOnlyEditor } : {}
}

/** What the compare matrix adds after a permission: its own line, or "· people only". */
export function matrixNote(p: Permission): { line?: MessageDescriptor; suffix?: MessageDescriptor } {
  const own = ownLines[p.key]
  if (own) return { line: own.matrix }
  return p.peopleOnly ? { suffix: m.peopleOnlyMatrix } : {}
}

/** The note under a group whose rows are these permissions. */
export function groupNote(rows: Permission[]): MessageDescriptor | undefined {
  for (const p of rows) if (groupNotes[p.key]) return groupNotes[p.key]
  return undefined
}
