import type { useT } from '@/i18n'
import { m as lm } from './library.messages'

/** A library item's kind in words: "Saved section", "Campaign template". */
export function kindLabel(t: ReturnType<typeof useT>, kind: string) {
  switch (kind) {
    case 'section':
      return t(lm.kindSection)
    case 'template':
      return t(lm.kindTemplate)
    case 'automation':
      return t(lm.kindAutomation)
    case 'sms':
      return t(lm.kindSms)
    default:
      return t(lm.kindTransactional)
  }
}

const parts: Record<string, typeof lm.partLists> = {
  lists: lm.partLists,
  forms: lm.partForms,
  automations: lm.partAutomations,
  branding: lm.partBranding,
  senders: lm.partSenders,
  roles: lm.partRoles,
  policy: lm.partPolicy,
}

/** A part of a workspace template that applying it left out: "automations", "role:Content Writer". */
export function partName(t: ReturnType<typeof useT>, part: string) {
  if (part.startsWith('role:')) return t(lm.partRole, { name: part.slice('role:'.length) })
  return parts[part] ? t(parts[part]) : part
}
