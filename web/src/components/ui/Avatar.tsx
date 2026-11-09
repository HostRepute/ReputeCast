import { cn } from '@/lib/cn'

const palettes = {
  info: { bg: 'var(--info-bg)', fg: 'var(--info-fg)' },
  ok: { bg: 'var(--ok-bg)', fg: 'var(--ok-fg)' },
  warn: { bg: 'var(--warn-bg)', fg: 'var(--warn-fg)' },
  purple: { bg: 'var(--purple-bg)', fg: 'var(--purple-fg)' },
  neutral: { bg: 'var(--line-2)', fg: 'var(--ink-2)' },
  bad: { bg: 'var(--bad-bg)', fg: 'var(--bad-fg)' },
  accent: { bg: 'var(--accent)', fg: 'var(--on-strong)' },
  // Team boards: Marcus Chen's avatar, and muted rows (deactivated members, invitees, the system).
  pink: { bg: '#FCE7F3', fg: '#9D174D' },
  muted: { bg: 'var(--line-2)', fg: 'var(--ink-3)' },
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
 * Round initials avatar, or a photo with `src`. The initials are drawn by CSS
 * (`data-initials`) so they never become part of the visible label of a
 * button that holds the avatar (WCAG 2.5.3).
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
