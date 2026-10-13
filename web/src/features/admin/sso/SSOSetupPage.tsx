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
import { ago, andList, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { useInvalidateSSO } from '../queries'
import { EnforcementDialog } from './EnforcementDialog'
import { richHint, shortURL } from './hints'
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

function statusWords(d: Detail) {
  switch (d.status) {
    case 'enabled':
      return 'enabled'
    case 'draft':
      return 'setup in progress'
    case 'disabled':
      return 'disabled'
  }
  return 'needs attention'
}

function ExistingProvider({ providerId, test }: { providerId: string; test?: 'done' }) {
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
    <AdminPage documentTitle={q.data?.name ?? 'SSO provider'} subtitle={q.data ? `${q.data.name} · ${statusWords(q.data)}` : undefined}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load this SSO provider" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
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
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const list = $api.useQuery('get', '/api/v1/admin/sso')
  const create = $api.useMutation('post', '/api/v1/admin/sso')
  const invalidate = useInvalidateSSO()
  return (
    <AdminPage documentTitle="Add SSO provider" subtitle="New SSO provider">
      <section className="card" aria-labelledby="preset-h">
        <CardSection title={<span id="preset-h">1 · Preset</span>} caption="Pick your identity provider" />
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
                      toast({ kind: 'error', title: "Couldn't add the provider", detail: errorMessage(e) })
                    }
                  }}
                >
                  <ProviderMark preset={p.key} />
                  {p.name}
                </Button>
              ))}
            </div>
          )}
          <span className="k">Presets fill the issuer pattern, scopes and claim names. Everything stays editable, and nothing is live until you enable the provider.</span>
        </div>
      </section>
    </AdminPage>
  )
}

function SetupForm({ d }: { d: Detail }) {
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
            toast({ kind: 'success', title: 'Group mappings removed', detail: `${r.name} saved · JIT default role applies` })
          })
        }
      />
      {error ? (
        <ErrorPanel title="Not saved" actions={<Button onClick={() => setError(undefined)}>Dismiss</Button>}>
          {error}
        </ErrorPanel>
      ) : null}
      <div className="row justify-between gap-2.5">
        <span className="k">
          {current.status === 'enabled' ? 'Changes apply at the next sign-in. People already signed in stay signed in.' : 'Nothing is live until you enable the provider. Users keep their current sign-in until then.'}
        </span>
        <div className="row gap-2">
          {current.status === 'enabled' ? (
            <Button
              variant="primary"
              loading={save.isPending}
              onClick={() =>
                void run(async () => {
                  await persist()
                  toast({ kind: 'success', title: `${current.name} saved` })
                })
              }
            >
              Save changes
            </Button>
          ) : (
            <>
              <Button
                loading={save.isPending && !enable.isPending}
                onClick={() =>
                  void run(async () => {
                    await persist()
                    toast({ kind: 'success', title: 'Draft saved', detail: 'nothing is live yet' })
                  })
                }
              >
                Save as draft
              </Button>
              <Button
                variant="primary"
                loading={enable.isPending}
                onClick={() =>
                  void run(async () => {
                    const r = dirty ? await persist() : current
                    if (r.enableBlockers.length) throw new Error(`Before enabling: ${r.enableBlockers.join(' ')}`)
                    const e = await enable.mutateAsync(path)
                    setSaved(e)
                    setF(toForm(e))
                    await invalidate()
                    toast({ kind: 'success', title: `${e.name} enabled`, detail: 'its button is on the sign-in page' })
                  })
                }
              >
                Enable {current.name}
              </Button>
            </>
          )}
        </div>
      </div>
      {enforcing ? <EnforcementDialog d={current} onClose={() => setEnforcing(false)} onDone={(r) => { setSaved(r); setF(toForm(r)) }} /> : null}
      {unenforcing ? (
        <TurnOffEnforcement d={current} onClose={() => setUnenforcing(false)} onDone={(r) => { setSaved(r); setF(toForm(r)) }} />
      ) : null}
      <UnsavedChangesGuard dirty={dirty} title="Leave without saving?" description={`You changed ${current.name}'s settings. Nothing is saved until you save.`} onSave={() => persist()} />
    </>
  )
}

function useList() {
  return $api.useQuery('get', '/api/v1/admin/sso')
}

