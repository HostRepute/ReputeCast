import { useId, useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { andList, clock } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, Notice, useRefreshVerification, type ServiceView } from './shared'

type KeyTest = { ok: boolean; at: string; balance?: number; problem?: string; status?: number }

function KeyField({ id, value, onChange, name, invalid, problem }: { id: string; value: string; onChange: (v: string) => void; name: string; invalid?: boolean; problem?: string }) {
  const t = useT()
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <label htmlFor={id} className="text-[12.5px] font-semibold">
        {t(m.apiKey)}
      </label>
      <input
        id={id}
        className={invalid ? 'in border-bad-strong!' : 'in'}
        type="password"
        autoComplete="off"
        spellCheck={false}
        placeholder={t(m.keyPlaceholder, { name })}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={invalid || undefined}
        aria-describedby={`${id}-h`}
      />
      <span id={`${id}-h`} className={invalid ? 'text-[12.5px] font-medium text-bad-fg' : 'k'}>
        {invalid && problem ? problem : t(m.keyHint)}
      </span>
    </div>
  )
}

function TestResult({ test }: { test?: KeyTest }) {
  const t = useT()
  if (!test) return null
  return test.ok ? (
    <span className="inline-flex items-center gap-1.5 text-[13px] text-ink-nav">
      <Icon name="check" size={14} className="text-ok-fg" />
      {test.balance != null ? t(m.keyWorks, { credits: test.balance }) : t(m.keyWorksNoBalance)}
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 text-[13px] font-semibold text-bad-fg">
      <Icon name="close" size={14} />
      {t(m.testFailed, { time: clock(test.at) })}
    </span>
  )
}

/** Connect a service: the key, the price, the defaults, a test that reads the balance and the notice. */
export function ConnectDialog({ v, open, onOpenChange, otherForms, otherBulk }: { v: ServiceView; open: boolean; onOpenChange: (o: boolean) => void; otherForms?: string; otherBulk?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const me = useMe().data
  const refresh = useRefreshVerification()
  const ids = { key: useId(), price: useId(), region: useId() }
  const [key, setKey] = useState('')
  const [price, setPrice] = useState('0.008')
  const [bulk, setBulk] = useState(!otherBulk)
  const [forms, setForms] = useState(!otherForms && !bulk)
  const [region, setRegion] = useState(v.regions[0]?.key ?? '')
  const [test, setTest] = useState<KeyTest>()
  const [busy, setBusy] = useState<'test' | 'connect'>()
  const [error, setError] = useState<string>()
  const regionLabel = v.regions.find((r) => r.key === region)?.label ?? ''
  const runTest = async () => {
    setBusy('test')
    setError(undefined)
    const res = await http.POST(`${base}/services/test`, { params: { path: { workspace: slug } }, body: { service: v.key, apiKey: key.trim(), region: (region || undefined) as 'us' | undefined } })
    setBusy(undefined)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    setTest(res.data as KeyTest)
  }
  const connect = async () => {
    setBusy('connect')
    setError(undefined)
    const res = await http.POST(`${base}/services`, {
      params: { path: { workspace: slug } },
      body: { service: v.key, apiKey: key.trim(), price: Number(price) || 0, defaultBulk: bulk, defaultForms: forms, region: (region || undefined) as 'us' | undefined },
    })
    setBusy(undefined)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    onOpenChange(false)
    toast({ kind: 'success', title: t(m.connectedToast, { name: v.name }) })
  }
  const rejected = test && !test.ok
  const blocked = !key.trim() ? t(m.pasteKey) : !test?.ok ? t(m.testFirst) : !(Number(price) > 0) ? t(m.priceInvalid) : undefined
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{rejected ? t(m.keyRejected) : t(m.connectCrumb, { name: me?.user.name ?? '' })}</span>}
      title={t(m.connectTitle, { name: v.name })}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={() => void connect()} loading={busy === 'connect'} busyLabel={t(m.connecting)} disabledReason={busy ? undefined : blocked}>
            {t(m.connectTitle, { name: v.name })}
          </Button>
        </>
      }
    >
      <KeyField
        id={ids.key}
        value={key}
        onChange={(s) => {
          setKey(s)
          setTest(undefined)
        }}
        name={v.name}
        invalid={rejected}
        problem={test?.problem}
      />
      {rejected ? null : (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(220px,100%),1fr))' }}>
          <div className="flex min-w-0 flex-col gap-1.5">
            <label htmlFor={ids.price} className="text-[12.5px] font-semibold">
              {t(m.priceTitle)}
            </label>
            <div className="flex items-center gap-1.5">
              <span className="k text-[13px]!">$</span>
              <input id={ids.price} className="in w-full" inputMode="decimal" value={price} onChange={(e) => setPrice(e.target.value.replace(/[^\d.]/g, ''))} />
            </div>
            <span className="k">{t(m.priceHint)}</span>
          </div>
          <fieldset className="m-0 flex min-w-0 flex-col gap-2 border-0 p-0">
            <legend className="mb-1.5 p-0 text-[12.5px] font-semibold">{t(m.defaultTitle)}</legend>
            <label className="flex items-start gap-2 text-[13px]">
              <input type="checkbox" checked={bulk} onChange={(e) => setBulk(e.target.checked)} className="mt-0.5" />
              {t(m.bulkAndImports)}
            </label>
            <label className="flex items-start gap-2 text-[13px]">
              <input type="checkbox" checked={forms} onChange={(e) => setForms(e.target.checked)} className="mt-0.5" />
              {t(m.signupForms)}
            </label>
          </fieldset>
        </div>
      )}
      {v.regions.length > 1 && !rejected ? (
        <div className="flex min-w-0 flex-col gap-1.5">
          <label htmlFor={ids.region} className="text-[12.5px] font-semibold">
            {t(m.region)}
          </label>
          <select id={ids.region} className="in" value={region} onChange={(e) => setRegion(e.target.value)}>
            {v.regions.map((r) => (
              <option key={r.key} value={r.key}>
                {r.label}
              </option>
            ))}
          </select>
        </div>
      ) : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button size="sm" icon="refresh" onClick={() => void runTest()} loading={busy === 'test'} busyLabel={t(m.testing)} disabledReason={busy || key.trim() ? undefined : t(m.pasteKey)}>
          {t(m.testConnection)}
        </Button>
        <TestResult test={test} />
      </div>
      {error ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <span>{error}</span>
        </div>
      ) : null}
      {rejected ? null : <Notice head={regionLabel ? t(m.noticeWhenUsed, { name: v.name, region: regionLabel }) : t(m.noticeWhenUsedNoRegion, { name: v.name })} name={v.name} />}
    </Dialog>
  )
}

