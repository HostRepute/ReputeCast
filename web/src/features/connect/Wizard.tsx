import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { LogoMark } from '@/components/icons/Icon'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { Row } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { Setup } from './api'
import './connect.css'

/**
 * The setup wizard's page: its own top bar (brand, title, step badge, Exit
 * setup) instead of the app shell, then the steps and the content column.
 */
export function WizardLayout({
  title,
  badge,
  exit = true,
  children,
}: {
  title: string
  /** "Step 2 of 6", or the Done badge. */
  badge: ReactNode
  exit?: boolean
  children: ReactNode
}) {
  const { slug, me } = useWorkspace()
  useDocumentTitle(`${title} · ${me.workspace.name} · ReputeCast`)
  return (
    <div className="rc-app flex min-h-screen flex-col">
      <a href="#main" className="skip">
        Skip to content
      </a>
      <header className="cn-head">
        <Link to="/w/$workspace" params={{ workspace: slug }} className="cn-home" aria-label="ReputeCast home">
          <LogoMark size={26} />
          <span className="text-[15px] font-bold">ReputeCast</span>
        </Link>
        <span className="k" aria-hidden="true">
          /
        </span>
        <h1 className="m-0 text-[16px] font-semibold">{title}</h1>
        {badge}
        {exit ? (
          <ButtonLink to="/w/$workspace" params={{ workspace: slug }} className="ml-auto">
            Exit setup
          </ButtonLink>
        ) : null}
      </header>
      <SystemBanners />
      {children}
    </div>
  )
}

/** "Step 2 of 6" */
export function StepBadge({ n }: { n: number }) {
  return <Badge tone="neutral">{`Step ${n} of 6`}</Badge>
}

/** The steps column and the content column. */
export function WizardBody({ current, setup, children }: { current: number; setup?: Setup; children: ReactNode }) {
  return (
    <div className="cn-wiz">
      <Stepper current={current} setup={setup} />
      <main className="cn-main" id="main" tabIndex={-1}>
        {children}
      </main>
    </div>
  )
}

type StepDef = { title: string; catalog: string }

const STEP_DEFS: StepDef[] = [
  { title: 'Provider', catalog: 'Pick from the catalog' },
  { title: 'Connection', catalog: 'Keys, region, webhooks' },
  { title: 'Domain verification', catalog: 'DKIM, return-path, SPF, DMARC' },
  { title: 'Test send', catalog: 'Watch the event arrive' },
  { title: 'Streams', catalog: 'Broadcasts, Automations, System' },
  { title: 'Contacts', catalog: 'Import or start empty' },
]

/**
 * The six steps. Each line under a step says what it is about: the
 * catalog describes them all; later the current step says what it asks
 * for, the steps behind it what they set up (the last one in full) and the
 * next one, from the connection step, what it will check.
 */
export function Stepper({ current, setup }: { current: number; setup?: Setup }) {
  const s = setup?.summaries
  const info = setup?.info
  const line = (n: number): string => {
    if (current === 1) return STEP_DEFS[n - 1]!.catalog
    if (n === current) {
      switch (n) {
        case 2:
          return info?.connectionHint ?? ''
        case 3:
          return setup?.domain.name || s?.domainShort || ''
        case 4:
          return 'Watch the event arrive'
        case 5:
          return `What ${info?.short ?? 'it'} will carry`
        case 6:
          return 'Import or start empty'
      }
      return ''
    }
    if (n > current) return current === 2 && n === 3 ? (info?.domainHint ?? '') : ''
    // Steps behind: the provider by name, then what each one set up.
    if (n === 1) return info?.step1 ?? ''
    const latest = n === current - 1
    if (!latest && current >= 6) return ''
    switch (n) {
      case 2:
        return s?.connection ?? ''
      case 3:
        return (latest ? s?.domain : s?.domainShort) ?? ''
      case 4:
        return s?.test ?? ''
      case 5:
        return s?.streams ?? ''
    }
    return ''
  }
  return (
    <nav className="card cn-steps" aria-label="Setup steps">
      {STEP_DEFS.map((d, i) => {
        const n = i + 1
        const done = n < current
        const cur = n === current
        const sub = line(n)
        return (
          <div key={d.title} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? '✓' : n}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">Done: </span> : null}
                {d.title}
              </div>
              {sub ? <div className="k">{sub}</div> : null}
            </div>
          </div>
        )
      })}
      {current === 1 ? (
        <div className="cn-foot k mt-2.5 border-t border-line-2 pt-3">
          Most people finish in about 10 minutes. You can add more providers later and send through several at once.
        </div>
      ) : null}
    </nav>
  )
}

/** The step's heading and lead. */
export function StepHeading({ title, children }: { title: ReactNode; children?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <h2 className="m-0 text-[24px] font-bold tracking-[-0.02em]">{title}</h2>
      {children ? <span className="text-ink-2">{children}</span> : null}
    </div>
  )
}

/** Back on the left; later actions and the primary one on the right. */
export function WizardFooter({ back, children }: { back?: ReactNode; children?: ReactNode }) {
  return (
    <div className="flex flex-wrap justify-between gap-2.5">
      {back ?? <span />}
      {children ? <Row gap={10}>{children}</Row> : null}
    </div>
  )
}

/** The wizard's 40 px primary action. */
export const continueClass = 'min-h-10 px-4.5 aria-disabled:opacity-45'
