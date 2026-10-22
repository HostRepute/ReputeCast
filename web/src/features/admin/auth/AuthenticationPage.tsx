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
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, count, dayPartWords } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { m as am } from './messages'
import { SocialSettingsDialog } from './SocialSettingsDialog'

type Policy = Schemas['InstancePolicy']
type Input = Schemas['InstancePolicyInput']

const cards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(460px,100%),1fr))] max-md:grid-cols-1'

/** Instance admin › Authentication (InstanceAuth). */
export function AuthenticationPage() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/policy')
  return (
    <AdminPage documentTitle={t(am.title)}>
      {q.isError ? (
        <ErrorPanel title={t(am.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(am.loadFailedBody)}
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
  const t = useT()
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
      toast({ kind: 'success', title: t(am.savedToast), detail: t(am.savedDetail) })
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
        <ErrorPanel title={t(am.saveFailed)}>
          {error.message}
        </ErrorPanel>
      ) : null}
      <div className="row justify-end gap-2.5">
        <Button onClick={() => setV(toInput(p))}>
          {t(am.discard)}
        </Button>
        <Button variant="primary" loading={save.isPending} onClick={() => void submit()}>
          {t(am.savePolicies)}
        </Button>
      </div>
      {social ? <SocialSettingsDialog provider={social} onClose={() => setSocial(undefined)} /> : null}
      <UnsavedChangesGuard dirty={dirty} title={t(am.leaveTitle)} description={t(am.leaveBody)} saveLabel={t(am.saveAndLeave)} onSave={submit} />
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
  const t = useT()
  const social = (k: 'google' | 'microsoft' | 'github') => p.social.find((s) => s.kind === k)!
  const onBadge = (on: boolean, text = t(common.on)) => <Badge tone={on ? 'ok' : 'neutral'}>{on ? text : t(common.off)}</Badge>
  const enabledSSO = p.sso.filter((s) => s.status === 'enabled')
  const otherSSO = p.sso.filter((s) => s.status !== 'enabled')
  const ssoLine = p.sso.length === 0 ? t(am.ssoNone) : [...enabledSSO, ...otherSSO].map((s) => t(am.ssoItem, { name: s.name, status: s.status })).join(' · ')
  const socialRow = (k: 'google' | 'microsoft' | 'github', description: ReactNode, onText = t(common.on)) => {
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
          <Button size="sm" iconOnly icon="edit" aria-label={t(am.socialSettings, { name: s.name })} title={t(am.socialSettings, { name: s.name })} onClick={() => onSocial(s)} className="border-transparent" />
        }
      />
    )
  }
  const gh = social('github')
  return (
    <section className="card" aria-labelledby="methods-h">
      <CardSection title={<span id="methods-h">{t(am.methodsTitle)}</span>} caption={t(am.methodsCaption)} />
      <div className="pad col gap-2.5">
        <MethodRow
          checked={v.methods.password}
          onChange={(x) => set({ methods: { ...v.methods, password: x } })}
          title={t(am.passwordMethod)}
          description={t(am.passwordMethodHint, { min: count(v.passwordMinLength), check: v.breachedCheck ? 'on' : 'off', passwordOff: v.methods.password ? 'no' : 'yes' })}
          badge={onBadge(v.methods.password)}
        />
        <MethodRow
          checked={v.methods.passkeys}
          onChange={(x) => set({ methods: { ...v.methods, passkeys: x } })}
          title={t(am.passkeysMethod)}
          description={t(am.passkeysMethodHint)}
          badge={onBadge(v.methods.passkeys)}
        />
        {socialRow('google', t(am.googleHint))}
        {socialRow('microsoft', t(am.microsoftHint))}
        {socialRow(
          'github',
          gh.org ? <T msg={am.githubOrgHint} values={{ org: gh.org, mono: (c) => <span className="mono">{c}</span> }} /> : t(am.githubAnyHint),
          gh.org ? t(am.githubOrgOn) : t(common.on),
        )}
        <MethodRow
          checked={v.methods.sso}
          onChange={(x) => set({ methods: { ...v.methods, sso: x } })}
          title={t(am.ssoMethod)}
          description={ssoLine}
          badge={<Badge tone={v.methods.sso ? 'ok' : 'neutral'}>{v.methods.sso ? t(am.ssoOn, { count: p.sso.length }) : t(common.off)}</Badge>}
        />
      </div>
    </section>
  )
}

