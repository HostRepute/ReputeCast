import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { problemMessage } from '@/features/connect/parts'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { cn } from '@/lib/cn'
import { count, dayMonthYear, money, monthName } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ReadOnlyNote } from './ReadOnlyNote'
import './senders.css'

type Settings = Schemas['SettingsProviders']
type Card = Schemas['SettingsProviderCard']
type Defaults = Schemas['SendingDefaults']

const settingsPath = '/api/v1/workspaces/{workspace}/settings/providers' as const

const THROTTLES: { key: string; label: string; color: string }[] = [
  { key: 'gmail', label: 'Gmail', color: 'var(--chart-gmail)' },
  { key: 'outlook', label: 'Outlook', color: 'var(--chart-outlook)' },
  { key: 'yahoo', label: 'Yahoo', color: 'var(--chart-yahoo)' },
  { key: 'icloud', label: 'iCloud', color: 'var(--chart-icloud)' },
]

/** 7.13 Settings › Providers (SettingsProviders): provider summaries and workspace-wide sending defaults. */
export function ProvidersSettingsPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', settingsPath, { params: { path: { workspace: slug } } })
  return (
    <Page title="Settings" documentTitle="Providers · Settings" tabs={settingsTabs(slug)}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load the provider settings" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.data ? (
        <SettingsBody key={JSON.stringify(q.data.defaults)} data={q.data} />
      ) : (
        <div className="flex flex-col gap-5" aria-busy="true">
          <span className="vh" role="status">
            Loading the provider settings…
          </span>
          <Skeleton w="100%" h={190} r={10} />
          <Skeleton w="100%" h={300} r={10} />
        </div>
      )}
    </Page>
  )
}

/** The defaults as the board writes them in the inputs. */
function formFrom(d: Defaults) {
  return {
    bounce: `${d.bounce.toFixed(1)} %`,
    complaint: `${d.complaint.toFixed(2)} %`,
    mailboxComplaint: `${d.mailboxComplaint.toFixed(2)} %`,
    providerError: `${trim(d.providerError)} %`,
    pauseScope: d.pauseScope,
    confirmOver: `${count(d.confirmOver)} recipients · type the count to confirm`,
    throttles: Object.fromEntries(THROTTLES.map((t) => [t.key, throttleText(d.throttles[t.key])])),
    suppressionSync: d.suppressionSync,
  }
}

function trim(v: number): string {
  return String(Number(v.toFixed(2)))
}

function throttleText(v: number | undefined): string {
  return v ? `${trim(v)} per second` : 'no limit'
}

/** The first number in a field ("2.0 %" → 2, "5,000 recipients…" → 5000); "no limit" is 0. */
export function numberIn(s: string): number | undefined {
  if (/^\s*(no limit|none|unlimited)?\s*$/i.test(s)) return 0
  const m = s.replace(/,/g, '').match(/\d+(\.\d+)?/)
  return m ? Number(m[0]) : undefined
}