type CardProps = { d: Detail; f: Form; set: (p: Partial<Form>) => void }

function PresetCard({ d, f, set }: CardProps) {
  const presets = useList().data?.presets ?? []
  const live = d.status === 'enabled'
  const name = presets.find((p) => p.key === f.preset)?.name ?? d.presetInfo.name
  return (
    <section className="card" aria-labelledby="preset-h">
      <CardSection title={<span id="preset-h">1 · Preset</span>} caption={`${name} selected`} />
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
                disabledReason={!on && live ? `Disable ${d.name} before switching presets, or add another provider` : undefined}
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
        <span className="k">Presets fill the issuer pattern, scopes and claim names. Everything stays editable.</span>
      </div>
    </section>
  )
}

function presetIssuer(presets: Preset[], key: string) {
  return presets.find((p) => p.key === key)?.issuer ?? ''
}

/** A date without a time ("Apr 10, 2027"), the same in every timezone. */
function dateOnly(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { timeZone: 'UTC', month: 'short', day: 'numeric', year: 'numeric' })
}

function ConnectionCard({ d, f, set, preset, onDiscovered }: CardProps & { preset: Preset; onDiscovered: (d: Detail) => void }) {
  const id = useId()
  const discover = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/discovery')
  const [error, setError] = useState<string>()
  const disc = d.discovery
  const failed = disc && !disc.ok
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
          title={<span id="conn-h">Discovery failed</span>}
          caption={d.status === 'enabled' ? 'Step 2 · the saved issuer keeps working until a new one passes' : 'Step 2 · issuer unreachable or wrong URL'}
          actions={d.status === 'enabled' ? <Badge tone="warn">Still live</Badge> : <Badge tone="bad">Not configured</Badge>}
        />
        <div className="pad col gap-3">
          <label className="lbl">
            Issuer URL
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
            <Badge tone="bad">{disc.status ? String(disc.status) : 'Error'}</Badge>
            <span className="col gap-1">
              <span>
                {disc.url ? <span className="mono">{shortURL(disc.url)}</span> : 'The issuer'} {disc.error}
                {disc.ms && disc.status ? ` in ${disc.ms} ms` : ''}.
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
              Test discovery again
            </Button>
            <ButtonAnchor size="sm" href={preset.docsUrl}>
              {preset.docsLabel}
            </ButtonAnchor>
          </div>
        </div>
      </section>
    )
  const secretHint = d.secret.set
    ? [
        d.secret.setAt ? `Saved ${usDate(d.secret.setAt).replace(/, \d{4}$/, '')}${d.secret.setBy ? ` by ${d.secret.setBy}` : ''}` : 'Saved',
        d.secret.expiresOn ? `expires in ${preset.key === 'entra' ? 'Entra' : preset.shortName} on ${dateOnly(d.secret.expiresOn)}` : '',
      ]
        .filter(Boolean)
        .join(' · ') + '. Secrets can be replaced but never read back.'
    : 'Paste the secret from the ' + preset.app + '. It is encrypted and never shown again.'
  return (
    <section className="card" aria-labelledby="conn-h">
      <CardSection title={<span id="conn-h">2 · Connection</span>} />
      <div className="pad col gap-3.5">
        <div className="lbl">
          <span>
            Redirect URI <span className="k font-medium">· add this to the {preset.app}</span>
          </span>
          <div className="row flex-nowrap gap-2">
            <input className="in mono flex-1 bg-ground" value={d.redirectUri} readOnly aria-label="Redirect URI" aria-describedby={`${id}-redir`} />
            <Button
              onClick={async () => {
                const ok = await copyText(d.redirectUri)
                toast(ok ? { kind: 'success', title: 'Redirect URI copied' } : { kind: 'error', title: "Couldn't copy", detail: 'select it and copy' })
              }}
            >
              Copy
            </Button>
          </div>
          <span className="k font-medium" id={`${id}-redir`}>
            Built from the public URL. If the URL changes, update the app registration.
          </span>
        </div>
        <div className="lbl">
          <label htmlFor={`${id}-issuer`}>Issuer URL</label>
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
              Test discovery
            </Button>
          </div>
          <span className="k font-medium" id={`${id}-disc`}>
            {disc?.ok ? (
              <span className="row gap-1.5">
                <Badge tone="ok">OK</Badge> Discovery document found {ago(disc.checkedAt)} · {andList(disc.endpoints)} endpoints · signing keys cached
              </span>
            ) : !disc ? (
              `Enter the issuer, like ${preset.issuerPattern}, then test discovery.`
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
            Client ID
            <input className="in mono" value={f.clientId} onChange={(e) => set({ clientId: e.target.value })} spellCheck={false} autoComplete="off" />
          </label>
          <div className="lbl">
            <label htmlFor={`${id}-secret`}>
              Client secret <span className="k font-medium">· write-only</span>
            </label>
            {f.replacing ? (
              <div className="row flex-nowrap gap-2">
                <input
                  id={`${id}-secret`}
                  className="in mono flex-1"
                  type="password"
                  value={f.clientSecret}
                  placeholder={d.secret.set ? 'New secret' : 'Paste the client secret'}
                  autoComplete="new-password"
                  onChange={(e) => set({ clientSecret: e.target.value })}
                />
                {d.secret.set ? <Button onClick={() => set({ replacing: false, clientSecret: '' })}>Keep</Button> : null}
              </div>
            ) : (
              <div className="row flex-nowrap gap-2">
                <input id={`${id}-secret`} className="in flex-1 bg-ground" type="password" value="••••••••••••••••••••" readOnly aria-describedby={`${id}-sec-hint`} />
                <Button onClick={() => set({ replacing: true })}>Replace</Button>
              </div>
            )}
            {f.replacing && f.clientSecret ? (
              <span className="row flex-nowrap gap-2">
                <label className="k font-medium" htmlFor={`${id}-exp`}>
                  Expires at the IdP on
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
      <span id="scopes-l">Scopes</span>
      <div className="row gap-1.5" role="group" aria-labelledby="scopes-l">
        {f.scopes.map((s) => (
          <span key={s} className="chip">
            {s}
            {fixed.includes(s) ? null : (
              <button type="button" aria-label={`Remove scope ${s}`} onClick={() => set({ scopes: f.scopes.filter((x) => x !== s) })}>
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
              aria-label="Scope to add"
              placeholder="groups"
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
              Add
            </Button>
          </span>
        ) : (
          <Button size="sm" onClick={() => setAdding(true)}>
            Add scope
          </Button>
        )}
      </div>
      <span className="k font-medium">{preset.groupsHint}</span>
    </div>
  )
}

function ClaimsCard({ d, f, set, preset }: CardProps & { preset: Preset }) {
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
      <CardSection title={<span id="claims-h">3 · Claim mapping</span>} caption="Which claims fill the account" />
      <AccessTable caption="Claim mapping">
        <thead>
          <tr>
            <Th>ReputeCast field</Th>
            <Th>Claim name</Th>
            <Th>Sample from last test</Th>
            <Th>
              <span className="vh">Result</span>
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
                    aria-label={`Claim for ${r.label.replace('_', ' ')}`}
                    onChange={(e) => set({ claims: { ...f.claims, [r.field]: e.target.value } })}
                  />
                </Td>
                <Td>
                  {!s ? (
                    <span className="k">Run a test sign-in</span>
                  ) : s.found ? (
                    <>
                      <span className="mono">{s.value}</span>
                      {r.field === 'groups' && preset.groupIds ? <span className="k"> object IDs</span> : null}
                    </>
                  ) : (
                    <span className="k">{s.missing ?? `not sent by ${preset.shortName}`}</span>
                  )}
                </Td>
                <Td>
                  {!s ? null : s.found ? (
                    <Badge tone="ok">{r.field === 'groups' && s.count ? `Found · ${s.count}` : 'Found'}</Badge>
                  ) : (
                    <span className="row flex-nowrap gap-1.5">
                      <Badge tone="warn">Missing</Badge>
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
              {preset.shortName} sends group <b>object IDs</b>, not names. Paste the ID in a mapping and give it a readable label.
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
        <b>Group mapping is unavailable for {name}.</b>{' '}
        {preset === 'google'
          ? "Google doesn't send a groups claim over OIDC. Map people by email domain (JIT default role) or assign roles individually in Team."
          : 'This identity provider sends no groups claim. Map people by email domain (JIT default role) or assign roles individually in Team.'}
      </span>
    </div>
  )
}

function AccessCard({ d, f, set, preset, onEnforce, onUnenforce }: CardProps & { preset: Preset; onEnforce: () => void; onUnenforce: () => void }) {
  const [adding, setAdding] = useState(false)
  const [domain, setDomain] = useState('')
  const [domainError, setDomainError] = useState<string>()
  const ws = d.workspaces.find((w) => w.id === f.jit.workspaceId) ?? d.workspaces[0]
  const addDomain = () => {
    const v = domain.trim().toLowerCase().replace(/^@/, '')
    if (!/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(v)) {
      setDomainError('Enter a domain like acme.io.')
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
      <CardSection title={<span id="access-h">4 · Access</span>} caption="Domains, just-in-time accounts, group mapping, enforcement" />
      <div className="pad col gap-3.5">
        <div className="lbl">
          <span id="domains-l">Allowed email domains</span>
          <div className="row gap-1.5" role="group" aria-labelledby="domains-l">
            {f.allowedDomains.map((x) => (
              <span key={x} className="chip">
                {x}
                <button type="button" aria-label={`Remove ${x}`} onClick={() => set({ allowedDomains: f.allowedDomains.filter((y) => y !== x) })}>
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
                  aria-label="Domain to allow"
                  placeholder="acme.io"
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
                  Add
                </Button>
              </span>
            ) : (
              <Button size="sm" onClick={() => setAdding(true)}>
                Add domain
              </Button>
            )}
          </div>
          {domainError ? (
            <span className="err" role="alert">
              {domainError}
            </span>
          ) : null}
          <span className="k font-medium">{f.allowedDomains.length ? 'Sign-ins from other domains are rejected and logged.' : 'Any domain can sign in; add one to limit sign-ins to your organization.'}</span>
        </div>
        <div className="grid grid-cols-[auto_1fr_1fr] items-end gap-3.5 max-md:grid-cols-1">
          <label className="row flex-nowrap gap-3 pb-2">
            <input type="checkbox" className="tg" role="switch" checked={f.jit.enabled} onChange={(e) => set({ jit: { ...f.jit, enabled: e.target.checked } })} />
            <span className="font-semibold">Create accounts automatically (JIT)</span>
          </label>
          <label className="lbl">
            Default workspace
            <select className="in" value={f.jit.workspaceId} onChange={(e) => set({ jit: { ...f.jit, workspaceId: e.target.value } })}>
              {d.workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            Default role
            <select className="in" value={f.jit.roleKey} onChange={(e) => set({ jit: { ...f.jit, roleKey: e.target.value } })}>
              {(ws?.roles ?? [])
                .filter((r) => r.key !== 'owner')
                .map((r) => (
                  <option key={r.key} value={r.key} disabled={!r.grantable && r.key !== f.jit.roleKey}>
                    {r.name}
                    {!r.grantable ? ' (you can’t grant it)' : ''}
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
              <h3 className="h2 text-[14px]">Group → workspace → role</h3>
              <span className="k">Re-applied at every sign-in. Direct role assignments are overridden by a matching mapping.</span>
            </div>
            <Button
              size="sm"
              onClick={() =>
                set({
                  groupMappings: [...f.groupMappings, { group: '', label: '', workspaceId: d.workspaces[0]?.id ?? '', roleKey: 'viewer', lists: [] }],
                })
              }
            >
              Add mapping
            </Button>
          </div>
          {f.groupMappings.length === 0 ? (
            <div className="pad">
              <span className="k">
                No mappings yet. Add one per {preset.groupIds ? 'group object ID' : 'group'}: everyone in the group gets the role in that workspace at each sign-in.
              </span>
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
                  title: domains.length === 0 ? 'Add an allowed domain first' : `Enable ${d.name} first`,
                  detail: domains.length === 0 ? 'enforcement applies to the provider’s domains' : 'enforcement needs a live provider',
                })
                return
              }
              if (e.target.checked) onEnforce()
              else onUnenforce()
            }}
            aria-describedby="enforce-hint"
          />
          <span className="col gap-px">
            <span className="font-semibold">Enforce SSO for {domains.length ? domains.join(', ') : 'your domain'}</span>
            <span className="k" id="enforce-hint">
              {enforcedOn ? `Password sign-in is off for ${d.enforced.join(', ')}. ` : 'Turns off password sign-in for these users. '}
              {owners.length ? (
                <>
                  Break-glass Owners keep password + two-factor:{' '}
                  {owners.map((o, i) => (
                    <span key={o.email}>
                      {i > 0 ? ', ' : null}
                      <b>{o.name}</b>
                    </span>
                  ))}
                  .
                </>
              ) : (
                'No Owner at these domains keeps a break-glass sign-in.'
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
  const update = (i: number, patch: Partial<Mapping>) => set({ groupMappings: f.groupMappings.map((m, j) => (j === i ? { ...m, ...patch } : m)) })
  const known = new Set(d.groupMappings.map((m) => `${m.group}|${m.workspaceId}`))
  return (
    <AccessTable caption="Group mappings">
      <thead>
        <tr>
          <Th>IdP group</Th>
          <Th>Label</Th>
          <Th>Workspace</Th>
          <Th>Role</Th>
          <Th>Scope</Th>
          <Th>
            <span className="vh">Remove</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {f.groupMappings.map((m, i) => {
          const ws = d.workspaces.find((w) => w.id === m.workspaceId)
          const saved = known.has(`${m.group}|${m.workspaceId}`)
          const label = m.label || shortGroup(m.group) || 'new mapping'
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
                    placeholder={preset.groupIds ? 'Group object ID' : 'Group name'}
                    aria-label="IdP group"
                    onChange={(e) => update(i, { group: e.target.value.trim() })}
                  />
                )}
              </Td>
              <Td>
                <input className="in w-[150px]" value={m.label ?? ''} aria-label={`Label for group ${shortGroup(m.group)}`} onChange={(e) => update(i, { label: e.target.value })} />
              </Td>
              <Td>
                <select className="in w-[190px]" value={m.workspaceId} aria-label={`Workspace for group ${label}`} onChange={(e) => update(i, { workspaceId: e.target.value, lists: [] })}>
                  {d.workspaces.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </Td>
              <Td>
                <select className="in w-[150px]" value={m.roleKey} aria-label={`Role for group ${label}`} onChange={(e) => update(i, { roleKey: e.target.value })}>
                  {(ws?.roles ?? [])
                    .filter((r) => r.key !== 'owner')
                    .map((r) => (
                      <option key={r.key} value={r.key} disabled={!r.grantable && r.key !== m.roleKey} title={r.reason}>
                        {r.name}
                      </option>
                    ))}
                </select>
              </Td>
              <Td>
                <ScopePicker lists={ws?.lists ?? []} value={m.lists ?? []} group={label} onChange={(lists) => update(i, { lists })} />
              </Td>
              <Td>
                <Button size="sm" variant="danger" onClick={() => set({ groupMappings: f.groupMappings.filter((_, j) => j !== i) })}>
                  Remove
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
  const picked = lists.filter((l) => value.includes(l.id))
  const text = picked.length === 0 ? 'all lists' : picked.length === 1 ? `${picked[0]!.name} only` : `${picked.length} lists`
  if (lists.length === 0) return <span className="k">{text}</span>
  const toggle = (id: string) => onChange(value.includes(id) ? value.filter((x) => x !== id) : [...value, id])
  return (
    <Popover
      align="start"
      label={`Lists for group ${group}`}
      className="w-[280px] max-w-[calc(100vw-24px)]"
      trigger={
        <button
          type="button"
          className="k inline-flex min-h-6 cursor-pointer items-center border-0 bg-transparent p-0 text-left hover:text-ink hover:underline"
          aria-label={`Scope for group ${group}: ${text}`}
        >
          {text}
        </button>
      }
    >
      <div className="flex flex-col gap-1 p-1">
        <span className="mh">Lists the role applies to</span>
        {lists.map((l) => (
          <label key={l.id} className="chk items-center rounded-lg px-2.5 py-1.5 hover:bg-ground">
            <input type="checkbox" checked={value.includes(l.id)} onChange={() => toggle(l.id)} />
            <span className="flex-1">{l.name}</span>
          </label>
        ))}
        <span className="k border-t border-line-2 px-2.5 pt-2 pb-1">With no list picked, the role covers every list.</span>
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
  const t = d.test
  const blocked = d.enableBlockers.length ? `Before a test sign-in: ${d.enableBlockers.join(' ')}` : undefined
  const missing = t && !t.error && t.groupsMissing
  const claims = t?.claims && missing ? t.claims.replace(/\n}$/, '\n  // no "groups" claim\n}') : t?.claims
  if (missing)
    // SSOStates "Groups claim missing": the claims, why the mappings don't
    // apply and how to send the claim.
    return (
      <section className="card" aria-labelledby="test-h">
        <CardSection title={<span id="test-h">Groups claim missing</span>} caption="Step 5 · test sign-in returned no groups" actions={<Badge tone="warn">Check IdP</Badge>} />
        <div className="pad col gap-3">
          <pre className="code" aria-label={`Claims returned ${ago(t.at)}`}>
            {claims}
          </pre>
          <Note tone="warn" label="Mappings inactive">
            Your {d.groupMappings.length} group {d.groupMappings.length === 1 ? 'mapping' : 'mappings'} won&apos;t apply, so JIT users get the default role (
            {d.jit.roleName || 'Viewer'}) and existing roles stay as they are. {richHint(preset.groupsMissingHint)}
          </Note>
          <div className="row gap-2">
            <Button size="sm" variant="primary" loading={running} disabledReason={blocked} onClick={onRun}>
              Run test sign-in again
            </Button>
            <Button size="sm" loading={dropping} onClick={onDropMappings}>
              Save without group mapping
            </Button>
          </div>
        </div>
      </section>
    )
  return (
    <section className="card" aria-labelledby="test-h">
      <CardSection title={<span id="test-h">5 · Test sign-in</span>} caption="Debug view of the returned claims and the resulting accounts and roles" />
      <div className="pad col gap-3.5">
        <div className="row gap-2">
          <Button variant="primary" loading={running} disabledReason={blocked} onClick={onRun}>
            Run test sign-in
          </Button>
          <span className="k">Opens {preset.shortName} in a popup, shows the claims, signs nobody in and changes nothing.</span>
        </div>
        {t?.error ? (
          <Note tone="bad" label="Test failed">
            {t.error} ({ago(t.at)})
          </Note>
        ) : null}
        {t && !t.error ? (
          <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
            <div className="col gap-1.5">
              <span className="text-[12px] font-semibold text-ink-2">Claims returned · {ago(t.at)}</span>
              <pre className="code">{claims}</pre>
            </div>
            <div className="col gap-2">
              <span className="text-[12px] font-semibold text-ink-2">What this sign-in would do</span>
              <AccessTable caption="What this sign-in would do">
                <thead>
                  <tr>
                    <Th>Result</Th>
                    <Th>Detail</Th>
                  </tr>
                </thead>
                <tbody>
                  {t.outcome.map((o, i) => (
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
        {!t ? (
          <span className="k row flex-nowrap items-start gap-1.5">
            <Icon name="info" size={14} className="mt-px shrink-0" />
            <span>The test shows what {preset.name} returns for the account you sign in with and what a real sign-in would create or change.</span>
          </span>
        ) : null}
      </div>
    </section>
  )
}

function TurnOffEnforcement({ d, onClose, onDone }: { d: Detail; onClose: () => void; onDone: (d: Detail) => void }) {
  const set = $api.useMutation('put', '/api/v1/admin/sso/{providerId}/enforcement')
  const invalidate = useInvalidateSSO()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge="Domain enforcement"
      title={`Turn off enforcement for ${d.enforced.join(', ')}?`}
      description={`Password, passkey and social sign-in work again for ${d.enforced.join(', ')} accounts. ${d.name} keeps working for everyone who uses it.`}
      confirmLabel="Turn off enforcement"
      onConfirm={async () => {
        const r = await set.mutateAsync({ params: { path: { providerId: d.id } }, body: { domains: [], confirm: true } })
        onDone(r)
        await invalidate()
        toast({ kind: 'success', title: 'Enforcement turned off', detail: `password sign-in works again for ${d.enforced.join(', ')}` })
      }}
    />
  )
}

function SetupSkeleton() {
  return (
    <div className="col gap-5" aria-busy="true">
      <span className="vh" role="status">
        Loading the provider
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
