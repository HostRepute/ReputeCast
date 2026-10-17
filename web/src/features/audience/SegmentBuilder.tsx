import { keepPreviousData, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { StackBar } from '@/components/ui/Data'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { clock, count, pct, shortDate } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { AudienceTabs } from './AudienceTabs'
import { FitSelect } from './FitSelect'
import { SqlBlock, SqlEditor } from './SqlCode'
import { tierColor, tierLabel, type Tier } from './labels'
import { BetweenTo } from '@/features/scoring/ContactsScore'
import { DraftBanner, DraftFooter, DraftMark, DraftSqlNote, draftCountLabels, useSegmentAssist, type SegmentDraft } from '@/features/ai/segment'
import { useT } from '@/i18n'
import './audience.css'

type Catalog = Schemas['SegmentCatalog']
type CatalogField = Schemas['SegmentCatalogField']

interface Cond {
  key: number
  field: string
  op: string
  value: string
  /** The upper end of "is between" (Score, X13). */
  to: string
  amount: number
  unit: 'days' | 'weeks'
}
interface Group {
  key: number
  match: 'all' | 'any'
  conditions: Cond[]
}
/** An AI segment draft being looked at: what it replaced, and its marks. */
interface AIDraft {
  draft: SegmentDraft
  before: Group[]
  beforeCount: number
  beforeEmpty: boolean
  added: Set<number>
  /** Conditions the draft removes, by group: shown struck through, not counted. */
  removed: Cond[][]
}
interface Draft {
  name: string
  refresh: 'dynamic' | 'static'
  mode: 'builder' | 'sql'
  groups: Group[]
  sql: string
}

let nextKey = 1
const k = () => nextKey++

function blankCond(f: CatalogField | undefined): Cond {
  return { key: k(), field: f?.key ?? 'mailbox_provider', op: f?.ops[0]?.key ?? 'is', value: f?.options[0]?.value ?? '', to: '', amount: 30, unit: 'days' }
}

function fromDetail(d: Schemas['SegmentDetail']): Draft {
  return {
    name: d.name,
    refresh: d.refresh === 'static' ? 'static' : 'dynamic',
    mode: d.mode === 'sql' ? 'sql' : 'builder',
    sql: d.sql,
    groups: (d.definition?.groups ?? []).map((g) => ({
      key: k(),
      match: g.match === 'any' ? 'any' : 'all',
      conditions: (g.conditions ?? []).map((c) => ({
        key: k(),
        field: c.field,
        op: c.op,
        value: c.value ?? '',
        to: c.to ?? '',
        amount: c.amount || 30,
        unit: c.unit === 'weeks' ? 'weeks' : 'days',
      })),
    })),
  }
}

function toInput(d: Draft, catalog?: Catalog): Schemas['SegmentInput'] {
  const kindOf = (field: string) => catalog?.fields.find((f) => f.key === field)?.kind
  return {
    name: d.name,
    mode: d.mode,
    refresh: d.refresh,
    sql: d.mode === 'sql' ? d.sql : undefined,
    definition: {
      groups: d.groups.map((g) => ({
        match: g.match,
        conditions: g.conditions.map((c) =>
          kindOf(c.field) === 'period'
            ? { field: c.field, op: c.op, amount: c.amount, unit: c.unit }
            : { field: c.field, op: c.op, value: c.value, to: c.op === 'between' ? c.to : undefined },
        ),
      })),
    },
  }
}

/** Value a debounced copy of `v`, updated `ms` after the last change. */
function useDebounced<T>(v: T, ms: number): T {
  const [d, setD] = useState(v)
  useEffect(() => {
    const h = window.setTimeout(() => setD(v), ms)
    return () => window.clearTimeout(h)
  }, [v, ms])
  return d
}

/** Parses "body.definition.groups[1].conditions[0].value" into [1, 0]. */
function invalidAt(err: unknown): [number, number] | null {
  if (!(err instanceof ApiError)) return null
  const loc = err.problem.errors?.[0]?.location ?? ''
  const m = /groups\[(\d+)\]\.conditions\[(\d+)\]/.exec(loc)
  return m ? [Number(m[1]), Number(m[2])] : null
}

/** /audience/segments: the first saved segment, or a new one. */
export function SegmentsIndex() {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } })
  const first = list.data?.items[0]?.id
  useEffect(() => {
    if (!list.data) return
    void navigate({ to: '/w/$workspace/audience/segments/$segmentId', params: { workspace: slug, segmentId: first ?? 'new' }, replace: true })
  }, [list.data, first, navigate, slug])
  if (list.isError) throw list.error
  return (
    <Page title="Audience" documentTitle="Segments" filters={<AudienceTabs />}>
      <BuilderSkeleton />
    </Page>
  )
}

