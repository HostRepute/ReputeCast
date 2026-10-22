import { keepPreviousData } from '@tanstack/react-query'
import { useSearch } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { cn } from '@/lib/cn'
import { count, longDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { PageCopyView, PageLanguageSection, usePageCopy, type PageGroup } from '@/features/page-languages/PageCopyView'
import { copyText, FormsTabs, hostPath, throwIfForbidden, useInvalidateForms, usePublicPages, type PublicPages } from './common'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m as fb } from './builder.messages'
import { m } from './pages.messages'

type Pages = Schemas['Pages']
type PageInput = Schemas['PublicPageInput']
type PreviewPage = 'subscribe' | 'confirm' | 'preferences' | 'unsubscribe' | 'archive'
/** The page a tab edits, as its messages name it. */
type TabPage = 'subscribe' | 'preferences' | 'unsubscribe' | 'archive'

/** What a tab edits: some pages' options, the hosted form, the preview contact. */
interface Draft {
  pages: Pages
  hostedFormId: string
}

/** The part of the draft a tab saves and previews. */
type Pick = (d: Draft) => PageInput

/**
 * Loads the public page settings and hands a tab its editor once they're
 * there; the editor starts over after each save (new updatedAt).
 */
function PageTab({ name, children }: { name: TabPage; children: (settings: PublicPages) => ReactNode }) {
  const t = useT()
  const settings = usePublicPages()
  throwIfForbidden(settings.error)
  if (settings.isError) {
    return (
      <Page title={t(fb.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(m.loadFailed, { page: name })} actions={<Button onClick={() => settings.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(settings.error)} {t(m.keepsWorking)}
        </ErrorPanel>
      </Page>
    )
  }
  if (settings.isPending) {
    return (
      <Page title={t(fb.pageTitle)} filters={<FormsTabs />} contentClassName="fb-main">
        <div className="fb-cols" aria-busy="true">
          <span className="vh" role="status">
            {t(m.loading, { page: name })}
          </span>
          <aside className="fb-left">
            {[3, 4, 2].map((n, i) => (
              <div key={i} className="fb-sec">
                <Skeleton w={90} h={12} />
                {Array.from({ length: n }, (_, j) => (
                  <Skeleton key={j} h={18} w={j % 2 ? '70%' : '85%'} />
                ))}
              </div>
            ))}
          </aside>
          <section className="fb-center">
            <div className="fb-bar">
              <Skeleton w={220} h={12} />
            </div>
            <div className="fb-stage wide">
              <Skeleton h={520} r={10} />
            </div>
          </section>
        </div>
      </Page>
    )
  }
  return <>{children(settings.data)}</>
}

function draftOf(s: PublicPages): Draft {
  return { pages: structuredClone(s.pages), hostedFormId: s.hostedFormId }
}

interface EditorProps {
  settings: PublicPages
  /** Which page tab: what of the pages' own text it edits in other languages. */
  group: PageGroup
  pick: Pick
  preview: PreviewPage
  previewLabel: string
  previewURL: string
  extraBar?: ReactNode
  form?: string
  options: (draft: Draft, set: (fn: (d: Draft) => Draft) => void, readOnly: boolean) => ReactNode
}

/**
 * One tab: the options on the left, the page as subscribers see it on the
 * right (rendered by the server with the unsaved options), and Save in the
 * top bar while something changed. With ?language= (a workspace language
 * other than the pages' own), the tab edits the page's text in it.
 */
function PageEditor(props: EditorProps) {
  const { language } = useSearch({ strict: false }) as { language?: string }
  const copy = usePageCopy()
  const other = Boolean(language && copy.data && language !== copy.data.copyLanguage && copy.data.languages.some((l) => l.code === language))
  if (other && language) {
    const { settings, pick, preview, previewLabel, previewURL, extraBar, form, group } = props
    const body = pick(draftOf(settings))
    return (
      <PageCopyView
        group={group}
        language={language}
        lists={settings.lists}
        previewLabel={previewLabel}
        previewURL={previewURL}
        extraBar={extraBar}
        preview={(device) => (
          <div className={cn('fb-stage wide', device === 'mobile' && 'phone')}>
            <PagePreview page={preview} draft={body} form={form} language={language} />
          </div>
        )}
      />
    )
  }
  return <BaseEditor {...props} />
}

function BaseEditor({ settings, pick, preview, previewLabel, previewURL, extraBar, options, form }: EditorProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason)
  const initial = useMemo(() => draftOf(settings), [settings])
  const [draft, setDraft] = useState<Draft>(initial)
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/pages')
  const body = pick(draft)
  const dirty = JSON.stringify(body) !== JSON.stringify(pick(initial))
  const submit = async () => {
    try {
      await save.mutateAsync({ params: { path: { workspace: slug } }, body })
      toast({ kind: 'success', title: tt(m.saved), detail: tt(m.savedBody) })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.saveFailed), detail: errorMessage(e), retry: () => void submit().catch(() => undefined) })
      throw e
    }
  }
  return (
    <Page
      title={t(fb.pageTitle)}
      filters={<FormsTabs />}
      actions={
        dirty ? (
          <>
            <Button onClick={() => setDraft(initial)}>{t(fb.discard)}</Button>
            <Button variant="primary" onClick={() => void submit().catch(() => undefined)} loading={save.isPending} busyLabel={t(m.saving)} disabledReason={reason}>
              {t(common.saveChanges)}
            </Button>
          </>
        ) : null
      }
      contentClassName="fb-main"
    >
      <div className="fb-cols">
        <aside className="fb-left" aria-label={t(m.pageOptions)}>
          <PageLanguageSection />
          {options(draft, setDraft, readOnly)}
        </aside>
        <section className="fb-center" aria-label={t(fb.livePreview)}>
          <div className="fb-bar">
            <span className="k">{t(fb.livePreviewWhere, { where: previewURL ? hostPath(previewURL) : previewLabel })}</span>
            <div className="ml-auto flex flex-wrap items-center gap-2.5">
              {extraBar}
              <div className="fb-seg" role="group" aria-label={t(fb.previewSize)}>
                <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                  {t(fb.desktop)}
                </button>
                <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                  {t(fb.mobile)}
                </button>
              </div>
            </div>
          </div>
          <div className={cn('fb-stage wide', device === 'mobile' && 'phone')}>
            <PagePreview page={preview} draft={body} form={form} />
          </div>
        </section>
      </div>
      <UnsavedChangesGuard dirty={dirty && !readOnly} description={t(m.unsaved)} saveLabel={t(m.saveLeave)} onSave={submit} />
    </Page>
  )
}

