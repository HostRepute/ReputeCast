import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { ApiError } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { WorkspaceNotFound } from '@/components/shell/SystemScreens'
import { useT } from '@/i18n'
import { useMediaQuery } from '@/lib/hooks'
import { m } from './messages'
import { closeAssistant, getAssistant, openAssistant, setAssistantThread } from './store'
import { ThreadList } from './ThreadList'
import { ThreadComposer, ThreadLog, ThreadSub, useThreadController } from './ThreadView'
import './assistant.css'

/**
 * Open in full page (assistant brief §3.1): the thread at
 * /w/{workspace}/assistant/{thread}, with the same parts as the panel and,
 * on a desktop, the thread list beside it.
 */
export function AssistantPage({ thread }: { thread: string }) {
  const t = useT()
  const navigate = useNavigate()
  // A new thread starts in the panel, on the screen the person goes back to.
  const c = useThreadController(thread, { onNewThread: () => openAssistant() })
  const wide = useMediaQuery('(min-width: 1101px)')
  const pane = useRef<HTMLDivElement>(null)
  // The panel shows the same thread next time; one stream of it is enough now.
  useEffect(() => {
    setAssistantThread(c.slug, thread)
    if (getAssistant().open) closeAssistant({ refocus: false })
  }, [c.slug, thread])
  const title = c.state.thread?.title || t(m.untitled)
  // Another person's thread, or one that was deleted, isn't found.
  if (c.query.error instanceof ApiError && c.query.error.status === 404) return <WorkspaceNotFound />
  return (
    <Page title={title} documentTitle={t(m.documentTitle, { title })} contentClassName="asst-page">
      <div className="asst-page-grid">
        {wide ? (
          <aside className="asst-page-list card" aria-label={t(m.threadsTitle)}>
            <ThreadList
              current={thread}
              onOpen={(id) => void navigate({ to: '/w/$workspace/assistant/$thread', params: { workspace: c.slug, thread: id } })}
              onDeleted={(id) => {
                if (id === thread) void navigate({ to: '/w/$workspace', params: { workspace: c.slug } })
              }}
            />
          </aside>
        ) : null}
        <section className="asst asst-pane card" ref={pane} aria-label={title}>
          <ThreadSub c={c} />
          <ThreadLog c={c} />
          <ThreadComposer c={c} bound={() => pane.current} />
          <div className="vh" role="status" aria-live="polite">
            {c.announce}
          </div>
        </section>
      </div>
    </Page>
  )
}