/** 7.10 Segment builder: conditions, generated SQL, live count and saved segments. */
export function SegmentBuilderPage({ segmentId }: { segmentId: string }) {
  const { slug } = useWorkspace()
  const isNew = segmentId === 'new'
  const seg = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/segments/{segment}',
    { params: { path: { workspace: slug, segment: segmentId } } },
    { enabled: !isNew },
  )
  const catalog = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments/catalog', { params: { path: { workspace: slug } } })
  if (!isNew && seg.error instanceof ApiError && (seg.error.status === 404 || seg.error.status === 403)) throw seg.error
  const ready = catalog.data && (isNew || seg.data)
  return ready ? (
    <Builder key={segmentId} segmentId={segmentId} detail={isNew ? undefined : seg.data} catalog={catalog.data} />
  ) : (
    <Page title="Audience" documentTitle="Segments" filters={<AudienceTabs />}>
      {seg.isError || catalog.isError ? (
        <ErrorPanel
          title="Couldn't load the segment"
          actions={
            <Button
              onClick={() => {
                void seg.refetch()
                void catalog.refetch()
              }}
            >
              Retry
            </Button>
          }
        >
          {errorMessage(seg.error ?? catalog.error)} Nothing was changed.
        </ErrorPanel>
      ) : (
        <BuilderSkeleton />
      )}
    </Page>
  )
}