/**
 * The page as the server renders it, inert, in a frame sized to its content:
 * in a language once its copy is Live there, else as its visitors see it.
 */
function PagePreview({ page, draft, form, language }: { page: PreviewPage; draft: PageInput; form?: string; language?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const json = JSON.stringify(draft)
  const [settled, setSettled] = useState(json)
  useEffect(() => {
    const h = window.setTimeout(() => setSettled(json), 300)
    return () => window.clearTimeout(h)
  }, [json])
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/public-pages/preview/{page}',
    { params: { path: { workspace: slug, page }, query: { draft: settled, form, language } } },
    { placeholderData: keepPreviousData },
  )
  const frame = useRef<HTMLIFrameElement>(null)
  const observer = useRef<ResizeObserver | null>(null)
  const [height, setHeight] = useState(640)
  useEffect(() => () => observer.current?.disconnect(), [])
  const onLoad = () => {
    const doc = frame.current?.contentDocument
    if (!doc?.body) return
    observer.current?.disconnect()
    const measure = () => setHeight(Math.max(doc.documentElement.scrollHeight, 200))
    measure()
    observer.current = new ResizeObserver(measure)
    observer.current.observe(doc.body)
  }
  if (q.isError) {
    return (
      <ErrorPanel title={t(m.previewFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  }
  if (!q.data) {
    return <Skeleton h={520} r={10} className="fb-frame" />
  }
  return (
    <div className="fb-frame">
      <iframe ref={frame} title={t(m.previewOf, { page })} srcDoc={q.data.html} sandbox="allow-same-origin" style={{ height }} onLoad={onLoad} />
    </div>
  )
}

/** "Page address": the page's URL with copy and open. */
function AddressSection({ title, url, note }: { title: string; url: string; note?: ReactNode }) {
  const t = useT()
  return (
    <div className="fb-sec">
      <span className="fb-title">{title}</span>
      {note ? <span className="k">{note}</span> : null}
      <input className="fb-in mono" aria-label={t(m.urlOf, { title })} value={url} readOnly />
      <div className="flex gap-1.5">
        <Button
          size="sm"
          onClick={async () =>
            (await copyText(url))
              ? toast({ kind: 'success', title: tt(fb.linkCopied), detail: hostPath(url) })
              : toast({ kind: 'error', title: tt(fb.copyFailed), detail: tt(fb.copyYourself) })
          }
        >
          {t(fb.copyLink)}
        </Button>
        <a className="btn sm" href={url} target="_blank" rel="noreferrer">
          {t(fb.open)}
        </a>
      </div>
    </div>
  )
}

/** "Preview as": whose page the previews show. */
function PreviewAsSection({ draft, set, readOnly }: { draft: Draft; set: (fn: (d: Draft) => Draft) => void; readOnly: boolean }) {
  const t = useT()
  return (
    <div className="fb-sec last">
      <span className="fb-title">{t(m.previewAs)}</span>
      <input
        className="fb-in mono"
        aria-label={t(m.previewAsLabel)}
        placeholder={t(m.previewAsPlaceholder)}
        value={draft.pages.previewEmail}
        disabled={readOnly}
        onChange={(e) => set((d) => ({ ...d, pages: { ...d.pages, previewEmail: e.target.value } }))}
      />
      <span className="k">{t(m.previewAsNote)}</span>
    </div>
  )
}

/** Per-list lines ("Every Tuesday morning. {subscribers} subscribers."). */
function ListLines({
  lists,
  values,
  readOnly,
  onChange,
  hint,
}: {
  lists: Schemas['PublicPageList'][]
  values: Record<string, string>
  readOnly: boolean
  onChange: (next: Record<string, string>) => void
  hint?: ReactNode
}) {
  return (
    <>
      {lists.map((l) => (
        <label key={l.id} className="lbl">
          {l.name}
          <input
            className="fb-in"
            value={values[l.id] ?? ''}
            placeholder={l.description}
            maxLength={300}
            disabled={readOnly}
            onChange={(e) => onChange({ ...values, [l.id]: e.target.value })}
          />
        </label>
      ))}
      {hint ? <span className="k">{hint}</span> : null}
    </>
  )
}

/** The lists the subscribe page offers: its form's, or every public list. */
function OfferedLists({ formId, lists, children }: { formId: string; lists: Schemas['PublicPageList'][]; children: (offered: Schemas['PublicPageList'][]) => ReactNode }) {
  const { slug } = useWorkspace()
  const form = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}', { params: { path: { workspace: slug, form: formId } } }, { enabled: Boolean(formId) })
  const offered = formId ? lists.filter((l) => form.data?.listIds.includes(l.id)) : lists.filter((l) => l.public)
  return <>{children(offered)}</>
}

export function SubscribePageTab() {
  const t = useT()
  const [shown, setShown] = useState<'subscribe' | 'confirm'>('subscribe')
  return (
    <PageTab name="subscribe">
      {(settings) => {
        const hosted = (d: Draft) => d.hostedFormId
        return (
          <PageEditor
            key={settings.updatedAt}
            settings={settings}
            group="subscribe"
            pick={(d) => ({ subscribe: d.pages.subscribe, confirm: d.pages.confirm, hostedFormId: hosted(d) })}
            preview={shown}
            previewLabel={t(m.pageName, { page: shown })}
            previewURL={shown === 'subscribe' ? settings.urls.subscribe : ''}
            extraBar={
              <div className="fb-seg" role="group" aria-label={t(m.page)}>
                <button type="button" aria-pressed={shown === 'subscribe'} onClick={() => setShown('subscribe')}>
                  {t(m.subscribe)}
                </button>
                <button type="button" aria-pressed={shown === 'confirm'} onClick={() => setShown('confirm')}>
                  {t(m.confirmation)}
                </button>
              </div>
            }
            options={(d, set, readOnly) => {
              const p = d.pages.subscribe
              const c = d.pages.confirm
              const setSub = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, subscribe: { ...x.pages.subscribe, ...patch } } }))
              const setConf = (patch: Partial<typeof c>) => set((x) => ({ ...x, pages: { ...x.pages, confirm: { ...x.pages.confirm, ...patch } } }))
              return (
                <>
                  <AddressSection title={t(m.pageAddress)} url={settings.urls.subscribe} />
                  <div className="fb-sec">
                    <span className="k">{t(m.form)}</span>
                    <select aria-label={t(m.formOnPage)} className="fb-in" value={d.hostedFormId} disabled={readOnly} onChange={(e) => set((x) => ({ ...x, hostedFormId: e.target.value }))}>
                      <option value="">{t(m.builtinForm)}</option>
                      {settings.forms.map((f) => (
                        <option key={f.id} value={f.id}>
                          {f.status === 'draft' ? t(m.draftForm, { name: f.name }) : f.name}
                        </option>
                      ))}
                    </select>
                    <span className="k">{t(m.formNote)}</span>
                  </div>
                  <div className="fb-sec">
                    <span className="k">{t(m.copy)}</span>
                    <label className="lbl">
                      {t(m.heading)}
                      {/* The placeholder is the page's own default heading, in the page's language. */}
                      <input className="fb-in" value={p.heading} maxLength={120} placeholder={`The ${settings.branding.publisherName} newsletter`} disabled={readOnly} onChange={(e) => setSub({ heading: e.target.value })} />
                    </label>
                    <label className="lbl">
                      {t(m.description)}
                      <textarea className="fb-in" value={p.description} maxLength={600} disabled={readOnly} onChange={(e) => setSub({ description: e.target.value })} />
                    </label>
                    <label className="lbl">
                      {t(m.underButton)}
                      <textarea className="fb-in" value={p.footnote} maxLength={300} disabled={readOnly} onChange={(e) => setSub({ footnote: e.target.value })} />
                    </label>
                  </div>
                  <OfferedLists formId={d.hostedFormId} lists={settings.lists}>
                    {(offered) => (
                      <>
                        <div className="fb-sec">
                          <span className="k">{t(m.listLines)}</span>
                          <ListLines
                            lists={offered}
                            values={p.listText}
                            readOnly={readOnly}
                            onChange={(listText) => setSub({ listText })}
                            hint={t(m.subscribersHint, { example: count(31_988) })}
                          />
                        </div>
                        <div className="fb-sec">
                          <span className="k">{t(m.afterClick)}</span>
                          <ListLines lists={offered} values={c.listText} readOnly={readOnly} onChange={(listText) => setConf({ listText })} hint={t(m.nextIssueHint)} />
                        </div>
                      </>
                    )}
                  </OfferedLists>
                  <div className="fb-sec">
                    <span className="k">{t(m.confirmationPage)}</span>
                    <label className="fb-check start">
                      <input type="checkbox" checked={c.inboxTip} disabled={readOnly} onChange={(e) => setConf({ inboxTip: e.target.checked })} />
                      <span>{t(m.inboxTip, { named: settings.senderEmail ? 'yes' : 'no', sender: settings.senderEmail })}</span>
                    </label>
                  </div>
                  <div className="fb-sec last">
                    <span className="k">{t(m.searchEngines)}</span>
                    <label className="fb-check">
                      <input type="checkbox" checked={p.index} disabled={readOnly} onChange={(e) => setSub({ index: e.target.checked })} />
                      {t(m.indexSubscribe)}
                    </label>
                  </div>
                </>
              )
            }}
          />
        )
      }}
    </PageTab>
  )
}

