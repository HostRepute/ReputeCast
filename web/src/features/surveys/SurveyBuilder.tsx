import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { OptionCard } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, count, shortDate } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { FormsTabs, copyText, hostPath, throwIfForbidden, useFormsOverview } from '@/features/forms/common'
import { SurveyStatus } from './FormsList'
import { m } from './messages'
import { SurveyResultsTab } from './Results'
import { TYPES, TypeIcon, typeLabel } from './types'
import './surveys.css'

type Survey = Schemas['Survey']
type Question = Schemas['SurveyQuestion']
type Settings = Schemas['SurveySettings']
type QType = Question['type']
type Tab = 'questions' | 'settings' | 'results'

interface Draft {
  name: string
  description: string
  slug: string
  questions: Question[]
  settings: Settings
}

/** "2026-10-31": a closing date two weeks from now. */
function twoWeeksOut(): string {
  return new Date(Date.now() + 14 * 86_400_000).toISOString().slice(0, 10)
}

function draftOf(s: Survey): Draft {
  return { name: s.name, description: s.description, slug: s.slug, questions: s.questions, settings: s.settings }
}

function same(a: unknown, b: unknown) {
  return JSON.stringify(a) === JSON.stringify(b)
}

function nextKey(qs: Question[]): string {
  let n = 1
  while (qs.some((q) => q.key === `q${n}`)) n++
  return `q${n}`
}

function newQuestion(t: ReturnType<typeof useT>, type: QType, qs: Question[]): Question {
  const base: Question = { key: nextKey(qs), type, text: '', required: true, answers: [], save: { to: 'none' } }
  if (type === 'single' || type === 'multi') base.answers = [{ key: 1, label: '' }, { key: 2, label: '' }]
  if (type === 'scale') Object.assign(base, { nps: true, low: 'Not likely', high: 'Very likely' })
  if (type === 'emoji') base.faces = 5
  if (type === 'short' || type === 'long') base.required = false
  void t
  return base
}

/** What keeps each question from being published, as the server says it. */
function problemsOf(qs: Question[]): Map<string, string[]> {
  const out = new Map<string, string[]>()
  const add = (k: string, msg: string) => out.set(k, [...(out.get(k) ?? []), msg])
  for (const q of qs) {
    if (!q.text.trim()) add(q.key, 'text')
    if ((q.type === 'single' || q.type === 'multi') && q.answers.filter((a) => a.label.trim()).length < 2) add(q.key, 'answers')
    if (q.save.to === 'field' && !q.save.field) add(q.key, 'field')
    if (q.save.to === 'tags' && (q.type === 'single' || q.type === 'multi') && q.answers.some((a) => a.label.trim() && !a.tag?.trim())) add(q.key, 'tags')
  }
  return out
}

/** The survey builder (SurveyBuilder): Questions, Settings and Results. */
export function SurveyBuilderPage({ surveyId, tab, question }: { surveyId: string; tab: Tab; question?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const res = $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys/{surveyId}', { params: { path: { workspace: slug, surveyId } } }, { retry: 1 })
  throwIfForbidden(res.error)
  if (res.isError) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <section className="card sv-error" role="alert">
          <h2 className="m-0 text-[17px] font-semibold">{t(m.errorTitle)}</h2>
          <p className="m-0 max-w-[520px] text-ink-2">{res.error instanceof ApiError && res.error.status === 404 ? t(m.notFound) : t(m.errorBody, { detail: errorMessage(res.error) })}</p>
          <Button variant="primary" onClick={() => void res.refetch()}>
            {t(m.retry)}
          </Button>
        </section>
      </Page>
    )
  }
  if (res.isPending) {
    return (
      <Page title="Forms & public pages" filters={<FormsTabs />}>
        <section className="card sv-loading" aria-busy="true" aria-label={t(m.loading)}>
          <div className="flex gap-3">
            {[0, 1, 2].map((i) => (
              <span key={i} className="rep-sk sk" style={{ flex: 1, height: 88 }} />
            ))}
          </div>
          {[0, 1, 2, 3].map((i) => (
            <span key={i} className="sk" style={{ height: 24 }} />
          ))}
        </section>
      </Page>
    )
  }
  return <Builder key={res.data.updatedAt} s={res.data} tab={tab} question={question} />
}

