import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { OptionCard } from '@/components/ui/Field'
import { Icon } from '@/components/icons/Icon'
import { Menu, MenuItem, MenuLabel, MenuSeparator } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { blockDefs, type BlockDef } from '@/features/composer/editor/blocks'
import { PreviewDialog, TestDialog } from '../Dialogs'
import { useDigest, useRefreshDigest } from '../shared'
import { blockOf, DEFAULT_BLOCK, FEED_TYPE, isFeedItems, markerFor, setBlock, setFeedPreview, type FeedBlock } from './feedComponent'
import { dc } from './messages'

type Preview = 'next' | 'last' | 'one' | 'six'
type Render = Schemas['DigestRender']
type Digest = Schemas['DigestDetail']

interface DigestMode {
  digest: Digest | undefined
  render: Render | undefined
  preview: Preview
  setPreview(p: Preview): void
  /** The canvas's Feed items component, if it has one. */
  feed: Component | null
  block: FeedBlock
}

const Ctx = createContext<DigestMode | null>(null)

/** Digest mode of the composer, when the campaign is a digest's template; null otherwise. */
export function useDigestMode(): DigestMode | null {
  return useContext(Ctx)
}

/** Whether the composer edits a recurring digest's template. */
export function isDigestTemplate(campaign: { digestRole?: string; digestId?: string }): boolean {
  return campaign.digestRole === 'template' && Boolean(campaign.digestId)
}

/**
 * Wraps the composer: in digest mode it keeps the "Preview with" choice,
 * draws the Feed items block with that preview's posts (the server renders
 * them) and shares the render with the subject previews and the checks.
 */
export function DigestModeProvider({ children }: { children: ReactNode }) {
  const c = useComposer()
  if (!isDigestTemplate(c.campaign)) return children
  return <Provider digestId={c.campaign.digestId!}>{children}</Provider>
}

function Provider({ digestId, children }: { digestId: string; children: ReactNode }) {
  const c = useComposer()
  const q = useDigest(digestId)
  const [preview, setPreview] = useState<Preview>('next')
  // The canvas's Feed items component and its settings, read again on every edit.
  const feed = useMemo(() => (c.editor ? ((c.editor.getWrapper()?.findType(FEED_TYPE)[0] as Component | undefined) ?? null) : null), [c.editor, c.rev]) // eslint-disable-line react-hooks/exhaustive-deps
  const block = feed ? blockOf(feed) : DEFAULT_BLOCK
  const blockKey = JSON.stringify(block)
  // Typing settles before the server draws the posts again.
  const [settled, setSettled] = useState({ subject: c.subject, preheader: c.preheader, blockKey })
  useEffect(() => {
    const h = window.setTimeout(() => setSettled({ subject: c.subject, preheader: c.preheader, blockKey }), 350)
    return () => window.clearTimeout(h)
  }, [c.subject, c.preheader, blockKey])
  const render = useQuery({
    queryKey: ['digest-render', c.slug, digestId, preview, settled],
    queryFn: async () => {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/render', {
        params: { path: { workspace: c.slug, digestId } },
        // "6 items" shows the whole issue with its size by number of posts.
        body: { preview, block: JSON.parse(settled.blockKey) as FeedBlock, subject: settled.subject, preheader: settled.preheader, sizes: preview === 'six' },
      })
      return data!
    },
    placeholderData: (prev) => prev,
    staleTime: 30_000,
  })
  useEffect(() => {
    if (render.data) setFeedPreview(render.data.mjml)
  }, [render.data])
  // Leaving digest mode clears the canvas preview.
  useEffect(() => () => setFeedPreview(''), [])
  const value = useMemo<DigestMode>(() => ({ digest: q.data, render: render.data, preview, setPreview, feed, block }), [q.data, render.data, preview, feed, block])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

/** The Feed items block of the Blocks panel (ReputeCast group), in digest mode only. */
export function feedItemsBlock(label: string): BlockDef {
  return { id: 'rc-feed-items', label, group: 'reputecast', icon: 'rcFeed', kind: 'section', content: markerFor(DEFAULT_BLOCK), keywords: 'feed rss posts digest' }
}