export function PreferencesTab() {
  const t = useT()
  return (
    <PageTab name="preferences">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="preferences"
          pick={(d) => ({ preferences: d.pages.preferences, previewEmail: d.pages.previewEmail })}
          preview="preferences"
          previewLabel={t(m.pageName, { page: 'preferences' })}
          previewURL={settings.urls.preferences}
          options={(d, set, readOnly) => {
            const p = d.pages.preferences
            const setPrefs = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, preferences: { ...x.pages.preferences, ...patch } } }))
            const offered = (id: string, pub: boolean) => (p.lists.length ? p.lists.includes(id) : pub)
            return (
              <>
                <AddressSection title={t(m.pageAddress)} url={settings.urls.preferences} note={t(m.prefsNote)} />
                <div className="fb-sec">
                  <span className="k">{t(m.listsOffered)}</span>
                  {settings.lists.map((l) => (
                    <label key={l.id} className="fb-check">
                      <input
                        type="checkbox"
                        checked={offered(l.id, l.public)}
                        disabled={readOnly}
                        onChange={(e) => {
                          const current = settings.lists.filter((x) => offered(x.id, x.public)).map((x) => x.id)
                          setPrefs({ lists: e.target.checked ? [...current, l.id] : current.filter((x) => x !== l.id) })
                        }}
                      />
                      {l.name}
                    </label>
                  ))}
                  <span className="k">{t(m.alwaysSee)}</span>
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.listLines)}</span>
                  <ListLines lists={settings.lists.filter((l) => offered(l.id, l.public))} values={p.listText} readOnly={readOnly} onChange={(listText) => setPrefs({ listText })} />
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.choices)}</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.frequency} disabled={readOnly} onChange={(e) => setPrefs({ frequency: e.target.checked })} />
                    {t(m.howOften)}
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.pause} disabled={readOnly} onChange={(e) => setPrefs({ pause: e.target.checked })} />
                    {t(m.needBreak)}
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.firstName} disabled={readOnly} onChange={(e) => setPrefs({ firstName: e.target.checked })} />
                    {t(m.firstName)}
                  </label>
                </div>
                <PreviewAsSection draft={d} set={set} readOnly={readOnly} />
              </>
            )
          }}
        />
      )}
    </PageTab>
  )
}

