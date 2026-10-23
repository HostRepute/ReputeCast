import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading } from '@/components/ui/Feedback'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { andList, clock, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { channelIcon } from '@/features/channels/ChannelSwitch'
import { browserName, throwIfForbidden } from './common'
import { m } from './composer.messages'
import { GuardNotes } from './Guards'
import { PreviewColumns, type PushPreview } from './Previews'
import { cutTitle, EXPIRIES, platform } from './cuts'
import './push.css'
import './composer.css'

type Campaign = Schemas['PushCampaign']
type Draft = Omit<Schemas['PushCampaignUpdate'], 'version'>
type CampaignKey = ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}', { params: { path: { workspace: string; campaignId: string } } }]

const statusWord: Partial<Record<Campaign['status'], MessageDescriptor>> = {
  approval: m.statusApproval,
  approved: m.statusApproved,
  scheduled: m.statusScheduled,
  sending: m.statusSending,
  paused: m.statusPaused,
  sent: m.statusSent,
  cancelled: m.statusCancelled,
  failed: m.statusFailed,
}

const expiryWord: Record<number, MessageDescriptor> = { 60: m.expiry60, 240: m.expiry240, 720: m.expiry720, 1440: m.expiry1440, 40320: m.expiry40320 }

function toDraft(c: Campaign): Draft {
  return {
    name: c.name,
    title: c.title,
    message: c.message,
    link: c.link,
    buttons: c.buttons.map((b) => ({ ...b })),
    imageMediaId: c.image?.mediaId ?? '',
    ttlMinutes: c.ttlMinutes,
    tag: c.tag,
    replaceTag: c.replaceTag,
    siteIds: c.sites.filter((s) => s.selected).map((s) => s.id),
    lists: c.lists.map((l) => l.id),
    segments: c.segments.map((s) => s.id),
  }
}