function Builder({ segmentId, detail, catalog }: { segmentId: string; detail?: Schemas['SegmentDetail']; catalog: Catalog }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const field = (key: string) => catalog.fields.find((f) => f.key === key)
  const initial = useMemo<Draft>(
    () => (detail ? fromDetail(detail) : { name: '', refresh: 'dynamic', mode: 'builder', sql: '', groups: [{ key: k(), match: 'all', conditions: [blankCond(catalog.fields[0])] }] }),
    [detail, catalog],
  )
  const [draft, setDraft] = useState<Draft>(initial)
  const canSQL = useCan('segments:sql')
  const editReason = usePermissionReason('segments:edit')
  const sqlReason = usePermissionReason('segments:sql')
  const strip = (d: Draft) => JSON.stringify(toInput(d, catalog))
  const dirty = strip(draft) !== strip(initial)
  const input = useDebounced(toInput(draft, catalog), 300)
  const preview = $api.useQuery(
    'post',
    '/api/v1/workspaces/{workspace}/segments/preview',
    { params: { path: { workspace: slug } }, body: input },
    { placeholderData: keepPreviousData, retry: false, enabled: input.mode === 'builder' || (canSQL && input.sql !== '') },
  )
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/segments')
  const update = $api.useMutation('put', '/api/v1/workspaces/{workspace}/segments/{segment}')
  const [saveErr, setSaveErr] = useState<string | null>(null)
  const save = async () => {
    setSaveErr(null)
    try {
      const body = toInput(draft, catalog)
      const out =
        segmentId === 'new'
          ? await create.mutateAsync({ params: { path: { workspace: slug } }, body })
          : await update.mutateAsync({ params: { path: { workspace: slug, segment: segmentId } }, body })
      await qc.invalidateQueries({ predicate: (q) => String((q.queryKey as unknown[])[1] ?? '').includes('/segments') })
      toast({ kind: 'success', title: segmentId === 'new' ? `Saved the segment "${out.name}"` : `Saved "${out.name}"` })
      if (segmentId === 'new') {
        setDraft(fromDetail(out))
        void navigate({ to: '/w/$workspace/audience/segments/$segmentId', params: { workspace: slug, segmentId: out.id }, replace: true })
      }
      return out
    } catch (e) {
      setSaveErr(errorMessage(e))
      throw e
    }
  }
  const bad = invalidAt(preview.error)
  const set = (patch: Partial<Draft>) => setDraft((d) => ({ ...d, ...patch }))
  const setGroup = (gi: number, patch: Partial<Group>) => setDraft((d) => ({ ...d, groups: d.groups.map((g, i) => (i === gi ? { ...g, ...patch } : g)) }))
  const setCond = (gi: number, ci: number, patch: Partial<Cond>) =>
    setDraft((d) => ({
      ...d,
      groups: d.groups.map((g, i) => (i === gi ? { ...g, conditions: g.conditions.map((c, j) => (j === ci ? { ...c, ...patch } : c)) } : g)),
    }))
  const generated = preview.data?.generatedSql || detail?.generatedSql || ''
  // Describe the segment (features/ai): a draft shows in the conditions, marked, until it's decided.
  const t = useT()
  const [ai, setAi] = useState<AIDraft | null>(null)
  // A new segment's first condition, as the builder put it there, isn't one the person chose.
  const conditionsOf = (d: Draft) => JSON.stringify(toInput({ ...d, name: '' }, catalog).definition)
  const untouched = segmentId === 'new' && conditionsOf(draft) === conditionsOf(initial)
  const segAI = useSegmentAssist({
    current: () => (untouched ? { groups: [] } : (toInput(draft, catalog).definition ?? { groups: [] })),
    segmentId,
    fields: catalog.fields.map((f) => f.label),
    onDraft: (d) => {
      const built = d.groups
        .map((g) => {
          const conditions: Cond[] = []
          const removed: Cond[] = []
          const marks: number[] = []
          g.conditions.forEach((x) => {
            const c: Cond = { key: k(), field: x.field, op: x.op, value: x.value ?? '', to: x.to ?? '', amount: x.amount || 30, unit: x.unit === 'weeks' ? 'weeks' : 'days' }
            if (x.mark === 'removed') removed.push(c)
            else {
              if (x.mark === 'added') marks.push(c.key)
              conditions.push(c)
            }
          })
          return { group: { key: k(), match: g.match === 'any' ? ('any' as const) : ('all' as const), conditions }, removed, marks }
        })
        .filter((x) => x.group.conditions.length)
      setAi({
        draft: d,
        before: ai?.before ?? draft.groups,
        beforeCount: ai?.beforeCount ?? (untouched ? preview.data?.total : (preview.data?.count ?? detail?.count)) ?? 0,
        beforeEmpty: ai?.beforeEmpty ?? untouched,
        added: new Set(built.flatMap((x) => x.marks)),
        removed: built.map((x) => x.removed),
      })
      setDraft((dd) => ({ ...dd, groups: built.map((x) => x.group) }))
    },
  })
  const decide = (how: 'accepted' | 'accepted_edits' | 'rejected') => {
    if (!ai) return
    segAI.decide(ai.draft, how)
    if (how === 'rejected') setDraft((dd) => ({ ...dd, groups: ai.before }))
    setAi(null)
  }
  return (
    <Page
      title="Audience"
      documentTitle={draft.name || 'New segment'}
      filters={<AudienceTabs />}
      actions={
        <>
          <Button
            disabled={!dirty && segmentId !== 'new'}
            onClick={() => {
              if (segmentId === 'new' && !dirty) void navigate({ to: '/w/$workspace/audience/segments', params: { workspace: slug } })
              else {
                if (ai) decide('rejected')
                setDraft(initial)
              }
            }}
          >
            Discard
          </Button>
          <Button variant="primary" disabledReason={editReason} loading={create.isPending || update.isPending} onClick={() => void save().catch(() => undefined)}>
            Save segment
          </Button>
        </>
      }
      contentClassName="flex-row flex-wrap items-start"
    >
      <UnsavedChangesGuard dirty={dirty} description="You changed this segment since it was saved. Leaving now keeps the saved version." onSave={save} />
      <section className="flex min-w-0 flex-[2_1_600px] flex-col gap-4 segb" aria-label="Segment">
        {saveErr ? <Note tone="bad">{saveErr}</Note> : null}
        <div className="card flex flex-wrap items-center gap-3 px-4.5 py-4">
          <label className="flex flex-[1_1_300px] flex-col gap-1">
            <span className="k">Segment name</span>
            <input className="in segname" value={draft.name} placeholder="Name this segment" onChange={(e) => set({ name: e.target.value })} maxLength={120} />
          </label>
          <label className="flex flex-col gap-1">
            <span className="k">Refresh</span>
            <select className="in" value={draft.refresh} onChange={(e) => set({ refresh: e.target.value as Draft['refresh'] })}>
              <option value="dynamic">Dynamic · re-evaluated at send time</option>
              <option value="static">Static snapshot</option>
            </select>
          </label>
        </div>
        {draft.mode === 'builder' ? segAI.card : null}
        {draft.mode === 'builder' ? (
          <div className="card flex flex-col gap-3 p-4.5">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="h2">Conditions</h2>
              <span className="k">Groups combine with AND; inside a group pick ALL or ANY</span>
            </div>
            {ai ? (
              <DraftBanner
                draft={ai.draft}
                model={segAI.assist.model}
                onAccept={() => decide('accepted')}
                onEdit={() => decide('accepted_edits')}
                onReject={() => decide('rejected')}
                onRetry={segAI.retry}
              />
            ) : null}
            {draft.groups.map((g, gi) => (
              <div key={g.key} className="contents">
                {gi > 0 ? (
                  <div className="flex justify-center">
                    <span className="sop and">AND</span>
                  </div>
                ) : null}
                <div className="sgroup" role="group" aria-label={`Group ${gi + 1}`}>
                  <div className="flex flex-wrap items-center gap-2.5">
                    <span className="sop">Group {gi + 1}</span>
                    <select aria-label={`Match rule for Group ${gi + 1}`} className="in match" value={g.match} onChange={(e) => setGroup(gi, { match: e.target.value as Group['match'] })}>
                      {g.match === 'all' ? (
                        <>
                          <option value="all">Match ALL of these</option>
                          <option value="any">Match ANY of these</option>
                        </>
                      ) : (
                        <>
                          <option value="any">Match ANY of these</option>
                          <option value="all">Match ALL of these</option>
                        </>
                      )}
                    </select>
                    <Button
                      size="sm"
                      className="ml-auto"
                      aria-label="Remove group"
                      disabled={draft.groups.length === 1}
                      onClick={() => set({ groups: draft.groups.filter((_, i) => i !== gi) })}
                    >
                      Remove group
                    </Button>
                  </div>
                  {(ai?.removed[gi] ?? []).map((c) => (
                    <Condition key={c.key} c={c} catalog={catalog} invalid={false} onChange={() => undefined} onField={() => undefined} onRemove={() => undefined} canRemove={false} mark="removed" />
                  ))}
                  {g.conditions.map((c, ci) => (
                    <Condition
                      key={c.key}
                      mark={ai?.added.has(c.key) ? 'added' : undefined}
                      c={c}
                      catalog={catalog}
                      invalid={bad?.[0] === gi && bad?.[1] === ci}
                      onChange={(patch) => setCond(gi, ci, patch)}
                      onField={(key) => setCond(gi, ci, { ...blankCond(field(key)), key: c.key })}
                      onRemove={() =>
                        g.conditions.length === 1 && draft.groups.length > 1
                          ? set({ groups: draft.groups.filter((_, i) => i !== gi) })
                          : setGroup(gi, { conditions: g.conditions.filter((_, j) => j !== ci) })
                      }
                      canRemove={g.conditions.length > 1 || draft.groups.length > 1}
                    />
                  ))}
                  <Button size="sm" className="self-start" onClick={() => setGroup(gi, { conditions: [...g.conditions, blankCond(catalog.fields[0])] })}>
                    Add condition
                  </Button>
                </div>
              </div>
            ))}
            <div className="flex flex-wrap gap-2">
              <Button size="sm" onClick={() => set({ groups: [...draft.groups, { key: k(), match: 'any', conditions: [blankCond(catalog.fields[0])] }] })}>
                Add group
              </Button>
              <span className="k self-center">Suppressed, bounced and role addresses are always excluded at send time, whatever the segment says.</span>
            </div>
            {ai ? <DraftFooter draft={ai.draft} /> : null}
          </div>
        ) : (
          <div className="card flex flex-col gap-3 p-4.5">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="h2">SQL</h2>
              <span className="k">Read only, 10 s limit · views: {catalog.views.join(', ')}</span>
            </div>
            <SqlEditor value={draft.sql} onChange={(v) => set({ sql: v })} label="Segment SQL" invalid={preview.isError} />
            {preview.error instanceof ApiError && preview.error.code === 'sql_error' ? <Note tone="bad">{preview.error.message}</Note> : null}
            <div className="flex flex-wrap items-center gap-2">
              <Button size="sm" onClick={() => set({ mode: 'builder', sql: '' })}>
                Back to the builder
              </Button>
              <span className="k">The query must return contact ids (c.id). It runs read-only with a 10-second limit.</span>
            </div>
          </div>
        )}
        {draft.mode === 'builder' ? (
          <div className="card flex flex-col">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3">
              <h2 className="h2">Generated SQL</h2>
              <div className="flex gap-1.5 max-md:flex-wrap">
                <Badge tone="neutral">read only</Badge>
                <Button
                  size="sm"
                  onClick={() => {
                    void navigator.clipboard.writeText(generated).then(() => toast({ kind: 'success', title: 'Copied the SQL' }))
                  }}
                >
                  Copy
                </Button>
                <Button
                  size="sm"
                  title="Needs segments:sql"
                  disabledReason={!catalog.sqlEnabled ? 'SQL segments are turned off on this database; an admin can enable them.' : sqlReason}
                  onClick={() => set({ mode: 'sql', sql: generated })}
                >
                  Open in SQL editor
                </Button>
                <span className="k self-center">needs segments:sql</span>
              </div>
            </div>
            <div className="flex flex-col gap-3 px-4.5 py-3.5">
              {ai ? <DraftSqlNote /> : null}
              {generated ? <SqlBlock sql={generated} label="Generated SQL" /> : <Skeleton w="100%" h={120} />}
            </div>
          </div>
        ) : null}
      </section>
      <aside className="flex max-w-[400px] flex-[1_1_320px] flex-col gap-4" aria-label="Count and saved segments">
        <LiveCount
          preview={preview}
          name={draft.name}
          detail={detail}
          segmentId={segmentId}
          draftLabels={ai ? draftCountLabels(t, { count: ai.beforeCount, empty: ai.beforeEmpty }) : undefined}
        />
        <SavedSegments current={segmentId} />
      </aside>
    </Page>
  )
}