export function UnsubscribeTab() {
  const t = useT()
  const { slug } = useWorkspace()
  const stats = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/unsubscribe/stats', { params: { path: { workspace: slug } } })
  return (
    <PageTab name="unsubscribe">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="unsubscribe"
          pick={(d) => ({ unsubscribe: d.pages.unsubscribe, previewEmail: d.pages.previewEmail })}
          preview="unsubscribe"
          previewLabel={t(m.pageName, { page: 'unsubscribe' })}
          previewURL=""
          options={(d, set, readOnly) => {
            const p = d.pages.unsubscribe
            const setUnsub = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, unsubscribe: { ...x.pages.unsubscribe, ...patch } } }))
            const s = stats.data
            return (
              <>
                <div className="fb-sec">
                  <span className="k">{t(m.fewerInstead)}</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.fewer} disabled={readOnly} onChange={(e) => setUnsub({ fewer: e.target.checked })} />
                    {t(m.offerBestOf)}
                  </label>
                  {p.fewer ? (
                    <>
                      <label className="lbl">
                        {t(m.heading)}
                        <input className="fb-in" value={p.fewerTitle} maxLength={120} disabled={readOnly} onChange={(e) => setUnsub({ fewerTitle: e.target.value })} />
                      </label>
                      <label className="lbl">
                        {t(m.text)}
                        <textarea className="fb-in" value={p.fewerText} maxLength={400} disabled={readOnly} onChange={(e) => setUnsub({ fewerText: e.target.value })} />
                      </label>
                      <label className="lbl">
                        {t(m.button)}
                        <input className="fb-in" value={p.fewerButton} maxLength={60} disabled={readOnly} onChange={(e) => setUnsub({ fewerButton: e.target.value })} />
                      </label>
                    </>
                  ) : null}
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.reason)}</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.reasons} disabled={readOnly} onChange={(e) => setUnsub({ reasons: e.target.checked })} />
                    {t(m.askWhy)}
                  </label>
                  <span className="k">{t(m.answersShow)}</span>
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.oneClick)}</span>
                  <span className="text-[13px] text-ink-2">{t(m.oneClickBody)}</span>
                </div>
                <div className="fb-sec">
                  <span className="fb-title">{t(m.lastDays, { days: s?.days ?? 30 })}</span>
                  {s ? (
                    <div className="fb-nums">
                      <div>
                        <div className="k">{t(m.leftList)}</div>
                        <div className="n text-[18px]">{count(s.unsubscribed)}</div>
                      </div>
                      <div>
                        <div className="k">{t(m.withOneClick)}</div>
                        <div className="n text-[18px]">{count(s.oneClick)}</div>
                      </div>
                      <div>
                        <div className="k">{t(m.choseFewer)}</div>
                        <div className="n text-[18px]">{count(s.choseFewer)}</div>
                      </div>
                      <div>
                        <div className="k">{t(m.saidWhy)}</div>
                        <div className="n text-[18px]">{count(s.answered)}</div>
                      </div>
                    </div>
                  ) : stats.isError ? (
                    <span className="k">{errorMessage(stats.error)}</span>
                  ) : (
                    <Skeleton h={90} />
                  )}
                </div>
                <PreviewAsSection draft={d} set={set} readOnly={readOnly} />
              </>
            )
          }}
        />
      )}
    </PageTab>
  )
}

