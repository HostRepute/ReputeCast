import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN, ProviderChip, useEarnStatus } from './EarnPage'
import { me } from './messages'
import { mp } from './plans.messages'
import './earn.css'

type Editor = Schemas['EarnUpgradeEditor']
type UpgradePage = Schemas['UpgradePage']
type As = 'visitor' | 'free' | 'paid' | 'offer'

/** The upgrade page editor (EarnUpgradeEditor): the page's settings on the left, its live preview on the right. */
export function UpgradePageEditorPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/upgrade-page`, { params: { path: { workspace: slug } } })
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
          {q.data ? (
            <Badge tone={q.data.live ? 'ok' : 'neutral'} dot>
              {q.data.live ? t(mp.liveAt, { host: q.data.host }) : t(mp.notLiveAt, { host: q.data.host })}
            </Badge>
          ) : null}
        </>
      }
      contentClassName="ea-editor-content"
    >
      {q.error ? (
        <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>} />
      ) : !q.data ? (
        <div className="card pad" aria-busy="true">
          <Skeleton w="100%" h={300} />
        </div>
      ) : (
        <EditorBody key={q.data.publishedAt ?? 'draft'} editor={q.data} />
      )}
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
  const order = useMemo(() => {
    const ids = page.order.length ? page.order : editor.tiers.map((x) => x.id)
    const known = ids.filter((id) => editor.tiers.some((x) => x.id === id))
    return [...known, ...editor.tiers.filter((x) => !known.includes(x.id)).map((x) => x.id)]
  }, [page.order, editor.tiers])
  const tier = (id: string) => editor.tiers.find((x) => x.id === id)
  const move = (i: number, d: -1 | 1) => {
    const next = [...order]
    const a = next[i]
    const b = next[i + d]
    if (a === undefined || b === undefined) return
    next[i] = b
    next[i + d] = a
    set('order', next)
  }
  const html = usePreview(page, as)
  const publish = () =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { ...page, order } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/upgrade-page`] })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
          toast({ kind: 'success', title: t(mp.published), detail: editor.host })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const freeListName = editor.lists.find((l) => l.id === page.freeList)?.name ?? ''
  return (
    <div className="ea-editor">
      <aside className="ea-editor-side" aria-label={t(mp.pageTitle)}>
        <div className="flex flex-wrap justify-end gap-2 px-4 pt-4">
          <Button size="sm" disabled={!dirty} onClick={() => (dirty ? setPage(editor.page) : void navigate({ to: '/w/$workspace/earn/plans', params: { workspace: slug } }))}>
            {t(mp.discard)}
          </Button>
          {ro ? null : (
            <Button size="sm" variant="primary" onClick={publish} loading={save.isPending} busyLabel={t(mp.saving)}>
              {t(mp.publish)}
            </Button>
          )}
        </div>
        {ro ? (
          <div className="px-4 pt-3">
            <Note tone="neutral">
              <T msg={editor.helpers.length ? mp.readOnly : mp.readOnlyNoHelpers} values={{ permission: <span className="mono">earn:edit</span>, helpers: orList(editor.helpers) }} />
            </Note>
          </div>
        ) : null}
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.headline)}</h2>
          <Field label={t(mp.headline)}>
            <Input value={page.headline} maxLength={120} disabled={ro} onChange={(e) => set('headline', e.target.value)} />
          </Field>
          <Field label={t(mp.intro)}>
            <Textarea value={page.intro} maxLength={600} rows={3} disabled={ro} onChange={(e) => set('intro', e.target.value)} />
          </Field>
        </div>
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.tiersOrder)}</h2>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            <li className="flex items-center justify-between gap-2 text-[13px]">
              <label className="flex items-center gap-2">
                <input type="checkbox" checked disabled />
                {t(mp.freeName)}
                {freeListName ? ` · ${freeListName}` : ''}
              </label>
              <span className="k">{t(mp.freeAlwaysFirst)}</span>
            </li>
            {order.map((id, i) => {
              const x = tier(id)
              if (!x || x.visibility !== 'public') return null
              const shown = !page.off.includes(id)
              return (
                <li key={id} className="flex items-center justify-between gap-2 text-[13px]">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={shown}
                      disabled={ro}
                      aria-label={t(mp.showTier, { tier: x.name })}
                      onChange={(e) => set('off', e.target.checked ? page.off.filter((o) => o !== id) : [...page.off, id])}
                    />
                    {x.name}
                  </label>
                  {ro ? null : (
                    <span className="flex gap-1">
                      <Button size="xs" iconOnly icon="chevronUp" aria-label={t(mp.moveUp, { tier: x.name })} disabled={i === 0} onClick={() => move(i, -1)} />
                      <Button size="xs" iconOnly icon="chevronDown" aria-label={t(mp.moveDown, { tier: x.name })} disabled={i === order.length - 1} onClick={() => move(i, 1)} />
                    </span>
                  )}
                </li>
              )
            })}
          </ul>
        </div>
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.defaultPeriod)}</h2>
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
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.highlighted)}</h2>
          <div className="flex gap-2">
            <Select aria-label={t(mp.highlighted)} value={page.highlight} disabled={ro} onChange={(e) => set('highlight', e.target.value)}>
              {editor.tiers
                .filter((x) => x.visibility !== 'archived')
                .map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
              <option value="">{t(mp.none)}</option>
            </Select>
            <Input aria-label={t(mp.highlightLabel)} value={page.highlightLabel} maxLength={30} disabled={ro || !page.highlight} onChange={(e) => set('highlightLabel', e.target.value)} />
          </div>
        </div>
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.freeCard)}</h2>
          <Field label={t(mp.freeListLabel)}>
            <Select value={page.freeList} disabled={ro} onChange={(e) => set('freeList', e.target.value)}>
              {editor.lists.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t(mp.freeText)}>
            <Textarea value={page.freeText} maxLength={300} rows={2} disabled={ro} onChange={(e) => set('freeText', e.target.value)} />
          </Field>
        </div>
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.faq, { count: page.faq.length })}</h2>
          <ul className="m-0 flex list-none flex-col gap-2 p-0">
            {page.faq.map((f, i) => (
              <li key={i} className="flex flex-col gap-2">
                <div className="flex items-center justify-between gap-2 text-[13px]">
                  <span className="font-semibold">{f.q || t(mp.question)}</span>
                  <Button size="xs" onClick={() => setFaqOpen(faqOpen === i ? undefined : i)}>
                    {t(mp.edit)}
                  </Button>
                </div>
                {faqOpen === i ? (
                  <div className="flex flex-col gap-2">
                    <Field label={t(mp.question)}>
                      <Input value={f.q} maxLength={160} disabled={ro} onChange={(e) => set('faq', page.faq.map((x, j) => (j === i ? { ...x, q: e.target.value } : x)))} />
                    </Field>
                    <Field label={t(mp.answer)}>
                      <Textarea value={f.a} maxLength={800} rows={3} disabled={ro} onChange={(e) => set('faq', page.faq.map((x, j) => (j === i ? { ...x, a: e.target.value } : x)))} />
                    </Field>
                    {ro ? null : (
                      <Button size="xs" variant="danger" onClick={() => set('faq', page.faq.filter((_, j) => j !== i))}>
                        {t(mp.removeQuestion)}
                      </Button>
                    )}
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
          {ro || page.faq.length >= 8 ? null : (
            <Button
              size="sm"
              icon="plus"
              onClick={() => {
                set('faq', [...page.faq, { q: '', a: '' }])
                setFaqOpen(page.faq.length)
              }}
            >
              {t(mp.addQuestion)}
            </Button>
          )}
        </div>
        <div className="ea-editor-sec">
          <h2 className="ea-eyebrow">{t(mp.welcomeTitle)}</h2>
          <Field label={t(mp.welcomeText)} hint={t(mp.welcomeHint, { host: `${editor.host}/welcome`, first: '{{ first_name }}', next: '{{ next_issue }}' })}>
            <Textarea value={page.welcome} maxLength={400} rows={3} disabled={ro} onChange={(e) => set('welcome', e.target.value)} />
          </Field>
        </div>
      </aside>
      <section className="ea-editor-preview" aria-label={t(mp.previewLabel)}>
        <div className="ea-editor-bar">
          <label className="flex items-center gap-2 text-[13px]">
            <span className="k">{t(mp.previewAs)}</span>
            <Select value={as} onChange={(e) => setAs(e.target.value as As)} className="max-w-[230px]">
              <option value="visitor">{t(mp.asVisitor)}</option>
              <option value="free">{t(mp.asFree)}</option>
              <option value="paid">{t(mp.asPaid)}</option>
            </Select>
          </label>
          <Segmented
            label={t(mp.device)}
            value={device}
            onChange={setDevice}
            items={[
              { value: 'desktop', label: t(mp.desktop) },
              { value: 'mobile', label: t(mp.mobile) },
            ]}
          />
          <span className="flex-1" />
          {dirty ? <Badge tone="warn">{t(mp.unsaved)}</Badge> : null}
          {editor.testCheckout ? (
            <ButtonAnchor size="sm" href={`${editor.url}${editor.url.includes('?') ? '&' : '?'}preview=test`}>
              {t(mp.previewTest)}
            </ButtonAnchor>
          ) : (
            <span className="k">{t(mp.previewTestNone)}</span>
          )}
        </div>
        <div className="ea-editor-frame" data-device={device}>
          {html === undefined ? <Skeleton w="100%" h={600} /> : <iframe title={t(mp.previewLabel)} srcDoc={html} sandbox="allow-same-origin" />}
        </div>
      </section>
    </div>
  )
}

/** The page as readers would see it with these settings (rendered by the server, never published). */
function usePreview(page: UpgradePage, as: As): string | undefined {
  const { slug } = useWorkspace()
  const [html, setHtml] = useState<string>()
  const key = JSON.stringify([page, as])
  useEffect(() => {
    let live = true
    const timer = window.setTimeout(async () => {
      try {
        const { data } = await http.POST(`${EARN}/upgrade-page/preview`, { params: { path: { workspace: slug } }, body: { page, as } })
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