function Condition({
  c,
  catalog,
  invalid,
  onChange,
  onField,
  onRemove,
  canRemove,
  mark,
}: {
  c: Cond
  catalog: Catalog
  invalid: boolean
  onChange: (p: Partial<Cond>) => void
  onField: (key: string) => void
  onRemove: () => void
  canRemove: boolean
  /** A drafted condition (features/ai): added by the draft, or removed by it (struck through, read only). */
  mark?: 'added' | 'removed'
}) {
  const f = catalog.fields.find((x) => x.key === c.field)
  const noValue = ['never', 'is_empty', 'is_not_empty', 'is_true', 'is_false'].includes(c.op)
  if (mark === 'removed') {
    return (
      <fieldset disabled className="scond ai-removed m-0 min-w-0">
        <FitSelect aria-label="Condition field" className="in" value={c.field} label={f?.label ?? c.field} onChange={() => undefined}>
          {catalog.fields.map((x) => (
            <option key={x.key} value={x.key}>
              {x.label}
            </option>
          ))}
        </FitSelect>
        <select aria-label="Condition operator" className="in" value={c.op} onChange={() => undefined}>
          {(f?.ops ?? []).map((o) => (
            <option key={o.key} value={o.key}>
              {o.label}
            </option>
          ))}
        </select>
        {noValue ? null : f && (f.kind === 'enum' || f.kind === 'list' || f.kind === 'campaign') ? (
          <select aria-label="Condition value" className="in" value={c.value} onChange={() => undefined}>
            {f.options.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        ) : (
          <input className="in" aria-label="Value" value={f?.kind === 'period' ? `${c.amount} ${c.unit}` : c.value} readOnly />
        )}
        <DraftMark mark="removed" />
      </fieldset>
    )
  }
  return (
    <div className={`${invalid ? 'scond invalid' : 'scond'}${mark === 'added' ? ' ai-added' : ''}`} aria-invalid={invalid || undefined}>
      <FitSelect aria-label="Condition field" className="in" value={c.field} label={f?.label ?? c.field} onChange={(e) => onField(e.target.value)}>
        {catalog.fields.map((x) => (
          <option key={x.key} value={x.key}>
            {x.label}
          </option>
        ))}
      </FitSelect>
      <select aria-label="Condition operator" className="in" value={c.op} onChange={(e) => onChange({ op: e.target.value })}>
        {(f?.ops ?? []).map((o) => (
          <option key={o.key} value={o.key}>
            {o.label}
          </option>
        ))}
      </select>
      {noValue ? null : f?.kind === 'period' ? (
        <>
          <input className="in w-[70px]" inputMode="numeric" aria-label={c.unit === 'weeks' ? 'Weeks' : 'Days'} value={c.amount} onChange={(e) => onChange({ amount: Number(e.target.value.replace(/\D/g, '')) || 0 })} />
          <select aria-label="Time unit" className="in" value={c.unit} onChange={(e) => onChange({ unit: e.target.value as Cond['unit'] })}>
            <option value="days">days</option>
            <option value="weeks">weeks</option>
          </select>
        </>
      ) : f && (f.kind === 'enum' || f.kind === 'list' || f.kind === 'campaign') ? (
        <select aria-label="Condition value" className="in" value={c.value} onChange={(e) => onChange({ value: e.target.value })}>
          {f.options.length === 0 ? <option value="">Nothing to pick yet</option> : null}
          {f.options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      ) : (
        <input
          className="in"
          aria-label="Value"
          value={c.value}
          type={f?.kind === 'date' ? 'date' : 'text'}
          inputMode={f?.kind === 'number' ? 'decimal' : undefined}
          onChange={(e) => onChange({ value: e.target.value })}
        />
      )}
      {c.op === 'between' ? <BetweenTo value={c.to} onChange={(to) => onChange({ to })} /> : null}
      {mark === 'added' ? <DraftMark mark="added" /> : null}
      <Button size="sm" aria-label="Remove condition" disabled={!canRemove} onClick={onRemove}>
        ×
      </Button>
    </div>
  )
}

type PreviewQuery = { data?: Schemas['SegmentCountResult']; error: unknown; isPending: boolean; isFetching: boolean; refetch: () => unknown }

function LiveCount({
  preview,
  name,
  detail,
  segmentId,
  draftLabels,
}: {
  preview: PreviewQuery
  name: string
  detail?: Schemas['SegmentDetail']
  segmentId: string
  /** While an AI draft is open: "Live count · with the draft" and the count before it. */
  draftLabels?: { title: string; before: string }
}) {
  const { slug } = useWorkspace()
  const err = preview.error
  if (err instanceof ApiError && err.code === 'count_timeout') {
    const last = detail?.count
    return (
      <div role="alert" className="card flex flex-col gap-2 border-bad-line px-4.5 py-4">
        <div className="flex items-center gap-2.5 max-md:flex-wrap">
          <Badge tone="bad">Timed out</Badge>
          <span className="font-semibold">The live count for "{name || 'this segment'}" took longer than 10 s</span>
        </div>
        <span className="k">
          Segment counts over 40,000 contacts with event conditions can be slow on the default Postgres settings.
          {last !== undefined && last !== null && detail?.countedAt
            ? ` The last known count (${count(last)}, ${clock(detail.countedAt)} ${sameDay(detail.countedAt) ? 'today' : shortDate(detail.countedAt)}) is shown instead.`
            : ''}{' '}
          The segment still works at send time.
        </span>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="primary" onClick={() => void preview.refetch()}>
            Retry count
          </Button>
          <ButtonAnchor size="sm" href="https://github.com/hostrepute/reputecast/blob/main/docs/deploy/postgres-tuning.md">
            Postgres tuning guide
          </ButtonAnchor>
        </div>
      </div>
    )
  }
  const d = preview.data
  const invalid = err instanceof ApiError && err.status === 422
  const tiers: [Tier, number][] = d
    ? ([
        ['active', d.tiers.active],
        ['cooling', d.tiers.cooling],
        ['at_risk', d.tiers.atRisk],
        ['sunset', d.tiers.sunset],
      ] as [Tier, number][])
    : []
  const shown = tiers.filter(([, n]) => n > 0)
  const share = d && d.total > 0 ? (d.count / d.total) * 100 : 0
  return (
    <section className="card flex flex-col gap-2.5 border-accent-line px-5 py-4.5" aria-live="polite" aria-busy={preview.isFetching || undefined}>
      <span className="k">{draftLabels ? draftLabels.title : 'Live count · updates as you edit'}</span>
      {d ? (
        <>
          <span className="n text-[36px] leading-none tracking-[-0.02em]">{count(d.count)}</span>
          <span className="k">
            contacts match · {pct(share)} of {count(d.total)} · counted in {count(d.ms)} ms
          </span>
          {draftLabels ? <span className="k">{draftLabels.before}</span> : null}
          {d.count > 0 ? (
            <StackBar
              height={10}
              label={`Tier mix of the segment: ${shown.map(([t, n]) => `${tierLabel[t]} ${Math.round((n / d.count) * 100)} percent`).join(', ')}`}
              segments={shown.map(([t, n]) => ({ value: n, color: tierColor[t] }))}
            />
          ) : null}
          {shown.length ? (
            <div className="flex flex-wrap gap-3 text-[12px]">
              {shown.map(([t, n]) => (
                <span key={t} className="flex items-center gap-1.5">
                  <span className="dot" style={{ color: tierColor[t] }} aria-hidden="true" />
                  {tierLabel[t]} {count(n)}
                </span>
              ))}
            </div>
          ) : null}
        </>
      ) : invalid ? null : (
        <Loading label="Counting" className="flex flex-col gap-2">
          <Skeleton w={120} h={36} />
          <Skeleton w={240} h={12} />
          <Skeleton w="100%" h={10} r={5} />
        </Loading>
      )}
      {invalid ? <Note tone="warn">{(err as ApiError).message}</Note> : null}
      <div className="mt-1 flex flex-wrap gap-2">
        {segmentId !== 'new' ? (
          <ButtonLink size="sm" to="/w/$workspace/audience" params={{ workspace: slug }} search={{ segment: segmentId }}>
            Preview contacts
          </ButtonLink>
        ) : (
          <Button size="sm" disabledReason="Save the segment to preview its contacts.">
            Preview contacts
          </Button>
        )}
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
          Use in a campaign
        </ButtonLink>
      </div>
    </section>
  )
}

function sameDay(iso: string) {
  return shortDate(iso) === shortDate(new Date())
}

function SavedSegments({ current }: { current: string }) {
  const { slug } = useWorkspace()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } })
  const items = list.data?.items ?? []
  return (
    <section className="card flex flex-col px-4.5 py-4" aria-labelledby="saved-h">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="h2" id="saved-h">
          Saved segments
        </h2>
        <span className="k">{list.data ? count(items.length) : ''}</span>
      </div>
      {list.isPending ? (
        <Loading label="Loading segments" className="flex flex-col gap-2 py-2">
          {Array.from({ length: 5 }, (_, i) => (
            <Skeleton key={i} w="100%" h={14} />
          ))}
        </Loading>
      ) : (
        items.map((s, i) => (
          <div key={s.id} className={i ? 'segrow' : 'segrow first'}>
            <Link
              to="/w/$workspace/audience/segments/$segmentId"
              params={{ workspace: slug, segmentId: s.id }}
              className="font-medium text-ink no-underline"
              aria-current={s.id === current ? 'page' : undefined}
            >
              {s.name}
            </Link>
            <span className="n">{s.count !== undefined && s.count !== null ? count(s.count) : '–'}</span>
          </div>
        ))
      )}
      {current !== 'new' ? (
        <Link to="/w/$workspace/audience/segments/$segmentId" params={{ workspace: slug, segmentId: 'new' }} className="segrow font-medium no-underline">
          + New segment
        </Link>
      ) : null}
    </section>
  )
}

function BuilderSkeleton() {
  return (
    <Loading label="Loading the segment" className="flex flex-wrap items-start gap-5">
      <div className="flex min-w-0 flex-[2_1_600px] flex-col gap-4">
        <div className="card flex flex-col gap-2 px-4.5 py-4">
          <Skeleton w={120} h={12} />
          <Skeleton w="100%" h={38} />
        </div>
        <div className="card flex flex-col gap-3 p-4.5">
          <Skeleton w={140} h={16} />
          <Skeleton w="100%" h={120} r={10} />
          <Skeleton w="100%" h={90} r={10} />
        </div>
      </div>
      <div className="flex flex-[1_1_320px] flex-col gap-4 md:max-w-[400px]">
        <div className="card flex flex-col gap-2.5 px-5 py-4.5">
          <Skeleton w={160} h={12} />
          <Skeleton w={120} h={36} />
          <Skeleton w="100%" h={10} r={5} />
        </div>
      </div>
    </Loading>
  )
}
