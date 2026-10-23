import { useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Popover } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, copyText, ProviderMark } from '@/features/access/parts'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, andList, calendarDate, count, monthDay } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { useInvalidateSSO } from '../queries'
import { EnforcementDialog } from './EnforcementDialog'
import { richHint, shortURL } from './hints'
import { m as sm } from './messages'
import { m as ss } from './setup.messages'
import { finishTestPopup, onTestDone, openTestPopup } from './testSignIn'

type Detail = Schemas['SSOProviderDetail']
type Preset = Schemas['SSOPreset']
type Mapping = Schemas['SSOGroupMapping']

interface Form {
  preset: string
  issuerUrl: string
  clientId: string
  replacing: boolean
  clientSecret: string
  secretExpiresOn: string
  scopes: string[]
  claims: Schemas['SSOClaimMap']
  allowedDomains: string[]
  jit: { enabled: boolean; workspaceId: string; roleKey: string }
  groupMappings: Mapping[]
}

function toForm(d: Detail): Form {
  return {
    preset: d.preset,
    issuerUrl: d.issuer,
    clientId: d.clientId,
    replacing: !d.secret.set,
    clientSecret: '',
    secretExpiresOn: '',
    scopes: [...d.scopes],
    claims: { ...d.claims },
    allowedDomains: [...d.domains],
    jit: { enabled: d.jit.enabled, workspaceId: d.jit.workspaceId ?? d.workspaces[0]?.id ?? '', roleKey: d.jit.roleKey || 'viewer' },
    groupMappings: d.groupMappings.map((m) => ({ ...m, lists: [...(m.lists ?? [])] })),
  }
}

function toInput(d: Detail, f: Form): Schemas['SSOProviderInput'] {
  return {
    preset: f.preset !== d.preset ? (f.preset as Schemas['SSOProviderInput']['preset']) : undefined,
    name: d.name,
    issuerUrl: f.issuerUrl.trim(),
    clientId: f.clientId.trim(),
    clientSecret: f.replacing && f.clientSecret.trim() ? f.clientSecret.trim() : undefined,
    secretExpiresOn: f.replacing && f.clientSecret.trim() && f.secretExpiresOn ? f.secretExpiresOn : undefined,
    scopes: f.scopes,
    claims: f.claims,
    allowedDomains: f.allowedDomains,
    jit: { enabled: f.jit.enabled, workspaceId: f.jit.workspaceId || undefined, roleKey: f.jit.roleKey || undefined },
    groupMappings: f.groupMappings,
    twoFactorMode: d.twoFactorMode,
  }
}

/** Instance admin › SSO providers › a provider (SSOSetup, SSOStates). */
export function SSOSetupPage({ providerId, test }: { providerId: string; test?: 'done' }) {
  if (providerId === 'new') return <NewProvider />
  return <ExistingProvider providerId={providerId} test={test} />
}

