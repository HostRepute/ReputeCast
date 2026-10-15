import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { T, formatList, useT } from '@/i18n'
import { ago, clock, clockSeconds } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import {
  base,
  entryName,
  helpersText,
  isPlatform,
  itemLine,
  itemName,
  PermChips,
  PermChip,
  PLUGIN_URL,
  platformUrl,
  tags,
  useRefreshHub,
  type ConnectKind,
  type ItemLock,
  type Secret,
} from './shared'

type Info = Schemas['IntegrationConnectInfo']
type Status = Schemas['IntegrationStatus']
type AddressCheck = Schemas['IntegrationAddressCheck']
type Step = 'choose' | 'paste' | 'wait'

/** How long the last step waits before saying "No call yet". */
const NO_CALL_AFTER = 10 * 60_000

/**
 * Connecting, in three steps in one dialog (IntegrationsConnect, and the
 * WordPress board's "Connect a site"): choose what it can do, paste the
 * instance URL and the key, wait for the first call. It reconnects a
 * connection whose key stopped the same way.
 */
export function ConnectDialog({
  kind,
  open,
  onOpenChange,
  onDone,
}: {
  kind: ConnectKind
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Done, after the first call: the caller opens the integration's page. */
  onDone?: () => void
}) {
  const { slug } = useWorkspace()
  const info = $api.useQuery('get', `${base}/apps/{kind}/connect`, { params: { path: { workspace: slug, kind } } }, { enabled: open, staleTime: 0 })
  const [step, setStep] = useState<Step>('choose')
  const [secret, setSecret] = useState<Secret | null>(null)
  const [count, setCount] = useState(0)
  const refresh = useRefreshHub()
  const close = (o: boolean) => {
    if (!o) void refresh()
    onOpenChange(o)
  }
  return (
    <Dialog
      open={open}
      onOpenChange={close}
      title={<DialogTitle kind={kind} step={step} secret={secret} />}
      eyebrow={<Eyebrow kind={kind} step={step} secret={secret} />}
      description={step === 'choose' ? <ChooseDescription kind={kind} /> : step === 'paste' && kind !== 'wordpress' ? <PasteDescription kind={kind} /> : undefined}
      dismissible={step !== 'paste'}
      size="md"
      hideClose
    >
      {info.isError ? (
        <Note tone="bad">{errorMessage(info.error)}</Note>
      ) : !info.data ? (
        <DialogSkeleton />
      ) : step === 'choose' ? (
        <Choose
          kind={kind}
          info={info.data}
          onCancel={() => close(false)}
          onCreated={(s, n) => {
            setSecret(s)
            setCount(n)
            setStep('paste')
          }}
        />
      ) : step === 'paste' && secret ? (
        <Paste kind={kind} info={info.data} secret={secret} onContinue={() => setStep('wait')} />
      ) : secret ? (
        <Wait kind={kind} secret={secret} count={count} onClose={() => close(false)} onDone={() => (onDone ? onDone() : close(false))} />
      ) : null}
    </Dialog>
  )
}

function DialogSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-3" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingDialog)}
      </span>
      <Skeleton w="60%" h={14} />
      <Skeleton w="100%" h={44} />
      <Skeleton w="100%" h={44} />
      <Skeleton w="100%" h={44} />
    </div>
  )
}

/** "Connect Make · Marcus Chen", "Key created 14:25 · recorded in the audit log", "Listening". */
function Eyebrow({ kind, step, secret }: { kind: ConnectKind; step: Step; secret: Secret | null }) {
  const t = useT()
  const me = useMe().data
  const name = t(entryName[kind])
  if (step === 'choose') return <span className="k">{kind === 'wordpress' ? t(m.wpEyebrow) : t(m.connectEyebrow, { name, person: me?.user.name ?? '' })}</span>
  if (step === 'paste') return <span className="k">{t(m.keyCreatedEyebrow, { time: secret ? clock(secret.key.createdAt) : '' })}</span>
  return null
}

function DialogTitle({ kind, step, secret }: { kind: ConnectKind; step: Step; secret: Secret | null }) {
  const t = useT()
  const name = t(entryName[kind])
  if (kind === 'wordpress') {
    if (step === 'choose') return <>{t(m.wpChooseTitle)}</>
    if (step === 'paste') return <>{t(m.wpInstallTitle)}</>
    return <>{t(m.wpWaitTitle, { host: siteOf(secret) })}</>
  }
  if (step === 'choose') return <>{t(m.chooseTitle, { name })}</>
  if (step === 'paste') return <>{t(m.pasteTitle, { name })}</>
  return <>{t(m.waitTitle, { name })}</>
}

