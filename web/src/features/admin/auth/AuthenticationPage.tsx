import { useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, ProviderMark } from '@/features/access/parts'
import { andList, dayPartWords } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { SocialSettingsDialog } from './SocialSettingsDialog'

type Policy = Schemas['InstancePolicy']
type Input = Schemas['InstancePolicyInput']

const cards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(460px,100%),1fr))] max-md:grid-cols-1'

/** Instance admin › Authentication (InstanceAuth). */
export function AuthenticationPage() {
  const q = $api.useQuery('get', '/api/v1/admin/policy')
  return (
    <AdminPage documentTitle="Authentication">
      {q.isError ? (
        <ErrorPanel title="Couldn't load the authentication policy" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed; sign-in keeps working with the saved policy.
        </ErrorPanel>
      ) : q.isPending ? (
        <PolicySkeleton />
      ) : (
        <PolicyForm key={q.data.updatedAt} p={q.data} />
      )}
    </AdminPage>
  )
}

function toInput(p: Policy): Input {
  const on = (k: string) => p.social.find((s) => s.kind === k)?.enabled ?? false
  return {
    methods: { ...p.methods },
    social: { google: on('google'), microsoft: on('microsoft'), github: on('github') },
    passwordMinLength: p.passwordMinLength,
    passwordReuse: p.passwordReuse as Input['passwordReuse'],
    breachedCheck: p.breachedCheck,
    twoFactor: p.twoFactor,
    graceDays: p.graceDays,
    sessionLifetimeHours: p.sessionLifetimeHours,
    sessionIdleMinutes: p.sessionIdleMinutes,
    lockoutAttempts: p.lockoutAttempts,
    lockoutMinutes: p.lockoutMinutes,
    jitDomains: [...p.jitDomains],
  }
}

function PolicyForm({ p }: { p: Policy }) {
  const qc = useQueryClient()
  const [v, setV] = useState<Input>(() => toInput(p))
  const [error, setError] = useState<{ field?: string; message: string }>()
  const [social, setSocial] = useState<Schemas['SSOSocialProvider']>()
  const save = $api.useMutation('put', '/api/v1/admin/policy')
  const dirty = JSON.stringify(v) !== JSON.stringify(toInput(p))
  const set = (patch: Partial<Input>) => setV((x) => ({ ...x, ...patch }))
  const submit = async () => {
    setError(undefined)
    try {
      await save.mutateAsync({ body: v })
      await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/policy'] }), qc.invalidateQueries({ queryKey: ['meta'] })])
      toast({ kind: 'success', title: 'Policies saved', detail: 'new limits apply at the next sign-in' })
    } catch (e) {
      const problem = (e as { problem?: { errors?: { location?: string }[] } }).problem
      setError({ field: problem?.errors?.[0]?.location, message: errorMessage(e) })
    }
  }
  return (
    <>
      <div className={cards}>
        <MethodsCard p={p} v={v} set={set} onSocial={setSocial} />
        <PasswordCard v={v} set={set} />
        <TwoFactorCard p={p} v={v} set={set} />
        <SessionsCard v={v} set={set} />
        <AllowlistCard p={p} v={v} set={set} />
      </div>
      <SSOProvidersCard p={p} />
      {error ? (
        <ErrorPanel title="The policies weren't saved">
          {error.message}
        </ErrorPanel>
      ) : null}
      <div className="row justify-end gap-2.5">
        <Button onClick={() => setV(toInput(p))}>
          Discard
        </Button>
        <Button variant="primary" loading={save.isPending} onClick={() => void submit()}>
          Save policies
        </Button>
      </div>
      {social ? <SocialSettingsDialog provider={social} onClose={() => setSocial(undefined)} /> : null}
      <UnsavedChangesGuard
        dirty={dirty}
        title="Leave without saving the policies?"
        description="You changed the authentication policy. Nothing applies until you save."
        saveLabel="Save and leave"
        onSave={submit}
      />
    </>
  )
}