function ExistingProvider({ providerId, test }: { providerId: string; test?: 'done' }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = $api.useQuery('get', '/api/v1/admin/sso/{providerId}', { params: { path: { providerId } } })
  // Back from a test sign-in: tell the tab that started it, close the popup.
  useEffect(() => {
    if (test !== 'done') return
    finishTestPopup(providerId)
    void navigate({ to: '/w/$workspace/admin/sso/$providerId', params: { workspace: slug, providerId }, search: {}, replace: true })
  }, [test, providerId, navigate, slug])
  useEffect(() => onTestDone(providerId, () => void q.refetch()), [providerId, q])
  return (
    <AdminPage documentTitle={q.data?.name ?? t(ss.docTitle)} subtitle={q.data ? t(ss.subtitle, { name: q.data.name, status: q.data.status }) : undefined}>
      {q.isError ? (
        <ErrorPanel title={t(ss.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.isPending ? (
        <SetupSkeleton />
      ) : (
        <SetupForm d={q.data} />
      )}
    </AdminPage>
  )
}

/** Step 1 for a new provider: picking a preset creates a draft. */
function NewProvider() {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const list = $api.useQuery('get', '/api/v1/admin/sso')
  const create = $api.useMutation('post', '/api/v1/admin/sso')
  const invalidate = useInvalidateSSO()
  return (
    <AdminPage documentTitle={t(ss.newDocTitle)} subtitle={t(ss.newSubtitle)}>
      <section className="card" aria-labelledby="preset-h">
        <CardSection title={<span id="preset-h">{t(ss.presetTitle)}</span>} caption={t(ss.presetPick)} />
        <div className="pad col gap-2.5">
          {list.isPending ? (
            <Skeleton w="80%" h={30} r={8} />
          ) : (
            <div className="row gap-2">
              {(list.data?.presets ?? []).map((p) => (
                <Button
                  key={p.key}
                  size="sm"
                  aria-pressed={false}
                  loading={create.isPending && create.variables?.body.preset === p.key}
                  onClick={async () => {
                    try {
                      const d = await create.mutateAsync({ body: { preset: p.key } })
                      await invalidate()
                      void navigate({ to: '/w/$workspace/admin/sso/$providerId', params: { workspace: slug, providerId: d.id }, replace: true })
                    } catch (e) {
                      toast({ kind: 'error', title: t(ss.addFailed), detail: errorMessage(e) })
                    }
                  }}
                >
                  <ProviderMark preset={p.key} />
                  {p.name}
                </Button>
              ))}
            </div>
          )}
          <span className="k">{t(ss.presetNewHint)}</span>
        </div>
      </section>
    </AdminPage>
  )
}

function SetupForm({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateSSO()
  const [f, setF] = useState<Form>(() => toForm(d))
  // What a mutation returned, until the next refetch (a finished test
  // sign-in, discovery) brings a newer provider; the form keeps the edits
  // that aren't saved yet either way.
  const [returned, setReturned] = useState<{ from: Detail; value: Detail }>()
  const setSaved = (value: Detail) => setReturned({ from: d, value })
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/admin/sso/{providerId}')
  const enable = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/enable')
  const startTest = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/test')
  const [enforcing, setEnforcing] = useState(false)
  const [unenforcing, setUnenforcing] = useState(false)
  const set = (patch: Partial<Form>) => setF((x) => ({ ...x, ...patch }))
  const current = returned && returned.from === d ? returned.value : d
  const dirty = JSON.stringify(toInput(current, f)) !== JSON.stringify(toInput(current, toForm(current)))
  const presetInfo = (useList().data?.presets ?? []).find((p) => p.key === f.preset) ?? current.presetInfo
  const path = { params: { path: { providerId: current.id } } }

  const persist = async (form: Form = f): Promise<Detail> => {
    setError(undefined)
    const r = await save.mutateAsync({ ...path, body: toInput(current, form) })
    setF(toForm(r))
    setSaved(r)
    await invalidate()
    return r
  }
  const run = async (what: () => Promise<unknown>) => {
    try {
      await what()
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  const runTest = () =>
    run(async () => {
      if (dirty) await persist()
      const r = await startTest.mutateAsync({ ...path, body: { workspace: slug } })
      openTestPopup(r.authorizeUrl)
    })

  return (
    <>
      <PresetCard d={current} f={f} set={set} />
      <ConnectionCard d={current} f={f} set={set} preset={presetInfo} onDiscovered={(r) => { setSaved(r); set({ issuerUrl: r.issuer }) }} />
      <ClaimsCard d={current} f={f} set={set} preset={presetInfo} />
      <AccessCard d={current} f={f} set={set} preset={presetInfo} onEnforce={() => setEnforcing(true)} onUnenforce={() => setUnenforcing(true)} />
      <TestCard
        d={current}
        preset={presetInfo}
        running={startTest.isPending}
        dropping={save.isPending}
        onRun={() => void runTest()}
        onDropMappings={() =>
          void run(async () => {
            const r = await persist({ ...f, groupMappings: [] })
            toast({ kind: 'success', title: t(ss.mappingsRemoved), detail: t(ss.mappingsRemovedDetail, { name: r.name }) })
          })
        }
      />
      {error ? (
        <ErrorPanel title={t(ss.notSaved)} actions={<Button onClick={() => setError(undefined)}>{t(ss.dismiss)}</Button>}>
          {error}
        </ErrorPanel>
      ) : null}
      <div className="row justify-between gap-2.5">
        <span className="k">{current.status === 'enabled' ? t(ss.changesApply) : t(ss.nothingLive)}</span>
        <div className="row gap-2">
          {current.status === 'enabled' ? (
            <Button
              variant="primary"
              loading={save.isPending}
              onClick={() =>
                void run(async () => {
                  await persist()
                  toast({ kind: 'success', title: t(ss.savedToast, { name: current.name }) })
                })
              }
            >
              {t(common.saveChanges)}
            </Button>
          ) : (
            <>
              <Button
                loading={save.isPending && !enable.isPending}
                onClick={() =>
                  void run(async () => {
                    await persist()
                    toast({ kind: 'success', title: t(ss.draftSaved), detail: t(ss.draftSavedDetail) })
                  })
                }
              >
                {t(ss.saveDraft)}
              </Button>
              <Button
                variant="primary"
                loading={enable.isPending}
                onClick={() =>
                  void run(async () => {
                    const r = dirty ? await persist() : current
                    if (r.enableBlockers.length) throw new Error(t(ss.beforeEnabling, { blockers: r.enableBlockers.join(' ') }))
                    const e = await enable.mutateAsync(path)
                    setSaved(e)
                    setF(toForm(e))
                    await invalidate()
                    toast({ kind: 'success', title: t(sm.enabledToast, { name: e.name }), detail: t(sm.enabledDetail) })
                  })
                }
              >
                {t(ss.enableName, { name: current.name })}
              </Button>
            </>
          )}
        </div>
      </div>
      {enforcing ? <EnforcementDialog d={current} onClose={() => setEnforcing(false)} onDone={(r) => { setSaved(r); setF(toForm(r)) }} /> : null}
      {unenforcing ? (
        <TurnOffEnforcement d={current} onClose={() => setUnenforcing(false)} onDone={(r) => { setSaved(r); setF(toForm(r)) }} />
      ) : null}
      <UnsavedChangesGuard dirty={dirty} title={t(ss.leaveTitle)} description={t(ss.leaveBody, { name: current.name })} onSave={() => persist()} />
    </>
  )
}

function useList() {
  return $api.useQuery('get', '/api/v1/admin/sso')
}

type CardProps = { d: Detail; f: Form; set: (p: Partial<Form>) => void }

function PresetCard({ d, f, set }: CardProps) {
  const t = useT()
  const presets = useList().data?.presets ?? []
  const live = d.status === 'enabled'
  const name = presets.find((p) => p.key === f.preset)?.name ?? d.presetInfo.name
  return (
    <section className="card" aria-labelledby="preset-h">
      <CardSection title={<span id="preset-h">{t(ss.presetTitle)}</span>} caption={t(ss.presetSelected, { name })} />
      <div className="pad col gap-2.5">
        <div className="row gap-2">
          {presets.map((p) => {
            const on = p.key === f.preset
            return (
              <Button
                key={p.key}
                size="sm"
                aria-pressed={on}
                className={on ? 'border-accent! bg-accent-soft!' : undefined}
                disabledReason={!on && live ? t(ss.presetSwitchBlocked, { name: d.name }) : undefined}
                onClick={() => {
                  if (on) return
                  set({ preset: p.key, scopes: [...p.scopes], issuerUrl: p.issuer || (f.issuerUrl === presetIssuer(presets, f.preset) ? '' : f.issuerUrl) })
                }}
              >
                <ProviderMark preset={p.key} />
                {p.name}
              </Button>
            )
          })}
        </div>
        <span className="k">{t(ss.presetHint)}</span>
      </div>
    </section>
  )
}

function presetIssuer(presets: Preset[], key: string) {
  return presets.find((p) => p.key === key)?.issuer ?? ''
}

/** A date without a time ("Apr 10, 2027"), the same in every timezone. */
function dateOnly(iso: string) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : calendarDate(d.toISOString().slice(0, 10), 'date')
}

function ConnectionCard({ d, f, set, preset, onDiscovered }: CardProps & { preset: Preset; onDiscovered: (d: Detail) => void }) {
  const t = useT()
  const id = useId()
  const discover = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/discovery')
  const [error, setError] = useState<string>()
  const disc = d.discovery
  const failed = disc && !disc.ok
  const k = (c: ReactNode) => <span className="k font-medium">{c}</span>
  const test = async () => {
    setError(undefined)
    try {
      const r = await discover.mutateAsync({ params: { path: { providerId: d.id } }, body: { issuerUrl: f.issuerUrl.trim() || undefined } })
      onDiscovered(r)
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  if (failed)
    // SSOStates "Discovery failed": the issuer, what the URL answered and how
    // to find the right one. The rest of the step returns once it passes.
    return (
      <section className="card" aria-labelledby="conn-h">
        <CardSection
          title={<span id="conn-h">{t(ss.discoveryFailedTitle)}</span>}
          caption={d.status === 'enabled' ? t(ss.discoveryFailedLive) : t(ss.discoveryFailedDraft)}
          actions={d.status === 'enabled' ? <Badge tone="warn">{t(ss.stillLive)}</Badge> : <Badge tone="bad">{t(ss.notConfigured)}</Badge>}
        />
        <div className="pad col gap-3">
          <label className="lbl">
            {t(ss.issuerUrl)}
            <input
              className="in mono"
              value={f.issuerUrl}
              placeholder={preset.issuerPattern}
              onChange={(e) => set({ issuerUrl: e.target.value })}
              spellCheck={false}
              autoComplete="off"
              aria-invalid
              aria-describedby={`${id}-disc-err`}
            />
          </label>
          <div className="note b" role="alert" id={`${id}-disc-err`}>
            <Badge tone="bad">{disc.status ? String(disc.status) : t(ss.discoveryErrorBadge)}</Badge>
            <span className="col gap-1">
              <span>
                <T
                  msg={ss.discoveryAnswer}
                  values={{
                    hasUrl: disc.url ? 'yes' : 'no',
                    url: shortURL(disc.url),
                    error: disc.error ?? '',
                    timed: disc.ms && disc.status ? 'yes' : 'no',
                    ms: count(disc.ms),
                    mono: (c) => <span className="mono">{c}</span>,
                  }}
                />
              </span>
              <span>{richHint(preset.discoveryHint)}</span>
            </span>
          </div>
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
          <div className="row gap-2">
            <Button size="sm" variant="primary" loading={discover.isPending} onClick={() => void test()}>
              {t(ss.testDiscoveryAgain)}
            </Button>
            <ButtonAnchor size="sm" href={preset.docsUrl}>
              {preset.docsLabel}
            </ButtonAnchor>
          </div>
        </div>
      </section>
    )
  const secretHint = d.secret.set
    ? t(ss.secretSet, {
        saved: d.secret.setAt ? (d.secret.setBy ? 'byName' : 'date') : 'plain',
        date: d.secret.setAt ? monthDay(d.secret.setAt) : '',
        name: d.secret.setBy ?? '',
        expires: d.secret.expiresOn ? 'yes' : 'no',
        idp: preset.key === 'entra' ? 'Entra' : preset.shortName,
        expiresOn: d.secret.expiresOn ? dateOnly(d.secret.expiresOn) : '',
      })
    : t(ss.secretPaste, { app: preset.app })
  return (
    <section className="card" aria-labelledby="conn-h">
      <CardSection title={<span id="conn-h">{t(ss.connectionTitle)}</span>} />
      <div className="pad col gap-3.5">
        <div className="lbl">
          <span>
            <T msg={ss.redirectLabel} values={{ app: preset.app, k }} />
          </span>
          <div className="row flex-nowrap gap-2">
            <input className="in mono flex-1 bg-ground" value={d.redirectUri} readOnly aria-label={t(ss.redirectUri)} aria-describedby={`${id}-redir`} />
            <Button
              onClick={async () => {
                const ok = await copyText(d.redirectUri)
                toast(ok ? { kind: 'success', title: t(ss.redirectCopied) } : { kind: 'error', title: t(ss.copyFailed), detail: t(ss.copyFailedDetail) })
              }}
            >
              {t(common.copy)}
            </Button>
          </div>
          <span className="k font-medium" id={`${id}-redir`}>
            {t(ss.redirectHint)}
          </span>
        </div>
        <div className="lbl">
          <label htmlFor={`${id}-issuer`}>{t(ss.issuerUrl)}</label>
          <div className="row flex-nowrap gap-2">
            <input
              id={`${id}-issuer`}
              className="in mono flex-1"
              value={f.issuerUrl}
              placeholder={preset.issuerPattern}
              onChange={(e) => set({ issuerUrl: e.target.value })}
              spellCheck={false}
              autoComplete="off"
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? `${id}-disc ${id}-disc-err` : `${id}-disc`}
            />
            <Button loading={discover.isPending} onClick={() => void test()}>
              {t(ss.testDiscovery)}
            </Button>
          </div>
          <span className="k font-medium" id={`${id}-disc`}>
            {disc?.ok ? (
              <span className="row gap-1.5">
                <Badge tone="ok">{t(ss.discoveryOk)}</Badge> {t(ss.discoveryFound, { ago: ago(disc.checkedAt), endpoints: andList(disc.endpoints) })}
              </span>
            ) : !disc ? (
              t(ss.issuerPrompt, { pattern: preset.issuerPattern })
            ) : null}
          </span>
        </div>
        {error ? (
          <span className="err" role="alert" id={`${id}-disc-err`}>
            {error}
          </span>
        ) : null}
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <label className="lbl">
            {t(ss.clientId)}
            <input className="in mono" value={f.clientId} onChange={(e) => set({ clientId: e.target.value })} spellCheck={false} autoComplete="off" />
          </label>
          <div className="lbl">
            <label htmlFor={`${id}-secret`}>
              <T msg={ss.clientSecretLabel} values={{ k }} />
            </label>
            {f.replacing ? (
              <div className="row flex-nowrap gap-2">
                <input
                  id={`${id}-secret`}
                  className="in mono flex-1"
                  type="password"
                  value={f.clientSecret}
                  placeholder={d.secret.set ? t(ss.newSecret) : t(ss.pasteSecret)}
                  autoComplete="new-password"
                  onChange={(e) => set({ clientSecret: e.target.value })}
                />
                {d.secret.set ? <Button onClick={() => set({ replacing: false, clientSecret: '' })}>{t(ss.keepSecret)}</Button> : null}
              </div>
            ) : (
              <div className="row flex-nowrap gap-2">
                <input id={`${id}-secret`} className="in flex-1 bg-ground" type="password" value="••••••••••••••••••••" readOnly aria-describedby={`${id}-sec-hint`} />
                <Button onClick={() => set({ replacing: true })}>{t(ss.replaceSecret)}</Button>
              </div>
            )}
            {f.replacing && f.clientSecret ? (
              <span className="row flex-nowrap gap-2">
                <label className="k font-medium" htmlFor={`${id}-exp`}>
                  {t(ss.secretExpiresLabel)}
                </label>
                <input id={`${id}-exp`} className="in h-[30px]" type="date" value={f.secretExpiresOn} onChange={(e) => set({ secretExpiresOn: e.target.value })} />
              </span>
            ) : null}
            <span className="k font-medium" id={`${id}-sec-hint`}>
              {secretHint}
            </span>
          </div>
        </div>
        <ScopesField f={f} set={set} preset={preset} />
      </div>
    </section>
  )
}

function ScopesField({ f, set, preset }: { f: Form; set: (p: Partial<Form>) => void; preset: Preset }) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [value, setValue] = useState('')
  const fixed = ['openid', 'profile', 'email']
  const add = () => {
    const v = value.trim()
    if (v && !/\s/.test(v) && !f.scopes.includes(v)) set({ scopes: [...f.scopes, v] })
    setValue('')
    setAdding(false)
  }
  return (
    <div className="lbl">
      <span id="scopes-l">{t(ss.scopes)}</span>
      <div className="row gap-1.5" role="group" aria-labelledby="scopes-l">
        {f.scopes.map((s) => (
          <span key={s} className="chip">
            {s}
            {fixed.includes(s) ? null : (
              <button type="button" aria-label={t(ss.removeScope, { scope: s })} onClick={() => set({ scopes: f.scopes.filter((x) => x !== s) })}>
                ×
              </button>
            )}
          </span>
        ))}
        {adding ? (
          <span className="row flex-nowrap gap-1.5">
            <input
              className="in h-[30px] w-[140px]"
              value={value}
              autoFocus
              aria-label={t(ss.scopeToAdd)}
              placeholder={'groups'}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  add()
                }
                if (e.key === 'Escape') setAdding(false)
              }}
            />
            <Button size="sm" onClick={add}>
              {t(ss.add)}
            </Button>
          </span>
        ) : (
          <Button size="sm" onClick={() => setAdding(true)}>
            {t(ss.addScope)}
          </Button>
        )}
      </div>
      <span className="k font-medium">{preset.groupsHint}</span>
    </div>
  )
}

function ClaimsCard({ d, f, set, preset }: CardProps & { preset: Preset }) {
  const t = useT()
  const samples = d.test?.samples ?? []
  const rows: { field: keyof Schemas['SSOClaimMap']; label: string }[] = [
    { field: 'email', label: 'email' },
    { field: 'emailVerified', label: 'email_verified' },
    { field: 'name', label: 'name' },
    ...(preset.groupsClaim ? [{ field: 'groups' as const, label: 'groups' }] : []),
  ]
  const sampleOf = (label: string) => samples.find((s) => s.field === label)
  return (
    <section className="card" aria-labelledby="claims-h">
      <CardSection title={<span id="claims-h">{t(ss.claimsTitle)}</span>} caption={t(ss.claimsCaption)} />
      <AccessTable caption={t(ss.claimsTable)}>
        <thead>
          <tr>
            <Th>{t(ss.colField)}</Th>
            <Th>{t(ss.colClaim)}</Th>
            <Th>{t(ss.colSample)}</Th>
            <Th>
              <span className="vh">{t(ss.colResult)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const s = sampleOf(r.label)
            return (
              <tr key={r.field}>
                <Td>{r.label}</Td>
                <Td>
                  <input
                    className="in mono w-[180px]"
                    value={f.claims[r.field]}
                    aria-label={t(ss.claimFor, { field: r.label.replace('_', ' ') })}
                    onChange={(e) => set({ claims: { ...f.claims, [r.field]: e.target.value } })}
                  />
                </Td>
                <Td>
                  {!s ? (
                    <span className="k">{t(ss.runTestFirst)}</span>
                  ) : s.found ? (
                    <>
                      <span className="mono">{s.value}</span>
                      {r.field === 'groups' && preset.groupIds ? <span className="k"> {t(ss.objectIds)}</span> : null}
                    </>
                  ) : (
                    <span className="k">{s.missing ?? t(ss.notSentBy, { idp: preset.shortName })}</span>
                  )}
                </Td>
                <Td>
                  {!s ? null : s.found ? (
                    <Badge tone="ok">{r.field === 'groups' && s.count ? t(ss.foundCount, { count: count(s.count) }) : t(ss.found)}</Badge>
                  ) : (
                    <span className="row flex-nowrap gap-1.5">
                      <Badge tone="warn">{t(ss.missing)}</Badge>
                      {s.note ? <span className="k">{s.note}</span> : null}
                    </span>
                  )}
                </Td>
              </tr>
            )
          })}
        </tbody>
      </AccessTable>
      {preset.groupIds ? (
        <div className="pad pt-3">
          <div className="note i">
            <ProviderMark preset={preset.key} />
            <span>
              <T msg={ss.groupIdsNote} values={{ idp: preset.shortName, b: (c) => <b>{c}</b> }} />
            </span>
          </div>
        </div>
      ) : null}
    </section>
  )
}

