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

type Pages = Schemas['Pages']
type PageInput = Schemas['PublicPageInput']
type PreviewPage = 'subscribe' | 'confirm' | 'preferences' | 'unsubscribe' | 'archive'

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
function PageTab({ name, children }: { name: string; children: (settings: PublicPages) => ReactNode }) {
  const settings = usePublicPages()
  throwIfForbidden(settings.error)
  if (settings.isError) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <ErrorPanel title={`Couldn't load the ${name}`} actions={<Button onClick={() => settings.refetch()}>Retry</Button>}>
          {errorMessage(settings.error)} Nothing was changed; the page keeps working for subscribers.
        </ErrorPanel>
      </Page>
    )
  }
  if (settings.isPending) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />} contentClassName="fb-main">
        <div className="fb-cols" aria-busy="true">
          <span className="vh" role="status">
            Loading the {name}
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
            <PagePreview page={preview} draft={body} form={form} label={previewLabel} language={language} />
          </div>
        )}
      />
    )
  }
  return <BaseEditor {...props} />
}

function BaseEditor({ settings, pick, preview, previewLabel, previewURL, extraBar, options, form }: EditorProps) {
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
      toast({ kind: 'success', title: 'Saved', detail: 'Subscribers see the change on their next visit.' })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't save the page", detail: errorMessage(e), retry: () => void submit().catch(() => undefined) })
      throw e
    }
  }
  return (
    <Page
      title="Forms & public pages"
      filters={<FormsTabs />}
      actions={
        dirty ? (
          <>
            <Button onClick={() => setDraft(initial)}>Discard</Button>
            <Button variant="primary" onClick={() => void submit().catch(() => undefined)} loading={save.isPending} busyLabel="Saving…" disabledReason={reason}>
              Save changes
            </Button>
          </>
        ) : null
      }
      contentClassName="fb-main"
    >
      <div className="fb-cols">
        <aside className="fb-left" aria-label="Page options">
          <PageLanguageSection />
          {options(draft, setDraft, readOnly)}
        </aside>
        <section className="fb-center" aria-label="Live preview">
          <div className="fb-bar">
            <span className="k">Live preview · {previewURL ? hostPath(previewURL) : previewLabel}</span>
            <div className="ml-auto flex flex-wrap items-center gap-2.5">
              {extraBar}
              <div className="fb-seg" role="group" aria-label="Preview size">
                <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                  Desktop
                </button>
                <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                  Mobile
                </button>
              </div>
            </div>
          </div>
          <div className={cn('fb-stage wide', device === 'mobile' && 'phone')}>
            <PagePreview page={preview} draft={body} form={form} label={previewLabel} />
          </div>
        </section>
      </div>
      <UnsavedChangesGuard dirty={dirty && !readOnly} description="You changed this page's options since they were saved." saveLabel="Save and leave" onSave={submit} />
    </Page>
  )
}

/**
 * The page as the server renders it, inert, in a frame sized to its content:
 * in a language once its copy is Live there, else as its visitors see it.
 */
function PagePreview({ page, draft, form, label, language }: { page: PreviewPage; draft: PageInput; form?: string; label: string; language?: string }) {
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
      <ErrorPanel title="Couldn't show the preview" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  }
  if (!q.data) {
    return <Skeleton h={520} r={10} className="fb-frame" />
  }
  return (
    <div className="fb-frame">
      <iframe ref={frame} title={`Preview of the ${label}`} srcDoc={q.data.html} sandbox="allow-same-origin" style={{ height }} onLoad={onLoad} />
    </div>
  )
}

