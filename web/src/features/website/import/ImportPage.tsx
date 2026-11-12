import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton, Spinner } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { byteSize, clock, count, dayMonthDate, dayMonthYear, orList } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { FormsTabs } from '@/features/forms/common'
import { bare, useInvalidateWebsite, WS, type Overview } from '../api'
import { visibilityIcon } from '../postFormat'
import { m } from '../messages'
import { mi } from './messages'
import './import.css'

type Import = Schemas['WebsiteImport']
type Staged = Schemas['WebsiteImportPost']
type OptionsInput = Schemas['WebsiteImportOptionsInput']

const IMPORTS = `${WS}/imports` as const
const ONE = `${WS}/imports/{import}` as const
/** The most an export may weigh (the server refuses more). */
const MAX_BYTES = 100 * 1024 * 1024
const PAGE = 500

const sourceName = (s: string) => (s === 'ghost' ? 'Ghost' : s === 'wordpress' ? 'WordPress' : 'Substack')

/**
 * Website › Posts › Import posts (WebsiteImport and its phone board,
 * WebsiteImportStates): upload an export, review what it holds and the
 * choices, import, and the totals with the report.
 */
export function ImportPage({ overview, importId, onImport }: { overview: Overview; importId?: string; onImport: (id: string | undefined) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const live = overview.status === 'on'
  const q = $api.useQuery(
    'get',
    ONE,
    { params: { path: { workspace: slug, import: importId ?? '' }, query: { limit: PAGE } } },
    {
      enabled: Boolean(importId),
      refetchInterval: (s) => {
        const st = s.state.data?.status
        return st === 'uploaded' || st === 'analysing' || st === 'importing' ? 1500 : false
      },
    },
  )
  const imp = importId ? q.data : undefined
  let body: ReactNode
  if (!live) {
    body = <SiteOff overview={overview} />
  } else if (!importId) {
    body = <Upload onUploaded={(i) => onImport(i.id)} />
  } else if (q.isError && !q.data) {
    const missing = q.error instanceof ApiError && q.error.status === 404
    body = (
      <ErrorPanel
        title={t(missing ? mi.notFoundTitle : mi.loadFailed)}
        actions={
          missing ? (
            <Button onClick={() => onImport(undefined)}>{t(mi.uploadAnother)}</Button>
          ) : (
            <Button onClick={() => void q.refetch()}>{t(mi.retry)}</Button>
          )
        }
      >
        {missing ? t(mi.notFoundText) : errorMessage(q.error)}
      </ErrorPanel>
    )
  } else if (!imp) {
    body = <ReviewSkeleton />
  } else if (imp.status === 'uploaded' || imp.status === 'analysing') {
    body = <Reading imp={imp} />
  } else if (imp.status === 'ready') {
    body = <Review imp={imp} overview={overview} onAnother={() => onImport(undefined)} />
  } else if (imp.status === 'importing') {
    body = <Importing imp={imp} />
  } else if (imp.status === 'failed') {
    body = <Failed imp={imp} onAnother={() => onImport(undefined)} />
  } else {
    body = <Done imp={imp} />
  }
  const step = !importId || !imp ? (importId ? 2 : 1) : imp.status === 'uploaded' || imp.status === 'analysing' || imp.status === 'ready' ? 2 : imp.status === 'done' ? 4 : 3
  return (
    <Page title={t(m.screenTitle)} documentTitle={t(mi.documentTitle)} filters={<FormsTabs />}>
      <div className="wi-main">
        <div className="wi-head">
          <div className="tt">
            <Link to="/w/$workspace/forms/website" params={{ workspace: slug }} className="lnk wi-back">
              <Icon name="chevronLeft" size={14} />
              {t(mi.back)}
            </Link>
            <h2>{t(mi.title)}</h2>
            <span className="lead">{t(mi.lead)}</span>
          </div>
          {live && importId ? <Trail step={step} /> : null}
        </div>
        {body}
      </div>
    </Page>
  )
}

/** Upload › Review › Import, with the steps done ticked. */
function Trail({ step }: { step: number }) {
  const t = useT()
  const steps = [t(mi.stepUpload), t(mi.stepReview), t(mi.stepImport)]
  return (
    <ol className="wi-trail" aria-label={t(mi.steps)}>
      {steps.map((label, i) => {
        const n = i + 1
        const done = n < step
        const on = n === step
        return (
          <li key={label} className={done ? 'done' : on ? 'on' : undefined} aria-current={on ? 'step' : undefined}>
            <span className="n">{done ? <Icon name="check" size={12} /> : n}</span>
            {done ? <span className="vh">{t(mi.stepDone)} </span> : null}
            {label}
            {n < 3 ? (
              <span className="sep" aria-hidden="true">
                ›
              </span>
            ) : null}
          </li>
        )
      })}
    </ol>
  )
}

function Upload({ onUploaded }: { onUploaded: (i: Import) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canPublish = useCan('website:publish')
  const input = useRef<HTMLInputElement>(null)
  const [over, setOver] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const [refused, setRefused] = useState<{ kind: 'large' | 'wrong' | 'other'; name: string; size: number; detail: string } | null>(null)
  const send = async (file: File) => {
    setRefused(null)
    if (file.size > MAX_BYTES) {
      setRefused({ kind: 'large', name: file.name, size: file.size, detail: '' })
      return
    }
    setBusy(file.name)
    try {
      const { data } = await http.POST(IMPORTS, {
        params: { path: { workspace: slug }, query: { fileName: file.name } },
        body: file as unknown as string,
        bodySerializer: (b: unknown) => b as BodyInit,
        headers: { 'Content-Type': 'application/octet-stream' },
      })
      if (data) onUploaded(data)
    } catch (e) {
      const wrong = e instanceof ApiError && e.status === 422
      setRefused({ kind: wrong ? 'wrong' : 'other', name: file.name, size: file.size, detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const pick = (files: FileList | null) => {
    const f = files?.[0]
    if (f) void send(f)
  }
  const reason = canPublish ? undefined : t(mi.needsPublish)
  return (
    <section className="card wi-card" aria-labelledby="wi-up-h">
      <div className="wi-bar">
        <h3 id="wi-up-h">{t(mi.uploadTitle)}</h3>
        <Trail step={1} />
      </div>
      {refused?.kind === 'wrong' ? (
        <Note tone="bad">
          <b>{t(mi.notExportTitle)}</b>
          <br />
          <T msg={mi.notExportText} values={{ name: refused.name, detail: refused.detail, b: (c: ReactNode) => <b className="wi-mono">{c}</b> }} />
        </Note>
      ) : refused?.kind === 'large' ? (
        <Note tone="bad">
          <b>{t(mi.tooLargeTitle, { name: refused.name, size: byteSize(refused.size) })}</b>
          <br />
          {t(mi.tooLargeText)}
        </Note>
      ) : refused ? (
        <Note tone="bad">
          <b>{t(mi.uploadFailed)}</b> {refused.detail}
        </Note>
      ) : null}
      <div
        className={over ? 'wi-drop over' : 'wi-drop'}
        onDragOver={(e) => {
          if (!canPublish) return
          e.preventDefault()
          setOver(true)
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setOver(false)
          if (canPublish) pick(e.dataTransfer.files)
        }}
      >
        <span className="ic" aria-hidden="true">
          <Icon name="upload" size={20} />
        </span>
        <b style={{ fontSize: 15 }}>{busy ? t(mi.uploading, { name: busy }) : t(mi.dropTitle)}</b>
        <span className="ds">{refused ? t(mi.dropTextShort) : t(mi.dropText)}</span>
        <input ref={input} type="file" className="vh" accept=".zip,.json,.xml,application/zip,application/json,application/xml,text/xml" aria-label={t(mi.fileLabel)} tabIndex={-1} onChange={(e) => pick(e.target.files)} />
        <Button variant="primary" loading={Boolean(busy)} disabledReason={reason} onClick={() => input.current?.click()}>
          {refused ? t(mi.chooseAnother) : t(mi.choose)}
        </Button>
      </div>
      {refused?.kind === 'wrong' ? (
        <span className="k">
          <T msg={mi.movingSubscribers} values={{ link: (c: ReactNode) => <Link to="/w/$workspace/import" params={{ workspace: slug }}>{c}</Link> }} />
        </span>
      ) : (
        <>
          <h4 className="wi-hd3">{t(mi.whereTitle)}</h4>
          <div className="wi-srcs">
            {(
              [
                ['Substack', mi.whereSubstack],
                ['Ghost', mi.whereGhost],
                ['WordPress', mi.whereWordPress],
              ] as const
            ).map(([name, msg]) => (
              <div key={name}>
                <b>{name}</b>
                <span className="k">{t(msg)}</span>
              </div>
            ))}
          </div>
          <span className="k">
            <T msg={mi.onlyPosts} values={{ link: (c: ReactNode) => <Link to="/w/$workspace/import" params={{ workspace: slug }}>{c}</Link> }} />
          </span>
        </>
      )}
    </section>
  )
}

/** "acme-weekly-substack-export.zip · Substack export · 4.2 MB · uploaded by …". */
function FileCard({ imp, actions, read }: { imp: Import; actions?: ReactNode; read?: boolean }) {
  const t = useT()
  const when = `${dayMonthDate(imp.createdAt)}, ${clock(imp.createdAt)}`
  const seconds = Math.max(1, Math.round((Date.parse(imp.updatedAt) - Date.parse(imp.createdAt)) / 1000))
  const args = { source: sourceName(imp.source), size: byteSize(imp.fileSize), name: imp.createdBy, date: when, seconds }
  return (
    <section className="card" aria-label={t(mi.exportLabel)}>
      <div className="pad">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2.5">
          <span className="flex min-w-0 flex-[1_1_300px] items-center gap-3">
            <Icon name="file" size={20} />
            <span className="ws-tline">
              <b style={{ overflowWrap: 'anywhere' }}>{imp.fileName}</b>
              <span className="k">{read ? t(mi.fileLineRead, args) : t(mi.fileLine, args)}</span>
            </span>
          </span>
          {actions ? <span className="flex flex-wrap items-center gap-2">{actions}</span> : null}
        </div>
      </div>
    </section>
  )
}

function useCancel(imp: Import) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [busy, setBusy] = useState(false)
  const run = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST(`${ONE}/cancel`, { params: { path: { workspace: slug, import: imp.id } } })
      await qc.invalidateQueries({ queryKey: ['get', ONE] })
      toast({ title: t(mi.canceled) })
      if (data && imp.status !== 'importing') void navigate({ to: '/w/$workspace/forms/website', params: { workspace: slug } })
    } catch (e) {
      toast({ kind: 'error', title: t(mi.cancelFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return { busy, run }
}

function Reading({ imp }: { imp: Import }) {
  const t = useT()
  return (
    <>
      <FileCard imp={imp} />
      <section className="card wi-card" aria-labelledby="wi-read-h">
        <h3 id="wi-read-h">{t(mi.readingTitle)}</h3>
        <ul className="wi-steps" aria-live="polite">
          <li className={imp.status === 'analysing' ? 'done' : undefined}>
            <span className="ic">{imp.status === 'analysing' ? <Icon name="check" size={14} /> : <Spinner size={14} />}</span>
            {t(mi.readingOpened)}
          </li>
          <li className={imp.status === 'analysing' ? undefined : 'todo'}>
            <span className="ic">{imp.status === 'analysing' ? <Spinner size={14} /> : null}</span>
            {t(mi.readingChecking)}
          </li>
          <li className="todo">
            <span className="ic" />
            {t(mi.readingMapping)}
          </li>
        </ul>
        <Loading label={t(mi.readingTitle)} className="wi-skel">
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i}>
              <Skeleton w="70%" />
              <Skeleton w="45%" h={18} />
            </div>
          ))}
        </Loading>
        <span className="k">{t(mi.readingNote)}</span>
      </section>
    </>
  )
}

function ReviewSkeleton() {
  const t = useT()
  return (
    <Loading label={t(mi.loading)} className="wi-main">
      <div className="card pad">
        <Skeleton w={320} h={16} />
      </div>
      <div className="ws-stats">
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="card ws-stat">
            <Skeleton w={90} />
            <Skeleton w={40} h={22} />
            <Skeleton w={140} />
          </div>
        ))}
      </div>
      <div className="card pad">
        <Skeleton w="100%" h={320} />
      </div>
    </Loading>
  )
}

/** A staged post's date ("12 Mar 2026") or Draft. */
function postDate(p: Staged, t: ReturnType<typeof useT>) {
  return p.status === 'draft' || !p.publishedAt ? t(mi.draft) : dayMonthYear(p.publishedAt)
}

function Review({ imp, overview, onAnother }: { imp: Import; overview: Overview; onAnother: () => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const qc = useQueryClient()
  const invalidate = useInvalidateWebsite()
  const canPublish = useCan('website:publish')
  const phone = useMediaQuery('(max-width: 767px)')
  const cancel = useCancel(imp)
  const [filter, setFilter] = useState<'all' | 'problems' | 'skipped'>('all')
  const [starting, setStarting] = useState(false)
  const host = bare(overview.address.url)
  const source = sourceName(imp.source)
  const c = imp.counts
  const key = ['get', ONE, { params: { path: { workspace: slug, import: imp.id }, query: { limit: PAGE } } }]
  const save = async (body: OptionsInput) => {
    try {
      const { data } = await http.PUT(ONE, { params: { path: { workspace: slug, import: imp.id } }, body })
      if (data) qc.setQueryData(key, data)
    } catch (e) {
      toast({ kind: 'error', title: t(mi.saveFailed), detail: errorMessage(e) })
      void qc.invalidateQueries({ queryKey: ['get', ONE] })
    }
  }
  const start = async () => {
    setStarting(true)
    try {
      const { data } = await http.POST(`${ONE}/start`, { params: { path: { workspace: slug, import: imp.id } } })
      if (data) qc.setQueryData(key, data)
      void invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mi.startFailed), detail: errorMessage(e) })
    } finally {
      setStarting(false)
    }
  }
  const total = c.toImport + c.toUpdate
  const withProblems = imp.posts.filter((p) => p.problems.some((x) => x.code !== 'already_imported'))
  const skipped = imp.posts.filter((p) => p.decision === 'skip')
  const shown = filter === 'problems' ? withProblems : filter === 'skipped' ? skipped : imp.posts
  const example = imp.posts.find((p) => p.status === 'published')?.sourceUrl ?? imp.posts[0]?.sourceUrl ?? ''
  const examplePath = example.replace(/^https?:\/\/[^/]+/, '')
  const paidPosts = imp.tiers.reduce((n, x) => n + x.posts, 0)
  const reason = canPublish ? undefined : t(mi.needsPublish)
  const nothing = total === 0
  const helpers = overview.helpers.publish
  return (
    <>
      <FileCard
        imp={imp}
        read
        actions={
          <>
            {nothing ? null : (
              <Badge tone="ok" dot>
                {t(mi.ready)}
              </Badge>
            )}
            <Button size="sm" loading={cancel.busy} disabledReason={reason} onClick={() => void cancel.run()}>
              {t(mi.cancelImport)}
            </Button>
          </>
        }
      />
      {!canPublish ? (
        <section className="ws-ro" role="status">
          <div className="hd">
            <Icon name="info" size={15} />
            {t(mi.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}
          </div>
          <div className="bd">{helpers.length ? t(mi.readOnlyText, { helpers: orList(helpers) }) : t(mi.readOnlyNoHelpers)}</div>
        </section>
      ) : null}
      {nothing ? (
        <section className="card wi-card" aria-labelledby="wi-nothing-h">
          <h3 id="wi-nothing-h">{t(mi.nothingTitle)}</h3>
          <span>
            {c.alreadyImported > 0 ? (
              <T msg={mi.nothingText} values={{ count: c.found - c.drafts, file: imp.fileName, drafts: c.draftsSkipped, b: (x: ReactNode) => <b>{x}</b> }} />
            ) : (
              <T msg={mi.nothingTextShort} values={{ file: imp.fileName, b: (x: ReactNode) => <b>{x}</b> }} />
            )}
          </span>
          <div>
            <Button onClick={onAnother}>{t(mi.uploadAnother)}</Button>
          </div>
        </section>
      ) : null}
      <div className="ws-stats" role="group" aria-label={t(mi.statsLabel)}>
        <Stat k={t(mi.statToImport)} v={total} s={t(mi.statToImportSub, { source })} />
        <Stat k={t(mi.statDrafts)} v={c.draftsSkipped} s={t(mi.statDraftsSub)} />
        <Stat k={t(mi.statAlready)} v={c.alreadyImported} s={t(mi.statAlreadySub, { source })} />
        <Stat k={t(mi.statPaid)} v={c.paid} s={t(mi.statPaidSub)} />
        <Stat k={t(mi.statMembers)} v={c.members} s={t(mi.statMembersSub)} />
      </div>
      <div className="wi-grid2">
        <div className="wi-col">
          <section className="card">
            <div className="sec">
              <h3 className="h2">{t(mi.datesTitle)}</h3>
            </div>
            <fieldset className="pad flex flex-col gap-2" style={{ border: 0, margin: 0 }}>
              <legend className="vh">{t(mi.datesTitle)}</legend>
              <Radio name="wi-pub" on={imp.options.publish === 'as_published'} disabled={!canPublish} onPick={() => void save({ publish: 'as_published' })} title={t(mi.keepDates)} text={t(mi.keepDatesText, { host, source })} />
              <Radio name="wi-pub" on={imp.options.publish === 'as_drafts'} disabled={!canPublish} onPick={() => void save({ publish: 'as_drafts' })} title={t(mi.asDrafts)} text={t(mi.asDraftsText)} />
            </fieldset>
          </section>
          <section className="card">
            <div className="sec">
              <h3 className="h2">{t(mi.imagesTitle)}</h3>
            </div>
            <fieldset className="pad flex flex-col gap-2" style={{ border: 0, margin: 0 }}>
              <legend className="vh">{t(mi.imagesTitle)}</legend>
              <Radio name="wi-img" on={imp.options.images === 'copy'} disabled={!canPublish} onPick={() => void save({ images: 'copy' })} title={t(mi.copyImages)} text={t(mi.copyImagesText, { count: c.images })} />
              <Radio
                name="wi-img"
                on={imp.options.images === 'link'}
                disabled={!canPublish}
                onPick={() => void save({ images: 'link' })}
                title={t(mi.linkImages)}
                text={imp.source === 'substack' ? t(mi.linkImagesFrom, { host: 'substackcdn.com' }) : t(mi.linkImagesText)}
              />
            </fieldset>
          </section>
        </div>
        <div className="wi-col">
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h3 className="h2">{t(mi.tiersTitle)}</h3>
                <span className="k">{imp.tiers.length ? t(mi.tiersText, { count: paidPosts, host }) : t(mi.noTiersText)}</span>
              </div>
            </div>
            {imp.tiers.length ? (
              <div className="pad flex flex-col">
                {imp.tiers.map((x) => (
                  <div key={x.ref} className="wi-maprow">
                    <span className="ws-tline">
                      <b>{x.ref}</b>
                      <span className="k">{t(mi.tierPosts, { source, count: x.posts })}</span>
                    </span>
                    <span className="arw" aria-hidden="true">
                      →
                    </span>
                    <select className="in" aria-label={t(mi.tierFor, { name: x.ref })} value={x.tierId} disabled={!canPublish} onChange={(e) => void save({ tiers: { [x.ref]: e.target.value } })}>
                      {imp.paidTiers.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name}
                        </option>
                      ))}
                      <option value="any">{t(mi.anyTier)}</option>
                    </select>
                  </div>
                ))}
              </div>
            ) : null}
          </section>
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h3 className="h2">{t(mi.authorsTitle)}</h3>
                <span className="k">{imp.authors.length ? t(mi.authorsText) : t(mi.noAuthorsText)}</span>
              </div>
            </div>
            {imp.authors.length ? (
              <div className="pad flex flex-col">
                {imp.authors.map((a) => (
                  <div key={a.name} className="wi-maprow">
                    <span className="ws-tline">
                      <b>{a.name}</b>
                      <span className="k">{t(mi.authorPosts, { count: a.posts })}</span>
                    </span>
                    <span className="arw" aria-hidden="true">
                      →
                    </span>
                    <select className="in" aria-label={t(mi.bylineFor, { name: a.name })} value={a.userId} disabled={!canPublish} onChange={(e) => void save({ author: { [a.name]: e.target.value } })}>
                      <option value="keep">{t(mi.keepName)}</option>
                      {imp.team.map((m) => (
                        <option key={m.id} value={m.id}>
                          {t(mi.teamMember, { name: m.name })}
                        </option>
                      ))}
                    </select>
                  </div>
                ))}
              </div>
            ) : null}
          </section>
        </div>
      </div>
      <Note tone="neutral">
        <b>{t(mi.oldTitle)}</b>
        <br />
        <T msg={mi.oldText} values={{ host, source, path: examplePath || '/p/…', mono: (x: ReactNode) => <span className="wi-mono">{x}</span> }} />
      </Note>
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h3 className="h2">{t(mi.postsTitle, { count: c.found })}</h3>
            <span className="k">{t(mi.postsText)}</span>
          </div>
          <select className="in" style={{ width: 'auto' }} aria-label={t(mi.show)} value={filter} onChange={(e) => setFilter(e.target.value as typeof filter)}>
            <option value="all">{t(mi.showAll, { count: c.found })}</option>
            <option value="problems">{t(mi.showProblems, { count: withProblems.length })}</option>
            <option value="skipped">{t(mi.showSkipped, { count: skipped.length })}</option>
          </select>
        </div>
        {shown.length === 0 ? (
          <div className="pad k">{t(mi.noMatch)}</div>
        ) : phone ? (
          <div className="wi-list">
            {shown.map((p) => (
              <div key={p.id} className="pi">
                <b>{p.title}</b>
                <div className="pm">
                  <span className="ws-vis">
                    <Icon name={visibilityIcon(p.visibility)} size={14} />
                    {visibilityWord(t, p.visibility)}
                  </span>
                  {p.visibility === 'paid' && p.tierRefs.length ? <span>{p.tierRefs.join(', ')}</span> : null}
                  <span>{postDate(p, t)}</span>
                  {p.author ? <span>{p.author}</span> : null}
                  {p.tags.map((x) => (
                    <span key={x}>{x}</span>
                  ))}
                </div>
                {p.status === 'draft' || p.alreadyImported || p.problems.length ? <Problems p={p} /> : null}
                <label className="lbl">
                  {t(mi.decision)}
                  <Decision p={p} disabled={!canPublish} onChange={(d) => void save({ decisions: [{ id: p.id, decision: d }] })} full />
                </label>
              </div>
            ))}
          </div>
        ) : (
          <TableScroll label={t(mi.tableLabel)}>
            <table className="tbl wi-tbl" aria-label={t(mi.tableLabel)}>
              <thead>
                <tr>
                  <th scope="col">{t(mi.colPost)}</th>
                  <th scope="col">{t(mi.colDate)}</th>
                  <th scope="col">{t(mi.colVisibility)}</th>
                  <th scope="col">{t(mi.colTags)}</th>
                  <th scope="col">{t(mi.colProblems)}</th>
                  <th scope="col">{t(mi.colDecision)}</th>
                </tr>
              </thead>
              <tbody>
                {shown.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <div className="pt2">
                        <b>{p.title}</b>
                        <span className="k">
                          {p.author ? `${t(mi.byline, { name: p.author })} · ` : null}
                          <span className="wi-mono">{p.sourceUrl.replace(/^https?:\/\/[^/]+/, '')}</span>
                        </span>
                      </div>
                    </td>
                    <td className="nw">{postDate(p, t)}</td>
                    <td>
                      <VisibilityText p={p} />
                    </td>
                    <td>
                      {p.tags.length ? (
                        <div className="ws-tags" style={{ marginTop: 0 }}>
                          {p.tags.map((x) => (
                            <span key={x} className="ws-tagc">
                              {x}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="k">{t(mi.noTags)}</span>
                      )}
                    </td>
                    <td>
                      <Problems p={p} />
                    </td>
                    <td>
                      <Decision p={p} disabled={!canPublish} onChange={(d) => void save({ decisions: [{ id: p.id, decision: d }] })} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        )}
        <div className="wi-foot">
          <span className="k">{t(mi.postsFoot, { count: c.found, problems: c.withProblems })}</span>
        </div>
      </section>
      <div className="wi-bar">
        <span className="k">{t(mi.neverEmail)}</span>
        <div className="flex flex-wrap gap-2">
          <ButtonLink to="/w/$workspace/forms/website" params={{ workspace: slug }}>
            {t(mi.backToPosts)}
          </ButtonLink>
          <Button
            variant="primary"
            loading={starting}
            disabledReason={reason ?? (nothing ? t(mi.nothingTitle) : undefined)}
            onClick={() => void start()}
          >
            {t(mi.importN, { count: total })}
          </Button>
        </div>
      </div>
    </>
  )
}

function Stat({ k, v, s }: { k: string; v: number; s: string }) {
  return (
    <div className="card ws-stat">
      <span className="k">{k}</span>
      <span className="v">{count(v)}</span>
      <span className="s">{s}</span>
    </div>
  )
}

function Radio({ name, on, title, text, onPick, disabled }: { name: string; on: boolean; title: string; text: string; onPick: () => void; disabled?: boolean }) {
  return (
    // eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its radio; the title and text inside are its text
    <label className={on ? 'ws-radio on' : disabled ? 'ws-radio off' : 'ws-radio'}>
      <input type="radio" name={name} checked={on} disabled={disabled} onChange={onPick} />
      <span className="ws-tline">
        <b>{title}</b>
        <span className="k2">{text}</span>
      </span>
    </label>
  )
}

function VisibilityText({ p }: { p: Staged }) {
  const t = useT()
  const label = p.visibility === 'paid' ? t(mi.paid) : p.visibility === 'members' ? t(mi.members) : t(mi.public)
  return (
    <span className="ws-stk">
      <span className="ws-vis">
        <Icon name={visibilityIcon(p.visibility)} size={14} />
        {label}
      </span>
      {p.visibility === 'paid' && p.tierRefs.length ? <span className="k">{p.tierRefs.join(', ')}</span> : null}
    </span>
  )
}

function Problems({ p }: { p: Staged }) {
  const t = useT()
  const list = p.problems.filter((x) => x.code !== 'already_imported')
  if (p.status === 'draft' && list.length === 0) {
    return (
      <div className="wi-probs">
        <Badge tone="neutral">{t(mi.draft)}</Badge>
      </div>
    )
  }
  if (p.alreadyImported) {
    return (
      <div className="wi-probs">
        <Badge tone="info">{t(mi.importedBefore)}</Badge>
      </div>
    )
  }
  if (list.length === 0) return <span className="k">{t(mi.none)}</span>
  return (
    <div className="wi-probs">
      {list.map((x, i) => (
        <span key={`${x.code}-${i}`} className="contents">
          <Badge tone={x.code === 'slug_taken' ? 'info' : x.code === 'convert_failed' || x.code === 'image_failed' ? 'bad' : 'warn'}>{x.text}</Badge>
          {x.note && (x.code === 'slug_taken' || x.code === 'extra_tags') ? <span className="k">{x.note}</span> : null}
        </span>
      ))}
    </div>
  )
}

function Decision({ p, onChange, disabled, full }: { p: Staged; onChange: (d: 'import' | 'skip' | 'update') => void; disabled: boolean; full?: boolean }) {
  const t = useT()
  return (
    <select className="in" style={{ width: full ? '100%' : 110 }} aria-label={t(mi.decisionFor, { title: p.title })} value={p.decision} disabled={disabled} onChange={(e) => onChange(e.target.value as 'import' | 'skip' | 'update')}>
      {p.alreadyImported ? <option value="update">{t(mi.update)}</option> : <option value="import">{t(mi.import)}</option>}
      <option value="skip">{t(mi.skip)}</option>
    </select>
  )
}

function eta(t: ReturnType<typeof useT>, imp: Import) {
  const p = imp.progress
  if (!p?.startedAt || p.done <= 0 || p.total <= p.done) return ''
  const left = ((Date.now() - Date.parse(p.startedAt)) * (p.total - p.done)) / p.done
  return left < 60_000 ? t(mi.etaSoon) : t(mi.eta, { minutes: Math.ceil(left / 60_000) })
}

function Importing({ imp }: { imp: Import }) {
  const t = useT()
  const cancel = useCancel(imp)
  const canPublish = useCan('website:publish')
  const p = imp.progress ?? { done: 0, total: 0, imagesCopied: 0, imagesFailed: 0, published: 0, drafts: 0, updated: 0, failed: 0, skipped: 0 }
  const pct = p.total ? Math.min(99, Math.floor((p.done * 100) / p.total)) : 0
  const ran = imp.posts.filter((x) => x.decision !== 'skip' && x.outcome)
  const left = imp.posts.filter((x) => x.decision !== 'skip' && !x.outcome)
  const recent = ran.slice(-2)
  const current = left[0]
  const later = Math.max(0, left.length - 1)
  const images = imp.options.images === 'copy' ? imp.counts.images : 0
  const e = eta(t, imp)
  return (
    <>
      <FileCard imp={imp} />
      <section className="card wi-card" aria-labelledby="wi-run-h">
        <div className="wi-bar">
          <h3 id="wi-run-h">{t(mi.importingTitle, { count: p.total })}</h3>
          <Trail step={3} />
        </div>
        <div className="ws-prog">
          <div className="hd">
            <b>
              <Spinner size={14} />
              {t(mi.importedOf, { done: p.done, total: p.total })}
            </b>
            <span className="k num">{pct}%</span>
          </div>
          <div className="ws-track" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct} aria-label={t(mi.importedOf, { done: p.done, total: p.total })}>
            <span style={{ width: `${pct}%` }} />
          </div>
          <span className="k">
            {images > 0
              ? e
                ? t(mi.imagesProgress, { copied: p.imagesCopied + p.imagesFailed, total: images, eta: e })
                : t(mi.imagesProgressNoEta, { copied: p.imagesCopied + p.imagesFailed, total: images })
              : e || t(mi.leaveNote)}
          </span>
        </div>
        <ul className="wi-steps" aria-live="polite">
          {recent.map((x) => (
            <li key={x.id} className="done">
              <span className="ic">
                <Icon name="check" size={14} />
              </span>
              <span>
                {x.title}
                <span className="sub">{outcomeLine(t, x, imp)}</span>
              </span>
            </li>
          ))}
          {current ? (
            <li>
              <span className="ic">
                <Spinner size={14} />
              </span>
              <span>
                {current.title}
                <span className="sub">
                  {current.images > 0
                    ? t(mi.inProgress, { visibility: visibilityWord(t, current.visibility), count: current.images })
                    : t(mi.inProgressNoImages, { visibility: visibilityWord(t, current.visibility) })}
                </span>
              </span>
            </li>
          ) : null}
          {later > 0 ? (
            <li className="todo">
              <span className="ic" />
              {t(mi.morePending, { count: later })}
            </li>
          ) : null}
        </ul>
        <div className="wi-bar">
          <span className="k">{t(mi.nothingEmailed)}</span>
          <Button size="sm" loading={cancel.busy} disabledReason={canPublish ? undefined : t(mi.needsPublish)} onClick={() => void cancel.run()}>
            {t(mi.cancelImport)}
          </Button>
        </div>
      </section>
    </>
  )
}

function visibilityWord(t: ReturnType<typeof useT>, v: string) {
  return v === 'paid' ? t(mi.paid) : v === 'members' ? t(mi.members) : t(mi.public)
}

function outcomeLine(t: ReturnType<typeof useT>, x: Staged, imp: Import) {
  const moved = x.problems.find((p) => p.code === 'slug_taken')
  const date = x.publishedAt ? dayMonthYear(x.publishedAt) : ''
  const kept = Boolean(x.author) && imp.authors.find((a) => a.name === x.author)?.userId === 'keep'
  switch (x.outcome) {
    case 'published':
      return moved ? t(mi.importedAs, { slug: x.slug, date }) : kept ? t(mi.bylineAsWritten, { name: x.author }) : t(mi.importedLive, { date })
    case 'updated':
      return t(mi.importedUpdated)
    case 'failed':
      return t(mi.importedFailed)
    default:
      return t(mi.importedDraft)
  }
}

function reportHref(slug: string, imp: Import) {
  return `/api/v1/workspaces/${encodeURIComponent(slug)}/website/imports/${imp.id}/report.csv`
}

function Done({ imp }: { imp: Import }) {
  const t = useT()
  const { slug } = useWorkspace()
  const source = sourceName(imp.source)
  const p = imp.progress ?? { done: 0, total: 0, imagesCopied: 0, imagesFailed: 0, published: 0, drafts: 0, updated: 0, failed: 0, skipped: 0 }
  const ran = imp.posts.filter((x) => x.outcome && x.outcome !== 'skipped' && x.outcome !== 'failed')
  const published = ran.filter((x) => x.outcome === 'published' && x.publishedAt).map((x) => x.publishedAt!).sort()
  const summary = summarize(imp, ran)
  const finished = imp.finishedAt ?? imp.updatedAt
  const count_ = p.published + p.drafts + p.updated
  return (
    <section className="card wi-card" aria-labelledby="wi-done-h">
      <div className="wi-bar">
        <div className="flex items-center gap-3">
          <span className="wi-done-ic" aria-hidden="true">
            <Icon name={imp.status === 'canceled' ? 'stop' : 'check'} size={18} />
          </span>
          <div className="ws-tline">
            <h3 id="wi-done-h">{imp.status === 'canceled' ? t(mi.canceledTitle, { source }) : t(mi.doneTitle, { count: count_, source })}</h3>
            <span className="k">{t(mi.doneLine, { date: `${dayMonthDate(finished)}, ${clock(finished)}`, name: imp.createdBy })}</span>
          </div>
        </div>
        <Trail step={4} />
      </div>
      <dl className="wi-kv" aria-label={t(mi.totals)}>
        <dt>{t(mi.kvPublished)}</dt>
        <dd>
          {published.length === 0
            ? t(mi.kvPublishedNone)
            : published.length === 1
              ? t(mi.kvPublishedOne, { count: 1, source, date: dayMonthYear(published[0]!) })
              : t(mi.kvPublishedValue, { count: published.length, source, from: dayMonthYear(published[0]!), to: dayMonthYear(published[published.length - 1]!) })}
        </dd>
        {p.drafts > 0 ? (
          <>
            <dt>{t(mi.kvDrafts)}</dt>
            <dd>{t(mi.kvDraftsValue, { count: p.drafts })}</dd>
          </>
        ) : null}
        {p.updated > 0 ? (
          <>
            <dt>{t(mi.kvUpdated)}</dt>
            <dd>{count(p.updated)}</dd>
          </>
        ) : null}
        <dt>{t(mi.kvSkipped)}</dt>
        <dd>{imp.counts.draftsSkipped === p.skipped && p.skipped > 0 ? t(mi.kvSkippedDrafts, { count: p.skipped }) : t(mi.kvSkippedValue, { count: p.skipped })}</dd>
        {p.failed > 0 ? (
          <>
            <dt>{t(mi.kvFailed)}</dt>
            <dd>{t(mi.kvFailedValue, { count: p.failed })}</dd>
          </>
        ) : null}
        <dt>{t(mi.kvAccess)}</dt>
        <dd>{summary.access}</dd>
        {summary.bylines ? (
          <>
            <dt>{t(mi.kvBylines)}</dt>
            <dd>{summary.bylines}</dd>
          </>
        ) : null}
        <dt>{t(mi.kvImages)}</dt>
        <dd>{imp.options.images === 'link' ? t(mi.kvImagesLinked) : p.imagesFailed > 0 ? t(mi.kvImagesFailed, { count: p.imagesCopied, failed: p.imagesFailed }) : t(mi.kvImagesValue, { count: p.imagesCopied })}</dd>
        <dt>{t(mi.kvProblems)}</dt>
        <dd>{summary.problems}</dd>
        <dt>{t(mi.kvOld)}</dt>
        <dd>{t(mi.kvOldValue, { count: ran.length })}</dd>
        <dt>{t(mi.kvEmails)}</dt>
        <dd>{t(mi.kvEmailsValue)}</dd>
      </dl>
      <div className="flex flex-wrap gap-2">
        <ButtonLink variant="primary" to="/w/$workspace/forms/website" params={{ workspace: slug }}>
          {t(mi.seePosts)}
        </ButtonLink>
        {imp.reportUrl ? (
          <ButtonAnchor href={reportHref(slug, imp)} download="" newTab={false} icon="download">
            {t(mi.downloadReport)}
          </ButtonAnchor>
        ) : null}
      </div>
      <span className="k">{t(mi.reportNote)}</span>
    </section>
  )

  function summarize(i: Import, posts: Staged[]) {
    const paid = posts.filter((x) => x.visibility === 'paid')
    const members = posts.filter((x) => x.visibility === 'members').length
    const pub = posts.filter((x) => x.visibility === 'public').length
    const tierName = (ref: string) => {
      const m = i.tiers.find((x) => x.ref === ref)
      return m?.tierId === 'any' || !m ? t(mi.anyTier) : (i.paidTiers.find((x) => x.id === m.tierId)?.name ?? ref)
    }
    const byTier = new Map<string, number>()
    for (const x of paid) {
      const name = x.tierRefs.length ? tierName(x.tierRefs[0]!) : t(mi.anyTier)
      byTier.set(name, (byTier.get(name) ?? 0) + 1)
    }
    const parts: string[] = []
    if (paid.length) parts.push(`${t(mi.accessPaid, { count: paid.length })}${byTier.size ? `: ${[...byTier].map(([n, c]) => `${count(c)} ${n}`).join(', ')}` : ''}`)
    if (members) parts.push(t(mi.accessMembers, { count: members }))
    if (pub) parts.push(t(mi.accessPublic, { count: pub }))
    const authors = new Map<string, { n: number; kept: boolean }>()
    for (const x of posts) {
      if (!x.author) continue
      const v = i.authors.find((a) => a.name === x.author)
      const kept = !v || v.userId === 'keep'
      const name = kept ? x.author : (i.team.find((m) => m.id === v.userId)?.name ?? x.author)
      const cur = authors.get(name) ?? { n: 0, kept }
      authors.set(name, { n: cur.n + 1, kept })
    }
    const bylines = [...authors].map(([name, a]) => (a.kept ? t(mi.bylineKept, { count: a.n, name }) : t(mi.bylineCount, { count: a.n, name }))).join(' · ')
    const counts = { embed: 0, alt: 0, slug: 0, image: 0, other: 0 }
    let withProblems = 0
    for (const x of posts) {
      const list = x.problems.filter((q) => q.code !== 'already_imported')
      if (list.length) withProblems++
      for (const q of list) {
        if (q.code === 'embed_link') counts.embed++
        else if (q.code === 'image_alt') counts.alt++
        else if (q.code === 'slug_taken') counts.slug++
        else if (q.code === 'image_failed') counts.image++
        else counts.other++
      }
    }
    const list = [
      counts.embed ? t(mi.probEmbeds, { count: counts.embed }) : '',
      counts.alt ? t(mi.probAlt, { count: counts.alt }) : '',
      counts.slug ? t(mi.probSlug, { count: counts.slug }) : '',
      counts.image ? t(mi.probImages, { count: counts.image }) : '',
      counts.other ? t(mi.probOther, { count: counts.other }) : '',
    ].filter(Boolean)
    return {
      access: parts.join(' · ') || t(mi.kvProblemsNone),
      bylines,
      problems: withProblems ? t(mi.kvProblemsValue, { count: withProblems, list: list.join(', ') }) : t(mi.kvProblemsNone),
    }
  }
}

function Failed({ imp, onAnother }: { imp: Import; onAnother: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canPublish = useCan('website:publish')
  const [busy, setBusy] = useState(false)
  const p = imp.progress
  const retry = async () => {
    setBusy(true)
    try {
      await http.POST(`${ONE}/start`, { params: { path: { workspace: slug, import: imp.id } } })
      await qc.invalidateQueries({ queryKey: ['get', ONE] })
    } catch (e) {
      toast({ kind: 'error', title: t(mi.startFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const stopped = imp.finishedAt ?? imp.updatedAt
  return (
    <>
      <FileCard imp={imp} />
      <ErrorPanel
        title={p ? t(mi.stoppedTitle, { done: p.done, total: p.total }) : t(mi.failedTitle)}
        actions={
          <>
            {p ? (
              <Button variant="primary" loading={busy} disabledReason={canPublish ? undefined : t(mi.needsPublish)} onClick={() => void retry()}>
                {t(mi.tryAgain)}
              </Button>
            ) : null}
            {imp.reportUrl ? (
              <ButtonAnchor href={reportHref(slug, imp)} download="" newTab={false} icon="download">
                {t(mi.downloadReport)}
              </ButtonAnchor>
            ) : null}
            <Button onClick={onAnother}>{t(mi.uploadAnother)}</Button>
          </>
        }
      >
        {imp.error}
        <span className="k mt-2 block">{t(mi.importRef, { id: imp.id.slice(0, 8), date: `${dayMonthDate(stopped)}, ${clock(stopped)}` })}</span>
      </ErrorPanel>
    </>
  )
}

function SiteOff({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canManage = useCan('website:manage')
  return (
    <section className="card wi-card" aria-labelledby="wi-off-h">
      <h3 id="wi-off-h">{t(mi.offTitle)}</h3>
      <span>{t(mi.offText, { host: bare(overview.address.url) })}</span>
      <div>
        {canManage ? (
          <ButtonLink variant="primary" to="/w/$workspace/forms/website/turn-on" params={{ workspace: slug }}>
            {t(mi.turnOn)}
          </ButtonLink>
        ) : (
          <Button variant="primary" disabledReason={t(mi.turnOnNeeds)}>
            {t(mi.turnOn)}
          </Button>
        )}
      </div>
      <span className="k">{t(mi.turnOnNeeds)}</span>
    </section>
  )
}