export function ArchiveTab() {
  const t = useT()
  return (
    <PageTab name="archive">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="archive"
          pick={(d) => ({ archive: d.pages.archive })}
          preview="archive"
          previewLabel={t(m.pageName, { page: 'archive' })}
          previewURL={settings.urls.archive}
          options={(d, set, readOnly) => {
            const p = d.pages.archive
            const setArch = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, archive: { ...x.pages.archive, ...patch } } }))
            const archived = (id: string, pub: boolean) => (p.lists.length ? p.lists.includes(id) : pub)
            return (
              <>
                <div className="fb-sec">
                  <span className="fb-title">{t(m.archive)}</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.enabled} disabled={readOnly} onChange={(e) => setArch({ enabled: e.target.checked })} />
                    {t(m.publishAt)}
                  </label>
                  <input className="fb-in mono" aria-label={t(m.archiveUrl)} value={settings.urls.archive} readOnly />
                  <div className="flex flex-wrap gap-1.5">
                    <Button
                      size="sm"
                      onClick={async () =>
                        (await copyText(settings.urls.archive))
                          ? toast({ kind: 'success', title: tt(fb.linkCopied), detail: hostPath(settings.urls.archive) })
                          : toast({ kind: 'error', title: tt(fb.copyFailed) })
                      }
                    >
                      {t(fb.copyLink)}
                    </Button>
                    <a className="btn sm" href={settings.urls.archive} target="_blank" rel="noreferrer">
                      {t(fb.open)}
                    </a>
                    <a className="btn sm" href={settings.urls.feed} target="_blank" rel="noreferrer">
                      {t(m.rssFeed)}
                    </a>
                  </div>
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.listsPublished)}</span>
                  {settings.lists.map((l) => (
                    <label key={l.id} className="fb-check">
                      <input
                        type="checkbox"
                        checked={archived(l.id, l.public)}
                        disabled={readOnly}
                        onChange={(e) => {
                          const current = settings.lists.filter((x) => archived(x.id, x.public)).map((x) => x.id)
                          setArch({ lists: e.target.checked ? [...current, l.id] : current.filter((x) => x !== l.id) })
                        }}
                      />
                      {l.name}
                    </label>
                  ))}
                  <span className="k">{t(m.appearOnceSent)}</span>
                </div>
                <div className="fb-sec">
                  <span className="k">{t(m.options)}</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.subscribeBox} disabled={readOnly} onChange={(e) => setArch({ subscribeBox: e.target.checked })} />
                    {t(m.subscribeBox)}
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.index} disabled={readOnly} onChange={(e) => setArch({ index: e.target.checked })} />
                    {t(m.indexArchive)}
                  </label>
                </div>
                <ArchiveIssues readOnly={readOnly} lists={settings.lists} />
              </>
            )
          }}
        />
      )}
    </PageTab>
  )
}