/** The Blocks panel in digest mode: Feed items first among ReputeCast's blocks, and why it's there. */
export function digestLeftPanel(campaign: { digestRole?: string; digestId?: string }, address: string): { blocks?: BlockDef[]; note?: ReactNode; reputecastFirst?: boolean } {
  if (!isDigestTemplate(campaign)) return {}
  return {
    blocks: [feedItemsBlock(tt(dc.blockTitle)), ...blockDefs(address)],
    note: (
      <p className="rounded-lg border border-line bg-ground-3 text-[12px] leading-[1.45] text-ink-2" style={{ margin: '6px 12px 12px', padding: '10px 12px' }}>
        {tt(dc.blockNote)}
      </p>
    ),
    reputecastFirst: true,
  }
}

/** The header's "Recurring digest" chip, on the template and on its issues. */
export function DigestBadge() {
  const t = useT()
  const c = useComposer()
  if (!c.campaign.digestId || !c.campaign.digestRole) return null
  return (
    <span className="chip">
      <Icon name="repeat" size={13} className="text-ink-3" />
      {t(dc.recurringDigest)}
    </span>
  )
}

/** The header's actions in digest mode: Send test (the next issue) and Check next issue. */
export function DigestHeaderActions() {
  const t = useT()
  const c = useComposer()
  const mode = useDigestMode()
  const navigate = useNavigate()
  const refresh = useRefreshDigest()
  const canEdit = useCan('campaigns:edit')
  const [testing, setTesting] = useState(false)
  const [checking, setChecking] = useState(false)
  const digest = mode?.digest
  if (!mode) return null
  // Without campaigns:edit the template is read-only: its next issue can still be previewed.
  if (!canEdit) return <PreviewNextIssueButton />
  const check = async () => {
    if (!digest) return
    setChecking(true)
    try {
      // The next issue is built from what the template has now.
      await c.save({ background: false })
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/check', { params: { path: { workspace: c.slug, digestId: digest.id } } })
      refresh()
      toast({
        kind: data?.verdict === 'blocked' ? 'error' : 'success',
        title: t(dc.checked, { verdict: data?.verdict ?? 'ready' }),
        action: data?.campaignId
          ? { label: t(dc.openChecks), onClick: () => void navigate({ to: '/w/$workspace/campaigns/$campaignId/review', params: { workspace: c.slug, campaignId: data.campaignId } }) }
          : undefined,
      })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setChecking(false)
    }
  }
  return (
    <>
      <button type="button" className="btn" disabled={!digest} onClick={() => setTesting(true)}>
        {t(dc.sendTest)}
      </button>
      <button type="button" className="btn pri" disabled={!digest || checking} aria-busy={checking || undefined} onClick={() => void check()}>
        {checking ? t(dc.checking) : t(dc.checkNext)}
      </button>
      {testing && digest ? <TestDialog digest={digest} onOpenChange={() => setTesting(false)} /> : null}
    </>
  )
}

/** "Preview next issue": what a member who can't edit the template can do from its header. */
function PreviewNextIssueButton() {
  const t = useT()
  const mode = useDigestMode()
  const [open, setOpen] = useState(false)
  if (!mode?.digest) return null
  return (
    <>
      <button type="button" className="btn" onClick={() => setOpen(true)}>
        {t(dc.previewNextIssue)}
      </button>
      {open ? <PreviewDialog digest={mode.digest} onOpenChange={() => setOpen(false)} /> : null}
    </>
  )
}

/** A Liquid token as the boards show one: mono, on purple. */
export const tokenClass = 'mono inline-flex items-center whitespace-nowrap rounded-[5px] bg-purple-bg px-1.5 text-[0.86em] font-medium leading-[1.55] text-purple-fg'

const contactToken = '{{ first_name | default: "there" }}'

/** "Feed tokens": the issue's tokens with their values for the next issue; picking one adds it to the subject. */
export function FeedTokensMenu({ onInsert }: { onInsert: (token: string) => void }) {
  const t = useT()
  const mode = useDigestMode()
  if (!mode) return null
  const tokens = mode.render?.tokens ?? []
  return (
    <Menu
      label={t(dc.feedTokensLabel)}
      className="w-[min(460px,calc(100vw-24px))]"
      trigger={
        <button type="button" className="btn sm" style={{ minHeight: 24, padding: '0 8px', fontSize: 11.5 }}>
          <span className="mono" aria-hidden="true">
            {'{ }'}
          </span>
          {t(dc.feedTokens)}
        </button>
      }
    >
      <MenuLabel>{t(dc.tokensHead)}</MenuLabel>
      {tokens.map((x) => (
        <MenuItem key={x.token} onSelect={() => onInsert(x.token)} className="justify-between">
          <span className={tokenClass}>{x.token}</span>
          <span className="k min-w-0 truncate">{x.value}</span>
        </MenuItem>
      ))}
      <div className="flex flex-col gap-[3px]" style={{ padding: '6px 10px 4px' }}>
        <span className={cn(tokenClass, 'self-start')}>issue.items</span>
        <span className="k">{t(dc.tokensItems)}</span>
      </div>
      <MenuSeparator />
      <MenuLabel>{t(dc.tokensContact)}</MenuLabel>
      <MenuItem onSelect={() => onInsert(contactToken)} className="justify-between">
        <span className={tokenClass}>{contactToken}</span>
        <span className="k min-w-0 truncate">{t(dc.tokensContactValue)}</span>
      </MenuItem>
    </Menu>
  )
}

