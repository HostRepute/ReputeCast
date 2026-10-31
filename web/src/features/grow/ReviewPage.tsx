import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { mv } from './review.messages'
import { GROW, GrowPage, LoadError, ProgramNav, orNames, useGrowRefresh, useReferralPrograms } from './shared'

type Queue = Schemas['ReferralReviewQueue']
type Group = Schemas['ReferralReviewGroup']
type Item = Schemas['ReferralHeldItem']
type Decision = 'count' | 'dont' | 'dont_pause'

export interface ReviewSearch {
  program?: string
  signal?: 'burst' | 'shared_ip' | 'numbered'
  referrer?: string
}

/**
 * Referrals › Review (ReferralReview, ReferralReviewPhone, ReferralStates):
 * held referrals grouped by referrer with their evidence, decided by a person
 * (grow:review), with Undo for 10 seconds.
 */
export function ReviewPage({ search, onSearch }: { search: ReviewSearch; onSearch: (s: ReviewSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const { list, program, q: pq } = useReferralPrograms(search.program)
  const q = $api.useQuery(
    'get',
    `${GROW}/referrals/{program}/review`,
    { params: { path: { workspace: slug, program: program?.id ?? '' }, query: { signal: search.signal, referrer: search.referrer } } },
    { enabled: Boolean(program), placeholderData: (prev) => prev },
  )
  const d = q.data
  const err = pq.error ?? q.error
  return (
    <GrowPage tab="referrals" documentTitle={t(m.docReferrals)}>
      <div className="gw">
        <ProgramNav current="review" held={d?.held} programId={search.program} />
        {!list || (program && !d) ? (
          err ? (
            <LoadError error={err} title={t(mv.loadError)} onRetry={() => void (pq.isError ? pq.refetch() : q.refetch())} />
          ) : (
            <Loading label={t(mv.loadingLabel)}>
              <Skeleton w="100%" h={480} r={10} />
            </Loading>
          )
        ) : !program || !d ? null : (
          <Queue d={d} programId={program.id} listName={program.listName} search={search} onSearch={onSearch} />
        )}
      </div>
    </GrowPage>
  )
}

function Queue({ d, programId, listName, search, onSearch }: { d: Queue; programId: string; listName: string; search: ReviewSearch; onSearch: (s: ReviewSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canReview = useCan('grow:review')
  const [picked, setSelected] = useState<Set<string>>(new Set())
  // Referrals no longer held drop out of the selection.
  const ids = useMemo(() => new Set(d.groups.flatMap((g) => g.items.map((x) => x.id))), [d])
  const selected = useMemo(() => new Set([...picked].filter((id) => ids.has(id))), [picked, ids])
  const toggle = (id: string, on: boolean) =>
    setSelected((cur) => {
      const next = new Set(cur)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })
  const filtered = Boolean(search.signal || search.referrer)
  const signalName = (k: string) => (k === 'burst' ? t(mv.sigBurst) : k === 'shared_ip' ? t(mv.sigShared) : t(mv.sigNumbered))
  if (!d.groups.length && !filtered) {
    return (
      <EmptyState
        title={t(mv.emptyTitle)}
        action={
          <Link to="/w/$workspace/grow/referrals/settings" params={{ workspace: slug }} search={search.program ? { program: search.program } : {}} className="btn">
            {t(mv.rules)}
          </Link>
        }
      >
        {t(mv.emptyText)}
      </EmptyState>
    )
  }
  return (
    <>
      <div className="flex flex-col gap-1">
        <h2 className="h2">{d.oldest ? t(mv.heading, { count: d.held, date: dayMonthDate(d.oldest) }) : t(mv.headingNoDate, { count: d.held })}</h2>
        <span className="k">{d.holdDays ? t(mv.lead, { days: d.holdDays }) : t(mv.leadNoHold)}</span>
      </div>
      {!canReview ? (
        <Note tone="neutral">
          {t(d.helpers.review.length ? mv.readOnly : mv.readOnlyNoHelpers, { helpers: orNames(d.helpers.review) })}
        </Note>
      ) : null}
      <div className="flex flex-wrap items-center gap-2.5">
        <Select aria-label={t(mv.signalFilter)} style={{ width: 240 }} value={search.signal ?? ''} onChange={(e) => onSearch({ ...search, signal: (e.target.value || undefined) as ReviewSearch['signal'] })}>
          <option value="">{t(mv.allSignals)}</option>
          {d.signals.map((s) => (
            <option key={s.kind} value={s.kind}>
              {t(mv.signalOption, { signal: signalName(s.kind), count: s.referrers })}
            </option>
          ))}
        </Select>
        <Select aria-label={t(mv.referrerFilter)} style={{ width: 240 }} value={search.referrer ?? ''} onChange={(e) => onSearch({ ...search, referrer: e.target.value || undefined })}>
          <option value="">{t(mv.allReferrers)}</option>
          {d.referrers.map((r) => (
            <option key={r.linkId} value={r.linkId}>
              {t(mv.referrerOption, { name: r.name, count: r.count })}
            </option>
          ))}
        </Select>
        <Link to="/w/$workspace/grow/referrals/settings" params={{ workspace: slug }} search={search.program ? { program: search.program } : {}} className="gw-lnk ml-auto">
          {t(mv.rules)}
        </Link>
      </div>
      {d.groups.length ? (
        d.groups.map((g) => (
          <GroupCard
            key={g.referrer.linkId}
            g={g}
            d={d}
            listName={listName}
            programId={programId}
            selected={selected}
            onToggle={toggle}
            onSelectAll={(on) => g.items.forEach((x) => toggle(x.id, on))}
          />
        ))
      ) : (
        <section className="card gw-pad flex flex-wrap items-center justify-between gap-3">
          <span className="k">{t(mv.emptyFiltered)}</span>
          <Button size="sm" onClick={() => onSearch({ program: search.program })}>
            {t(mv.clearFilters)}
          </Button>
        </section>
      )}
      {canReview && selected.size ? <DecisionBar d={d} programId={programId} selected={selected} onDone={() => setSelected(new Set())} /> : null}
    </>
  )
}

function networkText(t: ReturnType<typeof useT>, n: Schemas['ReferralNetwork']): string {
  if (!n.name) return t(mv.netUnknown, { cidr: n.cidr })
  const v = { name: n.name, asn: n.asn }
  switch (n.kind) {
    case 'hosting':
      return t(mv.netHosting, v)
    case 'office':
      return t(mv.netOffice, v)
    case 'residential':
      return t(mv.netResidential, v)
    case 'mobile':
      return t(mv.netMobile, v)
  }
  return t(mv.netNamed, v)
}

function signalChip(t: ReturnType<typeof useT>, s: Schemas['ReferralSignal']): string {
  if (s.kind === 'burst') return t(mv.chipBurst, { count: s.count, minutes: s.minutes })
  if (s.kind === 'shared_ip') return t(mv.chipShared, { count: s.count })
  return t(mv.chipNumbered)
}

function signalLine(t: ReturnType<typeof useT>, s: Schemas['ReferralSignal']): string {
  if (s.kind === 'burst') {
    // A burst's window: from the first confirmation to the last.
    if (!s.from || !s.to) return t(mv.lineBurstNoTimes, { count: s.count, minutes: s.minutes, threshold: s.threshold })
    return t(mv.lineBurst, { count: s.count, minutes: s.minutes, first: clock(s.from), last: clock(s.to), date: dayMonthDate(s.to), threshold: s.threshold })
  }
  if (s.kind === 'shared_ip') {
    const v = { count: s.count, ip: s.ip, network: networkText(t, s.network), threshold: s.threshold }
    return s.all ? t(mv.lineSharedAll, v) : t(mv.lineShared, v)
  }
  // Numbered addresses: the first and the last of them.
  return t(mv.lineNumbered, { from: s.first, to: s.last })
}

function engagementText(t: ReturnType<typeof useT>, e: Schemas['ReferralEngagement']): string {
  const parts: string[] = []
  if (e.short.length) parts.push(t(mv.engOpened, { list: formatList(e.short) }))
  if (e.clicks) parts.push(t(mv.engClicks, { count: e.clicks }))
  if (parts.length) return parts.join(' · ')
  if (e.joinedAfter) return t(mv.engJoinedAfter, { campaign: e.joinedAfter })
  return t(mv.engNone)
}

function GroupCard({
  g,
  d,
  listName,
  programId,
  selected,
  onToggle,
  onSelectAll,
}: {
  g: Group
  d: Queue
  listName: string
  programId: string
  selected: Set<string>
  onToggle: (id: string, on: boolean) => void
  onSelectAll: (on: boolean) => void
}) {
  const t = useT()
  const canReview = useCan('grow:review')
  const r = g.referrer
  const all = g.items.every((x) => selected.has(x.id))
  const some = g.items.some((x) => selected.has(x.id))
  const [deciding, setDeciding] = useState<Decision>()
  const signals = formatList(g.signals.map((s) => (s.kind === 'burst' ? t(mv.nameBurst) : s.kind === 'shared_ip' ? t(mv.nameShared) : t(mv.nameNumbered))))
  return (
    <section className="card">
      <div className="sec flex-wrap" style={{ alignItems: 'flex-start' }}>
        <div className="flex min-w-0 flex-[1_1_380px] items-start gap-3">
          {canReview ? (
            <input
              type="checkbox"
              className="mt-2"
              aria-label={t(mv.selectAll, { count: g.items.length, name: r.name })}
              checked={all}
              ref={(el) => {
                if (el) el.indeterminate = some && !all
              }}
              onChange={(e) => onSelectAll(e.target.checked)}
            />
          ) : null}
          <Avatar name={r.name} size={36} color="muted" />
          <div className="flex min-w-0 flex-col gap-0.5">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="h2">{r.name}</h2>
              <Badge tone="warn">{t(mv.held, { count: g.items.length })}</Badge>
            </div>
            <span className="k">
              {d.showAddresses && r.email ? <span className="mono">{r.email}</span> : null}
              {d.showAddresses && r.email ? ' · ' : null}
              {r.since
                ? t(mv.personLine, { list: listName, date: dayMonthYear(r.since), counted: count(r.counted), status: r.linkStatus === 'paused' ? t(mv.linkPaused) : t(mv.linkActive) })
                : t(mv.personLineNoSince, { counted: count(r.counted), status: r.linkStatus === 'paused' ? t(mv.linkPaused) : t(mv.linkActive) })}
            </span>
          </div>
        </div>
        {canReview ? (
          <div className="seg" role="group" aria-label={t(mv.decideAll, { count: g.items.length })}>
            {(['count', 'dont', 'dont_pause'] as Decision[]).map((x) => (
              <button key={x} type="button" aria-pressed={deciding === x} onClick={() => setDeciding(x)}>
                {x === 'count' ? t(mv.count) : x === 'dont' ? t(mv.dont) : t(mv.dontPause)}
              </button>
            ))}
          </div>
        ) : null}
      </div>
      <div className="gw-pad flex flex-col gap-3.5">
        {g.signals.length ? (
          <div className="flex flex-col gap-2">
            <h3 className="gw-hd3">{t(mv.signals)}</h3>
            <div className="gw-siglist">
              {g.signals.map((s, i) => (
                <div key={`${s.kind}-${i}`}>
                  <span className="gw-sig">{signalChip(t, s)}</span>
                  <span>{signalLine(t, s)}</span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
        {g.suggestion === 'dont_pause' ? (
          <Note tone="warn">
            <span className="flex flex-col gap-1">
              <span className="font-bold">{t(mv.suggestDontPause)}</span>
              <span>{t(mv.suggestDontPauseText, { count: g.items.length, signals })}</span>
            </span>
          </Note>
        ) : g.suggestion === 'count' ? (
          <Note tone="ok">
            <span className="flex flex-col gap-1">
              <span className="font-bold">{t(mv.suggestCount)}</span>
              <span>
                {g.openedIn
                  ? t(mv.suggestCountText, {
                      names: g.differentNames ? 'yes' : 'no',
                      domain: g.sameDomain ? 'yes' : 'none',
                      days: g.days,
                      opened: g.opened,
                      count: g.items.length,
                      issue: g.openedIn,
                      confirmed: formatList(g.confirmedDays.map((x) => dayMonthDate(x))),
                      ends: formatList(g.holdEnds.map((x) => dayMonthDate(x))),
                    })
                  : t(mv.suggestCountTextNoIssue, {
                      names: g.differentNames ? 'yes' : 'no',
                      domain: g.sameDomain ? 'yes' : 'none',
                      days: g.days,
                      confirmed: formatList(g.confirmedDays.map((x) => dayMonthDate(x))),
                      ends: formatList(g.holdEnds.map((x) => dayMonthDate(x))),
                    })}
              </span>
            </span>
          </Note>
        ) : null}
      </div>
      <HeldTable g={g} show={d.showAddresses} selected={selected} onToggle={onToggle} />
      {deciding ? (
        <DecisionBar
          d={d}
          programId={programId}
          selected={new Set(g.items.map((x) => x.id))}
          initial={deciding}
          inline
          onDone={() => setDeciding(undefined)}
          onCancel={() => setDeciding(undefined)}
        />
      ) : null}
    </section>
  )
}

function HeldTable({ g, show, selected, onToggle }: { g: Group; show: boolean; selected: Set<string>; onToggle: (id: string, on: boolean) => void }) {
  const t = useT()
  const canReview = useCan('grow:review')
  const name = (x: Item, i: number) => (show && x.email ? x.email : t(mv.addressHidden, { n: i + 1 }))
  return (
    <>
      <div className="max-md:hidden">
        <TableScroll label={t(mv.tableLabel, { name: g.referrer.name })}>
          <table className="tbl gw-tbl">
            <thead>
              <tr>
                {canReview ? (
                  <th scope="col" style={{ width: 36 }}>
                    <span className="vh">{t(mv.selectAll, { count: g.items.length, name: g.referrer.name })}</span>
                  </th>
                ) : null}
                <th scope="col">{t(mv.colReferral)}</th>
                <th scope="col">{t(mv.colSigned)}</th>
                <th scope="col">{t(mv.colConfirmed)}</th>
                <th scope="col">{t(mv.colIp)}</th>
                <th scope="col">{t(mv.colEngagement)}</th>
                <th scope="col">{t(mv.colHold)}</th>
              </tr>
            </thead>
            <tbody>
              {g.items.map((x, i) => (
                <tr key={x.id} className={selected.has(x.id) ? 'sel' : undefined}>
                  {canReview ? (
                    <td>
                      <input type="checkbox" aria-label={t(mv.selectOne, { email: name(x, i) })} checked={selected.has(x.id)} onChange={(e) => onToggle(x.id, e.target.checked)} />
                    </td>
                  ) : null}
                  <td className="mono text-[12.5px]">{name(x, i)}</td>
                  <td className="nw">
                    {dayMonthDate(x.signedUpAt)}, {clock(x.signedUpAt)}
                  </td>
                  <td className="nw">
                    {x.confirmedAt ? (dayMonthDate(x.confirmedAt) === dayMonthDate(x.signedUpAt) ? clock(x.confirmedAt) : `${dayMonthDate(x.confirmedAt)}, ${clock(x.confirmedAt)}`) : t(m.none)}
                  </td>
                  <td>
                    <span className="flex flex-col">
                      <span className="mono text-[12.5px]">{x.ip || x.network.cidr}</span>
                      {x.network.name ? (
                        <span className="k">
                          {x.network.name}
                          {x.network.asn ? ` · ${x.network.asn}` : ''}
                        </span>
                      ) : null}
                    </span>
                  </td>
                  <td>{engagementText(t, x.engagement)}</td>
                  <td className="nw">{x.holdEnds ? t(mv.holdEnds, { date: dayMonthDate(x.holdEnds) }) : t(m.none)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="md:hidden">
        {g.items.map((x, i) => (
          <label key={x.id} className="gw-prow" style={{ alignItems: 'flex-start' }}>
            {canReview ? <input type="checkbox" className="mt-1" aria-label={t(mv.selectOne, { email: name(x, i) })} checked={selected.has(x.id)} onChange={(e) => onToggle(x.id, e.target.checked)} /> : null}
            <span className="tx">
              <span className="mono text-[12.5px] font-semibold">{name(x, i)}</span>
              <span className="k">
                {x.confirmedAt
                  ? t(mv.phoneSigned, { signed: `${dayMonthDate(x.signedUpAt)}, ${clock(x.signedUpAt)}`, confirmed: clock(x.confirmedAt) })
                  : t(mv.phoneSignedOnly, { signed: `${dayMonthDate(x.signedUpAt)}, ${clock(x.signedUpAt)}` })}
              </span>
              <span className="k">
                <span className="mono">{x.ip || x.network.cidr}</span>
                {x.network.name ? ` · ${x.network.name}` : ''}
                {x.network.asn ? ` · ${x.network.asn}` : ''}
              </span>
              <span className="k">
                {x.holdEnds
                  ? t(mv.phoneEngagement, { text: engagementText(t, x.engagement), date: dayMonthDate(x.holdEnds) })
                  : t(mv.phoneEngagementNoHold, { text: engagementText(t, x.engagement) })}
              </span>
            </span>
          </label>
        ))}
      </div>
    </>
  )
}

/** The decision for the selected referrals: preview, note, decide, then Undo for 10 seconds. */
function DecisionBar({
  d,
  programId,
  selected,
  initial = 'count',
  inline,
  onDone,
  onCancel,
}: {
  d: Queue
  programId: string
  selected: Set<string>
  initial?: Decision
  inline?: boolean
  onDone: () => void
  onCancel?: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [decision, setDecision] = useState<Decision>(initial)
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const ids = [...selected].sort()
  const groups = d.groups.filter((g) => g.items.some((x) => selected.has(x.id)))
  const name = groups.length === 1 ? groups[0]!.referrer.name : ''
  const preview = $api.useQuery(
    'post',
    `${GROW}/referrals/{program}/review/preview`,
    { params: { path: { workspace: slug, program: programId } }, body: { decision, ids } },
    { enabled: ids.length > 0, placeholderData: (prev) => prev },
  )
  const p = preview.data
  const previewText = !p
    ? t(mv.previewing)
    : decision === 'count'
      ? `${p.now ? t(mv.previewCountNow, { count: p.count, referrer: p.referrer, after: p.after }) : t(mv.previewCount, { count: p.count, referrer: p.referrer, after: p.after, date: p.holdEnds ? dayMonthDate(p.holdEnds) : '' })}${p.rewards.length ? ` ${t(mv.previewRewards, { rewards: formatList(p.rewards) })}` : ''}`
      : decision === 'dont'
        ? t(mv.previewDont, { count: p.count, referrer: p.referrer, counted: p.counted })
        : t(mv.previewDontPause, { count: p.count, referrer: p.referrer })
  const decide = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/review/decide', {
        params: { path: { workspace: slug, program: programId } },
        body: { decision, ids, note: note.trim() || undefined },
      })
      const n = ids.length
      const title = decision === 'count' ? t(mv.doneCount, { count: n }) : decision === 'dont' ? t(mv.doneDont, { count: n }) : t(mv.doneDontPause, { count: n, name: data?.preview.referrer ?? name })
      toast({
        kind: 'success',
        title,
        detail: t(mv.doneDetail, { note: note.trim() ? 'yes' : 'no' }),
        undo: data
          ? () => {
              void http
                .POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/review/decisions/{decision}/undo', {
                  params: { path: { workspace: slug, program: programId, decision: data.id } },
                })
                .then(() => {
                  toast({ title: t(mv.undone), detail: t(mv.undoneDetail) })
                  void refresh()
                })
                .catch((e: unknown) => toast({ kind: 'error', title: t(mv.undoFailed), detail: errorMessage(e) }))
            }
          : undefined,
      })
      setNote('')
      onDone()
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const action = decision === 'count' ? t(mv.actCount, { count: ids.length }) : decision === 'dont' ? t(mv.actDont, { count: ids.length }) : t(mv.actDontPause)
  return (
    <section className={inline ? 'gw-pad flex flex-col gap-2.5 border-t border-line-2' : 'gw-dbar'} aria-label={t(mv.barLabel, { count: ids.length })}>
      <span className="font-semibold">{name ? t(mv.barSelected, { count: ids.length, name }) : t(mv.barSelectedMany, { count: ids.length, referrers: groups.length })}</span>
      <div className="flex flex-wrap items-center gap-2.5">
        <div className="seg" role="radiogroup" aria-label={t(mv.decision)}>
          {(['count', 'dont', 'dont_pause'] as Decision[]).map((x) => (
            <button key={x} type="button" role="radio" aria-checked={decision === x} className={decision === x ? 'on' : undefined} onClick={() => setDecision(x)}>
              {x === 'count' ? t(mv.count) : x === 'dont' ? t(mv.dont) : t(mv.dontPause)}
            </button>
          ))}
        </div>
        <Input className="min-w-[200px] flex-1" aria-label={t(mv.note)} placeholder={t(mv.notePlaceholder)} value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />
      </div>
      <span className="text-[13px] text-ink-nav">{previewText}</span>
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onCancel ?? onDone}>{t(mv.cancel)}</Button>
        <Button variant={decision === 'count' ? 'primary' : 'danger-primary'} loading={busy} onClick={() => void decide()}>
          {action}
        </Button>
      </div>
    </section>
  )
}
