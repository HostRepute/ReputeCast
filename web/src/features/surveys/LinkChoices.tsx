import { defineMessages, useT } from '@/i18n'
import { useSurveys } from './api'

/**
 * The composer's link editor offers the workspace's surveys (design
 * expansion, X11; SurveyBuilder · publishing): the link carries each
 * contact's token, so their answers are linked to them.
 */
const m = defineMessages('surveyLink', {
  title: 'Or link to a survey',
  survey: 'Survey: {name}',
  draft: 'not published yet',
  hint: 'Adds the contact’s token, so answers are linked to them.',
})

export function SurveyLinkChoices({ value, onPick }: { value: string; onPick: (link: string) => void }) {
  const t = useT()
  const surveys = useSurveys()
  const items = (surveys.data?.items ?? []).filter((s) => s.status !== 'closed')
  if (!items.length) return null
  return (
    <div className="flex flex-col gap-1" role="group" aria-label={t(m.title)}>
      <span className="k">{t(m.title)}</span>
      {items.map((s) => (
        <button
          key={s.id}
          type="button"
          className={`btn sm${value === s.variable ? ' pri' : ''}`}
          style={{ justifyContent: 'flex-start', textAlign: 'left', height: 'auto', minHeight: 30, whiteSpace: 'normal' }}
          aria-pressed={value === s.variable}
          onClick={() => onPick(s.variable)}
        >
          {t(m.survey, { name: s.name })}
          {s.status === 'draft' ? <span className="k"> · {t(m.draft)}</span> : null}
        </button>
      ))}
      <span className="k">{t(m.hint)}</span>
    </div>
  )
}
