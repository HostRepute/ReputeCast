import { Navigate } from '@tanstack/react-router'
import { $api, ApiError } from '@/api/client'
import { ForbiddenState, NotFoundState, problemReference, ServerErrorState } from '@/components/shell/SystemScreens'
import { ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { ComposerSkeleton } from '@/features/composer/ComposerPage'
import { useWebsite, WS } from '../api'
import { mc } from './messages'
import { PostComposer, type PostComposerSearch } from './PostComposer'

export type { PostComposerSearch }

/**
 * /w/$workspace/forms/website/posts/$post: loads the post and the composer's
 * context and opens post mode. A campaign's post opens its campaign's
 * composer on the Web tab.
 */
export function PostComposerPage({ postId, search, onSearch }: { postId: string; search: PostComposerSearch; onSearch: (s: PostComposerSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const post = $api.useQuery('get', `${WS}/posts/{post}`, { params: { path: { workspace: slug, post: postId } } }, { refetchOnWindowFocus: false, retry: (n, e) => !(e as unknown as ApiError)?.status && n < 2 })
  const context = $api.useQuery('get', '/api/v1/workspaces/{workspace}/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const overview = useWebsite()
  useDocumentTitle(post.data ? t(mc.documentTitle, { title: post.data.title }) : t(mc.documentTitle, { title: t(mc.untitled) }))
  const error = post.error ?? context.error ?? overview.error
  if (error) return <PostError error={error} path={`/forms/website/posts/${postId}`} retry={() => void Promise.all([post.refetch(), context.refetch(), overview.refetch()])} />
  if (!post.data || !context.data || !overview.data) return <ComposerSkeleton title={t(mc.opening)} detail={t(mc.openingDetail)} />
  if (post.data.kind === 'campaign' && post.data.campaign) {
    return <Navigate to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: post.data.campaign.id }} search={{ panel: 'web' } as never} replace />
  }
  return <PostComposer key={post.data.id} post={post.data} context={context.data} overview={overview.data} search={search} onSearch={onSearch} />
}

/** 403, 404 and failures, in the composer's centred frame. */
function PostError({ error, path, retry }: { error: unknown; path: string; retry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const back = (
    <ButtonLink to="/w/$workspace/forms/website" params={{ workspace: slug }}>
      {t(mc.back)}
    </ButtonLink>
  )
  const status = error instanceof ApiError ? error.status : 0
  let body
  if (error instanceof ApiError && status === 403 && error.problem.permission) {
    body = <ForbiddenState permission={error.problem.permission} title={t(mc.noAccess)} readOnly={back} />
  } else if (status === 404) {
    body = (
      <NotFoundState
        path={path}
        reference={problemReference(error)}
        primary={
          <ButtonLink variant="primary" to="/w/$workspace/forms/website" params={{ workspace: slug }}>
            {t(mc.goToPosts)}
          </ButtonLink>
        }
      >
        {t(mc.notFoundText)}
      </NotFoundState>
    )
  } else {
    body = <ServerErrorState reference={problemReference(error)} onRetry={retry} back={back} />
  }
  return (
    <div className="cmp rc-app items-center" style={{ padding: '56px 16px' }}>
      <main id="main" className="w-full max-w-[640px]">
        {body}
      </main>
    </div>
  )
}
