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