const countToken = /\s*(?:and|\+)?\s*\{\{\s*issue\.items\.size\s*\|\s*minus:\s*1\s*\}\}(?:\s*more)?/i
const sizeGuard = /\{%-?\s*if\s+issue\.items\.size\s*>\s*1\s*-?%\}/

/** The subject with its "and N more" wrapped so it only shows with 2 or more items, or null when there is nothing to wrap. */
export function wrapCount(subject: string): string | null {
  if (sizeGuard.test(subject)) return null
  const m = countToken.exec(subject)
  if (!m) return null
  return `${subject.slice(0, m.index)}{% if issue.items.size > 1 %}${m[0]}{% endif %}${subject.slice(m.index + m[0].length)}`
}

/** "Subject with 1 item · 3 items · 6 items": the subject as each issue size would read, so "…and 0 more" is caught. */
export function SubjectPreviews({ subject, onSubject }: { subject: string; onSubject: (v: string) => void }) {
  const t = useT()
  const c = useComposer()
  const mode = useDigestMode()
  if (!mode) return null
  const subjects = mode.render?.subjects ?? []
  if (!subjects.length) return null
  const zero = subjects.find((x) => x.zeroMore)
  const wrapped = zero ? wrapCount(subject) : null
  const starter = mode.render?.starterSubject ?? ''
  return (
    // Sections, not divs: the subject bar lays its divs out as columns.
    <>
      <section className="flex w-full basis-full flex-wrap items-center gap-2" aria-label={t(dc.subjectWith)}>
        <span className="k">{t(dc.subjectWith)}</span>
        {subjects.map((x) => (
          <span
            key={x.items}
            className={cn('flex min-w-0 flex-[1_1_260px] items-center gap-2 rounded-md border px-2.5 py-1.5 text-[12.5px]', x.zeroMore ? 'border-warn-line bg-warn-tint' : 'border-line bg-surface')}
          >
            <span className="shrink-0 font-semibold">{t(dc.subjectItems, { count: x.items })}</span>
            <span className={cn('shrink-0', x.zeroMore ? 'text-warn-fg' : 'text-ok-fg')} aria-label={x.zeroMore ? t(dc.zeroMore) : t(dc.subjectOk)} role="img">
              <Icon name={x.zeroMore ? 'alert' : 'check'} size={13} />
            </span>
            <span className="truncate">{x.text}</span>
          </span>
        ))}
      </section>
      {zero ? (
        <section className="note w w-full basis-full" role="note" aria-label={t(dc.zeroMoreTitle)}>
          <span className="text-warn-fg">
            <Icon name="alert" size={16} />
          </span>
          <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
            <span className="font-bold">{t(dc.zeroMoreTitle)}</span>
            <span>{t(dc.zeroMoreBody, { count: zero.items, text: zero.text })}</span>
            {c.canEdit ? (
              <span className="mt-1 flex flex-wrap gap-2">
                {wrapped ? (
                  <button type="button" className="btn sm pri" onClick={() => onSubject(wrapped)}>
                    {t(dc.wrapIt)}
                  </button>
                ) : null}
                {starter && starter !== subject ? (
                  <button type="button" className="btn sm" onClick={() => onSubject(starter)}>
                    {t(dc.useStarter)}
                  </button>
                ) : null}
              </span>
            ) : null}
          </div>
        </section>
      ) : null}
    </>
  )
}

