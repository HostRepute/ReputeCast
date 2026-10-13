import { cn } from '@/lib/cn'

const palettes = {
  info: { bg: '#DBEAFE', fg: '#1E40AF' },
  ok: { bg: '#DCFCE7', fg: '#166534' },
  warn: { bg: '#FEF3C7', fg: '#92400E' },
  purple: { bg: '#EDE9FE', fg: '#5B21B6' },
  neutral: { bg: '#EEF0F3', fg: '#4B5563' },
  bad: { bg: '#FEE2E2', fg: '#991B1B' },
  accent: { bg: '#0F766E', fg: '#FFFFFF' },
  // Team boards: Marcus Chen's avatar, and muted rows (deactivated members, invitees, the system).
  pink: { bg: '#FCE7F3', fg: '#9D174D' },
  muted: { bg: '#EEF0F3', fg: '#5B6472' },
  // Kenji Watanabe on InstanceUsers.
  sky: { bg: '#E0F2FE', fg: '#075985' },
} as const

export type AvatarColor = keyof typeof palettes

export function initials(name: string): string {
  const parts = name.replace(/@.*/, '').split(/[\s._-]+/).filter(Boolean)
  const first = parts[0]?.[0] ?? '?'
  const last = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

/**
 * Round initials avatar; square when `shape="square"` (workspace switcher).
 * With `src` (a profile photo) the photo replaces the initials. The initials
 * are drawn by CSS (`data-initials`): they decorate, so they are no part of
 * the visible label of a button that holds the avatar (WCAG 2.5.3).
 */
export function Avatar({
  name,
  color = 'info',
  size = 28,
  shape = 'round',
  className,
  text,
  src,
  fontSize,
}: {
  name: string
  color?: AvatarColor | string
  size?: number
  shape?: 'round' | 'square'
  className?: string
  /** Shown instead of the initials ("AI" for an agent, "?" for an invitee). */
  text?: string
  src?: string
  fontSize?: number
}) {
  const p = palettes[(color in palettes ? color : 'info') as AvatarColor]
  const radius = shape === 'square' ? 6 : '50%'
  return (
    <span
      className={cn('av', className)}
      style={{ width: size, height: size, fontSize: fontSize ?? (size <= 24 ? 10 : 11), background: p.bg, color: p.fg, borderRadius: radius, overflow: src ? 'hidden' : undefined }}
      aria-hidden="true"
      data-initials={src ? undefined : (text ?? initials(name))}
    >
      {src ? <img src={src} alt="" width={size} height={size} style={{ width: size, height: size, objectFit: 'cover', borderRadius: radius }} /> : null}
    </span>
  )
}
