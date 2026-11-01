import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { LogoMark } from '@/components/icons/Icon'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { Row } from '@/components/ui/Layout'
import { m as shell } from '@/components/shell/messages'
import { t as tt, useT, type MessageDescriptor } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import type { Setup } from './api'
import { m } from './messages'
import './connect.css'
import { Stepper } from '@/components/ui/Stepper'

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
  const t = useT()
  const { slug, me } = useWorkspace()
  useDocumentTitle(t(shell.docTitleWorkspace, { title, workspace: me.workspace.name }))
  return (
    <div className="rc-app flex min-h-screen flex-col">
      <a href="#main" className="skip">
        {t(shell.skipToContent)}
      </a>
      <header className="cn-head">
        <Link to="/w/$workspace" params={{ workspace: slug }} className="cn-home" aria-label={t(shell.homeLabel)}>
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
            {t(m.exitSetup)}
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
  const t = useT()
  return <Badge tone="neutral">{t(m.stepBadge, { n })}</Badge>
}

/** The steps column and the content column. */
export function WizardBody({ current, setup, children }: { current: number; setup?: Setup; children: ReactNode }) {
  return (
    <div className="cn-wiz">
      <WizardSteps current={current} setup={setup} />
      <main className="cn-main" id="main" tabIndex={-1}>
        {children}
      </main>
    </div>
  )
}

type StepDef = { key: string; title: MessageDescriptor; catalog: MessageDescriptor }

const STEP_DEFS: StepDef[] = [
  { key: 'provider', title: m.stepProvider, catalog: m.stepProviderLine },
  { key: 'connection', title: m.stepConnection, catalog: m.stepConnectionLine },
  { key: 'domain', title: m.stepDomain, catalog: m.stepDomainLine },
  { key: 'test', title: m.stepTest, catalog: m.stepTestLine },
  { key: 'streams', title: m.stepStreams, catalog: m.stepStreamsLine },
  { key: 'contacts', title: m.stepContacts, catalog: m.stepContactsLine },
]

/**
 * The line under step n of the stepper. The catalog describes every step;
 * later the current step says what it asks for, the steps behind it what
 * they set up (the last one in full) and, from the connection step, the
 * next one what it will check (SES and SMTP: SetupError has none for API
 * providers).
 */
export function stepLine(n: number, current: number, setup?: Pick<Setup, 'summaries' | 'info' | 'domain'>): string {
  const s = setup?.summaries
  const info = setup?.info
  if (current === 1) return tt(STEP_DEFS[n - 1]!.catalog)
  if (n === current) {
    switch (n) {
      case 2:
        return info?.connectionHint ?? ''
      case 3:
        return setup?.domain.name || s?.domainShort || ''
      case 4:
        return tt(m.stepTestLine)
      case 5:
        return info?.short ? tt(m.stepStreamsCurrent, { provider: info.short }) : tt(m.stepStreamsCurrentNoName)
      case 6:
        return tt(m.stepContactsLine)
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

/** The six steps, each with its line (stepLine). */
function WizardSteps({ current, setup }: { current: number; setup?: Setup }) {
  const t = useT()
  return (
    <Stepper
      label={t(m.stepsLabel)}
      current={current}
      steps={STEP_DEFS.map((d, i) => ({ key: d.key, title: t(d.title), line: stepLine(i + 1, current, setup) }))}
      foot={current === 1 ? t(m.stepsFoot) : undefined}
    />
  )
}

export function StepHeading({ title, children, note }: { title: ReactNode; children?: ReactNode; note?: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <h2 className="m-0 text-[24px] font-bold tracking-[-0.02em]">{title}</h2>
      {children ? <span className="text-ink-2">{children}</span> : null}
      {note ? <span className="k">{note}</span> : null}
    </div>
  )
}

/**
 * Back on the left; later actions and the primary one on the right. A
 * caption under them says why the primary action is (un)available; give it
 * `captionId` and point the action's aria-describedby at it.
 */
export function WizardFooter({ back, children, caption, captionId }: { back?: ReactNode; children?: ReactNode; caption?: ReactNode; captionId?: string }) {
  const row = (
    <div className="flex flex-wrap justify-between gap-2.5">
      {back ?? <span />}
      {children ? <Row gap={10}>{children}</Row> : null}
    </div>
  )
  if (!caption) return row
  return (
    <div className="flex flex-col gap-2">
      {row}
      <p className="k m-0 text-right max-md:text-left" id={captionId}>
        {caption}
      </p>
    </div>
  )
}

/** The wizard's 40 px primary action. */
export const continueClass = 'min-h-10 px-4.5 aria-disabled:opacity-45'
