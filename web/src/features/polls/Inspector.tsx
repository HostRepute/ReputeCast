import type { Component } from 'grapesjs'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, shortDate } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { tagOf } from '@/features/composer/editor/labels'
import { ATTR_POLL } from '@/features/conditional/mjml'
import { encodePoll, normalizePoll, parsePoll, pollAnswers, pollSection, type PollConfig } from './config'
import { m } from './messages'
import { AnswerPagePreview } from './PagePreview'
import './polls.css'

/** The poll a component is, null when it isn't one. */
export function pollOf(c: Component | null | undefined): PollConfig | null {
  if (!c || tagOf(c) !== 'mj-section') return null
  const raw = (c.getAttributes() as Record<string, unknown>)[ATTR_POLL]
  return typeof raw === 'string' ? parsePoll(raw) : null
}

/** A face image's address on the instance, as the email shows it. */
export function faceUrl(base: string) {
  return (faces: number, key: number) => `${base.replace(/\/$/, '')}/p/_/faces/${faces}-${key}.png?c=0f766e`
}

/**
 * Writes a poll's settings: its rc-poll attribute and the MJML they draw,
 * in one undo step.
 */
export function applyPoll(comp: Component, cfg: PollConfig, publicUrl: string) {
  const mjml = pollSection(cfg, faceUrl(publicUrl))
  const start = mjml.indexOf('<mj-column>')
  const end = mjml.lastIndexOf('</mj-section>')
  comp.addAttributes({ [ATTR_POLL]: encodePoll(cfg) })
  if (start >= 0 && end > start) comp.components(mjml.slice(start, end))
}

function Prop({ title, children, hint }: { title: ReactNode; children?: ReactNode; hint?: ReactNode }) {
  const id = useId()
  return (
    <div className="prop" role="group" aria-labelledby={id}>
      <span className="t" id={id}>
        {title}
      </span>
      {children}
      {hint ? <span className="k">{hint}</span> : null}
    </div>
  )
}

/** A text input that commits on Enter or blur. */
function Draft({ value, onCommit, label, placeholder, multiline, type = 'text', disabled }: { value: string; onCommit: (v: string) => void; label: string; placeholder?: string; multiline?: boolean; type?: string; disabled?: boolean }) {
  const [draft, setDraft] = useState(value)
  const focused = useRef(false)
  useEffect(() => {
    if (!focused.current) setDraft(value)
  }, [value])
  const commit = () => draft !== value && onCommit(draft)
  const common = {
    className: 'in',
    'aria-label': label,
    placeholder,
    value: draft,
    disabled,
    onFocus: () => (focused.current = true),
    onBlur: () => {
      focused.current = false
      commit()
    },
  }
  return multiline ? (
    <textarea {...common} rows={3} onChange={(e) => setDraft(e.target.value)} />
  ) : (
    <input
      {...common}
      type={type}
      onChange={(e) => setDraft(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          commit()
          ;(e.target as HTMLInputElement).blur()
        }
      }}
    />
  )
}

