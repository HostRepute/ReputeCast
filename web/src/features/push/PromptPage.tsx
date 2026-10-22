import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { FormsTabs, useFormsOverview } from '@/features/forms/common'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, monthDayTimeSeconds, pct } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { browserName, styleName, throwIfForbidden, type PushPromptConfig } from './common'
import { m } from './prompt.messages'
import { BrowserPrompts, IosSheet, Mark, PromptOver, SiteFrame, type SiteLook } from './PromptPreviews'
import './composer.css'
import './prompt.css'

type Prompt = Schemas['PushPrompt']
type SiteOption = Schemas['PushPromptSiteOption']
type View = 'both' | 'ios' | 'notnow' | 'blocked'

const PATH_RE = /^\/[\w\-./~%]*$/

function siteKey(o: SiteOption): string {
  return o.id ?? `origin:${o.origin}`
}

/** The push prompt designer: a form of the "Push prompt" type (board PushPrompt). */
export function PushPromptPage({ formId }: { formId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/prompts/{formId}', { params: { path: { workspace: slug, formId } } })
  throwIfForbidden(q.error)
  if (q.data) return <Designer key={`${q.data.formId}:${q.data.updatedAt}`} data={q.data} />
  return (
    <Page title={t(m.screenTitle)} filters={<FormsTabs />}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.loadFailedText)}
        </ErrorPanel>
      ) : (
        <Loading label={t(m.loading)} className="flex flex-col gap-3">
          <Skeleton w={260} h={16} />
          <Skeleton w="100%" h={300} />
        </Loading>
      )}
    </Page>
  )
}

