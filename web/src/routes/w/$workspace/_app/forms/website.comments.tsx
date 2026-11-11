import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { CommentsPage, type CommentsSearch } from '@/features/website/comments/CommentsPage'
import { OnlyWhenOn } from '@/features/website/WebsiteTab'

const search = z.object({
  status: z.enum(['pending', 'published', 'hidden']).optional().catch(undefined),
  post: z.uuid().optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
})

/** Forms & pages › Website › Comments (WebsiteComments, WebsiteCommentsStates). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/comments')({
  validateSearch: search,
  component: function CommentsRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: CommentsSearch, replace?: boolean) => void navigate({ search: next, replace }), [navigate])
    return <OnlyWhenOn>{(ov) => <CommentsPage overview={ov} search={s} onSearch={onSearch} />}</OnlyWhenOn>
  },
})
