import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useRef, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { cn } from '@/lib/cn'
import { count, dayMonthDate } from '@/lib/format'
import { useDocumentTitle, useOverflowX } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { m } from './messages'
import './imports.css'

export type Detail = Schemas['ImportDetail']
export type StepNo = 1 | 2 | 3 | 4 | 5

const steps: { n: StepNo; label: MessageDescriptor }[] = [
  { n: 1, label: m.stepSource },
  { n: 2, label: m.stepFields },
  { n: 3, label: m.stepConsent },
  { n: 4, label: m.stepPreview },
  { n: 5, label: m.stepImport },
]

/** Where each wizard step lives. */
export const stepPath = {
  source: '/w/$workspace/import/$importId/source',
  fields: '/w/$workspace/import/$importId/fields',
  consent: '/w/$workspace/import/$importId/consent',
  preview: '/w/$workspace/import/$importId/preview',
  progress: '/w/$workspace/import/$importId/progress',
  report: '/w/$workspace/import/$importId/report',
} as const

/** The screen an import belongs on now. */
export function currentStep(d: Pick<Detail, 'status' | 'step'>): keyof typeof stepPath {
  if (d.status === 'done' || d.status === 'failed' || d.status === 'cancelled') return 'report'
  if (d.status === 'running' || (d.status === 'paused' && d.step === 'import')) return 'progress'
  if (d.step === 'fields' || d.step === 'consent' || d.step === 'preview') return d.step
  return 'source'
}

/**
 * The full-screen import frame (Migration boards): the step rail beside the
 * content, or the content alone (summary, states).
 */
export function ImportFrame({
  title,
  documentTitle,
  badge,
  exit,
  step,
  hints = {},
  footer,
  children,
  narrow,
  mainClassName,
}: {
  title: string
  documentTitle?: string
  badge?: ReactNode
  exit?: ReactNode
  step?: StepNo
  hints?: Partial<Record<StepNo, string>>
  footer?: ReactNode
  children: ReactNode
  /** The summary layout: no rail, 960px wide. */
  narrow?: boolean
  /** Board-specific control sizes (s34, sfull, ftd8 in imports.css). */
  mainClassName?: string
}) {
  const t = useT()
  const ws = useWorkspace()
  useDocumentTitle(`${documentTitle ?? title} · ${ws.me.workspace.name} · ReputeCast`)
  // On phones the steps scroll sideways; then the rail takes keyboard focus.
  const stepsRef = useRef<HTMLElement>(null)
  const stepsScroll = useOverflowX(stepsRef)
  return (
    <div className="mig rc-app">
      <header>
        <Link to="/w/$workspace" params={{ workspace: ws.slug }} className="flex items-center gap-2.5 text-ink" aria-label={t(m.home)}>
          <LogoMark size={26} />
          <span className="text-[15px] font-bold">ReputeCast</span>
        </Link>
        <span className="k">/</span>
        <h1>{title}</h1>
        {badge}
        {exit ? <div className="ml-auto">{exit}</div> : null}
      </header>
      {step === undefined ? (
        <main
          id="main"
          tabIndex={-1}
          className={cn('mx-auto box-border flex w-full flex-col', narrow ? 'max-w-[960px] gap-[18px] px-7 py-9' : 'max-w-[1300px] gap-7 p-7')}
        >
          {children}
        </main>
      ) : (
        <div className="wiz">
          {/* A scrolling region is the exception to the rule: it must take focus. */}
          {/* eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex */}
          <nav ref={stepsRef} className="card wizsteps" aria-label={t(m.steps)} tabIndex={stepsScroll ? 0 : undefined}>
            {steps.map((s) => {
              const state = s.n < step ? 'done' : s.n === step ? 'cur' : ''
              return (
                <div key={s.n} className={cn('step', state)} aria-current={state === 'cur' ? 'step' : undefined}>
                  <span className="num">{state === 'done' ? '✓' : s.n}</span>
                  <div>
                    <div className="font-semibold">
                      {t(s.label)}
                      {state === 'done' ? <span className="vh"> {t(m.stepDone)}</span> : null}
                    </div>
                    {hints[s.n] ? <div className="k">{hints[s.n]}</div> : null}
                  </div>
                </div>
              )
            })}
            {footer ? <div className="railfoot k mt-2.5 border-t border-line-2 pt-3">{footer}</div> : null}
          </nav>
          <main id="main" tabIndex={-1} className={mainClassName}>
            {children}
          </main>
        </div>
      )}
    </div>
  )
}

export function WizardHeading({ title, lead }: { title: ReactNode; lead?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <h2 className="wizh">{title}</h2>
      {lead ? <span className="lead">{lead}</span> : null}
    </div>
  )
}

/** "Previous imports: Brevo, 18 Sep, 46,118 contacts. View report" */
export function PreviousImport({ p }: { p?: Schemas['ImportSummary'] }) {
  const { slug } = useWorkspace()
  if (!p) return null
  return (
    <T
      msg={m.previous}
      values={{
        source: p.sourceName,
        date: dayMonthDate(p.finishedAt ?? p.createdAt),
        n: p.contacts,
        count: count(p.contacts),
        link: (c) => (
          <Link to={stepPath.report} params={{ workspace: slug, importId: p.id }}>
            {c}
          </Link>
        ),
      }}
    />
  )
}

/** One import, polled while it reads the source or runs. */
export function useImportDetail(id: string) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/imports/{id}',
    { params: { path: { workspace: slug, id } } },
    {
      refetchInterval: (q) => {
        const s = q.state.data?.status
        return s === 'staging' || s === 'running' || s === 'paused' ? 1000 : false
      },
    },
  )
}

/** Refreshes everything the wizard shows about one import. */
export function useRefreshImport() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ predicate: (q) => String((q.queryKey as unknown[])[1] ?? '').includes('/imports') })
}

/** "Mailchimp · Acme Customers" for the rail. */
export function sourceHint(d: Detail) {
  if (d.fileName && (d.source === 'csv' || d.source === 'xlsx')) return d.fileName
  const picked = new Set(d.options.lists.map((l) => l.sourceId))
  const names = (d.connection?.lists ?? []).filter((l) => picked.has(l.id)).map((l) => l.name)
  return names.length > 0 ? `${d.sourceName} · ${names.join(', ')}` : d.sourceName
}
