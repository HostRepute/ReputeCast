import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { Footer } from '@/features/integrations/ConnectDialog'
import { AttentionIcon } from '@/features/integrations/shared'
import '@/features/integrations/integrations.css'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { ago, clock } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { storesBase, useConnectState, useRefreshStores, type ConnectState, type Platform } from './api'
import { cc, cm } from './messages'
import './commerce.css'

/** The dialog's screens: Shopify's five steps, WooCommerce's four. */
type View = 'address' | 'app' | 'keys' | 'install' | 'sync' | 'key' | 'webhooks' | 'plugin'

const shopifySteps: View[] = ['address', 'app', 'keys', 'install', 'sync']
const wooSteps: View[] = ['address', 'key', 'webhooks', 'plugin']

const stepLabel: Record<View, MessageDescriptor> = {
  address: cc.stepAddress,
  app: cc.stepApp,
  keys: cc.stepKeys,
  install: cc.stepInstall,
  sync: cc.stepSync,
  key: cc.stepKey,
  webhooks: cc.stepWebhooks,
  plugin: cc.stepPlugin,
}

const bold = (c: ReactNode[]) => <b className="font-semibold">{c}</b>
const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

/** The screen a store's server step opens on. */
function viewOf(st: ConnectState): View {
  switch (st.step) {
    case 'app':
    case 'keys':
    case 'install':
    case 'sync':
    case 'key':
    case 'plugin':
      return st.step
    case 'webhooks':
      return 'key'
    case 'done':
      return st.store.platform === 'woocommerce' ? 'plugin' : 'sync'
  }
  return 'address'
}

export type ConnectMode = 'connect' | 'reconnect' | 'settings'

/**
 * Connect Shopify (the address, the app to create, its keys, Shopify's
 * approval, what syncs) or WooCommerce (the address, a REST API key, the
 * webhooks, the plugin). A store already started resumes where it stopped.
 * "reconnect" starts again from the keys or the approval; "settings" is a
 * connected store's "Change what syncs".
 */