function Builder({ s, tab, question }: { s: Survey; tab: Tab; question?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const meQ = useMe()
  const [draft, setDraft] = useState<Draft>(() => draftOf(s))
  const [busy, setBusy] = useState<'save' | 'publish' | 'close' | 'reopen'>()
  const [dialog, setDialog] = useState<'publish' | 'close' | 'reopen' | 'delete'>()
  const saved = useMemo(() => draftOf(s), [s])
  const dirty = !same(draft, saved)
  const problems = useMemo(() => problemsOf(draft.questions), [draft.questions])
  const readOnly = !s.canEdit
  const path = { workspace: slug, surveyId: s.id }
  const refresh = async () => {
    await qc.invalidateQueries({ predicate: (q) => typeof q.queryKey[1] === 'string' && q.queryKey[1].includes('/surveys') })
  }
  const go = (next: { tab?: Tab; q?: string }) => void navigate({ to: '/w/$workspace/forms/surveys/$surveyId', params: { workspace: slug, surveyId: s.id }, search: { tab: next.tab ?? tab, q: next.q }, replace: true })
  const selected = draft.questions.find((q) => q.key === question) ?? draft.questions[0]

  const save = async (d: Draft = draft): Promise<boolean> => {
    setBusy('save')
    try {
      await http.PATCH('/api/v1/workspaces/{workspace}/surveys/{surveyId}', { params: { path }, body: d })
      return true
    } catch (e) {
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
      return false
    } finally {
      setBusy(undefined)
    }
  }
  const saveDraft = async () => {
    if (await save()) {
      toast({ kind: 'success', title: t(m.saved) })
      await refresh()
    }
  }
  const publishChanges = async () => {
    if (await save()) {
      toast({ kind: 'success', title: t(m.changesLive) })
      await refresh()
    }
  }
  const copy = async () => {
    const ok = await copyText(s.url)
    toast(ok ? { kind: 'success', title: t(m.copied), detail: hostPath(s.url) } : { kind: 'error', title: t(m.copyFailed, { url: s.url }) })
  }
  const linked = s.linked.find((l) => l.via === 'poll') ?? s.linked[0]
  const metaUrl = hostPath(s.url)
  const meta = linked
    ? t(m.metaLinked, { url: metaUrl, name: s.createdBy || '—', date: shortDate(s.createdAt), linked: t(m.quoted, { name: linked.name }) })
    : t(m.meta, { url: metaUrl, name: s.createdBy || '—', date: shortDate(s.createdAt) })
  const actions = readOnly ? (
    s.status !== 'draft' ? (
      <Button icon="copy" onClick={() => void copy()}>
        {t(m.copyLink)}
      </Button>
    ) : null
  ) : (
    <>
      {s.status !== 'draft' ? (
        <Button icon="copy" onClick={() => void copy()}>
          {t(m.copyLink)}
        </Button>
      ) : null}
      {s.status === 'open' ? <Button onClick={() => setDialog('close')}>{t(m.closeSurvey)}</Button> : null}
      {s.status === 'closed' ? <Button onClick={() => setDialog('reopen')}>{t(m.reopenSurvey)}</Button> : null}
      {s.status === 'draft' ? (
        <>
          <Button disabled={!dirty} loading={busy === 'save'} onClick={() => void saveDraft()}>
            {t(m.saveDraft)}
          </Button>
          <Button variant="primary" disabled={problems.size > 0 || !draft.questions.length} onClick={() => setDialog('publish')}>
            {t(m.publish)}
          </Button>
        </>
      ) : (
        <Button variant="primary" disabled={!dirty || problems.size > 0} loading={busy === 'save'} onClick={() => void publishChanges()}>
          {t(m.publishChanges)}
        </Button>
      )}
    </>
  )
  const me = meQ.data?.user?.name ?? ''
  return (
    <Page title="Forms & public pages" filters={<FormsTabs />} actions={actions} contentClassName="sv-main">
      <div className="sv-head">
        <div className="flex min-w-0 items-start gap-3">
          <Link to="/w/$workspace/forms" params={{ workspace: slug }} className="btn ic sm" aria-label={t(m.back)}>
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M10 3L5 8l5 5" />
            </svg>
          </Link>
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="k">{t(m.survey)}</span>
            <h1 className="sv-title">{s.name}</h1>
          </div>
          <SurveyStatus s={s} />
        </div>
        <p className="k m-0">{s.status === 'draft' ? t(m.notPublished) : meta}</p>
        <nav className="utabs sv-tabs" aria-label={t(m.sections)}>
          {(['questions', 'settings', 'results'] as const).map((x) => (
            <button key={x} type="button" aria-current={tab === x ? 'page' : undefined} className={tab === x ? 'on' : undefined} onClick={() => go({ tab: x, q: question })}>
              {x === 'questions' ? t(m.tabQuestions) : x === 'settings' ? t(m.tabSettings) : s.completed ? t(m.tabResults, { count: count(s.completed) }) : t(m.tabResultsEmpty)}
            </button>
          ))}
        </nav>
      </div>
      {readOnly ? (
        <div className="note n sv-ro" role="note">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
            <circle cx="8" cy="8" r="6.5" />
            <path d="M8 7.2v3.6M8 5.2v.1" />
          </svg>
          <span>{s.editHelpers.length ? t(m.readOnly, { name: me, helpers: formatList(s.editHelpers, 'disjunction') }) : t(m.readOnlyNoHelpers, { name: me })}</span>
        </div>
      ) : null}
      {s.status === 'closed' && tab !== 'settings' ? (
        <div className="note n sv-closed" role="note">
          <span className="flex flex-col gap-1">
            <b>{t(m.closedHeading)}</b>
            <span>{t(m.closedNote, { date: s.closedAt ? shortDate(s.closedAt) : '', time: s.closedAt ? clock(s.closedAt) : '' })}</span>
          </span>
        </div>
      ) : null}
      {tab === 'questions' ? (
        readOnly ? (
          <ReadOnlyQuestions s={s} />
        ) : draft.questions.length === 0 ? (
          <FirstQuestion name={draft.name} onAdd={(type) => {
            const q = newQuestion(t, type, draft.questions)
            setDraft({ ...draft, questions: [q] })
            go({ q: q.key })
          }} />
        ) : (
          <QuestionsTab
            s={s}
            draft={draft}
            setDraft={setDraft}
            selected={selected}
            onSelect={(key) => go({ q: key })}
            problems={problems}
            onPublish={() => (s.status === 'draft' ? setDialog('publish') : void publishChanges())}
            busy={busy === 'save'}
            dirty={dirty}
          />
        )
      ) : tab === 'settings' ? (
        <SettingsTab s={s} draft={draft} readOnly={readOnly} onChange={async (settings, extra) => {
          const next = { ...draft, ...extra, settings }
          setDraft(next)
          // Settings save right away (the board: "Changes apply right away").
          if (await save({ ...saved, ...extra, settings })) await refresh()
        }} />
      ) : (
        <SurveyResultsTab s={s} />
      )}
      {dialog === 'publish' ? (
        <PublishDialog
          s={s}
          draft={draft}
          onClose={() => setDialog(undefined)}
          onPublish={async () => {
            if (dirty && !(await save())) throw new Error(t(m.publishFailed))
            await http.POST('/api/v1/workspaces/{workspace}/surveys/{surveyId}/publish', { params: { path } })
            setDialog(undefined)
            toast({ kind: 'success', title: t(m.published), detail: t(m.publishedDetail), action: { label: t(m.copyLink), onClick: () => void copy() } })
            await refresh()
          }}
        />
      ) : null}
      <ConfirmDialog
        open={dialog === 'close'}
        onOpenChange={(o) => !o && setDialog(undefined)}
        tone="warn"
        badge={t(m.closeBadge)}
        confirmVariant="primary"
        title={t(m.closeTitle, { name: s.name })}
        description={t(m.closeBody)}
        confirmLabel={t(m.closeSurvey)}
        onConfirm={async () => {
          await http.POST('/api/v1/workspaces/{workspace}/surveys/{surveyId}/close', { params: { path } })
          setDialog(undefined)
          toast({ kind: 'success', title: t(m.closedToast) })
          await refresh()
        }}
      />
      {dialog === 'reopen' ? (
        <ReopenDialog
          s={s}
          onClose={() => setDialog(undefined)}
          onReopen={async (close) => {
            await http.POST('/api/v1/workspaces/{workspace}/surveys/{surveyId}/reopen', { params: { path }, body: { close } })
            setDialog(undefined)
            toast({ kind: 'success', title: t(m.reopenedToast) })
            await refresh()
          }}
        />
      ) : null}
    </Page>
  )
}

/** A new survey with no questions: pick the first question's type. */
function FirstQuestion({ name, onAdd }: { name: string; onAdd: (type: QType) => void }) {
  const t = useT()
  return (
    <section className="card sv-first">
      <span className="sv-first-ic" aria-hidden="true">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
          <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" />
          <path d="M14 3v5h5M9 13h6M9 17h6" />
        </svg>
      </span>
      <h2 className="m-0 text-[17px] font-semibold">{t(m.firstTitle)}</h2>
      <p className="m-0 max-w-[560px] text-ink-2">{t(m.firstBody, { name })}</p>
      <div className="sv-types sv-types-3">
        {TYPES.map((x) => (
          <button key={x.type} type="button" className="sv-type" onClick={() => onAdd(x.type)}>
            <TypeIcon type={x.type} />
            {t(m[x.label])}
          </button>
        ))}
      </div>
      <span className="k">{t(m.notPublished)}</span>
    </section>
  )
}

function QuestionsTab({
  s,
  draft,
  setDraft,
  selected,
  onSelect,
  problems,
  onPublish,
  busy,
  dirty,
}: {
  s: Survey
  draft: Draft
  setDraft: (d: Draft) => void
  selected: Question | undefined
  onSelect: (key: string) => void
  problems: Map<string, string[]>
  onPublish: () => void
  busy: boolean
  dirty: boolean
}) {
  const t = useT()
  const qs = draft.questions
  const setQuestions = (questions: Question[]) => setDraft({ ...draft, questions })
  const update = (key: string, patch: Partial<Question>) => setQuestions(qs.map((q) => (q.key === key ? { ...q, ...patch } : q)))
  const add = (type: QType) => {
    if (qs.length >= 20) return
    const q = newQuestion(t, type, qs)
    setQuestions([...qs, q])
    onSelect(q.key)
  }
  const move = (from: number, to: number) => {
    if (to < 0 || to >= qs.length || from === to) return
    const next = [...qs]
    const [x] = next.splice(from, 1)
    next.splice(to, 0, x!)
    setQuestions(next)
  }
  const index = selected ? qs.findIndex((q) => q.key === selected.key) : -1
  const firstBad = qs.findIndex((q) => problems.has(q.key))
  return (
    <div className="sv-cols">
      <aside className="sv-left" aria-label={t(m.questionList)}>
        <div className="sv-left-h">
          <h2 className="m-0 text-[14px] font-semibold">{t(m.questions, { count: qs.length })}</h2>
          <span className="k">{t(m.dragToReorder)}</span>
        </div>
        <QuestionList qs={qs} selected={selected?.key} problems={problems} onSelect={onSelect} onMove={move} />
        <div className="sv-add">
          <h3 className="m-0 text-[13px] font-semibold">{t(m.addQuestion)}</h3>
          <div className="sv-types">
            {TYPES.map((x) => (
              <button key={x.type} type="button" className="sv-type" disabled={qs.length >= 20} onClick={() => add(x.type)}>
                <TypeIcon type={x.type} />
                {t(m[x.label])}
              </button>
            ))}
          </div>
          <span className="k">{qs.length >= 20 ? t(m.full) : t(m.upTo)}</span>
        </div>
        <div className="sv-left-foot">
          <span className="k">{firstBad < 0 ? t(m.allComplete) : firstBad > 0 ? t(m.completeUpTo, { n: firstBad }) : ''}</span>
          {problems.size ? <span className="sv-bad">{t(m.needsFix, { count: problems.size })}</span> : null}
        </div>
      </aside>
      {selected ? (
        <QuestionEditor
          key={selected.key}
          s={s}
          q={selected}
          n={index + 1}
          total={qs.length}
          problems={problems.get(selected.key) ?? []}
          onChange={(patch) => update(selected.key, patch)}
          onDuplicate={() => {
            if (qs.length >= 20) return
            const copy = { ...selected, key: nextKey(qs), answers: selected.answers.map((a) => ({ ...a })) }
            const next = [...qs]
            next.splice(index + 1, 0, copy)
            setQuestions(next)
            onSelect(copy.key)
          }}
          onDelete={() => {
            const next = qs.filter((q) => q.key !== selected.key)
            setQuestions(next)
            const after = next[Math.min(index, next.length - 1)]
            if (after) onSelect(after.key)
          }}
          footer={
            <div className="sv-ed-foot">
              <span className="k">{t(m.publishWaits)}</span>
              <Button variant="primary" disabled={problems.size > 0 || (s.status !== 'draft' && !dirty)} loading={busy} onClick={onPublish}>
                {s.status === 'draft' ? t(m.publish) : t(m.publishChanges)}
              </Button>
            </div>
          }
        />
      ) : null}
      <LivePreview s={s} draft={draft} selected={selected?.key} n={index + 1} />
    </div>
  )
}

/** The questions, reorderable by dragging or with the arrow buttons. */
function QuestionList({
  qs,
  selected,
  problems,
  onSelect,
  onMove,
}: {
  qs: Question[]
  selected?: string
  problems: Map<string, string[]>
  onSelect: (key: string) => void
  onMove: (from: number, to: number) => void
}) {
  const t = useT()
  const dragging = useRef<number | null>(null)
  const [over, setOver] = useState<number | null>(null)
  return (
    <ol className="sv-qs">
      {qs.map((q, i) => {
        const bad = problems.get(q.key)?.length ?? 0
        return (
          <li
            key={q.key}
            className={cn('sv-q', q.key === selected && 'on', bad > 0 && 'bad', over === i && 'over')}
            draggable
            onDragStart={(e) => {
              dragging.current = i
              e.dataTransfer.effectAllowed = 'move'
            }}
            onDragOver={(e) => {
              e.preventDefault()
              setOver(i)
            }}
            onDragLeave={() => setOver(null)}
            onDrop={(e) => {
              e.preventDefault()
              if (dragging.current !== null) onMove(dragging.current, i)
              dragging.current = null
              setOver(null)
            }}
            onDragEnd={() => {
              dragging.current = null
              setOver(null)
            }}
          >
            <span className="sv-grip" aria-hidden="true">
              ⠿
            </span>
            <button type="button" className="sv-q-main" aria-current={q.key === selected ? 'true' : undefined} onClick={() => onSelect(q.key)}>
              <span className="sv-num">{i + 1}</span>
              <span className="flex min-w-0 flex-col gap-1">
                <span className="sv-q-text">{q.text || t(m.questionPlaceholder)}</span>
                <span className="k">
                  {typeLabel(t, q.type)} · {q.required ? t(m.required) : t(m.optional)}
                  {q.save.to === 'field' && q.save.field ? (
                    <>
                      {' · '}
                      <code className="sv-chip">{q.save.field}</code>
                    </>
                  ) : null}
                  {bad ? (
                    <>
                      {' · '}
                      <span className="badge bad sm">{t(m.problems, { count: bad })}</span>
                    </>
                  ) : null}
                </span>
              </span>
            </button>
            <span className="sv-move">
              <button type="button" className="btn ic xs" aria-label={t(m.moveUp, { n: i + 1 })} disabled={i === 0} onClick={() => onMove(i, i - 1)}>
                ▲
              </button>
              <button type="button" className="btn ic xs" aria-label={t(m.moveDown, { n: i + 1 })} disabled={i === qs.length - 1} onClick={() => onMove(i, i + 1)}>
                ▼
              </button>
            </span>
          </li>
        )
      })}
    </ol>
  )
}

function Field({ label, hint, children, id }: { label: ReactNode; hint?: ReactNode; children: ReactNode; id?: string }) {
  return (
    <div className="sv-f">
      <label className="sv-f-l" htmlFor={id}>
        {label}
      </label>
      {children}
      {hint ? <span className="k">{hint}</span> : null}
    </div>
  )
}

/** The selected question: type, text, description, required, answers and where it's saved. */
function QuestionEditor({
  s,
  q,
  n,
  total,
  problems,
  onChange,
  onDuplicate,
  onDelete,
  footer,
}: {
  s: Survey
  q: Question
  n: number
  total: number
  problems: string[]
  onChange: (patch: Partial<Question>) => void
  onDuplicate: () => void
  onDelete: () => void
  footer: ReactNode
}) {
  const t = useT()
  const id = useId()
  const fields = useFormsOverview().data?.customFields ?? []
  const choice = q.type === 'single' || q.type === 'multi'
  const fieldOptions = fields.filter((f) => (q.type === 'scale' || q.type === 'emoji' ? f.type === 'number' : f.type !== 'number' && f.type !== 'date' && f.type !== 'boolean'))
  const setType = (type: QType) => {
    const patch: Partial<Question> = { type }
    if ((type === 'single' || type === 'multi') && q.answers.length < 2) patch.answers = [...q.answers, ...Array.from({ length: 2 - q.answers.length }, (_, i) => ({ key: q.answers.length + i + 1, label: '' }))]
    if (type === 'scale' && q.nps === undefined) Object.assign(patch, { nps: true, low: 'Not likely', high: 'Very likely' })
    if (type === 'emoji' && !q.faces) patch.faces = 5
    if (q.save.to === 'tags' && type !== 'single' && type !== 'multi') patch.save = { to: 'none' }
    if (q.save.to === 'field') patch.save = { to: 'field' }
    onChange(patch)
  }
  const answers = q.answers
  const setAnswer = (i: number, patch: Partial<Question['answers'][number]>) => onChange({ answers: answers.map((a, j) => (j === i ? { ...a, ...patch } : a)) })
  const badAnswers = problems.includes('answers')
  return (
    <section className="sv-ed" aria-labelledby={`${id}-h`}>
      <div className="sv-ed-h">
        <div className="flex flex-col gap-0.5">
          <span className="k">{t(m.questionOf, { n, total })}</span>
          <h2 className="m-0 text-[15px] font-semibold" id={`${id}-h`}>
            {typeLabel(t, q.type)}
          </h2>
        </div>
        <div className="flex items-center gap-1.5">
          {problems.length ? <span className="badge bad sm">{t(m.problems, { count: problems.length })}</span> : null}
          <Button size="sm" onClick={onDuplicate} disabled={total >= 20}>
            {t(m.duplicate)}
          </Button>
          <Button size="sm" variant="danger" onClick={onDelete}>
            {t(m.delete)}
          </Button>
        </div>
      </div>
      <div className="sv-ed-b">
        <Field label={t(m.type)} id={`${id}-type`}>
          <select id={`${id}-type`} className="in" value={q.type} onChange={(e) => setType(e.target.value as QType)}>
            {TYPES.map((x) => (
              <option key={x.type} value={x.type}>
                {t(m[x.label])}
              </option>
            ))}
          </select>
        </Field>
        <Field label={t(m.question)} id={`${id}-text`}>
          <input
            id={`${id}-text`}
            className="in"
            value={q.text}
            placeholder={t(m.questionPlaceholder)}
            aria-invalid={problems.includes('text') || undefined}
            onChange={(e) => onChange({ text: e.target.value })}
          />
        </Field>
        <Field label={<>{t(m.description)} <span className="k">· {t(m.optional)}</span></>} id={`${id}-desc`}>
          <input id={`${id}-desc`} className="in" value={q.description ?? ''} placeholder={t(m.descriptionPlaceholder)} onChange={(e) => onChange({ description: e.target.value })} />
        </Field>
        <div className="sv-f sv-switch-row">
          <span className="flex flex-col gap-0.5">
            <span className="sv-f-l">{t(m.requiredLabel)}</span>
            <span className="k">{t(m.requiredHint)}</span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(m.requiredLabel)} checked={q.required} onChange={(e) => onChange({ required: e.target.checked })} />
        </div>
        {choice ? (
          <div className="sv-f" role="group" aria-labelledby={`${id}-ans`}>
            <span className="sv-f-l" id={`${id}-ans`}>
              {t(m.answers)}
            </span>
            {answers.map((a, i) => (
              <div key={i} className="sv-ans">
                <input
                  className="in"
                  aria-label={t(m.answerN, { n: i + 1 })}
                  value={a.label}
                  aria-invalid={(badAnswers && !a.label.trim()) || undefined}
                  onChange={(e) => setAnswer(i, { label: e.target.value })}
                />
                <button type="button" className="btn sm ic" aria-label={t(m.removeAnswer, { n: i + 1 })} onClick={() => onChange({ answers: answers.filter((_, j) => j !== i) })}>
                  ×
                </button>
              </div>
            ))}
            {badAnswers ? (
              <span className="sv-err" role="alert">
                {t(m.needTwo)}
              </span>
            ) : null}
            <Button size="sm" icon="plus" className="self-start" onClick={() => onChange({ answers: [...answers, { key: Math.max(0, ...answers.map((x) => x.key)) + 1, label: '' }] })}>
              {t(m.addAnswer)}
            </Button>
          </div>
        ) : null}
        {q.type === 'scale' ? (
          <div className="sv-f">
            <span className="sv-f-l">{t(m.wording)}</span>
            <label className="tg">
              <input type="checkbox" checked={!!q.nps} onChange={(e) => onChange({ nps: e.target.checked, ...(e.target.checked ? { low: 'Not likely', high: 'Very likely' } : {}) })} />
              {t(m.useNps)}
            </label>
            <div className="row2">
              <label className="k flex flex-col gap-1">
                {t(m.lowReads)}
                <input className="in" value={q.low ?? ''} onChange={(e) => onChange({ low: e.target.value })} />
              </label>
              <label className="k flex flex-col gap-1">
                {t(m.highReads)}
                <input className="in" value={q.high ?? ''} onChange={(e) => onChange({ high: e.target.value })} />
              </label>
            </div>
            <span className="k">{t(m.npsHint)}</span>
          </div>
        ) : null}
        {q.type === 'emoji' ? (
          <div className="sv-f">
            <span className="sv-f-l">{t(m.faces)}</span>
            <div className="seg self-start" role="group" aria-label={t(m.faces)}>
              {[3, 5].map((f) => (
                <button key={f} type="button" aria-pressed={(q.faces ?? 5) === f} onClick={() => onChange({ faces: f })}>
                  {f === 3 ? t(m.facesThree) : t(m.facesFive)}
                </button>
              ))}
            </div>
          </div>
        ) : null}
        <div className="sv-f" role="radiogroup" aria-labelledby={`${id}-save`}>
          <span className="sv-f-l" id={`${id}-save`}>
            {t(m.save)}
          </span>
          <div className="flex flex-col gap-1.5">
            <OptionCard name={`${id}-save`} value="none" checked={q.save.to === 'none'} onChange={() => onChange({ save: { to: 'none' } })} title={t(m.saveNone)} description={t(m.saveNoneHint)} />
            <OptionCard
              name={`${id}-save`}
              value="field"
              checked={q.save.to === 'field'}
              onChange={() => onChange({ save: { to: 'field', field: q.save.field } })}
              title={q.type === 'scale' || q.type === 'emoji' ? t(m.saveFieldNumber) : t(m.saveField)}
              description={t(m.saveFieldHint, { example: q.type === 'scale' ? 'beta_nps' : 'role' })}
            />
            {q.save.to === 'field' ? (
              <select className="in" aria-label={t(m.pickField)} value={q.save.field ?? ''} aria-invalid={problems.includes('field') || undefined} onChange={(e) => onChange({ save: { to: 'field', field: e.target.value } })}>
                <option value="">{t(m.pickField)}</option>
                {fieldOptions.map((f) => (
                  <option key={f.key} value={f.key}>
                    {f.key} · {f.type}
                  </option>
                ))}
              </select>
            ) : null}
            <OptionCard name={`${id}-save`} value="tags" checked={q.save.to === 'tags'} disabled={!choice} onChange={() => onChange({ save: { to: 'tags' } })} title={t(m.saveTags)} description={t(m.saveTagsHint)} />
            {q.save.to === 'tags' && choice ? (
              <div className="sv-tags">
                {answers.map((a, i) => (
                  <label key={i} className="k">
                    {a.label || t(m.answerN, { n: i + 1 })}
                    <input className="in" aria-label={t(m.tagFor, { answer: a.label || String(i + 1) })} value={a.tag ?? ''} onChange={(e) => setAnswer(i, { tag: e.target.value })} />
                  </label>
                ))}
              </div>
            ) : null}
          </div>
        </div>
        {s.status !== 'draft' && s.started > 0 ? (
          <div className="note i">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
              <circle cx="8" cy="8" r="6.5" />
              <path d="M8 7.2v3.6M8 5.2v.1" />
            </svg>
            <span>{t(m.startedNote, { count: s.started })}</span>
          </div>
        ) : null}
      </div>
      {problems.length ? footer : null}
    </section>
  )
}