function SettingsBody({ data }: { data: Settings }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('providers:manage', data.helpers)
  const initial = formFrom(data.defaults)
  const [f, setF] = useState(initial)
  const [error, setError] = useState<string>()
  const [rotating, setRotating] = useState(false)
  const save = $api.useMutation('put', settingsPath)
  const set = (patch: Partial<typeof f>) => setF((x) => ({ ...x, ...patch }))
  const submit = () => {
    const nums = { bounce: numberIn(f.bounce), complaint: numberIn(f.complaint), mailboxComplaint: numberIn(f.mailboxComplaint), providerError: numberIn(f.providerError), confirmOver: numberIn(f.confirmOver) }
    const bad = Object.entries(nums).find(([, v]) => v === undefined)
    if (bad) {
      setError('Each threshold needs a number, like 2.0 %.')
      return
    }
    const throttles: Record<string, number> = {}
    for (const t of THROTTLES) {
      const v = numberIn(f.throttles[t.key] ?? '')
      if (v === undefined) {
        setError(`Write the ${t.label} throttle as "5 per second", or "no limit".`)
        return
      }
      throttles[t.key] = v
    }
    setError(undefined)
    save.mutate(
      {
        params: { path: { workspace: slug } },
        body: {
          bounce: nums.bounce!,
          complaint: nums.complaint!,
          mailboxComplaint: nums.mailboxComplaint!,
          providerError: nums.providerError!,
          confirmOver: Math.round(nums.confirmOver!),
          pauseScope: f.pauseScope,
          throttles,
          suppressionSync: f.suppressionSync,
        },
      },
      {
        onSuccess: (next) => {
          qc.setQueryData($api.queryOptions('get', settingsPath, { params: { path: { workspace: slug } } }).queryKey, next)
          toast({ title: 'Sending defaults saved', detail: 'New campaigns start from them.' })
        },
        onError: (e) => setError(problemMessage(e)),
      },
    )
  }
  const month = monthName(data.monthStart)
  return (
    <>
      {reason ? (
        <ReadOnlyNote perm="providers:manage" helpers={data.helpers}>
          You can view the providers and the sending defaults.
        </ReadOnlyNote>
      ) : null}
      <section className="card flex flex-wrap" aria-label="Connected providers">
        {data.providers.length === 0 ? (
          <div className="flex-[1_1_100%] px-5 py-4.5 text-ink-2">No sending provider is connected yet.</div>
        ) : (
          data.providers.map((p, i) => <ProviderColumn key={p.id} card={p} month={month} last={i === data.providers.length - 1} />)
        )}
        <div className="flex flex-[1_1_100%] flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-3">
          <span className="k">Health, routing, warm-up, events and adapters live on the Sending providers page.</span>
          <div className="flex gap-2">
            <ButtonLink size="sm" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
              Add provider
            </ButtonLink>
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers" params={{ workspace: slug }}>
              Open Sending providers
            </ButtonLink>
          </div>
        </div>
      </section>

      {error ? <Note tone="bad">{error}</Note> : null}

      <Grid min={380} gap={20}>
        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="defaults-h">
          <h2 className="h2" id="defaults-h">
            Workspace-wide sending defaults
          </h2>
          <span className="k">New campaigns start from these; each campaign can override them in Review &amp; send.</span>
          <div className="grid grid-cols-2 gap-2.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!">
            <TextField label="Auto-pause · bounce rate" value={f.bounce} onChange={(v) => set({ bounce: v })} disabled={Boolean(reason)} />
            <TextField label="Auto-pause · complaint rate" value={f.complaint} onChange={(v) => set({ complaint: v })} disabled={Boolean(reason)} />
            <TextField label="Per mailbox provider complaint" value={f.mailboxComplaint} onChange={(v) => set({ mailboxComplaint: v })} disabled={Boolean(reason)} />
            <TextField label="Provider error rate · 3 min" value={f.providerError} onChange={(v) => set({ providerError: v })} disabled={Boolean(reason)} />
          </div>
          <label className="lbl">
            Pause scope
            <select className="in" value={f.pauseScope} onChange={(e) => set({ pauseScope: e.target.value as Defaults['pauseScope'] })} disabled={Boolean(reason)}>
              <option value="route">Only the affected route or mailbox provider</option>
              <option value="campaign">The whole campaign</option>
            </select>
          </label>
          <TextField label="Confirmation for sends over" value={f.confirmOver} onChange={(v) => set({ confirmOver: v })} disabled={Boolean(reason)} />
        </section>

        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="throttles-h">
          <h2 className="h2" id="throttles-h">
            Throttles per mailbox provider
          </h2>
          <span className="k">Applied on every route. ReputeCast lowers them automatically when a provider defers.</span>
          <div className="grid grid-cols-2 gap-2.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!">
            {THROTTLES.map((t) => (
              <label key={t.key} className="lbl">
                <span className="flex items-center gap-1.5">
                  <span className="dot" style={{ color: t.color }} aria-hidden="true" />
                  {t.label}
                </span>
                <input className="in" value={f.throttles[t.key] ?? ''} onChange={(e) => set({ throttles: { ...f.throttles, [t.key]: e.target.value } })} disabled={Boolean(reason)} />
              </label>
            ))}
          </div>
          <label className="lbl">
            Suppression sync interval
            <select className="in" value={f.suppressionSync} onChange={(e) => set({ suppressionSync: e.target.value as Defaults['suppressionSync'] })} disabled={Boolean(reason)}>
              <option value="15m">Every 15 minutes, plus immediately after each bounce</option>
              <option value="1h">Hourly</option>
            </select>
          </label>
          <div className="lbl">
            <span id="secrets-l">Webhook signing secrets</span>
            <div className="flex gap-2 max-md:flex-wrap">
              <input
                className="in flex-1 bg-white max-md:w-0 max-md:min-w-full"
                value={data.defaults.secretsRotatedAt ? `Rotated ${dayMonthYear(data.defaults.secretsRotatedAt)}` : 'Not rotated since the providers were connected'}
                readOnly
                aria-labelledby="secrets-l"
              />
              <Button onClick={() => setRotating(true)} disabledReason={reason ?? (data.providers.length ? undefined : 'Connect a provider first.')}>
                Rotate
              </Button>
            </div>
          </div>
        </section>
      </Grid>
      <div className="flex justify-end gap-2">
        <Button
          onClick={() => {
            setF(initial)
            setError(undefined)
          }}
        >
          Discard
        </Button>
        <Button variant="primary" onClick={submit} loading={save.isPending} disabledReason={reason}>
          Save defaults
        </Button>
      </div>
      <ConfirmDialog
        open={rotating}
        onOpenChange={setRotating}
        tone="warn"
        badge="Rotate secrets"
        title="Rotate every provider's webhook signing secret?"
        description="ReputeCast makes a new secret for each provider and registers it where the provider's API allows. Providers whose webhook you set up by hand need the new secret there too; until then their events are refused."
        confirmLabel="Rotate secrets"
        confirmVariant="primary"
        busyLabel="Rotating…"
        onConfirm={async () => {
          const res = await rotate(slug)
          qc.setQueryData($api.queryOptions('get', settingsPath, { params: { path: { workspace: slug } } }).queryKey, res.page)
          toast({
            title: `Rotated ${res.rotated} ${res.rotated === 1 ? 'secret' : 'secrets'}`,
            detail: res.skipped.length ? `Update the webhook by hand in ${res.skipped.join(', ')}.` : undefined,
          })
        }}
      />
    </>
  )
}