/** A switch row with a border (Allowed sign-in methods). */
function MethodRow({
  checked,
  onChange,
  title,
  description,
  badge,
  mark,
  extra,
  disabled,
}: {
  checked: boolean
  onChange: (on: boolean) => void
  title: ReactNode
  description: ReactNode
  badge: ReactNode
  mark?: ReactNode
  extra?: ReactNode
  disabled?: boolean
}) {
  return (
    <div className="row flex-nowrap gap-3 rounded-lg border border-line px-3 py-2.5">
      {/* eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text */}
      <label className="row min-w-0 flex-1 flex-nowrap gap-3">
        <input type="checkbox" className="tg" role="switch" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
        <span className="col min-w-0 flex-1 gap-px">
          <span className="font-semibold">{title}</span>
          <span className="k whitespace-normal">{description}</span>
        </span>
      </label>
      {extra}
      {mark ? (
        <span className="row flex-nowrap gap-1.5">
          {mark}
          {badge}
        </span>
      ) : (
        badge
      )}
    </div>
  )
}

function MethodsCard({ p, v, set, onSocial }: { p: Policy; v: Input; set: (x: Partial<Input>) => void; onSocial: (s: Schemas['SSOSocialProvider']) => void }) {
  const social = (k: 'google' | 'microsoft' | 'github') => p.social.find((s) => s.kind === k)!
  const onBadge = (on: boolean, text = 'On') => <Badge tone={on ? 'ok' : 'neutral'}>{on ? text : 'Off'}</Badge>
  const enabledSSO = p.sso.filter((s) => s.status === 'enabled')
  const otherSSO = p.sso.filter((s) => s.status !== 'enabled')
  const ssoLine =
    p.sso.length === 0
      ? 'No provider yet: add one below'
      : [...enabledSSO.map((s) => `${s.name} enabled`), ...otherSSO.map((s) => `${s.name} ${s.status === 'draft' ? 'in progress' : s.status}`)].join(' · ')
  const socialRow = (k: 'google' | 'microsoft' | 'github', description: ReactNode, onText = 'On') => {
    const s = social(k)
    const on = v.social[k]
    return (
      <MethodRow
        checked={on}
        onChange={(x) => {
          if (x && !s.configured) {
            onSocial(s)
            return
          }
          set({ social: { ...v.social, [k]: x } })
        }}
        title={s.name}
        description={description}
        mark={<ProviderMark preset={k} />}
        badge={onBadge(on, onText)}
        extra={
          <Button size="sm" iconOnly icon="edit" aria-label={`${s.name} sign-in settings`} title={`${s.name} sign-in settings`} onClick={() => onSocial(s)} className="border-transparent" />
        }
      />
    )
  }
  const gh = social('github')
  return (
    <section className="card" aria-labelledby="methods-h">
      <CardSection title={<span id="methods-h">Allowed sign-in methods</span>} caption="Methods can be turned on separately. SAML and SCIM are not part of this version." />
      <div className="pad col gap-2.5">
        <MethodRow
          checked={v.methods.password}
          onChange={(x) => set({ methods: { ...v.methods, password: x } })}
          title="Email and password"
          description={`Minimum ${v.passwordMinLength} characters · breached-password check ${v.breachedCheck ? 'on' : 'off'}${v.methods.password ? '' : ' · break-glass Owners keep theirs'}`}
          badge={onBadge(v.methods.password)}
        />
        <MethodRow
          checked={v.methods.passkeys}
          onChange={(x) => set({ methods: { ...v.methods, passkeys: x } })}
          title="Passkeys"
          description="Single-step sign-in and second factor · WebAuthn"
          badge={onBadge(v.methods.passkeys)}
        />
        {socialRow('google', 'For invited people only · never creates accounts')}
        {socialRow('microsoft', 'Personal and work accounts')}
        {socialRow(
          'github',
          gh.org ? (
            <>
              Restricted to members of the GitHub organization <span className="mono">{gh.org}</span>
            </>
          ) : (
            'Any GitHub account of an invited person'
          ),
          gh.org ? 'On · org restricted' : 'On',
        )}
        <MethodRow
          checked={v.methods.sso}
          onChange={(x) => set({ methods: { ...v.methods, sso: x } })}
          title="Single sign-on (OpenID Connect)"
          description={ssoLine}
          badge={<Badge tone={v.methods.sso ? 'ok' : 'neutral'}>{v.methods.sso ? `On · ${p.sso.length} ${p.sso.length === 1 ? 'provider' : 'providers'}` : 'Off'}</Badge>}
        />
      </div>
    </section>
  )
}

