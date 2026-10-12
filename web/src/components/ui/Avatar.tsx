import { cn } from '@/lib/cn'

const palettes = {
  info: { bg: '#DBEAFE', fg: '#1E40AF' },
  ok: { bg: '#DCFCE7', fg: '#166534' },
  warn: { bg: '#FEF3C7', fg: '#92400E' },
  purple: { bg: '#EDE9FE', fg: '#5B21B6' },
  neutral: { bg: '#EEF0F3', fg: '#4B5563' },
  bad: { bg: '#FEE2E2', fg: '#991B1B' },
  accent: { bg: '#0F766E', fg: '#FFFFFF' },
} as const

export type AvatarColor = keyof typeof palettes

export function initials(name: string): string {
  const parts = name.replace(/@.*/, '').split(/[\s._-]+/).filter(Boolean)
  const first = parts[0]?.[0] ?? '?'
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

/** Round initials avatar; square when `shape="square"` (workspace switcher). */
export function Avatar({
  name,
  color = 'info',
  size = 28,
  shape = 'round',
  className,
}: {
  name: string
  color?: AvatarColor | string
  size?: number
  shape?: 'round' | 'square'
  className?: string
}) {
  const p = palettes[(color in palettes ? color : 'info') as AvatarColor]
  return (
    <span
      className={cn('av', className)}
      style={{ width: size, height: size, fontSize: size <= 24 ? 10 : 11, background: p.bg, color: p.fg, borderRadius: shape === 'square' ? 6 : '50%' }}
      aria-hidden="true"
    >
      {initials(name)}
    </span>
  )
}
