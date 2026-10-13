import { useLayoutEffect, useRef, type ReactNode } from 'react'
import { GitHubMark, GoogleMark, Icon, MicrosoftMark } from '@/components/icons/Icon'
import './auth.css'

/** Title and caption of an auth card. */
export function AuthHeading({ id, title, sub }: { id: string; title: ReactNode; sub?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <h1 id={id}>{title}</h1>
      {sub ? <span className="k">{sub}</span> : null}
    </div>
  )
}

/** "linh@acme.io · not you?" */
export function NotYou({ email, onBack }: { email: ReactNode; onBack: () => void }) {
  return (
    <>
      {email} ·{' '}
      <button type="button" className="prose-link cursor-pointer border-0 bg-transparent p-0 text-[12px] font-medium text-accent hover:text-accent-hover" onClick={onBack}>
        not you?
      </button>
    </>
  )
}

/** A text link styled control inside running text (a button that looks like the boards' inline links). */
export function InlineAction({ children, onClick }: { children: ReactNode; onClick: () => void }) {
  return (
    <button type="button" className="prose-link inline cursor-pointer border-0 bg-transparent p-0 font-[inherit] text-[length:inherit] text-accent hover:text-accent-hover" onClick={onClick}>
      {children}
    </button>
  )
}

export type SocialProvider = 'google' | 'microsoft' | 'github'

const socialLabel: Record<SocialProvider, string> = { google: 'Google', microsoft: 'Microsoft', github: 'GitHub' }

export function SocialMark({ provider }: { provider: string }) {
  switch (provider) {
    case 'google':
      return <GoogleMark />
    case 'microsoft':
      return <MicrosoftMark />
    case 'github':
      return <GitHubMark />
  }
  return <Icon name="key" />
}

/** Google · Microsoft · GitHub buttons; each starts that provider's sign-in. */
export function SocialButtons({ providers, href }: { providers: string[]; href: (p: SocialProvider) => string }) {
  const known = providers.filter((p): p is SocialProvider => p in socialLabel)
  if (!known.length) return null
  return (
    <div className="soc">
      {known.map((p) => (
        <a key={p} className="btn" href={href(p)}>
          <SocialMark provider={p} />
          {socialLabel[p]}
        </a>
      ))}
    </div>
  )
}

/** The lock-shaped passkey glyph from the boards. */
export function PasskeyGlyph({ size = 16, strokeWidth = 1.8, color }: { size?: number; strokeWidth?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke={color ?? 'currentColor'} strokeWidth={strokeWidth} aria-hidden="true" focusable="false">
      <rect x="2" y="7" width="12" height="7" rx="1.5" />
      <path d="M5 7V5a3 3 0 0 1 6 0v2" />
    </svg>
  )
}

/**
 * Six one-digit boxes for a TOTP code. Pasting or autofilling the whole
 * code spreads it over the boxes; arrows and backspace move between them.
 */
export function OtpInput({
  value,
  onChange,
  invalid,
  describedBy,
  labelledBy,
  autoFocus = true,
}: {
  value: string
  onChange: (v: string) => void
  invalid?: boolean
  describedBy?: string
  labelledBy: string
  autoFocus?: boolean
}) {
  const refs = useRef<(HTMLInputElement | null)[]>([])
  // Typing fast fires several events before React re-renders, so keep the
  // latest value in a ref instead of reading it from the render closure.
  const latest = useRef(value)
  useLayoutEffect(() => {
    latest.current = value
  }, [value])
  const emit = (v: string) => {
    latest.current = v
    onChange(v)
  }
  const digits = value.padEnd(6, ' ').slice(0, 6).split('')
  const set = (i: number, d: string) => {
    const next = latest.current.padEnd(6, ' ').slice(0, 6).split('')
    next[i] = d || ' '
    emit(next.join('').replace(/\s+$/, ''))
  }
  return (
    <div className="otp" role="group" aria-labelledby={labelledBy}>
      {digits.map((d, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el
          }}
          value={d.trim()}
          inputMode="numeric"
          autoComplete={i === 0 ? 'one-time-code' : 'off'}
          aria-label={`Digit ${i + 1}`}
          aria-invalid={invalid ? true : undefined}
          aria-describedby={describedBy}
          maxLength={6}
          autoFocus={autoFocus && i === 0}
          onFocus={(e) => e.currentTarget.select()}
          onPaste={(e) => {
            const text = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, 6)
            if (text) {
              e.preventDefault()
              emit(text)
              refs.current[Math.min(text.length, 5)]?.focus()
            }
          }}
          onKeyDown={(e) => {
            if (e.key === 'Backspace' && !e.currentTarget.value && i > 0) refs.current[i - 1]?.focus()
            if (e.key === 'ArrowLeft' && i > 0) refs.current[i - 1]?.focus()
            if (e.key === 'ArrowRight' && i < 5) refs.current[i + 1]?.focus()
          }}
          onChange={(e) => {
            const v = e.target.value.replace(/\D/g, '')
            if (v.length > 1) {
              // Autofill or typing over a filled box: spread the digits from here.
              const next = (latest.current.slice(0, i) + v).slice(0, 6)
              emit(next)
              refs.current[Math.min(next.length, 5)]?.focus()
              return
            }
            set(i, v)
            if (v && i < 5) refs.current[i + 1]?.focus()
          }}
        />
      ))}
    </div>
  )
}

/** The organization name for copy such as "Work for Acme?": the instance name without "ReputeCast". */
export function orgName(instanceName?: string) {
  const n = (instanceName ?? '').replace(/\s*ReputeCast\s*$/i, '').trim()
  return n || undefined
}