function PasswordCard({ v, set }: { v: Input; set: (x: Partial<Input>) => void }) {
  return (
    <section className="card" aria-labelledby="pw-h">
      <CardSection title={<span id="pw-h">Password policy</span>} />
      <div className="pad col gap-3">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            Minimum length
            <input
              className="in max-w-[120px]"
              type="number"
              min={8}
              max={128}
              value={v.passwordMinLength}
              onChange={(e) => set({ passwordMinLength: Number(e.target.value) || 8 })}
              aria-describedby="pw-min-hint"
            />
            <span className="k font-medium" id="pw-min-hint">
              Default 12. Passphrases count.
            </span>
          </label>
          <label className="lbl">
            Re-use
            <select className="in" value={v.passwordReuse} onChange={(e) => set({ passwordReuse: Number(e.target.value) as Input['passwordReuse'] })}>
              <option value={5}>Block the last 5 passwords</option>
              <option value={10}>Block the last 10</option>
              <option value={0}>Allow re-use</option>
            </select>
          </label>
        </div>
        {/* eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text */}
        <label className="row flex-nowrap gap-3">
          <input type="checkbox" className="tg" role="switch" checked={v.breachedCheck} onChange={(e) => set({ breachedCheck: e.target.checked })} />
          <span className="col gap-px">
            <span className="font-semibold">Check new passwords against known breaches</span>
            <span className="k">Uses k-anonymity: only the first 5 characters of the hash leave the server. Turn off for offline installs.</span>
          </span>
        </label>
      </div>
    </section>
  )
}

const graceOptions = [7, 1, 3, 14, 0]

function TwoFactorCard({ p, v, set }: { p: Policy; v: Input; set: (x: Partial<Input>) => void }) {
  const options: { value: Input['twoFactor']; title: string; description: string }[] = [
    { value: 'everyone', title: 'Everyone', description: 'Every member of every workspace' },
    {
      value: 'sensitive',
      title: 'Members with a sensitive permission',
      description: 'Anyone holding a ⚑ permission (send, approve, publish, export, providers, team, keys, agents). SSO users with two-factor at the IdP count as covered.',
    },
    { value: 'off', title: 'Off', description: 'Recommended only for test instances' },
  ]
  const grace = graceOptions.includes(v.graceDays) ? graceOptions : [...graceOptions, v.graceDays]
  const inGrace = p.inGrace
  return (
    <section className="card" aria-labelledby="tfa-h">
      <CardSection title={<span id="tfa-h">Two-factor enforcement</span>} />
      <div className="pad col gap-2.5">
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="vh">Who must use two-factor</legend>
          {options.map((o) => (
            // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text
            <label key={o.value} className="row flex-nowrap gap-2.5 rounded-lg border border-line px-2.5 py-2">
              <input type="radio" name="tfa" checked={v.twoFactor === o.value} onChange={() => set({ twoFactor: o.value })} />
              <span className="col gap-px">
                <span className="font-semibold">{o.title}</span>
                <span className="k whitespace-normal">{o.description}</span>
              </span>
            </label>
          ))}
        </fieldset>
        <div className="row gap-3.5">
          <label className="lbl">
            Grace period
            <select className="in max-w-[160px]" value={v.graceDays} onChange={(e) => set({ graceDays: Number(e.target.value) })}>
              {grace.map((d) => (
                <option key={d} value={d}>
                  {d === 0 ? 'None' : `${d} ${d === 1 ? 'day' : 'days'}`}
                </option>
              ))}
            </select>
          </label>
          <span className="k">
            {inGrace.length === 0
              ? 'Nobody is in a grace period now'
              : `${inGrace.length} ${inGrace.length === 1 ? 'member' : 'members'} in grace now: ${andList(inGrace.map((g) => `${g.name}, ends ${untilDays(g.until)}`))}`}
          </span>
        </div>
      </div>
    </section>
  )
}