function ChooseDescription({ kind }: { kind: ConnectKind }) {
  const t = useT()
  if (kind === 'wordpress') return undefined
  return <span className="text-[13.5px]">{t(m.chooseBody, { name: t(entryName[kind]) })}</span>
}

function PasteDescription({ kind }: { kind: ConnectKind }) {
  const t = useT()
  return <span className="text-[13.5px]">{t(m.pasteBody, { name: t(entryName[kind]) })}</span>
}

/** The site's host from its key's name ("WordPress · acme.io"). */
function siteOf(secret: Secret | null): string {
  const n = secret?.key.name ?? ''
  const i = n.indexOf(' · ')
  return i >= 0 ? n.slice(i + 3) : n
}

/** The three steps, with the current one marked (and the later ones off when they can't be reached). */
export function Steps({ kind, step, blocked }: { kind: ConnectKind; step: Step; blocked?: boolean }) {
  const t = useT()
  const name = t(entryName[kind])
  const labels =
    kind === 'wordpress' ? [t(m.wpStep1), t(m.wpStep2), t(m.wpStep3)] : [t(m.step1, { name }), t(m.step2, { name }), t(m.step3, { name })]
  const at = step === 'choose' ? 0 : step === 'paste' ? 1 : 2
  return (
    <ol className="ih-stp -mt-1" aria-label={t(m.steps)}>
      {labels.map((label, i) => (
        <li key={i} className="contents">
          {i > 0 ? (
            <span className="ih-sep" aria-hidden="true">
              ›
            </span>
          ) : null}
          <span
            className={`flex items-center gap-1.5 text-[12.5px] font-semibold whitespace-nowrap ${i < at ? 'text-ok-fg' : i === at ? 'text-accent' : 'text-ink-3'}`}
            aria-current={i === at ? 'step' : undefined}
            aria-disabled={blocked && i > at ? true : undefined}
          >
            <span
              className={`ih-n ${i < at ? '!border-[#86efac] !bg-ok-bg' : i === at ? '!border-accent !bg-accent !text-white' : blocked ? '!border-dashed' : ''}`}
              aria-hidden="true"
            >
              {i < at ? <Icon name="check" size={11} strokeWidth={2.4} /> : i + 1}
            </span>
            {label}
          </span>
        </li>
      ))}
    </ol>
  )
}

/** The permissions the key gets for the items that are on, shortest form first. */
export function keyPermissions(info: Pick<Info, 'order' | 'includes'>, items: ItemLock[], on: Set<string>) {
  const wanted = new Set<string>()
  let trigger = false
  for (const it of items) {
    if (!on.has(it.key)) continue
    it.permissions.forEach((p) => wanted.add(p))
    if (it.kind === 'trigger') trigger = true
  }
  if (trigger) wanted.add('webhooks:manage')
  const included = new Map<string, string>()
  for (const p of wanted) {
    for (const q of wanted) {
      if (q !== p && (info.includes[q] ?? []).includes(p)) included.set(p, q)
    }
  }
  const rank = (p: string) => {
    const i = info.order.indexOf(p)
    return i < 0 ? info.order.length : i
  }
  const short = [...wanted].filter((p) => !included.has(p)).sort((a, b) => rank(a) - rank(b))
  const sensitive = new Set(items.filter((it) => it.sensitive && on.has(it.key)).flatMap((it) => it.permissions))
  return { short, included, sensitive: [...sensitive] }
}