export function ConnectStoreDialog({
  platform,
  storeId,
  error,
  mode = 'connect',
  onOpenChange,
}: {
  platform: Platform
  storeId?: string
  /** What went wrong when Shopify sent the browser back ("secret_refused"). */
  error?: string
  mode?: ConnectMode
  onOpenChange: (open: boolean) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const [id, setId] = useState(storeId)
  const [view, setView] = useState<View | null>(null)
  const reconnect = $api.useMutation('post', `${storesBase}/{id}/reconnect`)
  const started = useRef(false)
  const [ready, setReady] = useState(mode !== 'reconnect')
  const q = useConnectState(ready ? id : undefined)
  const refresh = useRefreshStores()
  const st = q.data
  useEffect(() => {
    if (mode !== 'reconnect' || !storeId || started.current) return
    started.current = true
    reconnect.mutate(
      { params: { path: { workspace: slug, id: storeId } } },
      {
        onSuccess: (s) => {
          setView(viewOf(s))
          setReady(true)
        },
        onError: (e) => {
          toast({ kind: 'error', title: errorMessage(e) })
          onOpenChange(false)
        },
      },
    )
  }, [mode, storeId, slug, reconnect, onOpenChange])
  const close = (o: boolean) => {
    if (!o) void refresh()
    onOpenChange(o)
  }
  const shopify = platform === 'shopify'
  const settings = mode === 'settings'
  const steps = shopify ? shopifySteps : wooSteps
  // Until a step is chosen here, the store's own step; a new store starts at its address.
  const shown: View | null = view ?? (st ? viewOf(st) : id ? null : 'address')
  const current = shown ?? 'address'
  const at = Math.max(0, steps.indexOf(current))
  const name = st?.store.name ?? ''
  const eyebrow = shopify ? t(cc.eyebrowShopify) : st?.store.demo ? t(cc.eyebrowWooSample) : t(cc.eyebrowWoo)
  const title = settings ? t(cm.changeSyncs) : mode === 'reconnect' && name ? t(cc.titleReconnect, { name }) : shopify ? t(cc.titleShopify) : t(cc.titleWoo)
  const errorCode = st?.error || error
  const loading = shown === null || (!!id && !st)
  return (
    <Dialog
      open
      onOpenChange={close}
      title={title}
      eyebrow={<span className="k">{eyebrow}</span>}
      size="md"
      className="max-w-[720px]"
      hideClose
      dismissible={current !== 'keys' && current !== 'key'}
    >
      {settings ? null : <Stepper steps={steps} at={at} />}
      {q.isError ? (
        <Notice tone="b">{errorMessage(q.error)}</Notice>
      ) : loading && current !== 'address' ? (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton w="60%" h={14} />
          <Skeleton w="100%" h={36} />
          <Skeleton w="100%" h={36} />
        </div>
      ) : current === 'address' || !st ? (
        <Address
          platform={platform}
          onCancel={() => close(false)}
          onDone={(s) => {
            setId(s.store.id)
            setView(viewOf(s))
          }}
        />
      ) : current === 'app' ? (
        <App st={st} onBack={() => setView('address')} onNext={() => setView('keys')} />
      ) : current === 'keys' ? (
        <Keys
          st={st}
          refused={errorCode === 'secret_refused'}
          onBack={() => setView('app')}
          onDone={() => {
            void q.refetch()
            setView('install')
          }}
        />
      ) : current === 'install' ? (
        <Install st={st} problem={errorCode} onBack={() => setView('keys')} />
      ) : current === 'sync' ? (
        <Sync st={st} settings={settings} onBack={() => setView('install')} onDone={() => close(false)} />
      ) : current === 'key' || current === 'webhooks' ? (
        <WooKey
          st={st}
          onBack={() => setView('address')}
          onDone={(s) => {
            void q.refetch()
            setView(viewOf(s))
          }}
        />
      ) : (
        <Plugin st={st} onBack={() => setView('key')} onDone={() => close(false)} />
      )}
    </Dialog>
  )
}

/** "1 Store address › 2 Create the app › …", the steps before ticked. */
function Stepper({ steps, at }: { steps: View[]; at: number }) {
  const t = useT()
  return (
    <ol className="ih-stp -mt-1" aria-label={t(cc.steps)}>
      {steps.flatMap((s, i) => [
        i > 0 ? (
          <li key={`${s}-sep`} className="ih-sep" aria-hidden="true">
            ›
          </li>
        ) : null,
        <li key={s} className={i < at ? 'done' : i === at ? 'cur' : undefined} aria-current={i === at ? 'step' : undefined}>
          <span className="ih-n" aria-hidden={i < at ? true : undefined}>
            {i < at ? <Icon name="check" size={11} strokeWidth={2.4} /> : i + 1}
          </span>
          {t(stepLabel[s])}
        </li>,
      ])}
    </ol>
  )
}

/** The boards' note with its icon: a warning (w) or a problem (b). */
function Notice({ tone, children }: { tone: 'w' | 'b'; children: ReactNode }) {
  return (
    <div className={`note ${tone}`} role={tone === 'b' ? 'alert' : 'note'}>
      <AttentionIcon size={16} color={tone === 'b' ? 'var(--bad-fg)' : '#B45309'} />
      <div className="min-w-0 flex-[1_1_320px]">{children}</div>
    </div>
  )
}

/** A field's error under it, in the boards' red. */
/**
 * A field's own message in the server's words, as typed ("acme-merch.com
 * doesn't answer like a Shopify store."), else the problem's sentence.
 */
function fieldMessage(e: unknown): string {
  return (e instanceof ApiError && e.problem.errors?.[0]?.message) || errorMessage(e)
}

function FieldError({ id, children }: { id: string; children: ReactNode }) {
  return (
    <span className="k font-medium" id={id}>
      <span className="font-semibold text-bad-fg">{children}</span>
    </span>
  )
}

function Address({ platform, onCancel, onDone }: { platform: Platform; onCancel: () => void; onDone: (s: ConnectState) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [address, setAddress] = useState('')
  const [err, setErr] = useState<string>()
  const m = $api.useMutation('post', storesBase)
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        setErr(undefined)
        m.mutate({ params: { path: { workspace: slug } }, body: { platform, address: address.trim() } }, { onSuccess: onDone, onError: (e) => setErr(fieldMessage(e)) })
      }}
    >
      <label className="lbl">
        {t(cc.address)}
        <input
          className="in"
          type="text"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          aria-invalid={err ? true : undefined}
          aria-describedby={err ? 'cs-address-err' : undefined}
          style={err ? { borderColor: '#B91C1C' } : undefined}
          autoComplete="off"
          spellCheck={false}
          autoFocus
        />
        {err ? <FieldError id="cs-address-err">{err}</FieldError> : null}
      </label>
      <span className="k">{t(platform === 'shopify' ? cc.addressHint : cc.addressHintWoo)}</span>
      <Footer>
        <Button className="mr-auto" onClick={onCancel}>
          {t(cc.cancel)}
        </Button>
        <Button type="submit" variant="primary" loading={m.isPending} disabled={!address.trim()}>
          {t(cc.continue)}
        </Button>
      </Footer>
    </form>
  )
}