/** The survey page as the draft draws it, desktop or mobile, the selected question marked. */
function LivePreview({ s, draft, selected, n }: { s: Survey; draft: Draft; selected?: string; n: number }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const frame = useRef<HTMLIFrameElement>(null)
  const [got, setGot] = useState<{ html: string; error: string } | null>(null)
  useEffect(() => {
    let cancelled = false
    const h = window.setTimeout(() => {
      void http
        .POST('/api/v1/workspaces/{workspace}/public-pages/survey-preview', {
          params: { path: { workspace: slug } },
          body: { name: draft.name, description: draft.description, slug: draft.slug, questions: draft.questions, settings: draft.settings },
        })
        .then(({ data }) => !cancelled && setGot({ html: data?.html ?? '', error: '' }))
        .catch((e: unknown) => !cancelled && setGot({ html: '', error: errorMessage(e) }))
    }, 350)
    return () => {
      cancelled = true
      window.clearTimeout(h)
    }
  }, [draft, slug])
  const mark = selected
    ? `<style>#${selected}{outline:2px solid #1D4ED8;outline-offset:8px;border-radius:4px;position:relative;margin-top:22px}#${selected}::before{content:${JSON.stringify(
        t(m.questionOf, { n, total: draft.questions.length }).split(' ').slice(0, 2).join(' '),
      )};position:absolute;top:-36px;left:-10px;background:#1D4ED8;color:#fff;font:600 11px/20px 'IBM Plex Sans',sans-serif;padding:0 8px;border-radius:4px 4px 0 0}</style>`
    : ''
  const html = got?.html ? got.html.replace('</head>', `${mark}</head>`) : ''
  return (
    <section className="sv-prev" aria-label={t(m.previewTitle)}>
      <div className="sv-prev-h">
        <span className="k">{s.status === 'draft' ? t(m.livePreviewDraft) : t(m.livePreview, { url: hostPath(s.url) })}</span>
        <div className="seg" role="group" aria-label={t(m.device)}>
          {(['desktop', 'mobile'] as const).map((d) => (
            <button key={d} type="button" aria-pressed={device === d} onClick={() => setDevice(d)}>
              {d === 'desktop' ? t(m.desktop) : t(m.mobile)}
            </button>
          ))}
        </div>
      </div>
      <div className="sv-prev-b">
        {got?.error ? (
          <p role="alert" className="k">
            {t(m.previewFailed, { detail: got.error })}
          </p>
        ) : html ? (
          // Same origin without scripts: the builder scrolls to the selected question.
          <iframe
            ref={frame}
            title={t(m.previewTitle)}
            className={cn('sv-frame', device === 'mobile' && 'mobile')}
            sandbox="allow-same-origin"
            srcDoc={html}
            onLoad={() => frame.current?.contentDocument?.getElementById(selected ?? '')?.scrollIntoView({ block: 'center' })}
          />
        ) : (
          <p className="k" role="status">
            {t(m.previewLoading)}
          </p>
        )}
      </div>
    </section>
  )
}

