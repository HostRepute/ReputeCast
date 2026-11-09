import { useEffect, type ReactNode } from 'react'
import { formatList, T } from '@/i18n'
import { ms } from '../messages'
import { tags } from '../parts'
import { IdeasPanel } from './ideas'
import { RewritePanel } from './rewrite'
import { useComposerAssist } from './state'

/** The composer's right column: the assistant's panel while one is open, else the inspector. */
export function AssistRight({ fallback }: { fallback: ReactNode }) {
  const ca = useComposerAssist()
  const kind = ca?.panel?.kind
  // At 1100 px and below the right column sits under the canvas: a panel
  // opened from the header (Subject ideas) scrolls into view instead of
  // opening out of sight (AIComposerTablet).
  useEffect(() => {
    if (!kind || !window.matchMedia('(max-width: 1100px)').matches) return
    const reveal = () => document.querySelector<HTMLElement>('.cmp .ai-panel')?.scrollIntoView?.({ block: 'start' })
    let frame = requestAnimationFrame(reveal)
    // The Assist menu gives focus back to its button just after, which
    // scrolls the page back up: reveal the panel once more then.
    const again = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(reveal)
    }
    document.addEventListener('focusin', again, { once: true })
    const stop = setTimeout(() => document.removeEventListener('focusin', again), 1000)
    return () => {
      cancelAnimationFrame(frame)
      clearTimeout(stop)
      document.removeEventListener('focusin', again)
    }
  }, [kind])
  if (kind === 'ideas') return <IdeasPanel />
  if (kind === 'rewrite') return <RewritePanel />
  return <>{fallback}</>
}

/**
 * The Content Writer's banner with the assistant (AIComposer): "You can
 * edit this campaign and use the assistant. Sending needs campaigns:send,
 * so it goes out after Marcus Chen or Linh Nguyen approve it." Undefined
 * when the person can't use it (the banner keeps its own words).
 */
export function useAssistBanner(): ((p: { name: string; role: string; helpers: string[] }) => ReactNode) | undefined {
  const ca = useComposerAssist()
  if (!ca || !ca.assist.can || !ca.assist.configured) return undefined
  return ({ name, role, helpers }) =>
    helpers.length ? (
      <T msg={ms.bannerAssist} values={{ ...tags, name, role, helpers: formatList(helpers, 'disjunction') }} />
    ) : (
      <T msg={ms.bannerAssistAlone} values={{ ...tags, name, role }} />
    )
}