/** The archive's issues: hide one, show it again, or change its title. */
function ArchiveIssues({ readOnly, lists }: { readOnly: boolean; lists: Schemas['PublicPageList'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const [page, setPage] = useState(1)
  const [q, setQ] = useState('')
  const [search, setSearch] = useState('')
  const [editing, setEditing] = useState<Schemas['ArchiveIssueItem']>()
  useEffect(() => {
    const h = window.setTimeout(() => {
      setSearch(q.trim())
      setPage(1)
    }, 250)
    return () => window.clearTimeout(h)
  }, [q])
  const issues = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/public-pages/archive/issues',
    { params: { path: { workspace: slug }, query: { q: search, page } } },
    { placeholderData: keepPreviousData },
  )
  const update = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/archive/issues/{campaign}')
  const toggle = async (it: Schemas['ArchiveIssueItem']) => {
    try {
      await update.mutateAsync({
        params: { path: { workspace: slug, campaign: it.campaignId } },
        body: { hidden: !it.hidden, title: it.titleOverride, excerpt: it.excerptOverride, label: it.label, listId: it.listId },
      })
      toast({
        kind: 'success',
        title: it.hidden ? tt(m.backInArchive, { title: it.title }) : tt(m.hidden, { title: it.title }),
        detail: it.hidden ? undefined : tt(m.hiddenBody),
        undo: () => void toggle({ ...it, hidden: !it.hidden }),
      })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.issueFailed), detail: errorMessage(e) })
    }
  }
  const d = issues.data
  return (
    <div className="fb-sec last">
      <span className="fb-title">{d ? t(m.issuesPublished, { count: count(d.published) }) : t(m.issues)}</span>
      <input className="fb-in" type="search" aria-label={t(m.searchIssues)} placeholder={t(m.searchIssues)} value={q} onChange={(e) => setQ(e.target.value)} />
      {issues.isError ? (
        <span className="k">{errorMessage(issues.error)}</span>
      ) : !d ? (
        <Skeleton h={160} />
      ) : d.issues.length === 0 ? (
        <span className="k">{search ? t(m.noMatch, { search }) : t(m.noIssues)}</span>
      ) : (
        <div className="flex flex-col">
          {d.issues.map((it) => (
            <div key={it.campaignId} className="fb-issue">
              <div className="flex items-start justify-between gap-2">
                <a href={it.url} target="_blank" rel="noreferrer" className={cn('min-w-0 text-[13px] font-semibold text-ink no-underline', it.hidden && 'text-ink-3')}>
                  {it.title}
                </a>
                {it.hidden ? <Badge tone="neutral">{t(m.hiddenBadge)}</Badge> : null}
              </div>
              <span className="k">
                {[it.listName, longDate(it.sentAt), it.label].filter(Boolean).join(' · ')}
              </span>
              <div className="flex gap-1.5">
                <Button size="xs" onClick={() => void toggle(it)} disabledReason={readOnly ? t(m.needsEdit) : undefined}>
                  {it.hidden ? t(m.show) : t(m.hide)}
                </Button>
                <Button size="xs" onClick={() => setEditing(it)} disabledReason={readOnly ? t(m.needsEdit) : undefined}>
                  {t(common.edit)}
                </Button>
              </div>
            </div>
          ))}
          <div className="flex items-center justify-between gap-2 pt-2">
            <span className="k" role="status">
              {t(m.range, { from: count((d.page - 1) * d.pageSize + 1), to: count((d.page - 1) * d.pageSize + d.issues.length), total: count(d.total) })}
            </span>
            <div className="flex gap-1.5">
              <Button size="xs" disabled={page <= 1} onClick={() => setPage((x) => x - 1)}>
                {t(m.newer)}
              </Button>
              <Button size="xs" disabled={(d.page - 1) * d.pageSize + d.issues.length >= d.total} onClick={() => setPage((x) => x + 1)}>
                {t(m.older)}
              </Button>
            </div>
          </div>
        </div>
      )}
      {editing ? <IssueDialog issue={editing} lists={lists} onClose={() => setEditing(undefined)} /> : null}
    </div>
  )
}