function Designer({ data }: { data: Prompt }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const overview = useFormsOverview()
  const canWebsite = useCan('website:view')
  const canManage = useCan('providers:manage')
  const website = $api.useQuery('get', '/api/v1/workspaces/{workspace}/website', { params: { path: { workspace: slug } } }, { enabled: canWebsite })
  const posts = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/website/posts',
    { params: { path: { workspace: slug }, query: { status: 'published', pageSize: 1 } } },
    { enabled: canWebsite },
  )
  const [cfg, setCfg] = useState<PushPromptConfig>(() => structuredClone(data.draft))
  const [siteKeyState, setSiteKey] = useState<string>(() => (data.site ? siteKey(data.site) : data.sites[0] ? siteKey(data.sites[0]) : ''))
  const [view, setView] = useState<View>('both')
  const [deleting, setDeleting] = useState(false)
  const editable = data.canEdit
  const site = data.sites.find((o) => siteKey(o) === siteKeyState) ?? data.site
  const dirty = JSON.stringify(cfg) !== JSON.stringify(data.draft) || (site?.id ?? '') !== (data.site?.id ?? '')
  const key = ['get', '/api/v1/workspaces/{workspace}/push/prompts/{formId}', { params: { path: { workspace: slug, formId: data.formId } } }] as const
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push/prompts/{formId}')
  const publish = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/prompts/{formId}/publish')
  const discard = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/prompts/{formId}/discard')
  const status = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push/prompts/{formId}/status')
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/push/prompts/{formId}')
  const addSite = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/sites')
  const set = <K extends keyof PushPromptConfig>(k: K, v: PushPromptConfig[K]) => setCfg((c) => ({ ...c, [k]: v }))
  const setWhen = (patch: Partial<PushPromptConfig['when']>) => setCfg((c) => ({ ...c, when: { ...c.when, ...patch } }))
  const path = { workspace: slug, formId: data.formId }
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: key })
    void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && (x.queryKey[1].includes('/push/setup') || x.queryKey[1].includes('/forms')) })
  }
  const saveDraft = async (siteId = site?.id) => {
    await save.mutateAsync({ params: { path }, body: { config: cfg, siteId } })
  }
  const onPublish = async () => {
    try {
      if (dirty) await saveDraft()
      await publish.mutateAsync({ params: { path } })
      toast({ kind: 'success', title: t(m.published, { host: site?.host ?? '' }) })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
    }
  }
  const onSaveDraft = async () => {
    try {
      await saveDraft()
      toast({ kind: 'success', title: t(m.savedDraft) })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
    }
  }
  const onDiscard = async () => {
    try {
      if (data.changed) await discard.mutateAsync({ params: { path } })
      setCfg(structuredClone(data.draft))
      setSiteKey(data.site ? siteKey(data.site) : (data.sites[0] && siteKey(data.sites[0])) || '')
      toast({ kind: 'info', title: t(m.discarded) })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const onAddSite = async () => {
    if (!site) return
    try {
      const added = await addSite.mutateAsync({ params: { path: { workspace: slug } }, body: { address: site.origin } })
      await save.mutateAsync({ params: { path }, body: { config: cfg, siteId: added.id } })
      toast({ kind: 'success', title: t(m.siteAdded, { host: added.host }) })
      refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const host = site?.host ?? ''
  const branding = overview.data?.branding
  const pubTitle = website.data?.title
  const post = posts.data?.posts[0]
  const look: SiteLook = {
    host,
    name: site?.kind === 'publication' && pubTitle ? pubTitle : (branding?.publisherName ?? host),
    initial: (branding?.initial || branding?.publisherName?.[0] || host[0] || 'A').toUpperCase(),
    logoUrl: branding?.logo?.url,
    iconUrl: data.iconUrl,
    accent: branding?.accent || '#4338CA',
    publication: site?.kind === 'publication',
    post:
      site?.kind === 'publication' && post
        ? {
            title: post.title,
            meta: t(m.byline, { date: dayMonthYear(post.publishedAt ?? post.updatedAt), name: post.byline.name }),
            paragraphs: post.excerpt ? [post.excerpt] : [],
          }
        : undefined,
  }
  const postPath = site?.kind === 'publication' && post ? new URL(post.url, 'https://x').pathname : '/'
  const headerStatus =
    data.status === 'live' ? (
      <Badge tone="ok" dot>
        {host ? t(m.statusLive, { host }) : t(m.statusLiveNoSite)}
      </Badge>
    ) : data.status === 'paused' ? (
      <Badge tone="warn">{t(m.statusPaused, { host })}</Badge>
    ) : (
      <Badge tone="neutral">{t(m.statusDraft)}</Badge>
    )
  const busy = save.isPending || publish.isPending
  // "After “Not now”, ask again in [14] days": one sentence around its number field.
  const askAgain = (
    <T
      msg={m.askAgain}
      values={{
        text: (c) => <span className="flex-1">{c}</span>,
        n: <NumberInput value={cfg.askAgainDays} onValue={(n) => set('askAgainDays', Math.max(1, Math.min(365, n)))} label={t(m.daysLabel)} />,
        unit: (c) => <span className="k">{c}</span>,
      }}
    />
  )
  return (
    <Page
      title={t(m.screenTitle)}
      documentTitle={t(m.docTitle, { name: data.name })}
      filters={<FormsTabs />}
      contentClassName="fb-main"
      actions={
        <>
          {headerStatus}
          {editable ? (
            <>
              <Button onClick={() => void onDiscard()} disabled={!dirty && !data.changed} loading={discard.isPending}>
                {t(m.discard)}
              </Button>
              <Button variant="primary" onClick={() => void onPublish()} loading={busy} busyLabel={t(m.publishing)} disabled={!site?.pushSite}>
                {t(m.saveAndPublish)}
              </Button>
            </>
          ) : null}
        </>
      }
    >
      <UnsavedChangesGuard dirty={dirty && editable} description={t(m.savedDraft)} onSave={() => saveDraft()} />
      <div className="pp-cols">
        <aside className="pp-left" aria-label={t(m.settingsLabel)}>
          <div className="pp-sec">
            <label className="t" htmlFor="pp-form">
              {t(m.form)}
            </label>
            <select
              id="pp-form"
              className="in w-full"
              value={data.formId}
              onChange={(e) => {
                const f = overview.data?.forms.find((x) => x.id === e.target.value)
                if (!f) return
                void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: f.kind === 'push' ? { prompt: f.id } : { form: f.id } })
              }}
            >
              {(overview.data?.forms ?? [{ id: data.formId, name: data.name, kind: 'push' as const }]).map((f) => (
                <option key={f.id} value={f.id}>
                  {f.kind === 'push' ? t(m.formOption, { name: f.name }) : f.name}
                </option>
              ))}
            </select>
          </div>
          <div className="pp-sec">
            <span className="t">{t(m.type)}</span>
            <div className="pp-type">
              <div className="seg" role="group" aria-label={t(m.type)}>
                {[m.typeEmbedded, m.typePopup, m.typeHosted].map((x) => (
                  <button key={x.id} type="button" aria-pressed="false" aria-disabled="true" title={t(m.typeLocked)}>
                    {t(x)}
                  </button>
                ))}
                <button type="button" aria-pressed="true">
                  {t(m.typePush)}
                </button>
              </div>
            </div>
            <span className="k text-ink-2">{t(m.typeHint)}</span>
          </div>
          <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
            <legend className="t float-left mb-[9px] w-full">{t(m.style)}</legend>
            <div className="pp-styles clear-both" role="radiogroup" aria-label={t(m.style)}>
              {(
                [
                  ['bar', m.styleBar, m.styleBarHint],
                  ['dialog', m.styleDialog, m.styleDialogHint],
                  ['bell', m.styleBell, m.styleBellHint],
                ] as const
              ).map(([v, name, hint]) => (
                <label key={v} className={cfg.style === v ? 'pp-style on' : 'pp-style'}>
                  <span className="pp-top">
                    <StyleIcon style={v} />
                    <input type="radio" name="pp-style" checked={cfg.style === v} onChange={() => set('style', v)} />
                  </span>
                  <b>{t(name)}</b>
                  <span className="k">{t(hint)}</span>
                </label>
              ))}
            </div>
          </fieldset>
          <TextSection cfg={cfg} set={set} editable={editable} look={look} site={site} canManage={canManage} onIcon={refresh} />
          <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
            <legend className="t float-left mb-[9px] w-full">{t(m.when)}</legend>
            {cfg.style === 'bell' ? (
              <>
                <span className="k clear-both text-ink-2">{t(m.bellWhen)}</span>
                <div className="lbl">
                  <label htmlFor="pp-corner">{t(m.corner)}</label>
                  <select id="pp-corner" className="in" value={cfg.corner || 'bottom_right'} onChange={(e) => set('corner', e.target.value as PushPromptConfig['corner'])}>
                    <option value="bottom_right">{t(m.cornerRight)}</option>
                    <option value="bottom_left">{t(m.cornerLeft)}</option>
                  </select>
                </div>
              </>
            ) : (
              <>
                <Rule
                  checked={cfg.when.seconds.on}
                  onCheck={(on) => setWhen({ seconds: { ...cfg.when.seconds, on } })}
                  msg={m.ruleSeconds}
                  value={cfg.when.seconds.n}
                  label={t(m.secondsLabel)}
                  onValue={(n) => setWhen({ seconds: { ...cfg.when.seconds, n } })}
                />
                <Rule
                  checked={cfg.when.scroll.on}
                  onCheck={(on) => setWhen({ scroll: { ...cfg.when.scroll, on } })}
                  msg={m.ruleScroll}
                  value={cfg.when.scroll.n}
                  label={t(m.percentLabel)}
                  onValue={(n) => setWhen({ scroll: { ...cfg.when.scroll, n: Math.min(100, n) } })}
                />
                <Rule
                  checked={cfg.when.posts.on}
                  onCheck={(on) => setWhen({ posts: { ...cfg.when.posts, on } })}
                  msg={m.rulePosts}
                  value={cfg.when.posts.n}
                  label={t(m.postsLabel)}
                  onValue={(n) => setWhen({ posts: { ...cfg.when.posts, n } })}
                />
                <label className="pp-rule">
                  <input type="checkbox" checked={cfg.when.button} onChange={(e) => setWhen({ button: e.target.checked })} />
                  <span className="min-w-0 flex-1">{t(m.ownButton)}</span>
                </label>
                {cfg.when.button ? (
                  <span className="k">
                    <T msg={m.ownButtonHint} values={{ mono: (c) => <span className="mono">{c}</span> }} />
                  </span>
                ) : null}
                {cfg.when.seconds.on || cfg.when.scroll.on || cfg.when.posts.on || cfg.when.button ? (
                  <span className="k text-ink-2">{t(m.whenHint)}</span>
                ) : (
                  <span className="err">{t(m.whenNone)}</span>
                )}
              </>
            )}
          </fieldset>
          <WhereSection data={data} cfg={cfg} set={set} editable={editable} siteKeyState={siteKeyState} setSiteKey={setSiteKey} />
          <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
            <legend className="t float-left mb-[9px] w-full">{t(m.frequency)}</legend>
            <label className="pp-rule clear-both">{askAgain}</label>
            <span className="k text-ink-2">{t(m.frequencyHint)}</span>
          </fieldset>
          <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
            <legend className="t float-left mb-[9px] w-full">{t(m.iphone)}</legend>
            <label className={cfg.ios === 'steps' ? 'pc-optc on clear-both' : 'pc-optc clear-both'}>
              <input type="radio" name="pp-ios" checked={cfg.ios === 'steps'} onChange={() => set('ios', 'steps')} />
              <span className="flex flex-col gap-0.5 font-semibold">
                {t(m.iosSteps)}
                <span className="k">{t(m.iosStepsHint)}</span>
              </span>
            </label>
            <label className={cfg.ios === 'hidden' ? 'pc-optc on' : 'pc-optc'}>
              <input type="radio" name="pp-ios" checked={cfg.ios === 'hidden'} onChange={() => set('ios', 'hidden')} />
              <span className="font-semibold">{t(m.iosHidden)}</span>
            </label>
            <span className="k text-ink-2">{t(m.iosHint)}</span>
          </fieldset>
          {editable ? (
            <div className="pp-sec">
              <span className="t">{t(m.manage)}</span>
              <div className="flex flex-wrap gap-2">
                {data.status !== 'draft' ? (
                  <Button
                    size="sm"
                    loading={status.isPending}
                    onClick={() =>
                      status.mutate(
                        { params: { path }, body: { status: data.status === 'paused' ? 'live' : 'paused' } },
                        {
                          onSuccess: () => {
                            toast({ kind: 'success', title: data.status === 'paused' ? t(m.resumed) : t(m.paused) })
                            refresh()
                          },
                          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                        },
                      )
                    }
                  >
                    {data.status === 'paused' ? t(m.resume) : t(m.pause)}
                  </Button>
                ) : null}
                <Button size="sm" variant="danger" onClick={() => setDeleting(true)}>
                  {t(m.deletePrompt)}
                </Button>
              </div>
            </div>
          ) : null}
        </aside>
        <section className="pp-area" aria-label={t(m.livePreview, { host })}>
          <div className="pp-bar">
            <span className="k">{t(m.livePreview, { host })}</span>
            <Segmented
              className="ml-auto"
              label={t(m.previewLabel)}
              value={view}
              onChange={setView}
              items={[
                { value: 'both', label: t(m.viewBoth) },
                { value: 'ios', label: t(m.viewIos) },
                { value: 'notnow', label: t(m.viewNotNow) },
                { value: 'blocked', label: t(m.viewBlocked) },
              ]}
            />
          </div>
          <div className="pp-body">
            {!data.pushOn ? (
              <div className="note w" role="note">
                <span>{t(m.pushOff)}</span>
              </div>
            ) : null}
            {site && !site.pushSite ? (
              <div className="note w" role="alert">
                <WarnIcon />
                <div className="px-note-body">
                  <b>{t(m.notPushTitle, { host })}</b>
                  <div>
                    {t(site.kind === 'publication' ? m.notPushText : m.notPushTextSnippet)}
                    {editable ? (
                      <div className="px-note-acts">
                        {canManage && site.kind === 'publication' && data.pushOn ? (
                          <Button size="sm" variant="primary" loading={addSite.isPending} onClick={() => void onAddSite()}>
                            {t(m.addInSetup, { host })}
                          </Button>
                        ) : (
                          <Link to="/w/$workspace/providers/push" params={{ workspace: slug }} className="btn sm pri">
                            {t(m.openSetup)}
                          </Link>
                        )}
                        <Button size="sm" loading={save.isPending} onClick={() => void onSaveDraft()}>
                          {t(m.saveDraft)}
                        </Button>
                      </div>
                    ) : null}
                  </div>
                </div>
              </div>
            ) : null}
            <Preview view={view} cfg={cfg} look={look} path={postPath} data={data} />
            {view === 'both' ? <BrowserPrompts look={look} stats={data.stats} /> : null}
            <Stats data={data} cfg={cfg} host={host} />
            <Consent data={data} />
          </div>
        </section>
      </div>
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        badge={t(m.deletePrompt)}
        title={t(m.deleteTitle, { name: data.name })}
        description={t(m.deleteText)}
        confirmLabel={t(m.deletePrompt)}
        onConfirm={async () => {
          await remove.mutateAsync({ params: { path } })
          toast({ kind: 'deleted', title: t(m.deleted, { name: data.name }) })
          refresh()
          void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: {} })
        }}
      />
    </Page>
  )
}

function WarnIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

function StyleIcon({ style }: { style: PushPromptConfig['style'] }) {
  return (
    <svg width="44" height="30" viewBox="0 0 44 30" aria-hidden="true">
      <rect x=".5" y=".5" width="43" height="29" rx="4" fill="#fff" stroke="#C9CED6" />
      {style === 'bar' ? (
        <>
          <rect x="5" y="20" width="34" height="6" rx="2" fill="#4338CA" />
          <rect x="6" y="5" width="20" height="3" rx="1.5" fill="#D4D8DF" />
          <rect x="6" y="11" width="28" height="3" rx="1.5" fill="#E3E6EB" />
        </>
      ) : null}
      {style === 'dialog' ? <rect x="11" y="8" width="22" height="14" rx="2.5" fill="#4338CA" /> : null}
      {style === 'bell' ? (
        <>
          <rect x="6" y="5" width="20" height="3" rx="1.5" fill="#D4D8DF" />
          <rect x="6" y="11" width="28" height="3" rx="1.5" fill="#E3E6EB" />
          <circle cx="35" cy="22" r="4" fill="#4338CA" />
        </>
      ) : null}
    </svg>
  )
}

function NumberInput({ value, onValue, label }: { value: number; onValue: (n: number) => void; label: string }) {
  const [text, setText] = useState(String(value))
  const [seen, setSeen] = useState(value)
  if (seen !== value) {
    setSeen(value)
    setText(String(value))
  }
  return (
    <input
      className="in pp-num"
      inputMode="numeric"
      value={text}
      aria-label={label}
      onChange={(e) => {
        const v = e.target.value.replace(/[^0-9]/g, '').slice(0, 4)
        setText(v)
        if (v) onValue(Number(v))
      }}
      onBlur={() => setText(String(value))}
    />
  )
}

