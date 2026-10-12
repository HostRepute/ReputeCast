import { Link, useRouter, type ErrorComponentProps } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { ApiError } from '@/api/client'
import { useDocumentTitle } from '@/lib/hooks'
import { Icon, LogoMark } from '../icons/Icon'
import { Badge, type Tone } from '../ui/Badge'
import { Button } from '../ui/Button'

/** A full-page state card: icon, status badge, sentence, actions, footnote. */
export function FullPageState({
  icon,
  tone = 'neutral',
  badge,
  title,
  children,
  actions,
  footnote,
}: {
  icon: 'searchLarge' | 'lockLarge' | 'warningTriangle'
  tone?: Tone
  badge: ReactNode
  title: ReactNode
  children?: ReactNode
  actions?: ReactNode
  footnote?: ReactNode
}) {
  const ico = { neutral: undefined, warn: { background: '#FEF3C7', color: '#92400E' }, bad: { background: '#FEE2E2', color: '#991B1B' } }[tone as 'neutral' | 'warn' | 'bad']
  return (
    <section className="card full">
      <div className="ico" style={ico}>
        <Icon name={icon} size={22} />
      </div>
      <Badge tone={tone}>{badge}</Badge>
      <h2>{title}</h2>
      {children ? <p>{children}</p> : null}
      {actions ? <div className="flex flex-wrap justify-center gap-2.5">{actions}</div> : null}
      {footnote ? <span className="k">{footnote}</span> : null}
    </section>
  )
}

/** Centered frame for states shown before a workspace is known. */
function Frame({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col items-center gap-6 px-4 py-14">
      <Link to="/" className="flex items-center gap-2.5 text-[16px] font-bold text-ink hover:text-ink" aria-label="ReputeCast home">
        <LogoMark size={28} />
        ReputeCast
      </Link>
      <main id="main" className="w-full max-w-[640px]">
        {children}
      </main>
    </div>
  )
}

export function NotFoundScreen() {
  useDocumentTitle('Not found · ReputeCast')
  const path = typeof window === 'undefined' ? '' : window.location.pathname
  return (
    <Frame>
      <FullPageState
        icon="searchLarge"
        badge="404"
        title={`There's nothing at ${path}`}
        actions={
          <Link to="/" className="btn pri">
            Go to overview
          </Link>
        }
      >
        The page may have been deleted, or the link was copied before it was saved.
      </FullPageState>
    </Frame>
  )
}

export function ErrorScreen({ error, reset }: ErrorComponentProps) {
  useDocumentTitle('Something broke · ReputeCast')
  const router = useRouter()
  const ref = error instanceof ApiError ? (error.problem as { instance?: string }).instance : undefined
  return (
    <Frame>
      <FullPageState
        icon="warningTriangle"
        tone="bad"
        badge={error instanceof ApiError ? String(error.status) : '500'}
        title="Something broke on our side"
        actions={
          <>
            <Button
              variant="primary"
              onClick={() => {
                reset()
                void router.invalidate()
              }}
            >
              Retry
            </Button>
            <Link to="/" className="btn">
              Back to overview
            </Link>
          </>
        }
        footnote={
          <>
            Self-hosted? Check <span className="mono">docker compose logs app</span>
            {ref ? <> for {ref}</> : null}.
          </>
        }
      >
        The page couldn't load. Retry now, or wait a minute; the error is already in the server log.
      </FullPageState>
    </Frame>
  )
}