function PasswordCard({ v, set }: { v: Input; set: (x: Partial<Input>) => void }) {
  const t = useT()
  return (
    <section className="card" aria-labelledby="pw-h">
      <CardSection title={<span id="pw-h">{t(am.passwordPolicy)}</span>} />
      <div className="pad col gap-3">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            {t(am.minLength)}
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
              {t(am.minLengthHint)}
            </span>
          </label>
          <label className="lbl">
            {t(am.reuse)}
            <select className="in" value={v.passwordReuse} onChange={(e) => set({ passwordReuse: Number(e.target.value) as Input['passwordReuse'] })}>
              <option value={5}>{t(am.reuseBlock5)}</option>
              <option value={10}>{t(am.reuseBlock10)}</option>
              <option value={0}>{t(am.reuseAllow)}</option>
            </select>
          </label>
        </div>
        {/* eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text */}
        <label className="row flex-nowrap gap-3">
          <input type="checkbox" className="tg" role="switch" checked={v.breachedCheck} onChange={(e) => set({ breachedCheck: e.target.checked })} />
          <span className="col gap-px">
            <span className="font-semibold">{t(am.breachedTitle)}</span>
            <span className="k">{t(am.breachedHint)}</span>
          </span>
        </label>
      </div>
    </section>
  )
}

const graceOptions = [7, 1, 3, 14, 0]

function TwoFactorCard({ p, v, set }: { p: Policy; v: Input; set: (x: Partial<Input>) => void }) {
  const t = useT()
  const options: { value: Input['twoFactor']; title: string; description: string }[] = [
    { value: 'everyone', title: t(am.tfaEveryone), description: t(am.tfaEveryoneHint) },
    { value: 'sensitive', title: t(am.tfaSensitive), description: t(am.tfaSensitiveHint) },
    { value: 'off', title: t(am.tfaOff), description: t(am.tfaOffHint) },
  ]
  const grace = graceOptions.includes(v.graceDays) ? graceOptions : [...graceOptions, v.graceDays]
  const inGrace = p.inGrace
  return (
    <section className="card" aria-labelledby="tfa-h">
      <CardSection title={<span id="tfa-h">{t(am.tfaTitle)}</span>} />
      <div className="pad col gap-2.5">
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="vh">{t(am.tfaLegend)}</legend>
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
            {t(am.grace)}
            <select className="in max-w-[160px]" value={v.graceDays} onChange={(e) => set({ graceDays: Number(e.target.value) })}>
              {grace.map((d) => (
                <option key={d} value={d}>
                  {d === 0 ? t(am.graceNone) : t(am.durationDays, { count: d })}
                </option>
              ))}
            </select>
          </label>
          <span className="k">
            {inGrace.length === 0
              ? t(am.inGraceNone)
              : t(am.inGrace, { count: inGrace.length, list: andList(inGrace.map((g) => t(am.graceEntry, { name: g.name, days: untilDays(g.until) }))) })}
          </span>
        </div>
      </div>
    </section>
  )
}

/** Whole days until a grace period ends: 0 once it ends today. */
function untilDays(at: string) {
  return Math.max(0, Math.ceil((new Date(at).getTime() - Date.now()) / 86_400_000))
}

const lifetimes = [14 * 24, 24, 7 * 24, 30 * 24]
const idles = [12 * 60, 60, 4 * 60, 24 * 60]
const locks = [15, 5, 60]

function durationLabel(hours: number) {
  if (hours % 24 === 0) return tt(am.durationDays, { count: hours / 24 })
  return tt(am.durationHours, { count: hours })
}