function Choose({ kind, info, onCancel, onCreated }: { kind: ConnectKind; info: Info; onCancel: () => void; onCreated: (s: Secret, items: number) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const navigate = useNavigate()
  const name = t(entryName[kind])
  const [on, setOn] = useState(() => new Set(info.items.filter((i) => i.on && !i.locked).map((i) => i.key)))
  const [keyName, setKeyName] = useState(info.name)
  const [listId, setListId] = useState('')
  const [expiry, setExpiry] = useState('none')
  const [date, setDate] = useState('')
  const [address, setAddress] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const wp = kind === 'wordpress'
  const unreachable = (kind === 'zapier' || kind === 'make') && !info.reachability.reachable
  const triggers = info.items.filter((i) => i.kind === 'trigger')
  const actions = info.items.filter((i) => i.kind === 'action')
  const perms = keyPermissions(info, info.items, on)
  const locked = info.items.filter((i) => i.locked)
  const host = address.trim().replace(/^https?:\/\//, '').replace(/\/+$/, '')

  async function create() {
    setBusy(true)
    setError(undefined)
    let expiresAt: string | undefined
    if (expiry === 'date' && date) expiresAt = new Date(`${date}T23:59:59`).toISOString()
    else if (expiry !== 'none' && expiry !== 'date') expiresAt = new Date(Date.now() + Number(expiry) * 86_400_000).toISOString()
    const body = {
      items: [...on],
      name: wp ? undefined : keyName.trim(),
      lists: listId ? [listId] : undefined,
      expiresAt,
      address: wp ? address.trim() : undefined,
    }
    try {
      const res = info.connectionId
        ? await http.POST(`${base}/connections/{id}/reconnect`, { params: { path: { workspace: slug, id: info.connectionId } }, body })
        : await http.POST(`${base}/apps/{kind}/connect`, { params: { path: { workspace: slug, kind } }, body })
      onCreated(res.data!, on.size)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  if (unreachable) {
    const r = info.reachability
    return (
      <>
        <span className="k -mt-2">{t(m.privateEyebrow)}</span>
        <Steps kind={kind} step="choose" blocked />
        <Note tone="bad" role="alert">
          <span className="font-semibold">{t(m.unreachableTitle, { name })}</span>
          <div className="mt-0.5">{t(m.unreachableBody, { name, url: r.url })}</div>
          <div className="mt-2">
            <CheckLines reach={r} />
          </div>
          <div className="row mt-2.5 gap-2">
            <Button size="sm" onClick={() => void navigate({ to: '/w/$workspace/admin/instance', params: { workspace: slug } })}>
              {t(m.instanceSettings)}
            </Button>
            <Button size="sm" onClick={() => void navigate({ to: '/w/$workspace/settings/integrations/$kind', params: { workspace: slug, kind: 'n8n' } })}>
              {t(m.connectN8nInstead)}
            </Button>
          </div>
        </Note>
        <div className="row flex-nowrap items-start gap-2 text-[13px] text-ink-2">
          <Icon name="info" size={15} className="mt-0.5 flex-none" />
          <span>{t(m.unreachableListed, { triggers: triggers.length, actions: actions.length, name })}</span>
        </div>
        <Footer>
          <span className="k mr-auto">{t(m.needsPublic)}</span>
          <Button onClick={onCancel}>{t(m.cancel)}</Button>
          <Button variant="primary" disabledReason={t(m.needsPublic)}>
            {t(m.createKey)}
          </Button>
        </Footer>
      </>
    )
  }

  const toggle = (k: string, v: boolean) =>
    setOn((prev) => {
      const next = new Set(prev)
      if (v) next.add(k)
      else next.delete(k)
      return next
    })

  return (
    <>
      <Steps kind={kind} step="choose" />
      {wp ? (
        <label className="lbl">
          {t(m.siteAddress)}
          <input
            className="in"
            value={address}
            onChange={(e) => {
              setAddress(e.target.value)
            }}
            placeholder={t(m.siteAddressPlaceholder)}
            autoComplete="url"
            spellCheck={false}
          />
          <span className="k font-medium">{t(m.siteAddressHint)}</span>
        </label>
      ) : null}
      <div className="ih-prl">
        {triggers.length ? (
          <div className="ih-phd">
            {t(m.triggers)} <span className="k">{t(m.triggersHintConnect, { name })}</span>
          </div>
        ) : null}
        {triggers.map((it) => (
          <ItemRow key={it.key} item={it} on={on.has(it.key)} onChange={(v) => toggle(it.key, v)} />
        ))}
        {!wp ? (
          <div className="ih-phd">
            {t(m.actions)} <span className="k">{t(m.actionsHintConnect, { name })}</span>
          </div>
        ) : null}
        {actions.map((it) => (
          <ItemRow key={it.key} item={it} on={on.has(it.key)} onChange={(v) => toggle(it.key, v)} wp={wp} />
        ))}
      </div>
      <div className="ih-gets">
        <div className="row justify-between gap-2">
          <span className="font-semibold">{t(m.keyGets)}</span>
          <span className="k">{t(m.updatesAsYouSwitch)}</span>
        </div>
        {perms.short.length ? (
          <PermChips
            perms={perms.short}
            sensitive={perms.sensitive}
            notes={wp ? { 'subscribers:view': t(m.onlyListForms), 'campaigns:edit': t(m.onlyCreateDrafts) } : undefined}
          />
        ) : (
          <span className="k">{t(m.nothingOn)}</span>
        )}
        <span className="k">
          {wp
            ? t(m.siteOneWorkspace)
            : [
                ...[...perms.included].map(([p, by]) => t(m.includesView, { perm: by, implied: p })),
                t(m.worksOnlyFor, { count: on.size }),
                locked.length ? t(m.unavailableItems, { count: locked.length, items: formatList(locked.map((l) => itemName(t, l.key))) }) : '',
              ]
                .filter(Boolean)
                .join(' ')}
        </span>
      </div>
      <div className={wp ? 'flex flex-col gap-3' : 'grid gap-3 sm:grid-cols-3'}>
        <label className="lbl">
          {t(m.keyName)}
          <input
            className="in"
            value={wp ? t(m.siteKeyName, { host: host || t(m.siteAddressPlaceholder) }) : keyName}
            onChange={(e) => setKeyName(e.target.value)}
            readOnly={wp}
            maxLength={100}
          />
        </label>
        {!wp ? (
          <>
            <label className="lbl">
              {t(m.listScope)}
              <select className="in" value={listId} onChange={(e) => setListId(e.target.value)}>
                <option value="">{t(m.allLists)}</option>
                {info.lists.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="lbl">
              {t(m.expiry)}
              <select className="in" value={expiry} onChange={(e) => setExpiry(e.target.value)}>
                <option value="none">{t(m.expiryNone)}</option>
                <option value="30">{t(m.expiry30)}</option>
                <option value="90">{t(m.expiry90)}</option>
                <option value="365">{t(m.expiry365)}</option>
                <option value="date">{t(m.expiryDate)}</option>
              </select>
            </label>
          </>
        ) : null}
      </div>
      {expiry === 'date' && !wp ? (
        <label className="lbl max-w-[220px]">
          {t(m.expiryDateLabel)}
          <input className="in" type="date" value={date} min={new Date().toISOString().slice(0, 10)} onChange={(e) => setDate(e.target.value)} />
        </label>
      ) : null}
      {!wp ? <Receives kind={kind} on={on} /> : null}
      {error ? (
        <Note tone="bad" role="alert">
          {error}
        </Note>
      ) : null}
      <Footer>
        <Button onClick={onCancel}>{t(m.cancel)}</Button>
        <Button
          variant="primary"
          onClick={create}
          loading={busy}
          disabledReason={on.size === 0 ? t(m.nothingOn) : wp && !host ? t(m.siteAddressHint) : expiry === 'date' && !date ? t(m.expiryDateLabel) : undefined}
        >
          {t(m.createKey)}
        </Button>
      </Footer>
    </>
  )
}

/**
 * The dialog footer inside the body: the Dialog's own footer slot sits
 * outside the steps, so each step brings its buttons along.
 */
export function Footer({ children }: { children: ReactNode }) {
  return <div className="-mx-[22px] -mb-4 mt-1 flex flex-wrap items-center justify-end gap-2.5 border-t border-line-2 px-[22px] pt-3.5 pb-[18px]">{children}</div>
}

/** One trigger or action with its permissions and its switch (locked when the person can't grant it). */
export function ItemRow({ item, on, onChange, wp }: { item: ItemLock; on: boolean; onChange: (v: boolean) => void; wp?: boolean }) {
  const t = useT()
  const name = itemName(t, item.key)
  const line = item.kind === 'trigger' ? item.event : itemLine(t, item.key)
  return (
    <div className={`ih-prow${item.locked ? ' lk' : ''}`}>
      <div className="ih-pl">
        <span className="ih-pn">
          {name}
          {item.new ? <Badge tone="info">{t(m.newBadge)}</Badge> : null}
          {item.locked ? <Badge tone="neutral">{t(m.unavailable)}</Badge> : null}
        </span>
        {line ? <span className={`ih-pd${item.kind === 'trigger' ? ' mono' : ''}`}>{line}</span> : null}
        {item.sensitive && !item.locked && !wp ? <span className="ih-pd">{t(m.startsOffSensitive, { perm: item.permissions[0] ?? '' })}</span> : null}
        {item.locked ? (
          <span className="ih-pw">
            {item.helpers?.length ? t(m.cantGrant, { perm: item.missing ?? '', helpers: helpersText(item.helpers) }) : t(m.cantGrantAlone, { perm: item.missing ?? '' })}
          </span>
        ) : null}
      </div>
      <div className="ih-pp">
        {item.permissions.map((p) => (
          <PermChip key={p} perm={p} />
        ))}
        {item.sensitive ? (
          <span className="flag" title={t(m.sensitivePermission)}>
            <span aria-hidden="true">⚑</span>
            {t(m.sensitive)}
          </span>
        ) : null}
      </div>
      <input
        type="checkbox"
        role="switch"
        className="tg"
        checked={on}
        disabled={item.locked}
        aria-label={name}
        onChange={(e) => onChange(e.target.checked)}
      />
    </div>
  )
}

/** "What Make will receive": the data of the triggers that are on. */
function Receives({ kind, on }: { kind: ConnectKind; on: Set<string> }) {
  const t = useT()
  const name = t(entryName[kind])
  const lines = receiveLines(t, on)
  if (!lines.length) return null
  return (
    <div className="flex flex-col gap-1.5">
      <span className="lbl">{t(m.willReceive, { name })}</span>
      <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13px] text-ink-nav">
        {lines.map((l, i) => (
          <li key={i}>{l}</li>
        ))}
      </ul>
      <span className="k">{t(m.receivesFoot, { kind })}</span>
    </div>
  )
}

/** One line per trigger that is on; the three contact triggers share one when they're all on. */
export function receiveLines(t: ReturnType<typeof useT>, on: Set<string> | string[]): ReactNode[] {
  const has = (k: string) => (Array.isArray(on) ? on.includes(k) : on.has(k))
  const out: ReactNode[] = []
  const contacts = ['contact.subscribed', 'contact.unsubscribed', 'contact.updated']
  if (contacts.every(has)) out.push(<T key="c" msg={m.receivesContacts} values={tags} />)
  else
    contacts.filter(has).forEach((k) => out.push(<T key={k} msg={m.receivesContact} values={{ ...tags, trigger: itemName(t, k) }} />))
  if (has('form.submitted')) out.push(<T key="f" msg={m.receivesForm} values={tags} />)
  if (has('campaign.completed')) out.push(<T key="s" msg={m.receivesCampaign} values={tags} />)
  if (has('message.bounced')) out.push(<T key="b" msg={m.receivesBounce} values={tags} />)
  return out
}

/** "DNS · TLS · Reachable from the internet · checked 4 min ago", each with its mark. */
export function CheckLines({ reach }: { reach: Schemas['IntegrationReachability'] }) {
  const t = useT()
  const mark = (status: string) =>
    status === 'ok' ? (
      <Icon name="check" size={14} className="text-ok-fg" strokeWidth={2.2} />
    ) : status === 'skipped' ? (
      <Icon name="clock" size={14} className="text-ink-3" />
    ) : (
      <Icon name="close" size={14} className={status === 'warn' ? 'text-warn-fg' : 'text-bad-fg'} strokeWidth={2.2} />
    )
  const line = (l: Schemas['CheckLine'], ok: string) => (l.status === 'ok' ? ok : l.detail || ok)
  return (
    <div className="ih-chk3">
      <span>
        {mark(reach.dns.status)}
        {line(reach.dns, t(m.checkDns))}
      </span>
      <span>
        {mark(reach.tls.status)}
        {line(reach.tls, t(m.checkTls))}
      </span>
      <span>
        {mark(reach.reach.status)}
        {line(reach.reach, t(m.checkReach))}
      </span>
      {reach.checkedAt ? <span className="k">{t(m.checkedAgo, { ago: ago(reach.checkedAt) })}</span> : null}
    </div>
  )
}

/** A read-only value with a Copy button (the instance URL, the key). */
export function CopyField({ label, value, primary, copyLabel }: { label: string; value: string; primary?: boolean; copyLabel: string }) {
  const t = useT()
  const [done, setDone] = useState(false)
  return (
    <label className="lbl">
      {label}
      <div className="ih-inl">
        <input className="in mono text-[12.5px]" value={value} readOnly onFocus={(e) => e.currentTarget.select()} />
        <Button
          size="sm"
          variant={primary ? 'primary' : 'default'}
          aria-label={copyLabel}
          icon={done ? 'check' : 'copy'}
          onClick={async () => {
            const ok = await copyText(value)
            setDone(ok)
            toast({ kind: ok ? 'success' : 'error', title: ok ? t(m.copiedToast) : errorMessage(new Error(value)) })
          }}
        >
          {done ? t(m.copied) : t(m.copy)}
        </Button>
      </div>
    </label>
  )
}

/** Step 2: the instance URL and the key, shown once, with the other tool's own steps. */
function Paste({ kind, info, secret, onContinue }: { kind: ConnectKind; info: Info; secret: Secret; onContinue: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[kind])
  const [copied, setCopied] = useState(false)
  const [address, setAddress] = useState('')
  const [check, setCheck] = useState<AddressCheck | null>(null)
  const [saving, setSaving] = useState(false)
  const wp = kind === 'wordpress'
  const instanceUrl = secret.instanceUrl || info.instanceUrl

  // n8n's address: checked as it's typed (private addresses need an instance admin for triggers).
  useEffect(() => {
    if (kind !== 'n8n' || !address.trim()) return
    const id = window.setTimeout(async () => {
      try {
        const res = await http.POST(`${base}/address-check`, { params: { path: { workspace: slug } }, body: { address: address.trim() } })
        setCheck(res.data ?? null)
      } catch {
        setCheck(null)
      }
    }, 450)
    return () => window.clearTimeout(id)
  }, [address, kind, slug])

  async function next() {
    if (kind === 'n8n' && address.trim()) {
      setSaving(true)
      try {
        await http.PUT(`${base}/connections/{id}/address`, { params: { path: { workspace: slug, id: secret.integrationId } }, body: { address: address.trim() } })
      } catch (e) {
        toast({ kind: 'error', title: errorMessage(e) })
      } finally {
        setSaving(false)
      }
    }
    onContinue()
  }

  const stepsIn = wp
    ? [t(m.wpStepInstall), t(m.wpStepOpen), t(m.wpStepPaste)]
    : kind === 'zapier'
      ? [m.zapierStep1, m.zapierStep2, m.zapierStep3]
      : kind === 'make'
        ? [m.makeStep1, m.makeStep2, m.makeStep3]
        : [m.n8nStep1, m.n8nStep2, m.n8nStep3]
  // A check answers for the address typed; an emptied field shows none.
  const seen = kind === 'n8n' && address.trim() ? check : null
  const openUrl = wp ? PLUGIN_URL : isPlatform(kind) ? platformUrl(kind, seen?.address ?? (address.trim() || undefined)) : undefined

  const howTo = (
    <div className="flex flex-col gap-2">
      <span className="font-semibold">{wp ? t(m.inWordPress) : t(m.inPlatform, { name })}</span>
      <ol className="ih-ol">
        {stepsIn.map((s, i) => (
          <li key={i}>
            <span>{typeof s === 'string' ? s : <T msg={s} values={tags} />}</span>
          </li>
        ))}
      </ol>
      {openUrl ? (
        <div>
          <ButtonAnchor size="sm" href={openUrl} iconRight="external">
            {wp ? t(m.getPlugin) : t(m.openPlatform, { name })}
            <span className="vh">{t(m.newTab)}</span>
          </ButtonAnchor>
        </div>
      ) : null}
    </div>
  )

  return (
    <>
      <Steps kind={kind} step="paste" />
      {wp ? howTo : null}
      <CopyField label={t(m.instanceUrl)} value={instanceUrl} copyLabel={t(m.copyInstanceUrl)} />
      {kind === 'zapier' || kind === 'make' ? <CheckLines reach={info.reachability} /> : null}
      <CopyField label={t(m.keyLabel, { key: secret.key.name })} value={secret.secret} primary copyLabel={t(m.copyKey)} />
      {kind === 'n8n' ? (
        <label className="lbl">
          {t(m.n8nAddress)}
          <input
            className="in mono"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder={t(m.n8nAddressPlaceholder)}
            spellCheck={false}
            autoComplete="url"
          />
          <span className="k font-medium">{t(m.n8nAddressHint)}</span>
        </label>
      ) : null}
      {seen?.private ? <PrivateNote check={seen} /> : null}
      <Note tone="warn">
        <span className="font-semibold">{t(m.shownOnceTitle)}</span>
        <div>{wp ? t(m.wpShownOnceBody) : t(m.shownOnceBody, { name })}</div>
      </Note>
      {wp ? null : howTo}
      <Footer>
        <label className="row mr-auto gap-2 text-[13px]">
          <input type="checkbox" checked={copied} onChange={(e) => setCopied(e.target.checked)} />
          {t(m.iCopied)}
        </label>
        <Button variant="primary" onClick={next} loading={saving} disabledReason={copied ? undefined : t(m.iCopied)}>
          {t(m.continue)}
        </Button>
      </Footer>
    </>
  )
}

/** n8n on a private address: actions work, triggers need an instance admin to allow private addresses. */
function PrivateNote({ check }: { check: AddressCheck }) {
  const { slug } = useWorkspace()
  const t = useT()
  const navigate = useNavigate()
  const [asked, setAsked] = useState(false)
  if (check.allowed) return <Note tone="ok">{t(m.privateAllowed, { host: check.host, ip: check.resolves ?? '' })}</Note>
  const admin = check.admins[0]
  return (
    <Note tone="warn">
      <span className="font-semibold">{t(m.privateTitle)}</span>
      <div>
        {t(m.privateBody, { host: check.host, ip: check.resolves ?? '', count: check.admins.length, admins: formatList(check.admins) })}
        <div className="row mt-2 gap-2">
          <Button size="sm" onClick={() => void navigate({ to: '/w/$workspace/admin/instance', params: { workspace: slug } })}>
            {t(m.instanceSettings)}
          </Button>
          {admin ? (
            <Button
              size="sm"
              disabled={asked}
              onClick={async () => {
                const ok = await copyText(`${check.host} (${check.resolves ?? ''}): ${t(m.privateTitle)}`)
                setAsked(true)
                toast({ kind: ok ? 'success' : 'info', title: t(m.askAdminDone, { name: admin }) })
              }}
            >
              {asked ? t(m.askAdminDone, { name: admin }) : t(m.askAdmin, { name: admin })}
            </Button>
          ) : null}
        </div>
      </div>
    </Note>
  )
}

/** "1 min 40 s" */
function listeningFor(ms: number, t: ReturnType<typeof useT>): string {
  const s = Math.max(0, Math.floor(ms / 1000))
  const min = Math.floor(s / 60)
  return min ? t(m.durationMinSec, { min, sec: s % 60 }) : t(m.durationSec, { sec: s })
}

/** The operation of the first call, in words ("Test connection"). */
export function operationLabel(t: ReturnType<typeof useT>, op: string): string {
  switch (op) {
    case 'get-integration':
      return t(m.opMe)
    case 'subscribe-integration-hook':
      return t(m.opSubscribe)
    case 'unsubscribe-integration-hook':
      return t(m.opUnsubscribe)
    case 'get-integration-trigger-sample':
      return t(m.opSample)
    case 'list-integration-lists':
      return t(m.opLists)
    case 'list-integration-fields':
      return t(m.opFields)
    case 'report-wordpress-site':
      return t(m.opReport)
    case 'list-wordpress-forms':
      return t(m.item_list_forms)
    case 'create-wordpress-draft':
      return t(m.item_create_drafts)
    case 'upsert-contact':
      return t(m.item_upsert_contact)
    case 'add-contact-to-list':
      return t(m.item_add_to_list)
    case 'tag-contact':
      return t(m.item_add_tag)
    case 'send-automation-event':
      return t(m.item_record_event)
    case 'send-transactional-message':
      return t(m.item_send_transactional)
  }
  return op
}

/** Step 3: a live line until the first call (or the site's first report), then Connected. */
function Wait({ kind, secret, count, onClose, onDone }: { kind: ConnectKind; secret: Secret; count: number; onClose: () => void; onDone: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[kind])
  const wp = kind === 'wordpress'
  const host = siteOf(secret)
  const createdAt = new Date(secret.key.createdAt)
  const [since, setSince] = useState(() => Date.now())
  const [now, setNow] = useState(() => Date.now())
  const status = $api.useQuery(
    'get',
    `${base}/connections/{id}/status`,
    { params: { path: { workspace: slug, id: secret.integrationId } } },
    { refetchInterval: (q) => (q.state.data && isDone(q.state.data, wp) ? false : 3000), staleTime: 0 },
  )
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [])
  const done = status.data && isDone(status.data, wp)
  const late = !done && now - since > NO_CALL_AFTER

  if (done && status.data) {
    const fc = status.data.firstCall
    const at = fc ? new Date(fc.at) : status.data.reported ? new Date(status.data.reported) : new Date()
    return (
      <>
        <span className="k -mt-2">{clock(at)}</span>
        <Steps kind={kind} step="wait" />
        <div className="flex flex-col">
          <Ev icon="ok" title={t(m.evKeyCreated)} sub={t(m.evKeyCreatedLine, { time: clock(createdAt), key: secret.key.name, count })} time={clockSeconds(createdAt)} />
          <Ev
            icon="ok"
            title={wp ? t(m.wpReportedAt, { host, time: clock(at) }) : t(m.evCalledAt, { name, time: clock(at) })}
            sub={
              fc
                ? fc.ip || fc.client
                  ? t(m.evCalledSub, { operation: operationLabel(t, fc.operation), status: fc.status, ip: fc.ip ?? '', client: fc.client ?? '' })
                  : t(m.evCalledSubShort, { operation: operationLabel(t, fc.operation), status: fc.status })
                : undefined
            }
            time={clockSeconds(at)}
          />
          <Ev icon="ok" title={t(m.evConnected)} sub={wp ? t(m.wpConnectedSub) : t(m.evReady, { kind })} time={clockSeconds(at)} />
        </div>
        <Note tone="ok">{wp ? t(m.wpConnectedNote) : t(m.connectedNote, { name })}</Note>
        <Footer>
          <Button variant="primary" onClick={onDone}>
            {t(m.done)}
          </Button>
        </Footer>
      </>
    )
  }

  if (late) {
    const minutes = Math.floor((now - createdAt.getTime()) / 60_000)
    return (
      <>
        <span className="k -mt-2">{t(m.afterMinutes)}</span>
        <Steps kind={kind} step="wait" />
        <Ev icon="warn" title={t(m.noCallYet)} sub={t(m.noCallSub, { key: secret.key.name, time: clock(createdAt), minutes })} />
        <Note tone="warn">{t(m.noCallNote)}</Note>
        <KeyValues
          className="text-[12.5px]"
          items={[
            [t(m.instanceUrl), <span className="mono">{secret.instanceUrl}</span>],
            [t(m.key), <T msg={m.keyShownAt} values={{ ...tags, key: secret.key.name, masked: secret.key.masked, time: clock(createdAt) }} />],
          ]}
        />
        <Footer>
          <Button
            icon="copy"
            onClick={async () => {
              const ok = await copyText(secret.instanceUrl)
              toast({ kind: ok ? 'success' : 'error', title: t(m.copiedToast) })
            }}
          >
            {t(m.copyUrl)}
          </Button>
          <Button onClick={onClose}>{t(m.close)}</Button>
          <Button variant="primary" onClick={() => setSince(Date.now())}>
            {t(m.keepWaiting)}
          </Button>
        </Footer>
      </>
    )
  }

  return (
    <>
      <span className="k -mt-2">{wp ? t(m.wpWaitEyebrow) : t(m.listening)}</span>
      <Steps kind={kind} step="wait" />
      <div className="flex flex-col" aria-live="polite">
        <Ev
          icon="live"
          title={wp ? t(m.wpWaiting, { host }) : t(m.waitingFirstCall, { name })}
          sub={t(m.waitingSub, { key: secret.key.name, time: clock(createdAt), duration: listeningFor(now - createdAt.getTime(), t) })}
        />
        <Ev icon="none" title={wp ? t(m.wpFirstReport) : t(m.evTestConnection)} sub={wp ? t(m.wpFirstReportSub) : t(m.evTestConnectionSub, { name })} />
        <Ev icon="none" title={t(m.evConnected)} sub={wp ? t(m.wpConnectedSub) : t(m.evConnectedCard, { name })} />
      </div>
      <span className="k">{wp ? t(m.wpWaitFoot) : t(m.closeKeeps, { name })}</span>
      <Footer>
        <Button onClick={onClose}>{t(m.close)}</Button>
        <Button variant="primary" disabledReason={wp ? t(m.wpWaiting, { host }) : t(m.waitingFirstCall, { name })}>
          {t(m.done)}
        </Button>
      </Footer>
    </>
  )
}

function isDone(s: Status, wp: boolean): boolean {
  return wp ? Boolean(s.reported) : Boolean(s.firstCall)
}

/** One line of the live list: a mark, what happened, and when. */
export function Ev({ icon, title, sub, time }: { icon: 'ok' | 'live' | 'warn' | 'bad' | 'none'; title: ReactNode; sub?: ReactNode; time?: string }) {
  return (
    <div className="ih-ev">
      <span className={`ih-ic${icon === 'none' ? '' : ` ${icon}`}`} aria-hidden="true">
        {icon === 'ok' ? <Icon name="check" size={14} strokeWidth={2.4} /> : icon === 'live' ? <span className="ih-spin" /> : icon === 'warn' ? <Icon name="clock" size={14} /> : icon === 'bad' ? <Icon name="close" size={14} /> : null}
      </span>
      <div className="ih-tx">
        <span>{title}</span>
        {sub ? <span className="k">{sub}</span> : null}
      </div>
      {time ? <span className="ih-tm">{time}</span> : null}
    </div>
  )
}