/** "news.acme.io/p/x" for "https://news.acme.io/p/x". */
function bare(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** The push composer: full screen, outside the app shell (board PushComposer). */
export function PushComposerPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  if (q.data) return <Composer key={q.data.id} data={q.data} />
  return (
    <div className="pc">
      <main className="flex flex-1 items-start justify-center p-10" id="main">
        {q.isError ? (
          <ErrorPanel
            className="max-w-[560px]"
            title={t(m.errorTitle)}
            reference={problemReference(q.error)}
            actions={
              <>
                <Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>
                <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                  {t(m.back)}
                </ButtonLink>
              </>
            }
          >
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : (
          <Loading label={t(m.loading)}>
            <span className="k">{t(m.loading)}</span>
          </Loading>
        )}
      </main>
    </div>
  )
}

function Composer({ data }: { data: Campaign }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const key: CampaignKey = ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: data.id } } }]
  const editable = data.access.canEdit && (data.status === 'draft' || data.status === 'approval' || data.status === 'approved')
  const [draft, setDraft] = useState<Draft>(() => toDraft(data))
  const [saved, setSaved] = useState(() => JSON.stringify(toDraft(data)))
  const [saving, setSaving] = useState<'idle' | 'saving' | 'error'>('idle')
  const [saveError, setSaveError] = useState<string>()
  const [image, setImage] = useState<Schemas['PushImage'] | undefined>(data.image)
  const [audienceOpen, setAudienceOpen] = useState(false)
  const [show, setShow] = useState<'all' | 'desktop' | 'phones'>('all')
  const [keepTitle, setKeepTitle] = useState<string>()
  const version = useRef(data.version)
  useDocumentTitle(`${t(m.docTitle, { name: draft.name || data.name })} · ${me.workspace.name} · ReputeCast`)
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}')
  const set = <K extends keyof Draft>(k: K, v: Draft[K]) => setDraft((d) => ({ ...d, [k]: v }))
  const current = JSON.stringify(draft)

  // Autosave: a moment after the last change.
  useEffect(() => {
    if (!editable || current === saved) return
    const id = window.setTimeout(() => {
      setSaving('saving')
      const body = JSON.parse(current) as Draft
      update.mutate(
        { params: { path: { workspace: slug, campaignId: data.id } }, body: { ...body, imageMediaId: body.imageMediaId || undefined, version: version.current } },
        {
          onSuccess: (next) => {
            version.current = next.version
            setSaved(current)
            qc.setQueryData(key, next)
            setImage(next.image)
            setSaving('idle')
          },
          onError: (e) => {
            if (e instanceof ApiError && e.status === 409) {
              toast({ kind: 'info', title: t(m.conflict) })
              void qc.invalidateQueries({ queryKey: key })
              return
            }
            setSaving('error')
            setSaveError(errorMessage(e))
          },
        },
      )
    }, 900)
    return () => window.clearTimeout(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current, saved, editable])

  const host = data.sites.find((s) => draft.siteIds.includes(s.id))?.host ?? data.host
  const n: PushPreview = useMemo(
    () => ({ title: draft.title, message: draft.message, host, iconUrl: data.iconUrl || undefined, imageUrl: image?.url, buttons: draft.buttons.map((b) => b.label) }),
    [draft.title, draft.message, draft.buttons, host, data.iconUrl, image?.url],
  )
  const status =
    data.status === 'draft'
      ? saving === 'saving'
        ? t(m.saving)
        : data.autosavedAt
          ? t(m.draftSaved, { time: clock(data.autosavedAt) })
          : t(m.draftNew)
      : t(statusWord[data.status] ?? m.draftNew)
  const selectedSites = data.sites.filter((s) => draft.siteIds.includes(s.id))
  const sent = data.status === 'sent' || data.status === 'sending' || data.status === 'paused' || data.status === 'cancelled'
  return (
    <div className="pc">
      <a href="#main" className="skip">
        {t(m.skip)}
      </a>
      <header className="pc-top">
        <ButtonLink iconOnly to="/w/$workspace/campaigns" params={{ workspace: slug }} aria-label={t(m.back)}>
          <Icon name="chevronLeft" size={16} />
        </ButtonLink>
        <div className="flex min-w-0 flex-col">
          <label className="k text-[11px]" htmlFor="push-name">
            {t(m.kind)}
          </label>
          <input id="push-name" className="nm" value={draft.name} readOnly={!editable} onChange={(e) => set('name', e.target.value)} maxLength={200} />
        </div>
        <Badge tone={saving === 'error' ? 'bad' : 'neutral'} title={saving === 'error' ? saveError : undefined}>
          {saving === 'error' ? t(m.saveFailed) : status}
        </Badge>
        {!data.access.canEdit ? <Badge tone="neutral">{t(m.readOnly)}</Badge> : null}
        <div className="ml-2 flex flex-wrap items-center gap-2">
          <span className="chip h-7 gap-1.5">
            {channelIcon.push}
            {t(m.push)}
          </span>
          <button type="button" className="btn" onClick={() => setAudienceOpen(true)} disabled={!editable}>
            <Icon name="audience" size={14} />
            {selectedSites.length ? t(m.audienceBtn, { sites: andList(selectedSites.map((s) => s.host)), count: data.audience.final }) : t(m.audienceNone)}
          </button>
        </div>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          {sent ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/push-report" params={{ workspace: slug, campaignId: data.id }}>
              {t(m.report)}
            </ButtonLink>
          ) : (
            <>
              <SendTestButton data={data} disabled={current !== saved} />
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/push-review" params={{ workspace: slug, campaignId: data.id }}>
                {data.access.canSend ? t(m.review) : t(m.reviewApproval)}
              </ButtonLink>
            </>
          )}
        </div>
      </header>
      {data.pushOn ? null : (
        <div className="note w m-0 rounded-none border-x-0 border-t-0" role="note">
          <span>{t(m.pushOff)}</span>
          <Link to="/w/$workspace/providers/push" params={{ workspace: slug }} className="px-lnk">
            {t(m.openSetup)}
          </Link>
        </div>
      )}
      <div className="pc-cols">
        <aside className="pc-panel" aria-label={t(m.panelLabel)}>
          <Fields data={data} draft={draft} set={set} editable={editable} image={image} setImage={setImage} keepTitle={keepTitle} setKeepTitle={setKeepTitle} host={host} />
        </aside>
        <section className="pc-area" id="main" aria-label={t(m.previews)}>
          <div className="flex flex-wrap items-center gap-2.5">
            <span className="font-bold">{t(m.previews)}</span>
            <span className="k">{t(m.previewsHint)}</span>
            <Segmented
              className="ml-auto"
              label={t(m.platformsLabel)}
              value={show}
              onChange={setShow}
              items={[
                { value: 'all', label: t(m.all, { n: 5 }) },
                { value: 'desktop', label: t(m.desktop) },
                { value: 'phones', label: t(m.phones) },
              ]}
            />
          </div>
          <PreviewColumns n={n} show={show} maxMessage={data.limits.message} />
          <GuardNotes data={data} editable={editable} />
        </section>
      </div>
      <AudienceDialog open={audienceOpen} onOpenChange={setAudienceOpen} data={data} draft={draft} set={set} />
    </div>
  )
}