/** "Page address": the page's URL with copy and open. */
function AddressSection({ title, url, note }: { title: string; url: string; note?: ReactNode }) {
  return (
    <div className="fb-sec">
      <span className="fb-title">{title}</span>
      {note ? <span className="k">{note}</span> : null}
      <input className="fb-in mono" aria-label={`${title} URL`} value={url} readOnly />
      <div className="flex gap-1.5">
        <Button
          size="sm"
          onClick={async () =>
            (await copyText(url))
              ? toast({ kind: 'success', title: 'Link copied', detail: hostPath(url) })
              : toast({ kind: 'error', title: "Couldn't copy the link", detail: 'Select it in the field and copy it yourself.' })
          }
        >
          Copy link
        </Button>
        <a className="btn sm" href={url} target="_blank" rel="noreferrer">
          Open
        </a>
      </div>
    </div>
  )
}

/** "Preview as": whose page the previews show. */
function PreviewAsSection({ draft, set, readOnly }: { draft: Draft; set: (fn: (d: Draft) => Draft) => void; readOnly: boolean }) {
  return (
    <div className="fb-sec last">
      <span className="fb-title">Preview as</span>
      <input
        className="fb-in mono"
        aria-label="Preview as subscriber"
        placeholder="The latest confirmed subscriber"
        value={draft.pages.previewEmail}
        disabled={readOnly}
        onChange={(e) => set((d) => ({ ...d, pages: { ...d.pages, previewEmail: e.target.value } }))}
      />
      <span className="k">The preview shows this subscriber's lists and settings. Nothing changes for them.</span>
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
  const [shown, setShown] = useState<'subscribe' | 'confirm'>('subscribe')
  return (
    <PageTab name="subscribe page">
      {(settings) => {
        const hosted = (d: Draft) => d.hostedFormId
        return (
          <PageEditor
            key={settings.updatedAt}
            settings={settings}
            group="subscribe"
            pick={(d) => ({ subscribe: d.pages.subscribe, confirm: d.pages.confirm, hostedFormId: hosted(d) })}
            preview={shown}
            previewLabel={shown === 'subscribe' ? 'subscribe page' : 'confirmation page'}
            previewURL={shown === 'subscribe' ? settings.urls.subscribe : ''}
            extraBar={
              <div className="fb-seg" role="group" aria-label="Page">
                <button type="button" aria-pressed={shown === 'subscribe'} onClick={() => setShown('subscribe')}>
                  Subscribe
                </button>
                <button type="button" aria-pressed={shown === 'confirm'} onClick={() => setShown('confirm')}>
                  Confirmation
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
                  <AddressSection title="Page address" url={settings.urls.subscribe} />
                  <div className="fb-sec">
                    <span className="k">Form</span>
                    <select aria-label="Form on the subscribe page" className="fb-in" value={d.hostedFormId} disabled={readOnly} onChange={(e) => set((x) => ({ ...x, hostedFormId: e.target.value }))}>
                      <option value="">Built-in form · email, first name, public lists</option>
                      {settings.forms.map((f) => (
                        <option key={f.id} value={f.id}>
                          {f.name}
                          {f.status === 'draft' ? ' · draft' : ''}
                        </option>
                      ))}
                    </select>
                    <span className="k">The page shows this form's fields, lists and consent text. The copy below replaces its heading.</span>
                  </div>
                  <div className="fb-sec">
                    <span className="k">Copy</span>
                    <label className="lbl">
                      Heading
                      <input className="fb-in" value={p.heading} maxLength={120} placeholder={`The ${settings.branding.publisherName} newsletter`} disabled={readOnly} onChange={(e) => setSub({ heading: e.target.value })} />
                    </label>
                    <label className="lbl">
                      Description
                      <textarea className="fb-in" value={p.description} maxLength={600} disabled={readOnly} onChange={(e) => setSub({ description: e.target.value })} />
                    </label>
                    <label className="lbl">
                      Under the button
                      <textarea className="fb-in" value={p.footnote} maxLength={300} disabled={readOnly} onChange={(e) => setSub({ footnote: e.target.value })} />
                    </label>
                  </div>
                  <OfferedLists formId={d.hostedFormId} lists={settings.lists}>
                    {(offered) => (
                      <>
                        <div className="fb-sec">
                          <span className="k">Lists · the line under each</span>
                          <ListLines lists={offered} values={p.listText} readOnly={readOnly} onChange={(listText) => setSub({ listText })} hint="{subscribers} shows the list's size, like 31,988." />
                        </div>
                        <div className="fb-sec">
                          <span className="k">After the confirmation click · what each list brings</span>
                          <ListLines lists={offered} values={c.listText} readOnly={readOnly} onChange={(listText) => setConf({ listText })} hint="The next scheduled issue is added after it." />
                        </div>
                      </>
                    )}
                  </OfferedLists>
                  <div className="fb-sec">
                    <span className="k">Confirmation page</span>
                    <label className="fb-check start">
                      <input type="checkbox" checked={c.inboxTip} disabled={readOnly} onChange={(e) => setConf({ inboxTip: e.target.checked })} />
                      <span>Ask new subscribers to add {settings.senderEmail || 'your sender'} to their contacts</span>
                    </label>
                  </div>
                  <div className="fb-sec last">
                    <span className="k">Search engines</span>
                    <label className="fb-check">
                      <input type="checkbox" checked={p.index} disabled={readOnly} onChange={(e) => setSub({ index: e.target.checked })} />
                      Let search engines list the subscribe page
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
  return (
    <PageTab name="preference center">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="preferences"
          pick={(d) => ({ preferences: d.pages.preferences, previewEmail: d.pages.previewEmail })}
          preview="preferences"
          previewLabel="preference center"
          previewURL={settings.urls.preferences}
          options={(d, set, readOnly) => {
            const p = d.pages.preferences
            const setPrefs = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, preferences: { ...x.pages.preferences, ...patch } } }))
            const offered = (id: string, pub: boolean) => (p.lists.length ? p.lists.includes(id) : pub)
            return (
              <>
                <AddressSection
                  title="Page address"
                  url={settings.urls.preferences}
                  note="Every email links to each subscriber's own page. This address emails that link to whoever asks for it."
                />
                <div className="fb-sec">
                  <span className="k">Lists offered</span>
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
                  <span className="k">Subscribers always see the lists they're on.</span>
                </div>
                <div className="fb-sec">
                  <span className="k">Lists · the line under each</span>
                  <ListLines lists={settings.lists.filter((l) => offered(l.id, l.public))} values={p.listText} readOnly={readOnly} onChange={(listText) => setPrefs({ listText })} />
                </div>
                <div className="fb-sec">
                  <span className="k">Choices</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.frequency} disabled={readOnly} onChange={(e) => setPrefs({ frequency: e.target.checked })} />
                    How often · every issue, weekly roundup, monthly best-of
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.pause} disabled={readOnly} onChange={(e) => setPrefs({ pause: e.target.checked })} />
                    Need a break · pause 30 or 90 days
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.firstName} disabled={readOnly} onChange={(e) => setPrefs({ firstName: e.target.checked })} />
                    First name
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
  const { slug } = useWorkspace()
  const stats = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/unsubscribe/stats', { params: { path: { workspace: slug } } })
  return (
    <PageTab name="unsubscribe page">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="unsubscribe"
          pick={(d) => ({ unsubscribe: d.pages.unsubscribe, previewEmail: d.pages.previewEmail })}
          preview="unsubscribe"
          previewLabel="unsubscribe page"
          previewURL=""
          options={(d, set, readOnly) => {
            const p = d.pages.unsubscribe
            const setUnsub = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, unsubscribe: { ...x.pages.unsubscribe, ...patch } } }))
            const s = stats.data
            return (
              <>
                <div className="fb-sec">
                  <span className="k">Fewer emails instead</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.fewer} disabled={readOnly} onChange={(e) => setUnsub({ fewer: e.target.checked })} />
                    Offer the monthly best-of before they leave
                  </label>
                  {p.fewer ? (
                    <>
                      <label className="lbl">
                        Heading
                        <input className="fb-in" value={p.fewerTitle} maxLength={120} disabled={readOnly} onChange={(e) => setUnsub({ fewerTitle: e.target.value })} />
                      </label>
                      <label className="lbl">
                        Text
                        <textarea className="fb-in" value={p.fewerText} maxLength={400} disabled={readOnly} onChange={(e) => setUnsub({ fewerText: e.target.value })} />
                      </label>
                      <label className="lbl">
                        Button
                        <input className="fb-in" value={p.fewerButton} maxLength={60} disabled={readOnly} onChange={(e) => setUnsub({ fewerButton: e.target.value })} />
                      </label>
                    </>
                  ) : null}
                </div>
                <div className="fb-sec">
                  <span className="k">Reason</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.reasons} disabled={readOnly} onChange={(e) => setUnsub({ reasons: e.target.checked })} />
                    Ask why they're leaving · optional for them
                  </label>
                  <span className="k">Answers show in each campaign's report.</span>
                </div>
                <div className="fb-sec">
                  <span className="k">One-click unsubscribe</span>
                  <span className="text-[13px] text-ink-2">
                    Every email carries List-Unsubscribe and List-Unsubscribe-Post (RFC 8058). Mail apps unsubscribe with one click, without this page, and it takes effect at once.
                  </span>
                </div>
                <div className="fb-sec">
                  <span className="fb-title">Last {s?.days ?? 30} days</span>
                  {s ? (
                    <div className="fb-nums">
                      <div>
                        <div className="k">Left a list</div>
                        <div className="n text-[18px]">{count(s.unsubscribed)}</div>
                      </div>
                      <div>
                        <div className="k">With one click</div>
                        <div className="n text-[18px]">{count(s.oneClick)}</div>
                      </div>
                      <div>
                        <div className="k">Chose fewer emails</div>
                        <div className="n text-[18px]">{count(s.choseFewer)}</div>
                      </div>
                      <div>
                        <div className="k">Said why</div>
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
  return (
    <PageTab name="archive">
      {(settings) => (
        <PageEditor
          key={settings.updatedAt}
          settings={settings}
          group="archive"
          pick={(d) => ({ archive: d.pages.archive })}
          preview="archive"
          previewLabel="archive"
          previewURL={settings.urls.archive}
          options={(d, set, readOnly) => {
            const p = d.pages.archive
            const setArch = (patch: Partial<typeof p>) => set((x) => ({ ...x, pages: { ...x.pages, archive: { ...x.pages.archive, ...patch } } }))
            const archived = (id: string, pub: boolean) => (p.lists.length ? p.lists.includes(id) : pub)
            return (
              <>
                <div className="fb-sec">
                  <span className="fb-title">Archive</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.enabled} disabled={readOnly} onChange={(e) => setArch({ enabled: e.target.checked })} />
                    Publish past issues at
                  </label>
                  <input className="fb-in mono" aria-label="Archive URL" value={settings.urls.archive} readOnly />
                  <div className="flex flex-wrap gap-1.5">
                    <Button
                      size="sm"
                      onClick={async () =>
                        (await copyText(settings.urls.archive))
                          ? toast({ kind: 'success', title: 'Link copied', detail: hostPath(settings.urls.archive) })
                          : toast({ kind: 'error', title: "Couldn't copy the link" })
                      }
                    >
                      Copy link
                    </Button>
                    <a className="btn sm" href={settings.urls.archive} target="_blank" rel="noreferrer">
                      Open
                    </a>
                    <a className="btn sm" href={settings.urls.feed} target="_blank" rel="noreferrer">
                      RSS feed
                    </a>
                  </div>
                </div>
                <div className="fb-sec">
                  <span className="k">Lists published</span>
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
                  <span className="k">Sent campaigns of these lists appear once they finish sending.</span>
                </div>
                <div className="fb-sec">
                  <span className="k">Options</span>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.subscribeBox} disabled={readOnly} onChange={(e) => setArch({ subscribeBox: e.target.checked })} />
                    Show "Get the next one in your inbox"
                  </label>
                  <label className="fb-check">
                    <input type="checkbox" checked={p.index} disabled={readOnly} onChange={(e) => setArch({ index: e.target.checked })} />
                    Let search engines list it · sitemap.xml
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
        title: it.hidden ? `"${it.title}" is back in the archive` : `"${it.title}" is hidden`,
        detail: it.hidden ? undefined : 'It leaves the archive, the feed and the sitemap. The email itself is unchanged.',
        undo: () => void toggle({ ...it, hidden: !it.hidden }),
      })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't change the issue", detail: errorMessage(e) })
    }
  }
  const d = issues.data
  return (
    <div className="fb-sec last">
      <span className="fb-title">Issues{d ? ` · ${count(d.published)} published` : ''}</span>
      <input className="fb-in" type="search" aria-label="Search issues" placeholder="Search issues" value={q} onChange={(e) => setQ(e.target.value)} />
      {issues.isError ? (
        <span className="k">{errorMessage(issues.error)}</span>
      ) : !d ? (
        <Skeleton h={160} />
      ) : d.issues.length === 0 ? (
        <span className="k">{search ? `No issue matches “${search}”.` : 'No issues yet. Sent campaigns of the published lists appear here.'}</span>
      ) : (
        <div className="flex flex-col">
          {d.issues.map((it) => (
            <div key={it.campaignId} className="fb-issue">
              <div className="flex items-start justify-between gap-2">
                <a href={it.url} target="_blank" rel="noreferrer" className={cn('min-w-0 text-[13px] font-semibold text-ink no-underline', it.hidden && 'text-ink-3')}>
                  {it.title}
                </a>
                {it.hidden ? <Badge tone="neutral">Hidden</Badge> : null}
              </div>
              <span className="k">
                {[it.listName, longDate(it.sentAt), it.label].filter(Boolean).join(' · ')}
              </span>
              <div className="flex gap-1.5">
                <Button size="xs" onClick={() => void toggle(it)} disabledReason={readOnly ? 'Editing the archive needs the forms:edit permission.' : undefined}>
                  {it.hidden ? 'Show' : 'Hide'}
                </Button>
                <Button size="xs" onClick={() => setEditing(it)} disabledReason={readOnly ? 'Editing the archive needs the forms:edit permission.' : undefined}>
                  Edit
                </Button>
              </div>
            </div>
          ))}
          <div className="flex items-center justify-between gap-2 pt-2">
            <span className="k" role="status">
              {count((d.page - 1) * d.pageSize + 1)} to {count((d.page - 1) * d.pageSize + d.issues.length)} of {count(d.total)}
            </span>
            <div className="flex gap-1.5">
              <Button size="xs" disabled={page <= 1} onClick={() => setPage((x) => x - 1)}>
                Newer
              </Button>
              <Button size="xs" disabled={(d.page - 1) * d.pageSize + d.issues.length >= d.total} onClick={() => setPage((x) => x + 1)}>
                Older
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
      toast({ kind: 'success', title: 'Issue updated', detail: 'The archive, its feed and the issue page show it now.' })
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
      title="Edit archived issue"
      description={`Sent as "${issue.subject}". The email itself doesn't change.`}
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" onClick={() => void submit()} loading={update.isPending}>
            Save issue
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
        <Field label="Title" hint="Empty uses the subject line." error={error}>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder={issue.subject} maxLength={200} className="w-full" />
        </Field>
        <Field label="Excerpt" hint="Empty uses the preheader.">
          <Textarea value={excerpt} onChange={(e) => setExcerpt(e.target.value)} maxLength={500} className="w-full" />
        </Field>
        <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
          <Field label="Label" hint='Shown after the date, like "#141".'>
            <Input value={label} onChange={(e) => setLabel(e.target.value)} maxLength={40} className="w-full" />
          </Field>
          <Field label="Filed under">
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
