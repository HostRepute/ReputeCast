import { $api, type Schemas } from '@/api/client'
import { defineMessages, useT } from '@/i18n'
import { dayMonthDate, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'

type Item = Schemas['ContactActivityItem']

/**
 * Poll answers and survey responses on the contact profile (design
 * expansion, X11; brief §6): timeline items, and where a field came from.
 */
const m = defineMessages('pollContact', {
  answered: 'Answered {answer} to “{question}” in {campaign}',
  answeredComment: 'Answered {answer} to “{question}” in {campaign} · with a comment',
  changed: 'changed from {previous}',
  quote: '“{text}”',
  responded: 'Responded to the survey “{name}” · {answered} of {questions} answered',
  started: 'Started the survey “{name}” · {answered} of {questions} answered',
  editable: 'can change it until {date}',
  fromPoll: 'from a poll, {date}',
  fromSurvey: 'from a survey, {date}',
})

/** "How likely are you to recommend the Weekly Digest…": long questions end in an ellipsis. */
function short(q: string, max = 52): string {
  return q.length > max ? `${q.slice(0, max).replace(/\s+\S*$/, '')}…` : q
}

/** The title and the line under it of a poll answer or a survey response; null for other items. */
export function useAnswerActivity(): (it: Item) => { title: string; meta: string[] } | null {
  const t = useT()
  return (it) => {
    if (it.type === 'poll_answer') {
      const values = { answer: it.answer ?? '', question: short(it.question ?? ''), campaign: it.subject }
      const meta: string[] = []
      if (it.previous) meta.push(t(m.changed, { previous: it.previous }))
      if (it.comment) meta.push(t(m.quote, { text: it.comment }))
      return { title: t(it.comment ? m.answeredComment : m.answered, values), meta }
    }
    if (it.type === 'survey_response' || it.type === 'survey_started') {
      const meta: string[] = []
      if (it.editableUntil) meta.push(t(m.editable, { date: shortDate(it.editableUntil) }))
      return { title: t(it.type === 'survey_response' ? m.responded : m.started, { name: it.subject, answered: it.answered ?? 0, questions: it.questions ?? 0 }), meta }
    }
    return null
  }
}

/** Where each field set by a poll or a survey came from, by field key. */
export function useFieldSources(contactId: string): Map<string, Schemas['FieldSource']> {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/field-sources', { params: { path: { workspace: slug, contactId } } }, { staleTime: 60_000, retry: 0 })
  return new Map((q.data?.items ?? []).map((s) => [s.field, s]))
}

/** "· from a poll, 7 Oct". */
export function FieldSourceNote({ source }: { source: Schemas['FieldSource'] | undefined }) {
  const t = useT()
  if (!source) return null
  return <span className="k"> · {t(source.kind === 'survey' ? m.fromSurvey : m.fromPoll, { date: dayMonthDate(source.at) })}</span>
}