/** Google Workspace has no groups claim over OIDC (SSOStates). */
export function GroupMappingNote({ preset, name }: { preset: string; name: string }) {
  return (
    <div className="note i">
      <ProviderMark preset={preset} />
      <span>
        <T msg={sm.groupMappingUnavailable} values={{ name, google: preset === 'google' ? 'yes' : 'no', b: (c) => <b>{c}</b> }} />
      </span>
    </div>
  )
}

function AccessCard({ d, f, set, preset, onEnforce, onUnenforce }: CardProps & { preset: Preset; onEnforce: () => void; onUnenforce: () => void }) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [domain, setDomain] = useState('')
  const [domainError, setDomainError] = useState<string>()
  const ws = d.workspaces.find((w) => w.id === f.jit.workspaceId) ?? d.workspaces[0]
  const addDomain = () => {
    const v = domain.trim().toLowerCase().replace(/^@/, '')
    if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(v)) {
      setDomainError(t(ss.domainInvalid))
      return
    }
    if (!f.allowedDomains.includes(v)) set({ allowedDomains: [...f.allowedDomains, v] })
    setDomain('')
    setDomainError(undefined)
    setAdding(false)
  }
  const enforcedOn = d.enforced.length > 0
  const domains = d.domains.length ? d.domains : f.allowedDomains
  const owners = d.breakGlassOwners
  return (
    <section className="card" aria-labelledby="access-h">
      <CardSection title={<span id="access-h">{t(ss.accessTitle)}</span>} caption={t(ss.accessCaption)} />
      <div className="pad col gap-3.5">
        <div className="lbl">
          <span id="domains-l">{t(ss.allowedEmailDomains)}</span>
          <div className="row gap-1.5" role="group" aria-labelledby="domains-l">
            {f.allowedDomains.map((x) => (
              <span key={x} className="chip">
                {x}
                <button type="button" aria-label={t(ss.removeDomain, { domain: x })} onClick={() => set({ allowedDomains: f.allowedDomains.filter((y) => y !== x) })}>
                  ×
                </button>
              </span>
            ))}
            {adding ? (
              <span className="row flex-nowrap gap-1.5">
                <input
                  className="in h-[30px] w-[160px]"
                  value={domain}
                  autoFocus
                  aria-label={t(ss.domainToAdd)}
                  placeholder={'acme.io'}
                  onChange={(e) => setDomain(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      addDomain()
                    }
                    if (e.key === 'Escape') setAdding(false)
                  }}
                  aria-invalid={domainError ? true : undefined}
                />
                <Button size="sm" onClick={addDomain}>
                  {t(ss.add)}
                </Button>
              </span>
            ) : (
              <Button size="sm" onClick={() => setAdding(true)}>
                {t(ss.addDomain)}
              </Button>
            )}
          </div>
          {domainError ? (
            <span className="err" role="alert">
              {domainError}
            </span>
          ) : null}
          <span className="k font-medium">{f.allowedDomains.length ? t(ss.domainsRejected) : t(ss.domainsAny)}</span>
        </div>
        <div className="grid grid-cols-[auto_1fr_1fr] items-end gap-3.5 max-md:grid-cols-1">
          <label className="row flex-nowrap gap-3 pb-2">
            <input type="checkbox" className="tg" role="switch" checked={f.jit.enabled} onChange={(e) => set({ jit: { ...f.jit, enabled: e.target.checked } })} />
            <span className="font-semibold">{t(ss.jitToggle)}</span>
          </label>
          <label className="lbl">
            {t(ss.jitWorkspace)}
            <select className="in" value={f.jit.workspaceId} onChange={(e) => set({ jit: { ...f.jit, workspaceId: e.target.value } })}>
              {d.workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            {t(ss.jitRole)}
            <select className="in" value={f.jit.roleKey} onChange={(e) => set({ jit: { ...f.jit, roleKey: e.target.value } })}>
              {(ws?.roles ?? [])
                .filter((r) => r.key !== 'owner')
                .map((r) => (
                  <option key={r.key} value={r.key} disabled={!r.grantable && r.key !== f.jit.roleKey}>
                    {r.grantable ? roleName(r) : t(ss.roleNotGrantable, { role: roleName(r) })}
                  </option>
                ))}
            </select>
          </label>
        </div>
      </div>
      {preset.groupsClaim ? (
        <>
          <div className="sec border-t border-line-2">
            <div className="col gap-0.5">
              <h3 className="h2 text-[14px]">{t(ss.mappingsTitle)}</h3>
              <span className="k">{t(ss.mappingsCaption)}</span>
            </div>
            <Button
              size="sm"
              onClick={() =>
                set({
                  groupMappings: [...f.groupMappings, { group: '', label: '', workspaceId: d.workspaces[0]?.id ?? '', roleKey: 'viewer', lists: [] }],
                })
              }
            >
              {t(ss.addMapping)}
            </Button>
          </div>
          {f.groupMappings.length === 0 ? (
            <div className="pad">
              <span className="k">{t(ss.mappingsEmpty, { ids: preset.groupIds ? 'yes' : 'no' })}</span>
            </div>
          ) : (
            <MappingsTable d={d} f={f} set={set} preset={preset} />
          )}
        </>
      ) : (
        <div className="pad border-t border-line-2">
          <GroupMappingNote preset={preset.key} name={preset.name} />
        </div>
      )}
      <div className="pad col gap-2.5 border-t border-line-2">
        {/* eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text */}
        <label className="row flex-nowrap gap-3">
          <input
            type="checkbox"
            className="tg"
            role="switch"
            checked={enforcedOn}
            onChange={(e) => {
              if (d.status !== 'enabled' || domains.length === 0) {
                toast({
                  kind: 'info',
                  title: domains.length === 0 ? t(ss.enforceNeedsDomain) : t(ss.enforceNeedsEnable, { name: d.name }),
                  detail: domains.length === 0 ? t(ss.enforceNeedsDomainDetail) : t(ss.enforceNeedsEnableDetail),
                })
                return
              }
              if (e.target.checked) onEnforce()
              else onUnenforce()
            }}
            aria-describedby="enforce-hint"
          />
          <span className="col gap-px">
            <span className="font-semibold">{t(ss.enforceToggle, { has: domains.length ? 'yes' : 'no', domains: domains.join(', ') })}</span>
            <span className="k" id="enforce-hint">
              {enforcedOn ? t(ss.enforcedFor, { domains: d.enforced.join(', ') }) : t(ss.enforceTurnsOff)}{' '}
              {owners.length ? (
                <T
                  msg={ss.breakGlassOwners}
                  values={{
                    names: (
                      <>
                        {owners.map((o, i) => (
                          <span key={o.email}>
                            {i > 0 ? ', ' : null}
                            <b>{o.name}</b>
                          </span>
                        ))}
                      </>
                    ),
                  }}
                />
              ) : (
                t(ss.breakGlassNone)
              )}
            </span>
          </span>
        </label>
      </div>
    </section>
  )
}