/** Daniel Park's view: the questions, no editor. */
function ReadOnlyQuestions({ s }: { s: Survey }) {
  const t = useT()
  return (
    <ol className="sv-qs sv-ro-list">
      {s.questions.map((q, i) => (
        <li key={q.key} className="sv-q">
          <span className="sv-q-main">
            <span className="sv-num">{i + 1}</span>
            <span className="flex min-w-0 flex-col gap-1">
              <span className="sv-q-text">{q.text}</span>
              <span className="k">
                {typeLabel(t, q.type)} · {q.required ? t(m.required) : t(m.optional)}
              </span>
            </span>
          </span>
        </li>
      ))}
    </ol>
  )
}

/** "Publish 'Beta feedback · October 2026'?": what goes live. */
function PublishDialog({ s, draft, onClose, onPublish }: { s: Survey; draft: Draft; onClose: () => void; onPublish: () => Promise<void> }) {
  const t = useT()
  const now = useNow(60_000)
  const who = useMe().data?.user?.name ?? s.createdBy
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const fields = [...new Set(draft.questions.filter((q) => q.save.to === 'field' && q.save.field).map((q) => q.save.field!))]
  const c = draft.settings.close
  const linked = s.linked.find((l) => l.via === 'poll') ?? s.linked[0]
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      eyebrow={t(m.publishEyebrow, { date: shortDate(now), name: who })}
      title={t(m.publishTitle, { name: draft.name })}
      description={t(m.publishBody)}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>
            {t(m.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={busy}
            busyLabel={t(m.publishing)}
            onClick={() => {
              setBusy(true)
              setError('')
              onPublish().catch((e: unknown) => {
                setBusy(false)
                setError(errorMessage(e))
              })
            }}
          >
            {t(m.publish)}
          </Button>
        </>
      }
    >
      <dl className="sv-dl">
        <dt>{t(m.page)}</dt>
        <dd className="mono">{s.url}</dd>
        <dt>{t(m.tabQuestions)}</dt>
        <dd>{t(m.questionsAll, { count: draft.questions.length })}</dd>
        <dt>{t(m.openUntil)}</dt>
        <dd>{c.mode === 'date' && c.date ? `${c.date}${c.time ? `, ${c.time}` : ''} ${draft.settings.timezone ?? ''}`.trim() : t(m.untilClosed)}</dd>
        <dt>{t(m.responses)}</dt>
        <dd>{draft.settings.oneResponse ? t(m.oneEditable) : t(m.manyResponses)}</dd>
        <dt>{t(m.nonContacts)}</dt>
        <dd>{draft.settings.acceptNonContacts ? t(m.accepted) : t(m.notAccepted)}</dd>
        <dt>{t(m.linkedFrom)}</dt>
        <dd>{linked ? t(linked.via === 'poll' ? m.viaPoll : m.viaLink, { name: linked.name }) : t(m.notLinked)}</dd>
      </dl>
      <div className="note i" style={{ marginTop: 12 }}>
        <span>{fields.length ? t(m.publishNoteFields, { fields: formatList(fields) }) : t(m.publishNote)}</span>
      </div>
      {error ? (
        <p role="alert" className="sv-err" style={{ marginTop: 10 }}>
          {error}
        </p>
      ) : null}
    </Dialog>
  )
}