function untilDays(at: string) {
  const days = Math.ceil((new Date(at).getTime() - Date.now()) / 86_400_000)
  if (days <= 0) return 'today'
  return days === 1 ? 'in 1 day' : `in ${days} days`
}

const lifetimes = [14 * 24, 24, 7 * 24, 30 * 24]
const idles = [12 * 60, 60, 4 * 60, 24 * 60]
const locks = [15, 5, 60]

function durationLabel(hours: number) {
  if (hours % 24 === 0) {
    const d = hours / 24
    return `${d} ${d === 1 ? 'day' : 'days'}`
  }
  return `${hours} ${hours === 1 ? 'hour' : 'hours'}`
}

function SessionsCard({ v, set }: { v: Input; set: (x: Partial<Input>) => void }) {
  const withCurrent = (list: number[], cur: number) => (list.includes(cur) ? list : [...list, cur])
  return (
    <section className="card" aria-labelledby="sess-h">
      <CardSection title={<span id="sess-h">Sessions and lockout</span>} />
      <div className="pad col gap-3">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]">
          <label className="lbl">
            Session lifetime
            <select className="in" value={v.sessionLifetimeHours} onChange={(e) => set({ sessionLifetimeHours: Number(e.target.value) })}>
              {withCurrent(lifetimes, v.sessionLifetimeHours).map((h) => (
                <option key={h} value={h}>
                  {durationLabel(h)}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            Idle timeout
            <select className="in" value={v.sessionIdleMinutes} onChange={(e) => set({ sessionIdleMinutes: Number(e.target.value) })}>
              {withCurrent(idles, v.sessionIdleMinutes).map((m) => (
                <option key={m} value={m}>
                  {m % 60 === 0 ? durationLabel(m / 60) : `${m} minutes`}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            Lock after
            <span className="row flex-nowrap gap-2">
              <input
                className="in max-w-[90px]"
                type="number"
                min={3}
                max={100}
                value={v.lockoutAttempts}
                onChange={(e) => set({ lockoutAttempts: Number(e.target.value) || 3 })}
                aria-describedby="lock-unit"
              />
              <span className="k" id="lock-unit">
                failed attempts
              </span>
            </span>
          </label>
          <label className="lbl">
            Lock for
            <select className="in" value={v.lockoutMinutes} onChange={(e) => set({ lockoutMinutes: Number(e.target.value) })}>
              {withCurrent(locks, v.lockoutMinutes).map((m) => (
                <option key={m} value={m}>
                  {m % 60 === 0 ? durationLabel(m / 60) : `${m} minutes`}
                </option>
              ))}
            </select>
          </label>
        </div>
        <span className="k">Changing session settings signs nobody out; new limits apply at the next sign-in.</span>
      </div>
    </section>
  )
}

function AllowlistCard({ p, v, set }: { p: Policy; v: Input; set: (x: Partial<Input>) => void }) {
  const [adding, setAdding] = useState(false)
  const [domain, setDomain] = useState('')
  const [error, setError] = useState<string>()
  const add = () => {
    const d = domain.trim().toLowerCase().replace(/^@/, '')
    if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(d)) {
      setError('Enter a domain like acme.io.')
      return
    }
    if (!v.jitDomains.includes(d)) set({ jitDomains: [...v.jitDomains, d] })
    setDomain('')
    setError(undefined)
    setAdding(false)
  }
  const last = p.lastRejection
  return (
    <section className="card" aria-labelledby="allow-h">
      <CardSection title={<span id="allow-h">Email domain allowlist for automatic accounts</span>} />
      <div className="pad col gap-2.5">
        <div className="row gap-1.5">
          {v.jitDomains.map((d) => (
            <span key={d} className="chip">
              {d}
              <button type="button" aria-label={`Remove ${d}`} onClick={() => set({ jitDomains: v.jitDomains.filter((x) => x !== d) })}>
                ×
              </button>
            </span>
          ))}
          {adding ? (
            <span className="row flex-nowrap gap-1.5">
              <label className="vh" htmlFor="allow-add">
                Domain to allow
              </label>
              <input
                id="allow-add"
                className="in h-[30px] w-[160px]"
                value={domain}
                placeholder="acme-labs.io"
                autoFocus
                onChange={(e) => setDomain(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    add()
                  }
                  if (e.key === 'Escape') setAdding(false)
                }}
                aria-invalid={error ? true : undefined}
                aria-describedby={error ? 'allow-err' : undefined}
              />
              <Button size="sm" onClick={add}>
                Add
              </Button>
            </span>
          ) : (
            <Button size="sm" onClick={() => setAdding(true)}>
              Add domain
            </Button>
          )}
        </div>
        {error ? (
          <span className="err" id="allow-err" role="alert">
            {error}
          </span>
        ) : null}
        <span className="k">
          Applies to just-in-time accounts from SSO. Invited people can use any email.{' '}
          {v.jitDomains.length === 0 ? 'With no domain listed, any domain an SSO provider allows can get an account.' : 'Sign-ins from other domains are rejected and logged'}
          {v.jitDomains.length > 0 && last ? ` (for example ${last.email} ${dayPartWords(last.at)})` : ''}
          {v.jitDomains.length > 0 ? '.' : ''}
        </span>
      </div>
    </section>
  )
}

export function ssoStatusBadge(status: string) {
  switch (status) {
    case 'enabled':
      return (
        <Badge tone="ok" dot>
          Enabled
        </Badge>
      )
    case 'draft':
      return <Badge tone="warn">Setup in progress</Badge>
    case 'error':
      return <Badge tone="bad">Needs attention</Badge>
  }
  return <Badge tone="neutral">Disabled</Badge>
}

function SSOProvidersCard({ p }: { p: Policy }) {
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="ssolist-h">
      <CardSection
        title={<span id="ssolist-h">SSO providers</span>}
        caption="Configured at instance level · presets for Google Workspace, Microsoft Entra ID, Okta, Keycloak, Authentik and generic OIDC"
        actions={
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
            Add SSO provider
          </ButtonLink>
        }
      />
      {p.sso.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>No SSO provider yet. Add one to let people sign in with your organization&apos;s identity provider, create accounts automatically and map groups to roles.</span>
          </div>
        </div>
      ) : (
        <AccessTable caption="SSO providers">
          <thead>
            <tr>
              <Th>Provider</Th>
              <Th>Domains</Th>
              <Th>JIT</Th>
              <Th>Group mapping</Th>
              <Th>Enforced</Th>
              <Th>Status</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {p.sso.map((s) => (
              <tr key={s.id}>
                <Td>
                  <div className="row flex-nowrap gap-3">
                    <ProviderMark preset={s.preset} />
                    <span className="font-semibold">{s.name}</span>
                  </div>
                </Td>
                <Td>
                  <span className="mono">{s.domains.join(', ') || '–'}</span>
                </Td>
                <Td>{s.jit.enabled ? `On → ${s.jit.workspaceName} · ${s.jit.roleName}` : 'Off'}</Td>
                <Td>{!s.groupsClaim ? <span className="k">Unavailable for {s.name}</span> : s.mappings === 0 ? 'None' : `${s.mappings} ${s.mappings === 1 ? 'mapping' : 'mappings'}`}</Td>
                <Td>{s.enforced.length ? 'Yes · except break-glass Owners' : 'No'}</Td>
                <Td>{ssoStatusBadge(s.status)}</Td>
                <Td>
                  <ButtonLink size="sm" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: s.id }}>
                    {s.status === 'draft' ? 'Continue setup' : 'Edit'}
                  </ButtonLink>
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

function PolicySkeleton() {
  return (
    <div className={cards} aria-busy="true">
      <span className="vh" role="status">
        Loading the authentication policy
      </span>
      {Array.from({ length: 4 }, (_, i) => (
        <section key={i} className="card">
          <div className="sec">
            <Skeleton w={180} h={14} />
          </div>
          <div className="pad col gap-2.5">
            <Skeleton w="100%" h={44} r={8} />
            <Skeleton w="100%" h={44} r={8} />
            <Skeleton w="70%" h={14} />
          </div>
        </section>
      ))}
    </div>
  )
}