function shortGroup(g: string) {
  return g.length === 36 && g.split('-').length === 5 ? `${g.slice(0, 8)}-…-${g.slice(33)}` : g
}

function MappingsTable({ d, f, set, preset }: CardProps & { preset: Preset }) {
  const t = useT()
  const update = (i: number, patch: Partial<Mapping>) => set({ groupMappings: f.groupMappings.map((m, j) => (j === i ? { ...m, ...patch } : m)) })
  const known = new Set(d.groupMappings.map((m) => `${m.group}|${m.workspaceId}`))
  return (
    <AccessTable caption={t(ss.mappingsTable)}>
      <thead>
        <tr>
          <Th>{t(ss.colGroup)}</Th>
          <Th>{t(ss.colLabel)}</Th>
          <Th>{t(ss.colWorkspace)}</Th>
          <Th>{t(ss.colRole)}</Th>
          <Th>{t(ss.colScope)}</Th>
          <Th>
            <span className="vh">{t(ss.colRemove)}</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {f.groupMappings.map((m, i) => {
          const ws = d.workspaces.find((w) => w.id === m.workspaceId)
          const saved = known.has(`${m.group}|${m.workspaceId}`)
          const label = m.label || shortGroup(m.group) || t(ss.newMapping)
          return (
            <tr key={i}>
              <Td>
                {saved ? (
                  <span className="mono" title={m.group}>
                    {shortGroup(m.group)}
                  </span>
                ) : (
                  <input
                    className="in mono w-[220px]"
                    value={m.group}
                    placeholder={preset.groupIds ? t(ss.groupIdPlaceholder) : t(ss.groupNamePlaceholder)}
                    aria-label={t(ss.colGroup)}
                    onChange={(e) => update(i, { group: e.target.value.trim() })}
                  />
                )}
              </Td>
              <Td>
                <input className="in w-[150px]" value={m.label ?? ''} aria-label={t(ss.labelFor, { group: shortGroup(m.group) })} onChange={(e) => update(i, { label: e.target.value })} />
              </Td>
              <Td>
                <select className="in w-[190px]" value={m.workspaceId} aria-label={t(ss.workspaceFor, { group: label })} onChange={(e) => update(i, { workspaceId: e.target.value, lists: [] })}>
                  {d.workspaces.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </Td>
              <Td>
                <select className="in w-[150px]" value={m.roleKey} aria-label={t(ss.roleFor, { group: label })} onChange={(e) => update(i, { roleKey: e.target.value })}>
                  {(ws?.roles ?? [])
                    .filter((r) => r.key !== 'owner')
                    .map((r) => (
                      <option key={r.key} value={r.key} disabled={!r.grantable && r.key !== m.roleKey} title={r.reason}>
                        {roleName(r)}
                      </option>
                    ))}
                </select>
              </Td>
              <Td>
                <ScopePicker lists={ws?.lists ?? []} value={m.lists ?? []} group={label} onChange={(lists) => update(i, { lists })} />
              </Td>
              <Td>
                <Button size="sm" variant="danger" onClick={() => set({ groupMappings: f.groupMappings.filter((_, j) => j !== i) })}>
                  {t(ss.removeMapping)}
                </Button>
              </Td>
            </tr>
          )
        })}
      </tbody>
    </AccessTable>
  )
}

/**
 * A mapping's scope as the board writes it ("all lists", "Weekly Digest
 * only", "2 lists"): the text opens a picker of the workspace's lists, and
 * no list picked means every list.
 */
function ScopePicker({ lists, value, group, onChange }: { lists: Schemas['SSOListChoice'][]; value: string[]; group: string; onChange: (lists: string[]) => void }) {
  const t = useT()
  const picked = lists.filter((l) => value.includes(l.id))
  const text = t(ss.mappingScope, { count: picked.length, name: picked[0]?.name ?? '' })
  if (lists.length === 0) return <span className="k">{text}</span>
  const toggle = (id: string) => onChange(value.includes(id) ? value.filter((x) => x !== id) : [...value, id])
  return (
    <Popover
      align="start"
      label={t(ss.listsFor, { group })}
      className="w-[280px] max-w-[calc(100vw-24px)]"
      trigger={
        <button
          type="button"
          className="k inline-flex min-h-6 cursor-pointer items-center border-0 bg-transparent p-0 text-left hover:text-ink hover:underline"
          aria-label={t(ss.scopeFor, { group, scope: text })}
        >
          {text}
        </button>
      }
    >
      <div className="flex flex-col gap-1 p-1">
        <span className="mh">{t(ss.listsHeading)}</span>
        {lists.map((l) => (
          <label key={l.id} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground">
            <input type="checkbox" checked={value.includes(l.id)} onChange={() => toggle(l.id)} />
            <span className="flex-1">{l.name}</span>
          </label>
        ))}
        <span className="k border-t border-line-2 px-2.5 pt-2 pb-1">{t(ss.listsHint)}</span>
      </div>
    </Popover>
  )
}

const toneOf = (t: string): Tone => (['ok', 'warn', 'bad', 'info'].includes(t) ? (t as Tone) : 'neutral')

/** "Acme Newsletter: Admin → Editor from group…" with the change in bold and "amr" in mono. */
function OutcomeText({ o }: { o: Schemas['SSOTestOutcome'] }) {
  let parts: ReactNode = o.detail
  if (o.strong && o.detail.includes(o.strong)) {
    const [a, b] = o.detail.split(o.strong, 2) as [string, string]
    parts = (
      <>
        {a}
        <b>{o.strong}</b>
        {b}
      </>
    )
  } else if (o.code && o.detail.startsWith(o.code)) {
    parts = (
      <>
        <span className="mono">{o.code}</span>
        {o.detail.slice(o.code.length)}
      </>
    )
  }
  return <span className="wrap">{parts}</span>
}

function TestCard({
  d,
  preset,
  running,
  dropping,
  onRun,
  onDropMappings,
}: {
  d: Detail
  preset: Preset
  running: boolean
  dropping: boolean
  onRun: () => void
  onDropMappings: () => void
}) {
  const t = useT()
  const test = d.test
  const blocked = d.enableBlockers.length ? t(ss.beforeTest, { blockers: d.enableBlockers.join(' ') }) : undefined
  const missing = test && !test.error && test.groupsMissing
  const claims = test?.claims && missing ? test.claims.replace(/\n}$/, '\n  // no "groups" claim\n}') : test?.claims
  // The default role in the app language (built-in roles are translated).
  const defaultRole = d.jit.roleName ? roleName({ key: d.jit.roleKey, name: d.jit.roleName }) : roleName({ key: 'viewer', name: 'Viewer' })
  if (missing)
    // SSOStates "Groups claim missing": the claims, why the mappings don't
    // apply and how to send the claim.
    return (
      <section className="card" aria-labelledby="test-h">
        <CardSection title={<span id="test-h">{t(ss.groupsMissingTitle)}</span>} caption={t(ss.groupsMissingCaption)} actions={<Badge tone="warn">{t(ss.checkIdp)}</Badge>} />
        <div className="pad col gap-3">
          <pre className="code" aria-label={t(ss.claimsReturnedLabel, { ago: ago(test.at) })}>
            {claims}
          </pre>
          <Note tone="warn" label={t(ss.mappingsInactive)}>
            {t(ss.mappingsInactiveBody, { count: d.groupMappings.length, role: defaultRole })} {richHint(preset.groupsMissingHint)}
          </Note>
          <div className="row gap-2">
            <Button size="sm" variant="primary" loading={running} disabledReason={blocked} onClick={onRun}>
              {t(ss.runTestAgain)}
            </Button>
            <Button size="sm" loading={dropping} onClick={onDropMappings}>
              {t(ss.saveWithoutMappings)}
            </Button>
          </div>
        </div>
      </section>
    )
  return (
    <section className="card" aria-labelledby="test-h">
      <CardSection title={<span id="test-h">{t(ss.testTitle)}</span>} caption={t(ss.testCaption)} />
      <div className="pad col gap-3.5">
        <div className="row gap-2">
          <Button variant="primary" loading={running} disabledReason={blocked} onClick={onRun}>
            {t(ss.runTest)}
          </Button>
          <span className="k">{t(ss.testHint, { idp: preset.shortName })}</span>
        </div>
        {test?.error ? (
          <Note tone="bad" label={t(ss.testFailed)}>
            {test.error} ({ago(test.at)})
          </Note>
        ) : null}
        {test && !test.error ? (
          <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
            <div className="col gap-1.5">
              <span className="text-[12px] font-semibold text-ink-2">{t(ss.claimsReturned, { ago: ago(test.at) })}</span>
              <pre className="code">{claims}</pre>
            </div>
            <div className="col gap-2">
              <span className="text-[12px] font-semibold text-ink-2">{t(ss.wouldDo)}</span>
              <AccessTable caption={t(ss.wouldDo)}>
                <thead>
                  <tr>
                    <Th>{t(ss.colOutcome)}</Th>
                    <Th>{t(ss.colDetail)}</Th>
                  </tr>
                </thead>
                <tbody>
                  {test.outcome.map((o, i) => (
                    <tr key={i}>
                      <Td>
                        <Badge tone={toneOf(o.tone)}>{o.label}</Badge>
                      </Td>
                      <Td>
                        <OutcomeText o={o} />
                      </Td>
                    </tr>
                  ))}
                </tbody>
              </AccessTable>
            </div>
          </div>
        ) : null}
        {!test ? (
          <span className="k row flex-nowrap items-start gap-1.5">
            <Icon name="info" size={14} className="mt-px shrink-0" />
            <span>{t(ss.testIntro, { name: preset.name })}</span>
          </span>
        ) : null}
      </div>
    </section>
  )
}

