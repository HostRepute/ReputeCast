import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { GitHubMark, GoogleMark, MicrosoftMark } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, SensitiveFlag, type Tone } from '@/components/ui/Badge'
import { cn } from '@/lib/cn'

/** The avatar + name + mono email cell of the team and keys tables. */
export function PersonCell({
  name,
  sub,
  color,
  text,
  muted,
}: {
  name: ReactNode
  sub?: ReactNode
  color?: string
  /** Avatar text override ("?" for invitees, "AI" for agents). */
  text?: string
  muted?: boolean
}) {
  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <Avatar name={typeof name === 'string' ? name : '?'} color={muted ? 'muted' : color} text={text} />
      <div className="flex min-w-0 flex-col leading-[1.3]">
        <span className="font-semibold">{name}</span>
        {sub ? <span className="k mono">{sub}</span> : null}
      </div>
    </div>
  )
}

/** Role badge colours on the boards: Owner purple, other built-ins grey, custom roles blue. */
export function roleTone(role: { key: string; builtin: boolean }): Tone {
  if (role.builtin && role.key === 'owner') return 'purple'
  return role.builtin ? 'neutral' : 'info'
}

export function RoleBadge({ role }: { role: { key: string; name: string; builtin: boolean } }) {
  return <Badge tone={roleTone(role)}>{role.name}</Badge>
}

/** Sign-in methods: "Password · passkey" or the provider mark and "Google Workspace SSO". */
export function SignInText({ methods }: { methods: Schemas['MemberSignInMethod'][] }) {
  if (methods.length === 0) return <span className="k">–</span>
  return (
    <>
      {methods.map((m, i) => (
        <span key={i}>
          {i > 0 ? ' · ' : null}
          {m.kind === 'sso' || m.kind === 'google' || m.kind === 'microsoft' || m.kind === 'github' ? (
            <>
              <ProviderMark preset={m.preset ?? m.kind} /> {m.label}
            </>
          ) : (
            m.label
          )}
        </span>
      ))}
    </>
  )
}

/** The IdP mark drawn inline before a sign-in method (14px on the boards). */
export function ProviderMark({ preset, size = 14 }: { preset: string; size?: number }) {
  switch (preset) {
    case 'google':
      return <GoogleMark size={size} />
    case 'entra':
    case 'microsoft':
      return <MicrosoftMark size={size} />
    case 'github':
      return <GitHubMark size={size} />
  }
  return null
}

/** Two-factor badge: On · TOTP, Via IdP, Required · off with the grace period, Off. */
export function TwoFABadge({ twoFA }: { twoFA: Schemas['MemberTwoFA'] }) {
  switch (twoFA.state) {
    case 'on':
      return <Badge tone="ok">On · {twoFA.method === 'passkey' ? 'passkey' : 'TOTP'}</Badge>
    case 'idp':
      return <Badge tone="ok">Via IdP</Badge>
    case 'required':
      return (
        <span className="col gap-0.5">
          <Badge tone="warn">Required · off</Badge>
          {twoFA.graceDays !== undefined ? (
            <span className="k">{twoFA.graceDays === 0 ? 'grace ends today' : `grace ends in ${twoFA.graceDays} ${twoFA.graceDays === 1 ? 'day' : 'days'}`}</span>
          ) : (
            <span className="k">grace period over</span>
          )}
        </span>
      )
  }
  return <Badge tone="neutral">Off</Badge>
}

/** A permission key in mono with its ⚑ flag when sensitive. */
export function PermissionKey({ k, sensitive, className }: { k: string; sensitive?: boolean; className?: string }) {
  return (
    <span className={cn('mono', className)}>
      {k}
      {sensitive ? (
        <>
          {' '}
          <SensitiveFlag />
        </>
      ) : null}
    </span>
  )
}

/** Copies text and reports whether it worked. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