/** A "When" rule: one sentence with its number field in it ("After [30] seconds on a page"). */
function Rule({
  checked,
  onCheck,
  msg,
  value,
  label,
  onValue,
}: {
  checked: boolean
  onCheck: (on: boolean) => void
  msg: MessageDescriptor
  value: number
  label: string
  onValue: (n: number) => void
}) {
  const t = useT()
  return (
    <div className="pp-rule clear-both">
      <input type="checkbox" checked={checked} onChange={(e) => onCheck(e.target.checked)} aria-label={t(msg, { n: value })} />
      <span className="min-w-0 flex-1">
        <T msg={msg} values={{ n: <NumberInput value={value} onValue={(n) => onValue(Math.max(1, n))} label={label} /> }} />
      </span>
    </div>
  )
}

function TextSection({
  cfg,
  set,
  editable,
  look,
  site,
  canManage,
  onIcon,
}: {
  cfg: PushPromptConfig
  set: <K extends keyof PushPromptConfig>(k: K, v: PushPromptConfig[K]) => void
  editable: boolean
  look: SiteLook
  site?: SiteOption
  canManage: boolean
  onIcon: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const file = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const tl = [...cfg.title].length
  const ml = [...cfg.message].length
  const changeIcon = async (files: FileList | null) => {
    const f = files?.[0]
    if (!f || !site?.id) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', f)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
      const media = (await r.json().catch(() => ({}))) as Schemas['MediaAsset'] & { detail?: string }
      if (!r.ok) throw new Error(media.detail || String(r.status))
      const setup = await http.GET('/api/v1/workspaces/{workspace}/push/setup', { params: { path: { workspace: slug } } })
      const s = setup.data?.sites.find((x) => x.id === site.id)
      if (!s) throw new Error(site.host)
      await http.PUT('/api/v1/workspaces/{workspace}/push/sites/{siteId}', {
        params: { path: { workspace: slug, siteId: site.id } },
        body: { label: s.label, workerMode: s.worker.mode, workerPath: s.worker.path, iconMediaId: media.id },
      })
      toast({ kind: 'success', title: t(m.iconChanged, { host: site.host }) })
      onIcon()
    } catch (e) {
      toast({ kind: 'error', title: t(m.iconFailed), detail: e instanceof ApiError ? e.message : errorMessage(e) })
    } finally {
      setUploading(false)
      if (file.current) file.current.value = ''
    }
  }
  return (
    <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
      <legend className="t float-left mb-[9px] w-full">{t(m.text)}</legend>
      <div className="pc-fl clear-both">
        <span className="hd">
          <label htmlFor="pp-title">{t(m.title)}</label>
          <span className={tl > 40 ? 'pc-cnt w' : 'pc-cnt'}>{`${tl} / 40`}</span>
        </span>
        <input id="pp-title" className={tl > 40 ? 'in w' : 'in'} value={cfg.title} maxLength={80} onChange={(e) => set('title', e.target.value)} />
      </div>
      <div className="pc-fl">
        <span className="hd">
          <label htmlFor="pp-message">{t(m.message)}</label>
          <span className={ml > 120 ? 'pc-cnt w' : 'pc-cnt'}>{`${ml} / 120`}</span>
        </span>
        <textarea id="pp-message" className={ml > 120 ? 'in w' : 'in'} rows={2} value={cfg.message} maxLength={240} onChange={(e) => set('message', e.target.value)} />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div className="lbl">
          <label htmlFor="pp-allow">{t(m.allowLabel)}</label>
          <input id="pp-allow" className="in w-full" value={cfg.allow} maxLength={24} onChange={(e) => set('allow', e.target.value)} />
        </div>
        <div className="lbl">
          <label htmlFor="pp-notnow">{t(m.notNowLabel)}</label>
          <input id="pp-notnow" className="in w-full" value={cfg.notNow} maxLength={24} onChange={(e) => set('notNow', e.target.value)} />
        </div>
      </div>
      <div className="pp-icon">
        <Mark look={look} size={30} />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="text-[13px] font-semibold">{t(m.icon)}</span>
          <span className="k">{t(m.iconHint)}</span>
        </span>
        {canManage && site?.id ? (
          <>
            <Button size="sm" loading={uploading} onClick={() => file.current?.click()}>
              {t(m.change)}
            </Button>
            <input ref={file} type="file" accept="image/png,image/jpeg" className="vh" tabIndex={-1} aria-hidden="true" onChange={(e) => void changeIcon(e.target.files)} />
          </>
        ) : null}
      </div>
    </fieldset>
  )
}

