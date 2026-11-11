import { useState, type ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { SwitchInput } from '@/components/ui/Field'
import { StepTrail } from '@/components/ui/Stepper'
import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { storesBase } from '@/features/commerce/api'
import { T, formatList, useT } from '@/i18n'
import { clock, dayMonthYear } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { originOf, secretPath, sitePath, sitesPath, useRecentEvents, useRefreshSites, type Site, type SiteList } from './api'
import { tm } from './messages'
import { EventsList } from './TrackingPage'

type Step = 'site' | 'snippet' | 'first'
type Tab = 'script' | 'pixel' | 'wordpress'

const code = (v: string) => <span className="mono text-[12.5px]">{v}</span>
const platformName = (p?: string) => (p === 'woocommerce' ? 'WooCommerce' : 'Shopify')

/** The stores' flows a site feeds: its linked store's browse, cart, stock and price-drop flows. */
const trackerKinds = new Set(['viewed_product', 'added_to_cart', 'back_in_stock', 'price_drop'])

/**
 * Add a site (the site › the snippet with the identify secret › the first
 * event), set an existing one up again (the snippet without the secret),
 * or edit it (TrackingSetup).
 */
export function SiteDialog({
  list,
  mode,
  site: given,
  origin,
  onClose,
  onAdded,
}: {
  list: SiteList
  mode: 'add' | 'setup' | 'edit'
  site?: Site
  origin?: string
  onClose: () => void
  onAdded?: (id: string) => void
}) {
  const t = useT()
  const [added, setAdded] = useState<{ site: Site; secret: string }>()
  const [step, setStep] = useState<Step>(mode === 'setup' ? 'snippet' : 'site')
  const site = added?.site ?? (given ? (list.items.find((s) => s.id === given.id) ?? given) : undefined)
  const at = step === 'site' ? 1 : step === 'snippet' ? 2 : 3
  const trail =
    mode === 'edit' ? null : (
      <StepTrail
        className="-mt-1"
        label={t(tm.steps)}
        current={at}
        steps={[
          { key: 'site', title: t(tm.stepSite) },
          { key: 'snippet', title: t(tm.stepSnippet) },
          { key: 'first', title: t(tm.stepFirst) },
        ]}
      />
    )
  if (mode === 'edit' && site) return <SiteForm list={list} site={site} onClose={onClose} />
  if (step === 'site' || !site) {
    return (
      <SiteForm
        list={list}
        origin={origin}
        trail={trail}
        onClose={onClose}
        onCreated={(s, secret) => {
          setAdded({ site: s, secret })
          setStep('snippet')
          onAdded?.(s.id)
        }}
      />
    )
  }
  if (step === 'snippet') {
    return <Snippet site={site} secret={added?.secret} fresh={!!added} trail={trail} onClose={onClose} onNext={() => setStep('first')} />
  }
  return <FirstEvent site={site} trail={trail} onClose={onClose} />
}

/** Step 1 (Add a site) or Edit {site}: name, origin, store and the consent switch. */
function SiteForm({
  list,
  site,
  origin: initialOrigin,
  trail,
  onClose,
  onCreated,
}: {
  list: SiteList
  site?: Site
  origin?: string
  trail?: ReactNode
  onClose: () => void
  onCreated?: (s: Site, secret: string) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSites()
  const editing = !!site
  const [name, setName] = useState(site?.name ?? '')
  const [origin, setOrigin] = useState(site?.origin ?? initialOrigin ?? '')
  const [store, setStore] = useState(site ? (site.store?.id ?? '') : list.stores.length === 1 ? list.stores[0]!.id : '')
  const [wait, setWait] = useState(site?.waitConsent ?? true)
  const [tried, setTried] = useState(false)
  const [serverError, setServerError] = useState<{ field: 'name' | 'origin' | 'form'; text: string }>()
  const create = $api.useMutation('post', sitesPath)
  const update = $api.useMutation('put', sitePath)
  const parsed = originOf(origin)
  const nameError = tried && (!name.trim() || name.trim().length > 80) ? t(tm.nameMissing) : serverError?.field === 'name' ? serverError.text : undefined
  const originError = editing
    ? undefined
    : tried && parsed.path
      ? t(tm.originPath, { origin: parsed.path })
      : tried && !parsed.origin
        ? t(tm.originBad)
        : serverError?.field === 'origin'
          ? serverError.text
          : undefined
  const failed = (e: unknown) => {
    const text = errorMessage(e)
    const loc = JSON.stringify((e as { errors?: unknown })?.errors ?? '')
    setServerError({ field: loc.includes('origin') ? 'origin' : loc.includes('name') ? 'name' : 'form', text })
  }
  const submit = () => {
    setTried(true)
    setServerError(undefined)
    if (!name.trim() || name.trim().length > 80) return
    if (editing) {
      update.mutate(
        { params: { path: { workspace: slug, site: site.id } }, body: { name: name.trim(), storeId: store || undefined, waitConsent: wait } },
        {
          onSuccess: (s) => {
            void refresh()
            toast({ kind: 'success', title: t(tm.saved, { name: s.name }) })
            onClose()
          },
          onError: failed,
        },
      )
      return
    }
    if (!parsed.origin) return
    create.mutate(
      { params: { path: { workspace: slug } }, body: { name: name.trim(), origin: parsed.origin, storeId: store || undefined, waitConsent: wait } },
      {
        onSuccess: (out) => {
          void refresh()
          toast({ kind: 'success', title: t(tm.added, { name: out.site.name }) })
          onCreated?.(out.site, out.identifySecret)
        },
        onError: failed,
      },
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={editing ? t(tm.editTitle, { name: site.name }) : t(tm.addTitle)}
      eyebrow={
        <span className="k">
          {editing
            ? site.createdBy
              ? t(tm.eyebrowAddedBy, { name: site.name, by: site.createdBy, date: dayMonthYear(site.createdAt) })
              : t(tm.eyebrowAdded, { name: site.name, when: dayMonthYear(site.createdAt) })
            : t(tm.eyebrowPath)}
        </span>
      }
      className="max-w-[640px]"
      hideClose
      footer={
        <>
          {editing ? <span className="k mr-auto">{t(tm.audited)}</span> : null}
          <Button onClick={onClose}>{t(tm.cancel)}</Button>
          <Button variant="primary" onClick={submit} loading={create.isPending || update.isPending}>
            {editing ? t(tm.save) : t(tm.addSite)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault()
          submit()
        }}
      >
        {trail}
        {serverError?.field === 'form' ? (
          <div className="note b" role="alert">
            <Icon name="alertCircle" size={16} className="mt-px flex-none text-bad-strong" />
            <div className="min-w-0 flex-[1_1_320px]">{serverError.text}</div>
          </div>
        ) : null}
        <div className="lbl">
          <label htmlFor="tk-name">{t(tm.name)}</label>
          <input id="tk-name" className="in" value={name} maxLength={80} aria-invalid={nameError ? true : undefined} aria-describedby="tk-name-hint" onChange={(e) => setName(e.target.value)} />
          <span className={nameError ? 'k text-bad-fg' : 'k'} id="tk-name-hint" role={nameError ? 'alert' : undefined}>
            {nameError ?? t(tm.nameHint)}
          </span>
        </div>
        <div className="lbl">
          <label htmlFor="tk-origin">{t(tm.origin)}</label>
          <input
            id="tk-origin"
            className="in mono"
            value={origin}
            readOnly={editing}
            inputMode="url"
            autoComplete="off"
            spellCheck={false}
            aria-invalid={originError ? true : undefined}
            aria-describedby="tk-origin-hint"
            onChange={(e) => {
              setOrigin(e.target.value)
              if (serverError?.field === 'origin') setServerError(undefined)
            }}
          />
          <span className={originError ? 'k text-bad-fg' : 'k'} id="tk-origin-hint" role={originError ? 'alert' : undefined}>
            {originError ?? (editing ? t(tm.originFixed) : t(tm.originHint))}
          </span>
        </div>
        <div className="lbl">
          <label htmlFor="tk-store">{t(tm.store)}</label>
          <select id="tk-store" className="in" value={store} aria-describedby="tk-store-hint" onChange={(e) => setStore(e.target.value)}>
            <option value="">{t(tm.notLinked)}</option>
            {list.stores.map((s) => (
              <option key={s.id} value={s.id}>
                {t(tm.storeOption, { name: s.name, platform: platformName(s.platform) })}
              </option>
            ))}
          </select>
          <span className="k" id="tk-store-hint">
            {t(tm.storeHint)}
          </span>
        </div>
        <div className="cm-srow items-center">
          <span>
            <b className="text-[13px] font-semibold">{t(tm.wait)}</b>
            <span className="k" id="tk-wait-sub">
              <T msg={tm.waitSub} values={{ call: code("reputecast('consent', true)") }} />
            </span>
          </span>
          <SwitchInput aria-label={t(tm.wait)} aria-describedby="tk-wait-sub" checked={wait} onChange={(e) => setWait(e.target.checked)} />
        </div>
        <span className="k">{t(tm.waitNote)}</span>
        <button type="submit" hidden />
      </form>
    </Dialog>
  )
}

/** A dark code box with its language and a Copy button. */
function CodeBox({ lang, copy, label, children }: { lang: string; copy: string; label: string; children: ReactNode }) {
  const t = useT()
  return (
    <div className="ih-cbx">
      <div className="ih-cbh tk-cbh">
        <button type="button" className="on" aria-pressed="true" tabIndex={-1}>
          {lang}
        </button>
        <button type="button" className="tk-cp" aria-label={label} onClick={() => void copied(t, copy)}>
          <Icon name="copy" size={13} />
          {t(tm.copy)}
        </button>
      </div>
      <pre>{children}</pre>
    </div>
  )
}

async function copied(t: ReturnType<typeof useT>, value: string): Promise<boolean> {
  const ok = await copyText(value)
  toast({ kind: ok ? 'success' : 'error', title: ok ? t(tm.copiedToast) : t(tm.copyFailed) })
  return ok
}

/** A read-only value with Copy (the site id, the identify secret). */
function CopyValue({ label, value, copyLabel, primary }: { label: string; value: string; copyLabel: string; primary?: boolean }) {
  const t = useT()
  const [done, setDone] = useState(false)
  return (
    <div className="ih-inl">
      <input className="in mono text-[12.5px]" aria-label={label} value={value} readOnly onFocus={(e) => e.currentTarget.select()} />
      <Button size="sm" variant={primary ? 'primary' : undefined} aria-label={copyLabel} icon={done ? 'check' : 'copy'} onClick={async () => setDone(await copied(t, value))}>
        {done ? t(tm.copied) : t(tm.copy)}
      </Button>
    </div>
  )
}

/** The identify secret, shown once, with how to sign an address. */
function SecretOnce({ secret, title }: { secret: string; title: string }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-2">
      <span className="lbl">{title}</span>
      <CopyValue label={t(tm.secret)} value={secret} copyLabel={t(tm.copySecret)} primary />
      <div className="note w" role="note">
        <Icon name="lock" size={16} className="mt-px flex-none text-warn-bar" />
        <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
          <b className="font-semibold">{t(tm.shownOnce)}</b>
          <span>{t(tm.secretBody)}</span>
        </div>
      </div>
      <span className="k text-ink-2">
        <T msg={tm.signHow} values={{ formula: code('hex(HMAC-SHA256(secret, lowercase(email)))'), call: code("reputecast('identify', {email, signature})") }} />
      </span>
    </div>
  )
}

/** Step 2: the snippet's three tabs, and the identify secret (once) or where it went. */
function Snippet({ site, secret, fresh, trail, onClose, onNext }: { site: Site; secret?: string; fresh: boolean; trail: ReactNode; onClose: () => void; onNext: () => void }) {
  const t = useT()
  const [tab, setTab] = useState<Tab>(site.store?.platform === 'shopify' ? 'pixel' : site.store?.platform === 'woocommerce' ? 'wordpress' : 'script')
  const [kept, setKept] = useState(false)
  const s = site.snippet
  const eyebrow = fresh
    ? t(tm.eyebrowAdded, { name: site.name, when: clock(site.createdAt) })
    : site.createdBy
      ? t(tm.eyebrowAddedWhenBy, { name: site.name, when: dayMonthYear(site.createdAt), by: site.createdBy })
      : t(tm.eyebrowAdded, { name: site.name, when: dayMonthYear(site.createdAt) })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(tm.snippetTitle, { origin: site.origin })}
      eyebrow={<span className="k">{eyebrow}</span>}
      className="max-w-[720px]"
      hideClose
      dismissible={!secret}
      footer={
        secret ? (
          <>
            <label className="mr-auto flex items-center gap-2 text-[13px]">
              <input type="checkbox" checked={kept} onChange={(e) => setKept(e.target.checked)} />
              {t(tm.secretCopied)}
            </label>
            <Button variant="primary" onClick={onNext} disabled={!kept}>
              {t(tm.continue)}
            </Button>
          </>
        ) : (
          <>
            <Button onClick={onClose}>{t(tm.close)}</Button>
            <Button variant="primary" onClick={onNext}>
              {t(tm.checkEvents)}
            </Button>
          </>
        )
      }
    >
      <div className="flex flex-col gap-3.5">
        {trail}
        <PanelTabs
          value={tab}
          onChange={setTab}
          label={t(tm.snippetTabs)}
          items={[
            { value: 'script', label: t(tm.tabScript) },
            { value: 'pixel', label: t(tm.tabPixel) },
            { value: 'wordpress', label: t(tm.tabWordPress) },
          ]}
        >
          <TabPanel value="script" className="flex flex-col gap-2.5 pt-3.5">
            <span className="text-[13px]"><T msg={tm.scriptHelp} values={{ origin: site.origin, head: code('</head>') }} /></span>
            <CodeBox lang={t(tm.html)} copy={s.script} label={t(tm.copyScript)}>
              {s.script}
            </CodeBox>
            <span className="k text-ink-2">
              <T msg={tm.storeCalls} values={{ product: code("reputecast('product', {id})"), cart: code("reputecast('cart', {id, quantity})") }} />
            </span>
          </TabPanel>
          <TabPanel value="pixel" className="flex flex-col gap-2.5 pt-3.5">
            <span className="text-[13px]">{site.waitConsent ? t(tm.pixelHelpWait) : t(tm.pixelHelp)}</span>
            <CodeBox lang={t(tm.javascript)} copy={s.shopifyPixel} label={t(tm.copyPixel)}>
              {s.shopifyPixel}
            </CodeBox>
          </TabPanel>
          <TabPanel value="wordpress" className="flex flex-col gap-2.5 pt-3.5">
            <b className="text-[13px] font-semibold">{t(tm.wpTitle)}</b>
            <ol className="ih-ol">
              <li>{t(tm.wp1)}</li>
              <li>{t(tm.wp2)}</li>
              <li>{t(tm.wp3)}</li>
              <li>{t(tm.wp4)}</li>
            </ol>
            <label className="lbl">
              {t(tm.siteId)}
              <CopyValue label={t(tm.siteId)} value={site.id} copyLabel={t(tm.copySiteId)} />
            </label>
          </TabPanel>
        </PanelTabs>
        {secret ? (
          <SecretOnce secret={secret} title={t(tm.secretOnce)} />
        ) : (
          <div className="flex flex-col gap-1">
            <span className="lbl">{t(tm.secret)}</span>
            <span className="k text-ink-2">{t(tm.secretExisting, { name: site.name })}</span>
          </div>
        )}
      </div>
    </Dialog>
  )
}