/** Reopen with a new closing date, or none. */
function ReopenDialog({ s, onClose, onReopen }: { s: Survey; onClose: () => void; onReopen: (close: Schemas['SurveyClose']) => Promise<void> }) {
  const t = useT()
  const [mode, setMode] = useState<'date' | 'manual'>('date')
  const [date, setDate] = useState(twoWeeksOut)
  const [time, setTime] = useState('23:59')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="ok"
      badge={t(m.reopenSurvey)}
      confirmVariant="primary"
      title={t(m.reopenTitle, { name: s.name })}
      description={t(m.reopenBody)}
      confirmLabel={t(m.reopenSurvey)}
      onConfirm={() => onReopen(mode === 'date' ? { mode, date, time } : { mode })}
    >
      <div className="flex flex-col gap-2">
        <label className="tg">
          <input type="radio" name="sv-reopen" checked={mode === 'date'} onChange={() => setMode('date')} />
          {t(m.reopenUntil)}
        </label>
        {mode === 'date' ? (
          <div className="row2">
            <input className="in" type="date" aria-label={t(m.closeDate)} value={date} onChange={(e) => setDate(e.target.value)} />
            <input className="in" type="time" aria-label={t(m.closeTime)} value={time} onChange={(e) => setTime(e.target.value)} />
          </div>
        ) : null}
        <label className="tg">
          <input type="radio" name="sv-reopen" checked={mode === 'manual'} onChange={() => setMode('manual')} />
          {t(m.reopenNoDate)}
        </label>
      </div>
    </ConfirmDialog>
  )
}

