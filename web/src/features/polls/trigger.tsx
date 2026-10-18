import { $api, type Schemas } from '@/api/client'
import { defineMessages, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'

type Trigger = Schemas['AutomationTrigger']

/**
 * The automation trigger "Answered a poll or survey" (design expansion,
 * X11; brief §6): which poll or survey, which answers, and whether a
 * changed answer starts the flow again.
 */
const m = defineMessages('pollTrigger', {
  source: 'Poll or survey',
  pick: 'Pick a poll or survey',
  polls: 'Polls',
  surveys: 'Surveys',
  pollIn: '{question} · {campaign}',
  answer: 'Answer',
  any: 'Any answer',
  is: 'Is one of',
  isNot: 'Isn’t one of',
  answers: 'Answers',
  onChange: 'Start again when the answer changes',
  note: 'Test answers and answers from people who aren’t contacts never start it.',
  surveyNote: 'A survey starts the flow when someone submits it.',
  loading: 'Loading the polls and surveys…',
})

export function PollTriggerProps({ t, disabled, set }: { t: Trigger; disabled: boolean; set: (patch: Partial<Trigger>) => void }) {
  const tr = useT()
  const { slug } = useWorkspace()
  const sources = $api.useQuery('get', '/api/v1/workspaces/{workspace}/poll-sources', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const items = sources.data?.items ?? []
  const picked = items.find((s) => s.ref === t.poll)
  const answers = t.answers ?? []
  const op = t.answerOp ?? 'any'
  const toggle = (key: number, on: boolean) => set({ answers: on ? [...answers.filter((a) => a !== key), key].sort((a, b) => a - b) : answers.filter((a) => a !== key) })
  return (
    <>
      <select
        className="in"
        aria-label={tr(m.source)}
        disabled={disabled}
        value={t.poll ?? ''}
        onChange={(e) => set({ poll: e.target.value || undefined, answers: undefined, answerOp: undefined })}
      >
        <option value="">{sources.isPending ? tr(m.loading) : tr(m.pick)}</option>
        {items.some((s) => s.kind === 'poll') ? (
          <optgroup label={tr(m.polls)}>
            {items
              .filter((s) => s.kind === 'poll')
              .map((s) => (
                <option key={s.ref} value={s.ref}>
                  {s.campaign ? tr(m.pollIn, { question: s.name, campaign: s.campaign }) : s.name}
                </option>
              ))}
          </optgroup>
        ) : null}
        {items.some((s) => s.kind === 'survey') ? (
          <optgroup label={tr(m.surveys)}>
            {items
              .filter((s) => s.kind === 'survey')
              .map((s) => (
                <option key={s.ref} value={s.ref}>
                  {s.name}
                </option>
              ))}
          </optgroup>
        ) : null}
      </select>
      {picked?.kind === 'poll' ? (
        <>
          <select className="in" aria-label={tr(m.answer)} disabled={disabled} value={op} onChange={(e) => set({ answerOp: e.target.value as Trigger['answerOp'] })}>
            <option value="any">{tr(m.any)}</option>
            <option value="is">{tr(m.is)}</option>
            <option value="is_not">{tr(m.isNot)}</option>
          </select>
          {op !== 'any' ? (
            <fieldset disabled={disabled} style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px', margin: 0, padding: 0, border: 0 }}>
              <legend className="vh">{tr(m.answers)}</legend>
              {picked.answers.map((a) => (
                <label key={a.key} className="fe-chk">
                  <input type="checkbox" checked={answers.includes(a.key)} onChange={(e) => toggle(a.key, e.target.checked)} />
                  {a.label}
                </label>
              ))}
            </fieldset>
          ) : null}
        </>
      ) : null}
      {picked?.kind === 'survey' ? <span className="k">{tr(m.surveyNote)}</span> : null}
      <label className="fe-chk">
        <input type="checkbox" checked={Boolean(t.onChange)} disabled={disabled} onChange={(e) => set({ onChange: e.target.checked || undefined })} />
        {tr(m.onChange)}
      </label>
      <span className="k">{tr(m.note)}</span>
    </>
  )
}