async function rotate(workspace: string) {
  const { data } = await http.POST('/api/v1/workspaces/{workspace}/settings/providers/rotate-secrets', { params: { path: { workspace } } })
  return data!
}

function TextField({ label, value, onChange, disabled }: { label: string; value: string; onChange: (v: string) => void; disabled: boolean }) {
  return (
    <label className="lbl">
      {label}
      <input className="in" value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled} />
    </label>
  )
}

function ProviderColumn({ card: p, month, last }: { card: Card; month: string; last: boolean }) {
  const cost = p.cost.basis === 'plan' ? `${money(p.cost.monthly)} plan` : p.cost.basis === 'self_hosted' ? `${money(p.cost.toDate, p.cost.toDate ? 2 : 0)} metered` : money(p.cost.toDate)
  return (
    <div className={cn('flex flex-[1_1_260px] flex-col gap-2 px-5 py-4.5', !last && 'border-r border-line-2')}>
      <div className="flex items-center justify-between">
        <span className="font-bold">{p.name}</span>
        <Badge tone={p.statusTone} dot>
          {p.statusLabel}
        </Badge>
      </div>
      <div className="sd-kv">
        <span className="k">Streams</span>
        <span>{p.streams || 'Not routed yet'}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{p.identityLabel}</span>
        <span className="mono">{p.identity}</span>
      </div>
      <div className="sd-kv">
        <span className="k">Connected</span>
        <span>{`${dayMonthYear(p.connectedAt)} · ${p.connectedVia}`}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{`Cost · ${month}`}</span>
        <span className="n">{cost}</span>
      </div>
    </div>
  )
}