/** Settings: the closing date, one response per contact, non-contacts and the thank-you text. */
function SettingsTab({ s, draft, readOnly, onChange }: { s: Survey; draft: Draft; readOnly: boolean; onChange: (st: Settings, extra?: Partial<Draft>) => void }) {
  const t = useT()
  const id = useId()
  const st = draft.settings
  const { me } = useWorkspace()
  const wsTz = me.timezone || 'UTC'
  const tz = st.timezone || wsTz
  const [thanks, setThanks] = useState(st.thanks ?? '')
  const [name, setName] = useState(draft.name)
  const [desc, setDesc] = useState(draft.description)
  const set = (patch: Partial<Settings>) => onChange({ ...st, ...patch })
  return (
    <section className="card sv-settings" aria-labelledby={`${id}-h`}>
      <div className="sv-settings-h">
        <h2 className="m-0 text-[15px] font-semibold" id={`${id}-h`}>
          {t(m.settings)}
        </h2>
        <span className="k">{s.status === 'open' ? t(m.settingsSub) : t(m.settingsSubDraft)}</span>
      </div>
      <fieldset className="sv-settings-b" disabled={readOnly}>
        <div className="row2">
          <Field label={t(m.name)} id={`${id}-name`}>
            <input id={`${id}-name`} className="in" value={name} onChange={(e) => setName(e.target.value)} onBlur={() => name.trim() && name !== draft.name && onChange(st, { name: name.trim() })} />
          </Field>
          <Field label={t(m.address)} id={`${id}-slug`} hint={t(m.addressHint, { url: hostPath(s.url) })}>
            <input id={`${id}-slug`} className="in mono" value={draft.slug} readOnly />
          </Field>
        </div>
        <Field label={t(m.intro)} id={`${id}-desc`} hint={t(m.introHint)}>
          <textarea id={`${id}-desc`} className="in" rows={2} value={desc} onChange={(e) => setDesc(e.target.value)} onBlur={() => desc !== draft.description && onChange(st, { description: desc })} />
        </Field>
        <div className="row2">
          <Field label={t(m.openUntilLabel)} id={`${id}-until`}>
            <select
              id={`${id}-until`}
              className="in"
              value={st.close.mode}
              onChange={(e) => set({ close: e.target.value === 'date' ? { mode: 'date', date: st.close.date || twoWeeksOut(), time: st.close.time || '23:59' } : { mode: 'manual' } })}
            >
              <option value="date">{st.close.mode === 'date' && st.close.date ? t(m.untilDate, { date: `${st.close.date}, ${st.close.time || '23:59'}` }) : t(m.untilDatePick)}</option>
              <option value="manual">{t(m.untilManual)}</option>
            </select>
          </Field>
          <Field label={t(m.timezone)} id={`${id}-tz`}>
            <select id={`${id}-tz`} className="in" value={tz} onChange={(e) => set({ timezone: e.target.value })}>
              <option value={wsTz}>{t(m.workspaceDefault, { tz: wsTz })}</option>
              {['UTC', 'Europe/London', 'Europe/Berlin', 'America/New_York', 'America/Los_Angeles', 'Asia/Ho_Chi_Minh']
                .filter((z) => z !== wsTz)
                .map((z) => (
                  <option key={z} value={z}>
                    {z}
                  </option>
                ))}
            </select>
          </Field>
        </div>
        {st.close.mode === 'date' ? (
          <div className="row2">
            <input className="in" type="date" aria-label={t(m.closeDate)} value={st.close.date ?? ''} onChange={(e) => e.target.value && set({ close: { ...st.close, date: e.target.value } })} />
            <input className="in" type="time" aria-label={t(m.closeTime)} value={st.close.time ?? '23:59'} onChange={(e) => e.target.value && set({ close: { ...st.close, time: e.target.value } })} />
          </div>
        ) : null}
        <div className="sv-toggle">
          <span className="flex flex-col gap-0.5">
            <b>{t(m.oneResponse)}</b>
            <span className="k">{t(m.oneResponseHint)}</span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(m.oneResponse)} checked={st.oneResponse} onChange={(e) => set({ oneResponse: e.target.checked })} />
        </div>
        <div className="sv-toggle">
          <span className="flex flex-col gap-0.5">
            <b>{t(m.acceptNonContacts)}</b>
            <span className="k">{t(m.acceptNonContactsHint)}</span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(m.acceptNonContacts)} checked={st.acceptNonContacts} onChange={(e) => set({ acceptNonContacts: e.target.checked })} />
        </div>
        <Field label={t(m.thanks)} id={`${id}-thanks`}>
          <textarea
            id={`${id}-thanks`}
            className="in"
            rows={3}
            placeholder={t(m.thanksPlaceholder)}
            value={thanks}
            onChange={(e) => setThanks(e.target.value)}
            onBlur={() => thanks !== (st.thanks ?? '') && set({ thanks })}
          />
        </Field>
      </fieldset>
    </section>
  )
}

