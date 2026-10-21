import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { deniedReason, useCan, useMe, useWorkspace } from '@/lib/session'
import { MilestoneTrack } from './MilestoneTrack'
import { MilestoneBadges } from './ResultsPage'
import { m } from './messages'
import { mf } from './referrers.messages'
import { GROW, GrowPage, LoadError, ProgramNav, copy, fileKind, fileSize, mono, orNames, useGrowRefresh, useReferralPrograms } from './shared'

type Row = Schemas['ReferralReferrerRow']
type Panel = Schemas['ReferralReferrerPanel']

export interface ReferrersSearch {
  program?: string
  q?: string
  milestone?: number
  held?: boolean
  paused?: boolean
  page?: number
  referrer?: string
}

/**
 * Referrals › Referrers (ReferralReferrers, ReferralStates): search and
 * filters, the referrers table, and a referrer's panel (?referrer=<link>).
 */
export function ReferrersPage({ search, onSearch }: { search: ReferrersSearch; onSearch: (s: ReferrersSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const { list, program, q: pq } = useReferralPrograms(search.program)
  const [text, setText] = useState(search.q ?? '')
  // Typing settles before the list asks again.
  useEffect(() => {
    const h = window.setTimeout(() => {
      if ((search.q ?? '') !== text.trim()) onSearch({ ...search, q: text.trim() || undefined, page: undefined })
    }, 300)
    return () => window.clearTimeout(h)
  }, [text]) // eslint-disable-line react-hooks/exhaustive-deps
  const q = $api.useQuery(
    'get',
    `${GROW}/referrals/{program}/referrers`,
    {
      params: {
        path: { workspace: slug, program: program?.id ?? '' },
        query: { q: search.q, milestone: search.milestone, held: search.held, paused: search.paused, page: search.page },
      },
    },
    { enabled: Boolean(program), placeholderData: (prev) => prev },
  )
  const d = q.data
  const err = pq.error ?? q.error
  const filtered = Boolean(search.q || search.milestone || search.held || search.paused)
  return (
    <GrowPage tab="referrals" documentTitle={t(m.docReferrals)}>
      <div className="gw">
        <ProgramNav current="referrers" programId={search.program} />
        {!list || (program && !d) ? (
          err ? (
            <LoadError error={err} title={t(mf.loadError)} onRetry={() => void (pq.isError ? pq.refetch() : q.refetch())} />
          ) : (
            <Loading label={t(mf.loadingLabel)}>
              <Skeleton w="100%" h={420} r={10} />
            </Loading>
          )
        ) : !program || !d ? null : (
          <section className="card">
            <Filters d={d} search={search} text={text} setText={setText} onSearch={onSearch} programId={program.id} />
            {d.items.length ? (
              <Table d={d} search={search} onOpen={(link) => onSearch({ ...search, referrer: link }, false)} />
            ) : (
              <Empty
                q={search.q}
                filtered={filtered}
                list={program.listName}
                onClear={() => {
                  setText('')
                  onSearch({ program: search.program })
                }}
              />
            )}
            <div className="gw-foot">
              <span className="k">{t(mf.foot, { count: d.withCounted })}</span>
              <Pager d={d} onPage={(p) => onSearch({ ...search, page: p > 1 ? p : undefined })} />
            </div>
          </section>
        )}
      </div>
      {program && search.referrer ? (
        <ReferrerPanel programId={program.id} link={search.referrer} listName={program.listName} onClose={() => onSearch({ ...search, referrer: undefined }, false)} />
      ) : null}
    </GrowPage>
  )
}

function Filters({
  d,
  search,
  text,
  setText,
  onSearch,
  programId,
}: {
  d: Schemas['ReferralReferrerList']
  search: ReferrersSearch
  text: string
  setText: (v: string) => void
  onSearch: (s: ReferrersSearch) => void
  programId: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const canExport = useCan('subscribers:export')
  const exportCsv = async () => {
    try {
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/grow/referrals/${programId}/referrers.csv`, { credentials: 'same-origin' })
      if (!r.ok) throw new Error(r.statusText)
      const a = document.createElement('a')
      a.href = URL.createObjectURL(await r.blob())
      a.download = 'referrers.csv'
      a.click()
      URL.revokeObjectURL(a.href)
      toast({ kind: 'success', title: t(mf.exported) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const chip = (on: boolean, label: string, n: number, toggle: () => void) => (
    <button type="button" className={on ? 'fchip on' : 'fchip'} aria-pressed={on} onClick={toggle}>
      {label} <b>{count(n)}</b>
    </button>
  )
  return (
    <div className="filters flex flex-wrap items-center gap-2.5 border-b border-line-2 px-[18px] py-3.5" role="search" aria-label={t(mf.filtersLabel)}>
      <span className="relative flex min-w-[220px] flex-[1_1_260px]">
        <Input
          type="search"
          aria-label={t(mf.search)}
          placeholder={t(mf.searchPlaceholder)}
          value={text}
          onChange={(e) => setText(e.target.value)}
          className="w-full"
        />
      </span>
      <Select
        aria-label={t(mf.milestoneLabel)}
        style={{ width: 240 }}
        value={String(search.milestone ?? 0)}
        onChange={(e) => onSearch({ ...search, milestone: Number(e.target.value) || undefined, page: undefined })}
      >
        <option value="0">{t(mf.anyMilestone)}</option>
        {d.milestoneCounts.map((x) => (
          <option key={x.count} value={x.count}>
            {t(mf.reached, { count: x.count, referrers: count(x.referrers) })}
          </option>
        ))}
        <option value="-1">{t(mf.noMilestone)}</option>
      </Select>
      {chip(Boolean(search.held), t(mf.hasHeld), d.heldReferrers, () => onSearch({ ...search, held: search.held ? undefined : true, page: undefined }))}
      {chip(Boolean(search.paused), t(mf.linkPaused), d.pausedLinks, () => onSearch({ ...search, paused: search.paused ? undefined : true, page: undefined }))}
      <Button size="sm" icon="download" className="ml-auto" disabledReason={canExport ? undefined : deniedReason('subscribers:export')} onClick={() => void exportCsv()}>
        {t(mf.exportCsv)}
      </Button>
    </div>
  )
}

function Table({ d, search, onOpen }: { d: Schemas['ReferralReferrerList']; search: ReferrersSearch; onOpen: (link: string) => void }) {
  const t = useT()
  const dash = <span className="k">{t(m.none)}</span>
  const num = (v: number) => (v ? count(v) : dash)
  return (
    <>
      <div className="max-md:hidden">
        <TableScroll label={t(mf.tableLabel)}>
          <table className="tbl gw-tbl">
            <thead>
              <tr>
                <th scope="col">{t(mf.colReferrer)}</th>
                <th scope="col" className="r">
                  {t(mf.colCounted)}
                </th>
                <th scope="col" className="r">
                  {t(mf.colInHold)}
                </th>
                <th scope="col" className="r">
                  {t(mf.colHeld)}
                </th>
                <th scope="col" className="r">
                  {t(mf.colNotCounted)}
                </th>
                <th scope="col">{t(mf.colMilestones)}</th>
                <th scope="col">{t(mf.colRewards)}</th>
                <th scope="col">{t(mf.colLink)}</th>
                <th scope="col">{t(mf.colLast)}</th>
              </tr>
            </thead>
            <tbody>
              {d.items.map((x) => (
                <tr key={x.linkId} className={search.referrer === x.linkId ? 'sel' : undefined}>
                  <td>
                    <button type="button" className="flex min-w-0 items-center gap-2.5 border-0 bg-transparent p-0 text-left text-ink" onClick={() => onOpen(x.linkId)}>
                      <Avatar name={x.name} size={28} color="muted" />
                      <span className="flex min-w-0 flex-col leading-[1.3]">
                        <span className="font-semibold">{x.name}</span>
                        {d.showAddresses && x.email ? <span className="k mono whitespace-nowrap">{x.email}</span> : null}
                      </span>
                    </button>
                  </td>
                  <td className="r gw-num">{num(x.counted)}</td>
                  <td className="r gw-num">{num(x.inHold)}</td>
                  <td className="r gw-num">{x.held ? <span className="gw-held">{count(x.held)}</span> : dash}</td>
                  <td className="r gw-num">{num(x.notCounted)}</td>
                  <td>
                    <MilestoneBadges list={x.milestones} />
                  </td>
                  <td>
                    {x.rewardsSent ? t(mf.rewardsSent, { count: count(x.rewardsSent) }) : dash}
                    {x.unclaimed ? <div className="k mt-[3px]">{t(mf.unclaimed, { reward: x.unclaimed })}</div> : null}
                  </td>
                  <td>
                    <Badge tone={x.linkStatus === 'paused' ? 'warn' : 'ok'} dot>
                      {x.linkStatus === 'paused' ? t(mf.linkPausedBadge) : t(mf.linkActive)}
                    </Badge>
                  </td>
                  <td className="nw">{x.lastReferral ? dayMonthDate(x.lastReferral) : dash}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="md:hidden">
        {d.items.map((x) => (
          <button key={x.linkId} type="button" className="gw-prow w-full border-x-0 border-b-0 bg-transparent text-left" onClick={() => onOpen(x.linkId)}>
            <Avatar name={x.name} size={28} color="muted" />
            <span className="tx">
              <span className="font-semibold">{x.name}</span>
              <span className="k">{phoneLine(t, x)}</span>
            </span>
            <MilestoneBadges list={x.milestones} />
          </button>
        ))}
      </div>
    </>
  )
}

function phoneLine(t: ReturnType<typeof useT>, x: Row) {
  const parts = [
    <span key="c">{t(mf.phoneLine, { counted: count(x.counted) })}</span>,
    x.inHold ? <span key="h">{t(mf.phoneHold, { count: count(x.inHold) })}</span> : null,
    x.held ? (
      <span key="r" className="gw-held">
        {t(mf.phoneHeld, { count: count(x.held) })}
      </span>
    ) : null,
    x.lastReferral ? <span key="l">{t(mf.phoneLast, { date: dayMonthDate(x.lastReferral) })}</span> : null,
  ].filter(Boolean)
  return parts.map((p, i) => (
    <span key={i}>
      {i ? ' · ' : ''}
      {p}
    </span>
  ))
}

function Empty({ q, filtered, list, onClear }: { q?: string; filtered: boolean; list: string; onClear: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="flex flex-col items-center gap-3 px-7 py-11 text-center">
      <span className="text-[17px] font-bold">{q ? t(mf.emptySearch, { q }) : filtered ? t(mf.emptyFilter) : t(mf.emptyNone)}</span>
      {filtered ? (
        <>
          <span className="max-w-[560px] text-ink-2">{t(mf.emptyText, { list })}</span>
          <div className="flex flex-wrap justify-center gap-2">
            <Button onClick={onClear}>{q ? t(mf.clearSearch) : t(mf.clearFilters)}</Button>
            {q ? (
              <ButtonLink to="/w/$workspace/audience" params={{ workspace: slug }} search={{ q } as never}>
                {t(mf.searchContacts)}
              </ButtonLink>
            ) : null}
          </div>
        </>
      ) : null}
    </div>
  )
}

function Pager({ d, onPage }: { d: Schemas['ReferralReferrerList']; onPage: (p: number) => void }) {
  const t = useT()
  const size = 25
  const pages = Math.max(1, Math.ceil(d.total / size))
  if (pages <= 1) return null
  return (
    <span className="flex items-center gap-2">
      <span className="k">{t(mf.pageOf, { page: d.page, pages })}</span>
      <Button size="sm" disabled={d.page <= 1} onClick={() => onPage(d.page - 1)}>
        {t(mf.previous)}
      </Button>
      <Button size="sm" disabled={d.page >= pages} onClick={() => onPage(d.page + 1)}>
        {t(mf.next)}
      </Button>
    </span>
  )
}

/** A referrer's panel: their link, progress, referrals and rewards. */
export function ReferrerPanel({ programId, link, listName, onClose }: { programId: string; link: string; listName: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/referrals/{program}/referrers/{link}`, { params: { path: { workspace: slug, program: programId, link } } })
  const p = q.data
  return (
    <RDialog.Root open onOpenChange={(o) => !o && onClose()}>
      <RDialog.Portal>
        <RDialog.Overlay className="gw-sheet-ovl" />
        <RDialog.Content className="gw-sheet" aria-describedby={undefined}>
          {!p ? (
            <div className="gw-sheet-body">
              <RDialog.Title className="vh">{q.isError ? t(mf.panelError) : t(mf.panelLoading)}</RDialog.Title>
              {q.isError ? (
                <LoadError error={q.error} title={t(mf.panelError)} onRetry={() => void q.refetch()} />
              ) : (
                <div className="flex flex-col gap-3" aria-busy="true">
                  <Skeleton w="60%" h={28} />
                  <Skeleton w="100%" h={80} />
                  <Skeleton w="100%" h={160} />
                  <Skeleton w="100%" h={220} />
                </div>
              )}
            </div>
          ) : (
            <PanelBody p={p} programId={programId} listName={listName} onClose={onClose} />
          )}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

function reasonText(t: ReturnType<typeof useT>, reason: string, leftKind: string): string {
  switch (reason) {
    case 'self':
      return t(mf.reasonSelf)
    case 'same_ip':
      return t(mf.reasonSameIp)
    case 'disposable':
      return t(mf.reasonDisposable)
    case 'left':
      return leftKind === 'hard_bounce' ? t(mf.reasonLeftBounce) : leftKind === 'complaint' ? t(mf.reasonLeftComplaint) : t(mf.reasonLeft)
    case 'already':
      return t(mf.reasonAlready)
    case 'never_confirmed':
      return t(mf.reasonNever)
    case 'reviewer':
      return t(mf.reasonReviewer)
  }
  return reason
}

function signalText(t: ReturnType<typeof useT>, s: string): string {
  return s === 'burst' ? t(mf.signalBurst) : s === 'shared_ip' ? t(mf.signalSharedIp) : s === 'numbered' ? t(mf.signalNumbered) : s
}

function PanelBody({ p, programId, listName, onClose }: { p: Panel; programId: string; listName: string; onClose: () => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const canReview = useCan('grow:review')
  const refresh = useGrowRefresh()
  const [pausing, setPausing] = useState(false)
  const [note, setNote] = useState('')
  const r = p.referrer
  const paused = r.linkStatus === 'paused'
  const first = r.firstName || r.name.split(' ')[0] || r.name
  const role = roleName({ key: me.workspace.roleKey, name: me.workspace.roleName })
  const resume = async () => {
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/referrers/{link}/resume', { params: { path: { workspace: slug, program: programId, link: r.linkId } } })
      toast({ kind: 'success', title: t(mf.resumed, { name: r.name }) })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const panelURL = `${window.location.origin}/w/${slug}/grow/referrals/referrers?referrer=${r.linkId}`
  return (
    <>
      <div className="gw-sheet-head">
        <Avatar name={r.name} size={40} color="muted" fontSize={13} />
        <div className="flex min-w-0 flex-1 flex-col gap-[3px]">
          <span className="k">{canReview ? t(mf.panelEyebrow) : t(mf.panelEyebrowRead, { name: user?.name ?? '', role })}</span>
          <RDialog.Title className="m-0 text-[19px] font-bold tracking-[-0.01em]">{r.name}</RDialog.Title>
          <span className="k">
            {p.showAddresses && r.email ? <span className="mono">{r.email}</span> : null}
            {p.showAddresses && r.email && r.since ? ' · ' : null}
            {r.since ? t(mf.since, { list: listName, date: dayMonthYear(r.since) }) : null}
          </span>
          <div className="mt-1 flex flex-wrap gap-1.5">
            <Badge tone={paused ? 'warn' : 'ok'} dot>
              {paused ? t(mf.statusPaused) : t(mf.statusActive)}
            </Badge>
          </div>
        </div>
        <div className="flex flex-none flex-nowrap items-center gap-1.5">
          {r.contactId ? (
            <ButtonLink to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: r.contactId }} size="sm">
              {t(mf.openContact)}
            </ButtonLink>
          ) : null}
          <Button size="sm" iconOnly icon="link" aria-label={t(mf.copyPanel)} onClick={() => void copy(panelURL, t(m.copiedLink))} />
          <RDialog.Close asChild>
            <Button size="sm" iconOnly icon="close" aria-label={t(mf.closePanel)} onClick={onClose} />
          </RDialog.Close>
        </div>
      </div>
      <div className="gw-sheet-body">
        {!canReview ? (
          <Note tone="neutral">
            <T msg={p.helpers.review.length ? mf.readOnly : mf.readOnlyNoHelpers} values={{ helpers: orNames(p.helpers.review), mono }} />
          </Note>
        ) : null}
        {paused ? (
          <Note tone="warn">
            <span className="flex flex-col gap-1.5">
              <span>
                {p.pausedBy
                  ? t(mf.pausedNote, {
                      by: p.pausedBy,
                      date: p.pausedAt ? dayMonthDate(p.pausedAt) : '',
                      count: p.pausedNotCounted,
                      reason: formatList(p.pausedSignals.map((s) => signalText(t, s))),
                    })
                  : t(mf.pausedNoteNoBy, {
                      date: p.pausedAt ? dayMonthDate(p.pausedAt) : '',
                      count: p.pausedNotCounted,
                      reason: formatList(p.pausedSignals.map((s) => signalText(t, s))),
                    })}
              </span>
              {p.pauseNote ? <span className="k">{t(mf.pausedNoteText, { note: p.pauseNote })}</span> : null}
              {canReview ? (
                <span>
                  <Button size="sm" onClick={() => void resume()}>
                    {t(mf.resumeLink)}
                  </Button>
                </span>
              ) : null}
            </span>
          </Note>
        ) : null}
        <section className="flex flex-col gap-2.5">
          <h3>{t(mf.linkHd)}</h3>
          <div className="gw-link">
            <Input className="mono text-[12.5px]" value={p.link} readOnly aria-label={t(mf.linkLabel, { name: first })} />
            <Button size="sm" icon="copy" onClick={() => void copy(`https://${p.link}`, t(m.copiedLink))}>
              {t(mf.copy)}
            </Button>
            {canReview && !paused ? (
              <Button size="sm" onClick={() => setPausing(true)}>
                {t(mf.pauseLink)}
              </Button>
            ) : null}
          </div>
          <span className="k">
            {paused ? (
              t(mf.pausedVisitors, { visits: p.visits, signups: p.signups })
            ) : (
              <T msg={mf.created} values={{ date: dayMonthYear(p.linkCreated), visits: p.visits, signups: p.signups, code: p.code, mono }} />
            )}
          </span>
          <span className="k">
            <T
              msg={mf.choices}
              values={{
                name: first,
                showName: p.showName ? t(m.on).toLowerCase() : t(m.off).toLowerCase(),
                showBoard: p.showLeaderboard ? t(m.on).toLowerCase() : t(m.off).toLowerCase(),
                b: (c) => <b>{c}</b>,
              }}
            />
          </span>
        </section>
        <Progress p={p} first={first} />
        <section className="flex flex-col gap-2.5">
          <h3>{t(mf.referralsHd, { count: p.referrals.length })}</h3>
          {p.referrals.length ? (
            <div className="flex flex-col rounded-[10px] border border-line px-3.5">
              {p.referrals.map((x) => (
                <ReferralRow key={x.id} x={x} show={p.showAddresses} />
              ))}
            </div>
          ) : (
            <span className="k">{t(mf.noReferrals)}</span>
          )}
        </section>
        <Rewards p={p} />
        <span className="k">
          {p.showAddresses ? t(mf.footnote, { name: first, count: p.referrals.length || 1 }) : t(mf.footnoteNoAddresses, { name: first })}
        </span>
      </div>
      <ConfirmDialog
        open={pausing}
        onOpenChange={setPausing}
        tone="warn"
        badge={t(mf.pauseLink)}
        title={t(mf.pauseTitle, { name: r.name })}
        description={t(mf.pauseText)}
        confirmLabel={t(mf.pauseLink)}
        confirmVariant="primary"
        onConfirm={async () => {
          await http.POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/referrers/{link}/pause', {
            params: { path: { workspace: slug, program: programId, link: r.linkId } },
            body: { note },
          })
          toast({ title: t(mf.paused, { name: r.name }) })
          setPausing(false)
          setNote('')
          await refresh()
        }}
      >
        <Field label={t(mf.pauseNoteLabel)} hint={t(mf.pauseNoteHint)} optional>
          <Textarea rows={3} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </ConfirmDialog>
    </>
  )
}

/** The track, the counts and what the next referral brings. */
function Progress({ p, first }: { p: Panel; first: string }) {
  const t = useT()
  const tr = p.track
  if (!tr.points.length) return null
  const items = tr.points
    .map((x) => (x.reachedAt ? t(mf.trackItemReached, { count: x.count }) : t(mf.trackItemToGo, { toGo: Math.max(0, x.count - tr.counted), reward: x.reward })))
    .join('; ')
  return (
    <section className="flex flex-col gap-2.5">
      <h3>{t(mf.progressHd)}</h3>
      <MilestoneTrack
        label={t(mf.trackLabel, { counted: tr.counted, soon: tr.inHold, items })}
        counted={tr.counted}
        soon={tr.inHold}
        here={t(mf.here, { name: first, count: tr.counted })}
        legend
        points={tr.points.map((x) => ({
          count: x.count,
          title: t(mf.pointTitle, { count: x.count, reward: x.reward }),
          sub: x.reachedAt ? t(mf.pointReached, { date: dayMonthDate(x.reachedAt) }) : t(mf.pointMore, { count: Math.max(0, x.count - tr.counted) }),
        }))}
      />
      <span className="text-[13px] text-ink-nav">
        <T
          msg={tr.held ? mf.countsLineHeld : mf.countsLine}
          values={{
            counted: count(tr.counted),
            hold: count(tr.inHold),
            held: count(tr.held),
            awaiting: count(tr.awaiting),
            b: (c) => <b>{c}</b>,
            hl: (c) => <b className="gw-held">{c}</b>,
          }}
        />
      </span>
      {tr.nextCount ? (
        <Note tone="info">
          <span className="flex flex-col gap-1">
            <span className="font-bold">{t(mf.nextTitle, { count: tr.toNext, reward: tr.nextName })}</span>
            <span>
              {tr.nextHold && tr.nextBySeq
                ? t(mf.nextText, { seq: tr.nextBySeq, reward: tr.nextName, date: dayMonthDate(tr.nextHold) })
                : t(mf.nextTextNoHold)}
            </span>
          </span>
        </Note>
      ) : (
        <span className="k">{t(mf.allReached)}</span>
      )}
    </section>
  )
}

function ReferralRow({ x, show }: { x: Schemas['ReferralItem']; show: boolean }) {
  const t = useT()
  const sameDay = x.confirmedAt && dayMonthDate(x.confirmedAt) === dayMonthDate(x.signedUpAt)
  const line = x.confirmedAt
    ? sameDay
      ? t(mf.signedConfirmedSame, { date: dayMonthDate(x.signedUpAt) })
      : t(mf.signedConfirmed, { signed: dayMonthDate(x.signedUpAt), confirmed: dayMonthDate(x.confirmedAt) })
    : x.remindedAt
      ? t(mf.signedReminded, { signed: dayMonthDate(x.signedUpAt), reminded: dayMonthDate(x.remindedAt) })
      : t(mf.signedOnly, { signed: dayMonthDate(x.signedUpAt) })
  let tone: Tone = 'neutral'
  let label = ''
  switch (x.status) {
    case 'counted':
      tone = 'ok'
      label = t(mf.stCounted, { date: x.countedAt ? dayMonthDate(x.countedAt) : '' })
      break
    case 'hold':
      tone = 'info'
      label = x.countsOn ? t(mf.stHold, { date: dayMonthDate(x.countsOn) }) : t(mf.stHoldNoDate)
      break
    case 'awaiting':
      label = t(mf.stAwaiting)
      break
    case 'review':
      tone = 'warn'
      label = t(mf.stReview)
      break
    case 'not_counted':
      tone = 'bad'
      label = t(mf.stNotCounted, { reason: reasonText(t, x.reason, x.leftKind) })
      break
  }
  return (
    <div className="gw-lrow flex-wrap" style={{ alignItems: 'flex-start', gap: '6px 12px' }}>
      <span className="k gw-num" style={{ width: 16, paddingTop: 2 }}>
        {x.number}
      </span>
      <span className="tx" style={{ flex: '1 1 220px' }}>
        <span className="mono text-[12.5px] font-semibold [overflow-wrap:anywhere]">{show && x.email ? x.email : t(mf.referralNumber, { n: x.number })}</span>
        <span className="k">{line}</span>
      </span>
      <span className="flex-none">
        <Badge tone={tone}>{label}</Badge>
      </span>
    </div>
  )
}

function rewardStatus(t: ReturnType<typeof useT>, x: Schemas['ReferralRewardItem']): { tone: Tone; label: string } {
  switch (x.status) {
    case 'sent':
      if (x.usedAt) return { tone: 'ok', label: x.type === 'coupon' ? t(mf.rwClaimed, { date: dayMonthDate(x.usedAt) }) : t(mf.rwDownloaded, { date: dayMonthDate(x.usedAt) }) }
      return { tone: x.type === 'coupon' ? 'info' : 'ok', label: x.type === 'coupon' ? t(mf.rwSentNotClaimed) : t(mf.rwSent) }
    case 'pending':
      return { tone: 'info', label: t(mf.rwPending) }
    case 'waiting_codes':
      return { tone: 'warn', label: t(mf.rwWaitingCodes) }
    case 'to_fulfill':
      return { tone: 'warn', label: t(mf.rwToFulfill) }
    case 'fulfilled':
      return { tone: 'ok', label: t(mf.rwFulfilled, { date: x.usedAt ? dayMonthDate(x.usedAt) : '' }) }
    case 'failed':
      return { tone: 'bad', label: t(mf.rwFailed) }
    case 'expired':
      return { tone: 'neutral', label: t(mf.rwExpired) }
  }
  return { tone: 'neutral', label: x.status }
}

function Rewards({ p }: { p: Panel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canReview = useCan('grow:review')
  const refresh = useGrowRefresh()
  const [busy, setBusy] = useState<string>()
  const act = async (x: Schemas['ReferralRewardItem'], what: 'resend' | 'retry' | 'claim') => {
    setBusy(`${x.id}-${what}`)
    const path = { workspace: slug, reward: x.id }
    try {
      if (what === 'resend') {
        await http.POST('/api/v1/workspaces/{workspace}/grow/rewards/{reward}/resend', { params: { path } })
        toast({ kind: 'success', title: t(mf.resent, { reward: x.name, name: p.referrer.name }), detail: `${clock(new Date())} · ${t(mf.resentDetail)}` })
      } else if (what === 'retry') {
        await http.POST('/api/v1/workspaces/{workspace}/grow/rewards/{reward}/retry', { params: { path } })
        toast({ title: t(mf.retried, { reward: x.name, name: p.referrer.name }) })
      } else {
        const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/rewards/{reward}/claim-link', { params: { path } })
        if (data) {
          await navigator.clipboard.writeText(data.url).catch(() => undefined)
          toast({
            kind: 'success',
            title: t(mf.claimCopied),
            detail: data.expiresAt ? t(mf.claimCopiedDetail, { date: dayMonthDate(data.expiresAt) }) : t(mf.claimCopiedDetailNoDate),
          })
        }
      }
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <section className="flex flex-col gap-2.5">
      <h3>{t(mf.rewardsHd, { count: p.rewards.length })}</h3>
      {p.rewards.length ? (
        p.rewards.map((x) => {
          const st = rewardStatus(t, x)
          const line = !x.sentAt
            ? t(mf.rewardLineReached, { count: x.milestone, date: dayMonthDate(x.reachedAt) })
            : x.expiresAt
              ? t(mf.rewardLineExpires, { count: x.milestone, date: dayMonthDate(x.sentAt), expires: dayMonthDate(x.expiresAt) })
              : x.fileName
                ? t(mf.rewardLineFile, { count: x.milestone, date: dayMonthDate(x.sentAt), type: fileKind(x.fileType), size: fileSize(x.fileSize) })
                : t(mf.rewardLine, { count: x.milestone, date: dayMonthDate(x.sentAt) })
          return (
            <div key={x.id} className="gw-lrow flex-wrap rounded-[10px] border border-line px-3.5 py-3" style={{ borderTop: '1px solid var(--line)' }}>
              <span className="gw-mtile" style={{ width: 40, height: 40 }} aria-hidden="true">
                <Icon name={x.type === 'download' ? 'download' : x.type === 'coupon' ? 'star' : x.type === 'codes' ? 'tag' : x.type === 'webhook' ? 'webhook' : 'send'} size={16} />
              </span>
              <span className="tx" style={{ flex: '1 1 200px' }}>
                <span className="gw-tt2">{x.name}</span>
                <span className="k">{line}</span>
                {x.note ? <span className="k">{x.note}</span> : null}
                {x.error ? <span className="k text-bad-fg">{x.error}</span> : null}
              </span>
              <Badge tone={st.tone}>{st.label}</Badge>
              {canReview ? (
                <span className="flex flex-wrap gap-1.5">
                  {x.status === 'failed' ? (
                    <Button size="sm" loading={busy === `${x.id}-retry`} onClick={() => void act(x, 'retry')}>
                      {t(mf.retry)}
                    </Button>
                  ) : x.sentAt ? (
                    <Button size="sm" loading={busy === `${x.id}-resend`} onClick={() => void act(x, 'resend')}>
                      {t(mf.resend)}
                    </Button>
                  ) : null}
                  {x.hasClaim ? (
                    <Button size="sm" loading={busy === `${x.id}-claim`} onClick={() => void act(x, 'claim')}>
                      {t(mf.copyClaim)}
                    </Button>
                  ) : null}
                </span>
              ) : null}
            </div>
          )
        })
      ) : (
        <span className="k">{t(mf.noRewards)}</span>
      )}
    </section>
  )
}