function SessionsCard({ v, set }: { v: Input; set: (x: Partial<Input>) => void }) {
  const t = useT()
  const withCurrent = (list: number[], cur: number) => (list.includes(cur) ? list : [...list, cur])
  return (
    <section className="card" aria-labelledby="sess-h">
      <CardSection title={<span id="sess-h">{t(am.sessionsTitle)}</span>} />
      <div className="pad col gap-3">
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]">
          <label className="lbl">
            {t(am.lifetime)}
            <select className="in" value={v.sessionLifetimeHours} onChange={(e) => set({ sessionLifetimeHours: Number(e.target.value) })}>
              {withCurrent(lifetimes, v.sessionLifetimeHours).map((h) => (
                <option key={h} value={h}>
                  {durationLabel(h)}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            {t(am.idle)}
            <select className="in" value={v.sessionIdleMinutes} onChange={(e) => set({ sessionIdleMinutes: Number(e.target.value) })}>
              {withCurrent(idles, v.sessionIdleMinutes).map((m) => (
                <option key={m} value={m}>
                  {m % 60 === 0 ? durationLabel(m / 60) : t(am.durationMinutes, { count: m })}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            {t(am.lockAfter)}
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
                {t(am.failedAttempts)}
              </span>
            </span>
          </label>
          <label className="lbl">
            {t(am.lockFor)}
            <select className="in" value={v.lockoutMinutes} onChange={(e) => set({ lockoutMinutes: Number(e.target.value) })}>
              {withCurrent(locks, v.lockoutMinutes).map((m) => (
                <option key={m} value={m}>
                  {m % 60 === 0 ? durationLabel(m / 60) : t(am.durationMinutes, { count: m })}
                </option>
              ))}
            </select>
          </label>
        </div>
        <span className="k">{t(am.sessionsHint)}</span>
      </div>
    </section>
  )
}

function AllowlistCard({ p, v, set }: { p: Policy; v: Input; set: (x: Partial<Input>) => void }) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [domain, setDomain] = useState('')
  const [error, setError] = useState<string>()
  const add = () => {
    const d = domain.trim().toLowerCase().replace(/^@/, '')
    if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(d)) {
      setError(t(am.domainInvalid))
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
      <CardSection title={<span id="allow-h">{t(am.allowlistTitle)}</span>} />
      <div className="pad col gap-2.5">
        <div className="row gap-1.5">
          {v.jitDomains.map((d) => (
            <span key={d} className="chip">
              {d}
              <button type="button" aria-label={t(am.removeDomain, { domain: d })} onClick={() => set({ jitDomains: v.jitDomains.filter((x) => x !== d) })}>
                ×
              </button>
            </span>
          ))}
          {adding ? (
            <span className="row flex-nowrap gap-1.5">
              <label className="vh" htmlFor="allow-add">
                {t(am.domainToAdd)}
              </label>
              <input
                id="allow-add"
                className="in h-[30px] w-[160px]"
                value={domain}
                placeholder={'acme-labs.io'}
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
                {t(am.add)}
              </Button>
            </span>
          ) : (
            <Button size="sm" onClick={() => setAdding(true)}>
              {t(am.addDomain)}
            </Button>
          )}
        </div>
        {error ? (
          <span className="err" id="allow-err" role="alert">
            {error}
          </span>
        ) : null}
        <span className="k">
          {t(am.allowlistHint)}{' '}
          {v.jitDomains.length === 0
            ? t(am.allowlistAny)
            : t(am.allowlistRejected, { last: last ? 'yes' : 'no', email: last?.email ?? '', when: last ? dayPartWords(last.at) : '' })}
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
          {tt(am.statusEnabled)}
        </Badge>
      )
    case 'draft':
      return <Badge tone="warn">{tt(am.statusDraft)}</Badge>
    case 'error':
      return <Badge tone="bad">{tt(am.statusError)}</Badge>
  }
  return <Badge tone="neutral">{tt(am.statusDisabled)}</Badge>
}

function SSOProvidersCard({ p }: { p: Policy }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="ssolist-h">
      <CardSection
        title={<span id="ssolist-h">{t(am.ssoProviders)}</span>}
        caption={t(am.ssoProvidersCaption)}
        actions={
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
            {t(am.addSsoProvider)}
          </ButtonLink>
        }
      />
      {p.sso.length === 0 ? (
        <div className="pad">
          <div className="note n">
            <span>{t(am.ssoProvidersEmpty)}</span>
          </div>
        </div>
      ) : (
        <AccessTable caption={t(am.ssoProviders)}>
          <thead>
            <tr>
              <Th>{t(am.colProvider)}</Th>
              <Th>{t(am.colDomains)}</Th>
              <Th>{t(am.colJit)}</Th>
              <Th>{t(am.colGroupMapping)}</Th>
              <Th>{t(am.colEnforced)}</Th>
              <Th>{t(am.colStatus)}</Th>
              <Th>
                <span className="vh">{t(am.colActions)}</span>
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
                <Td>
                  {s.jit.enabled
                    ? t(am.jitOn, { workspace: s.jit.workspaceName ?? '', role: roleName({ key: s.jit.roleKey, name: s.jit.roleName ?? '' }) })
                    : t(common.off)}
                </Td>
                <Td>{!s.groupsClaim ? <span className="k">{t(am.mappingsUnavailable, { name: s.name })}</span> : t(am.mappingsCount, { count: s.mappings })}</Td>
                <Td>{s.enforced.length ? t(am.enforcedYes) : t(common.no)}</Td>
                <Td>{ssoStatusBadge(s.status)}</Td>
                <Td>
                  <ButtonLink size="sm" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: s.id }}>
                    {s.status === 'draft' ? t(am.continueSetup) : t(common.edit)}
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
  const t = useT()
  return (
    <div className={cards} aria-busy="true">
      <span className="vh" role="status">
        {t(am.loading)}
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