function Seg<V extends string>({ value, options, onChange, label, disabled }: { value: V; options: { value: V; label: string }[]; onChange: (v: V) => void; label: string; disabled?: boolean }) {
  return (
    <div className="seg self-start" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={value === o.value} disabled={disabled} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

type Survey = Schemas['Survey']

const QTYPE: Record<string, keyof typeof m> = {
  single: 'typeSingle',
  multi: 'typeMulti',
  scale: 'typeScale',
  emoji: 'typeEmoji',
  short: 'typeShort',
  long: 'typeLong',
}

/** Whether a survey's first question takes a poll's answers (the same kind, the same answers). */
export function surveyFits(cfg: PollConfig, s: Survey): boolean {
  const q = s.questions[0]
  if (!q) return false
  if (cfg.kind === 'scale') return q.type === 'scale'
  if (cfg.kind === 'emoji') return q.type === 'emoji'
  if (q.type !== 'single' && q.type !== 'multi') return false
  const labels = new Set(q.answers.map((a) => a.label.trim().toLowerCase()))
  return pollAnswers(cfg).every((a) => labels.has(a.label.toLowerCase()))
}

function SurveyPicker({ cfg, onPick, disabled }: { cfg: PollConfig; onPick: (id: string) => void; disabled?: boolean }) {
  const t = useT()
  const c = useComposer()
  const [q, setQ] = useState('')
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/surveys', { params: { path: { workspace: c.slug } } }, { staleTime: 30_000 })
  const items = (list.data?.items ?? []).filter((s) => !q.trim() || s.name.toLowerCase().includes(q.trim().toLowerCase()))
  const chosen = list.data?.items.find((s) => s.id === cfg.survey)
  const status = (s: Survey) =>
    s.status === 'open'
      ? s.closesAt
        ? t(m.surveyOpen, { date: shortDate(s.closesAt), count: s.questions.length })
        : t(m.surveyOpenManual, { count: s.questions.length })
      : s.status === 'draft'
        ? t(m.surveyDraft, { count: s.questions.length })
        : t(m.surveyClosed, { count: s.questions.length })
  return (
    <div className="flex flex-col gap-1.5">
      <input className="in" type="search" aria-label={t(m.findSurvey)} placeholder={t(m.findSurvey)} value={q} onChange={(e) => setQ(e.target.value)} disabled={disabled} />
      <div className="flex flex-col gap-1" role="radiogroup" aria-label={t(m.survey)}>
        {items.map((s) => {
          const fits = surveyFits(cfg, s)
          const first = s.questions[0]
          return (
            <label key={s.id} className={`pl-survey${cfg.survey === s.id ? ' on' : ''}${fits ? '' : ' off'}`}>
              <input type="radio" name="poll-survey" checked={cfg.survey === s.id} disabled={disabled || !fits} onChange={() => onPick(s.id)} />
              <span className="font-semibold">{s.name}</span>
              <span className="k">{status(s)}</span>
              <span className={`k pl-fit${fits ? ' ok' : ''}`}>
                {!first ? t(m.surveyEmpty) : fits ? t(m.surveyMatches, { type: t(m[QTYPE[first.type] ?? 'typeSingle']) }) : t(m.surveyNoMatch, { type: t(m[QTYPE[first.type] ?? 'typeSingle']) })}
              </span>
            </label>
          )
        })}
        {list.data && !list.data.items.length ? <span className="k">{t(m.noSurveys)}</span> : null}
      </div>
      {chosen ? <span className="k">{t(m.surveyPage, { url: chosen.url.replace(/^https?:\/\//, '') })}</span> : null}
      <a className="k underline" href={`/w/${c.slug}/forms?new=survey`} target="_blank" rel="noreferrer">
        {t(m.createSurvey)}
      </a>
    </div>
  )
}

/** The campaign's test answers to this poll (after a test send). */
function TestAnswers({ blockId }: { blockId: string }) {
  const t = useT()
  const c = useComposer()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/poll-test-answers', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } }, { staleTime: 15_000 })
  const [clearing, setClearing] = useState(false)
  const items = (q.data?.items ?? []).filter((a) => a.blockId === blockId)
  const cfg = pollOf(c.selected)
  if (!items.length) return null
  const label = (n: number) => (cfg ? (pollAnswers(cfg).find((a) => a.key === n)?.label ?? String(n)) : String(n))
  const clear = async () => {
    setClearing(true)
    try {
      const { data } = await http.DELETE('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/poll-test-answers', { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } })
      toast({ kind: 'success', title: t(m.clearedTests, { count: data?.deleted ?? 0 }) })
      await q.refetch()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setClearing(false)
    }
  }
  return (
    <Prop title={t(m.testAnswers, { count: items.length })} hint={t(m.testAnswersHint)}>
      <table className="tbl pl-tests" aria-label={t(m.testTable)}>
        <thead>
          <tr>
            <th scope="col">{t(m.testFrom)}</th>
            <th scope="col">{t(m.testAnswer)}</th>
            <th scope="col">{t(m.testTime)}</th>
            <th scope="col">
              <span className="vh">{t(m.testBadge)}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((a) => (
            <tr key={`${a.pollId}-${a.email}`}>
              <td className="mono">{a.email}</td>
              <td>{label(a.answer)}</td>
              <td className="num">{clock(a.at)}</td>
              <td>
                <Badge tone="neutral">{t(m.testBadge)}</Badge>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {c.canEdit ? (
        <button type="button" className="btn sm self-start" onClick={() => void clear()} disabled={clearing}>
          {t(m.clearTests)}
        </button>
      ) : null}
    </Prop>
  )
}

function closesOn(c: ReturnType<typeof useComposer>, days: number): string | null {
  const at = c.campaign.scheduledAt
  if (!at) return null
  return shortDate(new Date(new Date(at).getTime() + days * 86_400_000))
}

/**
 * The Poll block's settings (brief §5.6): kind, question, answers, layout,
 * where the answer is saved, when it closes, what follows, non-contacts,
 * results, the answer page preview and the test answers.
 */
export function PollSettings({ c: comp }: { c: Component }) {
  const t = useT()
  const c = useComposer()
  const [preview, setPreview] = useState(false)
  const cfg = pollOf(comp)
  const fields = $api.useQuery('get', '/api/v1/workspaces/{workspace}/visibility/catalog', { params: { path: { workspace: c.slug } } }, { enabled: !!cfg && c.kind !== 'transactional', staleTime: 60_000 })
  if (!cfg || c.kind === 'transactional') return null
  const disabled = !c.canEdit
  const set = (patch: Partial<PollConfig>) => applyPoll(comp, normalizePoll({ ...cfg, ...patch }), c.context.publicUrl)
  const answers = cfg.answers
  const setAnswer = (i: number, patch: Partial<PollConfig['answers'][number]>) => set({ answers: answers.map((a, j) => (j === i ? { ...a, ...patch } : a)) })
  const move = (i: number, d: -1 | 1) => {
    const j = i + d
    if (j < 0 || j >= answers.length) return
    const next = [...answers]
    ;[next[i], next[j]] = [next[j]!, next[i]!]
    set({ answers: next })
  }
  const fieldOptions = (fields.data?.fields ?? []).filter((f) => (cfg.kind === 'choice' ? f.type !== 'number' && f.type !== 'date' && f.type !== 'boolean' : f.type === 'number'))
  return (
    <>
      <div className="flex flex-col gap-0.5 border-b border-line-2" style={{ padding: '10px 12px' }}>
        <span className="font-semibold">{t(m.title)}</span>
        <span className="k">{t(m.subtitle)}</span>
        {disabled ? <span className="k">{t(m.readOnly)}</span> : null}
      </div>
      <Prop title={t(m.kind)}>
        <Seg
          label={t(m.kind)}
          value={cfg.kind}
          disabled={disabled}
          onChange={(kind) =>
            set({
              kind,
              ...(kind === 'scale'
                ? { nps: true, low: cfg.low || t(m.scaleLow), high: cfg.high || t(m.scaleHigh), save: cfg.save === 'tag' ? 'none' : cfg.save }
                : kind === 'emoji'
                  ? { faces: 5, save: cfg.save === 'tag' ? 'none' : cfg.save }
                  : {}),
            })
          }
          options={[
            { value: 'choice', label: t(m.kindChoice) },
            { value: 'scale', label: t(m.kindScale) },
            { value: 'emoji', label: t(m.kindEmoji) },
          ]}
        />
      </Prop>
      <Prop title={t(m.question)}>
        <Draft label={t(m.question)} value={cfg.question} disabled={disabled} onCommit={(question) => set({ question })} />
      </Prop>
      {cfg.kind === 'choice' ? (
        <>
          <Prop title={t(m.answers, { count: answers.length })} hint={t(m.answersHint)}>
            {answers.map((a, i) => (
              <div key={i} className="pl-answer">
                <div className="flex">
                  <button type="button" className="btn sm ic pl-move" aria-label={t(m.moveUp, { n: i + 1 })} disabled={disabled || i === 0} onClick={() => move(i, -1)}>
                    ▲
                  </button>
                  <button type="button" className="btn sm ic pl-move" aria-label={t(m.moveDown, { n: i + 1 })} disabled={disabled || i === answers.length - 1} onClick={() => move(i, 1)}>
                    ▼
                  </button>
                </div>
                <Draft label={t(m.answerN, { n: i + 1 })} value={a.label} disabled={disabled} onCommit={(label) => setAnswer(i, { label })} />
                <button
                  type="button"
                  className="btn sm ic"
                  aria-label={t(m.removeAnswer, { n: i + 1 })}
                  disabled={disabled || answers.length <= 2}
                  onClick={() => set({ answers: answers.filter((_, j) => j !== i) })}
                >
                  ×
                </button>
              </div>
            ))}
            {!disabled ? (
              <button type="button" className="btn sm self-start" disabled={answers.length >= 5} onClick={() => set({ answers: [...answers, { label: t(m.newAnswer) }] })}>
                {t(m.addAnswer)}
              </button>
            ) : null}
          </Prop>
          <Prop title={t(m.layout)} hint={t(m.layoutHint)}>
            <Seg
              label={t(m.layout)}
              value={cfg.layout}
              disabled={disabled}
              onChange={(layout) => set({ layout })}
              options={[
                { value: 'stacked', label: t(m.layoutStacked) },
                { value: 'row', label: t(m.layoutRow) },
              ]}
            />
          </Prop>
        </>
      ) : cfg.kind === 'scale' ? (
        <Prop title={t(m.wording)} hint={t(m.npsHint)}>
          <label className="tg">
            <input type="checkbox" checked={!!cfg.nps} disabled={disabled} onChange={(e) => set({ nps: e.target.checked, ...(e.target.checked ? { low: t(m.scaleLow), high: t(m.scaleHigh) } : {}) })} />
            {t(m.useNps)}
          </label>
          <div className="row2">
            <label className="k">
              {t(m.lowReads)}
              <Draft label={t(m.lowReads)} value={cfg.low ?? ''} disabled={disabled} onCommit={(low) => set({ low })} />
            </label>
            <label className="k">
              {t(m.highReads)}
              <Draft label={t(m.highReads)} value={cfg.high ?? ''} disabled={disabled} onCommit={(high) => set({ high })} />
            </label>
          </div>
        </Prop>
      ) : (
        <Prop title={t(m.faces)} hint={t(m.facesHint)}>
          <Seg
            label={t(m.faces)}
            value={String(cfg.faces === 3 ? 3 : 5) as '3' | '5'}
            disabled={disabled}
            onChange={(v) => set({ faces: Number(v) })}
            options={[
              { value: '3', label: t(m.facesThree) },
              { value: '5', label: t(m.facesFive) },
            ]}
          />
        </Prop>
      )}
      <Prop title={t(m.save)}>
        <div className="pl-opts flex flex-col gap-1.5" role="radiogroup" aria-label={t(m.save)}>
          <label className="tg">
            <input type="radio" name="poll-save" checked={cfg.save === 'none'} disabled={disabled} onChange={() => set({ save: 'none' })} />
            <span className="flex flex-col">
              {t(m.saveNone)}
              <span className="k">{t(m.saveNoneHint)}</span>
            </span>
          </label>
          <label className="tg">
            <input type="radio" name="poll-save" checked={cfg.save === 'tag'} disabled={disabled || cfg.kind !== 'choice'} onChange={() => set({ save: 'tag' })} />
            <span className="flex flex-col">
              {t(m.saveTag)}
              {cfg.kind !== 'choice' ? <span className="k">{t(m.saveTagChoiceOnly)}</span> : null}
            </span>
          </label>
          {cfg.save === 'tag' && cfg.kind === 'choice' ? (
            <div className="pl-tags">
              {answers.map((a, i) => (
                <label key={i} className="k">
                  {a.label}
                  <Draft label={t(m.tagFor, { answer: a.label })} value={a.tag ?? ''} disabled={disabled} onCommit={(tag) => setAnswer(i, { tag: tag.trim() })} />
                </label>
              ))}
              <span className="k">{t(m.saveTagHint)}</span>
            </div>
          ) : null}
          <label className="tg">
            <input type="radio" name="poll-save" checked={cfg.save === 'field'} disabled={disabled} onChange={() => set({ save: 'field' })} />
            <span className="flex flex-col">
              {t(m.saveField)}
              <span className="k">{cfg.kind === 'choice' ? t(m.saveFieldHint) : t(m.saveFieldNumbers)}</span>
            </span>
          </label>
          {cfg.save === 'field' ? (
            <select className="in" aria-label={t(m.field)} value={cfg.field ?? ''} disabled={disabled} onChange={(e) => set({ field: e.target.value })}>
              <option value="">{t(m.pickField)}</option>
              {fieldOptions.map((f) => (
                <option key={f.key} value={f.key}>
                  {f.key} · {f.type}
                </option>
              ))}
            </select>
          ) : null}
        </div>
      </Prop>
      <Prop title={t(m.closes)} hint={t(m.closeHint)}>
        <select
          className="in"
          aria-label={t(m.closes)}
          value={cfg.close.mode}
          disabled={disabled}
          onChange={(e) => {
            const mode = e.target.value as PollConfig['close']['mode']
            set({ close: mode === 'days' ? { mode, days: cfg.close.days || 7 } : mode === 'date' ? { mode, date: cfg.close.date || new Date(Date.now() + 14 * 86_400_000).toISOString().slice(0, 10) } : { mode } })
          }}
        >
          <option value="days">
            {closesOn(c, cfg.close.days || 7) ? t(m.closeDaysOn, { days: cfg.close.days || 7, date: closesOn(c, cfg.close.days || 7)! }) : t(m.closeDaysValue, { days: cfg.close.days || 7 })}
          </option>
          <option value="date">{t(m.closeDate)}</option>
          <option value="manual">{t(m.closeManual)}</option>
        </select>
        {cfg.close.mode === 'days' ? (
          <label className="k">
            {t(m.closeDaysLabel)}
            <Draft label={t(m.closeDaysLabel)} type="number" value={String(cfg.close.days ?? 7)} disabled={disabled} onCommit={(v) => set({ close: { mode: 'days', days: Math.max(1, Math.min(365, Number(v) || 7)) } })} />
          </label>
        ) : cfg.close.mode === 'date' ? (
          <label className="k">
            {t(m.closeDateLabel)}
            <Draft label={t(m.closeDateLabel)} type="date" value={cfg.close.date ?? ''} disabled={disabled} onCommit={(date) => set({ close: { mode: 'date', date } })} />
          </label>
        ) : null}
      </Prop>
      <Prop title={t(m.after)}>
        <div className="pl-opts flex flex-col gap-1.5" role="radiogroup" aria-label={t(m.after)}>
          <label className="tg">
            <input type="radio" name="poll-after" checked={cfg.after === 'thanks'} disabled={disabled} onChange={() => set({ after: 'thanks' })} />
            {t(m.afterThanks)}
          </label>
          {cfg.after === 'thanks' ? <Draft label={t(m.thanks)} multiline placeholder={t(m.thanksPlaceholder)} value={cfg.thanks ?? ''} disabled={disabled} onCommit={(thanks) => set({ thanks })} /> : null}
          <label className="tg">
            <input type="radio" name="poll-after" checked={cfg.after === 'survey'} disabled={disabled} onChange={() => set({ after: 'survey' })} />
            <span className="flex flex-col">
              {t(m.afterSurvey)}
              <span className="k">{t(m.afterSurveyHint)}</span>
            </span>
          </label>
          {cfg.after === 'survey' ? <SurveyPicker cfg={cfg} disabled={disabled} onPick={(survey) => set({ survey })} /> : null}
        </div>
      </Prop>
      <Prop title={t(m.nonContacts)} hint={t(m.nonContactsHint)}>
        <select className="in" aria-label={t(m.nonContacts)} value={cfg.nonContacts} disabled={disabled} onChange={(e) => set({ nonContacts: e.target.value as PollConfig['nonContacts'] })}>
          <option value="count">{t(m.nonContactsCount)}</option>
          <option value="refuse">{t(m.nonContactsRefuse)}</option>
        </select>
      </Prop>
      <Prop title={t(m.results)} hint={t(m.showResultsHint)}>
        <label className="tg">
          <input type="checkbox" checked={cfg.results} disabled={disabled} onChange={(e) => set({ results: e.target.checked })} />
          {t(m.showResults)}
        </label>
      </Prop>
      <Prop title={t(m.previewPage)} hint={t(m.previewHint)}>
        {/* A link, so it opens in read-only mode too (the fieldset disables buttons). */}
        <a
          href="#poll-preview"
          role="button"
          className="btn sm pl-wide"
          onClick={(e) => {
            e.preventDefault()
            setPreview(true)
          }}
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true" focusable="false">
            <path d="M1.5 8S4 3.5 8 3.5 14.5 8 14.5 8 12 12.5 8 12.5 1.5 8 1.5 8Z" />
            <circle cx="8" cy="8" r="2" />
          </svg>
          {t(m.previewPage)}
        </a>
        {preview ? <AnswerPagePreview slug={c.slug} config={encodePoll(cfg)} campaign={c.name} onClose={() => setPreview(false)} /> : null}
      </Prop>
      <TestAnswers blockId={cfg.id} />
    </>
  )
}