function CopyField({ label, value, copyLabel }: { label: string; value: string; copyLabel: string }) {
  const t = useT()
  const [done, setDone] = useState(false)
  return (
    <div className="flex flex-col gap-1">
      <span className="lbl">{label}</span>
      <div className="ih-inl">
        <input className="in mono text-[12px]" value={value} readOnly aria-label={label} onFocus={(e) => e.currentTarget.select()} />
        <Button
          size="sm"
          icon={done ? 'check' : 'copy'}
          aria-label={copyLabel}
          onClick={async () => {
            const ok = await copyText(value)
            setDone(ok)
            if (ok) toast({ kind: 'success', title: t(cc.copied) })
          }}
        >
          {t(cc.copy)}
        </Button>
      </div>
    </div>
  )
}

function Scopes({ st, optional }: { st: ConnectState; optional?: boolean }) {
  const t = useT()
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {[...st.scopes, st.optionalScope].map((s) => (
        <span key={s} className="ih-mk">
          {s}
        </span>
      ))}
      {optional ? <span className="k">{t(cc.optional)}</span> : null}
    </div>
  )
}

function Check({ children }: { children: ReactNode }) {
  return (
    <span>
      <Icon name="check" size={13} strokeWidth={2.4} className="flex-none text-ok-fg" />
      {children}
    </span>
  )
}

function App({ st, onBack, onNext }: { st: ConnectState; onBack: () => void; onNext: () => void }) {
  const t = useT()
  return (
    <>
      <span className="text-ink-nav">
        <T msg={cc.appIntro} values={{ domain: st.store.domain, b: bold }} />
      </span>
      <CopyField label={t(cc.appUrl)} value={st.appUrl} copyLabel={t(cc.copyAppUrl)} />
      <CopyField label={t(cc.redirectUrl)} value={st.redirectUrl} copyLabel={t(cc.copyRedirect)} />
      <div className="flex flex-col gap-1.5">
        <span className="lbl">{t(cc.access)}</span>
        <Scopes st={st} optional />
        <span className="k">{t(cc.accessNote)}</span>
      </div>
      <div className="flex flex-col gap-1">
        <span className="lbl">{t(cc.protectedData)}</span>
        <span className="text-[13px] text-ink-nav">{t(cc.protectedText)}</span>
      </div>
      {st.reachable ? (
        <div className="ih-chk3 cm-wrap">
          <Check>{st.reachableCheckedAt ? t(cc.reachable, { ago: ago(st.reachableCheckedAt) }) : t(cc.reachableNow)}</Check>
        </div>
      ) : (
        <Notice tone="w">{t(cc.unreachable)}</Notice>
      )}
      <Footer>
        <Button className="mr-auto" onClick={onBack}>
          {t(cc.back)}
        </Button>
        <ButtonAnchor href="https://dev.shopify.com/dashboard" iconRight="external">
          {t(cc.devDashboard)}
          <span className="vh"> {t(cm.newTab)}</span>
        </ButtonAnchor>
        <Button variant="primary" onClick={onNext}>
          {t(cc.continue)}
        </Button>
      </Footer>
    </>
  )
}

