import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN, ProviderChip, useEarnStatus } from './EarnPage'
import { me } from './messages'
import { mp } from './plans.messages'
import { Grip, moved, useReorder } from './Reorder'
import './earn.css'

type Editor = Schemas['EarnUpgradeEditor']
type UpgradePage = Schemas['UpgradePage']
type As = 'visitor' | 'free' | 'paid'

/** The upgrade page editor (EarnUpgradeEditor): the page's options on the left, the page as readers see it on the right. */
export function UpgradePageEditorPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/upgrade-page`, { params: { path: { workspace: slug } } })
  if (q.error || !q.data) {
    return (
      <Frame editor={q.data}>
        {q.error ? (
          <div className="p-7">
            <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>} />
          </div>
        ) : (
          <div className="p-7" aria-busy="true">
            <Skeleton w="100%" h={420} />
          </div>
        )}
      </Frame>
    )
  }
  return <EditorBody key={q.data.publishedAt ?? 'draft'} editor={q.data} />
}

/** The top bar: the breadcrumb, the provider chip, the page's state and the actions. */
function Frame({ editor, actions, children }: { editor?: Editor; actions?: React.ReactNode; children: React.ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = useEarnStatus()
  const crumbs = [
    { label: t(me.title), to: '/w/$workspace/earn' as const, params: { workspace: slug } },
    { label: t(mp.editorCrumb), to: '/w/$workspace/earn/plans' as const, params: { workspace: slug } },
    { label: t(mp.pageCrumb) },
  ]
  return (
    <Page
      title={t(mp.pageTitle)}
      breadcrumb={crumbs}
      badges={
        <>
          <ProviderChip status={status.data} />
          {editor ? (
            <Badge tone={editor.live ? 'ok' : 'neutral'} dot>
              {editor.live ? t(mp.liveAt, { host: editor.host }) : t(mp.notLiveAt, { host: editor.host })}
            </Badge>
          ) : null}
        </>
      }
      actions={actions}
      contentClassName="ea-editor-content"
    >
      {children}
    </Page>
  )
}

function EditorBody({ editor }: { editor: Editor }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [page, setPage] = useState<UpgradePage>(editor.page)
  const [as, setAs] = useState<As>('visitor')
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [faqOpen, setFaqOpen] = useState<number>()
  const save = $api.useMutation('put', `${EARN}/upgrade-page`)
  const ro = !editor.canEdit
  const dirty = JSON.stringify(page) !== JSON.stringify(editor.page)
  const set = <K extends keyof UpgradePage>(k: K, v: UpgradePage[K]) => setPage((p) => ({ ...p, [k]: v }))
  // Public tiers in the page's order; the order keeps any it doesn't know at the end.
  const order = useMemo(() => {
    const pub = editor.tiers.filter((x) => x.visibility === 'public').map((x) => x.id)
    const known = page.order.filter((id) => pub.includes(id))
    return [...known, ...pub.filter((id) => !known.includes(id))]
  }, [page.order, editor.tiers])
  const tiers = useReorder((from, to) => set('order', moved(order, from, to)), order.length)
  const faqs = useReorder((from, to) => set('faq', moved(page.faq, from, to)), page.faq.length)
  const html = usePreview(page, order, as)
  const freeListName = editor.lists.find((l) => l.id === page.freeList)?.name ?? ''
  const publish = () =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { ...page, order } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/upgrade-page`] })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/checklist`] })
          toast({ kind: 'success', title: t(mp.published), detail: editor.host })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const actions = ro ? null : (
    <>
      <Button onClick={() => (dirty ? setPage(editor.page) : void navigate({ to: '/w/$workspace/earn/plans', params: { workspace: slug } }))}>{t(mp.discard)}</Button>
      <Button variant="primary" onClick={publish} loading={save.isPending} busyLabel={t(mp.saving)}>
        {t(mp.publish)}
      </Button>
    </>
  )
  return (
    <Frame editor={editor} actions={actions}>
      <div className="ea-editor">
        <aside className="ea-editor-side" aria-label={t(mp.optionsLabel)}>
          {ro ? (
            <div className="ea-osec">
              <Note tone="neutral">
                <T msg={editor.helpers.length ? mp.readOnly : mp.readOnlyNoHelpers} values={{ helpers: orList(editor.helpers) }} />
              </Note>
            </div>
          ) : null}
          <div className="ea-osec">
            <span className="k">{t(mp.headlineIntro)}</span>
            <label className="lbl">
              {t(mp.headline)}
              <input className="in" aria-label={t(mp.headline)} value={page.headline} maxLength={120} disabled={ro} onChange={(e) => set('headline', e.target.value)} />
            </label>
            <label className="lbl">
              {t(mp.intro)}
              <textarea className="in ea-ta" style={{ height: 72 }} aria-label={t(mp.intro)} value={page.intro} maxLength={600} disabled={ro} onChange={(e) => set('intro', e.target.value)} />
            </label>
          </div>
          <div className="ea-osec">
            <span className="k">{t(mp.tiersOrder)}</span>
            <div className="ea-orow">
              <Grip label={t(mp.moveTier, { tier: `${t(mp.freeName)} · ${freeListName}` })} disabled />
              <label className="ea-orow-l">
                <input type="checkbox" checked disabled />
                {t(mp.freeName)}
                {freeListName ? ` · ${freeListName}` : ''}
              </label>
              <span className="k">{t(mp.freeAlwaysFirst)}</span>
            </div>
            <div data-rows className="contents">
              {order.map((id, i) => {
                const x = editor.tiers.find((y) => y.id === id)
                if (!x) return null
                return (
                  <div key={id} className="ea-orow" data-row {...tiers.row(i)}>
                    <Grip label={t(mp.moveTier, { tier: x.name })} disabled={ro} {...tiers.grip(i)} />
                    <label className="ea-orow-l">
                      <input
                        type="checkbox"
                        checked={!page.off.includes(id)}
                        disabled={ro}
                        onChange={(e) => set('off', e.target.checked ? page.off.filter((o) => o !== id) : [...page.off, id])}
                      />
                      {x.name}
                    </label>
                  </div>
                )
              })}
            </div>
          </div>
          <div className="ea-osec">
            <span className="k">{t(mp.defaultPeriod)}</span>
            <div>
              <Segmented
                label={t(mp.defaultPeriod)}
                value={page.defaultPeriod}
                onChange={(v) => !ro && set('defaultPeriod', v)}
                items={[
                  { value: 'month', label: t(me.periodMonthly) },
                  { value: 'year', label: t(me.periodYearly) },
                ]}
              />
            </div>
          </div>
          <div className="ea-osec">
            <span className="k">{t(mp.highlighted)}</span>
            <div className="flex flex-nowrap gap-2">
              <select className="in flex-1" aria-label={t(mp.highlighted)} value={page.highlight} disabled={ro} onChange={(e) => set('highlight', e.target.value)}>
                {editor.tiers
                  .filter((x) => x.visibility !== 'archived')
                  .map((x) => (
                    <option key={x.id} value={x.id}>
                      {x.name}
                    </option>
                  ))}
                <option value="">{t(mp.none)}</option>
              </select>
              <input
                className="in w-[120px]"
                aria-label={t(mp.highlightLabel)}
                value={page.highlightLabel}
                maxLength={30}
                disabled={ro || !page.highlight}
                onChange={(e) => set('highlightLabel', e.target.value)}
              />
            </div>
          </div>
          <div className="ea-osec">
            <span className="k">{t(mp.freeCard)}</span>
            <label className="lbl">
              {t(mp.freeText)}
              <textarea className="in ea-ta" style={{ height: 56 }} aria-label={t(mp.freeCardText)} value={page.freeText} maxLength={300} disabled={ro} onChange={(e) => set('freeText', e.target.value)} />
            </label>
          </div>
          <div className="ea-osec">
            <span className="k">{t(mp.faq, { count: page.faq.length })}</span>
            <div data-rows className="contents">
              {page.faq.map((f, i) => (
                <div key={i} className="flex flex-col gap-2" data-row {...faqs.row(i)}>
                  <div className="ea-orow">
                    <Grip label={t(mp.moveQuestion)} disabled={ro} {...faqs.grip(i)} />
                    <span className="min-w-0 flex-1 font-medium">{f.q || t(mp.question)}</span>
                    <Button size="sm" aria-expanded={faqOpen === i} onClick={() => setFaqOpen(faqOpen === i ? undefined : i)}>
                      {faqOpen === i ? t(mp.done) : t(mp.edit)}
                    </Button>
                  </div>
                  {faqOpen === i ? (
                    <div className="ea-faq-edit">
                      <label className="lbl">
                        {t(mp.question)}
                        <input className="in" value={f.q} maxLength={160} disabled={ro} onChange={(e) => set('faq', page.faq.map((x, j) => (j === i ? { ...x, q: e.target.value } : x)))} />
                      </label>
                      <label className="lbl">
                        {t(mp.answer)}
                        <textarea
                          className="in ea-ta"
                          style={{ height: 88 }}
                          value={f.a}
                          maxLength={800}
                          disabled={ro}
                          onChange={(e) => set('faq', page.faq.map((x, j) => (j === i ? { ...x, a: e.target.value } : x)))}
                        />
                      </label>
                      {ro ? null : (
                        <Button
                          size="sm"
                          variant="danger"
                          className="self-start"
                          onClick={() => {
                            set(
                              'faq',
                              page.faq.filter((_, j) => j !== i),
                            )
                            setFaqOpen(undefined)
                          }}
                        >
                          {t(mp.removeQuestion)}
                        </Button>
                      )}
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
            {ro || page.faq.length >= 8 ? null : (
              <Button
                size="sm"
                icon="plus"
                className="self-start"
                onClick={() => {
                  set('faq', [...page.faq, { q: '', a: '' }])
                  setFaqOpen(page.faq.length)
                }}
              >
                {t(mp.addQuestion)}
              </Button>
            )}
          </div>
          <div className="ea-osec border-b-0">
            <span className="k">{t(mp.welcomeTitle)}</span>
            <textarea className="in ea-ta" style={{ minHeight: 56 }} aria-label={t(mp.welcomeText)} value={page.welcome} maxLength={400} disabled={ro} onChange={(e) => set('welcome', e.target.value)} />
            <span className="k">{t(mp.welcomeHint, { host: `${editor.host}/welcome` })}</span>
          </div>
        </aside>
        <section className="ea-editor-preview" aria-label={t(mp.previewLabel)}>
          <div className="ea-editor-bar">
            <span className="k">{t(mp.previewAs)}</span>
            <select className="in h-[30px] w-[210px]" aria-label={t(mp.previewAs)} value={as} onChange={(e) => setAs(e.target.value as As)}>
              <option value="visitor">{t(mp.asVisitor)}</option>
              <option value="free">{t(mp.asFree)}</option>
              <option value="paid">{t(mp.asPaid)}</option>
            </select>
            <Segmented
              label={t(mp.device)}
              value={device}
              onChange={setDevice}
              items={[
                { value: 'desktop', label: t(mp.desktop) },
                { value: 'mobile', label: t(mp.mobile) },
              ]}
            />
            {dirty ? <Badge tone="warn">{t(mp.unsaved)}</Badge> : null}
            {editor.testCheckoutUrl ? (
              <ButtonAnchor size="sm" className="ml-auto" href={editor.testCheckoutUrl}>
                {t(mp.previewTest)}
              </ButtonAnchor>
            ) : (
              <span className="k ml-auto">{t(mp.previewTestNone)}</span>
            )}
          </div>
          <div className="ea-pv">
            <PreviewFrame html={html} device={device} label={t(mp.previewLabel)} />
          </div>
        </section>
      </div>
    </Frame>
  )
}

/** The rendered page, as tall as its content. */
function PreviewFrame({ html, device, label }: { html?: string; device: 'desktop' | 'mobile'; label: string }) {
  const ref = useRef<HTMLIFrameElement>(null)
  const [height, setHeight] = useState(900)
  const fit = () => {
    const doc = ref.current?.contentDocument
    if (doc?.documentElement) setHeight(Math.max(400, doc.documentElement.scrollHeight))
  }
  useEffect(() => {
    const id = window.setTimeout(fit, 120)
    return () => window.clearTimeout(id)
  }, [html, device])
  return (
    <div className="ea-pvpage" data-device={device}>
      {html === undefined ? (
        <div className="p-7">
          <Skeleton w="100%" h={600} />
        </div>
      ) : (
        <iframe ref={ref} title={label} srcDoc={html} sandbox="allow-same-origin" style={{ height }} onLoad={fit} />
      )}
    </div>
  )
}

/** The page as readers would see it with these settings (rendered by the server, never published). */
function usePreview(page: UpgradePage, order: string[], as: As): string | undefined {
  const { slug } = useWorkspace()
  const [html, setHtml] = useState<string>()
  const key = JSON.stringify([page, order, as])
  useEffect(() => {
    let live = true
    const timer = window.setTimeout(async () => {
      try {
        const { data } = await http.POST(`${EARN}/upgrade-page/preview`, { params: { path: { workspace: slug } }, body: { page: { ...page, order }, as } })
        if (live && data) setHtml(data.html)
      } catch {
        // The last preview stays.
      }
    }, 350)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, slug])
  return html
}
