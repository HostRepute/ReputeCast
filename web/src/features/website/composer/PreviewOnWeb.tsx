import { errorMessage } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import type { Post } from '../api'
import { usePostPreview } from '../PostDialogs'
import { SiteFrame } from '../SiteFrame'
import { mc } from './messages'

/**
 * Preview on web (ComposerPublishToWeb · preview states): the post's page as
 * a visitor, a free member or a paid member sees it, rendered by the server
 * from the last saved version (inert: nothing submits, nothing counts).
 */
export function PreviewOnWeb({ post, as, onAs, onBack, asMain = true }: { post: Post; as: 'visitor' | 'free' | 'paid'; onAs: (as: 'visitor' | 'free' | 'paid') => void; onBack: () => void; asMain?: boolean }) {
  const t = useT()
  const preview = usePostPreview(post.id, as)
  const Root = asMain ? 'main' : 'section'
  return (
    <Root className="pc-preview" id={asMain ? 'main' : undefined} aria-label={asMain ? undefined : t(mc.previewAs)}>
      <div className="pc-pbar">
        <span className="font-semibold">{t(mc.previewAs)}</span>
        <div className="seg" role="group" aria-label={t(mc.previewAs)}>
          {(['visitor', 'free', 'paid'] as const).map((a) => (
            <button key={a} type="button" aria-pressed={as === a} onClick={() => onAs(a)}>
              {a === 'visitor' ? t(mc.asVisitor) : a === 'free' ? t(mc.asFree) : t(mc.asPaid)}
            </button>
          ))}
        </div>
        <span className="k">{t(mc.previewSaveFirst)}</span>
        <span className="flex-1" />
        <Button size="sm" onClick={onBack}>
          {t(mc.edit)}
        </Button>
      </div>
      <div className="stage">
        {preview.data ? (
          <div className="pc-pframe">
            <SiteFrame html={preview.data.html} title={t(mc.previewFrame, { title: post.title, who: as })} minHeight={900} />
          </div>
        ) : preview.isError ? (
          <Note tone="bad">{t(mc.previewFailed, { error: errorMessage(preview.error) })}</Note>
        ) : (
          <Loading label={t(mc.previewLoading)} className="w-full max-w-[1100px]">
            <Skeleton w="100%" h={900} r={8} />
          </Loading>
        )}
      </div>
    </Root>
  )
}