function Keys({ st, refused, onBack, onDone }: { st: ConnectState; refused: boolean; onBack: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [clientId, setClientId] = useState('')
  const [secret, setSecret] = useState('')
  const [err, setErr] = useState<string | undefined>(refused ? t(cc.secretRefused) : undefined)
  const m = $api.useMutation('put', `${storesBase}/{id}/app-keys`)
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        setErr(undefined)
        m.mutate(
          { params: { path: { workspace: slug, id: st.store.id } }, body: { clientId: clientId.trim(), clientSecret: secret.trim() } },
          { onSuccess: onDone, onError: (e) => setErr(fieldMessage(e)) },
        )
      }}
    >
      {err ? null : <span className="text-ink-nav">{t(cc.keysIntro)}</span>}
      <label className="lbl">
        {t(cc.clientId)}
        {/* Keys are write-only: a saved app shows its client ID's hint, to paste again. */}
        <input
          className="in mono"
          type="text"
          value={clientId}
          onChange={(e) => setClientId(e.target.value)}
          placeholder={st.clientIdHint}
          autoComplete="off"
          spellCheck={false}
        />
      </label>
      <label className="lbl">
        {t(cc.clientSecret)}
        <input
          className="in mono"
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          autoComplete="off"
          aria-invalid={err ? true : undefined}
          aria-describedby="cs-secret-note"
          style={err ? { borderColor: '#B91C1C' } : undefined}
        />
        {err ? (
          <FieldError id="cs-secret-note">{err}</FieldError>
        ) : (
          <span className="k font-medium" id="cs-secret-note">
            {t(cc.secretNote)}
          </span>
        )}
      </label>
      <Footer>
        <Button className="mr-auto" onClick={onBack}>
          {t(cc.back)}
        </Button>
        <Button type="submit" variant="primary" loading={m.isPending} disabled={!clientId.trim() || !secret.trim()}>
          {err ? t(cc.tryAgain) : t(cc.continue)}
        </Button>
      </Footer>
    </form>
  )
}