function Fields({
  data,
  draft,
  set,
  editable,
  image,
  setImage,
  keepTitle,
  setKeepTitle,
  host,
}: {
  data: Campaign
  draft: Draft
  set: <K extends keyof Draft>(k: K, v: Draft[K]) => void
  editable: boolean
  image: Schemas['PushImage'] | undefined
  setImage: (i: Schemas['PushImage'] | undefined) => void
  keepTitle?: string
  setKeepTitle: (s: string | undefined) => void
  host: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const file = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const lim = data.limits
  const titleLen = [...draft.title].length
  const msgLen = [...draft.message].length
  const win = cutTitle(draft.title, platform('chrome_windows').titleChars)
  const othersCut = ['chrome_mac', 'safari_mac', 'chrome_android', 'safari_ios'].some((k) => cutTitle(draft.title, platform(k as never).titleChars).hidden)
  const showTitleNote = Boolean(win.hidden) && keepTitle !== draft.title
  const upload = async (files: FileList | null) => {
    const f = files?.[0]
    if (!f) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', f)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const body = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok) throw new Error(body.errors?.[0]?.message || body.detail || body.title || String(r.status))
      setImage({ mediaId: body.id, url: body.url, name: body.fileName, width: body.width, height: body.height, bytes: body.sizeBytes })
      set('imageMediaId', body.id)
    } catch (e) {
      toast({ kind: 'error', title: t(m.uploadFailed), detail: errorMessage(e) })
    } finally {
      setUploading(false)
      if (file.current) file.current.value = ''
    }
  }
  const moveToMessage = () => {
    const hidden = win.hidden.replace(/^[\s,;:–—-]+/, '')
    const lead = hidden.charAt(0).toUpperCase() + hidden.slice(1)
    const sentence = /[.!?…]$/.test(lead) ? lead : `${lead}.`
    set('title', win.shown.replace(/[\s,;:–—-]+$/, ''))
    set('message', `${sentence} ${draft.message}`.trim())
  }
  return (
    <>
      <div className="pc-fs">
        <div className="pc-fl">
          <span className="hd">
            <label htmlFor="push-title">{t(m.title)}</label>
            <span className={titleLen > lim.title ? 'pc-cnt w' : 'pc-cnt'}>{`${titleLen} / ${lim.title}`}</span>
          </span>
          <input
            id="push-title"
            className={titleLen > lim.title || showTitleNote ? 'in w' : 'in'}
            value={draft.title}
            readOnly={!editable}
            onChange={(e) => set('title', e.target.value)}
            maxLength={200}
          />
        </div>
        {showTitleNote ? (
          <div className="note w" role="note">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <circle cx="8" cy="8" r="6.2" />
              <path d="M8 4.8v3.6M8 10.9v.1" />
            </svg>
            <div className="px-note-body">
              <b>{t(m.titleCutTitle, { win: lim.titleWindows })}</b>
              <div>
                {t(othersCut ? m.titleCutTextOthers : m.titleCutText, { win: lim.titleWindows, hidden: win.hidden })}
                {editable ? (
                  <div className="px-note-acts">
                    <Button size="sm" onClick={moveToMessage}>
                      {t(m.moveToMessage, { hidden: win.hidden })}
                    </Button>
                    <Button size="sm" onClick={() => setKeepTitle(draft.title)}>
                      {t(m.keepIt)}
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        ) : (
          <span className="k">{t(m.titleHint, { max: lim.title, win: lim.titleWindows })}</span>
        )}
      </div>
      <div className="pc-fs">
        <div className="pc-fl">
          <span className="hd">
            <label htmlFor="push-message">{t(m.message)}</label>
            <span className={msgLen > lim.message ? 'pc-cnt w' : 'pc-cnt'}>{`${msgLen} / ${lim.message}`}</span>
          </span>
          <textarea
            id="push-message"
            className={msgLen > lim.message ? 'in w' : 'in'}
            rows={3}
            value={draft.message}
            readOnly={!editable}
            onChange={(e) => set('message', e.target.value)}
            maxLength={500}
          />
        </div>
      </div>
      <div className="pc-fs">
        <label className="pc-fl">
          {t(m.link)}
          <input className="in mono text-[12px]" value={draft.link} readOnly={!editable} placeholder={t(m.linkPlaceholder)} onChange={(e) => set('link', e.target.value)} maxLength={2000} />
        </label>
        <span className="k">
          <T msg={m.linkHint} values={{ campaign: data.utm, mono: (c) => <span className="mono">{c}</span> }} />
        </span>
      </div>
      <div className="pc-fs">
        <span className="t">{t(m.iconAndImage)}</span>
        <div className="flex items-center gap-2.5">
          <span className="pc-mark" aria-hidden="true">
            {data.iconUrl ? <img src={data.iconUrl} alt="" /> : (host[0] ?? 'A').toUpperCase()}
          </span>
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="text-[13px] font-semibold">{t(m.icon)}</span>
            <span className="k">{host ? t(m.iconHint, { host }) : t(m.iconHintNoSite)}</span>
          </span>
        </div>
        <div className="flex items-center gap-2.5">
          {image ? (
            <>
              <span className="pc-thumb">
                <img src={image.url} alt="" />
              </span>
              <span className="flex min-w-0 flex-1 flex-col gap-px">
                <span className="text-[13px] font-semibold">{image.name}</span>
                <span className="k">{t(m.imageLine, { w: image.width, h: image.height, size: t(m.sizeKb, { n: Math.max(1, Math.round(image.bytes / 1024)) }) })}</span>
                {editable ? (
                  <span className="mt-1 flex gap-1.5">
                    <Button size="sm" loading={uploading} busyLabel={t(m.uploading)} onClick={() => file.current?.click()}>
                      {t(m.replace)}
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => {
                        setImage(undefined)
                        set('imageMediaId', '')
                      }}
                    >
                      {t(m.remove)}
                    </Button>
                  </span>
                ) : null}
              </span>
            </>
          ) : (
            <span className="flex min-w-0 flex-1 flex-col gap-1.5">
              <span className="k">{t(m.imageNone)}</span>
              {editable ? (
                <span>
                  <Button size="sm" icon="image" loading={uploading} busyLabel={t(m.uploading)} onClick={() => file.current?.click()}>
                    {t(m.addImage)}
                  </Button>
                </span>
              ) : null}
            </span>
          )}
          <input ref={file} type="file" accept="image/png,image/jpeg,image/gif" className="vh" tabIndex={-1} aria-hidden="true" onChange={(e) => void upload(e.target.files)} />
        </div>
      </div>
      <div className="pc-fs">
        <span className="t">{t(m.buttons)}</span>
        {draft.buttons.map((b, i) => (
          <div key={i} className="pc-btnrow">
            <label className="lbl">
              {t(m.buttonN, { n: i + 1 })}
              <input
                className="in"
                value={b.label}
                readOnly={!editable}
                aria-label={t(m.buttonLabel, { n: i + 1 })}
                maxLength={40}
                onChange={(e) => set('buttons', draft.buttons.map((x, j) => (j === i ? { ...x, label: e.target.value } : x)))}
              />
            </label>
            <label className="lbl">
              {t(m.opens)}
              <input
                className="in mono text-[11.5px]"
                value={bare(b.url)}
                readOnly={!editable}
                aria-label={t(m.buttonLink, { n: i + 1 })}
                maxLength={2000}
                onChange={(e) => set('buttons', draft.buttons.map((x, j) => (j === i ? { ...x, url: e.target.value } : x)))}
              />
            </label>
            {editable ? (
              <Button
                iconOnly
                icon="close"
                aria-label={t(m.removeButton, { n: i + 1 })}
                onClick={() =>
                  set(
                    'buttons',
                    draft.buttons.filter((_, j) => j !== i),
                  )
                }
              />
            ) : (
              <span />
            )}
          </div>
        ))}
        {editable && draft.buttons.length < 2 ? (
          <span>
            <Button size="sm" icon="plus" onClick={() => set('buttons', [...draft.buttons, { label: '', url: '' }])}>
              {t(m.addButton)}
            </Button>
          </span>
        ) : null}
      </div>
      <div className="pc-fs">
        <span className="t">{t(m.delivery)}</span>
        <div className="grid grid-cols-2 gap-2">
          <label className="lbl">
            {t(m.expiresAfter)}
            <select className="in" value={draft.ttlMinutes} disabled={!editable} onChange={(e) => set('ttlMinutes', Number(e.target.value))}>
              {(EXPIRIES as readonly number[]).includes(draft.ttlMinutes) ? null : <option value={draft.ttlMinutes}>{t(m.expiryOther, { n: draft.ttlMinutes })}</option>}
              {EXPIRIES.map((v) => (
                <option key={v} value={v}>
                  {t(expiryWord[v]!)}
                </option>
              ))}
            </select>
          </label>
          <label className="lbl">
            {t(m.tag)}
            <input className="in mono" value={draft.tag} readOnly={!editable} maxLength={60} onChange={(e) => set('tag', e.target.value)} />
          </label>
        </div>
        <label className="flex items-start gap-2 text-[13px]">
          <input type="checkbox" className="mt-0.5" checked={draft.replaceTag} disabled={!editable} onChange={(e) => set('replaceTag', e.target.checked)} />
          <span>
            {t(m.replaceTag)} <span className="k">{t(m.replaceTagHint)}</span>
          </span>
        </label>
        <span className="k">{t(m.expiryNote)}</span>
      </div>
      <div className="pc-fs">
        <span className="t">{t(m.test)}</span>
        <div className="flex flex-wrap items-center gap-2">
          <SendTestButton data={data} size="sm" />
          {data.test ? <TestLine test={data.test} /> : null}
        </div>
      </div>
    </>
  )
}

const testWord: Record<Schemas['PushTestResult']['status'], { tone: 'ok' | 'info' | 'neutral' | 'bad'; msg: MessageDescriptor }> = {
  shown: { tone: 'ok', msg: m.testShown },
  clicked: { tone: 'ok', msg: m.testClicked },
  sent: { tone: 'info', msg: m.testSent },
  queued: { tone: 'neutral', msg: m.testQueued },
  failed: { tone: 'bad', msg: m.testFailed },
}

function TestLine({ test }: { test: Schemas['PushTestResult'] }) {
  const t = useT()
  const w = testWord[test.status]
  return (
    <span className="k inline-flex flex-wrap items-center gap-1.5">
      {t(m.lastTest, { time: clock(test.at), browser: t(browserName(test.browser)) })} ·{' '}
      <Badge tone={w.tone} title={test.detail}>
        {t(w.msg)}
      </Badge>
    </span>
  )
}

/** Converts the VAPID public key (base64url) for PushManager.subscribe. */
function keyBytes(b64: string): Uint8Array<ArrayBuffer> {
  const pad = '='.repeat((4 - (b64.length % 4)) % 4)
  const raw = atob((b64 + pad).replace(/-/g, '+').replace(/_/g, '/'))
  const out = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

function b64url(buf: ArrayBuffer | null): string {
  if (!buf) return ''
  let s = ''
  for (const b of new Uint8Array(buf)) s += String.fromCharCode(b)
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/**
 * "Send to my browser": subscribes this browser with the workspace's keys
 * (through ReputeCast's own test worker) and sends the notification to it.
 */
function SendTestButton({ data, disabled, size }: { data: Campaign; disabled?: boolean; size?: 'sm' }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [busy, setBusy] = useState(false)
  const supported = typeof window !== 'undefined' && 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
  const run = async () => {
    if (!supported) {
      toast({ kind: 'error', title: t(m.testFailedToast), detail: t(m.testUnsupported) })
      return
    }
    setBusy(true)
    try {
      const permission = await Notification.requestPermission()
      if (permission !== 'granted') throw new Error(t(m.testBlocked))
      const setup = await http.GET('/api/v1/workspaces/{workspace}/push/setup', { params: { path: { workspace: slug } } })
      const publicKey = setup.data?.keys?.publicKey
      if (!publicKey) throw new Error(t(m.pushOff))
      const reg = await navigator.serviceWorker.register('/push/v1/test-sw.js', { scope: '/push/v1/' })
      let sub = await reg.pushManager.getSubscription()
      if (sub && b64url(sub.options.applicationServerKey) !== publicKey) {
        await sub.unsubscribe()
        sub = null
      }
      sub ??= await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: keyBytes(publicKey) })
      await http.POST('/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/test', {
        params: { path: { workspace: slug, campaignId: data.id } },
        body: { endpoint: sub.endpoint, p256dh: b64url(sub.getKey('p256dh')), auth: b64url(sub.getKey('auth')) },
      })
      toast({ kind: 'success', title: t(m.testSentToast) })
      window.setTimeout(() => void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}'] }), 2500)
    } catch (e) {
      toast({ kind: 'error', title: t(m.testFailedToast), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Button size={size} icon={size ? 'bell' : 'desktop'} loading={busy} busyLabel={t(m.sending)} onClick={() => void run()} disabled={disabled}>
      {t(m.sendTest)}
    </Button>
  )
}

/** Who gets it: the sites, narrowed to subscribers linked to contacts in lists or segments. */
function AudienceDialog({
  open,
  onOpenChange,
  data,
  draft,
  set,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  data: Campaign
  draft: Draft
  set: <K extends keyof Draft>(k: K, v: Draft[K]) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } }, { enabled: open })
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } }, { enabled: open })
  const toggle = (k: 'siteIds' | 'lists' | 'segments', id: string, on: boolean) => {
    const cur = draft[k]
    set(k, on ? [...cur, id] : cur.filter((x) => x !== id))
  }
  const a = data.audience
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t(m.audienceTitle)}
      description={t(m.audienceIntro)}
      footer={
        <Button variant="primary" onClick={() => onOpenChange(false)}>
          {t(m.done)}
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        <fieldset className="flex flex-col gap-2">
          <legend className="mb-1 text-[13px] font-semibold">{t(m.sitesLabel)}</legend>
          {data.sites.map((s) => (
            <label key={s.id} className="flex items-center gap-2 text-[13px]">
              <input type="checkbox" checked={draft.siteIds.includes(s.id)} onChange={(e) => toggle('siteIds', s.id, e.target.checked)} />
              <span>{t(m.siteRow, { host: s.host, count: s.subscribers })}</span>
            </label>
          ))}
          {draft.siteIds.length ? null : <span className="err">{t(m.sitesNone)}</span>}
        </fieldset>
        <fieldset className="flex flex-col gap-2">
          <legend className="mb-1 text-[13px] font-semibold">{t(m.narrow)}</legend>
          <span className="k">{t(m.narrowHint)}</span>
          <span className="k mt-1 font-semibold text-ink-2">{t(m.listsLabel)}</span>
          <div className="flex flex-wrap gap-x-4 gap-y-1.5">
            {(lists.data?.items ?? []).map((l) => (
              <label key={l.id} className="flex items-center gap-2 text-[13px]">
                <input type="checkbox" checked={draft.lists.includes(l.id)} onChange={(e) => toggle('lists', l.id, e.target.checked)} />
                {l.name}
              </label>
            ))}
          </div>
          <span className="k mt-1 font-semibold text-ink-2">{t(m.segmentsLabel)}</span>
          <div className="flex flex-wrap gap-x-4 gap-y-1.5">
            {(segments.data?.items ?? []).map((s) => (
              <label key={s.id} className="flex items-center gap-2 text-[13px]">
                <input type="checkbox" checked={draft.segments.includes(s.id)} onChange={(e) => toggle('segments', s.id, e.target.checked)} />
                {s.name}
              </label>
            ))}
          </div>
        </fieldset>
        <div className="note n" role="status">
          <div>
            {t(m.audienceSummary, { total: a.total, linked: count(a.linked), anonymous: count(a.anonymous) })}
            {a.narrowed ? ` · ${t(m.audienceNarrowed, { final: a.final })}` : null}
          </div>
        </div>
      </div>
    </Dialog>
  )
}