/** "Preview with": the posts the Feed items block shows on the canvas. */
export function DigestPreviewWith() {
  const t = useT()
  const mode = useDigestMode()
  if (!mode) return null
  const d = mode.digest
  const nextNumber = d?.next?.number ?? (d ? d.sentIssues + 1 : 1)
  const nextPosts = d?.next?.posts ?? 0
  const lastPosts = mode.preview === 'last' ? (mode.render?.count ?? 0) : 0
  return (
    <label className="inline-flex items-center gap-2 text-[12.5px]">
      <span className="k">{t(dc.previewWith)}</span>
      <select className="in" style={{ minHeight: 28, padding: '0 28px 0 8px', fontSize: 12.5 }} value={mode.preview} onChange={(e) => mode.setPreview(e.target.value as Preview)}>
        <option value="next">{t(dc.previewNext, { number: nextNumber, count: nextPosts })}</option>
        {d && d.sentIssues > 0 ? <option value="last">{t(dc.previewLast, { number: d.sentIssues, count: lastPosts || (mode.render?.count ?? 0) })}</option> : null}
        <option value="one">{t(dc.previewOne)}</option>
        <option value="six">{t(dc.previewSix)}</option>
      </select>
    </label>
  )
}

/** The footer's summary in digest mode: the next issue's size, the largest issue's, images and links. */
export function DigestFooterSummary() {
  const t = useT()
  const mode = useDigestMode()
  if (!mode?.render) return null
  const r = mode.render
  const kb = (n: number) => `${Math.max(1, Math.round(n / 1024))} KB`
  return (
    <span className="k">
      {t(dc.footerSummary, {
        preview: mode.preview,
        size: kb(r.checks.bytes || r.checks.maxBytes),
        count: r.checks.maxPosts,
        max: kb(r.checks.maxBytes),
        limit: kb(r.checks.limit),
        images: t(dc.footerImages, { count: r.checks.images, fromTitle: Math.min(r.checks.altFromTitle, r.checks.images) }),
        links: t(dc.footerLinks, { count: r.checks.links }),
      })}
    </span>
  )
}