/** Step 3: listening until the first events arrive, then the first events. */
function FirstEvent({ site, trail, onClose }: { site: Site; trail: ReactNode; onClose: () => void }) {
  const t = useT()
  const now = useNow(15_000)
  const [since] = useState(() => Date.now())
  const q = useRecentEvents(site.id, true)
  const events = q.data?.items ?? []
  const first = events[events.length - 1]
  const minutes = Math.max(1, Math.round((now.getTime() - since) / 60_000))
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(tm.checkEvents)}
      eyebrow={<span className="k">{first ? t(tm.eyebrowFirst, { name: site.name, time: clock(first.at) }) : t(tm.eyebrowListening, { name: site.name })}</span>}
      className="max-w-[720px]"
      hideClose
      footer={
        events.length ? (
          <Button variant="primary" onClick={onClose}>
            {t(tm.done)}
          </Button>
        ) : (
          <>
            <Button onClick={onClose}>{t(tm.close)}</Button>
            <Button variant="primary" disabled>
              {t(tm.done)}
            </Button>
          </>
        )
      }
    >
      <div className="flex flex-col gap-3.5">
        {trail}
        {events.length === 0 ? (
          <div className="card tk-wait border-dashed">
            <span className="tk-spin" aria-hidden="true" />
            <div className="flex min-w-0 flex-col gap-1">
              <b className="font-semibold" role="status">
                {t(tm.waitingFirst)}
              </b>
              <span className="k">{t(tm.listeningFor, { origin: site.origin, minutes })}</span>
              <span className="text-[13px]">{site.waitConsent ? t(tm.listenHelpWait, { origin: site.origin }) : t(tm.listenHelp, { origin: site.origin })}</span>
            </div>
          </div>
        ) : (
          <>
            <div className="note g" role="status">
              <Icon name="check" size={16} className="mt-px flex-none text-ok-fg" />
              <div className="min-w-0 flex-[1_1_320px]">
                {site.store ? t(tm.arrivingStore, { origin: site.origin, store: site.store.name }) : t(tm.arriving, { origin: site.origin })}
              </div>
            </div>
            <div className="card overflow-hidden">
              <EventsList site={site} events={events.slice(0, 6)} />
            </div>
            <span className="k">{t(tm.visitorsNote)}</span>
          </>
        )}
      </div>
    </Dialog>
  )
}