function TurnOffEnforcement({ d, onClose, onDone }: { d: Detail; onClose: () => void; onDone: (d: Detail) => void }) {
  const t = useT()
  const set = $api.useMutation('put', '/api/v1/admin/sso/{providerId}/enforcement')
  const invalidate = useInvalidateSSO()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={t(ss.turnOffBadge)}
      title={t(ss.turnOffTitle, { domains: d.enforced.join(', ') })}
      description={t(ss.turnOffBody, { domains: d.enforced.join(', '), name: d.name })}
      confirmLabel={t(ss.turnOffConfirm)}
      onConfirm={async () => {
        const r = await set.mutateAsync({ params: { path: { providerId: d.id } }, body: { domains: [], confirm: true } })
        onDone(r)
        await invalidate()
        toast({ kind: 'success', title: t(ss.turnedOff), detail: t(ss.turnedOffDetail, { domains: d.enforced.join(', ') }) })
      }}
    />
  )
}

function SetupSkeleton() {
  const t = useT()
  return (
    <div className="col gap-5" aria-busy="true">
      <span className="vh" role="status">
        {t(ss.loading)}
      </span>
      {[110, 300, 220].map((h, i) => (
        <section key={i} className="card">
          <div className="sec">
            <Skeleton w={160} h={14} />
          </div>
          <div className="pad">
            <Skeleton w="100%" h={h} r={8} />
          </div>
        </section>
      ))}
    </div>
  )
}