function Install({ st, problem, onBack }: { st: ConnectState; problem?: string; onBack: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [waiting, setWaiting] = useState(false)
  const m = $api.useMutation('post', `${storesBase}/{id}/install`)
  const note = problem === 'state_expired' ? t(cc.stateExpired) : problem === 'unavailable' ? t(cc.unavailable) : ''
  return (
    <>
      {note ? <Notice tone="w">{note}</Notice> : null}
      <span className="text-ink-nav">{t(cc.installIntro)}</span>
      <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
        <span className="font-semibold">{t(cc.installAble)}</span>
        <Scopes st={st} />
        <span className="k">{t(cc.installScopes)}</span>
      </div>
      <div>
        <Button
          variant="primary"
          className="min-h-10"
          iconRight="external"
          loading={m.isPending || waiting}
          onClick={() =>
            m.mutate(
              { params: { path: { workspace: slug, id: st.store.id } } },
              {
                onSuccess: (r) => {
                  setWaiting(true)
                  window.location.assign(r.url)
                },
                onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
              },
            )
          }
        >
          {t(cc.installOn, { domain: st.store.domain })}
        </Button>
      </div>
      <Footer>
        <Button className="mr-auto" onClick={onBack}>
          {t(cc.back)}
        </Button>
        <span className="k" role="status">
          {t(cc.waiting)}
        </span>
      </Footer>
    </>
  )
}

function Sync({ st, settings, onBack, onDone }: { st: ConnectState; settings: boolean; onBack: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const storeList = st.lists.find((l) => l.store && l.name.toLowerCase() === st.newListName.toLowerCase())
  const initial = settings ? st.syncs : { products: true, customers: true, orders: true, checkouts: true, writeBack: st.writeGranted, listId: storeList?.id }
  const [products, setProducts] = useState(initial.products)
  const [customers, setCustomers] = useState(initial.customers)
  const [list, setList] = useState<string>(initial.listId ?? 'new')
  const [orders, setOrders] = useState(initial.orders)
  const [months, setMonths] = useState<3 | 6 | 12>(st.months === 3 || st.months === 6 ? st.months : 12)
  const [checkouts, setCheckouts] = useState(initial.checkouts)
  const [writeBack, setWriteBack] = useState(initial.writeBack && st.writeGranted)
  const start = $api.useMutation('post', `${storesBase}/{id}/start`)
  const save = $api.useMutation('patch', `${storesBase}/{id}/settings`)
  const path = { workspace: slug, id: st.store.id }
  const listId = list === 'new' ? undefined : list
  const submit = () => {
    if (settings) {
      save.mutate(
        { params: { path }, body: { products, customers, orders, checkouts, writeBack: writeBack && st.writeGranted, listId } },
        {
          onSuccess: () => {
            toast({ kind: 'success', title: t(cm.saved) })
            onDone()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      )
      return
    }
    start.mutate(
      { params: { path }, body: { products, customers, orders, months, checkouts, writeBack: writeBack && st.writeGranted, listId, newList: customers && list === 'new' } },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: t(cc.started, { name: st.store.name }) })
          onDone()
          void navigate({ to: '/w/$workspace/settings/integrations/stores/$storeId', params: { workspace: slug, storeId: st.store.id }, search: {} })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  }
  const row = 'flex items-start gap-2.5 text-[13px]'
  return (
    <>
      {settings ? null : (
        <div className="ih-chk3 cm-wrap flex-col !items-start !gap-1.5">
          {st.clientIdHint ? (
            <Check>
              <T msg={cc.syncClient} values={{ hint: st.clientIdHint, mono }} />
            </Check>
          ) : null}
          {st.installedAt ? <Check>{t(cc.syncInstalled, { domain: st.store.domain, time: clock(st.installedAt) })}</Check> : null}
        </div>
      )}
      <fieldset className="m-0 flex flex-col gap-2.5 border-0 p-0">
        <legend className="lbl mb-1.5">{t(cc.syncWhat)}</legend>
        <label className={row}>
          <input type="checkbox" className="mt-0.5" checked={products} onChange={(e) => setProducts(e.target.checked)} />
          <span>
            <T msg={cc.syncProducts} values={{ b: bold }} />
          </span>
        </label>
        <div className={row}>
          <input id="cs-customers" type="checkbox" className="mt-0.5" checked={customers} onChange={(e) => setCustomers(e.target.checked)} />
          <span className="flex min-w-0 flex-1 flex-col gap-1.5">
            <label htmlFor="cs-customers">
              <T msg={cc.syncCustomers} values={{ b: bold }} />
            </label>
            <select className="in" aria-label={t(cc.syncListLabel)} value={list} disabled={!customers} onChange={(e) => setList(e.target.value)}>
              {storeList ? null : <option value="new">{t(cc.syncNewList, { name: st.newListName })}</option>}
              {st.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </span>
        </div>
        <Notice tone="w">{t(cm.consentReminder, { store: st.store.name })}</Notice>
        <div className={`${row} flex-wrap !items-center`}>
          <input id="cs-orders" type="checkbox" checked={orders} onChange={(e) => setOrders(e.target.checked)} />
          <label htmlFor="cs-orders">
            <T msg={cc.syncOrders} values={{ b: bold }} />
          </label>
          <span className="seg" role="group" aria-label={t(cc.syncOrdersLabel)}>
            {([3, 6, 12] as const).map((n) => (
              <button key={n} type="button" aria-pressed={months === n} className={months === n ? 'on' : undefined} disabled={!orders || settings} onClick={() => setMonths(n)}>
                {t(cc.months, { n })}
              </button>
            ))}
          </span>
        </div>
        <label className={row}>
          <input type="checkbox" className="mt-0.5" checked={checkouts} onChange={(e) => setCheckouts(e.target.checked)} />
          <span>
            <T msg={cc.syncCheckouts} values={{ b: bold }} />
          </span>
        </label>
        {st.store.platform === 'shopify' ? (
          <label className={row}>
            <input type="checkbox" className="mt-0.5" checked={writeBack} disabled={!st.writeGranted} onChange={(e) => setWriteBack(e.target.checked)} />
            <span>
              <T msg={st.writeGranted ? cc.syncWriteBack : cc.syncWriteBackOff} values={{ b: bold }} />
            </span>
          </label>
        ) : null}
      </fieldset>
      <Footer>
        {settings ? (
          <Button className="mr-auto" onClick={onDone}>
            {t(cc.cancel)}
          </Button>
        ) : (
          <Button className="mr-auto" onClick={onBack}>
            {t(cc.back)}
          </Button>
        )}
        <Button variant="primary" onClick={submit} loading={start.isPending || save.isPending}>
          {settings ? t(cc.save) : t(cc.startSync)}
        </Button>
      </Footer>
    </>
  )
}

function WooKey({ st, onBack, onDone }: { st: ConnectState; onBack: () => void; onDone: (s: ConnectState) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [key, setKey] = useState('')
  const [secret, setSecret] = useState('')
  const [err, setErr] = useState<string>()
  const m = $api.useMutation('put', `${storesBase}/{id}/rest-key`)
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        setErr(undefined)
        m.mutate({ params: { path: { workspace: slug, id: st.store.id } }, body: { key: key.trim(), secret: secret.trim() } }, { onSuccess: onDone, onError: (e) => setErr(fieldMessage(e)) })
      }}
    >
      {err ? null : <span className="text-ink-nav">{t(cc.keyIntro)}</span>}
      <label className="lbl">
        {t(cc.consumerKey)}
        <input className="in mono" type="text" value={key} onChange={(e) => setKey(e.target.value)} placeholder={'ck_…'} autoComplete="off" spellCheck={false} />
      </label>
      <label className="lbl">
        {t(cc.consumerSecret)}
        <input
          className="in mono"
          type="password"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
          placeholder={'cs_…'}
          autoComplete="off"
          aria-invalid={err ? true : undefined}
          aria-describedby="cs-key-note"
          style={err ? { borderColor: '#B91C1C' } : undefined}
        />
        {err ? (
          <FieldError id="cs-key-note">{err}</FieldError>
        ) : (
          <span className="k font-medium" id="cs-key-note">
            {t(cc.keyNote)}
          </span>
        )}
      </label>
      <Footer>
        <Button className="mr-auto" onClick={onBack}>
          {t(cc.back)}
        </Button>
        <Button type="submit" variant="primary" loading={m.isPending} disabled={!key.trim() || !secret.trim()}>
          {err ? t(cc.tryAgain) : t(cc.continue)}
        </Button>
      </Footer>
    </form>
  )
}

function Plugin({ st, onBack, onDone }: { st: ConnectState; onBack: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshStores()
  const check = $api.useMutation('post', `${storesBase}/{id}/plugin-check`)
  const finish = $api.useMutation('post', `${storesBase}/{id}/finish`)
  const path = { workspace: slug, id: st.store.id }
  const host = st.plugin?.host || st.store.address
  const found = !!st.plugin?.found
  const done = (withoutPlugin: boolean) =>
    finish.mutate(
      { params: { path }, body: { withoutPlugin } },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: t(cc.connectedWoo, { name: st.store.name }) })
          onDone()
          void navigate({ to: '/w/$workspace/settings/integrations/stores/$storeId', params: { workspace: slug, storeId: st.store.id }, search: {} })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  return (
    <>
      <div className="ih-chk3 cm-wrap flex-col !items-start !gap-1.5">
        <Check>
          <T msg={cc.wooFound} values={{ host: st.store.address, mono }} />
        </Check>
        <Check>{t(cc.wooKey)}</Check>
        <Check>{t(cc.wooHooks, { count: st.webhooks })}</Check>
      </div>
      <div className="flex flex-col gap-2 rounded-lg border border-line px-3.5 py-3">
        <span className="font-semibold">{t(cc.pluginOn, { host })}</span>
        <span className="k text-ink-nav">{t(cc.pluginWhy)}</span>
        {found ? (
          <span className="flex items-center gap-2 text-[13px] font-semibold text-ok-fg" role="status">
            <Icon name="check" size={14} strokeWidth={2.4} />
            {t(cc.pluginActive, { host, version: st.plugin?.version || 'none' })}
          </span>
        ) : (
          <>
            <span className="flex items-center gap-2 text-[13px] font-semibold text-[#991B1B]" role="status">
              <AttentionIcon size={14} color="#991B1B" />
              {t(cc.pluginNotFound, { host })}
            </span>
            <div className="flex flex-wrap gap-2">
              <ButtonLink size="sm" variant="primary" to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} search={{ connect: true }}>
                {t(cm.installPlugin)}
              </ButtonLink>
              <Button
                size="sm"
                loading={check.isPending}
                onClick={() =>
                  check.mutate(
                    { params: { path } },
                    {
                      onSuccess: (s) => {
                        void refresh()
                        if (!s.plugin?.found) toast({ kind: 'info', title: t(cm.pluginStillMissing, { host }) })
                      },
                      onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                    },
                  )
                }
              >
                {t(cm.checkAgain)}
              </Button>
            </div>
          </>
        )}
      </div>
      {found ? null : <span className="k">{t(cc.pluginWithout)}</span>}
      <Footer>
        <Button className="mr-auto" onClick={onBack}>
          {t(cc.back)}
        </Button>
        {found ? null : (
          <Button onClick={() => done(true)} disabled={finish.isPending}>
            {t(cc.withoutPlugin)}
          </Button>
        )}
        <Button variant="primary" onClick={() => done(false)} disabled={!found} loading={finish.isPending}>
          {t(cc.finish)}
        </Button>
      </Footer>
    </>
  )
}