/** Rotate the identify secret? Then the new one, shown once (TrackingStates › Dialogs). */
export function RotateSecretDialog({ site, onClose }: { site: Site; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const user = useMe().data
  const refresh = useRefreshSites()
  const m = $api.useMutation('post', secretPath)
  const [secret, setSecret] = useState<{ value: string; at: Date }>()
  const [kept, setKept] = useState(false)
  if (secret) {
    return (
      <Dialog
        open
        onOpenChange={() => undefined}
        dismissible={false}
        hideClose
        title={t(tm.rotatedTitle, { name: site.name })}
        eyebrow={<span className="k">{t(tm.rotatedEyebrow, { time: clock(secret.at), name: user?.user.name ?? '' })}</span>}
        className="max-w-[640px]"
        footer={
          <>
            <label className="mr-auto flex items-center gap-2 text-[13px]">
              <input type="checkbox" checked={kept} onChange={(e) => setKept(e.target.checked)} />
              {t(tm.secretCopied)}
            </label>
            <Button variant="primary" onClick={onClose} disabled={!kept}>
              {t(tm.done)}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3">
          <span className="text-[13px]">{t(tm.rotatedBody)}</span>
          <SecretOnce secret={secret.value} title={t(tm.secretOnce)} />
        </div>
      </Dialog>
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(tm.rotateTitle)}
      eyebrow={<span className="k">{t(tm.rotateEyebrow, { name: site.name, origin: site.origin })}</span>}
      description={t(tm.rotateBody)}
      className="max-w-[560px]"
      footer={
        <>
          <span className="k mr-auto">{t(tm.audited)}</span>
          <Button onClick={onClose}>{t(tm.cancel)}</Button>
          <Button
            variant="primary"
            loading={m.isPending}
            onClick={() =>
              m.mutate(
                { params: { path: { workspace: slug, site: site.id } } },
                {
                  onSuccess: (out) => {
                    void refresh()
                    setSecret({ value: out.identifySecret, at: new Date() })
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(tm.rotateButton)}
          </Button>
        </>
      }
    >
      <dl className="kv">
        <dt>{t(tm.stopsWorking)}</dt>
        <dd>{t(tm.stopsWorkingValue)}</dd>
        <dt>{t(tm.keepsWorking)}</dt>
        <dd>{t(tm.keepsWorkingValue)}</dd>
      </dl>
    </Dialog>
  )
}

/** Delete {site}? with what stops, what stays and the flows it feeds. */
export function DeleteSiteDialog({ site, onClose }: { site: Site; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSites()
  const m = $api.useMutation('delete', sitePath)
  const storePage = $api.useQuery('get', `${storesBase}/{id}`, { params: { path: { workspace: slug, id: site.store?.id ?? '' } } }, { enabled: !!site.store })
  const flows = site.store ? (storePage.data?.flows ?? []).filter((f) => trackerKinds.has(f.kind) && f.status !== 'draft').map((f) => f.name) : []
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(tm.deleteTitle, { name: site.name })}
      eyebrow={<span className="k">{t(tm.eyebrowPath)}</span>}
      description={
        site.store
          ? t(tm.deleteLineStore, { origin: site.origin, store: site.store.name, platform: platformName(site.store.platform), events: site.eventsToday })
          : t(tm.deleteLine, { origin: site.origin, events: site.eventsToday })
      }
      className="max-w-[600px]"
      footer={
        <>
          <span className="k mr-auto">{t(tm.audited)}</span>
          <Button onClick={onClose}>{t(tm.cancel)}</Button>
          <Button
            variant="danger-primary"
            loading={m.isPending}
            onClick={() =>
              m.mutate(
                { params: { path: { workspace: slug, site: site.id } } },
                {
                  onSuccess: () => {
                    void refresh()
                    toast({ kind: 'success', title: t(tm.deleted, { name: site.name }) })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(tm.deleteButton)}
          </Button>
        </>
      }
    >
      <dl className="kv">
        <dt>{t(tm.deleteScript)}</dt>
        <dd>{t(tm.deleteScriptValue, { origin: site.origin })}</dd>
        <dt>{t(tm.deleteEvents)}</dt>
        <dd>{t(tm.deleteEventsValue)}</dd>
        <dt>{t(tm.deleteFlows)}</dt>
        <dd>{flows.length ? <T msg={tm.deleteFlowsValue} values={{ flows: formatList(flows) }} /> : t(tm.deleteFlowsNone)}</dd>
      </dl>
      <div className="note w" role="note">
        <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
        <div className="min-w-0 flex-[1_1_320px]">{t(tm.deleteNote)}</div>
      </div>
    </Dialog>
  )
}

