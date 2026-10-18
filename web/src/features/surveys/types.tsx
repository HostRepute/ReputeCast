import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { m } from './messages'

type QType = Schemas['SurveyQuestion']['type']

/** The six question types, with their icons. */
export const TYPES: { type: QType; label: keyof typeof m; icon: ReactNode }[] = [
  { type: 'single', label: 'typeSingle', icon: <circle cx="8" cy="8" r="5.5" /> },
  { type: 'multi', label: 'typeMulti', icon: <rect x="2.5" y="2.5" width="11" height="11" rx="2" /> },
  { type: 'scale', label: 'typeScale', icon: <rect x="1.5" y="5" width="13" height="6" rx="2" /> },
  { type: 'emoji', label: 'typeEmoji', icon: <circle cx="8" cy="8" r="6" /> },
  { type: 'short', label: 'typeShort', icon: <path d="M3 6h10M3 10h7" /> },
  { type: 'long', label: 'typeLong', icon: <path d="M3 4h10M3 8h10M3 12h7" /> },
]

export function typeLabel(t: ReturnType<typeof useT>, type: string): string {
  const x = TYPES.find((y) => y.type === type)
  return x ? t(m[x.label]) : type
}

export function TypeIcon({ type }: { type: QType }) {
  const x = TYPES.find((y) => y.type === type)
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true" focusable="false">
      {x?.icon}
    </svg>
  )
}

