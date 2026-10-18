import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SurveyBuilderPage } from '@/features/surveys/SurveyBuilder'

const search = z.object({
  tab: z.enum(['questions', 'settings', 'results']).optional().catch(undefined),
  q: z.string().max(8).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/forms/surveys/$surveyId')({
  validateSearch: search,
  component: function SurveyRoute() {
    const { surveyId } = Route.useParams()
    const s = Route.useSearch()
    return <SurveyBuilderPage key={surveyId} surveyId={surveyId} tab={s.tab ?? 'questions'} question={s.q} />
  },
})
