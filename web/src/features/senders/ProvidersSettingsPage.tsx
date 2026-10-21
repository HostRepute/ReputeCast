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
import { SmsSettingsSummary } from '@/features/sms/SettingsSummary'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count, dayMonthYear, decimal, money, monthName } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { mp } from './providers.messages'
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
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', settingsPath, { params: { path: { workspace: slug } } })
  return (
    <Page title={t(m.settings)} documentTitle={t(mp.documentTitle)} tabs={settingsTabs(slug)}>
      {q.isError ? (
        <ErrorPanel title={t(mp.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : q.data ? (
        <SettingsBody key={JSON.stringify(q.data.defaults)} data={q.data} />
      ) : (
        <div className="flex flex-col gap-5" aria-busy="true">
          <span className="vh" role="status">
            {t(mp.loading)}
          </span>
          <section className="card flex flex-wrap" aria-hidden="true">
            {[0, 1, 2].map((i) => (
              <div key={i} className={cn('flex flex-[1_1_260px] flex-col gap-3 px-5 py-4.5', i < 2 && 'border-r border-line-2')}>
                <Skeleton w="50%" h={16} />
                <Skeleton w="100%" h={12} />
                <Skeleton w="90%" h={12} />
                <Skeleton w="80%" h={12} />
                <Skeleton w="40%" h={12} />
              </div>
            ))}
          </section>
          <Grid min={380} gap={20}>
            {[0, 1].map((i) => (
              <section key={i} className="card flex flex-col gap-3 px-5 py-4.5" aria-hidden="true">
                <Skeleton w="45%" h={16} />
                <Skeleton w="85%" h={12} />
                <Skeleton w="100%" h={36} r={8} />
                <Skeleton w="100%" h={36} r={8} />
                <Skeleton w="100%" h={36} r={8} />
                <Skeleton w="100%" h={36} r={8} />
              </section>
            ))}
          </Grid>
        </div>
      )}
    </Page>
  )
}

/** The defaults as the board writes them in the inputs, with the numbers in the region format. */
function formFrom(d: Defaults) {
  return {
    bounce: `${decimal(d.bounce, 1)} %`,
    complaint: `${decimal(d.complaint, 2)} %`,
    mailboxComplaint: `${decimal(d.mailboxComplaint, 2)} %`,
    providerError: `${trim(d.providerError)} %`,
    pauseScope: d.pauseScope,
    confirmOver: tt(mp.confirmOverValue, { count: count(d.confirmOver) }),
    throttles: Object.fromEntries(THROTTLES.map((t) => [t.key, throttleText(d.throttles[t.key])])),
    suppressionSync: d.suppressionSync,
  }
}

/** Up to two decimals, without trailing zeros: 5, 2.5, 0.25. */
function trim(v: number): string {
  const n = Number(v.toFixed(2))
  return decimal(n, (String(n).split('.')[1] ?? '').length)
}

function throttleText(v: number | undefined): string {
  return v ? tt(mp.perSecond, { n: trim(v) }) : tt(mp.noLimit)
}

/**
 * The first number in a field ("2.0 %" → 2, "5,000 recipients…" → 5000,
 * "2,5 %" → 2.5, "5.000 người nhận…" → 5000); "no limit", in English or the
 * app language, is 0. A single mark followed by three digits groups
 * thousands; otherwise it's the decimal point, whichever the region uses.
 */
export function numberIn(s: string): number | undefined {
  const text = s.trim().toLowerCase()
  if (!text || /^(no limit|none|unlimited)$/.test(text) || text === tt(mp.noLimit).toLowerCase()) return 0
  const token = /\d[\d.,\s]*/.exec(s)?.[0]
  if (!token) return undefined
  let n = token.replace(/\s/g, '')
  const marks = n.match(/[.,]/g) ?? []
  if (marks.length) {
    const last = Math.max(n.lastIndexOf('.'), n.lastIndexOf(','))
    const after = n.length - last - 1
    const decimalPoint = new Set(marks).size > 1 || (marks.length === 1 && (after !== 3 || /^0[.,]/.test(n)))
    n = decimalPoint ? `${n.slice(0, last).replace(/[.,]/g, '')}.${n.slice(last + 1)}` : n.replace(/[.,]/g, '')
  }
  const v = Number(n)
  return Number.isFinite(v) ? v : undefined
}

function SettingsBody({ data }: { data: Settings }) {
  const t = useT()
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
      setError(t(mp.needNumber, { example: `${decimal(2, 1)} %` }))
      return
    }
    const throttles: Record<string, number> = {}
    for (const th of THROTTLES) {
      const v = numberIn(f.throttles[th.key] ?? '')
      if (v === undefined) {
        setError(t(mp.throttleFormat, { provider: th.label, example: t(mp.perSecond, { n: decimal(5, 0) }), noLimit: t(mp.noLimit) }))
        return
      }
      throttles[th.key] = v
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
          toast({ title: t(mp.saved), detail: t(mp.savedDetail) })
        },
        onError: (e) => setError(problemMessage(e)),
      },
    )
  }
  const month = monthName(data.monthStart)
  return (
    <>
      {reason ? <ReadOnlyNote msg={mp.readOnlyText} perm="providers:manage" helpers={data.helpers} /> : null}
      <section className="card flex flex-wrap" aria-label={t(mp.connectedProviders)}>
        {data.providers.length === 0 ? (
          <div className="flex-[1_1_100%] px-5 py-4.5 text-ink-2">{t(mp.noneConnected)}</div>
        ) : (
          data.providers.map((p, i) => <ProviderColumn key={p.id} card={p} month={month} last={i === data.providers.length - 1} />)
        )}
        <div className="flex flex-[1_1_100%] flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-3">
          <span className="k">{t(mp.pageNote)}</span>
          <div className="flex gap-2">
            <ButtonLink size="sm" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
              {t(mp.addProvider)}
            </ButtonLink>
            <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers" params={{ workspace: slug }}>
              {t(mp.openProviders)}
            </ButtonLink>
          </div>
        </div>
      </section>
      {/* Phase 8: the SMS providers and the SMS budget. */}
      <SmsSettingsSummary />

      {error ? <Note tone="bad">{error}</Note> : null}

      <Grid min={380} gap={20}>
        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="defaults-h">
          <h2 className="h2" id="defaults-h">
            {t(mp.defaultsTitle)}
          </h2>
          <span className="k">{t(mp.defaultsLine)}</span>
          <div className="grid grid-cols-2 gap-2.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!">
            <TextField label={t(mp.bounce)} value={f.bounce} onChange={(v) => set({ bounce: v })} disabled={Boolean(reason)} />
            <TextField label={t(mp.complaint)} value={f.complaint} onChange={(v) => set({ complaint: v })} disabled={Boolean(reason)} />
            <TextField label={t(mp.mailboxComplaint)} value={f.mailboxComplaint} onChange={(v) => set({ mailboxComplaint: v })} disabled={Boolean(reason)} />
            <TextField label={t(mp.providerError)} value={f.providerError} onChange={(v) => set({ providerError: v })} disabled={Boolean(reason)} />
          </div>
          <label className="lbl">
            {t(mp.pauseScope)}
            <select className="in" value={f.pauseScope} onChange={(e) => set({ pauseScope: e.target.value as Defaults['pauseScope'] })} disabled={Boolean(reason)}>
              <option value="route">{t(mp.pauseRoute)}</option>
              <option value="campaign">{t(mp.pauseCampaign)}</option>
            </select>
          </label>
          <TextField label={t(mp.confirmOver)} value={f.confirmOver} onChange={(v) => set({ confirmOver: v })} disabled={Boolean(reason)} />
        </section>

        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="throttles-h">
          <h2 className="h2" id="throttles-h">
            {t(mp.throttlesTitle)}
          </h2>
          <span className="k">{t(mp.throttlesLine)}</span>
          <div className="grid grid-cols-2 gap-2.5 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!">
            {THROTTLES.map((th) => (
              <label key={th.key} className="lbl">
                <span className="flex items-center gap-1.5">
                  <span className="dot" style={{ color: th.color }} aria-hidden="true" />
                  {th.label}
                </span>
                <input className="in" value={f.throttles[th.key] ?? ''} onChange={(e) => set({ throttles: { ...f.throttles, [th.key]: e.target.value } })} disabled={Boolean(reason)} />
              </label>
            ))}
          </div>
          <label className="lbl">
            {t(mp.suppressionSync)}
            <select className="in" value={f.suppressionSync} onChange={(e) => set({ suppressionSync: e.target.value as Defaults['suppressionSync'] })} disabled={Boolean(reason)}>
              <option value="15m">{t(mp.sync15m)}</option>
              <option value="1h">{t(mp.sync1h)}</option>
            </select>
          </label>
          <div className="lbl">
            <span id="secrets-l">{t(mp.secrets)}</span>
            <div className="flex gap-2 max-md:flex-wrap">
              <input
                className="in flex-1 bg-white max-md:w-0 max-md:min-w-full"
                value={data.defaults.secretsRotatedAt ? t(mp.rotatedOn, { date: dayMonthYear(data.defaults.secretsRotatedAt) }) : t(mp.notRotated)}
                readOnly
                aria-labelledby="secrets-l"
              />
              <Button onClick={() => setRotating(true)} disabledReason={reason ?? (data.providers.length ? undefined : t(mp.connectFirst))}>
                {t(mp.rotate)}
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
          {t(mp.discard)}
        </Button>
        <Button variant="primary" onClick={submit} loading={save.isPending} disabledReason={reason}>
          {t(mp.saveDefaults)}
        </Button>
      </div>
      <ConfirmDialog
        open={rotating}
        onOpenChange={setRotating}
        tone="warn"
        badge={t(mp.rotateSecrets)}
        title={t(mp.rotateTitle)}
        description={t(mp.rotateBody)}
        confirmLabel={t(mp.rotateSecrets)}
        confirmVariant="primary"
        busyLabel={t(mp.rotating)}
        onConfirm={async () => {
          const res = await rotate(slug)
          qc.setQueryData($api.queryOptions('get', settingsPath, { params: { path: { workspace: slug } } }).queryKey, res.page)
          toast({
            title: t(mp.rotated, { count: res.rotated }),
            detail: res.skipped.length ? t(mp.rotatedByHand, { providers: res.skipped.join(', ') }) : undefined,
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
  const t = useT()
  const cost =
    p.cost.basis === 'plan'
      ? t(mp.costPlan, { amount: money(p.cost.monthly) })
      : p.cost.basis === 'self_hosted'
        ? t(mp.costMetered, { amount: money(p.cost.toDate, p.cost.toDate ? 2 : 0) })
        : money(p.cost.toDate)
  return (
    <div className={cn('flex flex-[1_1_260px] flex-col gap-2 px-5 py-4.5', !last && 'border-r border-line-2')}>
      <div className="flex items-center justify-between">
        <span className="font-bold">{p.name}</span>
        <Badge tone={p.statusTone} dot>
          {p.statusLabel}
        </Badge>
      </div>
      <div className="sd-kv">
        <span className="k">{t(mp.streams)}</span>
        <span>{p.streams || t(mp.notRouted)}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{p.identityLabel}</span>
        <span className="mono">{p.identity}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{t(mp.connected)}</span>
        <span>{t(mp.connectedValue, { date: dayMonthYear(p.connectedAt), via: p.connectedVia })}</span>
      </div>
      <div className="sd-kv">
        <span className="k">{t(mp.costMonth, { month })}</span>
        <span className="n">{cost}</span>
      </div>
    </div>
  )
}
