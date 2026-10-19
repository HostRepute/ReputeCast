import type { ReactNode } from 'react'
import { formatList, T } from '@/i18n'
import { ms } from '../messages'
import { tags } from '../parts'
import { IdeasPanel } from './ideas'
import { RewritePanel } from './rewrite'
import { useComposerAssist } from './state'

/** The composer's right column: the assistant's panel while one is open, else the inspector. */
export function AssistRight({ fallback }: { fallback: ReactNode }) {
  const ca = useComposerAssist()
  if (ca?.panel?.kind === 'ideas') return <IdeasPanel />
  if (ca?.panel?.kind === 'rewrite') return <RewritePanel />
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