function IssueDialog({ issue, lists, onClose }: { issue: Schemas['ArchiveIssueItem']; lists: Schemas['PublicPageList'][]; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateForms()
  const update = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/archive/issues/{campaign}')
  const [title, setTitle] = useState(issue.titleOverride)
  const [excerpt, setExcerpt] = useState(issue.excerptOverride)
  const [label, setLabel] = useState(issue.label)
  const [listId, setListId] = useState(issue.listId)
  const [error, setError] = useState<string>()
  const submit = async () => {
    try {
      await update.mutateAsync({ params: { path: { workspace: slug, campaign: issue.campaignId } }, body: { hidden: issue.hidden, title, excerpt, label, listId } })
      toast({ kind: 'success', title: tt(m.issueUpdated), detail: tt(m.issueUpdatedBody) })
      await invalidate()
      onClose()
    } catch (e) {
      setError(e instanceof ApiError ? (e.problem.errors?.[0]?.message ?? e.message) : errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.editIssue)}
      description={t(m.sentAs, { subject: issue.subject })}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={() => void submit()} loading={update.isPending}>
            {t(m.saveIssue)}
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        <Field label={t(m.title)} hint={t(m.titleHint)} error={error}>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder={issue.subject} maxLength={200} className="w-full" />
        </Field>
        <Field label={t(m.excerpt)} hint={t(m.excerptHint)}>
          <Textarea value={excerpt} onChange={(e) => setExcerpt(e.target.value)} maxLength={500} className="w-full" />
        </Field>
        <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
          <Field label={t(m.label)} hint={t(m.labelHint, { example: '#141' })}>
            <Input value={label} onChange={(e) => setLabel(e.target.value)} maxLength={40} className="w-full" />
          </Field>
          <Field label={t(m.filedUnder)}>
            <Select value={listId} onChange={(e) => setListId(e.target.value)} className="w-full">
              {lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </Select>
          </Field>
        </div>
      </form>
    </Dialog>
  )
}