/** The Feed items block's settings (the inspector's Settings tab when it is selected). */
export function FeedItemsInspector({ c: comp }: { c: Component }) {
  const t = useT()
  const composer = useComposer()
  const mode = useDigestMode()
  const b = blockOf(comp)
  const set = (patch: Partial<FeedBlock>) => setBlock(comp, { ...b, ...patch })
  const has = (f: string) => b.fields.includes(f)
  const toggle = (f: string) => set({ fields: has(f) ? b.fields.filter((x) => x !== f) : [...b.fields, f] })
  const r = mode?.render
  const field = (f: string, label: string, extra?: ReactNode) => (
    <div className="flex items-center gap-2">
      <input id={`feed-${f}`} type="checkbox" checked={has(f)} onChange={() => toggle(f)} />
      <label htmlFor={`feed-${f}`}>{label}</label>
      {extra}
    </div>
  )
  const layoutHint = b.layout === 'list' ? t(dc.layoutHintList) : b.layout === 'grid' ? t(dc.layoutHintGrid) : t(dc.layoutHintCards)
  const chooseFallback = () => {
    composer.setAssetTarget({ component: comp, attr: 'src' })
    composer.openDialog('assets')
  }
  return (
    <div className="flex flex-col text-[13px]">
      <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <span className="flex items-center gap-1.5 font-semibold">
          <Icon name="rss" size={13} />
          {t(dc.blockTitle)}
        </span>
        <span className="k">{t(dc.blockSub)}</span>
      </div>
      <section className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.layout)}>
        <span className="font-semibold">{t(dc.layout)}</span>
        <div className="seg w-fit" role="radiogroup" aria-label={t(dc.layout)}>
          {(['list', 'cards', 'grid'] as const).map((l) => (
            <button key={l} type="button" role="radio" aria-checked={b.layout === l} className={cn(b.layout === l && 'on')} onClick={() => set({ layout: l })}>
              {l === 'list' ? t(dc.layoutList) : l === 'cards' ? t(dc.layoutCards) : t(dc.layoutGrid)}
            </button>
          ))}
        </div>
        <span className="k">{layoutHint}</span>
      </section>
      <section className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.fields)}>
        <span className="font-semibold">{t(dc.fields)}</span>
        <div className="grid grid-cols-2 gap-x-3 gap-y-2">
          {field('image', t(dc.fieldImage))}
          {field('title', t(dc.fieldTitle))}
          {field('date', t(dc.fieldDate))}
          {field('author', t(dc.fieldAuthor))}
          {field('categories', t(dc.fieldCategories))}
        </div>
        {field(
          'excerpt',
          t(dc.fieldExcerpt),
          <>
            <input
              className="in w-[56px] text-center"
              type="number"
              min={5}
              max={200}
              aria-label={t(dc.excerptWords)}
              value={b.excerptWords}
              disabled={!has('excerpt')}
              onChange={(e) => set({ excerptWords: Math.max(5, Math.min(200, Number(e.target.value) || 40)) })}
            />
            <span>{t(dc.words)}</span>
          </>,
        )}
        {field(
          'button',
          t(dc.fieldButton),
          <input className="in min-w-0 flex-1" aria-label={t(dc.buttonText)} maxLength={60} value={b.buttonText} disabled={!has('button')} onChange={(e) => set({ buttonText: e.target.value })} />,
        )}
        <span className="k">{t(dc.excerptNote)}</span>
      </section>
      <section className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.posts)}>
        <span className="font-semibold">{t(dc.posts)}</span>
        <div className="grid grid-cols-2 gap-3">
          <label className="flex flex-col gap-1">
            <span className="k">{t(dc.maximum)}</span>
            <input className="in" type="number" min={1} max={20} value={b.max} onChange={(e) => set({ max: Math.max(1, Math.min(20, Number(e.target.value) || 6)) })} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="k">{t(dc.order)}</span>
            <select className="in" value={b.order} onChange={(e) => set({ order: e.target.value as FeedBlock['order'] })}>
              <option value="newest">{t(dc.orderNewest)}</option>
              <option value="oldest">{t(dc.orderOldest)}</option>
            </select>
          </label>
        </div>
        <span className="k">{t(dc.maximumHint)}</span>
      </section>
      <section className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.overflow)}>
        <span className="font-semibold">{t(dc.overflow)}</span>
        <OptionCard
          name="feed-overflow"
          value="drop"
          checked={b.overflow === 'drop'}
          onChange={() => set({ overflow: 'drop' })}
          title={t(dc.overflowDrop, { count: b.max })}
          description={t(dc.overflowDropHint)}
        />
        <OptionCard name="feed-overflow" value="keep" checked={b.overflow === 'keep'} onChange={() => set({ overflow: 'keep' })} title={t(dc.overflowKeep)} description={t(dc.overflowKeepHint)} />
      </section>
      <section className="flex flex-col gap-2 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.fallback)}>
        <span className="font-semibold">{t(dc.fallback)}</span>
        <div className="flex items-center gap-3">
          {b.fallback ? <img src={b.fallback} alt="" className="h-9 w-14 shrink-0 rounded border border-line object-cover" /> : null}
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate font-semibold">{b.fallbackName || t(dc.fallbackNone)}</span>
            <span className="k">{t(dc.fallbackHint)}</span>
          </span>
          <button type="button" className="btn sm" onClick={chooseFallback}>
            {b.fallback ? t(dc.replace) : t(dc.choose)}
          </button>
        </div>
      </section>
      <section className="flex flex-col gap-1 border-b border-line-2" style={{ padding: '10px 12px' }} aria-label={t(dc.images)}>
        <span className="font-semibold">{t(dc.images)}</span>
        <span className="k">{t(dc.imagesNote)}</span>
      </section>
      {r ? (
        <section className="flex flex-col gap-1.5" style={{ padding: '10px 12px', marginTop: 'auto' }} aria-label={t(dc.checksTitle)}>
          <span className="k">{t(dc.checksTitle)}</span>
          <Badge tone={r.checks.maxBytes > r.checks.limit ? 'bad' : 'ok'} className="self-start">
            {t(r.checks.maxBytes > r.checks.limit ? dc.checkSizeOver : dc.checkSize, {
              count: r.checks.maxPosts,
              size: `${Math.round(r.checks.maxBytes / 1024)} KB`,
              limit: `${Math.round(r.checks.limit / 1024)} KB`,
            })}
          </Badge>
          <Badge tone="ok" className="self-start">
            {t(dc.checkImages, { count: r.checks.images, alt: r.checks.altFromTitle > 0 && r.checks.altFromTitle === r.checks.images ? 'title' : 'feed' })}
          </Badge>
          <Badge tone="ok" className="self-start">
            {t(dc.checkLinks, { count: r.checks.links })}
          </Badge>
        </section>
      ) : null}
    </div>
  )
}

export { isFeedItems }
