import { Link } from '@tanstack/react-router'
import { useId, useState, type KeyboardEvent, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, pct } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { recBadge } from './OverviewPage'
import { PreviewButton } from './Previews'
import { m } from './messages'
import { mc } from './recommendations.messages'
import { GROW, GrowPage, LoadError, NewsletterTile, ReadOnlyNote, copy, orNames, uploadFile, useGrowRefresh } from './shared'

type View = Schemas['RecommendationsView']
type Item = Schemas['RecommendationItem']
type You = Schemas['RecommendingYouItem']

/**
 * Grow › Recommendations (Recommendations, RecommendationsTablet,
 * RecommendationsPhone, RecommendationStates): what you recommend, who
 * recommends you, where recommendations show, your partner profile, and the
 * dialogs to add a link, connect a partner or let a newsletter recommend you.
 */
export function RecommendationsPage({ search, onSearch }: { search: { add?: boolean }; onSearch: (s: { add?: boolean }) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/recommendations`, { params: { path: { workspace: slug } } })
  const v = q.data
  const canEdit = useCan('grow:edit')
  const canConnect = useCan('grow:connect')
  const [codeOpen, setCodeOpen] = useState(false)
  const status = q.error instanceof ApiError ? q.error.status : undefined
  const actions =
    v && (canEdit || canConnect) ? (
      <>
        <Button disabledReason={canConnect ? undefined : deniedReason('grow:connect', v.helpers.connect)} onClick={() => setCodeOpen(true)}>
          {t(mc.letRecommend)}
        </Button>
        {canEdit ? (
          <Button variant="primary" icon="plus" onClick={() => onSearch({ add: true })}>
            {t(mc.addRecommendation)}
          </Button>
        ) : null}
      </>
    ) : null
  return (
    <GrowPage tab="recommendations" documentTitle={t(mc.docTitle)} actions={actions}>
      {!v ? (
        q.isError ? (
          <LoadError
            error={q.error}
            title={t(mc.loadError)}
            onRetry={() => void q.refetch()}
            secondary={
              <Link to="/w/$workspace/grow" params={{ workspace: slug }} className="btn">
                {t(m.growOverview)}
              </Link>
            }
          >
            {status ? t(mc.loadErrorText, { status }) : t(mc.loadErrorTextNoStatus)}
          </LoadError>
        ) : (
          <Loading label={t(mc.loadingLabel)}>
            <div className="gw">
              <Skeleton w="100%" h={300} r={10} />
              <Skeleton w="100%" h={180} r={10} />
            </div>
          </Loading>
        )
      ) : (
        <Body v={v} onAdd={() => onSearch({ add: true })} onCode={() => setCodeOpen(true)} />
      )}
      {v && search.add ? <AddDialog v={v} onClose={() => onSearch({})} /> : null}
      {v && codeOpen ? <CodeDialog v={v} onClose={() => setCodeOpen(false)} /> : null}
    </GrowPage>
  )
}

function Body({ v, onAdd, onCode }: { v: View; onAdd: () => void; onCode: () => void }) {
  const t = useT()
  const canEdit = useCan('grow:edit')
  const canConnect = useCan('grow:connect')
  if (!v.items.length && !v.recommendingYou.length) {
    return (
      <div className="gw">
        <EmptyState
          title={t(mc.emptyTitle)}
          action={
            canEdit || canConnect ? (
              <>
                {canEdit ? (
                  <Button variant="primary" icon="plus" onClick={onAdd}>
                    {t(mc.addRecommendation)}
                  </Button>
                ) : null}
                {canConnect ? <Button onClick={onCode}>{t(mc.letRecommend)}</Button> : null}
              </>
            ) : null
          }
        >
          <p className="m-0">{t(mc.emptyText)}</p>
          <p className="k m-0 mt-2">{t(mc.noMarketplace)}</p>
        </EmptyState>
        <ProfileCard v={v} />
      </div>
    )
  }
  return (
    <div className="gw">
      {!canEdit && !canConnect ? (
        <ReadOnlyNote>
          {t(v.helpers.edit.length && v.helpers.connect.length ? mc.readOnlyText : mc.readOnlyTextNoHelpers, {
            editHelpers: orNames(v.helpers.edit),
            connectHelpers: orNames(v.helpers.connect),
          })}
        </ReadOnlyNote>
      ) : canEdit && !canConnect ? (
        <Note tone="neutral">
          {t(v.helpers.connect.length ? mc.editorNoConnect : mc.editorNoConnectNoHelpers, { helpers: orNames(v.helpers.connect) })}
        </Note>
      ) : null}
      <YouRecommend v={v} />
      <RecommendingYou v={v} />
      <div className="gw-g2 gw-st">
        <WhereCard v={v} />
        <ProfileCard v={v} />
      </div>
    </div>
  )
}

function showsOn(t: ReturnType<typeof useT>, it: Item): string {
  if (it.showConfirm && it.showBlock) return t(mc.showsBoth)
  if (it.showConfirm) return t(mc.showsConfirmOnly)
  if (it.showBlock) return t(mc.showsBlockOnly)
  return t(mc.showsNone)
}

function sinceText(t: ReturnType<typeof useT>, it: Item): string {
  if (!it.since) return ''
  const d = `${dayMonthDate(it.since)}${it.state === 'invalid' || it.state === 'unreachable' ? ` ${clock(it.since)}` : ''}`
  return it.hidden ? t(mc.sinceHidden, { date: d }) : t(mc.since, { date: d })
}

/** You recommend · 3: the order readers see, with numbers for 30 days. */
function YouRecommend({ v }: { v: View }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const refresh = useGrowRefresh()
  const [order, setOrder] = useState<string[] | null>(null)
  const items = order ? order.map((id) => v.items.find((x) => x.id === id)!).filter(Boolean) : v.items
  const [announce, setAnnounce] = useState('')
  const move = async (id: string, by: number) => {
    const ids = items.map((x) => x.id)
    const i = ids.indexOf(id)
    const j = i + by
    if (j < 0 || j >= ids.length) return
    ;[ids[i], ids[j]] = [ids[j]!, ids[i]!]
    setOrder(ids)
    const it = items[i]!
    setAnnounce(t(mc.moved, { name: it.name, place: j + 1 }))
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/recommendations/order', { params: { path: { workspace: slug } }, body: { ids } })
      await refresh()
      setOrder(null)
    } catch (e) {
      setOrder(null)
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const onKey = (e: KeyboardEvent, id: string) => {
    if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
      e.preventDefault()
      void move(id, -1)
    } else if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
      e.preventDefault()
      void move(id, 1)
    }
  }
  const [drag, setDrag] = useState<string>()
  const onDrop = (target: string) => {
    if (!drag || drag === target) return
    const from = items.findIndex((x) => x.id === drag)
    const to = items.findIndex((x) => x.id === target)
    void move(drag, to - from)
    setDrag(undefined)
  }
  return (
    <section className="card">
      <CardSection title={t(mc.youRecommend, { count: v.items.length })} caption={canEdit ? t(mc.youRecommendCaption) : t(mc.youRecommendCaptionRead)} actions={<span className="k">{t(mc.shownMeans)}</span>} />
      <span className="vh" role="status">
        {announce}
      </span>
      {v.items.length ? (
        <>
          <div className="max-md:hidden">
            <TableScroll label={t(mc.tableLabel)}>
              <table className="tbl gw-tbl">
                <thead>
                  <tr>
                    <th scope="col">{t(mc.colOrder)}</th>
                    <th scope="col">{t(mc.colNewsletter)}</th>
                    <th scope="col">{t(mc.colShowsOn)}</th>
                    <th scope="col">{t(mc.colStatus)}</th>
                    <th scope="col" className="r">
                      {t(mc.colShown)}
                    </th>
                    <th scope="col" className="r">
                      {t(mc.colRequests)}
                    </th>
                    <th scope="col" className="r">
                      {t(mc.colConfirmed)}
                    </th>
                    <th scope="col">
                      <span className="vh">{t(mc.colActions)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((it, i) => (
                    <RecRows key={it.id} it={it} i={i} n={items.length} onKey={onKey} onDragStart={() => setDrag(it.id)} onDrop={() => onDrop(it.id)} />
                  ))}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="md:hidden">
            {items.map((it, i) => (
              <RecCard key={it.id} it={it} i={i} onKey={onKey} />
            ))}
          </div>
        </>
      ) : (
        <div className="gw-pad k">{t(mc.emptyText)}</div>
      )}
    </section>
  )
}

function Handle({ it, i, onKey, draggable, onDragStart }: { it: Item; i: number; onKey: (e: KeyboardEvent, id: string) => void; draggable?: boolean; onDragStart?: () => void }) {
  const t = useT()
  const canEdit = useCan('grow:edit')
  return (
    <span className="flex items-center gap-1.5">
      {canEdit ? (
        <button
          type="button"
          className="gw-handle"
          aria-label={t(mc.moveLabel, { name: it.name })}
          draggable={draggable}
          onDragStart={onDragStart}
          onKeyDown={(e) => onKey(e, it.id)}
        >
          <Icon name="move" size={14} />
        </button>
      ) : null}
      <span className="gw-num k">{i + 1}</span>
    </span>
  )
}

function RecRows({ it, i, onKey, onDragStart, onDrop }: { it: Item; i: number; n: number; onKey: (e: KeyboardEvent, id: string) => void; onDragStart: () => void; onDrop: () => void }) {
  const t = useT()
  const b = recBadge(it)
  const dash = <span className="k">{t(m.none)}</span>
  const rate = it.requests ? pct((it.confirmed / it.requests) * 100, 0) : ''
  return (
    <>
      <tr onDragOver={(e) => e.preventDefault()} onDrop={onDrop}>
        <td>
          <Handle it={it} i={i} onKey={onKey} draggable onDragStart={onDragStart} />
        </td>
        <td>
          <span className="flex min-w-0 items-center gap-2.5">
            <NewsletterTile name={it.name} initials={it.initials} logo={it.logo?.url} />
            <span className="flex min-w-0 flex-col gap-px">
              <span className="gw-tt2">{it.name}</span>
              {it.description ? <span className="k">{it.description}</span> : null}
              <span className="k">
                <b className="font-semibold text-ink-nav">{it.kind === 'partner' ? t(mc.kindPartner) : t(mc.kindLink)}</b> · <span className="mono">{it.host}</span>
              </span>
            </span>
          </span>
        </td>
        <td>
          <span className="gw-where flex flex-col gap-0.5">
            {it.showConfirm ? <span>{t(mc.showsConfirm)}</span> : null}
            {it.showBlock ? <span>{t(mc.showsBlock)}</span> : null}
            {!it.showConfirm && !it.showBlock ? <span className="k">{t(mc.showsNowhere)}</span> : null}
          </span>
        </td>
        <td>
          <span className="flex flex-col items-start gap-0.5">
            <Badge tone={b.tone} dot>
              {b.label}
            </Badge>
            {it.since ? <span className="k">{sinceText(t, it)}</span> : null}
          </span>
        </td>
        <td className="r gw-num">{count(it.shown)}</td>
        <td className="r gw-num">
          {count(it.kind === 'partner' ? it.requests : it.clicks)}
          <div className="k">{it.kind === 'partner' ? t(mc.requestsUnit, { count: it.requests }) : t(mc.clicksUnit, { count: it.clicks })}</div>
        </td>
        <td className="r gw-num">
          {it.kind === 'partner' ? (
            <>
              {count(it.confirmed)} <span className="k">{rate ? t(mc.ratePart, { pct: rate }) : null}</span>
            </>
          ) : (
            dash
          )}
        </td>
        <td className="r">
          <RowMenu it={it} />
        </td>
      </tr>
      {it.state !== 'showing' && it.state !== 'connected' ? (
        <tr>
          <td />
          <td colSpan={7} style={{ paddingTop: 0 }}>
            <StateNote it={it} />
          </td>
        </tr>
      ) : null}
    </>
  )
}

function RecCard({ it, i, onKey }: { it: Item; i: number; onKey: (e: KeyboardEvent, id: string) => void }) {
  const t = useT()
  const b = recBadge(it)
  return (
    <div className="flex flex-col gap-2 border-t border-line-2 px-4 py-3">
      <div className="flex items-start gap-2.5">
        <Handle it={it} i={i} onKey={onKey} />
        <NewsletterTile name={it.name} initials={it.initials} logo={it.logo?.url} />
        <span className="flex min-w-0 flex-1 flex-col gap-px">
          <span className="gw-tt2">{it.name}</span>
          {it.description ? <span className="k">{it.description}</span> : null}
          <span className="k">
            <b className="font-semibold text-ink-nav">{it.kind === 'partner' ? t(mc.kindPartner) : t(mc.kindLink)}</b> · <span className="mono">{it.host}</span>
          </span>
        </span>
        <RowMenu it={it} />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Badge tone={b.tone} dot>
          {b.label}
        </Badge>
        <span className="k">
          {it.since ? `${sinceText(t, it)} · ` : ''}
          {showsOn(t, it)}
        </span>
      </div>
      <KeyValues
        items={[
          [t(mc.labelShown), count(it.shown)],
          [it.kind === 'partner' ? t(mc.labelRequests) : t(mc.labelClicks), count(it.kind === 'partner' ? it.requests : it.clicks)],
          [t(mc.labelConfirmed), it.kind === 'partner' ? t(mc.confirmedRate, { count: count(it.confirmed), pct: it.requests ? pct((it.confirmed / it.requests) * 100, 0) : '–' }) : t(m.none)],
        ]}
      />
      {it.state !== 'showing' && it.state !== 'connected' ? <StateNote it={it} /> : null}
    </div>
  )
}

/** What a recommendation that doesn't show needs: a new code, a resume, a check. */
function StateNote({ it }: { it: Item }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const canConnect = useCan('grow:connect')
  const refresh = useGrowRefresh()
  const [pasting, setPasting] = useState(false)
  const [removing, setRemoving] = useState(false)
  const since = it.since ? `${dayMonthDate(it.since)} ${clock(it.since)}` : ''
  const resume = async () => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/recommendations/{recommendation}/resume', { params: { path: { workspace: slug, recommendation: it.id } } })
      toast({ kind: 'success', title: t(mc.resumed, { name: it.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const check = async () => {
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/check', { params: { path: { workspace: slug, link: it.linkId } } })
      toast(data?.state === 'unreachable' ? { kind: 'error', title: t(mc.checked, { name: it.name }), detail: t(mc.checkedBad, { name: it.name }) } : { kind: 'success', title: t(mc.checked, { name: it.name }), detail: t(mc.checkedOk, { name: it.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  let body: ReactNode = null
  switch (it.state) {
    case 'invalid':
      body = (
        <Note tone="bad">
          <span className="flex flex-col gap-1.5">
            <span className="font-semibold">
              {it.invalidReason === 'ended'
                ? t(mc.invalidTextEnded, { date: since })
                : it.invalidReason === 'unknown_link'
                  ? t(mc.invalidTextUnknown, { date: since })
                  : t(mc.invalidText, { date: since })}
            </span>
            <span>{t(mc.invalidHelp)}</span>
            {canConnect || canEdit ? (
              <span className="flex flex-wrap gap-2">
                {canConnect ? (
                  <Button size="sm" variant="primary" onClick={() => setPasting(true)}>
                    {t(mc.pasteNewCode)}
                  </Button>
                ) : null}
                {canConnect ? (
                  <Button size="sm" onClick={() => setRemoving(true)}>
                    {t(mc.remove)}
                  </Button>
                ) : null}
              </span>
            ) : null}
          </span>
        </Note>
      )
      break
    case 'paused':
      body = (
        <Note tone="neutral">
          <span className="flex flex-col gap-1.5">
            <span className="font-semibold">{it.pausedBy && it.since ? t(mc.pausedBy, { by: it.pausedBy, date: `${dayMonthDate(it.since)}, ${clock(it.since)}` }) : t(mc.statePaused)}</span>
            <span>{t(mc.pausedText)}</span>
            {(it.kind === 'link' ? canEdit : canConnect) ? (
              <span>
                <Button size="sm" onClick={() => void (it.kind === 'link' ? resume() : resumeLink(slug, it, refresh, t))}>
                  {t(mc.resume)}
                </Button>
              </span>
            ) : null}
          </span>
        </Note>
      )
      break
    case 'partner_paused':
      body = <Note tone="warn">{t(mc.partnerPausedText, { name: it.name })}</Note>
      break
    case 'limit':
      body = (
        <Note tone="warn">
          <span className="flex flex-col gap-1">
            <span className="font-semibold">{t(mc.limitText, { name: it.name })}</span>
            <span className="k">{t(mc.limitLine, { today: it.today, limit: it.dailyLimit })}</span>
          </span>
        </Note>
      )
      break
    case 'unreachable':
      body = (
        <Note tone="warn">
          <span className="flex flex-col gap-1.5">
            <span className="font-semibold">{t(mc.unreachableText, { name: it.name, date: since, waiting: it.waiting })}</span>
            <span>
              {t(mc.unreachableReaderLabel)} {t(mc.unreachableReader, { name: it.name })}
            </span>
            {canConnect ? (
              <span>
                <Button size="sm" onClick={() => void check()}>
                  {t(mc.checkAgain)}
                </Button>
              </span>
            ) : null}
          </span>
        </Note>
      )
      break
    case 'ended':
      body = <Note tone="neutral">{t(mc.endedText)}</Note>
      break
  }
  return (
    <>
      {body}
      {pasting ? <AddDialog v={undefined} start="partner" onClose={() => setPasting(false)} /> : null}
      <RemoveDialog it={it} open={removing} onOpenChange={setRemoving} />
    </>
  )
}

async function resumeLink(slug: string, it: Item, refresh: () => Promise<void>, t: ReturnType<typeof useT>) {
  try {
    await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/resume', { params: { path: { workspace: slug, link: it.linkId } } })
    toast({ kind: 'success', title: t(mc.resumed, { name: it.name }) })
    await refresh()
  } catch (e) {
    toast({ kind: 'error', title: errorMessage(e) })
  }
}

function RemoveDialog({ it, open, onOpenChange }: { it: Item; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      badge={t(mc.remove)}
      title={t(mc.removeTitle, { name: it.name })}
      description={it.kind === 'partner' ? t(mc.removeTextPartner, { name: it.name }) : t(mc.removeText)}
      confirmLabel={it.kind === 'partner' ? t(mc.menuRemoveEnd) : t(mc.remove)}
      onConfirm={async () => {
        await http.DELETE('/api/v1/workspaces/{workspace}/grow/recommendations/{recommendation}', { params: { path: { workspace: slug, recommendation: it.id } } })
        toast({ title: t(mc.removed, { name: it.name }) })
        onOpenChange(false)
        await refresh()
      }}
    />
  )
}

/** Edit, Pause, Remove (and for partners: where it shows, the partner link, the log). */
function RowMenu({ it }: { it: Item }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const canConnect = useCan('grow:connect')
  const refresh = useGrowRefresh()
  const [dialog, setDialog] = useState<'edit' | 'remove' | 'log'>()
  if (!canEdit && !canConnect && it.kind !== 'partner') return null
  const pause = async () => {
    try {
      if (it.kind === 'link') {
        await http.POST('/api/v1/workspaces/{workspace}/grow/recommendations/{recommendation}/pause', { params: { path: { workspace: slug, recommendation: it.id } } })
      } else {
        await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/pause', { params: { path: { workspace: slug, link: it.linkId } } })
      }
      toast({ title: t(mc.paused, { name: it.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const paused = it.state === 'paused'
  return (
    <>
      <Menu
        label={t(mc.actionsFor, { name: it.name })}
        trigger={
          <button type="button" className="btn sm ic" aria-label={t(mc.actionsFor, { name: it.name })}>
            <Icon name="more" size={14} />
          </button>
        }
      >
        {canEdit ? <MenuItem onSelect={() => setDialog('edit')}>{it.kind === 'partner' ? t(mc.menuEditWhere) : t(mc.menuEdit)}</MenuItem> : null}
        {it.kind === 'link' && canEdit ? (
          <MenuItem onSelect={() => void (paused ? resumeLinkRec(slug, it, refresh, t) : pause())}>{paused ? t(mc.menuResume) : t(mc.menuPause)}</MenuItem>
        ) : null}
        {it.kind === 'partner' ? (
          <MenuItem disabled={!canConnect || it.state === 'invalid' || it.state === 'ended'} onSelect={() => void (paused ? resumeLink(slug, it, refresh, t) : pause())}>
            {paused ? t(mc.menuResume) : t(mc.menuPausePartner)}
          </MenuItem>
        ) : null}
        {it.kind === 'partner' ? <MenuItem onSelect={() => setDialog('log')}>{t(mc.menuRequests)}</MenuItem> : null}
        {(it.kind === 'link' && canEdit) || (it.kind === 'partner' && canConnect) ? (
          <>
            <MenuSeparator />
            <MenuItem className="danger" onSelect={() => setDialog('remove')}>
              {it.kind === 'partner' ? t(mc.menuRemoveEnd) : t(mc.menuRemove)}
            </MenuItem>
          </>
        ) : null}
        {it.kind === 'partner' && !canConnect ? (
          <span className="k block px-3 py-1.5">
            {t(mc.needsConnect)}
          </span>
        ) : null}
      </Menu>
      {dialog === 'edit' ? <EditDialog it={it} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'log' ? <LogDialog linkId={it.linkId} name={it.name} onClose={() => setDialog(undefined)} /> : null}
      <RemoveDialog it={it} open={dialog === 'remove'} onOpenChange={(o) => setDialog(o ? 'remove' : undefined)} />
    </>
  )
}

async function resumeLinkRec(slug: string, it: Item, refresh: () => Promise<void>, t: ReturnType<typeof useT>) {
  try {
    await http.POST('/api/v1/workspaces/{workspace}/grow/recommendations/{recommendation}/resume', { params: { path: { workspace: slug, recommendation: it.id } } })
    toast({ kind: 'success', title: t(mc.resumed, { name: it.name }) })
    await refresh()
  } catch (e) {
    toast({ kind: 'error', title: errorMessage(e) })
  }
}

/** Recommending you: partners on other instances that send you readers. */
function RecommendingYou({ v }: { v: View }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canConnect = useCan('grow:connect')
  const refresh = useGrowRefresh()
  const [limitFor, setLimitFor] = useState<You>()
  const [ending, setEnding] = useState<You>()
  const [logFor, setLogFor] = useState<You>()
  const [pausing, setPausing] = useState<You>()
  const list = v.profile.listName
  const revoke = async (id: string) => {
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/grow/partner-codes/{code}', { params: { path: { workspace: slug, code: id } } })
      toast({ title: t(mc.revoked) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const resume = async (y: You) => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/resume', { params: { path: { workspace: slug, link: y.linkId } } })
      toast({ kind: 'success', title: t(mc.resumedIn, { name: y.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const menu = (y: You) => (
    <Menu
      label={t(mc.actionsFor, { name: y.name })}
      trigger={
        <button type="button" className="btn sm ic" aria-label={t(mc.actionsFor, { name: y.name })}>
          <Icon name="more" size={14} />
        </button>
      }
    >
      <MenuItem disabled={!canConnect} onSelect={() => setLimitFor(y)}>
        {t(mc.menuChangeLimit)}
      </MenuItem>
      <MenuItem disabled={!canConnect || y.state === 'invalid'} onSelect={() => (y.state === 'paused' ? void resume(y) : setPausing(y))}>
        {y.state === 'paused' ? t(mc.menuResume) : t(mc.menuPause)}
      </MenuItem>
      <MenuItem onSelect={() => setLogFor(y)}>{t(mc.menuRequests)}</MenuItem>
      <MenuSeparator />
      <MenuItem className="danger" disabled={!canConnect} onSelect={() => setEnding(y)}>
        {t(mc.menuEnd)}
      </MenuItem>
    </Menu>
  )
  return (
    <section className="card">
      <CardSection
        title={t(mc.recommendingYou, { count: v.recommendingYou.length })}
        caption={list ? t(mc.recommendingYouCaption, { list }) : t(mc.recommendingYouCaptionNoList)}
      />
      {v.recommendingYou.length ? (
        <>
          <div className="max-md:hidden">
            <TableScroll label={t(mc.youTableLabel)}>
              <table className="tbl gw-tbl">
                <thead>
                  <tr>
                    <th scope="col">{t(mc.colPartner)}</th>
                    <th scope="col">{t(mc.colSince)}</th>
                    <th scope="col" className="r">
                      {t(mc.colRequestsOnly)}
                    </th>
                    <th scope="col" className="r">
                      {t(mc.colConfirmed)}
                    </th>
                    <th scope="col" className="r">
                      {t(mc.colRate)}
                    </th>
                    <th scope="col" className="r">
                      {t(mc.colLimit)}
                    </th>
                    <th scope="col">{t(mc.colStatus)}</th>
                    <th scope="col">
                      <span className="vh">{t(mc.colActions)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {v.recommendingYou.map((y) => {
                    const b = recBadge(y)
                    return (
                      <tr key={y.linkId}>
                        <td>
                          <span className="flex items-center gap-2.5">
                            <NewsletterTile name={y.name} />
                            <span className="flex min-w-0 flex-col gap-px">
                              <span className="gw-tt2">{y.name}</span>
                              <span className="k">
                                <b className="font-semibold text-ink-nav">{t(mc.kindPartner)}</b> · <span className="mono">{y.instance}</span>
                              </span>
                            </span>
                          </span>
                        </td>
                        <td className="nw">{dayMonthYear(y.since)}</td>
                        <td className="r gw-num">{count(y.requests)}</td>
                        <td className="r gw-num">{count(y.confirmed)}</td>
                        <td className="r gw-num">{y.requests ? pct((y.confirmed / y.requests) * 100, 0) : t(m.none)}</td>
                        <td className="r gw-num">
                          {count(y.dailyLimit)} <span className="k">{t(mc.perDay)}</span>
                        </td>
                        <td>
                          <Badge tone={b.tone} dot>
                            {b.label}
                          </Badge>
                        </td>
                        <td className="r">{menu(y)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="md:hidden">
            {v.recommendingYou.map((y) => {
              const b = recBadge(y)
              return (
                <div key={y.linkId} className="flex flex-col gap-2 border-t border-line-2 px-4 py-3">
                  <div className="flex items-start gap-2.5">
                    <NewsletterTile name={y.name} />
                    <span className="flex min-w-0 flex-1 flex-col gap-px">
                      <span className="gw-tt2">{y.name}</span>
                      <span className="k">
                        <b className="font-semibold text-ink-nav">{t(mc.kindPartner)}</b> · <span className="mono">{y.instance}</span>
                      </span>
                    </span>
                    {menu(y)}
                  </div>
                  <span className="flex flex-wrap items-center gap-2">
                    <Badge tone={b.tone} dot>
                      {b.label}
                    </Badge>
                    <span className="k">{t(mc.since, { date: dayMonthYear(y.since) })}</span>
                  </span>
                  <KeyValues
                    items={[
                      [t(mc.labelRequests), count(y.requests)],
                      [t(mc.labelConfirmed), t(mc.confirmedRate, { count: count(y.confirmed), pct: y.requests ? pct((y.confirmed / y.requests) * 100, 0) : '–' })],
                      [t(mc.colLimit), count(y.dailyLimit)],
                    ]}
                  />
                </div>
              )
            })}
          </div>
        </>
      ) : (
        <div className="gw-pad k">{t(mc.noneRecommendYou)}</div>
      )}
      {v.codes.length ? (
        <div className="flex flex-col gap-1.5 border-t border-line-2 px-[18px] py-3">
          <span className="gw-hd3">{t(mc.codesPending, { count: v.codes.length })}</span>
          {v.codes.map((c) => (
            <div key={c.id} className="flex flex-wrap items-center justify-between gap-2">
              <span className="k">{t(mc.codePendingLine, { label: c.label, list: c.listName, limit: c.dailyLimit, date: dayMonthDate(c.expiresAt) })}</span>
              {canConnect ? (
                <Button size="sm" variant="ghost" onClick={() => void revoke(c.id)}>
                  {t(mc.revoke)}
                </Button>
              ) : null}
            </div>
          ))}
        </div>
      ) : null}
      <div className="gw-foot">
        <span className="k">{t(mc.youPrivacy)}</span>
        {v.recommendingYou[0] ? (
          <button type="button" className="gw-lnk border-0 bg-transparent p-0 text-accent" onClick={() => setLogFor(v.recommendingYou[0])}>
            {t(mc.requestLog)}
          </button>
        ) : null}
      </div>
      {limitFor ? <LimitDialog y={limitFor} onClose={() => setLimitFor(undefined)} /> : null}
      {logFor ? <LogDialog linkId={logFor.linkId} name={logFor.name} onClose={() => setLogFor(undefined)} /> : null}
      {ending ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setEnding(undefined)}
          badge={t(mc.endAction)}
          title={t(mc.endTitle, { name: ending.name })}
          description={t(mc.endIntro, { name: ending.name, list: ending.listName || list, date: dayMonthYear(ending.since) })}
          confirmLabel={t(mc.endAction)}
          onConfirm={async () => {
            await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/end', { params: { path: { workspace: slug, link: ending.linkId } } })
            toast({ title: t(mc.ended, { name: ending.name }) })
            setEnding(undefined)
            await refresh()
          }}
        >
          <ul className="m-0 flex flex-col gap-1.5 pl-5 text-[13.5px]">
            <li>{t(mc.endStop, { name: ending.name, count: ending.confirmed, list: ending.listName || list })}</li>
            <li>{t(mc.endBoth, { name: ending.name, source: v.profile.name || list })}</li>
            <li>{t(mc.endNumbers)}</li>
          </ul>
          <p className="m-0 text-[13.5px]">{t(mc.endFinal, { name: ending.name })}</p>
          {v.items.some((it) => it.kind === 'partner' && it.name === ending.name) ? <Note tone="info">{t(mc.endSeparate, { name: ending.name })}</Note> : null}
          <span className="k">
            {t(mc.connectNeeds)}
          </span>
        </ConfirmDialog>
      ) : null}
      {pausing ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setPausing(undefined)}
          tone="warn"
          badge={t(mc.pauseInAction)}
          title={t(mc.pauseInTitle, { name: pausing.name })}
          description={t(mc.pauseInText, { name: pausing.name })}
          confirmLabel={t(mc.pauseInAction)}
          confirmVariant="primary"
          onConfirm={async () => {
            await http.POST('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/pause', { params: { path: { workspace: slug, link: pausing.linkId } } })
            toast({ title: t(mc.pausedIn, { name: pausing.name }) })
            setPausing(undefined)
            await refresh()
          }}
        />
      ) : null}
    </section>
  )
}

/** Where recommendations show: the confirmation page, welcome emails, the hosted page. */
function WhereCard({ v }: { v: View }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const refresh = useGrowRefresh()
  const [busy, setBusy] = useState(false)
  const flip = async (on: boolean) => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/recommendations/settings', { params: { path: { workspace: slug } }, body: { confirmPage: on } })
      toast({ title: on ? t(mc.confirmOn) : t(mc.confirmOff) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="card">
      <CardSection title={t(mc.whereTitle)} />
      <div className="gw-set">
        <div className="tx">
          <span className="font-semibold">{t(mc.whereConfirm)}</span>
          <span className="k">{t(mc.whereConfirmText, { count: v.confirmShows.length, names: formatList(v.confirmShows) })}</span>
        </div>
        <div className="ct">
          <input type="checkbox" className="tg" role="switch" aria-label={t(mc.whereConfirmSwitch)} checked={v.confirmPage} disabled={!canEdit || busy} onChange={(e) => void flip(e.target.checked)} />
        </div>
      </div>
      <div className="gw-set">
        <div className="tx">
          <span className="font-semibold">{t(mc.whereBlocks)}</span>
          {v.blocks.length ? (
            v.blocks.map((b) => (
              <span key={b.emailId} className="k">
                {t(mc.whereBlocksLine, { email: b.automation ? `${b.automation} · ${b.name}` : b.name, count: v.blockShows })}
              </span>
            ))
          ) : (
            <span className="k">{t(mc.whereBlocksNone)}</span>
          )}
        </div>
        {v.blocks[0]?.automationId ? (
          <div className="ct">
            <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: v.blocks[0].automationId }} className="btn sm">
              {t(mc.openInComposer)}
            </Link>
          </div>
        ) : null}
      </div>
      <div className="gw-set">
        <div className="tx">
          <span className="font-semibold">{t(mc.wherePage)}</span>
          <span className="k">
            <span className="mono">{v.pageUrl.replace(/^https?:\/\//, '')}</span> {t(mc.wherePageText)}
          </span>
        </div>
        <div className="ct">
          <PreviewButton page="recommends" label={t(mc.preview)} />
        </div>
      </div>
      <div className="gw-foot">
        <span className="k">{t(mc.whereFoot)}</span>
      </div>
    </section>
  )
}

/** Your partner profile: what partners see before they connect. */
function ProfileCard({ v }: { v: View }) {
  const t = useT()
  const canEdit = useCan('grow:edit')
  const [editing, setEditing] = useState(false)
  const p = v.profile
  return (
    <section className="card">
      <CardSection
        title={t(mc.profileTitle)}
        caption={t(mc.profileCaption)}
        actions={
          canEdit ? (
            <Button size="sm" onClick={() => setEditing(true)}>
              {t(mc.menuEdit)}
            </Button>
          ) : null
        }
      />
      <div className="gw-pad flex flex-col gap-3">
        <div className="flex items-center gap-3">
          <NewsletterTile name={p.name || '?'} initials={p.initials} logo={p.logo?.url} size={40} me />
          <span className="flex min-w-0 flex-col gap-px">
            <span className="gw-tt2">{p.name}</span>
            {p.description ? <span className="k">{p.description}</span> : null}
          </span>
        </div>
        <KeyValues
          items={[
            [t(mc.profileList), p.listName ? t(mc.profileListLine, { list: p.listName, count: p.listSubscribers }) : t(mc.profileNoList)],
            [t(mc.profileDoi), t(mc.profileDoiValue)],
            [t(mc.profileInstance), <span key="i" className="mono">{p.instance}</span>],
          ]}
        />
      </div>
      {editing ? <ProfileDialog v={v} onClose={() => setEditing(false)} /> : null}
    </section>
  )
}

function ProfileDialog({ v, onClose }: { v: View; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const p = v.profile
  const [name, setName] = useState(p.name)
  const [description, setDescription] = useState(p.description)
  const [listId, setListId] = useState(p.listId || v.lists[0]?.id || '')
  const [logo, setLogo] = useState(p.logo)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/recommendations/profile', {
        params: { path: { workspace: slug } },
        body: { name: name.trim(), description: description.trim(), listId, logoId: logo?.id },
      })
      toast({ kind: 'success', title: t(mc.profileSaved), detail: t(mc.profileSavedDetail) })
      await refresh()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mc.profileEdit)}
      description={t(mc.profileCaption)}
      footer={
        <>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            {t(mc.saveAction)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
      <Field label={t(mc.profileName)}>
        <Input value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
      </Field>
      <Field label={t(mc.profileDescription)} hint={t(mc.profileDescriptionHint, { count: description.length })}>
        <Textarea rows={2} value={description} maxLength={120} onChange={(e) => setDescription(e.target.value)} />
      </Field>
      <LogoField logo={logo} onChange={setLogo} />
      <Field label={t(mc.profileList)}>
        <Select value={listId} onChange={(e) => setListId(e.target.value)}>
          {v.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {t(mc.codeListOption, { list: l.name, count: l.subscribers })}
            </option>
          ))}
        </Select>
      </Field>
    </Dialog>
  )
}

function LogoField({ logo, onChange }: { logo?: Schemas['RecImage']; onChange: (l?: Schemas['RecImage']) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [busy, setBusy] = useState(false)
  const id = useId()
  const upload = async (file: File) => {
    setBusy(true)
    try {
      const a = await uploadFile<Schemas['MediaAsset']>(`/api/v1/workspaces/${encodeURIComponent(slug)}/media`, file)
      onChange({ id: a.id, name: a.fileName, url: a.url })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="flex flex-col gap-1">
      <span className="lbl">{t(mc.linkLogo)}</span>
      <div className="flex flex-wrap items-center gap-3 rounded-lg border border-line px-3 py-2">
        <span className="gw-mtile" aria-hidden="true">
          {logo ? <img src={logo.url} alt="" /> : null}
        </span>
        <span className="k flex-1">{logo ? logo.name : t(mc.logoNone)}</span>
        <input id={id} type="file" accept="image/png,image/jpeg,image/gif" className="vh" onChange={(e) => e.target.files?.[0] && void upload(e.target.files[0])} />
        <label htmlFor={id} className="btn sm" aria-busy={busy || undefined}>
          {logo ? t(mc.logoReplace) : t(mc.logoUpload)}
        </label>
        {logo ? (
          <Button size="sm" onClick={() => onChange(undefined)}>
            {t(mc.logoRemove)}
          </Button>
        ) : null}
      </div>
    </div>
  )
}

type AddStep = 'choose' | 'link' | 'partner' | 'checking' | 'confirm'

/** Add a recommendation: a link, or a partner on ReputeCast through its code. */
function AddDialog({ v, onClose, start = 'choose' }: { v?: View; onClose: () => void; start?: AddStep }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canConnect = useCan('grow:connect')
  const refresh = useGrowRefresh()
  const [step, setStep] = useState<AddStep>(start)
  const [kind, setKind] = useState<'partner' | 'link'>(canConnect ? 'partner' : 'link')
  // Link.
  const [url, setUrl] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [logo, setLogo] = useState<Schemas['RecImage']>()
  const [showConfirm, setShowConfirm] = useState(true)
  const [showBlock, setShowBlock] = useState(true)
  const [prefill, setPrefill] = useState<'idle' | 'busy' | 'done' | 'none'>('idle')
  const [error, setError] = useState<string>()
  // Partner.
  const [code, setCode] = useState('')
  const [preview, setPreview] = useState<Schemas['PartnerPreview']>()
  const [problem, setProblem] = useState<{ code: string; message: string }>()
  const [busy, setBusy] = useState(false)
  const showing = (v?.items ?? []).filter((x) => x.showConfirm && (x.state === 'showing' || x.state === 'connected')).length
  const blockEmail = v?.blocks[0] ? (v.blocks[0].automation ? `${v.blocks[0].automation} · ${v.blocks[0].name}` : v.blocks[0].name) : ''

  const doPrefill = async () => {
    if (!/^https:\/\//.test(url.trim())) return
    setPrefill('busy')
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/recommendations/prefill', { params: { path: { workspace: slug } }, body: { url: url.trim() } })
      if (data?.found) {
        if (!name) setName(data.name)
        if (!description) setDescription(data.description)
        if (data.logo) setLogo(data.logo)
        if (data.url) setUrl(data.url)
        setPrefill('done')
      } else setPrefill('none')
    } catch (e) {
      setPrefill('idle')
      setError(errorMessage(e))
    }
  }
  const addLink = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/recommendations', {
        params: { path: { workspace: slug } },
        body: { url: url.trim(), name: name.trim(), description: description.trim(), logoId: logo?.id, showConfirm, showBlock },
      })
      toast({ kind: 'success', title: t(mc.added, { name: name.trim() }), detail: t(mc.addedDetail) })
      await refresh()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const check = async () => {
    setProblem(undefined)
    setStep('checking')
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/partners/preview', { params: { path: { workspace: slug } }, body: { code: code.trim() } })
      setPreview(data)
      setStep('confirm')
    } catch (e) {
      setProblem({ code: e instanceof ApiError ? (e.code ?? '') : '', message: errorMessage(e) })
      setStep('partner')
    }
  }
  const connect = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/partners/connect', { params: { path: { workspace: slug } }, body: { code: code.trim(), showConfirm, showBlock } })
      toast({ kind: 'success', title: t(mc.connected, { name: preview?.name ?? '', source: v?.profile.name ?? '' }), detail: t(mc.connectedDetail) })
      await refresh()
      onClose()
    } catch (e) {
      setProblem({ code: e instanceof ApiError ? (e.code ?? '') : '', message: errorMessage(e) })
      setStep('partner')
    } finally {
      setBusy(false)
    }
  }
  const host = (() => {
    try {
      return new URL(code.trim()).host
    } catch {
      return ''
    }
  })()
  const where = (
    <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
      <legend className="lbl mb-1">{t(mc.whereShows)}</legend>
      <label className="inline-flex items-center gap-2 text-[13.5px]">
        <input type="checkbox" checked={showConfirm} onChange={(e) => setShowConfirm(e.target.checked)} />
        <span>
          {t(mc.showConfirm)} <span className="k">{showConfirm ? t(mc.showConfirmPlace, { place: showing + 1 }) : t(mc.showConfirmOff)}</span>
        </span>
      </label>
      <label className="inline-flex items-center gap-2 text-[13.5px]">
        <input type="checkbox" checked={showBlock} onChange={(e) => setShowBlock(e.target.checked)} />
        <span>
          {t(mc.showBlock)} {blockEmail ? <span className="k">{t(mc.showBlockWhere, { email: blockEmail })}</span> : null}
        </span>
      </label>
    </fieldset>
  )
  if (step === 'choose') {
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{t(mc.addEyebrow)}</span>}
        title={t(mc.addTitle)}
        footer={
          <>
            <Button onClick={onClose}>{t(mc.cancel)}</Button>
            <Button variant="primary" onClick={() => setStep(kind)}>
              {t(mc.continue)}
            </Button>
          </>
        }
      >
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1.5">{t(mc.addQuestion)}</legend>
          <OptionCard name="rec-kind" value="partner" checked={kind === 'partner'} disabled={!canConnect} onChange={() => setKind('partner')} title={t(mc.addPartner)} description={t(mc.addPartnerText)}>
            {!canConnect ? (
              <span className="k">
                {t(v?.helpers.connect.length ? mc.editorNoConnect : mc.editorNoConnectNoHelpers, { helpers: orNames(v?.helpers.connect) })}
              </span>
            ) : null}
          </OptionCard>
          <OptionCard name="rec-kind" value="link" checked={kind === 'link'} onChange={() => setKind('link')} title={t(mc.addLink)} description={t(mc.addLinkText)} />
        </fieldset>
      </Dialog>
    )
  }
  if (step === 'link') {
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{t(mc.linkEyebrow)}</span>}
        title={t(mc.linkTitle)}
        footer={
          <>
            <span className="k mr-auto">{t(mc.auditNote)}</span>
            {start === 'choose' ? <Button onClick={() => setStep('choose')}>{t(mc.back)}</Button> : null}
            <Button variant="primary" loading={busy} disabled={!name.trim() || !/^https:\/\//.test(url.trim()) || (!showConfirm && !showBlock)} onClick={() => void addLink()}>
              {t(mc.addAction)}
            </Button>
          </>
        }
      >
        {error ? <Note tone="bad">{error}</Note> : null}
        <Field label={t(mc.linkUrl)} hint={prefill === 'done' ? t(mc.prefilled) : prefill === 'none' ? t(mc.prefillNone) : t(mc.linkUrlHint)}>
          <span className="flex gap-2">
            <Input type="url" className="mono flex-1" aria-label={t(mc.linkUrlLabel)} placeholder={'https://'} value={url} onChange={(e) => setUrl(e.target.value)} onBlur={() => prefill === 'idle' && void doPrefill()} />
            <Button size="sm" loading={prefill === 'busy'} busyLabel={t(mc.prefilling)} disabled={!/^https:\/\//.test(url.trim())} onClick={() => void doPrefill()}>
              {t(mc.prefill)}
            </Button>
          </span>
        </Field>
        <Field label={t(mc.linkName)}>
          <Input value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
        </Field>
        <LogoField logo={logo} onChange={setLogo} />
        <Field label={t(mc.linkDescription)} hint={t(mc.descriptionCount, { count: description.length })}>
          <Textarea rows={2} value={description} maxLength={120} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        {where}
        <span className="k">{t(mc.ssrfNote)}</span>
      </Dialog>
    )
  }
  if (step === 'partner' || step === 'checking') {
    const title = problem ? errTitle(t, problem.code) : t(mc.partnerTitle)
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{t(mc.partnerEyebrow)}</span>}
        title={t(mc.partnerTitle)}
        description={t(mc.partnerText)}
        footer={
          <>
            {start === 'choose' ? (
              <Button onClick={() => setStep('choose')} disabled={step === 'checking'}>
                {t(mc.back)}
              </Button>
            ) : (
              <Button onClick={onClose}>{t(mc.cancel)}</Button>
            )}
            <Button variant="primary" loading={step === 'checking'} busyLabel={t(mc.checking)} disabled={!code.trim()} onClick={() => void check()}>
              {problem?.code === 'partner_unreachable' ? t(mc.tryAgain) : t(mc.continue)}
            </Button>
          </>
        }
      >
        <Field label={t(mc.partnerCode)} error={problem ? problem.message : undefined}>
          <Input className="mono" placeholder={'https://reputecast.example/p/XXXX-XXXX-XXXX'} value={code} onChange={(e) => setCode(e.target.value)} />
        </Field>
        {step === 'checking' ? (
          <span className="k flex items-center gap-2">
            <span className="spin sm" aria-hidden="true" />
            {host ? <T msg={mc.checkingLine} values={{ instance: host, b: (c) => <b>{c}</b> }} /> : t(mc.checkingLineNoHost)}
          </span>
        ) : null}
        {problem ? (
          <Note tone="bad" label={title}>
            <span className="flex flex-col gap-1.5">
              <span>{problem.message}</span>
              {problem.code === 'not_a_code' || problem.code === 'partner_unsupported' ? (
                <span>
                  <Button
                    size="sm"
                    onClick={() => {
                      setUrl(code.trim())
                      setProblem(undefined)
                      setStep('link')
                    }}
                  >
                    {problem.code === 'not_a_code' ? t(mc.addAsLink) : t(mc.addPageAsLink)}
                  </Button>
                </span>
              ) : null}
            </span>
          </Note>
        ) : null}
      </Dialog>
    )
  }
  // Confirm the partner's profile.
  const p = preview!
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mc.partnerEyebrow)}</span>}
      title={t(mc.connectTitle, { name: p.name })}
      description={t(mc.connectText, { name: p.name })}
      footer={
        <>
          <span className="k mr-auto">
            {t(mc.connectNeeds)}
          </span>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!showConfirm && !showBlock} onClick={() => void connect()}>
            {t(mc.connect)}
          </Button>
        </>
      }
    >
      <div className="flex items-start gap-3 rounded-[10px] border border-line px-3.5 py-3">
        <NewsletterTile name={p.name} initials={p.initials} logo={p.logoUrl} size={40} />
        <span className="flex min-w-0 flex-col gap-px">
          <span className="gw-tt2">{p.name}</span>
          {p.description ? <span className="k">{p.description}</span> : null}
        </span>
      </div>
      <KeyValues
        items={[
          [t(mc.connectList), p.list],
          [t(mc.connectDoi), t(mc.connectDoiAlways)],
          [t(mc.connectLimit), t(mc.connectLimitValue, { count: p.dailyLimit })],
          [t(mc.connectInstance), <span key="i" className="mono">{p.instance}</span>],
        ]}
      />
      {where}
      <Note tone="info">{t(mc.connectSecret, { instance: p.instance, source: v?.profile.name ?? '' })}</Note>
    </Dialog>
  )
}

function errTitle(t: ReturnType<typeof useT>, code: string): string {
  switch (code) {
    case 'code_expired':
      return t(mc.errExpired)
    case 'code_used':
      return t(mc.errUsed)
    case 'not_a_code':
      return t(mc.errNotCode)
    case 'partner_unreachable':
      return t(mc.errUnreachable)
    case 'partner_unsupported':
      return t(mc.errUnsupported)
    case 'partner_private':
      return t(mc.errPrivate)
    case 'code_revoked':
      return t(mc.errRevoked)
    case 'code_unknown':
      return t(mc.errUnknown)
    case 'no_public_url':
      return t(mc.errNoInstance)
  }
  return t(mc.cantConnect)
}

/** Edit a link (name, logo, description, where), or a partner's where it shows. */
function EditDialog({ it, onClose }: { it: Item; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [url, setUrl] = useState(it.url)
  const [name, setName] = useState(it.name)
  const [description, setDescription] = useState(it.description)
  const [logo, setLogo] = useState(it.logo)
  const [showConfirm, setShowConfirm] = useState(it.showConfirm)
  const [showBlock, setShowBlock] = useState(it.showBlock)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async () => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/recommendations/{recommendation}', {
        params: { path: { workspace: slug, recommendation: it.id } },
        body: { url: url.trim(), name: name.trim(), description: description.trim(), logoId: logo?.id, showConfirm, showBlock },
      })
      toast({ kind: 'success', title: t(mc.saved, { name: name.trim() }) })
      await refresh()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const partner = it.kind === 'partner'
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mc.editEyebrow)}</span>}
      title={t(mc.editTitle, { name: it.name })}
      footer={
        <>
          <span className="k mr-auto">{t(mc.auditNote)}</span>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!name.trim() || (!showConfirm && !showBlock)} onClick={() => void save()}>
            {t(mc.saveAction)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
      {partner ? null : (
        <>
          <Field label={t(mc.linkUrl)}>
            <Input type="url" className="mono" value={url} onChange={(e) => setUrl(e.target.value)} />
          </Field>
          <Field label={t(mc.linkName)}>
            <Input value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
          </Field>
          <LogoField logo={logo} onChange={setLogo} />
          <Field label={t(mc.linkDescription)} hint={t(mc.descriptionCount, { count: description.length })}>
            <Textarea rows={2} value={description} maxLength={120} onChange={(e) => setDescription(e.target.value)} />
          </Field>
        </>
      )}
      <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
        <legend className="lbl mb-1">{t(mc.whereShows)}</legend>
        <label className="inline-flex items-center gap-2 text-[13.5px]">
          <input type="checkbox" checked={showConfirm} onChange={(e) => setShowConfirm(e.target.checked)} />
          {t(mc.showConfirm)}
        </label>
        <label className="inline-flex items-center gap-2 text-[13.5px]">
          <input type="checkbox" checked={showBlock} onChange={(e) => setShowBlock(e.target.checked)} />
          {t(mc.showBlock)}
        </label>
      </fieldset>
    </Dialog>
  )
}

/** "Let a newsletter recommend you": create a partner code, then show it once. */
function CodeDialog({ v, onClose }: { v: View; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [label, setLabel] = useState('')
  const [listId, setListId] = useState(v.profile.listId || v.lists[0]?.id || '')
  const [limit, setLimit] = useState('50')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [created, setCreated] = useState<Schemas['PartnerCodeCreated']>()
  const create = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/partner-codes', {
        params: { path: { workspace: slug } },
        body: { label: label.trim(), listId, dailyLimit: Number(limit) || 50 },
      })
      setCreated(data)
      void refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  if (created) {
    return (
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        eyebrow={<span className="k">{t(mc.codeCreated, { time: clock(created.createdAt) })}</span>}
        title={t(mc.codeSend, { name: created.label })}
        description={t(mc.codeOnce)}
        footer={
          <Button
            variant="primary"
            onClick={() => {
              void copy(created.url)
              onClose()
            }}
          >
            {t(mc.codeCopiedClose)}
          </Button>
        }
      >
        <div className="gw-link">
          <Input className="mono" value={created.url} readOnly aria-label={t(mc.partnerCode)} />
          <Button icon="copy" onClick={() => void copy(created.url)}>
            {t(mc.codeCopy)}
          </Button>
        </div>
        <KeyValues
          items={[
            [t(mc.codeFor), created.label],
            [t(mc.codeList), created.listName],
            [t(mc.codeLimit), t(mc.connectLimitValue, { count: created.dailyLimit })],
            [t(mc.codeValid), t(mc.codeValidValue, { date: `${dayMonthDate(created.expiresAt)}, ${clock(created.expiresAt)}` })],
          ]}
        />
        <Note tone="warn" label={t(mc.codeShownOnce)}>
          {t(mc.codeLeak)}
        </Note>
      </Dialog>
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(mc.addEyebrow)}</span>}
      title={t(mc.codeTitle)}
      description={t(mc.codeText)}
      footer={
        <>
          <span className="k mr-auto">
            {t(mc.needsConnect)}
          </span>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!label.trim() || !listId} onClick={() => void create()}>
            {t(mc.codeCreate)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
      <Field label={t(mc.codeFor)} hint={t(mc.codeForHint)}>
        <Input value={label} maxLength={120} onChange={(e) => setLabel(e.target.value)} />
      </Field>
      <Field label={t(mc.codeList)}>
        <Select value={listId} onChange={(e) => setListId(e.target.value)}>
          {v.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {t(mc.codeListOption, { list: l.name, count: l.subscribers })}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t(mc.codeLimit)}>
        <span className="flex items-center gap-2">
          <Input type="number" min={1} max={10000} className="w-28" value={limit} onChange={(e) => setLimit(e.target.value)} />
          <span className="k">{t(mc.codeLimitUnit)}</span>
        </span>
      </Field>
      <span className="k">{label.trim() ? t(mc.codeNote, { name: label.trim() }) : t(mc.codeNoteNoName)}</span>
    </Dialog>
  )
}

function LimitDialog({ y, onClose }: { y: You; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [limit, setLimit] = useState(String(y.dailyLimit))
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/partner-links/{link}/limit', { params: { path: { workspace: slug, link: y.linkId } }, body: { dailyLimit: Number(limit) } })
      toast({ kind: 'success', title: t(mc.limitSaved, { name: y.name, count: Number(limit) }) })
      await refresh()
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={t(mc.limitTitle, { name: y.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!(Number(limit) >= 1)} onClick={() => void save()}>
            {t(mc.saveAction)}
          </Button>
        </>
      }
    >
      <Field label={t(mc.limitLabel)} hint={t(mc.limitHint, { name: y.name })}>
        <Input type="number" min={1} max={10000} value={limit} onChange={(e) => setLimit(e.target.value)} />
      </Field>
    </Dialog>
  )
}

function LogDialog({ linkId, name, onClose }: { linkId: string; name: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/partner-links/{link}/requests`, { params: { path: { workspace: slug, link: linkId } } })
  const d = q.data
  const status = (s: Schemas['PartnerRequestItem']['status']) =>
    ({ queued: t(mc.reqQueued), accepted: t(mc.reqAccepted), limit: t(mc.reqLimit), failed: t(mc.reqFailed), rejected: t(mc.reqRejected), paused: t(mc.reqPaused) })[s] ?? s
  const origin = (o: Schemas['PartnerRequestItem']['origin']) => ({ confirm: t(mc.originConfirm), page: t(mc.originPage), partner: t(mc.originPartner) })[o] ?? o
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="lg"
      title={t(mc.logTitle, { name })}
      description={d ? t(mc.logCaption, { direction: d.direction }) : undefined}
      footer={<Button onClick={onClose}>{t(m.close)}</Button>}
    >
      {!d ? (
        q.isError ? (
          <LoadError error={q.error} title={t(mc.loadError)} onRetry={() => void q.refetch()} />
        ) : (
          <Skeleton h={240} r={10} />
        )
      ) : d.items.length ? (
        <TableScroll label={t(mc.logTitle, { name })}>
          <table className="tbl gw-tbl">
            <thead>
              <tr>
                <th scope="col">{t(mc.colAt)}</th>
                <th scope="col">{t(mc.colOrigin)}</th>
                <th scope="col">{t(mc.colStatus)}</th>
                <th scope="col" className="r">
                  {t(mc.colAttempts)}
                </th>
              </tr>
            </thead>
            <tbody>
              {d.items.map((x) => (
                <tr key={x.id}>
                  <td className="nw">
                    {dayMonthDate(x.at)}, {clock(x.at)}
                  </td>
                  <td>{origin(x.origin)}</td>
                  <td>
                    {status(x.status)}
                    {x.confirmedAt ? <div className="k">{t(mc.reqConfirmed, { date: dayMonthDate(x.confirmedAt) })}</div> : null}
                    {x.lastError ? <div className="k">{x.lastError}</div> : null}
                  </td>
                  <td className="r gw-num">{x.attempts}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      ) : (
        <span className="k">{t(mc.logEmpty)}</span>
      )}
    </Dialog>
  )
}