/** Replace the key: checked first, then swapped in. */
export function ReplaceKeyDialog({ v, open, onOpenChange }: { v: ServiceView; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const id = useId()
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const replace = async () => {
    setBusy(true)
    setError(undefined)
    const res = await http.PUT(`${base}/services/{service}/key`, { params: { path: { workspace: slug, service: v.key } }, body: { apiKey: key.trim() } })
    setBusy(false)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    onOpenChange(false)
    toast({ kind: 'success', title: t(m.keyReplaced, { name: v.name }), detail: res.data.balance != null ? t(m.balanceRead, { credits: res.data.balance }) : undefined })
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{t(m.dialogCrumb)}</span>}
      title={t(m.replaceTitle, { name: v.name })}
      description={t(m.replaceBody)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={() => void replace()} loading={busy} busyLabel={t(m.replacing)} disabledReason={busy || key.trim() ? undefined : t(m.pasteKey)}>
            {t(m.replace)}
          </Button>
        </>
      }
    >
      <KeyField id={id} value={key} onChange={setKey} name={v.name} invalid={Boolean(error)} problem={error} />
    </Dialog>
  )
}

/** Disconnect: what stops, what goes, what stays and what takes over. */
export function DisconnectDialog({ v, open, onOpenChange, forms }: { v: ServiceView; open: boolean; onOpenChange: (o: boolean) => void; forms: string[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const go = async () => {
    setBusy(true)
    setError(undefined)
    const res = await http.DELETE(`${base}/services/{service}`, { params: { path: { workspace: slug, service: v.key }, query: { forms } } })
    setBusy(false)
    if (!res.data) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    onOpenChange(false)
    toast({ kind: 'deleted', title: t(m.disconnected, { name: v.name }) })
  }
  const stops = [
    v.defaultForms && forms.length ? t(m.stopsForms, { forms: andList(forms) }) : '',
    v.defaultBulk ? t(m.stopsBulk, { name: v.name }) : '',
  ].filter(Boolean)
  const removed = v.defaultBulk && v.defaultForms ? m.removedKeyBoth : v.defaultForms ? m.removedKeyForms : v.defaultBulk ? m.removedKeyBulk : m.removedKey
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{t(m.dialogCrumb)}</span>}
      title={t(m.disconnectQuestion, { name: v.name })}
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} disabled={busy}>
            {t(m.cancel)}
          </Button>
          <Button variant="danger-primary" onClick={() => void go()} loading={busy} busyLabel={t(m.disconnecting)}>
            {t(m.disconnectButton, { name: v.name })}
          </Button>
        </>
      }
    >
      <div className="note w" role="note">
        <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
        <div className="flex min-w-0 flex-col gap-1">
          <b className="font-bold">{t(m.whatStops)}</b>
          <span>{stops.length ? stops.join(' ') : t(m.stopsNothing, { name: v.name })}</span>
        </div>
      </div>
      <dl className="kv">
        <dt>{t(m.removed)}</dt>
        <dd>{t(removed, { name: v.name })}</dd>
        <dt>{t(m.stays)}</dt>
        <dd>{t(m.staysValue)}</dd>
        <dt>{t(m.instead)}</dt>
        <dd>{v.defaultForms ? t(m.insteadValue) : t(m.insteadBulk)}</dd>
      </dl>
      {error ? (
        <div className="note b" role="alert">
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