function WhereSection({
  data,
  cfg,
  set,
  editable,
  siteKeyState,
  setSiteKey,
}: {
  data: Prompt
  cfg: PushPromptConfig
  set: <K extends keyof PushPromptConfig>(k: K, v: PushPromptConfig[K]) => void
  editable: boolean
  siteKeyState: string
  setSiteKey: (k: string) => void
}) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [path, setPath] = useState('')
  const valid = PATH_RE.test(path.trim())
  const add = () => {
    const p = path.trim()
    if (!valid || cfg.paths.includes(p)) return
    set('paths', [...cfg.paths, p])
    setPath('')
    setAdding(false)
  }
  const chips = (
    <div className="pp-paths">
      {cfg.paths.map((p) => (
        <span key={p} className="chip mono">
          {p}
          {editable ? (
            <button type="button" aria-label={t(m.removePath, { path: p })} onClick={() => set('paths', cfg.paths.filter((x) => x !== p))}>
              <svg width="10" height="10" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                <path d="M4 4l8 8M12 4l-8 8" />
              </svg>
            </button>
          ) : null}
        </span>
      ))}
      {editable ? (
        adding ? (
          <span className="flex items-center gap-1.5">
            <input
              className="in mono h-8 w-28"
              value={path}
              aria-label={t(m.pathLabel)}
              placeholder={t(m.pathPlaceholder)}
              aria-invalid={path.trim() !== '' && !valid ? true : undefined}
              onChange={(e) => setPath(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  add()
                }
                if (e.key === 'Escape') setAdding(false)
              }}
              autoFocus
            />
            <Button size="sm" onClick={add}>
              {t(m.pathAdd)}
            </Button>
          </span>
        ) : (
          <Button size="sm" icon="plus" onClick={() => setAdding(true)}>
            {t(m.addPath)}
          </Button>
        )
      ) : null}
    </div>
  )
  return (
    <fieldset className="pp-sec m-0 min-w-0 border-x-0 border-t-0" disabled={!editable}>
      <legend className="t float-left mb-[9px] w-full">{t(m.where)}</legend>
      <div className="lbl clear-both">
        <label htmlFor="pp-site">{t(m.site)}</label>
        <select id="pp-site" className="in" value={siteKeyState} onChange={(e) => setSiteKey(e.target.value)}>
          {data.sites.map((o) => (
            <option key={siteKey(o)} value={siteKey(o)}>
              {!o.pushSite ? t(m.siteNotYet, { host: o.host }) : o.kind === 'publication' ? t(m.siteBuiltIn, { host: o.host }) : t(m.siteSnippet, { host: o.host })}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-col gap-1.5" role="radiogroup" aria-label={t(m.pages)}>
        <span className="text-[12px] font-semibold text-ink-2">{t(m.pages)}</span>
        <label className="flex items-center gap-2 text-[13px]">
          <input type="radio" name="pp-pages" checked={cfg.pages === 'all_except'} onChange={() => set('pages', 'all_except')} />
          {t(m.everyExcept)}
        </label>
        {cfg.pages === 'all_except' ? chips : null}
        <label className="flex items-center gap-2 text-[13px]">
          <input type="radio" name="pp-pages" checked={cfg.pages === 'only'} onChange={() => set('pages', 'only')} />
          {t(m.onlyThese)}
        </label>
        {cfg.pages === 'only' ? chips : null}
      </div>
    </fieldset>
  )
}

function whenCaption(t: ReturnType<typeof useT>, cfg: PushPromptConfig): string {
  if (cfg.style === 'bell') return t(m.whenBell)
  if (cfg.when.seconds.on) return t(m.whenSeconds, { n: cfg.when.seconds.n })
  if (cfg.when.scroll.on) return t(m.whenScroll, { n: cfg.when.scroll.n })
  if (cfg.when.posts.on) return t(m.whenPosts, { n: cfg.when.posts.n })
  if (cfg.when.button) return t(m.whenButton)
  return t(m.whenNever)
}

function Preview({ view, cfg, look, path, data }: { view: View; cfg: PushPromptConfig; look: SiteLook; path: string; data: Prompt }) {
  const t = useT()
  if (view === 'ios') {
    const st = data.stats
    return (
      <div className="pp-row" style={{ gap: 24 }}>
        <figure className="px-pvf" style={{ flex: '0 0 300px' }}>
          <figcaption>
            <b>{t(m.phone)}</b>
            <span className="k">{t(m.safariIphone)}</span>
          </figcaption>
          <SiteFrame look={look} device="iphone" path={path} height={600}>
            {cfg.ios === 'steps' ? <IosSheet cfg={cfg} look={look} /> : null}
          </SiteFrame>
        </figure>
        <div className="flex min-w-0 max-w-[560px] flex-[1_1_320px] flex-col gap-3.5">
          <div className="note i" role="note">
            <InfoIcon color="#1E40AF" />
            <div className="px-note-body">
              <b>{t(m.iosNoteTitle)}</b>
              <div>{t(cfg.ios === 'steps' ? m.iosNoteSteps : m.iosNoteHidden)}</div>
            </div>
          </div>
          <div className="px-funnel">
            <div>
              <span className="k">{t(m.iosHome)}</span>
              <span className="v">{count(st.iosSubscribers)}</span>
              <span className="k">{t(m.iosHomeSub)}</span>
            </div>
            <div>
              <span className="k">{t(m.iosShown)}</span>
              <span className="v">{count(st.iosSteps)}</span>
              <span className="k">{t(m.iosShownSub)}</span>
            </div>
            <div>
              <span className="k">{t(m.iosAdded)}</span>
              <span className="v">{count(st.iosAdded)}</span>
              <span className="k">{pct(st.iosAddedPct)}</span>
            </div>
          </div>
        </div>
      </div>
    )
  }
  if (view === 'blocked') {
    return (
      <div className="pp-row">
        <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-2.5">
          <SiteFrame look={look} device="desktop" path={path} height={250} />
          <div className="note n" role="note">
            <InfoIcon />
            <div>{t(m.blockedText, { host: look.host })}</div>
          </div>
        </div>
        <div className="flex min-w-0 flex-[1_1_360px] flex-col gap-2">
          <span className="k font-semibold text-ink-nav">{t(m.accountExplains)}</span>
          <div className="pp-account">
            <div>
              <span className="text-[15px] font-bold">{t(m.accountTitle)}</span>
              <div className="flex items-center justify-between gap-3">
                <span className="flex flex-col gap-0.5">
                  <span className="text-[14px] font-semibold">{t(m.accountRow)}</span>
                  <span className="text-[12.5px] font-semibold text-bad-fg">{t(m.accountBlocked)}</span>
                </span>
                <input type="checkbox" className="tg" aria-label={t(m.accountRow)} disabled />
              </div>
              <div className="how">
                <T msg={m.accountHow} values={{ b: (c) => <b>{c}</b> }} />
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }
  const notNow = view === 'notnow'
  return (
    <div className="flex flex-col gap-3">
      <div className="pp-row">
        <figure className="px-pvf" style={{ flex: '1 1 440px', minWidth: 0 }}>
          <figcaption>
            <b>{t(m.desktop)}</b>
            <span className="k">{whenCaption(t, cfg)}</span>
          </figcaption>
          <SiteFrame look={look} device="desktop" path={path}>
            {notNow ? null : <PromptOver cfg={cfg} look={look} device="desktop" />}
          </SiteFrame>
        </figure>
        <figure className="px-pvf" style={{ flex: '0 0 250px' }}>
          <figcaption>
            <b>{t(m.phone)}</b>
            <span className="k">{t(m.chromeAndroid)}</span>
          </figcaption>
          <SiteFrame look={look} device="phone" path={path}>
            {notNow ? null : <PromptOver cfg={cfg} look={look} device="phone" />}
          </SiteFrame>
        </figure>
      </div>
      {notNow ? (
        <div className="note n" role="note">
          <InfoIcon />
          <div>{t(m.notNowText, { days: cfg.askAgainDays })}</div>
        </div>
      ) : null}
    </div>
  )
}

function InfoIcon({ color = '#4B5563' }: { color?: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

function Stats({ data, cfg, host }: { data: Prompt; cfg: PushPromptConfig; host: string }) {
  const t = useT()
  const st = data.stats
  const bell = cfg.style === 'bell'
  const cell = (label: string, v: number, sub?: ReactNode) => (
    <div>
      <span className="k">{label}</span>
      <span className="v">{count(v)}</span>
      {sub ? <span className="k">{sub}</span> : null}
    </div>
  )
  return (
    <div className="card flex flex-col gap-3 px-[18px] py-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2.5">
        <h2 className="h2">{st.sinceSnippet ? t(m.sinceSnippet, { date: dayMonthDate(st.since) }) : t(m.last30)}</h2>
        <span className="k">
          {data.liveVersion
            ? t(m.statsWhat, { style: t(styleName[data.live?.style ?? cfg.style]), host, version: data.liveVersion })
            : t(m.statsWhatDraft, { style: t(styleName[cfg.style]), host })}
        </span>
      </div>
      <div className="px-funnel">
        {bell ? cell(t(m.nBell), st.bellClicked) : cell(t(m.nShown), st.shown, t(m.nShownSub))}
        {cell(t(m.nAllow), st.allowed, t(bell ? m.nAllowBellSub : m.nAllowSub, { pct: pct(st.allowedPct) }))}
        {cell(t(m.nGranted), st.granted, t(m.nGrantedSub, { pct: pct(st.grantedPct) }))}
        {cell(t(m.nNotNow), st.notNow, t(bell ? m.nOfClicks : m.nOfShown, { pct: pct(st.notNowPct) }))}
        {cell(t(m.nClosed), st.closed, t(bell ? m.nOfClicks : m.nOfShown, { pct: pct(st.closedPct) }))}
      </div>
      <span className="k">{t(m.statsNote, { days: cfg.askAgainDays })}</span>
    </div>
  )
}

/** “Title” · “Message”: the prompt's text as it was shown. */
function quoted(text: string): string {
  return text
    .split(' · ')
    .map((p) => `“${p}”`)
    .join(' · ')
}

function Consent({ data }: { data: Prompt }) {
  const t = useT()
  const { slug } = useWorkspace()
  const c = data.consent
  return (
    <div className="card flex flex-col gap-3 px-[18px] py-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2.5">
        <h2 className="h2">{t(m.consentTitle)}</h2>
        <span className="k">{t(m.consentHint)}</span>
      </div>
      {c ? (
        <dl className="kv">
          <dt>{t(m.consentSubscription)}</dt>
          <dd>
            {c.contact && c.contactId ? (
              <T
                msg={m.consentLinked}
                values={{
                  browser: t(browserName(c.browser, true)),
                  site: c.site,
                  name: c.contact,
                  // member, preferences or account; anything else came from a link.
                  how: c.linkedBy ?? '',
                  link: (x) => (
                    <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: c.contactId! }}>
                      {x}
                    </Link>
                  ),
                }}
              />
            ) : (
              t(m.consentAnon, { browser: t(browserName(c.browser, true)), site: c.site })
            )}
          </dd>
          <dt>{t(m.consentAllowed)}</dt>
          <dd>{c.ip ? t(m.consentAt, { date: monthDayTimeSeconds(c.at), zone: c.timeZone, ip: c.ip }) : t(m.consentAtNoIp, { date: monthDayTimeSeconds(c.at), zone: c.timeZone })}</dd>
          <dt>{t(m.consentPage)}</dt>
          <dd>
            <span className="mono">{c.page.replace(/^https?:\/\//, '')}</span>
          </dd>
          <dt>{t(m.consentText)}</dt>
          <dd>
            {c.publishedAt && c.publishedBy
              ? t(m.consentVersion, { version: c.version, date: dayMonthDate(c.publishedAt), name: c.publishedBy, text: quoted(c.text) })
              : t(m.consentVersionNoBy, { version: c.version, text: quoted(c.text) })}
          </dd>
        </dl>
      ) : (
        <span className="k">{t(m.consentNone)}</span>
      )}
    </div>
  )
}
