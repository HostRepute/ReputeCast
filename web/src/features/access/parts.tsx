import type { ComponentProps, ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { GitHubMark, GoogleMark, MicrosoftMark } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, SensitiveFlag, type Tone } from '@/components/ui/Badge'
import { Table } from '@/components/ui/Table'
import { cn } from '@/lib/cn'

/**
 * The avatar + name + mono email cell of the team and keys tables. On phones
 * the name drops under the avatar when the column is narrow, as on the boards.
 */
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
    <div className="flex min-w-0 items-center gap-2.5 max-md:flex-wrap">
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

/**
 * The IdP mark drawn inline before a sign-in method (14px on the boards).
 * The inline-block wrapper puts it on the text baseline, as an inline SVG
 * sits on the boards (the reset makes a bare SVG a block).
 */
export function ProviderMark({ preset, size = 14 }: { preset: string; size?: number }) {
  let mark: ReactNode = null
  switch (preset) {
    case 'google':
      mark = <GoogleMark size={size} />
      break
    case 'entra':
    case 'microsoft':
      mark = <MicrosoftMark size={size} />
      break
    case 'github':
      mark = <GitHubMark size={size} />
      break
  }
  return mark ? <span className="inline-block">{mark}</span> : null
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

/**
 * The table of the team and access boards (Team, member, Roles, Keys, Audit):
 * 11px cells that wrap, the first column at least 180px, and button rows
 * kept on one line.
 */
export function AccessTable({ className, ...rest }: ComponentProps<typeof Table>) {
  return <Table density="compact" className={cn('[&_td]:whitespace-normal [&_td_.row]:flex-nowrap', className)} {...rest} />
}

/** A permission-group heading row of the permission tables (grey band, uppercase). */
export function GroupRow({ label, cells }: { label: string; cells: number }) {
  return (
    <tr className="bg-ground-2">
      <td>
        <span className="text-[12px] font-bold uppercase tracking-[.04em] text-ink-2">{label}</span>
      </td>
      {Array.from({ length: cells }, (_, i) => (
        <td key={i} className="c" />
      ))}
    </tr>
  )
}

/** ✓ or – in the compare-roles matrix, with words for screen readers. */
export function HasMark({ on }: { on: boolean }) {
  return on ? (
    <>
      <span className="font-bold text-accent" aria-hidden="true">
        ✓
      </span>
      <span className="vh">Granted</span>
    </>
  ) : (
    <>
      <span className="k" aria-hidden="true">
        –
      </span>
      <span className="vh">Not granted</span>
    </>
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
