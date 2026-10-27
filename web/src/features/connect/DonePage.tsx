import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { useT, type MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { ms } from './setup.messages'
import { Ticks } from './text'
import { WizardLayout } from './Wizard'

type Row = Schemas['ConnectDoneRow']

const ROW_BADGE: Record<Row['status'], { tone: 'ok' | 'neutral' | 'bad' | 'warn'; label: MessageDescriptor }> = {
  done: { tone: 'ok', label: m.stateDone },
  verified: { tone: 'ok', label: m.stateVerified },
  skipped: { tone: 'neutral', label: m.stateSkipped },
  failed: { tone: 'bad', label: m.stateFailed },
  pending: { tone: 'warn', label: m.statePending },
}

/** 7.2 summary (SetupDone): what was configured and what to do next. */
export function DonePage({ setup }: { setup: Schemas['ConnectSetup'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const done = setup.done!
  return (
    <WizardLayout title={t(m.title)} badge={<Badge tone="ok">{t(ms.doneBadge)}</Badge>} exit={false}>
      {/* The summary isn't a wizard step: as on the board it keeps its 40 px × 28 px padding at every width. */}
      <main id="main" tabIndex={-1} className="mx-auto box-border flex w-full max-w-[860px] flex-col gap-[18px] px-7 py-10">
        <div className="flex flex-wrap items-center gap-[18px]">
          <svg width="56" height="56" viewBox="0 0 40 40" aria-hidden="true" className="pop">
            <circle cx="20" cy="20" r="19" fill="#BBF7D0" />
            <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
          </svg>
          <div className="flex flex-col gap-1">
            <h2 className="m-0 text-[26px] font-bold tracking-[-0.02em]">{done.title}</h2>
            <span className="text-ink-2">{done.domain ? t(ms.tookDomain, { minutes: done.minutes, domain: done.domain }) : t(ms.took, { minutes: done.minutes })}</span>
          </div>
        </div>

        <section className="card flex flex-col" aria-labelledby="configured-h">
          <div className="flex items-center justify-between px-4.5 py-3.5 max-md:flex-wrap">
            <h3 className="h2" id="configured-h">
              {t(ms.configuredTitle)}
            </h3>
            <span className="k">{t(ms.configuredLine)}</span>
          </div>
          {done.rows.map((r) => (
            <div key={r.lead} className="cn-row lg">
              <Badge tone={ROW_BADGE[r.status].tone} dot>
                {t(ROW_BADGE[r.status].label)}
              </Badge>
              <span className="cn-row-text min-w-0">
                <strong>{r.lead}</strong>
                <Ticks text={r.text} />
                {r.importLink ? (
                  <>
                    {' '}
                    <Link to="/w/$workspace/import" params={{ workspace: slug }}>
                      {t(ms.importNow)}
                    </Link>
                  </>
                ) : null}
              </span>
            </div>
          ))}
        </section>

        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="recommended-h">
          <h3 className="h2" id="recommended-h">
            {t(ms.recommendedTitle)}
          </h3>
          <div className="grid gap-3 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(240px, 100%), 1fr))' }}>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{t(ms.addProviderTitle)}</span>
              <span className="k">{t(ms.addProviderLine)}</span>
              <ButtonLink size="sm" variant="primary" className="self-start" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                {t(ms.addProvider)}
              </ButtonLink>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{t(ms.hostReputeTitle)}</span>
              <span className="k">{done.hostRepute ? t(ms.hostReputeConnected) : t(ms.hostReputeOptional)}</span>
              <ButtonLink size="sm" className="self-start" to={done.hostRepute ? '/w/$workspace/deliverability' : '/w/$workspace/deliverability/connect'} params={{ workspace: slug }}>
                {done.hostRepute ? t(ms.open) : t(ms.connect)}
              </ButtonLink>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">{t(ms.formTitle)}</span>
              <span className="k">{t(ms.formLine)}</span>
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/forms" params={{ workspace: slug }}>
                {t(ms.buildForm)}
              </ButtonLink>
            </div>
          </div>
        </section>

        <div className="flex flex-wrap justify-end gap-2.5">
          <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            {t(ms.addProvider)}
          </ButtonLink>
          <ButtonLink variant="primary" className="min-h-10 px-4.5" to="/w/$workspace" params={{ workspace: slug }}>
            {t(ms.goToOverview)}
          </ButtonLink>
        </div>
      </main>
    </WizardLayout>
  )
}
